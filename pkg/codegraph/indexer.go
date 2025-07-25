package codegraph

import (
	"codebase-indexer/pkg/codegraph/analyzer"
	"codebase-indexer/pkg/codegraph/lang"
	"codebase-indexer/pkg/codegraph/parser"
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
	"os"
	"strings"
	"time"
)

// Indexer 代码索引器
type Indexer struct {
	parser          *parser.SourceFileParser     // 单文件语法解析
	analyzer        *analyzer.DependencyAnalyzer // 跨文件依赖分析
	workspaceReader workspace.WorkspaceReader    // 进行工作区的文件读取、项目识别、项目列表维护
	storage         store.GraphStorage           // 存储
	MaxConcurrency  int                          // 最大并发度 TODO 配置类
	logger          logger.Logger
}

func NewCodeIndexer(parser *parser.SourceFileParser,
	analyzer *analyzer.DependencyAnalyzer,
	store store.GraphStorage,
	logger logger.Logger) *Indexer {

	return &Indexer{
		parser:   parser,
		analyzer: analyzer,
		storage:  store,
		logger:   logger,
	}
}

func (i *Indexer) IndexWorkspace(ctx context.Context, workspace string) error {
	i.logger.Info("index_workspace start to index workspace：%s", workspace)
	// 将workspace 拆分为项目
	projects := i.workspaceReader.FindProjects(workspace, types.VisitPattern{})
	if len(projects) == 0 {
		return fmt.Errorf("index_workspace find no projects in workspace: %s", workspace)
	}
	var errs []error
	workspaceStart := time.Now()
	workspaceTaskMetrics := types.IndexTaskMetrics{}
	// 循环项目，逐个处理
	for _, p := range projects {
		projectTaskMetrics, err := i.indexProject(ctx, p)
		if err != nil {
			i.logger.Error("index_workspace index project %s err: %v", p.Path, utils.TruncateError(errors.Join(err...)))
			errs = append(errs, err...)
			continue
		}

		workspaceTaskMetrics.TotalFiles += projectTaskMetrics.TotalFiles
		workspaceTaskMetrics.TotalSourceFiles += projectTaskMetrics.TotalSourceFiles
		workspaceTaskMetrics.TotalSucceedFiles += projectTaskMetrics.TotalSucceedFiles
		workspaceTaskMetrics.TotalFailedFiles += projectTaskMetrics.TotalFailedFiles
	}

	i.logger.Info("index_workspace %s index workspace finish, cost %d ms, visit %d files, "+
		"%d valid source files, parsed %d files successfully, failed %d files", workspace, time.Since(workspaceStart).Milliseconds(),
		workspaceTaskMetrics.TotalFiles, workspaceTaskMetrics.TotalSourceFiles,
		workspaceTaskMetrics.TotalSucceedFiles, workspaceTaskMetrics.TotalFailedFiles)

	return nil
}

func (i *Indexer) indexProject(ctx context.Context, p *workspace.Project) (types.IndexTaskMetrics, []error) {
	var errs []error
	projectUuid, err := p.Uuid()
	if err != nil {
		return types.IndexTaskMetrics{}, append(errs, err)
	}

	projectStart := time.Now()
	i.logger.Info("index_project start to index project：%s", p.Path)
	// TODO 日志
	// 1、项目符号表
	projectTable := &analyzer.ProjectElementTable{
		ProjectInfo:       p,
		FileElementTables: make([]*parser.FileElementTable, 0),
	}

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
		projectTable.FileElementTables = append(projectTable.FileElementTables, fileElementTable)
		return nil
	}, types.WalkOptions{
		IgnoreError:  true,
		VisitPattern: types.VisitPattern{ExcludeDirs: []string{".git", ".idea", ".vscode"}},
		//TODO exclude， 包括 系统ignore .gitignore 和 .coignore
	}); err != nil {
		errs = append(errs, err)
	}

	i.logger.Info("index_project project %s parse finish. cost %d ms, visit %d files, "+
		"%d valid source files, parsed %d files successfully, failed %d files",
		p.Path, time.Since(projectStart).Milliseconds(), projectTaskMetrics.TotalFiles,
		projectTaskMetrics.TotalSourceFiles,
		projectTaskMetrics.TotalSucceedFiles, projectTaskMetrics.TotalFailedFiles)

	if len(projectTable.FileElementTables) == 0 {
		errs = append(errs, fmt.Errorf("index_project project %s parsed no source files", p.Path))
		return types.IndexTaskMetrics{}, errs
	}

	// 2. 项目符号表构建与存储
	if err := i.analyzer.SaveSymbolDefinitions(ctx, projectUuid, projectTable.FileElementTables); err != nil {
		errs = append(errs, err)
		return types.IndexTaskMetrics{}, errs
	}

	// 3. 依赖分析
	if err := i.analyzer.Analyze(ctx, projectTable); err != nil {
		errs = append(errs, err)
		return types.IndexTaskMetrics{}, errs
	}

	// 4. 关系索引存储
	if err := i.storage.BatchSave(ctx, projectUuid, workspace.FileElementTables(projectTable.FileElementTables)); err != nil {
		errs = append(errs, err)
		return types.IndexTaskMetrics{}, errs
	}

	i.logger.Info("index_project index project finish：%s", p.Path)

	return projectTaskMetrics, nil
}

// RemoveIndexes 根据工作区路径、文件路径，批量删除索引
func (i *Indexer) RemoveIndexes(ctx context.Context, workspacePath string, filePaths []string) error {
	// 根据 workspaceReader 及 filepath 路径，匹配 到project
	// 再到对应project中，删除对应的index
	// 以filepath 为key，先将index查询出来，遍历index的element，找到它被谁引用，然后更新对应的引用关系，再删除index
	start := time.Now()
	i.logger.Info("remove_indexes start to remove workspace %s files: %v", workspacePath, filePaths)
	projects := i.workspaceReader.FindProjects(workspacePath, types.VisitPattern{})
	if len(projects) == 0 {
		return fmt.Errorf("no project found in workspace %s", workspacePath)
	}
	var errs []error
	// TODO 重复代码
	// project key -> filePaths
	var projectFiles map[string]struct {
		p     *workspace.Project
		files []string
	}
	// 如果一个文件路径是一个项目的子路径，则将其加入到该项目的文件列表中
	for _, p := range projects {
		for _, filePath := range filePaths {
			if strings.HasPrefix(filePath, p.Path) {
				projectUuid, err := p.Uuid()
				if err != nil {
					errs = append(errs, err)
					continue
				}
				pf, ok := projectFiles[projectUuid]
				if !ok {
					pf = struct {
						p     *workspace.Project
						files []string
					}{p: p, files: make([]string, 0)}
				}
				pf.files = append(pf.files, filePath)
				projectFiles[projectUuid] = pf
			}
		}
	}

	for puuid, pfiles := range projectFiles {

		pStart := time.Now()

		i.logger.Info("remove_indexes start to remove project %s files index", pfiles.p.Name)

		// 1.查询path 相应的 filetable
		var deleteFileTables []*parser.FileElementTable
		deletedPaths := make(map[string]interface{})
		for _, fp := range pfiles.files {
			fileTable, err := i.storage.Get(ctx, puuid, fp)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			ft := fileTable.(*parser.FileElementTable)
			deleteFileTables = append(deleteFileTables, ft)
			deletedPaths[ft.Path] = nil
		}

		// 2.找到所有与它相关的path，也就是被继承、实现、引用的地方，它作为定义。
		var referedElements []*resolver.Relation
		for _, ft := range deleteFileTables {
			for _, e := range ft.Elements {
				if len(e.GetRelations()) == 0 {
					continue
				}
				for _, r := range e.GetRelations() {
					if r.RelationType == resolver.RelationTypeSuperInterface ||
						r.RelationType == resolver.RelationTypeSuperClass ||
						r.RelationType == resolver.RelationTypeReference {
						referedElements = append(referedElements, r)
					}
				}
			}
		}

		// 3.查询相关的path，将相关path中与待删除path相关的relation删除
		for _, ref := range referedElements {
			// 获取引用该符号的文件
			refFileTable, err := i.storage.Get(ctx, puuid, store.PathKey(ref.ElementPath))
			if err != nil {
				errs = append(errs, err)
				continue
			}

			refTable := refFileTable.(*parser.FileElementTable)

			// 移除与该符号相关的relation
			for _, e := range refTable.Elements {
				var newRelations []*resolver.Relation
				for _, r := range e.GetRelations() {
					// 如果relation指向待删除的符号，则跳过
					if _, ok := deletedPaths[r.ElementPath]; ok {
						continue
					}
					newRelations = append(newRelations, r)
				}
				e.SetRelations(newRelations)
			}

			// 保存更新后的文件表
			if err := i.storage.Save(ctx, puuid, refTable); err != nil {
				errs = append(errs, err)
			}

		}

		// 4. 删除fileTable的相关definition symbol
		for _, ft := range deleteFileTables {
			for _, e := range ft.Elements {
				if e.GetType() == types.ElementTypeMethod || e.GetType() == types.ElementTypeFunction ||
					e.GetType() == types.ElementTypeInterface || e.GetType() == types.ElementTypeClass {
					sym, err := i.storage.Get(ctx, puuid, store.SymbolKey(e.GetName()))
					if err != nil {
						errs = append(errs, err)
						continue
					}
					symDefs := sym.(*analyzer.SymbolDefinition)
					newSymDefs := &analyzer.SymbolDefinition{Name: e.GetName(), Definitions: make([]*analyzer.Definition, 0)}
					for _, d := range symDefs.Definitions {
						if _, ok := deletedPaths[d.Path]; ok {
							continue
						}
						newSymDefs.Definitions = append(newSymDefs.Definitions, d)
					}
					// 保存更新后的文件表
					if err := i.storage.Save(ctx, puuid, newSymDefs); err != nil {
						errs = append(errs, err)
					}
				}
			}
		}

		// 5. 删除path、symbol
		for _, fp := range pfiles.files {
			// 删除path索引
			if err := i.storage.Delete(ctx, puuid, store.PathKey(fp)); err != nil {
				errs = append(errs, err)
			}

		}

		i.logger.Info("remove_indexes remove project %s files index end, cost %d ms", pfiles.p.Name,
			time.Since(pStart).Milliseconds())
	}

	err := errors.Join(errs...)
	i.logger.Info("remove_indexes remove workspace %s files index successfully, cost %d ms, errors: %v",
		workspacePath, time.Since(start).Milliseconds(), utils.TruncateError(err))
	return err
}

// IndexFiles 根据工作区路径、文件路径，批量保存索引
func (i *Indexer) IndexFiles(ctx context.Context, workspacePath string, filePaths []string) error {
	// 根据 workspaceReader 及 filepath 路径，匹配 到project
	// 再到对应project中，保存对应的index，存在直接覆盖
	// TODO 如果filepath 所在项目没有索引过，索引整个项目
	i.logger.Info("index_files start to index workspace %s files: %v", workspacePath, filePaths)
	projects := i.workspaceReader.FindProjects(workspacePath, types.VisitPattern{})
	if len(projects) == 0 {
		return fmt.Errorf("no project found in workspace %s", workspacePath)
	}
	var errs []error
	// project key -> filePaths
	var projectFiles map[string]struct {
		p     *workspace.Project
		files []string
	}
	// 如果一个文件路径是一个项目的子路径，则将其加入到该项目的文件列表中
	for _, p := range projects {
		for _, filePath := range filePaths {
			if strings.HasPrefix(filePath, p.Path) {
				projectUuid, err := p.Uuid()
				if err != nil {
					errs = append(errs, err)
					continue
				}
				pf, ok := projectFiles[projectUuid]
				if !ok {
					pf = struct {
						p     *workspace.Project
						files []string
					}{p: p, files: make([]string, 0)}
				}
				pf.files = append(pf.files, filePath)
				projectFiles[projectUuid] = pf
			}
		}
	}

	for puuid, pfiles := range projectFiles {
		if i.storage.Size(ctx, puuid) == 0 {
			// index project entirely
			i.indexProject(ctx, pfiles.p)
		} else {
			// index files
			// 单个项目的多个文件
			fileTables := make([]*parser.FileElementTable, 0)
			for _, f := range pfiles.files {

				if language, err := lang.InferLanguage(f); language == types.EmptyString || err != nil {
					continue
				}

				content, err := os.ReadFile(f)
				if err != nil {
					errs = append(errs, err)
					continue
				}

				fileElementTable, err := i.parser.Parse(ctx, &types.SourceFile{
					Path:    f,
					Content: content,
				})
				if err != nil {
					errs = append(errs, err)
					continue
				}
				fileTables = append(fileTables, fileElementTable)

			}
			// TODO 还是要根据symbol,查询   需要将Definition Symbol 存储起来，再查询。
			// 保存本地符号表
			err := i.analyzer.SaveSymbolDefinitions(ctx, puuid, fileTables)
			if err != nil {
				errs = append(errs, fmt.Errorf("save_index save symbol definitions error: %w", err))
				continue
			}
			projectTable := &analyzer.ProjectElementTable{
				ProjectInfo:       pfiles.p,
				FileElementTables: fileTables,
			}
			// 依赖分析
			if err := i.analyzer.Analyze(ctx, projectTable); err != nil {
				errs = append(errs, fmt.Errorf("save_index analyze dependency error: %w", err))
				continue
			}
			// 关系索引存储
			if err := i.storage.BatchSave(ctx, puuid, workspace.FileElementTables(projectTable.FileElementTables)); err != nil {
				errs = append(errs, fmt.Errorf("save_index batch save error: %w", err))
				continue
			}
		}
	}
	err := errors.Join(errs...)
	i.logger.Info("index_files index workspace %s files successfully, errors: %v", workspacePath, filePaths,
		utils.TruncateError(err))
	return err
}

// QueryIndexes 查询索引
func (i *Indexer) QueryIndexes(workspace string, filePaths []string) error {
	// 根据workspace 及 filepath 路径，匹配 到project
	// 再到对应project中，查询对应的index

	return nil
}
