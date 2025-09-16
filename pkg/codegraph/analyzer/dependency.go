package analyzer

import (
	packageclassifier "codebase-indexer/pkg/codegraph/analyzer/package_classifier"
	"codebase-indexer/pkg/codegraph/cache"
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
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/antlabs/strsim"
)

type DependencyAnalyzer struct {
	PackageClassifier     *packageclassifier.PackageClassifier
	workspaceReader       workspace.WorkspaceReader
	logger                logger.Logger
	store                 store.GraphStorage
	loadThreshold         int
	skipVariableThreshold int
}

func NewDependencyAnalyzer(logger logger.Logger,
	packageClassifier *packageclassifier.PackageClassifier,
	reader workspace.WorkspaceReader,
	store store.GraphStorage) *DependencyAnalyzer {

	return &DependencyAnalyzer{
		logger:                logger,
		PackageClassifier:     packageClassifier,
		workspaceReader:       reader,
		store:                 store,
		loadThreshold:         getLoadThresholdFromEnv(),
		skipVariableThreshold: getSkipVariableThresholdFromEnv(),
	}
}

const defaultLoadFromStoreThreshold = 9000 // 不存在则load，缓存key、value。

const defaultSkipVariableThreshold = 9000 // 不存在则load，缓存key、value。

func getLoadThresholdFromEnv() int {
	loadThreshold := defaultLoadFromStoreThreshold
	if envVal, ok := os.LookupEnv("LOAD_FROM_STORE_THRESHOLD"); ok {
		if val, err := strconv.Atoi(envVal); err == nil && val > 0 {
			loadThreshold = val
		}
	}
	return loadThreshold
}

func getSkipVariableThresholdFromEnv() int {
	skipVariableThreshold := defaultSkipVariableThreshold
	if envVal, ok := os.LookupEnv("SKIP_VARIABLE_THRESHOLD"); ok {
		if val, err := strconv.Atoi(envVal); err == nil && val > 0 {
			skipVariableThreshold = val
		}
	}
	return skipVariableThreshold
}

// SaveSymbolOccurrences 保存符号定义位置
func (da *DependencyAnalyzer) SaveSymbolOccurrences(ctx context.Context, projectUuid string, totalFiles int,
	fileElementTables []*parser.FileElementTable, symbolCache *cache.LRUCache[*codegraphpb.SymbolOccurrence]) (*types.IndexTaskMetrics, error) {
	taskMetrics := &types.IndexTaskMetrics{}
	if len(fileElementTables) == 0 {
		return taskMetrics, nil
	}
	// 2. 构建项目定义符号表  符号名 -> 元素列表，先根据符号名匹配，匹配符号名后，再根据导入路径、包名进行过滤。
	totalElements := 0
	totalVariables := 0
	totalElementsAfterFiltered := 0
	totalLoad, totalVariablesFiltered := 0, 0
	updatedSymbolOccurrences := make([]*codegraphpb.SymbolOccurrence, 0, 100)
	for _, fileTable := range fileElementTables {
		totalElements += len(fileTable.Elements)
		for _, element := range fileTable.Elements {
			switch element.(type) {
			// 处理定义
			case *resolver.Class, *resolver.Function, *resolver.Method, *resolver.Interface, *resolver.Variable:
				if element.GetType() == types.ElementTypeVariable {
					totalVariables++
					// 定义位置
					// 有些变量是函数类型（ts），只处理全局/包级变量
					if da.shouldSkipVariable(totalFiles, element) {
						totalVariablesFiltered++
						continue
					}
				}

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
				totalElementsAfterFiltered++
				// 引用位置
				// case *resolver.Reference, *resolver.Call:
			}
		}

	}
	taskMetrics.TotalSymbols = totalElements
	taskMetrics.TotalVariables = totalVariables
	// 3. 保存到存储中，后续查询使用
	if err := da.store.BatchSave(ctx, projectUuid, workspace.SymbolOccurrences(updatedSymbolOccurrences)); err != nil {
		return taskMetrics, fmt.Errorf("batch save symbol definitions error: %w", err)
	}
	taskMetrics.TotalSavedSymbols = totalElementsAfterFiltered
	taskMetrics.TotalSavedVariables = totalVariables - totalVariablesFiltered
	da.logger.Info("batch save symbols end, element_tables %d, total elements %d, after filtered %d, load from db %d, total variable %d, skipped %d, load threshold %d, skip threshold %d",
		len(fileElementTables), totalElements, totalElementsAfterFiltered, totalLoad, totalVariables, totalVariablesFiltered, da.loadThreshold, da.skipVariableThreshold)
	return taskMetrics, nil
}

func (da *DependencyAnalyzer) shouldSkipVariable(totalFiles int, element resolver.Element) bool {
	return totalFiles > da.skipVariableThreshold || (element.GetType() == types.ElementTypeVariable && (element.GetScope() != types.ScopePackage &&
		element.GetScope() != types.ScopeFile &&
		element.GetScope() != types.ScopeProject))
}

// loadSymbolOccurrenceByStrategy 根据策略加载，defaultLoadFromStoreThreshold、level2、level3
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
			continue
		}

		// 2、同包(同父路径)
		if utils.IsSameParentDir(def.Path, filePath) {
			found = append(found, def)
			continue
		}

		// 3、根据import判断，def的文件路径是否在imp的范围内
		for _, imp := range imports {
			if IsFilePathInImportPackage(def.Path, imp) {
				found = append(found, def)
				break
			}
		}
	}
	return found
}

func (da *DependencyAnalyzer) CalculateSymbolMatchScore(callerImports []*codegraphpb.Import, callerFilePath string, calleeFilePath string, calleeSymbolName string, callerSymbolName string) int {
	// 1、同文件
	if callerFilePath == calleeFilePath {
		return 100
	}

	// 2、同包(同父路径)
	if utils.IsSameParentDir(callerFilePath, calleeFilePath) {
		return 75
	}

	// 3、根据import判断，def的文件路径是否在imp的范围内
	for _, imp := range callerImports {
		if IsFilePathInImportPackage(calleeFilePath, imp) {
			return 50
		}
	}

	score := 0
	// 4、函数名可能有一定的关系
	similarity := strsim.Compare(calleeSymbolName, calleeSymbolName, strsim.JaroWinkler())
	score += int(similarity * 15)

	// 5、文件名理论上都有一定的关系（相似度）
	// 获取callee的文件名
	calleeFilePath = filepath.Base(calleeFilePath)
	calleeFileName := strings.TrimSuffix(calleeFilePath, filepath.Ext(calleeFilePath))

	// 获取caller的文件名
	callerFilePath = filepath.Base(callerFilePath)
	callerFileName := strings.TrimSuffix(callerFilePath, filepath.Ext(callerFilePath))

	similarity = strsim.Compare(calleeFileName, callerFileName, strsim.DiceCoefficient())
	score += int(similarity * 10)

	// 6、文件路径最长公共前缀
	packageLevel := calculatePackageLevel(callerFilePath, calleeFilePath)
	score += packageLevel
	return score
}
func calculatePackageLevel(callerPath string, calleePath string) int {
	callerPath = strings.ReplaceAll(callerPath, types.WindowsSeparator, types.Dot)
	callerPath = strings.ReplaceAll(callerPath, types.UnixSeparator, types.Dot)
	callerDir := filepath.Dir(callerPath)

	calleePath = strings.ReplaceAll(calleePath, types.WindowsSeparator, types.Dot)
	calleePath = strings.ReplaceAll(calleePath, types.UnixSeparator, types.Dot)
	calleeDir := filepath.Dir(calleePath)

	// 计算共同前缀长度
	commonPrefix := 0
	callerParts := strings.Split(callerDir, types.UnixSeparator)
	calleeParts := strings.Split(calleeDir, types.UnixSeparator)

	minLen := min(len(callerParts), len(calleeParts))
	for i := 0; i < minLen; i++ {
		if callerParts[i] == calleeParts[i] {
			commonPrefix++
		} else {
			break
		}
	}

	// 层级越近，分数越高
	return commonPrefix
}
