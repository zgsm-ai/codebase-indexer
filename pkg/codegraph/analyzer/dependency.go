package analyzer

import (
	packageclassifier "codebase-indexer/pkg/codegraph/analyzer/package_classifier"
	"codebase-indexer/pkg/codegraph/cache"
	"codebase-indexer/pkg/codegraph/parser"
	"codebase-indexer/pkg/codegraph/proto"
	"codebase-indexer/pkg/codegraph/proto/codegraphpb"
	"codebase-indexer/pkg/codegraph/resolver"
	"codebase-indexer/pkg/codegraph/store"
	"codebase-indexer/pkg/codegraph/utils"
	"codebase-indexer/pkg/codegraph/workspace"
	"codebase-indexer/pkg/logger"
	"context"
	"errors"
	"fmt"
)

type DependencyAnalyzer struct {
	PackageClassifier *packageclassifier.PackageClassifier
	workspaceReader   *workspace.WorkspaceReader
	logger            logger.Logger
	store             store.GraphStorage
}

func NewDependencyAnalyzer(logger logger.Logger,
	packageClassifier *packageclassifier.PackageClassifier,
	reader *workspace.WorkspaceReader,
	store store.GraphStorage) *DependencyAnalyzer {
	return &DependencyAnalyzer{
		logger:            logger,
		PackageClassifier: packageClassifier,
		workspaceReader:   reader,
		store:             store,
	}
}

// func (da *DependencyAnalyzer) Analyze(ctx context.Context, projectUuid string,
//
//		fromTable *codegraphpb.FileElementTable,
//		cache *cache.LRUCache[*codegraphpb.FileElementTable]) ([]*codegraphpb.FileElementTable, error) {
//
//		deduplicateUpdateTables := map[string]*codegraphpb.FileElementTable{fromTable.Path: fromTable}
//		cache.Put(fromTable.Path, fromTable)
//		// 迭代符号表，去解析依赖关系。 需要区分跨文件依赖、当前文件引用。
//		// 优先根据名字做匹配，匹配到多个，再根据作用域、导入、包、别名等信息进行二次过滤。
//		currentPath := fromTable.Path
//		imports := fromTable.Imports
//		//TODO relation 避免重复添加
//		for _, e := range fromTable.Elements {
//			extraData := e.ExtraData
//
//			switch e.ElementType {
//			// 函数、方法调用
//			case codegraphpb.ElementType_CALL:
//				// todo 函数重载，考虑参数
//				// TODO 先通过参数数量过滤
//				referredElements, err := da.findReferredElement(ctx, projectUuid, e.Name, currentPath, imports)
//				if err != nil {
//					da.logger.Debug("dependency_analyze find call %s referred element err:%v", e.Name, err)
//					continue
//				}
//				if len(referredElements) == 0 {
//					continue
//				}
//
//				for _, d := range referredElements {
//					toElement, toTable := da.findOrLoadElementTable(ctx, projectUuid, d, fromTable, cache)
//					if toTable != nil {
//						deduplicateUpdateTables[toTable.Path] = toTable
//					}
//
//					bindRelation(&RichElement{Element: e, Path: currentPath}, toElement,
//						codegraphpb.RelationType_RELATION_TYPE_DEFINITION,
//						codegraphpb.RelationType_RELATION_TYPE_REFERENCE)
//				}
//
//			// 类、结构体引用
//			case codegraphpb.ElementType_REFERENCE:
//				foundElements, err := da.findReferredElement(ctx, projectUuid, e.Name, currentPath, imports)
//				if err != nil {
//					da.logger.Debug("dependency_analyze find reference %s referred element err:%v", e.Name, err)
//					continue
//				}
//				if len(foundElements) == 0 {
//					continue
//				}
//
//				for _, d := range foundElements {
//					toElement, toTable := da.findOrLoadElementTable(ctx, projectUuid, d, fromTable, cache)
//					if toTable != nil {
//						deduplicateUpdateTables[toTable.Path] = toTable
//					}
//					bindRelation(&RichElement{Element: e, Path: currentPath}, toElement,
//						codegraphpb.RelationType_RELATION_TYPE_DEFINITION,
//						codegraphpb.RelationType_RELATION_TYPE_REFERENCE)
//				}
//
//			// 类继承、实现
//			case codegraphpb.ElementType_CLASS:
//				superClasses, err := proto.GetSuperClassesFromExtraData(extraData)
//				if err != nil {
//					da.logger.Debug("dependency_analyze get super_classes from extra_data err:%v", err)
//				} else {
//					// Handle inheritance
//					for _, superClassName := range superClasses {
//						foundElements, err := da.findReferredElement(ctx, projectUuid, superClassName, currentPath, imports)
//						if err != nil {
//							da.logger.Debug("dependency_analyze find class %s referred element err:%v", e.Name, err)
//							continue
//						}
//						if len(foundElements) == 0 {
//							continue
//						}
//						for _, d := range foundElements {
//							toElement, toTable := da.findOrLoadElementTable(ctx, projectUuid, d, fromTable, cache)
//							if toTable != nil {
//								deduplicateUpdateTables[toTable.Path] = toTable
//							}
//							bindRelation(&RichElement{Element: e, Path: currentPath}, toElement,
//								codegraphpb.RelationType_RELATION_TYPE_SUPER_CLASS,
//								codegraphpb.RelationType_RELATION_TYPE_INHERIT)
//						}
//					}
//				}
//				superInterfaces, err := proto.GetSuperInterfacesFromExtraData(extraData)
//				if err != nil {
//					da.logger.Debug("dependency_analyze get super_interfaces from extra_data err:%v", err)
//				} else {
//					// Handle implementation
//					for _, superInterfaceName := range superInterfaces {
//						foundElements, err := da.findReferredElement(ctx, projectUuid, superInterfaceName, currentPath, imports)
//						if err != nil {
//							da.logger.Debug("dependency_analyze find class %s referred element err:%v", e.Name, err)
//							continue
//						}
//						if len(foundElements) == 0 {
//							continue
//						}
//						for _, d := range foundElements {
//							toElement, toTable := da.findOrLoadElementTable(ctx, projectUuid, d, fromTable, cache)
//							if toTable != nil {
//								deduplicateUpdateTables[toTable.Path] = toTable
//							}
//							bindRelation(&RichElement{Element: e, Path: currentPath}, toElement,
//								codegraphpb.RelationType_RELATION_TYPE_SUPER_INTERFACE,
//								codegraphpb.RelationType_RELATION_TYPE_IMPLEMENT)
//						}
//					}
//				}
//
//			// 接口继承
//			case codegraphpb.ElementType_INTERFACE:
//				superInterfaces, err := proto.GetSuperInterfacesFromExtraData(extraData)
//				if err != nil {
//					da.logger.Debug("dependency_analyze get super_interfaces from extra_data err:%v", err)
//				} else {
//					// Handle interface extension
//					for _, superInterfaceName := range superInterfaces {
//						foundElements, err := da.findReferredElement(ctx, projectUuid, superInterfaceName, currentPath, imports)
//						if err != nil {
//							da.logger.Debug("dependency_analyze find interface %s referred element err:%v", e.Name, err)
//							continue
//						}
//						if len(foundElements) == 0 {
//							continue
//						}
//						for _, d := range foundElements {
//							toElement, toTable := da.findOrLoadElementTable(ctx, projectUuid, d, fromTable, cache)
//							if toTable != nil {
//								deduplicateUpdateTables[toTable.Path] = toTable
//							}
//							bindRelation(&RichElement{Element: e, Path: currentPath}, toElement,
//								codegraphpb.RelationType_RELATION_TYPE_SUPER_INTERFACE,
//								codegraphpb.RelationType_RELATION_TYPE_IMPLEMENT)
//						}
//					}
//				}
//			}
//		}
//		updateElementTables := make([]*codegraphpb.FileElementTable, 0, len(deduplicateUpdateTables))
//		for _, t := range deduplicateUpdateTables {
//			updateElementTables = append(updateElementTables, t)
//		}
//		return updateElementTables, nil
//	}
//
// func (da *DependencyAnalyzer) findOrLoadElementTable(ctx context.Context,
//
//		projectUuid string, toElement *RichElement, fromTable *codegraphpb.FileElementTable,
//		cache *cache.LRUCache[*codegraphpb.FileElementTable]) (*RichElement, *codegraphpb.FileElementTable) {
//		// 找到d 对应的fileElementTable，和它真实的element，进行更新
//		var toElementTable *codegraphpb.FileElementTable
//		if table, ok := cache.Get(toElement.Path); ok {
//			toElementTable = table
//		}
//		if toElementTable == nil {
//			bytes, err := da.store.Get(ctx, projectUuid, store.ElementPathKey{Path: toElement.Path, Language: lang.Language(fromTable.Language)})
//			if err != nil {
//				da.logger.Debug("dependency_analyze get referred element %s file_element_table err:%v", toElement.Path, err)
//			} else {
//				var dFileTable codegraphpb.FileElementTable
//				if err := store.UnmarshalValue(bytes, &dFileTable); err != nil {
//					da.logger.Debug("dependency_analyze unmarshal referred element %s file_element_table err:%v", toElement.Path, err)
//				} else {
//					toElementTable = &dFileTable
//				}
//			}
//		}
//
//		if toElementTable == nil {
//			return nil, nil
//		}
//
//		cache.Put(toElement.Path, toElementTable)
//
//		found := false
//		for _, element := range toElementTable.Elements {
//			if element.Name == toElement.Name && utils.SliceEqual(element.Range, toElement.Range) {
//				toElement = &RichElement{Element: element, Path: toElement.Path}
//				found = true
//				break
//			}
//		}
//		if found {
//			return toElement, toElementTable
//		}
//		return toElement, nil
//	}
//

// SaveSymbolOccurrences 保存符号定义位置
func (da *DependencyAnalyzer) SaveSymbolOccurrences(ctx context.Context, projectUuid string, totalFiles int,
	fileElementTables []*parser.FileElementTable, symbolCache *cache.LRUCache[*codegraphpb.SymbolOccurrence]) (int, error) {
	if len(fileElementTables) == 0 {
		return 0, nil
	}
	// 2. 构建项目定义符号表  符号名 -> 元素列表，先根据符号名匹配，匹配符号名后，再根据导入路径、包名进行过滤。
	totalSavedSymbols := 0
	totalLoad := 0
	updatedSymbolOccurrences := make([]*codegraphpb.SymbolOccurrence, 0, 100)
	for _, fileTable := range fileElementTables {
		for _, element := range fileTable.Elements {
			switch element.(type) {
			// 处理定义
			case *resolver.Class, *resolver.Function, *resolver.Method, *resolver.Interface:
				// 定义位置
				// 跳过局部作用域的变量 不处理变量
				//if element.GetType() == types.ElementTypeVariable && (element.GetScope() == types.ScopeBlock ||
				//	element.GetScope() == types.ScopeFunction) {
				//	continue
				//}

				symbol, load := da.loadSymbolOccurrenceByStrategy(ctx, projectUuid, totalFiles, element, symbolCache, fileTable)
				if load {
					totalLoad++
				}
				symbol.Occurrences = append(symbol.Occurrences, &codegraphpb.Occurrence{
					Path:        fileTable.Path,
					Range:       element.GetRange(),
					ElementType: proto.ElementTypeToProto(element.GetType()),
				})

				updatedSymbolOccurrences = append(updatedSymbolOccurrences, symbol)
				totalSavedSymbols++
				// TODO 引用位置
				// case *resolver.Reference, *resolver.Call:
			}
		}

	}

	// 3. 保存到存储中，后续查询使用
	if err := da.store.BatchSave(ctx, projectUuid, workspace.SymbolOccurrences(updatedSymbolOccurrences)); err != nil {
		return totalSavedSymbols, fmt.Errorf("dependency_analyze batch save symbol definitions error: %w", err)
	}
	da.logger.Debug("dependency_analyze batch save symbols end, total element_tables %d, total elements %d ,load from db %d",
		len(fileElementTables), totalSavedSymbols, totalLoad)
	return totalSavedSymbols, nil
}

// loadThreshold
const loadThreshold = 9000 // 不存在则load，缓存key、value。

// loadSymbolOccurrenceByStrategy 根据策略加载，loadThreshold、level2、level3
func (da *DependencyAnalyzer) loadSymbolOccurrenceByStrategy(ctx context.Context,
	projectUuid string,
	totalFiles int,
	elem resolver.Element,
	symbolCache *cache.LRUCache[*codegraphpb.SymbolOccurrence],
	fileTable *parser.FileElementTable) (*codegraphpb.SymbolOccurrence, bool) {
	load := false
	key := elem.GetName()
	// TODO 同名处理：按文件数采取降级措施
	symbol, ok := symbolCache.Get(key)

	loadFromDB := func() {
		nameKey := store.SymbolNameKey{Name: key, Language: fileTable.Language}
		bytes, err := da.store.Get(ctx, projectUuid, nameKey)
		if err == nil && len(bytes) > 0 {
			var exist codegraphpb.SymbolOccurrence
			if err := store.UnmarshalValue(bytes, &exist); err == nil {
				newOccurrences := make([]*codegraphpb.Occurrence, 0)
				// 去重，删除 path 和 range相同的
				for _, o := range exist.Occurrences {
					if o.Path == fileTable.Path && utils.SliceEqual(o.Range, elem.GetRange()) { // 去重
						continue
					}
					newOccurrences = append(newOccurrences, o)
				}
				exist.Occurrences = newOccurrences
				symbol = &exist
			} else {
				da.logger.Debug("dependency_analyze unmarshal symbol occurrence err:%v", err)
			}
		} else if !errors.Is(err, store.ErrKeyNotFound) {
			da.logger.Debug("dependency_analyze get symbol occurrence from db failed, value is zero length or err:%v", err)
		}
	}

	if !ok && totalFiles <= loadThreshold {
		load = true
		loadFromDB()
	}

	if symbol == nil {
		symbol = &codegraphpb.SymbolOccurrence{Name: key, Language: string(fileTable.Language),
			Occurrences: make([]*codegraphpb.Occurrence, 0)}
	}
	// 仅缓存名字
	symbolCache.Put(key, symbol)

	return symbol, load
}

type RichElement struct {
	*codegraphpb.Element
	Path string
}

//
//func (da *DependencyAnalyzer) findReferredElement(ctx context.Context,
//	projectUuid string,
//	referredName string,
//	currentPath string,
//	imports []*codegraphpb.Import,
//) ([]*RichElement, error) {
//	language, err := lang.InferLanguage(currentPath)
//	if err != nil {
//		return nil, nil
//	}
//
//	foundDef := make([]*RichElement, 0)
//
//	value, err := da.store.Get(ctx, projectUuid, store.SymbolNameKey{Language: language, Name: referredName})
//	if errors.Is(err, store.ErrKeyNotFound) {
//		return nil, nil
//	}
//	if err != nil {
//		return nil, fmt.Errorf("get symbol path %s name %s definitions error: %w", currentPath, referredName, err)
//	}
//	symbolDefs := new(codegraphpb.SymbolDefinition)
//	if err = store.UnmarshalValue(value, symbolDefs); err != nil {
//		return nil, err
//	}
//
//	// 同名的所有定义
//	for _, def := range symbolDefs.Definitions {
//		element := &RichElement{
//			Element: &codegraphpb.Element{
//				Name:        referredName,
//				ElementType: def.ElementType,
//				Range:       def.Range,
//			},
//			Path: def.Path,
//		}
//		if def.Path == types.EmptyString {
//			da.logger.Debug("dependency_analyzer definition symbol %s path is empty", referredName)
//		}
//
//		// 1、同文件
//		if def.Path == currentPath {
//			foundDef = append(foundDef, element)
//			break
//		}
//
//		// 2、同包(同父路径)
//		if utils.IsSameParentDir(def.Path, currentPath) {
//			foundDef = append(foundDef, element)
//			break
//		}
//
//		// 3、根据import，当前def的路径包含imp的路径
//		for _, imp := range imports {
//			if strings.Contains(def.Path, imp.Source) ||
//				strings.Contains(def.Path, imp.Name) {
//				foundDef = append(foundDef, element)
//				break
//			}
//		}
//	}
//	return foundDef, nil
//}
//
//// bindRelation establishes a bidirectional relationship between two elements.
//func bindRelation(from *RichElement, to *RichElement,
//	fromRelType codegraphpb.RelationType, toRelType codegraphpb.RelationType) {
//
//	// Add relation from -> to
//	from.Relations = append(from.GetRelations(), &codegraphpb.Relation{
//		ElementName:  to.GetName(),
//		ElementPath:  to.Path,
//		Range:        to.GetRange(),
//		RelationType: fromRelType,
//	})
//
//	// Add relation to -> from
//	to.Relations = append(to.GetRelations(), &codegraphpb.Relation{
//		ElementName:  from.GetName(),
//		ElementPath:  from.Path,
//		Range:        from.GetRange(),
//		RelationType: toRelType,
//	})
//}

func (da *DependencyAnalyzer) FilterByImports(filePath string, imports []*codegraphpb.Import,
	occurrences []*codegraphpb.Occurrence) []*codegraphpb.Occurrence {
	found := make([]*codegraphpb.Occurrence, 0)
	for _, def := range occurrences {
		// 1、同文件
		if def.Path == filePath {
			found = append(found, def)
			break
		}

		// 2、同包(同父路径)
		if utils.IsSameParentDir(def.Path, filePath) {
			found = append(found, def)
			break
		}

		// 3、根据import，当前def的路径包含imp的路径
		for _, imp := range imports {
			if IsImportPathInFilePath(imp, filePath) {
				found = append(found, def)
				break
			}
		}
	}
	return found
}
