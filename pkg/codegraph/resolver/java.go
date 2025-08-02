package resolver

import (
	"codebase-indexer/pkg/codegraph/types"
	"context"
	"fmt"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type JavaResolver struct {
}

var _ ElementResolver = &JavaResolver{}

func (j *JavaResolver) Resolve(ctx context.Context, element Element, rc *ResolveContext) ([]Element, error) {
	//复用resolve函数
	return resolve(ctx, j, element, rc)
}

func (j *JavaResolver) resolveImport(ctx context.Context, element *Import, rc *ResolveContext) ([]Element, error) {

	rootCap := rc.Match.Captures[0]
	updateRootElement(element, &rootCap, rc.CaptureNames[rootCap.Index], rc.SourceFile.Content)
	for _, cap := range rc.Match.Captures {
		captureName := rc.CaptureNames[cap.Index]
		if cap.Node.IsMissing() || cap.Node.IsError() {
			return nil, fmt.Errorf("import is missing or error")
		}
		content := cap.Node.Utf8Text(rc.SourceFile.Content)
		switch types.ToElementType(captureName) {
		case types.ElementTypeImportName:
			element.BaseElement.Name = content
		}
	}
	// 处理类导入
	elements := []Element{element}
	element.BaseElement.Scope = types.ScopePackage
	return elements, nil
}

func (j *JavaResolver) resolvePackage(ctx context.Context, element *Package, rc *ResolveContext) ([]Element, error) {
	rootCap := rc.Match.Captures[0]
	updateRootElement(element, &rootCap, rc.CaptureNames[rootCap.Index], rc.SourceFile.Content)

	for _, cap := range rc.Match.Captures {
		captureName := rc.CaptureNames[cap.Index]
		if cap.Node.IsMissing() || cap.Node.IsError() {
			return nil, fmt.Errorf("package is missing or error")
		}
		content := cap.Node.Utf8Text(rc.SourceFile.Content)
		switch types.ToElementType(captureName) {
		case types.ElementTypePackageName:
			element.BaseElement.Name = content
		}
	}
	element.BaseElement.Scope = types.ScopeProject
	// package不需要额外处理，直接返回
	return []Element{element}, nil
}

func (j *JavaResolver) resolveFunction(ctx context.Context, element *Function, rc *ResolveContext) ([]Element, error) {
	// TODO java中不存在单独的函数，暂时不实现
	return nil, fmt.Errorf("not implemented")
}

func (j *JavaResolver) resolveMethod(ctx context.Context, element *Method, rc *ResolveContext) ([]Element, error) {
	rootCap := rc.Match.Captures[0]
	updateRootElement(element, &rootCap, rc.CaptureNames[rootCap.Index], rc.SourceFile.Content)
	for _, cap := range rc.Match.Captures {
		captureName := rc.CaptureNames[cap.Index]
		if cap.Node.IsMissing() || cap.Node.IsError() {
			continue
		}
		content := cap.Node.Utf8Text(rc.SourceFile.Content)
		switch types.ElementType(captureName) {
		case types.ElementTypeMethodModifier:
			element.Declaration.Modifier = getElementModifier(content)
		case types.ElementTypeMethodName:
			element.BaseElement.Name = strings.TrimSpace(content)
			element.Declaration.Name = element.BaseElement.Name
		case types.ElementTypeMethodReturnType:
			element.Declaration.ReturnType = getFilteredTypes(content)
		case types.ElementTypeMethodParameters:
			element.Declaration.Parameters = getFilteredParameters(content)
		}
	}
	// 设置owner并且补充默认修饰符
	ownerNode := findMethodOwner(&rootCap.Node)
	var ownerKind types.NodeKind
	if ownerNode != nil {
		element.Owner = extractNodeName(ownerNode, rc.SourceFile.Content)
		ownerKind = types.ToNodeKind(ownerNode.Kind())
	}

	// 补充作用域
	element.BaseElement.Scope = getScopeFromModifiers(element.Declaration.Modifier, ownerKind)
	if element.Declaration.Modifier == types.EmptyString {
		switch ownerKind {
		case types.NodeKindClassDeclaration:
			element.Declaration.Modifier = types.PackagePrivate
		case types.NodeKindInterfaceDeclaration:
			element.Declaration.Modifier = types.PublicAbstract
		case types.NodeKindEnumDeclaration:
			element.Declaration.Modifier = types.PackagePrivate
		default:
			element.Declaration.Modifier = types.PackagePrivate
		}
	}
	return []Element{element}, nil
}

func (j *JavaResolver) resolveClass(ctx context.Context, element *Class, rc *ResolveContext) ([]Element, error) {
	rootCap := rc.Match.Captures[0]
	updateRootElement(element, &rootCap, rc.CaptureNames[rootCap.Index], rc.SourceFile.Content)
	var modifier string
	var refs = []*Reference{}
	var imports = []*Import{}
	for _, cap := range rc.Match.Captures {
		captureName := rc.CaptureNames[cap.Index]
		if cap.Node.IsMissing() || cap.Node.IsError() {
			continue
		}
		content := cap.Node.Utf8Text(rc.SourceFile.Content)
		elemType := types.ToElementType(captureName)
		switch elemType {
		case types.ElementTypeClassName, types.ElementTypeEnumName:
			// 解析类名/枚举名
			element.BaseElement.Name = content
		case types.ElementTypeClassExtends, types.ElementTypeClassImplements, types.ElementTypeEnumImplements:
			// 枚举的多继承、类的单继承、类的多实现
			typs := parseTypeList(&cap.Node, rc.SourceFile.Content)
			for _, typ := range typs {
				parts := strings.Split(typ, types.Dot)
				// owner 可能是包名，也可能是嵌套类的上层类调用
				owner := strings.Join(parts[:len(parts)-1], types.Dot)
				parent := parts[len(parts)-1]
				imports = append(imports, &Import{
					BaseElement: &BaseElement{
						Name: owner,
						Path: element.Path,
						Scope: types.ScopePackage,
						Type: types.ElementTypeImport,
						Range: element.Range,
					},
				})
				refs = append(refs, NewReference(element, &cap.Node, parent, owner))
				if elemType == types.ElementTypeClassExtends {
					element.SuperClasses = append(element.SuperClasses, parent)
				} else {
					element.SuperInterfaces = append(element.SuperInterfaces, parent)
				}
			}

		case types.ElementTypeClassModifiers:
			// 解析类的访问修饰符，并设置作用域
			// public、private、protected 或无修饰符
			// 无修饰符时，不走这个路径
			modifier = getElementModifier(content)
		}
	}
	element.BaseElement.Scope = getScopeFromModifiers(modifier, types.NodeKindClassDeclaration)
	
	elements := []Element{element}
	for _, r := range refs {
		elements = append(elements, r)
	}
	for _, i := range imports {
		elements = append(elements, i)
	}
	return elements, nil
}

func (j *JavaResolver) resolveVariable(ctx context.Context, element *Variable, rc *ResolveContext) ([]Element, error) {
	var refs = []*Reference{}
	rootCap := rc.Match.Captures[0]
	var elems []*Variable
	updateRootElement(element, &rootCap, rc.CaptureNames[rootCap.Index], rc.SourceFile.Content)
	for _, cap := range rc.Match.Captures {
		captureName := rc.CaptureNames[cap.Index]
		if cap.Node.IsMissing() || cap.Node.IsError() {
			continue
		}
		content := cap.Node.Utf8Text(rc.SourceFile.Content)
		kind := types.ToElementType(captureName)
		switch kind {
		case types.ElementTypeField, types.ElementTypeLocalVariable, types.ElementTypeEnumConstant:
			// 处理 int a=10,b,c=0的情况，a,b,c分别对应一个cap
			elem := &Variable{
				BaseElement: &BaseElement{
					Name: content,
					Path:  element.BaseElement.Path,
					Type:  types.ElementTypeVariable,
					Scope: types.ScopeClass,
					// 共用一套数据
					Range: element.BaseElement.Range,
				},
			}
			switch kind {
			case types.ElementTypeLocalVariable:
				elem.BaseElement.Scope = types.ScopeFunction
			case types.ElementTypeEnumConstant:
				elem.BaseElement.Scope = types.ScopeClass
				// 不关注枚举常量的类型，或者是可以由resolveCall解决，或者就是字面量
				elem.VariableType = []string{types.PrimitiveType}
			default:
				elem.BaseElement.Scope = types.ScopeClass
			}
			elems = append(elems, elem)
		case types.ElementTypeLocalVariableName, types.ElementTypeFieldName, types.ElementTypeEnumConstantName:
			// 用于处理这种 String managerName = "DefaultManager", managerVersion
			elems[len(elems)-1].BaseElement.Name = CleanParam(content)
		case types.ElementTypeLocalVariableType, types.ElementTypeFieldType:
			// TODO 可以优化，不用parseLocalVariableType这个函数
			// 左侧的类型声明
			//1. 标准类型 设置为primitive_type
			//2. 用户自定义或其他包里面的类型 设置为对应的类型
			// TODO 理论上来说是找这个类型及其子节点里面所有的type_identifier(排除包名的情况)
			// TODO 带包名的情况，还要抛出去一个import
			typs := parseLocalVariableType(&cap.Node, rc.SourceFile.Content)
			// 筛选出用户自定义的类型
			typs = types.FilterCustomTypes(typs)
			elems[len(elems)-1].VariableType = typs
			for _, typ := range typs {
				if typ == types.PrimitiveType {
					continue
				}
				// 自定义类型走引用
				refs = append(refs, NewReference(element, &cap.Node, typ, ""))
			}
		}
	}
	var elements []Element
	for _, v := range elems {
		elements = append(elements, v)
	}
	for _, r := range refs {
		elements = append(elements, r)
	}
	return elements, nil
}

func (j *JavaResolver) resolveInterface(ctx context.Context, element *Interface, rc *ResolveContext) ([]Element, error) {
	rootCap := rc.Match.Captures[0]
	updateRootElement(element, &rootCap, rc.CaptureNames[rootCap.Index], rc.SourceFile.Content)
	var modifier string
	var refs = []*Reference{}
	var imports = []*Import{}
	for _, cap := range rc.Match.Captures {
		captureName := rc.CaptureNames[cap.Index]
		if cap.Node.IsMissing() || cap.Node.IsError() {
			continue
		}
		content := cap.Node.Utf8Text(rc.SourceFile.Content)
		switch types.ToElementType(captureName) {
		case types.ElementTypeInterfaceName:
			element.BaseElement.Name = content
		case types.ElementTypeInterfaceModifiers:
			modifier = getElementModifier(content)
		case types.ElementTypeInterfaceExtends:
			typs := parseTypeList(&cap.Node, rc.SourceFile.Content)
			
			for _, typ := range typs {
				parts := strings.Split(typ, types.Dot)
				// owner 可能是包名，也可能是嵌套类的上层类调用
				owner := strings.Join(parts[:len(parts)-1], types.Dot)
				parent := parts[len(parts)-1]
				imports = append(imports, &Import{
					BaseElement: &BaseElement{
						Name: owner,
						Path: element.Path,
						Scope: types.ScopePackage,
						Type: types.ElementTypeImport,
						Range: element.Range,
					},
				})
				element.SuperInterfaces = append(element.SuperInterfaces, parent)
				refs = append(refs, NewReference(element, &cap.Node, parent,owner))
			}
		}

	}
	element.BaseElement.Scope = getScopeFromModifiers(modifier, types.NodeKindInterfaceDeclaration)
	elements := []Element{element}
	for _, r := range refs {
		elements = append(elements, r)
	}
	for _, i := range imports {
		elements = append(elements, i)
	}
	return elements, nil
}

func (j *JavaResolver) resolveCall(ctx context.Context, element *Call, rc *ResolveContext) ([]Element, error) {
	//
	rootCap := rc.Match.Captures[0]
	updateRootElement(element, &rootCap, rc.CaptureNames[rootCap.Index], rc.SourceFile.Content)
	for _, cap := range rc.Match.Captures {
		captureName := rc.CaptureNames[cap.Index]
		if cap.Node.IsMissing() || cap.Node.IsError() {
			continue
		}
		content := cap.Node.Utf8Text(rc.SourceFile.Content)
		switch types.ToElementType(captureName) {
		case types.ElementTypeCallName:
			element.BaseElement.Name = content
		case types.ElementTypeCallArguments:
			params := parseParameters(content)
			// 只有数量可以用于匹配
			for _, param := range params {
				element.Parameters = append(element.Parameters, &param)
			}
		case types.ElementTypeCallOwner:
			element.Owner = content
		}
	}
	element.BaseElement.Scope = types.ScopeFunction
	return []Element{element}, nil
}

func parseParameters(content string) []Parameter {
	// 参数格式 (int a, Function<String, Integer> func, Runnable r, List<String[]> arrs,int... nums)
	var params []Parameter
	// 容错处理：如果没有左括号，尝试从开头解析
	start := 0
	if len(content) > 0 && content[0] == '(' {
		// 去掉第一个(
		start = 1
	}
	level := 0
	for i, c := range content {
		switch c {
		case '<':
			level++
		case '>':
			level--
		case ',':
			if level == 0 {
				paramStr := strings.TrimSpace(content[start:i])
				if paramStr != "" {
					params = append(params, parseSingleParameter(paramStr))
				}
				start = i + 1
			}
		}
	}
	// 最后一个参数
	if start < len(content) {
		// 预防没有右括号的情况
		end := len(content)
		if end > 0 && content[end-1] == ')' {
			end = end - 1
		}
		paramStr := strings.TrimSpace(content[start:end])
		if paramStr != "" {
			params = append(params, parseSingleParameter(paramStr))
		}
	}
	return params
}

func parseSingleParameter(paramStr string) Parameter {
	// 过滤修饰符在CleanParam里面做完了
	// 拆分类型和名称
	// 解析如下三种参数格式，提取类型和名称，忽略默认值：
	// 1. std::vector<int>& scores      => 类型: std::vector<int>&, 名称: scores
	// 2. int rank = 1                  => 类型: int, 名称: rank
	// 3. double bonus = 0.0            => 类型: double, 名称: bonus
	// 实现：先去除=及其后面的默认值，再用空格分割，最后一个为名称，其余为类型
	if idx := strings.Index(paramStr, "="); idx != -1 {
		paramStr = paramStr[:idx]
	}
	paramStr = strings.TrimSpace(paramStr)
	parts := strings.Fields(paramStr)
	if len(parts) < 2 {
		// 省略了type的情况
		return Parameter{Name: paramStr, Type: []string{types.PrimitiveType}}
	}
	name := parts[len(parts)-1]
	typ := strings.Join(parts[:len(parts)-1], " ")
	return Parameter{
		Name: strings.TrimSpace(name),
		Type: []string{strings.TrimSpace(typ)},
	}
}

func parseLocalVariableValue(node *sitter.Node, content []byte) string {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		switch types.ToNodeKind(child.Kind()) {
		case types.NodeKindTypeIdentifier:
			// new Person("Alice", 30);
			// new HashMap<>()
			return child.Utf8Text(content)

		case types.NodeKindScopedTypeIdentifier:
			// new com.example.test.Person("Alice", 30);
			return child.Utf8Text(content)
		case types.NodeKindQualifiedIdentifier:
			// in = Outer::Inner{8}
			return child.ChildByFieldName("name").Utf8Text(content)
		}

	}
	return types.EmptyString
}

// getScopeFromModifiers 根据Java访问修饰符确定作用域
// 参数：
//   - modifiers: 包含修饰符的字符串，可能包含多个修饰符如 "public static final"
//
// 返回：
//   - 对应的作用域类型
func getScopeFromModifiers(modifiers string, kind types.NodeKind) types.Scope {
	// 按优先级检查修饰符（private > protected > public > default）
	if strings.Contains(modifiers, string(types.ModifierPrivate)) {
		// private修饰符：类作用域，仅在当前类内部可见
		return types.ScopeClass
	}

	if strings.Contains(modifiers, string(types.ModifierProtected)) {
		// protected修饰符：包作用域，在包内和子类中可见
		return types.ScopePackage
	}

	if strings.Contains(modifiers, string(types.ModifierPublic)) {
		// public修饰符：项目作用域，在整个项目中可见
		return types.ScopeProject
	}
	// 默认访问修饰符（无修饰符）：
	// 类：包作用域，仅在包内可见
	// 接口：项目作用域，在整个项目中可见
	// 枚举：包作用域，仅在包内可见
	// 都不匹配返回包作用域
	switch kind {
	case types.NodeKindClassDeclaration:
		return types.ScopePackage
	case types.NodeKindInterfaceDeclaration:
		return types.ScopeProject
	case types.NodeKindEnumDeclaration:
		return types.ScopePackage

		//c++的访问修饰符
	case types.NodeKindStructSpecifier:
		return types.ScopeProject
	case types.NodeKindClassSpecifier:
		return types.ScopeClass
	default:
		return types.ScopePackage
	}
}

// parseLocalVariableType 解析局部变量类型
// 参数：
//   - node: 节点
//   - content: 内容
//
// 返回：
//   - 类型名切片
func parseLocalVariableType(node *sitter.Node, content []byte) []string {

	// 从顶部判断是不是基础数据类型
	parentKind := node.Kind()
	switch types.ToNodeKind(parentKind) {
	// ---------------------------java返回值类型解析--------------------------------
	case types.NodeKindIntegralType:
		// 接收 int long short byte char
		return []string{types.PrimitiveType}
	case types.NodeKindFloatingPointType:
		// 接收 float double
		return []string{types.PrimitiveType}
	case types.NodeKindBooleanType:
		// 接收 boolean
		return []string{types.PrimitiveType}
	case types.NodeKindTypeIdentifier:
		// 接收类名
		return []string{node.Utf8Text(content)}
	case types.NodeKindArrayType:
		// 解析数组类型
		// type: array_type [26, 4] - [26, 10]
		//   element: integral_type [26, 4] - [26, 8]
		//   dimensions: dimensions [26, 8] - [26, 10]
		// 递归处理element
		if node.ChildCount() > 0 {
			return parseLocalVariableType(node.Child(0), content)
		}
		return []string{types.PrimitiveType}
	case types.NodeKindGenericType:
		// 解析泛型类型
		// Map<String, Person>
		// type: generic_type [18, 4] - [18, 24]
		//   type_identifier [18, 4] - [18, 7] Map
		//   type_arguments [18, 7] - [18, 24]
		//     type_identifier [18, 8] - [18, 14] String
		//     type_identifier [18, 16] - [18, 23] Person
		//
		return parseGenericType(node, content)

	// ---------------------------c/cpp返回值类型解析--------------------------------
	case types.NodeKindPrimitiveType:
		// c/cpp的基础类型都由这个接收
		return []string{types.PrimitiveType}

	default:
		// 可能有漏的情况，先返回primitive_type
		return []string{types.PrimitiveType}
	}

}

// parseGenericType 递归解析泛型类型节点，返回如 "Map<String, Person>" 这样的字符串
//
// 支持的情况示例：
// 1. 单层泛型：List<String>、Map<Integer, String>
// 2. 嵌套泛型：Map<String, List<Person>>、List<Map<String, Integer>>
// 3. 通配符泛型：List<?>、List<? extends Person>、List<? super Number>
// 4. 复杂嵌套与通配符：Map<String, List<? extends Person>>
//
// 例如：
//
//	Map<String, Person>         -> ["Map", "String", "Person"]
//	List<Person>                -> ["List", "Person"]
//	Map<String, List<Person>>   -> ["Map", "String", "List", "Person"]
//	List<? extends Person>      -> ["List", "Person"]
//	List<?>                     -> ["List"]

// parseGenericType 递归提取泛型类型节点中出现的所有类型名（不去重，顺序不保证）
// 例如：Map<String, List<Person>> -> ["Map", "String", "List", "Person"]
func parseGenericType(node *sitter.Node, content []byte) []string {
	// List<? extends Number> numbers = new ArrayList<>();
	// type: generic_type [19, 4] - [19, 26]
	//       type_identifier [19, 4] - [19, 8]
	//       type_arguments [19, 8] - [19, 26]
	//         wildcard [19, 9] - [19, 25]
	//           type_identifier [19, 19] - [19, 25]
	var result []string
	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		for i := uint(0); i < n.ChildCount(); i++ {
			child := n.Child(i)
			kind := types.ToNodeKind(child.Kind())
			switch kind {
			case types.NodeKindTypeIdentifier:
				typeName := child.Utf8Text(content)
				result = append(result, typeName)
			case types.NodeKindScopedTypeIdentifier:
				// 只会解析出来一个superType
				result = append(result, child.Utf8Text(content))
			case types.NodeKindGenericType, types.NodeKindTypeArguments, types.NodeKindWildcard:
				walk(child)
			}
		}
	}
	walk(node)

	return result
}

// findMethodOwner 通过遍历语法树找到方法的拥有者（类或接口），返回拥有者的节点
func findMethodOwner(node *sitter.Node) *sitter.Node {
	if node == nil {
		return nil
	}
	// 向上遍历父节点，查找类或接口声明
	current := node.Parent()
	for current != nil {
		kind := current.Kind()
		switch types.ToNodeKind(kind) {
		// 找到类、接口、方法声明，返回当前节点（支持java、c、cpp）
		case types.NodeKindClassDeclaration, types.NodeKindClassSpecifier, types.NodeKindStructSpecifier:
			return current
		// 找到接口声明，返回当前节点
		case types.NodeKindInterfaceDeclaration:
			return current
		// 找到枚举声明，返回当前节点
		case types.NodeKindEnumDeclaration:
			return current
		case types.NodeKindVariableDeclarator:
			return current
		}

		current = current.Parent()
	}
	return nil
}

// extractNodeName 从类/接口/枚举声明节点中提取名称
func extractNodeName(node *sitter.Node, content []byte) string {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		kind := types.ToNodeKind(child.Kind())
		if kind == types.NodeKindIdentifier || kind == types.NodeKindTypeIdentifier {
			return child.Utf8Text(content)
		}
	}
	return types.EmptyString
}
func getElementModifier(content string) string {
	if strings.Contains(content, types.ModifierPublic) {
		return types.ModifierPublic
	}
	if strings.Contains(content, types.ModifierProtected) {
		return types.ModifierProtected
	}
	if strings.Contains(content, types.ModifierPrivate) {
		return types.ModifierPrivate
	}
	return types.EmptyString
}

// 提取参数字符串中所有类型名，去除泛型、数组、可变参数等修饰，按出现顺序输出
func ExtractParameters(paramStr string) []Parameter {
	// 解析参数，只会返回 List<String[]> arrs，不会将类型拆开
	params := parseParameters(paramStr)
	var results []Parameter

	for _, param := range params {
		typs := extractTypeNames(param.Type[0])
		results = append(results, Parameter{
			Name: param.Name,
			Type: typs,
		})
	}
	return results
}

func getFilteredTypes(typ string) []string {
	typ = CleanParam(typ)
	typeNames := extractTypeNames(typ)
	return types.FilterCustomTypes(typeNames)
}

// 提取参数字符串并过滤类型
func getFilteredParameters(paramStr string) []Parameter {
	// 删除一些影响解析的修饰符
	paramStr = CleanParam(paramStr)
	// 提取参数
	params := ExtractParameters(paramStr)
	for i, param := range params {
		params[i].Type = types.FilterCustomTypes(param.Type)
	}
	return params
}

// 递归提取类型字符串中的所有类型名
func extractTypeNames(typeStr string) []string {
	var res []string
	typeStr = strings.TrimSpace(typeStr)
	// 去除可变参数
	typeStr = strings.TrimSuffix(typeStr, "...")
	// 去除数组
	for strings.HasSuffix(typeStr, "[]") {
		typeStr = strings.TrimSuffix(typeStr, "[]")
		typeStr = strings.TrimSpace(typeStr)
	}
	// 处理泛型
	lt := strings.Index(typeStr, "<")
	if lt != -1 {
		base := strings.TrimSpace(typeStr[:lt])
		if base != "" {
			res = append(res, base)
		}
		gt := strings.LastIndex(typeStr, ">")
		if gt != -1 && gt > lt {
			inner := typeStr[lt+1 : gt]
			// 递归处理泛型参数，支持多层嵌套
			innerTypes := splitGenericTypes(inner)
			for _, t := range innerTypes {
				res = append(res, extractTypeNames(t)...)
			}
		}
	} else {
		// 没有泛型，直接加
		if typeStr != "" {
			res = append(res, typeStr)
		}
	}
	return res
}

// 拆分泛型参数，支持嵌套
func splitGenericTypes(s string) []string {
	var res []string
	level := 0
	start := 0
	for i, c := range s {
		switch c {
		case '<':
			level++
		case '>':
			level--
		case ',':
			if level == 0 {
				part := strings.TrimSpace(s[start:i])
				if part != "" {
					res = append(res, part)
				}
				start = i + 1
			}
		}
	}
	// 最后一个
	if start < len(s) {
		part := strings.TrimSpace(s[start:])
		if part != "" {
			res = append(res, part)
		}
	}
	return res
}
