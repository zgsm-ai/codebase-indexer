package codegraph

import (
	"codebase-indexer/internal/repository"
	"codebase-indexer/pkg/codegraph/analyzer"
	"codebase-indexer/pkg/codegraph/lang"
	"codebase-indexer/pkg/codegraph/parser"
	"codebase-indexer/pkg/codegraph/pool"
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
	"strings"
	"sync"
	"time"
)

type IndexerConfig struct {
	MaxConcurrency int
	BatchSize      int
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

const (
	defaultConcurrency = 2
	defaultBatchSize   = 10
)

// TODO 多子项目、多语言支持。monorepo, 不光通过语言隔离、还要通过子项目隔离。

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
	if config.BatchSize <= 0 {
		config.BatchSize = defaultBatchSize
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
	projects := i.workspaceReader.FindProjects(ctx, workspacePath, workspace.DefaultVisitPattern)
	if len(projects) == 0 {
		return taskMetrics, fmt.Errorf("index_workspace find no projects in workspace: %s", workspacePath)
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
	// todo 并发、批量处理
	// 阶段0：收集要处理的源码文件（可并发）
	// 阶段1：解析 （可并发）
	// 阶段2：检查过滤元素 （可并发）
	// 阶段3：保存符号表 （可并发）
	// 阶段4：依赖分析 （依赖元素的具体类型）
	// 阶段5：保存索引
	var errs []error
	projectMetrics := &types.IndexTaskMetrics{FailedFilePaths: make([]string, 0)}
	projectUuid := project.Uuid
	projectStart := time.Now()
	i.logger.Info("index_project start to index project：%s", project.Path)

	workspaceModel, err := i.workspaceRepository.GetWorkspaceByPath(workspacePath)
	if err != nil {
		return nil, append(errs, err)
	}
	if workspaceModel == nil {
		return nil, append(errs, fmt.Errorf("index_project workspace %s not found in database", workspacePath))
	}
	previousNum := workspaceModel.CodegraphFileNum

	sourceFiles, err := i.CollectProjectSourceFiles(ctx, project.Path, i.config.VisitPattern)
	if err != nil {
		return projectMetrics, append(errs, fmt.Errorf("index_project collect project files err:%v", err))
	}
	totalFiles := len(sourceFiles)
	if totalFiles == 0 {
		i.logger.Info("index_project found no source files in project %s, not index.", project.Path)
		return projectMetrics, nil
	}
	projectMetrics.TotalFiles = totalFiles
	i.logger.Info("index_project collect project files finish. cost %d ms, found %d source files to index",
		time.Since(projectStart).Milliseconds(), totalFiles)

	taskPool := pool.NewTaskPool(i.config.MaxConcurrency, i.logger)

	defer taskPool.Close()

	var mux sync.Mutex
	var parsedFilesCnt int
	for m := 0; m < totalFiles; {
		batch := utils.Min(totalFiles-m, i.config.BatchSize)
		batchStart, batchEnd := m, m+batch
		err := taskPool.Submit(ctx, func(ctx context.Context, taskId uint64) {
			batchStartTime := time.Now()
			i.logger.Debug("index_project task-%d start, batch [%d:%d]/%d, batch_size %d",
				taskId, batchStart, batchEnd, totalFiles, batch)
			// 解析
			elementTables, err := func() ([]*parser.FileElementTable, error) {
				elementTables, metrics, err := i.parseFiles(ctx, sourceFiles[batchStart:batchEnd])
				mux.Lock()
				defer mux.Unlock()
				parsedFilesCnt += len(elementTables)
				projectMetrics.TotalFailedFiles += metrics.TotalFailedFiles
				projectMetrics.FailedFilePaths = append(projectMetrics.FailedFilePaths, metrics.FailedFilePaths...)
				return elementTables, err
			}()
			if len(elementTables) == 0 {
				return
			}
			i.logger.Debug("index_project task-%d batch [%d:%d]/%d parse files end, cost %d ms", taskId,
				batchStart, batchEnd, totalFiles, time.Since(batchStartTime).Milliseconds())
			// 检查elements，剔除不合法的，并打印日志
			i.checkElementTables(elementTables)

			symbolStart := time.Now()
			// 项目符号表存储
			if err = i.analyzer.SaveSymbolDefinitions(ctx, projectUuid, elementTables); err != nil {
				errs = append(errs, err)
			}
			i.logger.Debug("index_project task-%d batch [%d:%d]/%d save symbols end, cost %d ms", taskId,
				batchStart, batchEnd, totalFiles, time.Since(symbolStart).Milliseconds())

			// 预处理 import
			if err := i.preprocessImports(ctx, elementTables, project); err != nil {
				i.logger.Debug("index_project task-%d preprocess import error: %v", utils.TruncateError(err))
			}

			// element存储，后面依赖分析，基于磁盘，避免大型项目占用太多内存。
			protoElementTables := proto.FileElementTablesToProto(elementTables)
			// 关系索引存储
			if err = i.storage.BatchSave(ctx, projectUuid, workspace.FileElementTables(protoElementTables)); err != nil {
				errs = append(errs, err)
				projectMetrics.TotalFailedFiles += batch
				projectMetrics.FailedFilePaths = append(projectMetrics.FailedFilePaths, sourceFiles[batchStart:batchEnd]...)
			}
			i.logger.Debug("index_project task-%d batch [%d:%d]/%d save element_tables end, cost %d ms", taskId,
				batchStart, batchEnd, totalFiles, time.Since(batchStartTime).Milliseconds())

			i.logger.Debug("index_project task-%d batch [%d:%d]/%d end, cost %d ms", taskId,
				batchStart, batchEnd, totalFiles, time.Since(batchStartTime).Milliseconds())

		})
		if err != nil {
			errs = append(errs, err)
		}
		m += batch
	}
	// 等待所有文件处理完成，进行依赖分析
	taskPool.Wait()

	i.logger.Info("index_project project %s files parse finish. cost %d ms, visit %d files, "+
		"parsed %d files successfully, failed %d files",
		project.Path, time.Since(projectStart).Milliseconds(), projectMetrics.TotalFiles,
		projectMetrics.TotalFiles-projectMetrics.TotalFailedFiles, projectMetrics.TotalFailedFiles)

	if projectMetrics.TotalFiles-projectMetrics.TotalFailedFiles == 0 {
		i.logger.Info("index_project project %s parse and save 0 element_tables successfully, process end.",
			project.Path)
		return projectMetrics, errs
	}

	analyzeStart := time.Now()

	// 更新进度，更新10次
	// TODO 优化为由快到慢
	updateTimes := 10
	updateBatch := (parsedFilesCnt + updateTimes - 1) / updateTimes

	elementTablesCnt := 0
	// 迭代进行依赖分析
	iter := i.storage.Iter(ctx, projectUuid)
	defer iter.Close()
	for iter.Next() {
		// 遍历 element_table
		if !store.IsElementPathKey(iter.Key()) {
			continue
		}
		var elementTable codegraphpb.FileElementTable
		if err = store.UnmarshalValue(iter.Value(), &elementTable); err != nil {
			errs = append(errs, fmt.Errorf("index_project unmarshal element_table err:%w", err))
			continue
		}

		// 依赖分析
		if err = i.analyzer.Analyze(ctx, projectUuid, []*codegraphpb.FileElementTable{&elementTable}); err != nil {
			errs = append(errs, err)
			return projectMetrics, errs
		}

		// TODO 没发生更新，则不需要存储
		if err = i.storage.BatchSave(ctx, projectUuid,
			workspace.FileElementTables([]*codegraphpb.FileElementTable{&elementTable})); err != nil {
			projectMetrics.TotalFailedFiles++
			projectMetrics.FailedFilePaths = append(projectMetrics.FailedFilePaths, elementTable.Path)
			errs = append(errs, err)
			return projectMetrics, errs
		}
		if elementTablesCnt > 0 && elementTablesCnt%updateBatch == 0 {
			updateStart := time.Now()
			if err = i.workspaceRepository.UpdateCodegraphInfo(workspacePath, elementTablesCnt+previousNum,
				time.Now().Unix()); err != nil {
				i.logger.Error("index_project update workspace %s codegraph successful file num %d/%d, err:%v",
					workspacePath, elementTablesCnt, parsedFilesCnt, err)
			}
			i.logger.Debug("index_project start to updated workspace %s successful, file num %d/%d, update_batch %d, cost %d ms",
				workspacePath, elementTablesCnt+previousNum, parsedFilesCnt, updateBatch, time.Since(updateStart).Milliseconds())
		}
		elementTablesCnt++
	}

	// 最终更新
	if err = i.workspaceRepository.UpdateCodegraphInfo(workspacePath, elementTablesCnt+previousNum,
		time.Now().Unix()); err != nil {
		i.logger.Error("index_project update workspace %s codegraph successful file num %d/%d, err:%v",
			workspacePath, elementTablesCnt, parsedFilesCnt, err)
	}

	i.logger.Info("index_project project %s analyze dependency end, analyzed %d/%d element tables, cost %d ms.",
		project.Path, elementTablesCnt, parsedFilesCnt, time.Since(analyzeStart).Milliseconds())

	i.logger.Info("index_project: finish %s, cost %d ms, processed %d files.", project.Path,
		time.Since(projectStart).Milliseconds(), elementTablesCnt)

	return projectMetrics, nil
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

	projects := i.workspaceReader.FindProjects(ctx, workspacePath, workspace.DefaultVisitPattern)
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
	// 2. 收集关系
	referredElements := i.collectRelations(deleteFileTables)

	// 3. 清理关系
	if err = i.cleanupRelations(ctx, projectUuid, referredElements, deletePaths); err != nil {
		return fmt.Errorf("cleanup references failed: %w", err)
	}

	// 4. 清理符号定义
	if err = i.cleanupSymbolDefinitions(ctx, projectUuid, deleteFileTables, deletePaths); err != nil {
		return fmt.Errorf("cleanup symbol definitions failed: %w", err)
	}

	// 5. 删除path索引
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

// collectRelations 收集引用元素
func (i *Indexer) collectRelations(fileTables []*codegraphpb.FileElementTable) []*codegraphpb.Relation {
	var relations []*codegraphpb.Relation

	for _, ft := range fileTables {
		for _, e := range ft.Elements {
			if len(e.GetRelations()) == 0 {
				continue
			}
			relations = append(relations, e.Relations...)
		}
	}

	return relations
}

// cleanupRelations 清理引用关系
func (i *Indexer) cleanupRelations(ctx context.Context, projectUuid string,
	relations []*codegraphpb.Relation, deletedPaths map[string]interface{}) error {
	var errs []error

	for _, ref := range relations {
		language, err := lang.InferLanguage(ref.ElementPath)
		if err != nil {
			continue
		}
		key := store.ElementPathKey{Language: language, Path: ref.ElementPath}

		refTable, err := i.getFileElementTableByPath(ctx, projectUuid, ref.ElementPath)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		// 移除与该符号相关的relation
		for _, e := range refTable.Elements {
			var newRelations []*codegraphpb.Relation
			for _, r := range e.GetRelations() {
				// 如果relation指向待删除的符号，则跳过
				if _, ok := deletedPaths[r.ElementPath]; ok {
					continue
				}
				newRelations = append(newRelations, r)
			}
			e.Relations = newRelations
		}

		key = store.ElementPathKey{Language: language, Path: refTable.Path}
		// 保存更新后的文件表
		if err = i.storage.Put(ctx, projectUuid, &store.Entry{Key: key, Value: refTable}); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// cleanupSymbolDefinitions 清理符号定义
func (i *Indexer) cleanupSymbolDefinitions(ctx context.Context, projectUuid string,
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
				symDefs := new(codegraphpb.SymbolDefinition)
				if err = store.UnmarshalValue(sym, symDefs); err != nil {
					return fmt.Errorf("unmarshal symbolDefinition error:%w", err)
				}

				newSymDefs := &codegraphpb.SymbolDefinition{
					Name:        e.GetName(),
					Definitions: make([]*codegraphpb.Definition, 0),
				}
				for _, d := range symDefs.Definitions {
					if _, ok := deletedPaths[d.Path]; ok {
						continue
					}
					newSymDefs.Definitions = append(newSymDefs.Definitions, d)
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
	projects := i.workspaceReader.FindProjects(ctx, workspacePath, workspace.DefaultVisitPattern)
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
	if err := i.analyzer.SaveSymbolDefinitions(ctx, projectUuid, fileTables); err != nil {
		return fmt.Errorf("save symbol definitions error: %w", err)
	}

	// 转换为 proto
	protoElementTables := proto.FileElementTablesToProto(fileTables)
	// 依赖分析
	if err := i.analyzer.Analyze(ctx, projectUuid, protoElementTables); err != nil {
		return fmt.Errorf("analyze dependency error: %w", err)
	}

	// 关系索引存储
	if err := i.storage.BatchSave(ctx, projectUuid, workspace.FileElementTables(protoElementTables)); err != nil {
		return fmt.Errorf("batch save error: %w", err)
	}

	return nil
}

// QueryElements 查询elements
func (i *Indexer) QueryElements(ctx context.Context, workspacePath string, filePaths []string) ([]*codegraphpb.FileElementTable, error) {
	i.logger.Info("query_elements start to query workspace %s files: %v", workspacePath, filePaths)

	projects := i.workspaceReader.FindProjects(ctx, workspacePath, workspace.DefaultVisitPattern)
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
func (i *Indexer) QuerySymbols(ctx context.Context, workspacePath string, filePath string, symbolNames []string) ([]*codegraphpb.SymbolDefinition, error) {
	i.logger.Info("query_symbols start to query workspace %s file %s symbols: %v", workspacePath, filePath, symbolNames)

	projects := i.workspaceReader.FindProjects(ctx, workspacePath, workspace.DefaultVisitPattern)
	if len(projects) == 0 {
		return nil, fmt.Errorf("no project found in workspace %s", workspacePath)
	}

	var results []*codegraphpb.SymbolDefinition
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
			sd := new(codegraphpb.SymbolDefinition)
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

// QueryRelations 实现查询接口
func (b *Indexer) QueryRelations(ctx context.Context, opts *types.QueryRelationOptions) ([]*types.GraphNode, error) {

	filePath := opts.FilePath
	// TODO projectUuid
	project, err := b.workspaceReader.GetProjectByFilePath(ctx, opts.Workspace, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get project by file: %s, err: %w", filePath, err)
	}
	projectUuid := project.Uuid

	language, err := lang.InferLanguage(filePath)
	if err != nil {
		return nil, lang.ErrUnSupportedLanguage
	}

	exists, err := b.storage.ExistsProject(projectUuid)
	if err != nil {
		return nil, fmt.Errorf("failed to check project %s index store, err:%v", projectUuid, err)
	}
	if !exists {
		return nil, fmt.Errorf("project %s index not exists", projectUuid)
	}

	startTime := time.Now()
	defer func() {
		b.logger.Info("QueryRelations execution time: %d ms", time.Since(startTime).Milliseconds())
	}()

	// 1. 获取文档
	var fileElementTable codegraphpb.FileElementTable
	fileTableBytes, err := b.storage.Get(ctx, projectUuid, store.ElementPathKey{Language: language, Path: filePath})
	if err != nil {
		return nil, fmt.Errorf("failed to get file %s element_table, err: %v", filePath, err)
	}
	if err = store.UnmarshalValue(fileTableBytes, &fileElementTable); err != nil {
		return nil, fmt.Errorf("failed to unmarshal file %s element_table value, err: %v", filePath, err)
	}

	if err != nil {
		b.logger.Error("Failed to get document: %v", err)
		return nil, err
	}

	var res []*types.GraphNode
	var foundSymbols []*codegraphpb.Element

	// Find root symbols based on query options
	if opts.SymbolName != types.EmptyString {
		foundSymbols = b.querySymbolsByNameAndLine(&fileElementTable, opts)
		b.logger.Debug("Found %d symbols by name and line", len(foundSymbols))
	} else {
		foundSymbols = b.querySymbolsByPosition(ctx, &fileElementTable, opts)
		b.logger.Debug("Found %d symbols by position", len(foundSymbols))
	}

	// Check if any root symbols were found
	if len(foundSymbols) == 0 {
		return nil, fmt.Errorf("symbol not found: name %s startLine %d in document %s", opts.SymbolName, opts.StartLine, opts.FilePath)
	}

	// root
	// 找定义节点，以定义节点为根节点进行深度遍历
	for _, s := range foundSymbols {
		// 如果当前Symbol 就是定义，加入
		if s.IsDefinition {
			res = append(res, &types.GraphNode{
				FilePath:   fileElementTable.Path,
				SymbolName: s.Name,
				Position:   types.ToPosition(s.Range),
				NodeType:   string(types.NodeTypeDefinition),
			})
			continue
		}
		// 不是定义节点，找它的relation中的定义节点
		relations := s.Relations
		if len(relations) == 0 {
			continue
		}
		for _, r := range relations {
			if r.RelationType == codegraphpb.RelationType_RELATION_TYPE_DEFINITION {
				// 定义节点，加入root
				res = append(res, &types.GraphNode{
					FilePath:   r.ElementPath,
					SymbolName: r.ElementName,
					Position:   types.ToPosition(r.Range),
					NodeType:   string(types.NodeTypeDefinition),
				})
			}
		}
	}

	b.logger.Debug("Found %d root nodes", len(res))

	// Build the rest of the tree recursively
	// We need to build children for the root nodes found
	for _, rootNode := range res {
		// Pass the corresponding original symbol proto to the recursive function
		b.buildChildrenRecursive(ctx, projectUuid, language, rootNode, opts.MaxLayer)
	}
	return res, nil
}

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

// buildChildrenRecursive recursively builds the child nodes for a given GraphNode and its corresponding Symbol.
func (i *Indexer) buildChildrenRecursive(ctx context.Context, projectUuid string,
	language lang.Language, node *types.GraphNode, maxLayer int) {
	if maxLayer <= 0 || node == nil {
		i.logger.Debug("buildChildrenRecursive stopped: maxLayer=%d, node is nil=%v", maxLayer, node == nil)
		return
	}
	maxLayer-- // 防止死递归

	startTime := time.Now()
	defer func() {
		i.logger.Debug("buildChildrenRecursive for node %s took %v", node.SymbolName, time.Since(startTime))
	}()

	symbolPath := node.FilePath
	symbolName := node.SymbolName
	position := node.Position

	// 根据path和position，定义到 symbol，从而找到它的relation
	var elementTable codegraphpb.FileElementTable
	fileTableBytes, err := i.storage.Get(ctx, projectUuid, store.ElementPathKey{
		Language: language, Path: symbolPath})
	if err != nil {
		i.logger.Error("query_definitions failed to get fileTable by def: %s, err: %v", err)
		return
	}

	if err = store.UnmarshalValue(fileTableBytes, &elementTable); err != nil {
		i.logger.Error("Failed to unmarshal fileTable by path: %s, err:%v", symbolPath, err)
		return
	}

	if err != nil {
		i.logger.Debug("Failed to find elementTable for path %s: %v", symbolPath, err)
		return
	}

	symbol := i.findSymbolInDocByRange(&elementTable, types.ToRange(position))
	if symbol == nil {
		i.logger.Debug("Symbol not found in elementTable: path %s, name %s", symbolPath, symbolName)
		return
	}

	var referenceChildren []*types.GraphNode

	// 找到symbol 的relation. 只有定义的symbol 有reference，引用节点的relation是定义节点
	if len(symbol.Relations) > 0 {
		for _, r := range symbol.Relations {
			if r.RelationType == codegraphpb.RelationType_RELATION_TYPE_REFERENCE {
				// 引用节点，加入node的children
				referenceChildren = append(referenceChildren, &types.GraphNode{
					FilePath:   r.ElementPath,
					SymbolName: r.ElementName,
					Position:   types.ToPosition(r.Range),
					NodeType:   string(types.NodeTypeReference),
				})
			}
		}
		i.logger.Debug("Found %d reference relations for symbol %s", len(referenceChildren), symbolName)
	}

	if len(referenceChildren) == 0 {
		// 如果references 为空，说明当前 node 是引用节点， 找到它所属的函数/类/结构体的definition节点，再找引用
		foundDefSymbol := i.findReferenceSymbolBelonging(&elementTable, symbol)
		if foundDefSymbol != nil {
			referenceChildren = append(referenceChildren, &types.GraphNode{
				FilePath:   elementTable.Path,
				SymbolName: foundDefSymbol.Name,
				Position:   types.ToPosition(foundDefSymbol.Range),
				NodeType:   string(types.NodeTypeDefinition),
			})
		} else {
			i.logger.Debug("failed to find reference symbol %s belongs to", symbolName)
		}

	}

	//当前节点的子节点
	node.Children = referenceChildren

	// 继续递归
	for _, ch := range referenceChildren {
		i.buildChildrenRecursive(ctx, projectUuid, language, ch, maxLayer)
	}
}

func (i *Indexer) QueryDefinitions(ctx context.Context, options *types.QueryDefinitionOptions) ([]*types.Definition, error) {
	filePath := options.FilePath
	project, err := i.workspaceReader.GetProjectByFilePath(ctx, options.Workspace, filePath)
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

	var res []*types.Definition

	var foundSymbols []*codegraphpb.Element
	queryStartLine := int32(options.StartLine - 1)
	queryEndLine := int32(options.EndLine - 1)

	snippet := options.CodeSnippet

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
		// TODO 找到所有的外部依赖, 当前只处理call
		var callNames []string
		for _, e := range elements {
			if c, ok := e.(*resolver.Call); ok {
				callNames = append(callNames, c.Name)
			}
		}
		if len(callNames) == 0 {
			return nil, fmt.Errorf("no function/method call found in code snippet")
		}

		// 对imports预处理
		if filteredImps, err := i.analyzer.PreprocessImports(ctx, language, project, imports); err == nil {
			imports = filteredImps
		}

		// 根据所找到的call 的name + imports， 去模糊匹配symbol
		symDefs, err := i.searchSymbolNames(ctx, projectUuid, callNames, imports)
		if err != nil {
			return nil, fmt.Errorf("failed to search function/method call names: %w", err)
		}
		if len(symDefs) == 0 {
			return nil, fmt.Errorf("failed to search symbol by function/method call names")
		}
		for _, def := range symDefs {
			var fileTable codegraphpb.FileElementTable
			fileTableBytes, err := i.storage.Get(ctx, projectUuid, store.ElementPathKey{
				Language: language, Path: def.Path})
			if err != nil {
				i.logger.Error("query_definitions failed to get fileTable by def: %s, err: %v", err)
				continue
			}
			if err = store.UnmarshalValue(fileTableBytes, &fileTable); err != nil {
				i.logger.Error("Failed to unmarshal fileTable by path: %s, err:%v", def.Path, err)
				continue
			}
			if symbol := i.findSymbolInDocByRange(&fileTable, def.Range); symbol != nil {
				foundSymbols = append(foundSymbols, symbol)
			} else {
				i.logger.Error("Failed to find symbol in file_element_table %s by range: %v",
					def.Path, def.Range)
			}
		}

	} else {
		// 1. 获取文档
		var fileTable codegraphpb.FileElementTable
		fileTableBytes, err := i.storage.Get(ctx, projectUuid, store.ElementPathKey{Language: language, Path: filePath})
		if err != nil {
			return nil, fmt.Errorf("failed to get file %s element_table, err: %v", filePath, err)
		}
		if err = store.UnmarshalValue(fileTableBytes, &fileTable); err != nil {
			return nil, fmt.Errorf("failed to unmarshal file %s element_table value, err: %v", filePath, err)
		}

		foundSymbols = i.findSymbolInDocByLineRange(ctx, &fileTable, queryStartLine, queryEndLine)

	}

	// 去重, 如果range 出现过，则剔除
	existedDefs := make(map[string]bool)
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
			})
			continue
		} else { // 引用
			for _, r := range s.Relations {
				// 引用的 relation 是定义
				if r.RelationType == codegraphpb.RelationType_RELATION_TYPE_DEFINITION {
					// 如果已经访问过，就跳过
					if isSymbolExists(r.ElementPath, r.Range, existedDefs) {
						continue
					}
					existedDefs[symbolMapKey(r.ElementPath, r.Range)] = true
					// 引用节点，加入node的children
					res = append(res, &types.Definition{
						Path:  r.ElementPath,
						Name:  r.ElementName,
						Range: r.Range,
					})
				}
			}
		}
	}

	return res, nil
}

const eachSymbolKeepResult = 2

func (i *Indexer) searchSymbolNames(ctx context.Context, projectUuid string, names []string, imports []*resolver.Import) (
	[]*codegraphpb.Definition, error) {

	start := time.Now()
	// 去重
	names = utils.DeDuplicate(names)
	foundKeys := make(map[string][]*codegraphpb.Definition)

	iter := i.storage.Iter(ctx, projectUuid)
	defer iter.Close()

	for iter.Next() {

		key := iter.Key()
		symbol, err := store.ToSymbolNameKey(key)
		if err != nil {
			i.logger.Debug("search_symbol_names err: %v", err)
			continue
		}
		symbolName := symbol.Name

		var matchName string
		for _, name := range names {
			if symbolName == name { //TODO 是否包含特殊符号？ 将 namespace 放入 KeyRange, 用于过滤（map->[keyRange]）
				matchName = name
				break
			}

		}
		if matchName == types.EmptyString {
			continue
		}

		item := iter.Value()
		var keyset codegraphpb.SymbolDefinition
		if err := store.UnmarshalValue(item, &keyset); err != nil {
			return nil, fmt.Errorf("failed to deserialize document: %w", err)
		}

		if len(keyset.Definitions) == 0 {
			continue
		}

		if _, ok := foundKeys[matchName]; !ok {
			foundKeys[matchName] = make([]*codegraphpb.Definition, 0)
		}
		foundKeys[matchName] = append(foundKeys[matchName], keyset.Definitions...)
	}

	// TODO 同一个name可能检索出多条数据，再根据import 过滤一遍。 namespace 要么用. 要么用/ 得判断
	total := 0
	for _, v := range foundKeys {
		total += len(v)
	}

	filteredResult := make([]*codegraphpb.Definition, 0, total)

	if len(imports) > 0 {
		for _, v := range foundKeys {
			for _, doc := range v {
				for _, imp := range imports {
					if strings.Contains(doc.Path, imp.Name) || strings.Contains(doc.Path, imp.Source) { //TODO , go work ，多模块等特殊情况
						filteredResult = append(filteredResult, doc)
						break
					}
				}
			}
		}
	}

	if len(filteredResult) == 0 {
		i.logger.Debug("result after filter is empty, use origin result ans keep 2 with each symbol")
		// 每个仅保留2个
		for _, v := range foundKeys {
			if len(v) > eachSymbolKeepResult {
				v = v[:eachSymbolKeepResult]
			}
			filteredResult = append(filteredResult, v...)
		}
	}
	i.logger.Info("codegraph symbol name search end, cost %d ms, names: %d, key found:%d, filtered result:%d",
		time.Since(start).Milliseconds(), len(names), total, len(filteredResult))
	return filteredResult, nil
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
	projects := i.workspaceReader.FindProjects(ctx, workspacePath, workspace.DefaultVisitPattern)
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
	projects := i.workspaceReader.FindProjects(ctx, workspacePath, workspace.DefaultVisitPattern)
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
	projects := i.workspaceReader.FindProjects(ctx, workspacePath, workspace.DefaultVisitPattern)
	if len(projects) == 0 {
		i.logger.Info("rename_index found no projects in workspace %s", workspacePath)
		return nil
	}
	sourceProject, err := i.workspaceReader.GetProjectByFilePath(ctx, workspacePath, sourceFilePath)
	if err != nil {
		return fmt.Errorf("rename_index cannot find project for file %s, err:%v", sourceFilePath, err)
	}
	targetProject, err := i.workspaceReader.GetProjectByFilePath(ctx, workspacePath, targetFilePath)
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

		// 更新引用
		relations := i.collectRelations([]*codegraphpb.FileElementTable{st})
		// 将引用节点中当前节点的路径进行更新，将引用的路径
		for _, r := range relations {
			fileTable, err := i.getFileElementTableByPath(ctx, sourceProjectUuid, r.ElementPath)
			if err != nil {
				i.logger.Debug("rename_index get FileElementTable by path %s err:%v", r.ElementPath, err)
				continue
			}
			for _, e := range fileTable.Elements {
				if len(e.GetRelations()) == 0 {
					continue
				}
				for _, relation := range e.GetRelations() {
					if relation.ElementPath == oldPath {
						relation.ElementPath = newPath
					}
				}
			}
			// 保存
			if err = i.storage.Put(ctx, sourceProjectUuid, &store.Entry{
				Key: store.ElementPathKey{
					Language: lang.Language(fileTable.Language),
					Path:     fileTable.Path},
				Value: fileTable,
			}); err != nil {
				i.logger.Debug("rename_index save FileElementTable %s err:%v", r.ElementPath, err)
			}
		}

		// 更新符号定义，找到相关符号，将它的path由old改为new
		for _, e := range st.Elements {
			if !e.IsDefinition {
				continue
			}
			symbolDefinition, err := i.getSymbolDefinitionByName(ctx, sourceProjectUuid, lang.Language(oldLanguage), e.Name)
			if err != nil {
				i.logger.Debug("rename_index get symbol definition by name %s %s err:%v", oldLanguage, e.Name, err)
				continue
			}

			// 语言相同则更新，语言不同，则删除新增
			sameLanguage := oldLanguage == string(newLanguage)
			definitions := make([]*codegraphpb.Definition, 0, len(symbolDefinition.Definitions))
			for _, d := range symbolDefinition.Definitions {
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
					Language: lang.Language(symbolDefinition.Language),
					Name:     symbolDefinition.Name},
				Value: symbolDefinition,
			}); err != nil {
				i.logger.Debug("rename_index save symbolDefinition %s err:%v", e.Name, err)
			}
			// 不同语言，保存新的
			if !sameLanguage {
				newSymbolDefiniton := &codegraphpb.SymbolDefinition{
					Name:     e.Name,
					Language: string(newLanguage),
					Definitions: []*codegraphpb.Definition{
						{
							Path:        newPath,
							Range:       e.Range,
							ElementType: e.ElementType,
						},
					},
				}
				if err = i.storage.Put(ctx, sourceProjectUuid, &store.Entry{
					Key: store.SymbolNameKey{
						Language: lang.Language(symbolDefinition.Language),
						Name:     symbolDefinition.Name},
					Value: newSymbolDefiniton,
				}); err != nil {
					i.logger.Debug("rename_index save symbolDefinition %s err:%v", e.Name, err)
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
func (i *Indexer) getSymbolDefinitionByName(ctx context.Context, projectUuid string,
	language lang.Language, symbolName string) (*codegraphpb.SymbolDefinition, error) {

	bytes, err := i.storage.Get(ctx, projectUuid, store.SymbolNameKey{Language: language, Name: symbolName})
	if err != nil {
		return nil, err
	}
	var symbolDefinition codegraphpb.SymbolDefinition
	err = store.UnmarshalValue(bytes, &symbolDefinition)
	return &symbolDefinition, err
}

func (i *Indexer) CollectProjectSourceFiles(ctx context.Context, projectPath string, visitPattern *types.VisitPattern) ([]string, error) {

	filePaths := make([]string, 0, 100)

	err := i.workspaceReader.WalkFile(ctx, projectPath, func(walkCtx *types.WalkContext) error {
		if walkCtx.Info.IsDir {
			return nil
		}

		filePaths = append(filePaths, walkCtx.Path)
		return nil
	}, types.WalkOptions{IgnoreError: true, VisitPattern: visitPattern})
	if err != nil {
		return nil, err
	}
	return filePaths, nil
}

func (i *Indexer) parseFiles(ctx context.Context, filePaths []string) ([]*parser.FileElementTable, *types.IndexTaskMetrics, error) {
	fileElementTables := make([]*parser.FileElementTable, 0)
	projectTaskMetrics := &types.IndexTaskMetrics{TotalFiles: len(filePaths),
		FailedFilePaths: make([]string, 0),
	}
	var errs []error

	for _, path := range filePaths {
		language, err := lang.InferLanguage(path)
		if err != nil || language == types.EmptyString {
			// not supported language or not source file
			projectTaskMetrics.TotalFailedFiles++
			projectTaskMetrics.FailedFilePaths = append(projectTaskMetrics.FailedFilePaths, path)
			errs = append(errs, err)
			continue
		}
		bytes, err := i.workspaceReader.ReadFile(ctx, path, types.ReadOptions{})
		if err != nil {
			projectTaskMetrics.TotalFailedFiles++
			projectTaskMetrics.FailedFilePaths = append(projectTaskMetrics.FailedFilePaths, path)
			errs = append(errs, err)
			continue
		}

		fileElementTable, err := i.parser.Parse(ctx, &types.SourceFile{
			Path:    path,
			Content: bytes,
		})

		if err != nil {
			projectTaskMetrics.TotalFailedFiles++
			projectTaskMetrics.FailedFilePaths = append(projectTaskMetrics.FailedFilePaths, path)
			errs = append(errs, err)
			continue
		}
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
