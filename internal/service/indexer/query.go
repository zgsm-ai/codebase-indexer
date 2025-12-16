package indexer

import (
	"codebase-indexer/internal/errs"
	"codebase-indexer/pkg/codegraph/analyzer"
	"codebase-indexer/pkg/codegraph/lang"
	"codebase-indexer/pkg/codegraph/proto"
	"codebase-indexer/pkg/codegraph/proto/codegraphpb"
	"codebase-indexer/pkg/codegraph/resolver"
	"codebase-indexer/pkg/codegraph/store"
	"codebase-indexer/pkg/codegraph/types"
	"codebase-indexer/pkg/codegraph/workspace"
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// QueryReferences 实现查询接口
// 支持根据符号名全局查找引用
// 支持查询某个文件内的符号的引用
// 支持查询某个文件内的行范围的符号的引用
func (idx *Indexer) QueryReferences(ctx context.Context, opts *types.QueryReferenceOptions) ([]*types.RelationNode, error) {
	startTime := time.Now()
	filePath := opts.FilePath
	start, end := NormalizeLineRange(opts.StartLine, opts.EndLine, MaxQueryLineLimit)
	opts.StartLine = start
	opts.EndLine = end

	if filePath == types.EmptyString {
		if opts.SymbolName != types.EmptyString {
			// 根据符号名查询，根据SymbolName查询引用的位置
			return idx.queryReferencesBySymbolName(ctx, opts)
		}
		return nil, errs.NewMissingParamError("filePath")
	}

	if !filepath.IsAbs(filePath) {
		return nil, fmt.Errorf("param filePath must be absolute path")
	}
	project, err := idx.GetProjectByFilePath(ctx, opts.Workspace, filePath)
	if err != nil {
		return nil, err
	}
	projectUuid := project.Uuid
	defer func() {
		idx.logger.Info("Query_reference execution time: %d ms", time.Since(startTime).Milliseconds())
	}()

	// 1. 获取文件元素表
	fileElementTable, err := idx.getFileElementTableByPath(ctx, projectUuid, filePath)
	if err != nil {
		return nil, err
	}

	var definitions []*types.RelationNode
	var foundSymbols []*codegraphpb.Element

	// Find root symbols based on query options
	if opts.SymbolName != types.EmptyString {
		foundSymbols = idx.querySymbolsByName(fileElementTable, opts)
		idx.logger.Debug("Found %d symbols by name and line", len(foundSymbols))
	} else {
		foundSymbols = idx.querySymbolsByLines(ctx, fileElementTable, opts)
		idx.logger.Debug("Found %d symbols by position", len(foundSymbols))
	}

	// Check if any root symbols were found
	if len(foundSymbols) == 0 {
		idx.logger.Debug("symbol not found: name %s line %d:%d in document %s", opts.SymbolName,
			opts.StartLine, opts.EndLine, opts.FilePath)
		return definitions, nil
	}

	// root
	definitionNames := make(map[string]*types.RelationNode, len(foundSymbols))
	// 找定义节点，以定义节点为根节点进行深度遍历
	for _, s := range foundSymbols {
		// 定义作为根节点
		if !s.IsDefinition {
			continue
		}
		if s.ElementType != codegraphpb.ElementType_CLASS &&
			s.ElementType != codegraphpb.ElementType_INTERFACE &&
			s.ElementType != codegraphpb.ElementType_METHOD &&
			s.ElementType != codegraphpb.ElementType_FUNCTION {
			continue
		} // 只处理 类、接口、函数、方法
		// 是定义，查找它的引用。当前采用遍历的方式（通过import过滤）
		position := types.ToPosition(s.Range)
		def := &types.RelationNode{
			FilePath:   fileElementTable.Path,
			SymbolName: s.Name,
			Position:   &position,
			NodeType:   string(proto.ElementTypeFromProto(s.ElementType)),
			Children:   make([]*types.RelationNode, 0),
		}
		definitions = append(definitions, def)
		// TODO 未处理同名定义问题，存在覆盖情况
		definitionNames[s.Name] = def
	}
	if len(definitions) == 0 {
		return definitions, nil
	}
	// 找定义的所有引用，通过遍历所有文件的方式
	idx.findSymbolReferences(ctx, projectUuid, definitionNames, filePath)

	return definitions, nil
}

// queryReferencesBySymbolName 按符号名查询引用
func (idx *Indexer) queryReferencesBySymbolName(ctx context.Context, opts *types.QueryReferenceOptions) ([]*types.RelationNode, error) {
	startTime := time.Now()
	defer func() {
		idx.logger.Info("Query_reference execution time: %d ms", time.Since(startTime).Milliseconds())
	}()
	projects := idx.workspaceReader.FindProjects(ctx, opts.Workspace, true, workspace.DefaultVisitPattern)
	if len(projects) == 0 {
		return nil, fmt.Errorf("query references by symbol name [%s] failed, no project found in workspace %s", opts.SymbolName, opts.Workspace)
	}
	var defMap = make(map[string]*types.RelationNode)
	var definitions []*types.RelationNode
	findDefinition := func() {
		for _, project := range projects {
			// 查询同名定义
			iter := idx.storage.Iter(ctx, project.Uuid)
			defer iter.Close()
			for iter.Next() {
				key := iter.Key()
				if store.IsSymbolNameKey(key) {
					continue
				}
				var elementTable codegraphpb.FileElementTable
				if err := store.UnmarshalValue(iter.Value(), &elementTable); err != nil {
					idx.logger.Error("failed to unmarshal file element_table value, err: %v", err)
					continue
				}
				for _, elem := range elementTable.Elements {
					if !elem.IsDefinition {
						continue
					}
					if elem.ElementType != codegraphpb.ElementType_CLASS &&
						elem.ElementType != codegraphpb.ElementType_INTERFACE &&
						elem.ElementType != codegraphpb.ElementType_METHOD &&
						elem.ElementType != codegraphpb.ElementType_FUNCTION {
						continue
					}
					// 只处理 类、接口、函数、方法
					if elem.Name != opts.SymbolName {
						continue
					}
					def := &types.RelationNode{
						SymbolName: elem.Name,
						NodeType:   string(proto.ElementTypeFromProto(elem.ElementType)),
						Children:   make([]*types.RelationNode, 0),
					}
					definitions = append(definitions, def)
					// 相同的name得到的结果是一样的，所以不重复添加，找到一个就返回
					defMap[elem.Name] = def
					return
				}
			}
		}
	}
	findDefinition()
	if len(definitions) == 0 {
		// 提前返回，避免遍历所有文件
		return definitions, nil
	}
	for _, project := range projects {
		// 找定义的所有引用，通过遍历所有文件的方式
		idx.findSymbolReferences(ctx, project.Uuid, defMap, opts.FilePath)
	}
	return definitions, nil
}

// findSymbolReferences 查找符号被调用或引用的位置
func (idx *Indexer) findSymbolReferences(ctx context.Context, projectUuid string, definitionNames map[string]*types.RelationNode, filePath string) {
	iter := idx.storage.Iter(ctx, projectUuid)
	defer iter.Close()
	for iter.Next() {
		key := iter.Key()
		if store.IsSymbolNameKey(key) {
			continue
		}
		var elementTable codegraphpb.FileElementTable
		if err := store.UnmarshalValue(iter.Value(), &elementTable); err != nil {
			idx.logger.Error("failed to unmarshal file %s element_table value, err: %v", filePath, err)
			continue
		}
		// TODO 根据import 过滤

		for _, element := range elementTable.Elements {
			if element.IsDefinition {
				continue
			}
			if element.ElementType != codegraphpb.ElementType_REFERENCE &&
				element.ElementType != codegraphpb.ElementType_CALL {
				continue
			}
			// 引用
			if v, ok := definitionNames[element.Name]; ok {
				position := types.ToPosition(element.Range)
				v.Children = append(v.Children, &types.RelationNode{
					FilePath:   elementTable.Path,
					SymbolName: element.Name,
					Position:   &position,
					NodeType:   string(proto.ElementTypeFromProto(element.ElementType)),
				})
			}
		}
	}
}

// querySymbolsByLines 按位置查询 occurrence
func (idx *Indexer) querySymbolsByLines(ctx context.Context, fileTable *codegraphpb.FileElementTable,
	opts *types.QueryReferenceOptions) []*codegraphpb.Element {
	var nodes []*codegraphpb.Element
	if opts.StartLine <= 0 || opts.EndLine < opts.StartLine {
		idx.logger.Debug("query_symbol_by_line invalid opts startLine %d or endLine %d", opts.StartLine,
			opts.EndLine)
		return nodes
	}
	startLineRange, endLineRange := int32(opts.StartLine)-1, int32(opts.EndLine)-1

	for _, s := range fileTable.Elements {
		if !isValidRange(s.Range) {
			idx.logger.Debug("query_symbol_by_line invalid element %s %s position %v", fileTable.Path, s.Name, s.Range)
			continue
		}
		if startLineRange <= s.Range[0] && endLineRange >= s.Range[2] {
			nodes = append(nodes, s)
		}

	}
	return nodes
}

// querySymbolsByName 通过 symbolName + startLine
func (idx *Indexer) querySymbolsByName(doc *codegraphpb.FileElementTable, opts *types.QueryReferenceOptions) []*codegraphpb.Element {
	var nodes []*codegraphpb.Element
	queryName := opts.SymbolName
	// 根据名字和 行号， 找到symbol
	for _, s := range doc.Elements {
		// symbol 名字匹配
		if s.Name == queryName {
			nodes = append(nodes, s)
		}
	}
	return nodes
}

// QueryDefinitions 支持单符号全局查询、行号范围内的符号定义查询、代码片段内的符号定义查询
func (idx *Indexer) QueryDefinitions(ctx context.Context, opts *types.QueryDefinitionOptions) ([]*types.Definition, error) {
	// 参数验证
	if opts.Workspace == "" {
		return nil, fmt.Errorf("workspace cannot be empty")
	}
	if opts.FilePath == "" {
		// 查询符号可以不用文件路径
		// 不能超过默认最大批量查询符号定义数量，避免查询性能问题
		if opts.SymbolNames == "" {
			return nil, fmt.Errorf("file path cannot be empty")
		}
		symbolNames := make([]string, 0)
		for s := range strings.SplitSeq(opts.SymbolNames, ",") {
			idx := strings.LastIndex(s, ".")
			if idx != -1 {
				// types.QueryCallGraphOptions → QueryCallGraphOptions
				s = s[idx+1:] // 有点，取最后一个点后面
			}
			if t := strings.TrimSpace(s); t != "" {
				symbolNames = append(symbolNames, t)
			}
		}
		if len(symbolNames) > 0 {
			return idx.queryFuncDefinitionsBySymbolNames(ctx, opts.Workspace, symbolNames)
		}
		return nil, fmt.Errorf("file path cannot be empty")
	}
	if !filepath.IsAbs(opts.FilePath) {
		return nil, fmt.Errorf("param filePath must be absolute path")
	}
	_, err := lang.InferLanguage(opts.FilePath)
	if err != nil {
		return nil, errs.ErrUnSupportedLanguage
	}

	// 获取项目信息
	project, err := idx.GetProjectByFilePath(ctx, opts.Workspace, opts.FilePath)
	if err != nil {
		return nil, err
	}
	projectUuid := project.Uuid

	// 推断语言类型
	language, err := lang.InferLanguage(opts.FilePath)
	if err != nil {
		return nil, lang.ErrUnSupportedLanguage
	}

	// 性能监控
	startTime := time.Now()
	defer func() {
		idx.logger.Info("query func definitions cost %d ms", time.Since(startTime).Milliseconds())
	}()

	// 根据不同的查询模式处理
	// 优先根据SymbolName查询
	// 其次根据CodeSnippet查询
	// 最后根据行号范围查询
	switch {
	case len(opts.CodeSnippet) > 0:
		return idx.queryFuncDefinitionsBySnippet(ctx, project, language, opts.FilePath, opts.CodeSnippet)
	case opts.StartLine > 0 && opts.EndLine > 0:
		opts.StartLine, opts.EndLine = NormalizeLineRange(opts.StartLine, opts.EndLine, MaxQueryLineLimit)
		return idx.queryFuncDefinitionsByLineRange(ctx, projectUuid, language, opts)
	default:
		return nil, fmt.Errorf("invalid query definition options: at least one of CodeSnippet or line range must be provided")
	}
}

// queryFuncDefinitionsBySnippet 查询代码片段里面所有依赖的符号的定义
func (idx *Indexer) queryFuncDefinitionsBySnippet(ctx context.Context, project *workspace.Project, language lang.Language, filePath string, codeSnippet []byte) ([]*types.Definition, error) {
	parsedData, err := idx.parser.Parse(ctx, &types.SourceFile{
		Path:    filePath,
		Content: codeSnippet},
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
		return nil, nil
	}

	// TODO resolve go modules
	// 对imports预处理
	if filteredImps, err := idx.analyzer.PreprocessImports(ctx, language, project, imports); err == nil {
		imports = filteredImps
	}

	// 转化为protobuf格式
	var currentImports []*codegraphpb.Import
	for _, imp := range imports {
		currentImports = append(currentImports, &codegraphpb.Import{
			Name:   imp.Name,
			Alias:  imp.Alias,
			Source: imp.Source,
			Range:  imp.Range,
		})
	}

	// 根据所找到的call 的name + currentImports， 去模糊匹配symbol
	symDefs, err := idx.searchSymbolNames(ctx, project.Uuid, language, dependencyNames, currentImports)
	if err != nil {
		return nil, fmt.Errorf("failed to search index by names: %w", err)
	}
	if len(symDefs) == 0 {
		return nil, nil
	}

	// 封装返回结果
	var results []*types.Definition
	for name, def := range symDefs {
		for _, d := range def {
			if d == nil {
				continue
			}
			results = append(results, &types.Definition{
				Name:  name,
				Type:  string(proto.ElementTypeFromProto(d.ElementType)),
				Path:  d.Path,
				Range: d.Range,
			})
		}
	}
	return results, nil
}

// queryFuncDefinitionsByLineRange 通过行号范围查询函数定义
func (idx *Indexer) queryFuncDefinitionsByLineRange(ctx context.Context, projectUuid string, language lang.Language, opts *types.QueryDefinitionOptions) ([]*types.Definition, error) {
	// 首先查询出来范围内的所有符号
	var fileTable codegraphpb.FileElementTable
	fileTableBytes, err := idx.storage.Get(ctx, projectUuid, store.ElementPathKey{Language: language, Path: opts.FilePath})
	if errors.Is(err, store.ErrKeyNotFound) {
		return nil, fmt.Errorf("index not found for file %s", opts.FilePath)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get file %s index, err: %v", opts.FilePath, err)
	}
	if err = store.UnmarshalValue(fileTableBytes, &fileTable); err != nil {
		return nil, fmt.Errorf("failed to unmarshal file %s index value, err: %v", opts.FilePath, err)
	}

	// 查询范围内的所有符号
	queryStartLine := int32(opts.StartLine - 1)
	queryEndLine := int32(opts.EndLine - 1)
	foundSymbols := idx.findSymbolInDocByLineRange(ctx, &fileTable, queryStartLine, queryEndLine)
	currentImports := fileTable.Imports

	var results []*types.Definition
	for _, s := range foundSymbols {
		if s.IsDefinition {
			// 直接加入结果
			results = append(results, &types.Definition{
				Path:  opts.FilePath,
				Name:  s.Name,
				Range: s.Range,
				Type:  string(proto.ElementTypeFromProto(s.ElementType)),
			})
			continue
		} else {
			// 加载其他符号的定义
			bytes, err := idx.storage.Get(ctx, projectUuid, store.SymbolNameKey{Name: s.GetName(),
				Language: language})
			if err != nil {
				if !errors.Is(err, store.ErrKeyNotFound) {
					idx.logger.Debug("get symbol occurrence err:%v", err)
				}
				continue
			}
			var exist codegraphpb.SymbolOccurrence
			if err = store.UnmarshalValue(bytes, &exist); err != nil {
				idx.logger.Debug("unmarshal symbol occurrence err:%v", err)
				continue
			}

			// TODO 过滤效果待定
			filtered := idx.analyzer.FilterByImports(opts.FilePath, currentImports, exist.Occurrences)
			if len(filtered) == 0 {
				// 防止全部过滤掉
				filtered = exist.Occurrences
			}
			for _, o := range filtered {
				results = append(results, &types.Definition{
					Path:  o.Path,
					Name:  s.Name,
					Range: o.Range,
					Type:  string(proto.ToDefinitionElementType(proto.ElementTypeFromProto(s.ElementType))),
				})
			}
		}
	}

	// 最后返回结果
	return results, nil
}

// queryFuncDefinitionsBySymbolName 通过符号名查询函数定义
func (idx *Indexer) queryFuncDefinitionsBySymbolNames(ctx context.Context, workspacePath string, symbolNames []string) ([]*types.Definition, error) {
	// 遍历所有的语言，查询该符号的Occurrence
	var results []*types.Definition
	languages := lang.GetAllSupportedLanguages()
	projects := idx.workspaceReader.FindProjects(ctx, workspacePath, true, workspace.DefaultVisitPattern)
	if len(projects) == 0 {
		return nil, fmt.Errorf("query definitions by symbol names [%v] failed, no project found in workspace %s", symbolNames, workspacePath)
	}
	for _, project := range projects {
		for _, language := range languages {
			for _, symbolName := range symbolNames {
				bytes, err := idx.storage.Get(ctx, project.Uuid, store.SymbolNameKey{Name: symbolName,
					Language: language})
				if err != nil {
					continue
				}
				var exist codegraphpb.SymbolOccurrence
				if err = store.UnmarshalValue(bytes, &exist); err != nil {
					return nil, err
				}
				// 根据Occurrence信息封装为定义
				for _, o := range exist.Occurrences {
					results = append(results, &types.Definition{
						Path:  o.Path,
						Name:  symbolName,
						Range: o.Range,
						Type:  string(proto.ToDefinitionElementType(proto.ElementTypeFromProto(o.ElementType))),
					})
				}
			}
		}
	}
	return results, nil
}

// searchSymbolNames 搜索符号名
func (idx *Indexer) searchSymbolNames(ctx context.Context, projectUuid string, language lang.Language, names []string, imports []*codegraphpb.Import) (
	map[string][]*codegraphpb.Occurrence, error) {

	start := time.Now()
	// 去重
	uniqueNames := make(map[string]bool)
	var deduped []string
	for _, name := range names {
		if !uniqueNames[name] {
			uniqueNames[name] = true
			deduped = append(deduped, name)
		}
	}
	names = deduped
	found := make(map[string][]*codegraphpb.Occurrence)

	for _, name := range names {

		bytes, err := idx.storage.Get(ctx, projectUuid, store.SymbolNameKey{
			Language: language,
			Name:     name,
		})

		if err != nil {
			continue
		}

		var symbolOccurrence codegraphpb.SymbolOccurrence
		if err := store.UnmarshalValue(bytes, &symbolOccurrence); err != nil {
			return nil, fmt.Errorf("failed to deserialize index: %w", err)
		}

		if len(symbolOccurrence.Occurrences) == 0 {
			continue
		}

		if _, ok := found[name]; !ok {
			found[name] = make([]*codegraphpb.Occurrence, 0)
		}
		found[name] = append(found[name], symbolOccurrence.Occurrences...)
	}

	total := 0
	for _, v := range found {
		total += len(v)
	}

	if len(imports) > 0 {
		for k, v := range found {
			filtered := make([]*codegraphpb.Occurrence, 0, len(v))
			for _, occ := range v {
				for _, imp := range imports {
					if analyzer.IsFilePathInImportPackage(occ.Path, imp) {
						filtered = append(filtered, occ)
						break
					}
				}
			}
			found[k] = filtered
		}
	}

	idx.logger.Info("codegraph symbol name search end, cost %d ms, names count: %d, key found:%d",
		time.Since(start).Milliseconds(), len(names), total, len(found))
	return found, nil
}

// GetFileElementTable 通过工作区路径和文件路径获取FileElementTable（公开方法）
func (idx *Indexer) GetFileElementTable(ctx context.Context, workspacePath string, filePath string) (*codegraphpb.FileElementTable, error) {
	project, err := idx.GetProjectByFilePath(ctx, workspacePath, filePath)
	if err != nil {
		return nil, err
	}
	return idx.getFileElementTableByPath(ctx, project.Uuid, filePath)
}

// queryElements 查询elements
func (idx *Indexer) queryElements(ctx context.Context, workspacePath string, filePaths []string) ([]*codegraphpb.FileElementTable, error) {
	idx.logger.Info("start to query workspace %s files: %v", workspacePath, filePaths)

	projects := idx.workspaceReader.FindProjects(ctx, workspacePath, false, workspace.DefaultVisitPattern)
	if len(projects) == 0 {
		return nil, fmt.Errorf("no project found in workspace %s", workspacePath)
	}

	var results []*codegraphpb.FileElementTable
	var errs []error

	projectFilesMap, err := idx.groupFilesByProject(projects, filePaths)
	if err != nil {
		return nil, fmt.Errorf("group files by project failed: %w", err)
	}

	for puuid, pfiles := range projectFilesMap {
		for _, fp := range pfiles {
			language, err := lang.InferLanguage(fp)
			if err != nil {
				continue
			}
			fileTable, err := idx.storage.Get(context.Background(), puuid, store.ElementPathKey{Language: language, Path: fp})
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

	idx.logger.Info("query workspace %s files successfully, found %d elements", workspacePath, len(results))
	return results, nil
}

// querySymbols 查询symbols
func (idx *Indexer) querySymbols(ctx context.Context, workspacePath string, filePath string, symbolNames []string) ([]*codegraphpb.SymbolOccurrence, error) {
	idx.logger.Info("start to query workspace %s file %s symbols: %v", workspacePath, filePath, symbolNames)

	projects := idx.workspaceReader.FindProjects(ctx, workspacePath, false, workspace.DefaultVisitPattern)
	if len(projects) == 0 {
		return nil, fmt.Errorf("no project found in workspace %s", workspacePath)
	}

	var results []*codegraphpb.SymbolOccurrence
	var errs []error

	// 找到文件路径对应的项目
	_, targetProjectUuid, err := idx.findProjectForFile(projects, filePath)
	if err != nil {
		return nil, fmt.Errorf("find project for file failed: %w", err)
	}

	language, err := lang.InferLanguage(filePath)
	if err != nil {
		return nil, lang.ErrUnSupportedLanguage
	}
	// 查询每个符号名称
	for _, symbolName := range symbolNames {
		symbolDef, err := idx.storage.Get(context.Background(), targetProjectUuid,
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
		return results, errors.Join(errs...)
	}

	idx.logger.Info("query workspace %s file %s symbols successfully, found %d symbols",
		workspacePath, filePath, len(results))
	return results, nil
}
