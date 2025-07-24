package analyzer

import (
	"codebase-indexer/pkg/codegraph/parser"
	"codebase-indexer/pkg/codegraph/resolver"
	"codebase-indexer/pkg/codegraph/types"
	"codebase-indexer/pkg/codegraph/utils"
	"codebase-indexer/pkg/codegraph/workspace"
	"codebase-indexer/pkg/logger"
	"context"
	"strings"
)

type ProjectElementTable struct {
	ProjectInfo       *workspace.ProjectInfo
	FileElementTables []*parser.FileElementTable
}

type DependencyAnalyzer struct {
	workspaceReader *workspace.WorkspaceReader
	logger          logger.Logger
}

func NewDependencyAnalyzer(logger logger.Logger, reader *workspace.WorkspaceReader) *DependencyAnalyzer {
	return &DependencyAnalyzer{
		logger:          logger,
		workspaceReader: reader,
	}
}

func (da *DependencyAnalyzer) Analyze(ctx context.Context, projectSymbolTable *ProjectElementTable) error {
	// 1. 处理 import
	if err := da.preprocessImport(ctx, projectSymbolTable); err != nil {
		da.logger.Error("analyze import error: %v", err)
	}

	// 2. 构建项目定义符号表  符号名 -> 元素列表，先根据符号名匹配，匹配符号名后，再根据导入路径、包名进行过滤。
	definitionSymbols := make(map[string][]resolver.Element)
	for _, fileTable := range projectSymbolTable.FileElementTables {
		// 处理定义
		for _, elem := range fileTable.Elements {
			switch elem.(type) {
			case *resolver.Class, *resolver.Function, *resolver.Method, *resolver.Variable, *resolver.Interface:
				key := elem.GetName()
				definitionSymbols[key] = append(definitionSymbols[key], elem)
			}
		}

	}

	// 3. 迭代符号表，去解析依赖关系。 需要区分跨文件依赖、当前文件引用。
	// 优先根据名字做匹配，匹配到多个，再根据作用域、导入、包、别名等信息进行二次过滤。
	for _, fileTable := range projectSymbolTable.FileElementTables {
		currentPath := fileTable.Path
		imports := fileTable.Imports
		for _, elem := range fileTable.Elements {
			switch e := elem.(type) {
			// 函数、方法调用
			case *resolver.Call:
				// todo 函数重载，考虑参数
				foundElements := da.findReferredElement(definitionSymbols, e.Name, currentPath, imports)

				if len(foundElements) == 0 {
					da.logger.Debug(" %s referred element not found", e.Name)
					continue
				}

				for _, d := range foundElements {
					bindRelation(e, d, resolver.RelationTypeReference, resolver.RelationTypeDefinition)
				}

			// 类、结构体引用
			case *resolver.Reference:
				foundElements := da.findReferredElement(definitionSymbols, e.Name, currentPath, imports)

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
					foundElements := da.findReferredElement(definitionSymbols, superClassName, currentPath, imports)
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
					foundElements := da.findReferredElement(definitionSymbols, superInterfaceName, currentPath, imports)
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
					foundElements := da.findReferredElement(definitionSymbols, superInterfaceName, currentPath, imports)
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

func (da *DependencyAnalyzer) findReferredElement(definitionSymbols map[string][]resolver.Element,
	referredName string,
	currentPath string,
	imports []*resolver.Import,
) []resolver.Element {

	foundDef := make([]resolver.Element, 0)

	if defs, ok := definitionSymbols[referredName]; ok {
		// 同名的所有定义
		for _, def := range defs {
			if def.GetPath() == types.EmptyString {
				da.logger.Debug("dependency_analyzer definition symbol %s path is empty", def.GetName())
			}
			// 1、同文件
			if def.GetPath() == currentPath {
				foundDef = append(foundDef, def)
				break
			}

			// 2、同包(同父路径)
			if utils.IsSameParentDir(def.GetPath(), currentPath) {
				foundDef = append(foundDef, def)
				break
			}

			// 3、根据import，当前def的路径包含imp的路径
			for _, imp := range imports {
				if strings.Contains(def.GetPath(), imp.Source) ||
					strings.Contains(def.GetPath(), imp.Name) {
					foundDef = append(foundDef, def)
					break
				}
			}
		}
	}
	return foundDef
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
