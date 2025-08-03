package codegraph

import (
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
	"io"
	"strings"
	"time"
)

// projectFiles 用于存储项目和对应的文件列表
type projectFiles struct {
	p     *workspace.Project
	files []string
}

type IndexerConfig struct {
	MaxConcurrency int
	VisitPattern   types.VisitPattern
}

// Indexer 代码索引器
type Indexer struct {
	parser          *parser.SourceFileParser     // 单文件语法解析
	analyzer        *analyzer.DependencyAnalyzer // 跨文件依赖分析
	workspaceReader *workspace.WorkspaceReader   // 进行工作区的文件读取、项目识别、项目列表维护
	storage         store.GraphStorage           // 存储
	config          IndexerConfig
	logger          logger.Logger
}

// TODO 多子项目、多语言支持。monorepo, 不光通过语言隔离、还要通过子项目隔离。

// NewCodeIndexer 创建新的代码索引器
func NewCodeIndexer(
	parser *parser.SourceFileParser,
	analyzer *analyzer.DependencyAnalyzer,
	workspaceReader *workspace.WorkspaceReader,
	storage store.GraphStorage,
	config IndexerConfig,
	logger logger.Logger,
) *Indexer {
	return &Indexer{
		parser:          parser,
		analyzer:        analyzer,
		workspaceReader: workspaceReader,
		storage:         storage,
		config:          config,
		logger:          logger,
	}
}

// IndexWorkspace 索引整个工作区
func (i *Indexer) IndexWorkspace(ctx context.Context, workspace string) error {
	i.logger.Info("index_workspace start to index workspace：%s", workspace)
	exists, err := i.workspaceReader.Exists(ctx, workspace)
	if err == nil && !exists {
		return fmt.Errorf("index_workspace workspace %s not exists", workspace)
	}
	projects := i.workspaceReader.FindProjects(ctx, workspace, types.VisitPattern{})
	if len(projects) == 0 {
		return fmt.Errorf("index_workspace find no projects in workspace: %s", workspace)
	}

	var errs []error
	taskMetrics := &types.IndexTaskMetrics{}
	workspaceStart := time.Now()

	// 循环项目，逐个处理
	for _, p := range projects {
		projectTaskMetrics, err := i.indexProject(ctx, p)
		if err != nil {
			i.logger.Error("index_workspace index project %s err: %v", p.Path, utils.TruncateError(errors.Join(err...)))
			errs = append(errs, err...)
			continue
		}

		taskMetrics.TotalFiles += projectTaskMetrics.TotalFiles
		taskMetrics.TotalSourceFiles += projectTaskMetrics.TotalSourceFiles
		taskMetrics.TotalSucceedFiles += projectTaskMetrics.TotalSucceedFiles
		taskMetrics.TotalFailedFiles += projectTaskMetrics.TotalFailedFiles
	}

	i.logger.Info("index_workspace %s index workspace finish, cost %d ms, visit %d files, "+
		"%d valid source files, parsed %d files successfully, failed %d files", workspace, time.Since(workspaceStart).Milliseconds(),
		taskMetrics.TotalFiles, taskMetrics.TotalSourceFiles,
		taskMetrics.TotalSucceedFiles, taskMetrics.TotalFailedFiles)
	return nil
}

// indexProject 索引单个项目
func (i *Indexer) indexProject(ctx context.Context, p *workspace.Project) (types.IndexTaskMetrics, []error) {
	var errs []error
	projectUuid := p.Uuid
	projectStart := time.Now()
	i.logger.Info("index_project start to index project：%s", p.Path)

	fileElementTables, projectTaskMetrics, err := i.ParseProjectFiles(ctx, p)
	if err != nil {
		return types.IndexTaskMetrics{}, append(errs, err)
	}

	i.logger.Info("index_project project %s parse finish. cost %d ms, visit %d files, "+
		"%d valid source files, parsed %d files successfully, failed %d files",
		p.Path, time.Since(projectStart).Milliseconds(), projectTaskMetrics.TotalFiles,
		projectTaskMetrics.TotalSourceFiles,
		projectTaskMetrics.TotalSucceedFiles, projectTaskMetrics.TotalFailedFiles)

	if len(fileElementTables) == 0 {
		errs = append(errs, fmt.Errorf("index_project project %s parsed no source files", p.Path))
		return types.IndexTaskMetrics{}, errs
	}

	// 检查elements，剔除不合法的，并打印日志
	i.checkElementTables(fileElementTables)

	// 项目符号表构建与存储
	if err = i.analyzer.SaveSymbolDefinitions(ctx, projectUuid, fileElementTables); err != nil {
		errs = append(errs, err)
		return types.IndexTaskMetrics{}, errs
	}

	// 依赖分析
	if err = i.analyzer.Analyze(ctx, p, fileElementTables); err != nil {
		errs = append(errs, err)
		return types.IndexTaskMetrics{}, errs
	}

	// 转换为 proto
	protoElementTables := proto.FileElementTablesToProto(fileElementTables)

	// 关系索引存储
	if err = i.storage.BatchSave(ctx, projectUuid, workspace.FileElementTables(protoElementTables)); err != nil {
		errs = append(errs, err)
		return types.IndexTaskMetrics{}, errs
	}

	i.logger.Info("index_project: finish：%s, saved %d file element_tables", p.Path, len(protoElementTables))

	return projectTaskMetrics, nil
}

// ParseProjectFiles 解析项目中的所有文件
func (i *Indexer) ParseProjectFiles(ctx context.Context, p *workspace.Project) ([]*parser.FileElementTable, types.IndexTaskMetrics, error) {
	fileElementTables := make([]*parser.FileElementTable, 0)
	projectTaskMetrics := types.IndexTaskMetrics{}

	// 并发walk 目录，构建
	if err := i.workspaceReader.Walk(ctx, p.Path, func(walkCtx *types.WalkContext, reader io.ReadCloser) error {
		projectTaskMetrics.TotalFiles++
		language, err := lang.InferLanguage(walkCtx.Path)
		if err != nil || language == types.EmptyString {
			// not supported language or not source file
			return nil
		}
		projectTaskMetrics.TotalSourceFiles++

		content, err := io.ReadAll(reader)
		if err != nil {
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
		projectTaskMetrics.TotalSucceedFiles++
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

// RemoveIndexes 根据工作区路径、文件路径，批量删除索引
func (i *Indexer) RemoveIndexes(ctx context.Context, workspacePath string, filePaths []string) error {
	start := time.Now()
	i.logger.Info("remove_indexes start to remove workspace %s files: %v", workspacePath, filePaths)

	projects := i.workspaceReader.FindProjects(ctx, workspacePath, types.VisitPattern{})
	if len(projects) == 0 {
		return fmt.Errorf("no project found in workspace %s", workspacePath)
	}

	var errs []error
	projectFilesMap, err := i.groupFilesByProject(projects, filePaths)
	if err != nil {
		return fmt.Errorf("group files by project failed: %w", err)
	}

	for puuid, pfiles := range projectFilesMap {
		pStart := time.Now()
		i.logger.Info("remove_indexes start to remove project %s files index", pfiles.p.Name)

		if err := i.removeProjectIndexes(ctx, puuid, pfiles); err != nil {
			errs = append(errs, err)
		}

		i.logger.Info("remove_indexes remove project %s files index end, cost %d ms", pfiles.p.Name,
			time.Since(pStart).Milliseconds())
	}

	err = errors.Join(errs...)
	i.logger.Info("remove_indexes remove workspace %s files index successfully, cost %d ms, errors: %v",
		workspacePath, time.Since(start).Milliseconds(), utils.TruncateError(err))
	return err
}

// removeProjectIndexes 删除单个项目的索引
func (i *Indexer) removeProjectIndexes(ctx context.Context, puuid string, pfiles projectFiles) error {
	// 1. 查询path相应的filetable
	deleteFileTables, deletedPaths, err := i.getFileTablesForDeletion(ctx, puuid, pfiles.files)
	if err != nil {
		return fmt.Errorf("get file tables for deletion failed: %w", err)
	}

	// 2. 找到所有与它相关的引用关系
	referredElements, err := i.findReferencedElements(deleteFileTables)
	if err != nil {
		return fmt.Errorf("find referenced elements failed: %w", err)
	}

	// 3. 清理引用关系
	if err = i.cleanupReferences(ctx, puuid, referredElements, deletedPaths); err != nil {
		return fmt.Errorf("cleanup references failed: %w", err)
	}

	// 4. 清理符号定义
	if err = i.cleanupSymbolDefinitions(ctx, puuid, deleteFileTables, deletedPaths); err != nil {
		return fmt.Errorf("cleanup symbol definitions failed: %w", err)
	}

	// 5. 删除path索引
	if err = i.deleteFileIndexes(ctx, puuid, pfiles.files); err != nil {
		return fmt.Errorf("delete file indexes failed: %w", err)
	}

	return nil
}

// getFileTablesForDeletion 获取待删除的文件表和路径
func (i *Indexer) getFileTablesForDeletion(ctx context.Context, puuid string, filePaths []string) ([]*codegraphpb.FileElementTable, map[string]interface{}, error) {
	var deleteFileTables []*codegraphpb.FileElementTable
	deletedPaths := make(map[string]interface{})
	var errs []error

	for _, fp := range filePaths {
		language, err := lang.InferLanguage(fp)
		if err != nil {
			continue
		}
		key := store.ElementPathKey{Language: language, Path: fp}

		fileTable, err := i.storage.Get(ctx, puuid, key)
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
		deletedPaths[ft.Path] = nil
	}

	if len(errs) > 0 {
		return nil, nil, errors.Join(errs...)
	}

	return deleteFileTables, deletedPaths, nil
}

// findReferencedElements 查找引用元素
func (i *Indexer) findReferencedElements(deleteFileTables []*codegraphpb.FileElementTable) ([]*codegraphpb.Relation, error) {
	var referedElements []*codegraphpb.Relation

	for _, ft := range deleteFileTables {
		for _, e := range ft.Elements {
			if len(e.GetRelations()) == 0 {
				continue
			}
			for _, r := range e.GetRelations() {
				if r.RelationType == codegraphpb.RelationType_RELATION_TYPE_SUPER_INTERFACE ||
					r.RelationType == codegraphpb.RelationType_RELATION_TYPE_SUPER_CLASS ||
					r.RelationType == codegraphpb.RelationType_RELATION_TYPE_REFERENCE {
					referedElements = append(referedElements, r)
				}
			}
		}
	}

	return referedElements, nil
}

// cleanupReferences 清理引用关系
func (i *Indexer) cleanupReferences(ctx context.Context, puuid string, referedElements []*codegraphpb.Relation, deletedPaths map[string]interface{}) error {
	var errs []error

	for _, ref := range referedElements {
		language, err := lang.InferLanguage(ref.ElementPath)
		if err != nil {
			continue
		}
		key := store.ElementPathKey{Language: language, Path: ref.ElementPath}
		// 获取引用该符号的文件
		refFileTable, err := i.storage.Get(ctx, puuid, key)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		refTable := new(codegraphpb.FileElementTable)
		if err = store.UnmarshalValue(refFileTable, refTable); err != nil {
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
		if err = i.storage.Save(ctx, puuid, &store.Entry{Key: key, Value: refTable}); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// cleanupSymbolDefinitions 清理符号定义
func (i *Indexer) cleanupSymbolDefinitions(ctx context.Context, puuid string, deleteFileTables []*codegraphpb.FileElementTable, deletedPaths map[string]interface{}) error {
	var errs []error

	for _, ft := range deleteFileTables {
		for _, e := range ft.Elements {
			if e.ElementType == codegraphpb.ElementType_ELEMENT_TYPE_METHOD ||
				e.ElementType == codegraphpb.ElementType_ELEMENT_TYPE_FUNCTION ||
				e.ElementType == codegraphpb.ElementType_ELEMENT_TYPE_INTERFACE ||
				e.ElementType == codegraphpb.ElementType_ELEMENT_TYPE_CLASS {
				language, err := lang.InferLanguage(e.GetPath())
				if err != nil {
					continue
				}
				sym, err := i.storage.Get(ctx, puuid, store.SymbolNameKey{Language: language, Name: e.GetName()})
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
				if err := i.storage.Save(ctx, puuid, &store.Entry{Key: store.SymbolNameKey{Language: language, Name: newSymDefs.Name}, Value: newSymDefs}); err != nil {
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
	projects := i.workspaceReader.FindProjects(ctx, workspacePath, types.VisitPattern{})
	if len(projects) == 0 {
		return fmt.Errorf("no project found in workspace %s", workspacePath)
	}

	var errs []error
	projectFilesMap, err := i.groupFilesByProject(projects, filePaths)
	if err != nil {
		return fmt.Errorf("group files by project failed: %w", err)
	}

	for puuid, pfiles := range projectFilesMap {
		if i.storage.Size(ctx, puuid) == 0 {
			i.logger.Info("index_files project %s has not indexed yet, index project.", puuid)
			// 如果项目没有索引过，索引整个项目
			_, err := i.indexProject(ctx, pfiles.p)
			if err != nil {
				i.logger.Error("index_files index project %s err: %v", pfiles.p.Path, utils.TruncateError(errors.Join(err...)))
				errs = append(errs, err...)
			}
		} else {
			i.logger.Info("index_files project %s has index, index files.", puuid)
			// 索引指定文件
			if err := i.indexProjectFiles(ctx, puuid, pfiles); err != nil {
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
func (i *Indexer) indexProjectFiles(ctx context.Context, puuid string, pfiles projectFiles) error {
	var errs []error
	fileTables := make([]*parser.FileElementTable, 0)

	// 处理每个文件
	for _, f := range pfiles.files {
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
		return fmt.Errorf("no valid files to index in project %s", pfiles.p.Name)
	}

	// 保存本地符号表
	if err := i.analyzer.SaveSymbolDefinitions(ctx, puuid, fileTables); err != nil {
		return fmt.Errorf("save symbol definitions error: %w", err)
	}

	// 依赖分析
	if err := i.analyzer.Analyze(ctx, pfiles.p, fileTables); err != nil {
		return fmt.Errorf("analyze dependency error: %w", err)
	}

	// 转换为 proto
	protoElementTables := proto.FileElementTablesToProto(fileTables)

	// 关系索引存储
	if err := i.storage.BatchSave(ctx, puuid, workspace.FileElementTables(protoElementTables)); err != nil {
		return fmt.Errorf("batch save error: %w", err)
	}

	return nil
}

// QueryElements 查询elements
func (i *Indexer) QueryElements(ctx context.Context, workspacePath string, filePaths []string) ([]*codegraphpb.FileElementTable, error) {
	i.logger.Info("query_elements start to query workspace %s files: %v", workspacePath, filePaths)

	projects := i.workspaceReader.FindProjects(ctx, workspacePath, types.VisitPattern{})
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
		for _, fp := range pfiles.files {
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

	projects := i.workspaceReader.FindProjects(ctx, workspacePath, types.VisitPattern{})
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
		return nil, lang.UnSupportedLanguage
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
func (i *Indexer) groupFilesByProject(projects []*workspace.Project, filePaths []string) (map[string]projectFiles, error) {
	projectFilesMap := make(map[string]projectFiles)
	var errs []error

	for _, p := range projects {
		for _, filePath := range filePaths {
			if strings.HasPrefix(filePath, p.Path) {
				projectUuid := p.Uuid
				pf, ok := projectFilesMap[projectUuid]
				if !ok {
					pf = projectFiles{
						p:     p,
						files: make([]string, 0),
					}
				}
				pf.files = append(pf.files, filePath)
				projectFilesMap[projectUuid] = pf
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
	i.logger.Info("check_element: files total %d, elements before total %d, filtered %d.",
		len(elementTables), total, filtered)
}
