package codegraph

import (
	"codebase-indexer/internal/repository"
	"codebase-indexer/pkg/codegraph/analyzer"
	"codebase-indexer/pkg/codegraph/lang"
	"codebase-indexer/pkg/codegraph/parser"
	"codebase-indexer/pkg/codegraph/proto"
	"codebase-indexer/pkg/codegraph/proto/codegraphpb"
	"codebase-indexer/pkg/codegraph/resolver"
	"codebase-indexer/pkg/codegraph/store"
	"codebase-indexer/pkg/codegraph/types"
	"codebase-indexer/pkg/codegraph/utils"
	"codebase-indexer/pkg/codegraph/workspace"
	"codebase-indexer/pkg/logger"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

type IndexerConfig struct {
	MaxConcurrency int // TODO 支持环境变量
	MaxBatchSize   int // TODO 支持环境变量
	MaxFiles       int // TODO 支持环境变量
	MaxProjects    int // TODO 支持环境变量
	VisitPattern   *types.VisitPattern
}

// Indexer 代码索引器
type Indexer struct {
	parser              *parser.SourceFileParser     // 单文件语法解析
	analyzer            *analyzer.DependencyAnalyzer // 跨文件依赖分析
	workspaceReader     *workspace.WorkspaceReader   // 进行工作区的文件读取、项目识别、项目列表维护
	storage             store.GraphStorage           // 存储
	workspaceRepository repository.WorkspaceRepository
	config              IndexerConfig
	logger              logger.Logger
}

// BatchProcessParams 批处理参数
type BatchProcessParams struct {
	ProjectUuid string
	SourceFiles []string
	BatchStart  int
	BatchEnd    int
	BatchSize   int
	TotalFiles  int
	Project     *workspace.Project
}

// BatchProcessResult 批处理结果
type BatchProcessResult struct {
	ElementTablesCnt int
	Metrics          *types.IndexTaskMetrics
	Duration         time.Duration
}

// BatchProcessingParams 批处理阶段参数
type BatchProcessingParams struct {
	ProjectUuid   string
	SourceFiles   []string
	TotalFiles    int
	PreviousNum   int
	Project       *workspace.Project
	WorkspacePath string
	Concurrency   int
	BatchSize     int
}

// BatchProcessingResult 批处理阶段结果
type BatchProcessingResult struct {
	ParsedFilesCount int
	ProjectMetrics   *types.IndexTaskMetrics
	Errors           []error
	Duration         time.Duration
}

// DependencyAnalysisParams 依赖分析参数
type DependencyAnalysisParams struct {
	ProjectUuid    string
	ParsedFilesCnt int
	WorkspacePath  string
	PreviousNum    int
	BatchSize      int
}

// DependencyAnalysisResult 依赖分析结果
type DependencyAnalysisResult struct {
	ElementTablesCount int
	Errors             []error
	Duration           time.Duration
}

// ProgressInfo 进度信息
type ProgressInfo struct {
	Total         int
	Processed     int
	PreviousNum   int
	WorkspacePath string
}

const (
	defaultConcurrency = 1
	defaultBatchSize   = 50
	defaultMaxFiles    = 50000
	defaultMaxProjects = 3
)

// TODO 并行度、batch_size、cpu、内存使用优化。
// TODO resolve_dependency 优化，内存飙升。
// NewCodeIndexer 创建新的代码索引器
func NewCodeIndexer(
	parser *parser.SourceFileParser,
	analyzer *analyzer.DependencyAnalyzer,
	workspaceReader *workspace.WorkspaceReader,
	storage store.GraphStorage,
	workspaceRepository repository.WorkspaceRepository,
	config IndexerConfig,
	logger logger.Logger,
) *Indexer {
	if config.MaxConcurrency <= 0 {
		config.MaxConcurrency = defaultConcurrency
	}
	if config.MaxBatchSize <= 0 {
		config.MaxBatchSize = defaultBatchSize
	}
	if config.MaxFiles <= 0 {
		config.MaxFiles = defaultMaxFiles
	}
	if config.MaxProjects <= 0 {
		config.MaxProjects = defaultMaxProjects
	}
	return &Indexer{
		parser:              parser,
		analyzer:            analyzer,
		workspaceReader:     workspaceReader,
		storage:             storage,
		workspaceRepository: workspaceRepository,
		config:              config,
		logger:              logger,
	}
}

// IndexWorkspace 索引整个工作区
func (i *Indexer) IndexWorkspace(ctx context.Context, workspacePath string) (*types.IndexTaskMetrics, error) {
	taskMetrics := &types.IndexTaskMetrics{}
	workspaceStart := time.Now()
	i.logger.Info("index_workspace start to index workspace：%s", workspacePath)
	exists, err := i.workspaceReader.Exists(ctx, workspacePath)
	if err == nil && !exists {
		return taskMetrics, fmt.Errorf("index_workspace workspace %s not exists", workspacePath)
	}
	projects := i.workspaceReader.FindProjects(ctx, workspacePath, true, workspace.DefaultVisitPattern)
	projectsCnt := len(projects)
	if projectsCnt == 0 {
		return taskMetrics, fmt.Errorf("index_workspace find no projects in workspace: %s", workspacePath)
	}
	if projectsCnt > i.config.MaxProjects {
		projects = projects[:i.config.MaxProjects]
		i.logger.Debug("index_workspace %s found %d projects, exceed %d max_projects config, use config size.", workspacePath, projectsCnt)
	}

	var errs []error

	// 循环项目，逐个处理
	for _, project := range projects {
		projectTaskMetrics, err := i.indexProject(ctx, workspacePath, project)
		if err != nil {
			i.logger.Error("index_workspace index project %s err: %v",
				project.Path, utils.TruncateError(errors.Join(err...)))
			errs = append(errs, err...)
			continue
		}

		taskMetrics.TotalFiles += projectTaskMetrics.TotalFiles
		taskMetrics.TotalFailedFiles += projectTaskMetrics.TotalFailedFiles
		taskMetrics.FailedFilePaths = append(taskMetrics.FailedFilePaths, projectTaskMetrics.FailedFilePaths...)
	}

	i.logger.Info("index_workspace %s index workspace finish, cost %d ms, visit %d files, "+
		"parsed %d files successfully, failed %d files", workspacePath, time.Since(workspaceStart).Milliseconds(),
		taskMetrics.TotalFiles, taskMetrics.TotalFiles-taskMetrics.TotalFailedFiles, taskMetrics.TotalFailedFiles)
	return taskMetrics, nil
}

// indexProject 索引单个项目
func (i *Indexer) indexProject(ctx context.Context, workspacePath string, project *workspace.Project) (*types.IndexTaskMetrics, []error) {
	projectStart := time.Now()
	projectUuid := project.Uuid

	i.logger.Info("index_project start to index project：%s, max_concurrency: %d, batch_size: %d",
		project.Path, i.config.MaxConcurrency, i.config.MaxBatchSize)

	// 获取工作区信息
	workspaceModel, err := i.workspaceRepository.GetWorkspaceByPath(workspacePath)
	if err != nil {
		return nil, []error{err}
	}
	if workspaceModel == nil {
		return nil, []error{fmt.Errorf("index_project workspace %s not found in database", workspacePath)}
	}
	previousNum := workspaceModel.CodegraphFileNum

	// 阶段0：收集要处理的源码文件
	sourceFiles, err := i.collectSourceFiles(ctx, project.Path, i.config.MaxFiles)
	if err != nil {
		projectMetrics := &types.IndexTaskMetrics{TotalFiles: 0}
		return projectMetrics, []error{fmt.Errorf("index_project collect project files err:%v", err)}
	}
	totalFiles := len(sourceFiles)
	if totalFiles == 0 {
		i.logger.Info("index_project found no source files in project %s, not index.", project.Path)
		return &types.IndexTaskMetrics{TotalFiles: 0}, nil
	}

	i.logger.Info("index_project collect project files finish. cost %d ms, found %d source files to index",
		time.Since(projectStart).Milliseconds(), totalFiles)

	// CPU优化：使用增强版的并发度优化
	i.logger.Info("index_project %s, concurrency: %d, batch_size: %d",
		project.Path, i.config.MaxConcurrency, i.config.MaxBatchSize)

	// 阶段1-3：批量处理文件（解析、检查、保存符号表）
	batchProcessingParams := &BatchProcessingParams{
		ProjectUuid:   projectUuid,
		SourceFiles:   sourceFiles,
		TotalFiles:    totalFiles,
		Project:       project,
		WorkspacePath: workspacePath,
		PreviousNum:   previousNum,
		Concurrency:   i.config.MaxConcurrency,
		BatchSize:     i.config.MaxBatchSize,
	}

	batchResult, err := i.processParseInBatches(ctx, batchProcessingParams)
	if err != nil {
		return nil, []error{err}
	}

	// 合并错误
	var allErrs []error
	allErrs = append(allErrs, batchResult.Errors...)

	i.logger.Info("index_project project %s files parse finish. cost %d ms, visit %d files, "+
		"parsed %d files successfully, failed %d files",
		project.Path, time.Since(projectStart).Milliseconds(), batchResult.ProjectMetrics.TotalFiles,
		batchResult.ProjectMetrics.TotalFiles-batchResult.ProjectMetrics.TotalFailedFiles, batchResult.ProjectMetrics.TotalFailedFiles)

	if batchResult.ProjectMetrics.TotalFiles-batchResult.ProjectMetrics.TotalFailedFiles == 0 {
		i.logger.Info("index_project project %s parse and save 0 element_tables successfully, process end.",
			project.Path)
		return batchResult.ProjectMetrics, allErrs
	}

	//// sleep 5 秒，gc
	//// 阶段4：依赖分析
	//dependencyParams := &DependencyAnalysisParams{
	//	ProjectUuid:    projectUuid,
	//	Processed: batchResult.ParsedFilesCount,
	//	WorkspacePath:  workspacePath,
	//	PreviousNum:    previousNum,
	//	BatchSize:      i.config.MaxBatchSize,
	//}
	//
	//dependencyResult, err := i.processDependencyAnalysis(ctx, dependencyParams)
	//if err != nil {
	//	allErrs = append(allErrs, err)
	//	return batchResult.ProjectMetrics, allErrs
	//}
	//
	//allErrs = append(allErrs, dependencyResult.Errors...)
	//
	//i.logger.Debug("index_project project %s analyze dependency end, analyzed %d/%d element tables, cost %d ms.",
	//	project.Path, dependencyResult.Total, batchResult.ParsedFilesCount, dependencyResult.Duration.Milliseconds())

	i.logger.Info("index_project %s finish, cost %d ms, processed %d files.", project.Path,
		time.Since(projectStart).Milliseconds(), batchResult.ParsedFilesCount)

	return batchResult.ProjectMetrics, allErrs
}

// preprocessImports 预处理（过滤、转换分隔符）
func (i *Indexer) preprocessImports(ctx context.Context, elementTables []*parser.FileElementTable,
	project *workspace.Project) error {
	var errs []error
	for _, ft := range elementTables {
		imps, err := i.analyzer.PreprocessImports(ctx, ft.Language, project, ft.Imports)
		if err != nil {
			errs = append(errs, err)
		} else {
			ft.Imports = imps
		}
	}
	return errors.Join(errs...)
}

// ParseProjectFiles 解析项目中的所有文件
func (i *Indexer) ParseProjectFiles(ctx context.Context, p *workspace.Project) ([]*parser.FileElementTable, types.IndexTaskMetrics, error) {
	fileElementTables := make([]*parser.FileElementTable, 0)
	projectTaskMetrics := types.IndexTaskMetrics{}

	// TODO walk 目录收集列表， 并发构建，批量保存结果
	if err := i.workspaceReader.WalkFile(ctx, p.Path, func(walkCtx *types.WalkContext) error {
		projectTaskMetrics.TotalFiles++
		language, err := lang.InferLanguage(walkCtx.Path)
		if err != nil || language == types.EmptyString {
			// not supported language or not source file
			return nil
		}

		content, err := i.workspaceReader.ReadFile(ctx, walkCtx.Path, types.ReadOptions{})
		if err != nil {
			projectTaskMetrics.TotalFailedFiles++
			return err
		}
		fileElementTable, err := i.parser.Parse(ctx, &types.SourceFile{
			Path:    walkCtx.Path,
			Content: content,
		})

		if err != nil {
			projectTaskMetrics.TotalFailedFiles++
			return err
		}
		fileElementTables = append(fileElementTables, fileElementTable)
		return nil
	}, types.WalkOptions{
		IgnoreError:  true,
		VisitPattern: i.config.VisitPattern,
	}); err != nil {
		return nil, types.IndexTaskMetrics{}, err
	}

	return fileElementTables, projectTaskMetrics, nil
}

// RemoveIndexes 根据工作区路径、文件路径/文件夹路径前缀，批量删除索引
func (i *Indexer) RemoveIndexes(ctx context.Context, workspacePath string, filePaths []string) error {
	start := time.Now()
	i.logger.Info("remove_indexes start to remove workspace %s files: %v", workspacePath, filePaths)

	projects := i.workspaceReader.FindProjects(ctx, workspacePath, false, workspace.DefaultVisitPattern)
	if len(projects) == 0 {
		return fmt.Errorf("no project found in workspace %s", workspacePath)
	}

	var errs []error
	projectFilesMap, err := i.groupFilesByProject(projects, filePaths)
	if err != nil {
		return fmt.Errorf("group files by project failed: %w", err)
	}

	for projectUuid, files := range projectFilesMap {
		pStart := time.Now()
		i.logger.Info("remove_indexes start to remove project %s files index", projectUuid)

		if err := i.removeIndexByFilePaths(ctx, projectUuid, files); err != nil {
			errs = append(errs, err)
		}

		i.logger.Info("remove_indexes remove project %s files index end, cost %d ms", projectUuid,
			time.Since(pStart).Milliseconds())
	}

	err = errors.Join(errs...)
	i.logger.Info("remove_indexes remove workspace %s files index successfully, cost %d ms, errors: %v",
		workspacePath, time.Since(start).Milliseconds(), utils.TruncateError(err))
	return err
}

// removeIndexByFilePaths 删除单个项目的索引
func (i *Indexer) removeIndexByFilePaths(ctx context.Context, projectUuid string, filePaths []string) error {
	// 1. 查询path相应的file_table
	deleteFileTables, err := i.searchFileElementTablesByPath(ctx, projectUuid, filePaths)
	if err != nil {
		return fmt.Errorf("get file tables for deletion failed: %w", err)
	}
	deletePaths := make(map[string]any)
	for _, v := range deleteFileTables {
		deletePaths[v.Path] = nil
	}

	// 2. 清理符号定义
	if err = i.cleanupSymbolOccurrences(ctx, projectUuid, deleteFileTables, deletePaths); err != nil {
		return fmt.Errorf("cleanup symbol definitions failed: %w", err)
	}

	// 3. 删除path索引
	if err = i.deleteFileIndexes(ctx, projectUuid, filePaths); err != nil {
		return fmt.Errorf("delete file indexes failed: %w", err)
	}

	return nil
}

// searchFileElementTablesByPath 获取待删除的文件表和路径（包括文件夹）
func (i *Indexer) searchFileElementTablesByPath(ctx context.Context, puuid string, filePaths []string) ([]*codegraphpb.FileElementTable, error) {
	var deleteFileTables []*codegraphpb.FileElementTable
	var errs []error

	for _, filePath := range filePaths {
		language, err := lang.InferLanguage(filePath)
		var fileTable []byte
		if err == nil {
			key := store.ElementPathKey{Language: language, Path: filePath}
			fileTable, err = i.storage.Get(ctx, puuid, key)
		}

		if lang.IsUnSupportedFileError(err) || errors.Is(err, store.ErrKeyNotFound) {
			// 精确匹配不到，使用前缀模糊匹配
			i.logger.Info("indexer delete index, key path %s not found in store, use prefix search", filePath)
			tables, errors_ := i.searchFileElementTablesByPathPrefix(ctx, puuid, filePath)
			if len(errors_) > 0 {
				errs = append(errs, errors_...)
			}
			if len(tables) > 0 {
				deleteFileTables = append(deleteFileTables, tables...)
			}
			continue
		}

		if err != nil {
			errs = append(errs, err)
			continue
		}
		ft := new(codegraphpb.FileElementTable)
		if err = store.UnmarshalValue(fileTable, ft); err != nil {
			errs = append(errs, err)
			continue
		}
		deleteFileTables = append(deleteFileTables, ft)
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	return deleteFileTables, nil
}

func (i *Indexer) searchFileElementTablesByPathPrefix(ctx context.Context, projectUuid string, path string) (
	[]*codegraphpb.FileElementTable, []error) {
	var errs []error
	tables := make([]*codegraphpb.FileElementTable, 0)
	iter := i.storage.Iter(ctx, projectUuid)
	for iter.Next() {
		if !store.IsElementPathKey(iter.Key()) {
			continue
		}
		pathKey, err := store.ToElementPathKey(iter.Key())
		if err != nil {
			i.logger.Error("indexer delete index, parse element path key %s err:%v", iter.Key(), err)
			continue
		}
		// path 可能包含分隔符，也可能不包含，统一处理
		pathPrefix := utils.EnsureTrailingSeparator(path)
		if strings.HasPrefix(pathKey.Path, pathPrefix) {
			ft := new(codegraphpb.FileElementTable)
			if err = store.UnmarshalValue(iter.Value(), ft); err != nil {
				errs = append(errs, err)
				continue
			}
			tables = append(tables, ft)
		}
	}
	err := iter.Close()
	if err != nil {
		i.logger.Error("indexer close graph_store err:%v", err)
	}
	return tables, errs
}

// cleanupSymbolOccurrences 清理符号定义
func (i *Indexer) cleanupSymbolOccurrences(ctx context.Context, projectUuid string,
	deleteFileTables []*codegraphpb.FileElementTable, deletedPaths map[string]interface{}) error {
	var errs []error

	for _, ft := range deleteFileTables {
		for _, e := range ft.Elements {
			if e.IsDefinition {
				language := lang.Language(ft.Language)
				sym, err := i.storage.Get(ctx, projectUuid, store.SymbolNameKey{Language: language, Name: e.GetName()})
				if err != nil {
					errs = append(errs, err)
					continue
				}
				symDefs := new(codegraphpb.SymbolOccurrence)
				if err = store.UnmarshalValue(sym, symDefs); err != nil {
					return fmt.Errorf("unmarshal SymbolOccurrence error:%w", err)
				}

				newSymDefs := &codegraphpb.SymbolOccurrence{
					Name:        e.GetName(),
					Occurrences: make([]*codegraphpb.Occurrence, 0),
				}
				for _, d := range symDefs.Occurrences {
					if _, ok := deletedPaths[d.Path]; ok {
						continue
					}
					newSymDefs.Occurrences = append(newSymDefs.Occurrences, d)
				}
				// 保存更新后的文件表
				if err := i.storage.Put(ctx, projectUuid, &store.Entry{Key: store.SymbolNameKey{Language: language,
					Name: newSymDefs.Name}, Value: newSymDefs}); err != nil {
					errs = append(errs, err)
				}
			}
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// deleteFileIndexes 删除文件索引
func (i *Indexer) deleteFileIndexes(ctx context.Context, puuid string, filePaths []string) error {
	var errs []error

	for _, fp := range filePaths {
		// 删除path索引
		language, err := lang.InferLanguage(fp)
		if err != nil {
			continue
		}
		if err = i.storage.Delete(ctx, puuid, store.ElementPathKey{Language: language, Path: fp}); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// IndexFiles 根据工作区路径、文件路径，批量保存索引
func (i *Indexer) IndexFiles(ctx context.Context, workspacePath string, filePaths []string) error {
	i.logger.Info("index_files start to index workspace %s files: %v", workspacePath, filePaths)
	exists, err := i.workspaceReader.Exists(ctx, workspacePath)
	if err == nil && !exists {
		return fmt.Errorf("index_files workspace %s not exists", workspacePath)
	}
	projects := i.workspaceReader.FindProjects(ctx, workspacePath, true, workspace.DefaultVisitPattern)
	if len(projects) == 0 {
		return fmt.Errorf("no project found in workspace %s", workspacePath)
	}

	var errs []error
	projectFilesMap, err := i.groupFilesByProject(projects, filePaths)
	if err != nil {
		return fmt.Errorf("group files by project failed: %w", err)
	}

	for projectUuid, files := range projectFilesMap {
		if i.storage.Size(ctx, projectUuid, store.PathKeySystemPrefix) == 0 {
			i.logger.Info("index_files project %s has not indexed yet, index project.", projectUuid)
			var project *workspace.Project
			for _, p := range projects {
				if p.Uuid == projectUuid {
					project = p
				}
			}
			if project == nil {
				errs = append(errs, fmt.Errorf("index_files failed to find project by uuid %s", projectUuid))
				continue
			}
			// 如果项目没有索引过，索引整个项目
			_, err := i.indexProject(ctx, workspacePath, project)
			if err != nil {
				i.logger.Error("index_files index project %s err: %v", projectUuid, utils.TruncateError(errors.Join(err...)))
				errs = append(errs, err...)
			}
		} else {
			i.logger.Info("index_files project %s has index, index files.", projectUuid)
			// 索引指定文件
			if err := i.indexProjectFiles(ctx, projectUuid, files); err != nil {
				errs = append(errs, err)
			}
		}
	}

	err = errors.Join(errs...)
	i.logger.Info("index_files index workspace %s files successfully, errors: %v", workspacePath, filePaths,
		utils.TruncateError(err))
	return err
}

// indexProjectFiles 索引项目中的指定文件
func (i *Indexer) indexProjectFiles(ctx context.Context, projectUuid string, filePaths []string) error {
	var errs []error
	fileTables := make([]*parser.FileElementTable, 0)

	// 处理每个文件
	for _, f := range filePaths {
		if language, err := lang.InferLanguage(f); language == types.EmptyString || err != nil {
			continue
		}

		content, err := i.workspaceReader.ReadFile(ctx, f, types.ReadOptions{})
		if err != nil {
			errs = append(errs, fmt.Errorf("read file %s failed: %w", f, err))
			continue
		}

		fileElementTable, err := i.parser.Parse(ctx, &types.SourceFile{
			Path:    f,
			Content: content,
		})
		if err != nil {
			errs = append(errs, fmt.Errorf("parse file %s failed: %w", f, err))
			continue
		}
		fileTables = append(fileTables, fileElementTable)
	}

	if len(fileTables) == 0 {
		return fmt.Errorf("no valid filePaths to index in project %s", projectUuid)
	}

	// 保存本地符号表
	if err := i.analyzer.SaveSymbolOccurrences(ctx, projectUuid, fileTables); err != nil {
		return fmt.Errorf("save symbol definitions error: %w", err)
	}

	// 转换为 proto
	protoElementTables := proto.FileElementTablesToProto(fileTables)
	// 关系索引存储
	if err := i.storage.BatchSave(ctx, projectUuid, workspace.FileElementTables(protoElementTables)); err != nil {
		return fmt.Errorf("batch save error: %w", err)
	}

	return nil
}

// QueryElements 查询elements
func (i *Indexer) QueryElements(ctx context.Context, workspacePath string, filePaths []string) ([]*codegraphpb.FileElementTable, error) {
	i.logger.Info("query_elements start to query workspace %s files: %v", workspacePath, filePaths)

	projects := i.workspaceReader.FindProjects(ctx, workspacePath, false, workspace.DefaultVisitPattern)
	if len(projects) == 0 {
		return nil, fmt.Errorf("no project found in workspace %s", workspacePath)
	}

	var results []*codegraphpb.FileElementTable
	var errs []error

	projectFilesMap, err := i.groupFilesByProject(projects, filePaths)
	if err != nil {
		return nil, fmt.Errorf("group files by project failed: %w", err)
	}

	for puuid, pfiles := range projectFilesMap {
		for _, fp := range pfiles {
			language, err := lang.InferLanguage(fp)
			if err != nil {
				continue
			}
			fileTable, err := i.storage.Get(context.Background(), puuid, store.ElementPathKey{Language: language, Path: fp})
			if err != nil {
				errs = append(errs, fmt.Errorf("get file table %s failed: %w", fp, err))
				continue
			}
			ft := new(codegraphpb.FileElementTable)
			if err = store.UnmarshalValue(fileTable, ft); err != nil {
				errs = append(errs, err)
				continue
			}
			results = append(results, ft)
		}
	}

	if len(errs) > 0 {
		return results, fmt.Errorf("query elements completed with errors: %v", errs)
	}

	i.logger.Info("query_elements query workspace %s files successfully, found %d elements", workspacePath, len(results))
	return results, nil
}

// QuerySymbols 查询symbols
func (i *Indexer) QuerySymbols(ctx context.Context, workspacePath string, filePath string, symbolNames []string) ([]*codegraphpb.SymbolOccurrence, error) {
	i.logger.Info("query_symbols start to query workspace %s file %s symbols: %v", workspacePath, filePath, symbolNames)

	projects := i.workspaceReader.FindProjects(ctx, workspacePath, false, workspace.DefaultVisitPattern)
	if len(projects) == 0 {
		return nil, fmt.Errorf("no project found in workspace %s", workspacePath)
	}

	var results []*codegraphpb.SymbolOccurrence
	var errs []error

	// 找到文件路径对应的项目
	_, targetProjectUuid, err := i.findProjectForFile(projects, filePath)
	if err != nil {
		return nil, fmt.Errorf("find project for file failed: %w", err)
	}

	language, err := lang.InferLanguage(filePath)
	if err != nil {
		return nil, lang.ErrUnSupportedLanguage
	}
	// 查询每个符号名称
	for _, symbolName := range symbolNames {
		symbolDef, err := i.storage.Get(context.Background(), targetProjectUuid,
			store.SymbolNameKey{Language: language, Name: symbolName})
		if errors.Is(err, store.ErrKeyNotFound) {
			continue
		}
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to get symbol definition %s: %w", symbolName, err))
			continue
		}

		if symbolDef != nil {
			sd := new(codegraphpb.SymbolOccurrence)
			if err = store.UnmarshalValue(symbolDef, sd); err != nil {
				errs = append(errs, err)
				continue
			}
			results = append(results, sd)
		}
	}

	if len(errs) > 0 {
		return results, fmt.Errorf("query symbols completed with errors: %v", errs)
	}

	i.logger.Info("query_symbols query workspace %s file %s symbols successfully, found %d symbols",
		workspacePath, filePath, len(results))
	return results, nil
}

// groupFilesByProject 根据项目对文件进行分组
func (i *Indexer) groupFilesByProject(projects []*workspace.Project, filePaths []string) (map[string][]string, error) {
	projectFilesMap := make(map[string][]string)
	var errs []error

	for _, p := range projects {
		for _, filePath := range filePaths {
			if strings.HasPrefix(filePath, p.Path) {
				projectUuid := p.Uuid
				files, ok := projectFilesMap[projectUuid]
				if !ok {
					files = make([]string, 0)
				}
				files = append(files, filePath)
				projectFilesMap[projectUuid] = files
			}
		}
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	return projectFilesMap, nil
}

// findProjectForFile 查找文件所属的项目
func (i *Indexer) findProjectForFile(projects []*workspace.Project, filePath string) (*workspace.Project, string, error) {
	for _, p := range projects {
		if strings.HasPrefix(filePath, p.Path) {
			return p, p.Uuid, nil
		}
	}

	return nil, types.EmptyString, fmt.Errorf("no project found for file path %s", filePath)
}

// checkElementTables 检查element_tables
func (i *Indexer) checkElementTables(elementTables []*parser.FileElementTable) {
	start := time.Now()
	total, filtered := 0, 0
	for _, ft := range elementTables {
		newImports := make([]*resolver.Import, 0, len(ft.Imports))
		newElements := make([]resolver.Element, 0, len(ft.Elements))
		for _, imp := range ft.Imports {
			if resolver.IsValidElement(imp) {
				newImports = append(newImports, imp)
			} else {
				i.logger.Debug("check_element: invalid language %s file %s import {name:%s type:%s path:%s range:%v}",
					ft.Language, ft.Path, imp.Name, imp.Type, imp.Path, imp.Range)
			}
		}
		for _, ele := range ft.Elements {
			total++
			if resolver.IsValidElement(ele) {
				// 过滤掉 局部 变量
				variable, ok := ele.(*resolver.Variable)
				if ok {
					if variable.GetScope() == types.ScopeBlock || variable.GetScope() == types.ScopeFunction {
						continue
					}
				}
				newElements = append(newElements, ele)
			} else {
				filtered++
				i.logger.Debug("check_element: invalid language %s file %s element {name:%s type:%s path:%s range:%v}",
					ft.Language, ft.Path, ele.GetName(), ele.GetType(), ele.GetPath(), ele.GetRange())
			}
		}

		ft.Imports = newImports
		ft.Elements = newElements
	}
	i.logger.Debug("check_element: files total %d, elements before total %d, filtered %d, cost %d ms",
		len(elementTables), total, filtered, time.Since(start).Milliseconds())
}

//// QueryRelations 实现查询接口
//func (b *Indexer) QueryRelations(ctx context.Context, opts *types.QueryRelationOptions) ([]*types.GraphNode, error) {
//
//	filePath := opts.FilePath
//	// TODO projectUuid
//	project, err := b.workspaceReader.GetProjectByFilePath(ctx, opts.Workspace, filePath)
//	if err != nil {
//		return nil, fmt.Errorf("failed to get project by file: %s, err: %w", filePath, err)
//	}
//	projectUuid := project.Uuid
//
//	language, err := lang.InferLanguage(filePath)
//	if err != nil {
//		return nil, lang.ErrUnSupportedLanguage
//	}
//
//	exists, err := b.storage.ExistsProject(projectUuid)
//	if err != nil {
//		return nil, fmt.Errorf("failed to check project %s index store, err:%v", projectUuid, err)
//	}
//	if !exists {
//		return nil, fmt.Errorf("project %s index not exists", projectUuid)
//	}
//
//	startTime := time.Now()
//	defer func() {
//		b.logger.Info("QueryRelations execution time: %d ms", time.Since(startTime).Milliseconds())
//	}()
//
//	// 1. 获取文档
//	var fileElementTable codegraphpb.FileElementTable
//	fileTableBytes, err := b.storage.Get(ctx, projectUuid, store.ElementPathKey{Language: language, Path: filePath})
//	if err != nil {
//		return nil, fmt.Errorf("failed to get file %s element_table, err: %v", filePath, err)
//	}
//	if err = store.UnmarshalValue(fileTableBytes, &fileElementTable); err != nil {
//		return nil, fmt.Errorf("failed to unmarshal file %s element_table value, err: %v", filePath, err)
//	}
//
//	if err != nil {
//		b.logger.Error("Failed to get document: %v", err)
//		return nil, err
//	}
//
//	var res []*types.GraphNode
//	var foundSymbols []*codegraphpb.Element
//
//	// Find root symbols based on query options
//	if opts.SymbolName != types.EmptyString {
//		foundSymbols = b.querySymbolsByNameAndLine(&fileElementTable, opts)
//		b.logger.Debug("Found %d symbols by name and line", len(foundSymbols))
//	} else {
//		foundSymbols = b.querySymbolsByPosition(ctx, &fileElementTable, opts)
//		b.logger.Debug("Found %d symbols by position", len(foundSymbols))
//	}
//
//	// Check if any root symbols were found
//	if len(foundSymbols) == 0 {
//		return nil, fmt.Errorf("symbol not found: name %s startLine %d in document %s", opts.SymbolName, opts.StartLine, opts.FilePath)
//	}
//
//	// root
//	// 找定义节点，以定义节点为根节点进行深度遍历
//	for _, s := range foundSymbols {
//		// 如果当前Symbol 就是定义，加入
//		if s.IsDefinition {
//			res = append(res, &types.GraphNode{
//				FilePath:   fileElementTable.Path,
//				SymbolName: s.Name,
//				Position:   types.ToPosition(s.Range),
//				NodeType:   string(types.NodeTypeDefinition),
//			})
//			continue
//		}
//		// 不是定义节点，找它的relation中的定义节点
//		relations := s.Relations
//		if len(relations) == 0 {
//			continue
//		}
//		for _, r := range relations {
//			if r.RelationType == codegraphpb.RelationType_RELATION_TYPE_DEFINITION {
//				// 定义节点，加入root
//				res = append(res, &types.GraphNode{
//					FilePath:   r.ElementPath,
//					SymbolName: r.ElementName,
//					Position:   types.ToPosition(r.Range),
//					NodeType:   string(types.NodeTypeDefinition),
//				})
//			}
//		}
//	}
//
//	b.logger.Debug("Found %d root nodes", len(res))
//
//	// Build the rest of the tree recursively
//	// We need to build children for the root nodes found
//	for _, rootNode := range res {
//		// Pass the corresponding original symbol proto to the recursive function
//		b.buildChildrenRecursive(ctx, projectUuid, language, rootNode, opts.MaxLayer)
//	}
//	return res, nil
//}

// querySymbolsByPosition 按位置查询 occurrence
func (i *Indexer) querySymbolsByPosition(ctx context.Context, fileTable *codegraphpb.FileElementTable,
	opts *types.QueryRelationOptions) []*codegraphpb.Element {
	var nodes []*codegraphpb.Element
	if opts.StartLine <= 0 || opts.EndLine < opts.StartLine {
		i.logger.Debug("query_symbol_by_position invalid opts startLine %d or endLine %d", opts.StartLine,
			opts.EndLine)
		return nodes
	}
	startLineRange, endLineRange := int32(opts.StartLine)-1, int32(opts.EndLine)-1

	for _, s := range fileTable.Elements {
		if !isValidRange(s.Range) {
			i.logger.Debug("query_symbol_by_position invalid element %s %s position %v", fileTable.Path, s.Name, s.Range)
			continue
		}
		if startLineRange >= s.Range[0] && endLineRange <= s.Range[2] {
			nodes = append(nodes, s)
		}

	}
	return nodes
}

func isValidRange(range_ []int32) bool {
	return len(range_) == 4
}

// querySymbolsByNameAndLine 通过 symbolName + startLine
func (i *Indexer) querySymbolsByNameAndLine(doc *codegraphpb.FileElementTable, opts *types.QueryRelationOptions) []*codegraphpb.Element {
	var nodes []*codegraphpb.Element
	queryName := opts.SymbolName
	// 根据名字和 行号， 找到symbol
	for _, s := range doc.Elements {
		// symbol 名字 模糊匹配
		if strings.Contains(s.Name, queryName) {
			symbolRange := s.Range
			if symbolRange != nil && len(symbolRange) > 0 {
				if symbolRange[0] == int32(opts.StartLine-1) {
					nodes = append(nodes, s)
				}
			}
		}
	}
	return nodes
}

//
//// buildChildrenRecursive recursively builds the child nodes for a given GraphNode and its corresponding Symbol.
//func (i *Indexer) buildChildrenRecursive(ctx context.Context, projectUuid string,
//	language lang.Language, node *types.GraphNode, maxLayer int) {
//	if maxLayer <= 0 || node == nil {
//		i.logger.Debug("buildChildrenRecursive stopped: maxLayer=%d, node is nil=%v", maxLayer, node == nil)
//		return
//	}
//	maxLayer-- // 防止死递归
//
//	startTime := time.Now()
//	defer func() {
//		i.logger.Debug("buildChildrenRecursive for node %s took %v", node.SymbolName, time.Since(startTime))
//	}()
//
//	symbolPath := node.FilePath
//	symbolName := node.SymbolName
//	position := node.Position
//
//	// 根据path和position，定义到 symbol，从而找到它的relation
//	var elementTable codegraphpb.FileElementTable
//	fileTableBytes, err := i.storage.Get(ctx, projectUuid, store.ElementPathKey{
//		Language: language, Path: symbolPath})
//	if err != nil {
//		i.logger.Error("query_definitions failed to get fileTable by def: %s, err: %v", err)
//		return
//	}
//
//	if err = store.UnmarshalValue(fileTableBytes, &elementTable); err != nil {
//		i.logger.Error("Failed to unmarshal fileTable by path: %s, err:%v", symbolPath, err)
//		return
//	}
//
//	if err != nil {
//		i.logger.Debug("Failed to find elementTable for path %s: %v", symbolPath, err)
//		return
//	}
//
//	symbol := i.findSymbolInDocByRange(&elementTable, types.ToRange(position))
//	if symbol == nil {
//		i.logger.Debug("Symbol not found in elementTable: path %s, name %s", symbolPath, symbolName)
//		return
//	}
//
//	var referenceChildren []*types.GraphNode
//
//	// 找到symbol 的relation. 只有定义的symbol 有reference，引用节点的relation是定义节点
//	if len(symbol.Relations) > 0 {
//		for _, r := range symbol.Relations {
//			if r.RelationType == codegraphpb.RelationType_RELATION_TYPE_REFERENCE {
//				// 引用节点，加入node的children
//				referenceChildren = append(referenceChildren, &types.GraphNode{
//					FilePath:   r.ElementPath,
//					SymbolName: r.ElementName,
//					Position:   types.ToPosition(r.Range),
//					NodeType:   string(types.NodeTypeReference),
//				})
//			}
//		}
//		i.logger.Debug("Found %d reference relations for symbol %s", len(referenceChildren), symbolName)
//	}
//
//	if len(referenceChildren) == 0 {
//		// 如果references 为空，说明当前 node 是引用节点， 找到它所属的函数/类/结构体的definition节点，再找引用
//		foundDefSymbol := i.findReferenceSymbolBelonging(&elementTable, symbol)
//		if foundDefSymbol != nil {
//			referenceChildren = append(referenceChildren, &types.GraphNode{
//				FilePath:   elementTable.Path,
//				SymbolName: foundDefSymbol.Name,
//				Position:   types.ToPosition(foundDefSymbol.Range),
//				NodeType:   string(types.NodeTypeDefinition),
//			})
//		} else {
//			i.logger.Debug("failed to find reference symbol %s belongs to", symbolName)
//		}
//
//	}
//
//	//当前节点的子节点
//	node.Children = referenceChildren
//
//	// 继续递归
//	for _, ch := range referenceChildren {
//		i.buildChildrenRecursive(ctx, projectUuid, language, ch, maxLayer)
//	}
//}

func (i *Indexer) QueryDefinitions(ctx context.Context, options *types.QueryDefinitionOptions) ([]*types.Definition, error) {
	filePath := options.FilePath
	project, err := i.workspaceReader.GetProjectByFilePath(ctx, options.Workspace, filePath, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get project by file: %s, err: %w", filePath, err)
	}
	projectUuid := project.Uuid

	language, err := lang.InferLanguage(filePath)
	if err != nil {
		return nil, lang.ErrUnSupportedLanguage
	}

	exists, err := i.storage.ExistsProject(projectUuid)
	if err != nil {
		return nil, fmt.Errorf("failed to check project %s index store, err:%v", projectUuid, err)
	}
	if !exists {
		return nil, fmt.Errorf("project %s index not exists", projectUuid)
	}

	startTime := time.Now()
	defer func() {
		i.logger.Info("query definitions cost %d ms", time.Since(startTime).Milliseconds())
	}()

	res := make([]*types.Definition, 0)

	var foundSymbols []*codegraphpb.Element
	queryStartLine := int32(options.StartLine - 1)
	queryEndLine := int32(options.EndLine - 1)

	snippet := options.CodeSnippet

	var currentImports []*codegraphpb.Import
	// 根据代码片段中的标识符名模糊搜索
	if len(snippet) > 0 {
		// 调用tree_sitter 解析，获取所有的标识符及位置
		parsedData, err := i.parser.Parse(ctx, &types.SourceFile{
			Path:    filePath,
			Content: snippet},
		)
		if err != nil {
			return nil, fmt.Errorf("faled to parse code snippet for definition query: %w", err)
		}
		imports := parsedData.Imports
		elements := parsedData.Elements
		// TODO 找到所有的外部依赖, call、
		var dependencyNames []string
		for _, e := range elements {
			if c, ok := e.(*resolver.Call); ok {
				dependencyNames = append(dependencyNames, c.Name)
			} else if r, ok := e.(*resolver.Reference); ok {
				dependencyNames = append(dependencyNames, r.Name)
			}
		}
		if len(dependencyNames) == 0 {
			return res, nil
		}
		// TODO resolve go modules
		// 对imports预处理
		if filteredImps, err := i.analyzer.PreprocessImports(ctx, language, project, imports); err == nil {
			imports = filteredImps
		}
		for _, imp := range imports {
			currentImports = append(currentImports, &codegraphpb.Import{
				Name:   imp.Name,
				Alias:  imp.Alias,
				Source: imp.Source,
				Range:  imp.Range,
			})
		}

		// 根据所找到的call 的name + currentImports， 去模糊匹配symbol
		symDefs, err := i.searchSymbolNames(ctx, projectUuid, language, dependencyNames, currentImports)
		if err != nil {
			return nil, fmt.Errorf("failed to search function/method call names: %w", err)
		}
		if len(symDefs) == 0 {
			return nil, fmt.Errorf("failed to search symbol by function/method call names")
		}
		for name, def := range symDefs {
			for _, d := range def {
				if d == nil {
					continue
				}
				res = append(res, &types.Definition{
					Name:  name,
					Type:  string(proto.ElementTypeFromProto(d.ElementType)),
					Path:  d.Path,
					Range: d.Range,
				})
			}

		}

	} else {
		// 1. 获取文档
		var fileTable codegraphpb.FileElementTable
		fileTableBytes, err := i.storage.Get(ctx, projectUuid, store.ElementPathKey{Language: language, Path: filePath})
		if errors.Is(err, store.ErrKeyNotFound) {
			return nil, fmt.Errorf("index not found for file %s", filePath)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to get file %s element_table, err: %v", filePath, err)
		}
		if err = store.UnmarshalValue(fileTableBytes, &fileTable); err != nil {
			return nil, fmt.Errorf("failed to unmarshal file %s element_table value, err: %v", filePath, err)
		}

		foundSymbols = i.findSymbolInDocByLineRange(ctx, &fileTable, queryStartLine, queryEndLine)
		currentImports = fileTable.Imports
	}

	// 去重, 如果range 出现过，则剔除
	//existedDefs := make(map[string]bool)
	// 遍历，封装结果，取定义
	for _, s := range foundSymbols {
		// 本身是定义
		if s.IsDefinition {
			// 去掉本范围内的定义，仅过滤范围查询，不过滤全文检索
			if len(s.Range) > 0 && len(snippet) > 0 && isInLinesRange(s.Range[0], queryStartLine, queryEndLine) {
				continue
			}
			res = append(res, &types.Definition{
				Path:  filePath,
				Name:  s.Name,
				Range: s.Range,
				Type:  string(proto.ElementTypeFromProto(s.ElementType)),
			})
			continue
		} else { // 引用
			// 加载定义
			bytes, err := i.storage.Get(ctx, projectUuid, store.SymbolNameKey{Name: s.GetName(),
				Language: language})
			if err == nil {
				var exist codegraphpb.SymbolOccurrence
				if err = store.UnmarshalValue(bytes, &exist); err == nil {
					filtered := i.analyzer.FilterByImports(filePath, currentImports, exist.Occurrences)
					if len(filtered) == 0 {
						// 防止全部过滤掉
						filtered = exist.Occurrences
					}
					for _, o := range filtered {
						res = append(res, &types.Definition{
							Path:  o.Path,
							Name:  s.Name,
							Range: o.Range,
							Type:  string(proto.ToDefinitionElementType(proto.ElementTypeFromProto(s.ElementType))),
						})
					}
				} else {
					i.logger.Debug("query_definitions unmarshal symbol occurrence err:%v", err)
				}

			} else if !errors.Is(err, store.ErrKeyNotFound) {
				i.logger.Debug("query_definitions get symbol occurrence err:%v", err)
			}
		}
	}

	return res, nil
}

const eachSymbolKeepResult = 2

func (i *Indexer) searchSymbolNames(ctx context.Context, projectUuid string, language lang.Language, names []string, imports []*codegraphpb.Import) (
	map[string][]*codegraphpb.Occurrence, error) {

	start := time.Now()
	// 去重
	names = utils.DeDuplicate(names)
	found := make(map[string][]*codegraphpb.Occurrence)

	for _, name := range names {

		bytes, err := i.storage.Get(ctx, projectUuid, store.SymbolNameKey{
			Language: language,
			Name:     name,
		})

		if err != nil {
			continue
		}

		var symbolOccurrence codegraphpb.SymbolOccurrence
		if err := store.UnmarshalValue(bytes, &symbolOccurrence); err != nil {
			return nil, fmt.Errorf("failed to deserialize document: %w", err)
		}

		if len(symbolOccurrence.Occurrences) == 0 {
			continue
		}

		if _, ok := found[name]; !ok {
			found[name] = make([]*codegraphpb.Occurrence, 0)
		}
		found[name] = append(found[name], symbolOccurrence.Occurrences...)
	}

	// TODO 同一个name可能检索出多条数据，再根据import 过滤一遍。 namespace 要么用. 要么用/ 得判断
	total := 0
	for _, v := range found {
		total += len(v)
	}

	if len(imports) > 0 {
		for k, v := range found {
			filtered := make([]*codegraphpb.Occurrence, 0, len(v))
			for _, occ := range v {
				for _, imp := range imports {
					if analyzer.IsImportPathInFilePath(imp, occ.Path) { //TODO , go work ，多模块等特殊情况
						filtered = append(filtered, occ)
						break
					}
				}
			}
			found[k] = filtered
		}
	}

	i.logger.Info("codegraph symbol name search end, cost %d ms, names count: %d, key found:%d",
		time.Since(start).Milliseconds(), len(names), total, len(found))
	return found, nil
}

func (i *Indexer) findSymbolInDocByRange(fileElementTable *codegraphpb.FileElementTable, symbolRange []int32) *codegraphpb.Element {
	//TODO 二分查找
	for _, s := range fileElementTable.Elements {
		// 开始行
		if len(s.Range) < 2 {
			i.logger.Debug("findSymbolInDocByRange invalid range in doc:%s, less than 2: %v", s.Name, s.Range)
			continue
		}
		// 开始行、(TODO 列一致)   这里，当前tree-sitter 捕获的是 整个函数体，而scip则是name，暂时先只通过行号处理（要确保local被过滤）
		if s.Range[0] == symbolRange[0] {
			return s
		}
	}
	return nil
}

func (i *Indexer) findSymbolInDocByLineRange(ctx context.Context,
	fileElementTable *codegraphpb.FileElementTable, startLine int32, endLine int32) []*codegraphpb.Element {
	var res []*codegraphpb.Element
	for _, s := range fileElementTable.Elements {
		// s
		// 开始行
		if len(s.Range) < 2 {
			i.logger.Debug("findSymbolInDocByLineRange invalid range in fileElementTable:%s, less than 2: %v", s.Name, s.Range)
			continue
		}
		if s.Range[0] > endLine {
			break
		}
		// 开始行、(TODO 列一致)   这里，当前tree-sitter 捕获的是 整个函数体，而scip则是name，暂时先只通过行号处理（要确保local被过滤）
		if s.Range[0] >= startLine && s.Range[0] <= endLine {
			res = append(res, s)
		}
	}
	return res
}

func (i *Indexer) findReferenceSymbolBelonging(f *codegraphpb.FileElementTable,
	referenceElement *codegraphpb.Element) *codegraphpb.Element {
	if len(referenceElement.GetRange()) < 3 {
		i.logger.Debug("find_symbol_belong %s invalid referenceElement range %s %s %v",
			f.Path, referenceElement.Name, referenceElement.Range)
		return nil
	}
	for _, e := range f.Elements {
		if !e.IsDefinition {
			continue
		}
		if len(e.GetRange()) < 3 {
			i.logger.Debug("find_symbol_belong invalid range %s %s %v", f.Path, e.Name, e.Range)
			continue
		}
		// 判断行
		if referenceElement.Range[0] > e.Range[0] && referenceElement.Range[0] < e.Range[2] {
			return e
		}
	}
	return nil
}

func (i *Indexer) GetSummary(ctx context.Context, workspacePath string) (*types.CodeGraphSummary, error) {
	projects := i.workspaceReader.FindProjects(ctx, workspacePath, false, workspace.DefaultVisitPattern)
	if len(projects) == 0 {
		return nil, fmt.Errorf("found no projects in workspace %s", workspacePath)
	}
	summary := new(types.CodeGraphSummary)
	for _, p := range projects {
		summary.TotalFiles += i.storage.Size(ctx, p.Uuid, store.PathKeySystemPrefix)
	}
	return summary, nil
}

func (i *Indexer) RemoveAllIndexes(ctx context.Context, workspacePath string) error {
	projects := i.workspaceReader.FindProjects(ctx, workspacePath, false, workspace.DefaultVisitPattern)
	if len(projects) == 0 {
		i.logger.Info("remove_all_index found no projects in workspace %s", workspacePath)
		return nil
	}
	var errs []error
	for _, p := range projects {
		errs = append(errs, i.storage.DeleteAll(ctx, p.Uuid))
	}
	return errors.Join(errs...)
}

// RenameIndexes 重命名索引，根据路径（文件或文件夹）
func (i *Indexer) RenameIndexes(ctx context.Context, workspacePath string, sourceFilePath string, targetFilePath string) error {
	//TODO 查出来source，删除、重命名相关path、写入，更新symbol中指向source的路径为target（迭代式进行）
	projects := i.workspaceReader.FindProjects(ctx, workspacePath, false, workspace.DefaultVisitPattern)
	if len(projects) == 0 {
		i.logger.Info("rename_index found no projects in workspace %s", workspacePath)
		return nil
	}
	sourceProject, err := i.workspaceReader.GetProjectByFilePath(ctx, workspacePath, sourceFilePath, false)
	if err != nil {
		return fmt.Errorf("rename_index cannot find project for file %s, err:%v", sourceFilePath, err)
	}
	targetProject, err := i.workspaceReader.GetProjectByFilePath(ctx, workspacePath, targetFilePath, false)
	if err != nil {
		return fmt.Errorf("rename_index cannot find project for file %s, err:%v", targetFilePath, err)
	}
	sourceProjectUuid, targetProjectUuid := sourceProject.Uuid, targetProject.Uuid
	// 可能是文件，也可能是目录
	sourceTables, err := i.searchFileElementTablesByPath(ctx, sourceProjectUuid, []string{sourceFilePath})
	if err != nil {
		return fmt.Errorf("rename_index search source element tables by path %s err:%v", sourceFilePath, err)
	}
	if len(sourceTables) == 0 {
		i.logger.Debug("rename_index found no index by source path %s", sourceFilePath)
		return nil
	}
	// 统一去掉最后的分隔符（如果有），防止一个有，另一个没有
	trimedSourcePath, trimedTargetPath := utils.TrimLastSeparator(sourceFilePath), utils.TrimLastSeparator(targetFilePath)
	// 将source删除、key重命名为target，更新source相关的symbol 为target
	for _, st := range sourceTables {
		oldPath := st.Path
		oldLanguage := st.Language
		// 删除
		if err = i.storage.Delete(ctx, sourceProjectUuid, store.ElementPathKey{Language: lang.Language(st.Language), Path: st.Path}); err != nil {
			i.logger.Debug("rename_index delete index %s %s err:%v", st.Language, st.Path, err)
		}
		// 将path中 sourceFilePath 重命名为targetFilePath，
		newPath := strings.ReplaceAll(st.Path, trimedSourcePath, trimedTargetPath)
		newLanguage, err := lang.InferLanguage(newPath)
		if err != nil {
			i.logger.Debug("rename_index unsupported language for new path %s", newPath)
			// TODO 删除symbol 中的source_path、reference中的指向它的relation
			continue
		}
		st.Path = newPath
		st.Language = string(newLanguage)
		// 保存target
		if err = i.storage.Put(ctx, targetProjectUuid, &store.Entry{Key: store.ElementPathKey{
			Language: newLanguage, Path: newPath}, Value: st}); err != nil {
			i.logger.Debug("rename_index save new index %s err:%v ", newPath, err)
		}

		// 更新符号定义，找到相关符号，将它的path由old改为new
		for _, e := range st.Elements {
			if !e.IsDefinition {
				continue
			}
			SymbolOccurrence, err := i.getSymbolOccurrenceByName(ctx, sourceProjectUuid, lang.Language(oldLanguage), e.Name)
			if err != nil {
				i.logger.Debug("rename_index get symbol definition by name %s %s err:%v", oldLanguage, e.Name, err)
				continue
			}

			// 语言相同则更新，语言不同，则删除新增
			sameLanguage := oldLanguage == string(newLanguage)
			definitions := make([]*codegraphpb.Occurrence, 0, len(SymbolOccurrence.Occurrences))
			for _, d := range SymbolOccurrence.Occurrences {
				if d.Path == oldPath {
					if sameLanguage {
						d.Path = newPath
					}
				} else {
					definitions = append(definitions, d)
				}
			}
			// 保存
			if err = i.storage.Put(ctx, sourceProjectUuid, &store.Entry{
				Key: store.SymbolNameKey{
					Language: lang.Language(SymbolOccurrence.Language),
					Name:     SymbolOccurrence.Name},
				Value: SymbolOccurrence,
			}); err != nil {
				i.logger.Debug("rename_index save SymbolOccurrence %s err:%v", e.Name, err)
			}
			// 不同语言，保存新的
			if !sameLanguage {
				newSymbolDefinition := &codegraphpb.SymbolOccurrence{
					Name:     e.Name,
					Language: string(newLanguage),
					Occurrences: []*codegraphpb.Occurrence{
						{
							Path:        newPath,
							Range:       e.Range,
							ElementType: e.ElementType,
						},
					},
				}
				if err = i.storage.Put(ctx, sourceProjectUuid, &store.Entry{
					Key: store.SymbolNameKey{
						Language: lang.Language(SymbolOccurrence.Language),
						Name:     SymbolOccurrence.Name},
					Value: newSymbolDefinition,
				}); err != nil {
					i.logger.Debug("rename_index save SymbolOccurrence %s err:%v", e.Name, err)
				}
			}

		}

	}

	return nil
}

// getFileElementTableByPath 通过路径获取FileElementTable
func (i *Indexer) getFileElementTableByPath(ctx context.Context, projectUuid string, filePath string) (*codegraphpb.FileElementTable, error) {
	language, err := lang.InferLanguage(filePath)
	if err != nil {
		return nil, err
	}
	bytes, err := i.storage.Get(ctx, projectUuid, store.ElementPathKey{Language: language, Path: filePath})
	if err != nil {
		return nil, err
	}
	var referTable codegraphpb.FileElementTable
	err = store.UnmarshalValue(bytes, &referTable)
	return &referTable, err
}

// getFileElementTableByPath 通过路径获取FileElementTable
func (i *Indexer) getSymbolOccurrenceByName(ctx context.Context, projectUuid string,
	language lang.Language, symbolName string) (*codegraphpb.SymbolOccurrence, error) {

	bytes, err := i.storage.Get(ctx, projectUuid, store.SymbolNameKey{Language: language, Name: symbolName})
	if err != nil {
		return nil, err
	}
	var SymbolOccurrence codegraphpb.SymbolOccurrence
	err = store.UnmarshalValue(bytes, &SymbolOccurrence)
	return &SymbolOccurrence, err
}

// parseFilesOptimized 优化版本的文件解析函数，减少内存分配
func (i *Indexer) parseFiles(ctx context.Context, filePaths []string) ([]*parser.FileElementTable, *types.IndexTaskMetrics, error) {
	totalFiles := len(filePaths)

	// 优化：预分配切片容量，减少动态扩容
	fileElementTables := make([]*parser.FileElementTable, 0, totalFiles)
	projectTaskMetrics := &types.IndexTaskMetrics{
		TotalFiles:      totalFiles,
		FailedFilePaths: make([]string, 0, totalFiles/4), // 预估失败文件数约为文件数的25%
	}

	// TODO 大量文件，会导致错误很多
	var errs []error

	for _, path := range filePaths {
		language, err := lang.InferLanguage(path)
		if err != nil || language == types.EmptyString {
			// not supported language or not source file
			projectTaskMetrics.TotalFailedFiles++
			projectTaskMetrics.FailedFilePaths = append(projectTaskMetrics.FailedFilePaths, path)
			continue
		}

		// 直接读取文件并解析，避免不必要的中间变量
		content, err := i.workspaceReader.ReadFile(ctx, path, types.ReadOptions{})
		if err != nil {
			projectTaskMetrics.TotalFailedFiles++
			projectTaskMetrics.FailedFilePaths = append(projectTaskMetrics.FailedFilePaths, path)
			i.logger.Debug("index_project read file %s err:%v", path, err)
			continue
		}

		// 创建源文件对象并解析
		sourceFile := &types.SourceFile{
			Path:    path,
			Content: content,
		}

		fileElementTable, err := i.parser.Parse(ctx, sourceFile)

		if err != nil {
			projectTaskMetrics.TotalFailedFiles++
			projectTaskMetrics.FailedFilePaths = append(projectTaskMetrics.FailedFilePaths, path)
			i.logger.Debug("index_project parse file %s err:%v", path, err)
			// 显式清理内存
			content = nil
			sourceFile.Content = nil
			continue
		}

		// 显式清理内存，帮助GC及时回收
		content = nil
		sourceFile.Content = nil

		fileElementTables = append(fileElementTables, fileElementTable)
	}

	return fileElementTables, projectTaskMetrics, errors.Join(errs...)
}

func isInLinesRange(current, start, end int32) bool {
	return current >= start-1 && current <= end-1
}

func isSymbolExists(filePath string, ranges []int32, state map[string]bool) bool {
	key := symbolMapKey(filePath, ranges)
	_, ok := state[key]
	return ok
}
func symbolMapKey(filePath string, ranges []int32) string {
	return filePath + "-" + utils.SliceToString(ranges)
}

// processBatch 处理单个批次的文件
func (i *Indexer) processBatch(ctx context.Context, batchId int, params *BatchProcessParams) (*BatchProcessResult, error) {
	batchStartTime := time.Now()

	i.logger.Debug("index_project task-%d start, batch [%d:%d]/%d, batch_size %d",
		batchId, params.BatchStart, params.BatchEnd, params.TotalFiles, params.BatchSize)

	// 解析文件
	elementTables, metrics, err := i.parseFiles(ctx, params.SourceFiles)
	if err != nil {
		return nil, fmt.Errorf("parse files failed: %w", err)
	}
	elementTablesCnt := len(elementTables)
	if len(elementTables) == 0 {
		return &BatchProcessResult{
			ElementTablesCnt: elementTablesCnt,
			Metrics:          metrics,
			Duration:         time.Since(batchStartTime),
		}, nil
	}

	i.logger.Debug("index_project task-%d batch [%d:%d]/%d parse files end, cost %d ms", batchId,
		params.BatchStart, params.BatchEnd, params.TotalFiles, time.Since(batchStartTime).Milliseconds())

	// 检查elements，剔除不合法的，并打印日志
	i.checkElementTables(elementTables)

	// 项目符号表存储
	symbolStart := time.Now()

	if err = i.analyzer.SaveSymbolOccurrences(ctx, params.ProjectUuid, elementTables); err != nil {
		return nil, fmt.Errorf("save symbol definitions failed: %w", err)
	}

	i.logger.Debug("index_project task-%d batch [%d:%d]/%d save symbols end, cost %d ms", batchId,
		params.BatchStart, params.BatchEnd, params.TotalFiles, time.Since(symbolStart).Milliseconds())

	// 预处理 import
	if err := i.preprocessImports(ctx, elementTables, params.Project); err != nil {
		i.logger.Debug("index_project task-%d preprocess import error: %v", utils.TruncateError(err))
	}

	// element存储，后面依赖分析，基于磁盘，避免大型项目占用太多内存
	protoElementTables := proto.FileElementTablesToProto(elementTables)
	batchSaveStart := time.Now()
	// 关系索引存储
	if err = i.storage.BatchSave(ctx, params.ProjectUuid, workspace.FileElementTables(protoElementTables)); err != nil {
		metrics.TotalFailedFiles += params.BatchSize
		metrics.FailedFilePaths = append(metrics.FailedFilePaths, params.SourceFiles...)
		return nil, fmt.Errorf("batch save element tables failed: %w", err)
	}

	// 优化：显式清理内存，帮助GC及时回收
	elementTables = nil
	protoElementTables = nil

	i.logger.Debug("index_project task-%d batch [%d:%d]/%d save element_tables end, cost %d ms", batchId,
		params.BatchStart, params.BatchEnd, params.TotalFiles, time.Since(batchSaveStart).Milliseconds())

	i.logger.Debug("index_project task-%d batch [%d:%d]/%d end, cost %d ms", batchId,
		params.BatchStart, params.BatchEnd, params.TotalFiles, time.Since(batchStartTime).Milliseconds())

	return &BatchProcessResult{
		ElementTablesCnt: elementTablesCnt,
		Metrics:          metrics,
		Duration:         time.Since(batchStartTime),
	}, nil
}

// collectSourceFiles 收集源文件
func (i *Indexer) collectSourceFiles(ctx context.Context, projectPath string, maxFiles int) ([]string, error) {
	startTime := time.Now()
	filePaths := make([]string, 0, 500)

	err := i.workspaceReader.WalkFile(ctx, projectPath, func(walkCtx *types.WalkContext) error {
		if walkCtx.Info.IsDir {
			return nil
		}
		if len(filePaths) >= maxFiles {
			i.logger.Debug("collect_source_files max files %d reached, return.", maxFiles)
			return filepath.SkipAll
		}
		filePaths = append(filePaths, walkCtx.Path)
		return nil
	}, types.WalkOptions{IgnoreError: true, VisitPattern: i.config.VisitPattern})

	if err != nil {
		return nil, err
	}

	i.logger.Debug("collect_source_files finished, collected %d files, cost %d ms",
		len(filePaths), time.Since(startTime).Milliseconds())

	return filePaths, nil
}

// processParseInBatches 批量处理文件
func (i *Indexer) processParseInBatches(ctx context.Context, params *BatchProcessingParams) (*BatchProcessingResult, error) {
	startTime := time.Now()
	totalFiles := len(params.SourceFiles)

	// 基于文件数量预分配切片容量，优化内存使用
	var errs []error

	projectMetrics := &types.IndexTaskMetrics{
		TotalFiles:      totalFiles,
		FailedFilePaths: make([]string, 0, totalFiles/4), // 预估失败文件数约为文件数的25%
	}

	var processedFilesCnt int

	var batchId int
	// 处理批次
	for m := 0; m < totalFiles; {
		batch := utils.Min(totalFiles-m, params.BatchSize)
		batchStart, batchEnd := m, m+batch
		sourceFilesBatch := params.SourceFiles[batchStart:batchEnd]
		batchId++
		// 构建批处理参数
		batchParams := &BatchProcessParams{
			ProjectUuid: params.ProjectUuid,
			SourceFiles: sourceFilesBatch,
			BatchStart:  batchStart,
			BatchEnd:    batchEnd,
			BatchSize:   batch,
			TotalFiles:  totalFiles,
			Project:     params.Project,
		}

		// 提交任务
		err := func(ctx context.Context, taskID int) error {
			batchStartTime := time.Now()
			result, err := i.processBatch(ctx, taskID, batchParams)
			if err != nil {
				i.logger.Debug("index_project process batch err:%v", err)
				return fmt.Errorf("process batch err:%w", err)
			}

			processedFilesCnt += result.ElementTablesCnt
			projectMetrics.TotalFailedFiles += result.Metrics.TotalFailedFiles
			projectMetrics.FailedFilePaths = append(projectMetrics.FailedFilePaths, result.Metrics.FailedFilePaths...)
			//TODO 更新进度
			batchUpdateStart := time.Now()
			if err := i.updateProgress(ctx, &ProgressInfo{
				Total:         totalFiles,
				Processed:     processedFilesCnt,
				PreviousNum:   params.PreviousNum,
				WorkspacePath: params.WorkspacePath,
			}); err != nil {
				return fmt.Errorf("update progress failed: %w", err)
			}

			i.logger.Debug("update workspace %s successful, file num %d/%d, cost %d ms, batch %d cost %d ms",
				params.WorkspacePath, processedFilesCnt+params.PreviousNum,
				processedFilesCnt, time.Since(batchUpdateStart).Milliseconds(), batch,
				time.Since(batchStartTime).Milliseconds())
			return nil
		}(ctx, batchId)
		if err != nil {
			i.logger.Debug("index_project submit task err:%v", err)
		}

		m += batch
	}

	// 最终更新进度
	// 最终更新
	if err := i.updateProgress(ctx, &ProgressInfo{
		Total:         totalFiles,
		Processed:     processedFilesCnt,
		PreviousNum:   params.PreviousNum,
		WorkspacePath: params.WorkspacePath,
	}); err != nil {
		i.logger.Debug("index_project update progress failed: %v", err)
	}

	return &BatchProcessingResult{
		ParsedFilesCount: processedFilesCnt,
		ProjectMetrics:   projectMetrics,
		Errors:           errs,
		Duration:         time.Since(startTime),
	}, nil
}

// updateProgress 更新进度
func (i *Indexer) updateProgress(ctx context.Context, progress *ProgressInfo) error {

	if err := i.workspaceRepository.UpdateCodegraphInfo(progress.WorkspacePath,
		progress.Total+progress.PreviousNum, time.Now().Unix()); err != nil {
		i.logger.Error("update workspace %s codegraph successful file num %d/%d, err:%v",
			progress.WorkspacePath, progress.Total, progress.Processed, err)
		return err
	}

	return nil
}

func (i *Indexer) IndexIter(ctx context.Context, projectUuid string) store.Iterator {
	return i.storage.Iter(ctx, projectUuid)
}
