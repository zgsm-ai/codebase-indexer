package analyzer

import (
	"codebase-indexer/pkg/codegraph/parser"
	"codebase-indexer/pkg/codegraph/proto/codegraphpb"
	"codebase-indexer/pkg/codegraph/resolver"
	"codebase-indexer/pkg/codegraph/store"
	"codebase-indexer/pkg/codegraph/types"
	"codebase-indexer/pkg/codegraph/utils"
	"codebase-indexer/pkg/codegraph/workspace"
	"codebase-indexer/pkg/logger"
	"context"
	"fmt"
	"strings"
)

type DependencyAnalyzer struct {
	workspaceReader *workspace.WorkspaceReader
	logger          logger.Logger
	store           store.GraphStorage
}

func NewDependencyAnalyzer(logger logger.Logger,
	reader *workspace.WorkspaceReader,
	store store.GraphStorage) *DependencyAnalyzer {

	return &DependencyAnalyzer{
		logger:          logger,
		workspaceReader: reader,
		store:           store,
	}
}

func (da *DependencyAnalyzer) Analyze(ctx context.Context,
	projectInfo *workspace.Project, fileElementTables []*parser.FileElementTable) error {
	projectUuid, err := projectInfo.Uuid()
	if err != nil {
		return err
	}
	// 1. 处理 import
	if err = da.preprocessImport(ctx, projectInfo, fileElementTables); err != nil {
		da.logger.Error("analyze import error: %v", err)
	}

	// 2. 迭代符号表，去解析依赖关系。 需要区分跨文件依赖、当前文件引用。
	// 优先根据名字做匹配，匹配到多个，再根据作用域、导入、包、别名等信息进行二次过滤。
	for _, fileTable := range fileElementTables {
		currentPath := fileTable.Path
		imports := fileTable.Imports
		for _, elem := range fileTable.Elements {
			switch e := elem.(type) {
			// 函数、方法调用
			case *resolver.Call:
				// todo 函数重载，考虑参数
				// TODO 先通过参数数量过滤
				foundElements, err := da.findReferredElement(ctx, projectUuid, e.Name, currentPath, imports)
				if err != nil {
					da.logger.Error("dependency_analyze find referred element %s err:%v", err)
					continue
				}
				if len(foundElements) == 0 {
					da.logger.Debug("dependency_analyze %s referred element not found", e.Name)
					continue
				}

				for _, d := range foundElements {
					bindRelation(e, d, resolver.RelationTypeReference, resolver.RelationTypeDefinition)
				}

			// 类、结构体引用
			case *resolver.Reference:
				foundElements, err := da.findReferredElement(ctx, projectUuid, e.Name, currentPath, imports)
				if err != nil {
					da.logger.Error("dependency_analyze find referred element %s err:%v", e.Name, err)
					continue
				}
				if len(foundElements) == 0 {
					da.logger.Debug(" %s referred element not found", e.Name)
					continue
				}

				for _, d := range foundElements {
					bindRelation(e, d, resolver.RelationTypeReference, resolver.RelationTypeDefinition)
				}

			// 类继承、实现
			case *resolver.Class:
				// Handle inheritance
				for _, superClassName := range e.SuperClasses {
					foundElements, err := da.findReferredElement(ctx, projectUuid, superClassName, currentPath, imports)
					if err != nil {
						da.logger.Error("dependency_analyze find referred element %s err:%v", superClassName, err)
						continue
					}
					if len(foundElements) == 0 {
						da.logger.Debug(" %s referred element not found", e.Name)
						continue
					}
					for _, d := range foundElements {
						bindRelation(e, d, resolver.RelationTypeInherit, resolver.RelationTypeSuperClass)
					}
				}

				// Handle implementation
				for _, superInterfaceName := range e.SuperInterfaces {
					foundElements, err := da.findReferredElement(ctx, projectUuid, superInterfaceName, currentPath, imports)
					if err != nil {
						da.logger.Error("dependency_analyze find referred element %s err:%v", superInterfaceName, err)
						continue
					}
					if len(foundElements) == 0 {
						da.logger.Debug(" %s referred element not found", e.Name)
						continue
					}
					for _, d := range foundElements {
						bindRelation(e, d, resolver.RelationTypeImplement, resolver.RelationTypeSuperInterface)
					}
				}

			// 接口继承
			case *resolver.Interface:
				// Handle interface extension
				for _, superInterfaceName := range e.SuperInterfaces {
					foundElements, err := da.findReferredElement(ctx, projectUuid, superInterfaceName, currentPath, imports)
					if err != nil {
						da.logger.Error("dependency_analyze find referred element %s err:%v", superInterfaceName, err)
						continue
					}
					if len(foundElements) == 0 {
						da.logger.Debug(" %s referred element not found", e.Name)
						continue
					}
					for _, d := range foundElements {
						bindRelation(e, d, resolver.RelationTypeInherit, resolver.RelationTypeSuperInterface)
					}
				}
			}
		}
	}

	return nil
}

// SaveSymbolDefinitions 保存符号定义位置
func (da *DependencyAnalyzer) SaveSymbolDefinitions(ctx context.Context, projectUuid string,
	fileElementTables []*parser.FileElementTable) error {
	// 2. 构建项目定义符号表  符号名 -> 元素列表，先根据符号名匹配，匹配符号名后，再根据导入路径、包名进行过滤。
	definitionSymbolsMap := make(map[string]*codegraphpb.SymbolDefinition)
	for _, fileTable := range fileElementTables {
		// 处理定义
		for _, elem := range fileTable.Elements {
			switch elem.(type) {
			case *resolver.Class, *resolver.Function, *resolver.Method, *resolver.Variable, *resolver.Interface:
				key := elem.GetName()
				d, ok := definitionSymbolsMap[key]
				if !ok {
					d = &codegraphpb.SymbolDefinition{Name: key, Definitions: make([]*codegraphpb.Definition, 0)}
				}
				d.Definitions = append(d.Definitions, &codegraphpb.Definition{
					Path:        fileTable.Path,
					Range:       elem.GetRange(),
					ElementType: types.ElementTypeToProto(elem.GetType()),
				})
				definitionSymbolsMap[key] = d
			}
		}
	}
	definitionSymbols := make([]*codegraphpb.SymbolDefinition, 0)
	for _, d := range definitionSymbolsMap {
		definitionSymbols = append(definitionSymbols, d)
	}

	// 3. 保存到存储中，后续查询使用
	if err := da.store.BatchSave(ctx, projectUuid, workspace.SymbolDefinitions(definitionSymbols)); err != nil {
		return fmt.Errorf("dependency_analyze batch save symbol definitions error: %w", err)
	}
	return nil
}

func (da *DependencyAnalyzer) findReferredElement(ctx context.Context,
	projectUuid string,
	referredName string,
	currentPath string,
	imports []*resolver.Import,
) ([]resolver.Element, error) {

	foundDef := make([]resolver.Element, 0)

	value, err := da.store.Get(ctx, projectUuid, store.SymbolNameKey(referredName))
	if err != nil {
		return nil, fmt.Errorf("dependency_analyzer get symbol definitions error: %w", err)
	}
	symbolDefs := value.(*codegraphpb.SymbolDefinition)

	// 同名的所有定义
	for _, def := range symbolDefs.Definitions {
		element := &resolver.BaseElement{
			Name:  referredName,
			Path:  def.Path,
			Type:  types.ElementTypeFromProto(def.ElementType),
			Range: def.Range,
		}
		if def.Path == types.EmptyString {
			da.logger.Debug("dependency_analyzer definition symbol %s path is empty", referredName)
		}

		// 1、同文件
		if def.Path == currentPath {
			foundDef = append(foundDef, element)
			break
		}

		// 2、同包(同父路径)
		if utils.IsSameParentDir(def.Path, currentPath) {
			foundDef = append(foundDef, element)
			break
		}

		// 3、根据import，当前def的路径包含imp的路径
		for _, imp := range imports {
			if strings.Contains(def.Path, imp.Source) ||
				strings.Contains(def.Path, imp.Name) {
				foundDef = append(foundDef, element)
				break
			}
		}
	}
	return foundDef, nil
}

// bindRelation establishes a bi-directional relationship between two elements.
func bindRelation(from resolver.Element, to resolver.Element,
	fromRelType resolver.RelationType, toRelType resolver.RelationType) {

	// Add relation from -> to
	from.SetRelations(append(from.GetRelations(), &resolver.Relation{
		ElementName:  to.GetName(),
		ElementPath:  to.GetPath(),
		Range:        to.GetRange(),
		RelationType: fromRelType,
	}))

	// Add relation to -> from
	to.SetRelations(append(to.GetRelations(), &resolver.Relation{
		ElementName:  from.GetName(),
		ElementPath:  from.GetPath(),
		Range:        from.GetRange(),
		RelationType: toRelType,
	}))
}
