package codegraph

import (
	"codebase-indexer/pkg/codegraph/analyzer"
	"codebase-indexer/pkg/codegraph/parser"
	"codebase-indexer/pkg/codegraph/store"
	"codebase-indexer/pkg/codegraph/workspace"
	"context"
)

// Indexer 代码索引器
type Indexer struct {
	parser          *parser.SourceFileParser     // 单文件语法解析
	analyzer        *analyzer.DependencyAnalyzer // 跨文件依赖分析
	workspaceReader workspace.WorkspaceReader    // 进行工作区的文件读取、项目识别、项目列表维护
	storage         *store.GraphStorage          // 存储
	MaxConcurrency  int                          // 最大并发度
}

func NewCodeIndexer(parser *parser.SourceFileParser, analyzer *analyzer.DependencyAnalyzer, store *store.GraphStorage) *Indexer {

	return &Indexer{
		parser:   parser,
		analyzer: analyzer,
		storage:  store,
	}
}

func (i *Indexer) IndexWorkspace(ctx context.Context, workspace string) error {
	// 将workspace 拆分为项目
	//projects := i.workspaceReader.FindProjects(workspace)
	//if len(projects) == 0 {
	//	return fmt.Errorf("index_workspace find no projects in workspace: %s", workspace)
	//}
	//
	//// 循环项目，逐个处理
	//for _, p := range projects {
	//	// 并发walk 目录，构建
	//	i.workspaceReader.Walk(ctx, p.Path, func(walkCtx *types.WalkContext, reader io.ReadCloser) error {
	//
	//	}, types.WalkOptions{
	//		IgnoreError: true,
	//		ExcludeExts: []string{
	//
	//		}
	//	})
	//
	//}
	//
	//// 并行解析文件
	//parsedFiles := make([]*types.SourceFile, 0, len(files))
	//var mu sync.Mutex
	//var wg sync.WaitGroup
	//
	//for _, file := range files {
	//	wg.Add(1)
	//	go func(filePath string) {
	//		defer wg.Done()
	//
	//		// 检测文件语言
	//		language := detectLanguage(filePath)
	//
	//		// 获取对应解析器
	//		parser, exists := i.parserFactory.GetParser(language)
	//		if !exists {
	//			log.Printf("Unsupported language for file: %s", filePath)
	//			return
	//		}
	//
	//		// 解析文件
	//		sourceFile, err := parser.Parse(filePath)
	//		if err != nil {
	//			log.Printf("Error parsing file %s: %v", filePath, err)
	//			return
	//		}
	//
	//		// 添加到结果列表
	//		mu.Lock()
	//		parsedFiles = append(parsedFiles, sourceFile)
	//		mu.Unlock()
	//	}(file)
	//}
	//
	//wg.Wait()
	//
	//// 构建项目符号映射
	//projectMap := i.indexBuilder.BuildProjectSymbolMap(parsedFiles)
	//
	//// 分析关系
	//i.indexBuilder.AnalyzeRelations(projectMap, parsedFiles)
	//
	//// 保存到存储
	//for _, file := range parsedFiles {
	//	key := buildFileKey(file.Path)
	//	if err := i.storage.SaveFileIndex(key, file); err != nil {
	//		return err
	//	}
	//}
	//
	//// 保存符号索引
	//for symbol, elements := range projectMap {
	//	key := buildSymbolKey(symbol)
	//	if err := i.storage.SaveSymbolIndex(key, elements); err != nil {
	//		return err
	//	}
	//}

	return nil
}

// RemoveIndexes 根据工作区路径、文件路径，批量删除索引
func (i *Indexer) RemoveIndexes(workspace string, filePaths []string) error {
	// 根据 workspaceReader 及 filepath 路径，匹配 到project
	// 再到对应project中，删除对应的index
	// 以filepath 为key，先将index查询出来，遍历index的element，找到它被谁引用，然后更新对应的引用关系，再删除index

	return nil
}

// SaveIndexes 根据工作区路径、文件路径，批量保存索引
func (i *Indexer) SaveIndexes(workspace string, filePaths []string) error {
	// 根据 workspaceReader 及 filepath 路径，匹配 到project
	// 再到对应project中，保存对应的index，存在直接覆盖

	return nil
}

// QueryIndexes 查询索引
func (i *Indexer) QueryIndexes(workspace string, filePaths []string) error {
	// 根据workspace 及 filepath 路径，匹配 到project
	// 再到对应project中，查询对应的index

	return nil
}
