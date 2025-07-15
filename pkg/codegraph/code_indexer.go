package codegraph

// 代码索引器主系统
type Indexer struct {
	// parserFactory    *ParserFactory 语法解析
	// analyzer     Analyzer 依赖分析
	// storage          Storage 存储
}

func NewCodeIndexer() *Indexer {

	return &Indexer{
		// parserFactory:      parserFactory,
		// indexBuilder:       indexBuilder,
		// storage:            storage,
	}
}

func (i *Indexer) IndexProject(projectPath string) error {
	// 查找项目中的所有源文件
	//files, err := findSourceFiles(projectPath)
	//if err != nil {
	//	return err
	//}
	//
	//// 并行解析文件
	//parsedFiles := make([]*SourceFile, 0, len(files))
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
	//
	//return nil
	return nil
}
