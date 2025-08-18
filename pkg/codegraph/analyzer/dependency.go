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
	"os"
	"strconv"
)

type DependencyAnalyzer struct {
	PackageClassifier *packageclassifier.PackageClassifier
	workspaceReader   workspace.WorkspaceReader
	logger            logger.Logger
	store             store.GraphStorage
	loadThreshold     int
}

func NewDependencyAnalyzer(logger logger.Logger,
	packageClassifier *packageclassifier.PackageClassifier,
	reader workspace.WorkspaceReader,
	store store.GraphStorage) *DependencyAnalyzer {

	return &DependencyAnalyzer{
		logger:            logger,
		PackageClassifier: packageClassifier,
		workspaceReader:   reader,
		store:             store,
		loadThreshold:     getLoadThreshold(),
	}
}

// defaultLoadThreshold
const defaultLoadThreshold = 9000 // 不存在则load，缓存key、value。

func getLoadThreshold() int {
	loadThreshold := defaultLoadThreshold
	if envVal, ok := os.LookupEnv("LOAD_THRESHOLD"); ok {
		if val, err := strconv.Atoi(envVal); err == nil && val > 0 {
			loadThreshold = val
		}
	}
	return loadThreshold
}

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
		return totalSavedSymbols, fmt.Errorf("batch save symbol definitions error: %w", err)
	}
	da.logger.Info("batch save symbols end, total element_tables %d, total elements %d ,load from db %d, load threshold %d",
		len(fileElementTables), totalSavedSymbols, totalLoad, da.loadThreshold)
	return totalSavedSymbols, nil
}

// loadSymbolOccurrenceByStrategy 根据策略加载，defaultLoadThreshold、level2、level3
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
				da.logger.Debug("unmarshal symbol occurrence err:%v", err)
			}
		} else if !errors.Is(err, store.ErrKeyNotFound) {
			da.logger.Debug("get symbol occurrence from db failed, value is zero length or err:%v", err)
		}
	}

	if !ok && totalFiles <= da.loadThreshold {
		load = true
		loadFromDB()
	}

	if symbol == nil {
		symbol = &codegraphpb.SymbolOccurrence{Name: key, Language: string(fileTable.Language),
			Occurrences: make([]*codegraphpb.Occurrence, 0)}
	}
	symbolCache.Put(key, symbol)

	return symbol, load
}

type RichElement struct {
	*codegraphpb.Element
	Path string
}

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
