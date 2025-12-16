package indexer

import (
	"codebase-indexer/pkg/codegraph/lang"
	"codebase-indexer/pkg/codegraph/parser"
	"codebase-indexer/pkg/codegraph/proto/codegraphpb"
	"codebase-indexer/pkg/codegraph/resolver"
	"codebase-indexer/pkg/codegraph/store"
	"codebase-indexer/pkg/codegraph/types"
	"codebase-indexer/pkg/codegraph/utils"
	"codebase-indexer/pkg/codegraph/workspace"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"time"
)

// IndexWorkspace 索引整个工作区
func (idx *Indexer) IndexWorkspace(ctx context.Context, workspacePath string) (*types.IndexTaskMetrics, error) {
	taskMetrics := &types.IndexTaskMetrics{}
	workspaceStart := time.Now()
	idx.logger.Info("start to index workspace：%s", workspacePath)
	exists, err := idx.workspaceReader.Exists(ctx, workspacePath)
	if err == nil && !exists {
		return taskMetrics, fmt.Errorf("workspace %s not exists", workspacePath)
	}
	projects := idx.workspaceReader.FindProjects(ctx, workspacePath, true, workspace.DefaultVisitPattern)
	projectsCnt := len(projects)
	if projectsCnt == 0 {
		return taskMetrics, fmt.Errorf("find no projects in workspace: %s", workspacePath)
	}
	if projectsCnt > idx.config.MaxProjects {
		projects = projects[:idx.config.MaxProjects]
		idx.logger.Debug("%s found %d projects, exceed %d max_projects config, use config size.", workspacePath, projectsCnt)
	}

	var errs []error

	// 循环项目，逐个处理
	for _, project := range projects {
		projectTaskMetrics, err := idx.indexProject(ctx, workspacePath, project)
		if err != nil {
			idx.logger.Error("index project %s err: %v",
				project.Path, utils.TruncateError(errors.Join(err...)))
			errs = append(errs, err...)
			continue
		}

		taskMetrics.TotalFiles += projectTaskMetrics.TotalFiles
		taskMetrics.TotalFailedFiles += projectTaskMetrics.TotalFailedFiles
		taskMetrics.FailedFilePaths = append(taskMetrics.FailedFilePaths, projectTaskMetrics.FailedFilePaths...)
	}

	idx.logger.Info("workspace %s index end. cost %d ms, indexed %d projects, visited %d files, "+
		"parsed %d files successfully, failed %d files", workspacePath,
		time.Since(workspaceStart).Milliseconds(), len(projects), taskMetrics.TotalFiles,
		taskMetrics.TotalFiles-taskMetrics.TotalFailedFiles, taskMetrics.TotalFailedFiles)
	return taskMetrics, nil
}

// indexProject 索引单个项目
func (idx *Indexer) indexProject(ctx context.Context, workspacePath string, project *workspace.Project) (*types.IndexTaskMetrics, []error) {
	projectStart := time.Now()
	projectUuid := project.Uuid

	idx.logger.Info("start to index project：%s, max_concurrency: %d, batch_size: %d",
		project.Path, idx.config.MaxConcurrency, idx.config.MaxBatchSize)

	// 获取工作区信息
	workspaceModel, err := idx.workspaceRepository.GetWorkspaceByPath(workspacePath)
	if err != nil {
		return nil, []error{err}
	}
	if workspaceModel == nil {
		return nil, []error{fmt.Errorf("workspace %s not found in database", workspacePath)}
	}
	// 已有的文件数，如果工作区多个项目，需要累加
	databasePreviousFileNum := workspaceModel.CodegraphFileNum

	// 收集要处理的源码文件
	sourceFileTimestamps, err := idx.collectFiles(ctx, workspacePath, project.Path)
	if err != nil {
		return &types.IndexTaskMetrics{TotalFiles: 0}, []error{fmt.Errorf("collect project files err:%v", err)}
	}

	totalFilesCnt := len(sourceFileTimestamps)
	if totalFilesCnt == 0 {
		idx.logger.Info("found no source files in project %s, not index.", project.Path)
		return &types.IndexTaskMetrics{TotalFiles: 0}, nil
	}
	// 校验文件时间戳和索引时间戳，比对需要索引
	filterStart := time.Now()
	needIndexFiles := idx.filterSourceFilesByTimestamp(ctx, projectUuid, sourceFileTimestamps)
	// gc
	sourceFileTimestamps = nil

	filteredCnt := totalFilesCnt - len(needIndexFiles)

	idx.logger.Info("workspace %s filter files by timestamp cost %d ms, total %d files, remaining %d files, filtered %d files.", workspacePath,
		time.Since(filterStart).Milliseconds(), totalFilesCnt, len(needIndexFiles), filteredCnt)

	// 阶段1-3：批量处理文件（解析、检查、保存符号表）
	batchParams := &BatchProcessingParams{
		ProjectUuid:          projectUuid,
		NeedIndexSourceFiles: needIndexFiles,
		TotalFilesCnt:        totalFilesCnt,
		Project:              project,
		WorkspacePath:        workspacePath,
		PreviousFileNum:      databasePreviousFileNum + filteredCnt,
		Concurrency:          idx.config.MaxConcurrency,
		BatchSize:            idx.config.MaxBatchSize,
	}

	batchResult, err := idx.indexFilesInBatches(ctx, batchParams)
	// 合并错误
	if err != nil {
		return &types.IndexTaskMetrics{TotalFiles: 0}, []error{err}
	}

	idx.logger.Info("project %s files parse finish. cost %d ms, visit %d files, "+
		"parsed %d files successfully, failed %d files, total symbols: %d, saved symbols %d, total variables %d, saved variables %d",
		project.Path, time.Since(projectStart).Milliseconds(), batchResult.ProjectMetrics.TotalFiles,
		batchResult.ProjectMetrics.TotalFiles-batchResult.ProjectMetrics.TotalFailedFiles,
		batchResult.ProjectMetrics.TotalFailedFiles,
		batchResult.ProjectMetrics.TotalSymbols,
		batchResult.ProjectMetrics.TotalSavedSymbols,
		batchResult.ProjectMetrics.TotalVariables,
		batchResult.ProjectMetrics.TotalSavedVariables,
	)

	return batchResult.ProjectMetrics, nil
}

// filterSourceFilesByTimestamp 根据时间戳过滤需要索引的文件
func (idx *Indexer) filterSourceFilesByTimestamp(ctx context.Context, projectUuid string, sourceFileTimestamps map[string]int64) []*types.FileWithModTimestamp {
	iter := idx.storage.Iter(ctx, projectUuid)
	defer func(iter store.Iterator) {
		err := iter.Close()
		if err != nil {
			idx.logger.Error("project %s iter close err: %v", projectUuid, err)
		}
	}(iter)
	for iter.Next() {
		if !store.IsElementPathKey(iter.Key()) {
			continue
		}
		key, err := store.ToElementPathKey(iter.Key())
		if err != nil {
			idx.logger.Error("convert key %s to element_path_key err:%v", iter.Key(), err)
			continue
		}
		fileTimestamp, ok := sourceFileTimestamps[key.Path]
		if !ok {
			continue
		}
		var elementTable codegraphpb.FileElementTable
		if err = store.UnmarshalValue(iter.Value(), &elementTable); err != nil {
			idx.logger.Error("unmarshal key %s element_table value err:%v", iter.Key(), err)
			continue
		}
		if elementTable.Timestamp == fileTimestamp {
			delete(sourceFileTimestamps, key.Path)
		}
	}

	needIndexFiles := make([]*types.FileWithModTimestamp, 0, len(sourceFileTimestamps))
	for k, v := range sourceFileTimestamps {
		needIndexFiles = append(needIndexFiles, &types.FileWithModTimestamp{Path: k, ModTime: v})
	}
	return needIndexFiles
}

// preprocessImports 预处理（过滤、转换分隔符）
func (idx *Indexer) preprocessImports(ctx context.Context, elementTables []*parser.FileElementTable,
	project *workspace.Project) error {
	var errs []error
	for _, ft := range elementTables {
		imps, err := idx.analyzer.PreprocessImports(ctx, ft.Language, project, ft.Imports)
		if err != nil {
			errs = append(errs, err)
		} else {
			ft.Imports = imps
		}
	}
	return errors.Join(errs...)
}

// IndexFiles 根据工作区路径、文件路径，批量保存索引
func (idx *Indexer) IndexFiles(ctx context.Context, workspacePath string, filePaths []string) error {
	start := time.Now()
	idx.logger.Info("start to index workspace %s projectFiles: %v", workspacePath, filePaths)
	exists, err := idx.workspaceReader.Exists(ctx, workspacePath)
	if err == nil && !exists {
		return fmt.Errorf("workspace path %s not exists", workspacePath)
	}
	projects := idx.workspaceReader.FindProjects(ctx, workspacePath, true, workspace.DefaultVisitPattern)
	if len(projects) == 0 {
		return fmt.Errorf("no project found in workspace %s", workspacePath)
	}
	workspaceModel, err := idx.workspaceRepository.GetWorkspaceByPath(workspacePath)
	if err != nil {
		return fmt.Errorf("get workspace err:%w", err)
	}

	var errs []error
	projectFilesMap, err := idx.groupFilesByProject(projects, filePaths)
	if err != nil {
		return fmt.Errorf("group files by project failed: %w", err)
	}

	for projectUuid, projectFiles := range projectFilesMap {
		var project *workspace.Project
		for _, p := range projects {
			if p.Uuid == projectUuid {
				project = p
				break
			}
		}
		if project == nil {
			errs = append(errs, fmt.Errorf("failed to find project by uuid %s", projectUuid))
			continue
		}

		if idx.storage.Size(ctx, projectUuid, store.PathKeySystemPrefix) == 0 {
			idx.logger.Info("project %s has not indexed yet, index project.", projectUuid)
			// 如果项目没有索引过，索引整个项目
			_, err := idx.indexProject(ctx, workspacePath, project)
			if err != nil {
				idx.logger.Error("index project %s err: %v", projectUuid, utils.TruncateError(errors.Join(err...)))
				errs = append(errs, err...)
			}
		} else {
			projectStart := time.Now()
			idx.logger.Info("project %s has index, index projectFiles.", projectUuid)
			idx.logger.Info("%s, concurrency: %d, batch_size: %d",
				projectUuid, idx.config.MaxConcurrency, idx.config.MaxBatchSize)
			// 根据规则过滤
			fileWithTimestamps := idx.filterSourceFiles(ctx, workspacePath, projectFiles)

			// 阶段1-3：批量处理文件（解析、检查、保存符号表）
			batchParams := &BatchProcessingParams{
				ProjectUuid:          projectUuid,
				NeedIndexSourceFiles: fileWithTimestamps,
				TotalFilesCnt:        len(projectFiles),
				Project:              project,
				WorkspacePath:        workspacePath,
				PreviousFileNum:      workspaceModel.FileNum,
				Concurrency:          idx.config.MaxConcurrency,
				BatchSize:            idx.config.MaxBatchSize,
			}

			batchResult, err := idx.indexFilesInBatches(ctx, batchParams)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			idx.logger.Info("project %s projectFiles parse finish. cost %d ms, visit %d projectFiles, "+
				"parsed %d projectFiles successfully, failed %d projectFiles",
				projectUuid, time.Since(projectStart).Milliseconds(), batchResult.ProjectMetrics.TotalFiles,
				batchResult.ProjectMetrics.TotalFiles-batchResult.ProjectMetrics.TotalFailedFiles, batchResult.ProjectMetrics.TotalFailedFiles)
		}
	}

	err = errors.Join(errs...)
	idx.logger.Info("index workspace %s projectFiles successfully, cost %d ms, errors: %v", workspacePath,
		time.Since(start).Milliseconds(), utils.TruncateError(err))
	return err
}

// parseFiles 解析文件
func (idx *Indexer) parseFiles(ctx context.Context, files []*types.FileWithModTimestamp) ([]*parser.FileElementTable, *types.IndexTaskMetrics, error) {
	totalFiles := len(files)

	// 优化：预分配切片容量，减少动态扩容
	fileElementTables := make([]*parser.FileElementTable, 0, totalFiles)
	projectTaskMetrics := &types.IndexTaskMetrics{
		TotalFiles:      totalFiles,
		FailedFilePaths: make([]string, 0, totalFiles/4), // 预估失败文件数约为文件数的25%
	}

	var errs []error

	for _, f := range files {
		language, err := lang.InferLanguage(f.Path)
		if err != nil || language == types.EmptyString {
			continue
		}

		// 直接读取文件并解析，避免不必要的中间变量
		content, err := idx.workspaceReader.ReadFile(ctx, f.Path, types.ReadOptions{})
		if err != nil {
			projectTaskMetrics.TotalFailedFiles++
			projectTaskMetrics.FailedFilePaths = append(projectTaskMetrics.FailedFilePaths, f.Path)
			idx.logger.Debug("read file %s err:%v", f, err)
			continue
		}
		// 创建源文件对象并解析
		sourceFile := &types.SourceFile{
			Path:    f.Path,
			Content: content,
		}

		fileElementTable, err := idx.parser.Parse(ctx, sourceFile)
		if err != nil {
			projectTaskMetrics.TotalFailedFiles++
			projectTaskMetrics.FailedFilePaths = append(projectTaskMetrics.FailedFilePaths, f.Path)
			idx.logger.Debug("parse file %s err:%v", f, err)
			// 显式清理内存
			content = nil
			sourceFile.Content = nil
			continue
		}
		fileElementTable.Timestamp = f.ModTime
		fileElementTables = append(fileElementTables, fileElementTable)
	}

	return fileElementTables, projectTaskMetrics, errors.Join(errs...)
}

// collectFiles 收集文件用于index
func (idx *Indexer) collectFiles(ctx context.Context, workspacePath string, projectPath string) (map[string]int64, error) {
	startTime := time.Now()
	filePathModTimestamps := make(map[string]int64, 100)
	ignoreConfig := idx.ignoreScanner.LoadIgnoreConfig(workspacePath)
	if ignoreConfig == nil {
		idx.logger.Error("collect source files ignore_config is nil")
	}
	visitPattern := idx.config.VisitPattern
	if visitPattern == nil {
		visitPattern = workspace.DefaultVisitPattern
	}
	maxFiles := DefaultMaxFiles
	if ignoreConfig != nil {
		visitPattern.SkipFunc = func(fileInfo *types.FileInfo) (bool, error) {
			return idx.ignoreScanner.CheckIgnoreFile(ignoreConfig, workspacePath, fileInfo)
		}
		maxFiles = ignoreConfig.MaxFileCount
	}

	// 从配置中获取(环境变量)
	if idx.config.MaxFiles > 0 {
		maxFiles = idx.config.MaxFiles
	}

	err := idx.workspaceReader.WalkFile(ctx, projectPath, func(walkCtx *types.WalkContext) error {
		if walkCtx.Info.IsDir {
			return nil
		}
		if len(filePathModTimestamps) >= maxFiles {
			idx.logger.Info("collect source files max files %d reached, return.", maxFiles)
			return filepath.SkipAll
		}
		filePathModTimestamps[walkCtx.Path] = walkCtx.Info.ModTime.Unix()
		return nil
	}, types.WalkOptions{IgnoreError: true, VisitPattern: visitPattern})

	if err != nil {
		return nil, err
	}

	idx.logger.Info("collect project source files finish. cost %d ms, found %d source files to index, max files limit %d",
		time.Since(startTime).Milliseconds(), len(filePathModTimestamps), maxFiles)

	return filePathModTimestamps, nil
}

// filterSourceFiles 根据规则过滤源文件
func (idx *Indexer) filterSourceFiles(ctx context.Context, workspacePath string, files []string) []*types.FileWithModTimestamp {
	visitPattern := idx.config.VisitPattern
	if visitPattern == nil {
		visitPattern = workspace.DefaultVisitPattern
	}
	ignoreConfig := idx.ignoreScanner.LoadIgnoreConfig(workspacePath)
	maxFilesLimit := DefaultMaxFiles
	if ignoreConfig != nil {
		visitPattern.SkipFunc = func(fileInfo *types.FileInfo) (bool, error) {
			return idx.ignoreScanner.CheckIgnoreFile(ignoreConfig, workspacePath, fileInfo)
		}
		maxFilesLimit = ignoreConfig.MaxFileCount
	}
	var results []*types.FileWithModTimestamp
	for _, file := range files {
		fileInfo, err := idx.workspaceReader.Stat(file)
		if err != nil {
			idx.logger.Warn("failed to stat file %s, err:%v", file, err)
			continue
		}
		skip, err := visitPattern.ShouldSkip(fileInfo)
		if errors.Is(err, filepath.SkipAll) || errors.Is(err, filepath.SkipDir) {
			continue
		}
		if skip {
			continue
		}
		if len(results) >= maxFilesLimit {
			break
		}
		results = append(results, &types.FileWithModTimestamp{Path: file, ModTime: fileInfo.ModTime.Unix()})
	}
	return results
}

// checkElementTables 检查element_tables
func (idx *Indexer) checkElementTables(elementTables []*parser.FileElementTable) {
	start := time.Now()
	total, filtered := 0, 0
	for _, ft := range elementTables {
		newImports := make([]*resolver.Import, 0, len(ft.Imports))
		newElements := make([]resolver.Element, 0, len(ft.Elements))
		for _, imp := range ft.Imports {
			if resolver.IsValidElement(imp) {
				newImports = append(newImports, imp)
			} else {
				idx.logger.Debug("invalid language %s file %s import {name:%s type:%s path:%s range:%v}",
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
				idx.logger.Debug("invalid language %s file %s element {name:%s type:%s path:%s range:%v}",
					ft.Language, ft.Path, ele.GetName(), ele.GetType(), ele.GetPath(), ele.GetRange())
			}
		}

		ft.Imports = newImports
		ft.Elements = newElements
	}
	idx.logger.Debug("element tables %d, elements before total %d, filtered %d, cost %d ms",
		len(elementTables), total, filtered, time.Since(start).Milliseconds())
}
