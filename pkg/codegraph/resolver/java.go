package resolver

import (
	"codebase-indexer/pkg/codegraph/types"
	"codebase-indexer/pkg/codegraph/utils"
	"context"
	"fmt"
	"path/filepath"
	"regexp"
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

	element.FilePaths = []string{}
	var importName string
	// 获取import name
	for _, cap := range rc.Match.Captures {
		captureName := rc.CaptureNames[cap.Index]
		if cap.Node.IsMissing() || cap.Node.IsError() {
			return nil, fmt.Errorf("import is missing or error")
		}
		content := cap.Node.Utf8Text(rc.SourceFile.Content)
		switch types.ToElementType(captureName) {
		case types.ElementTypeImportName:
			importName = content
		}
	}
	// 处理类导入
	classPath := strings.ReplaceAll(importName, ".", "/") + ".java"
	pj := rc.ProjectInfo
	fullPath := utils.ToUnixPath(filepath.Join(pj.GetSourceRoot(), classPath))

	elements := []Element{element}
	element.BaseElement.Content = rc.SourceFile.Content
	element.BaseElement.Range = []int32{
		int32(rc.Match.Captures[0].Node.StartPosition().Row),
		int32(rc.Match.Captures[0].Node.StartPosition().Column),
		int32(rc.Match.Captures[0].Node.EndPosition().Row),
		int32(rc.Match.Captures[0].Node.EndPosition().Column),
	}
	element.BaseElement.Scope = types.ScopePackage
	element.BaseElement.Type = types.ElementTypeImport
	element.BaseElement.Name = importName

	if pj.IsEmpty() {
		// TODO logger
		fmt.Println("not support project file list, use default resolve")
		element.FilePaths = []string{fullPath}
		return elements, nil
	}

	// 处理静态导入，有则会移除，没有就不会动
	importName = strings.TrimPrefix(importName, "static ")
	// 处理包导入
	if strings.HasSuffix(importName, ".*") {
		pkgPath := strings.ReplaceAll(strings.TrimSuffix(importName, ".*"), ".", "/")
		fullPkgPath := utils.ToUnixPath(filepath.Join(pj.GetSourceRoot(), pkgPath))
		files := pj.FindFilesInDirIndex(fullPkgPath, ".java")
		element.FilePaths = files
		if len(element.FilePaths) == 0 {
			return nil, fmt.Errorf("cannot find file which package belongs to: %s", importName)
		}
		return elements, nil
	}

	element.FilePaths = pj.FindMatchingFiles(fullPath)
	if len(element.FilePaths) == 0 {
		return nil, fmt.Errorf("cannot find file which import belongs to: %s", importName)
	}

	return elements, nil
}

func (j *JavaResolver) resolvePackage(ctx context.Context, element *Package, rc *ResolveContext) ([]Element, error) {
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
	element.BaseElement.Type = types.ElementTypePackage
	element.BaseElement.Content = rc.SourceFile.Content
	element.BaseElement.Range = []int32{
		int32(rc.Match.Captures[0].Node.StartPosition().Row),
		int32(rc.Match.Captures[0].Node.StartPosition().Column),
		int32(rc.Match.Captures[0].Node.EndPosition().Row),
		int32(rc.Match.Captures[0].Node.EndPosition().Column),
	}
	// package不需要额外处理，直接返回
	return []Element{element}, nil
}

func (j *JavaResolver) resolveFunction(ctx context.Context, element *Function, rc *ResolveContext) ([]Element, error) {
	// TODO java中不存在单独的函数，暂时不实现
	return nil, fmt.Errorf("not implemented")
}

func (j *JavaResolver) resolveMethod(ctx context.Context, element *Method, rc *ResolveContext) ([]Element, error) {
	// TODO 补充方法解析
	return nil, fmt.Errorf("not implemented")
}

func (j *JavaResolver) resolveClass(ctx context.Context, element *Class, rc *ResolveContext) ([]Element, error) {
	rootCap := rc.Match.Captures[0]
	updateRootElement(element, &rootCap, rc.CaptureNames[rootCap.Index], rc.SourceFile.Content)
	var scope string
	for _, cap := range rc.Match.Captures {
		captureName := rc.CaptureNames[cap.Index]
		if cap.Node.IsMissing() || cap.Node.IsError() {
			continue
		}
		content := cap.Node.Utf8Text(rc.SourceFile.Content)
		switch types.ElementType(captureName) {
		case types.ElementTypeClassName:
			// 解析类名
			element.BaseElement.Name = content
		case types.ElementTypeClassExtends:
			// 解析父类名，并添加到SuperClasses切片
			element.SuperClasses = append(element.SuperClasses, content)
		case types.ElementTypeClassImplements:
			// 解析实现的接口名，并添加到SuperInterfaces切片，自动去除空格
			content = strings.TrimSpace(content)
			ifaces := strings.Split(content, ",")
			for _, iface := range ifaces {
				iface = strings.TrimSpace(iface)
				if iface != "" {
					element.SuperInterfaces = append(element.SuperInterfaces, iface)
				}
			}
		case types.ElementTypeClassModifiers:
			// 解析类的访问修饰符，并设置作用域
			// public、private、protected 或无修饰符
			// 无修饰符时，不走这个路径
			scope = content
		}
	}
	element.BaseElement.Scope = getScopeFromModifiers(scope)
	cls := parseClassNode(&rootCap.Node, rc.SourceFile.Content, element.BaseElement.Name)
	for _, field := range cls.Fields {
		if field.Modifier == types.EmptyString {
			field.Modifier = types.PackagePrivate
		}
	}
	element.Fields = cls.Fields
	for _, method := range cls.Methods {
		if method.Declaration.Modifier == types.EmptyString {
			method.Declaration.Modifier = types.PackagePrivate
		}
	}
	element.Methods = cls.Methods
	return []Element{element}, nil
}

func (j *JavaResolver) resolveVariable(ctx context.Context, element *Variable, rc *ResolveContext) ([]Element, error) {

	var refs = []*Reference{}
	rootCap := rc.Match.Captures[0]
	updateRootElement(element, &rootCap, rc.CaptureNames[rootCap.Index], rc.SourceFile.Content)
	for _, cap := range rc.Match.Captures {
		captureName := rc.CaptureNames[cap.Index]
		if cap.Node.IsMissing() || cap.Node.IsError() {
			continue
		}
		content := cap.Node.Utf8Text(rc.SourceFile.Content)
		switch types.ToElementType(captureName) {
		case types.ElementTypeLocalVariableName:
			element.BaseElement.Name = content
		case types.ElementTypeLocalVariableType:
			// 左侧的类型声明
			//1. 标准类型 设置为primitive_type
			//2. 用户自定义或其他包里面的类型 设置为对应的类型
			element.VariableType = parseLocalVariableType(&cap.Node, rc.SourceFile.Content)
			// 筛选出用户自定义的类型
			element.VariableType = types.FilterCustomTypes(element.VariableType)
		case types.ElementTypeLocalVariableValue:
			// 有可能是字面量，也有可能是类的创建，和方法调用
			// 字面量不处理，方法调用由resolveCall处理，只处理类的创建
			val := parseLocalVariableValue(&cap.Node, rc.SourceFile.Content)
			ref := &Reference{
				BaseElement: &BaseElement{
					Name:    val,
					Type:    types.ElementTypeReference,
					Content: rc.SourceFile.Content,
					Range: []int32{
						int32(cap.Node.StartPosition().Row),
						int32(cap.Node.StartPosition().Column),
						int32(cap.Node.EndPosition().Row),
						int32(cap.Node.EndPosition().Column),
					},
				},
				Owner: val, // 待定
			}
			refs = append(refs, ref)
		}
	}
	element.BaseElement.Scope = types.ScopeFunction
	elems := []Element{element}
	for _, ref := range refs {
		// 触发自动转换，将ref转换为Element
		elems = append(elems, ref)
	}
	// append不能直接使用进行转换，是go的设计限制
	// elems=append(elems,refs...)
	return elems, nil
}

func (j *JavaResolver) resolveInterface(ctx context.Context, element *Interface, rc *ResolveContext) ([]Element, error) {
	rootCap := rc.Match.Captures[0]
	updateRootElement(element, &rootCap, rc.CaptureNames[rootCap.Index], rc.SourceFile.Content)
	var scope string
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
			scope = content
		case types.ElementTypeInterfaceExtends:
			element.SuperInterfaces = append(element.SuperInterfaces, content)

		}

	}
	element.BaseElement.Scope = getScopeFromModifiers(scope)
	cls := parseClassNode(&rc.Match.Captures[0].Node, rc.SourceFile.Content, element.BaseElement.Name)
	for _, method := range cls.Methods {
		if method.Declaration.Modifier == types.EmptyString {
			method.Declaration.Modifier = types.PublicAbstract
		}
		element.Methods = append(element.Methods, &method.Declaration)
	}
	return []Element{element}, nil
}

func (j *JavaResolver) resolveCall(ctx context.Context, element *Call, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	for _, cap := range rc.Match.Captures {
		captureName := rc.CaptureNames[cap.Index]
		if cap.Node.IsMissing() || cap.Node.IsError() {
			return nil, fmt.Errorf("call is missing or error")
		}
		content := cap.Node.Utf8Text(rc.SourceFile.Content)
		switch types.ToElementType(captureName) {
		case types.ElementTypeCallName:
			element.BaseElement.Name = content
		// case types.ElementTypeCallArguments :
		// 	element.Parameters = parseParameters(content)
		case types.ElementTypeCallOwner:
			element.Owner = content
		}
	}
	element.BaseElement.Type = types.ElementTypeMethodCall
	element.BaseElement.Content = rc.SourceFile.Content
	element.BaseElement.Range = []int32{
		int32(rc.Match.Captures[0].Node.StartPosition().Row),
		int32(rc.Match.Captures[0].Node.StartPosition().Column),
		int32(rc.Match.Captures[0].Node.EndPosition().Row),
		int32(rc.Match.Captures[0].Node.EndPosition().Column),
	}
	return []Element{element}, nil
}

func parseClassNode(node *sitter.Node, content []byte, className string) *Class {
	class := &Class{}
	node = node.Child(node.ChildCount() - 1)
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		kind := types.ToNodeKind(child.Kind())
		switch kind {
		case types.NodeKindField:
			// 存在 int a,b,c; 的情况，需要解析多个field
			fields := parseFieldNode(child, content)
			for _, field := range fields {
				if field.Modifier == types.EmptyString {
					field.Modifier = types.PackagePrivate
				}
				class.Fields = append(class.Fields, field)
			}
		case types.NodeKindMethod, types.NodeKindConstructor:
			method := parseMethodNode(child, content)
			method.Owner = className
			class.Methods = append(class.Methods, method)
		}
	}
	return class
}

func parseFieldNode(node *sitter.Node, content []byte) []*Field {
	fields := []*Field{}
	var typ, modifier string
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		kind := types.ToNodeKind(child.Kind())
		txt := child.Utf8Text(content)
		switch kind {
		case types.NodeKindVariableDeclarator:
			// variable_declarator 下面找 identifier
			for j := uint(0); j < child.ChildCount(); j++ {
				sub := child.Child(j)
				if sub.Kind() == "identifier" {
					name := sub.Utf8Text(content)
					fields = append(fields, &Field{
						Name: name,
						Modifier: modifier,
						Type: typ,
					})
				}
			}
		case types.NodeKindModifier:
			modifier = txt
		default:
			if types.IsTypeNode(kind) {
				typ = txt
			}
		}
	}
	return fields
}

func parseMethodNode(node *sitter.Node, content []byte) *Method {
	var method = &Method{}
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		content := child.Utf8Text(content)
		kind := types.ToNodeKind(child.Kind())
		switch kind {
		case types.NodeKindIdentifier:
			method.Declaration.Name = content
		case types.NodeKindModifier:
			// @Override\n  @Resource \n  public -> @Override @Resource public
			lines := strings.Split(content, "\n")
			joined := strings.Join(lines, " ")                       // 把多行拼成一行（以空格连接）
			cleaned := strings.Fields(joined)                        // 按空白字符切分成词
			method.Declaration.Modifier = strings.Join(cleaned, " ") // 再重新拼接成整洁字符串
		case types.NodeKindFormalParameters:
			method.Declaration.Parameters = parseParameters(content)
		default:
			if types.IsTypeNode(kind) {
				method.Declaration.ReturnType = content
			}
		}
	}
	return method
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
	// 去掉注解
	paramStr = regexp.MustCompile(`@\w+\s+`).ReplaceAllString(paramStr, "")
	// 拆分类型和名称
	parts := strings.Fields(paramStr)
	// if len(parts) < 2 {
	// 	// 可能是省略了参数名，可特殊处理 或 报错 日志？
	// 	return &Parameter{Name: paramStr}
	// }
	name := parts[len(parts)-1]
	typ := strings.Join(parts[:len(parts)-1], " ")

	return Parameter{
		Name: strings.TrimSpace(name),
		Type: strings.TrimSpace(typ),
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
		}

	}
	return ""
}

// getScopeFromModifiers 根据Java访问修饰符确定作用域
// 参数：
//   - modifiers: 包含修饰符的字符串，可能包含多个修饰符如 "public static final"
//
// 返回：
//   - 对应的作用域类型
func getScopeFromModifiers(modifiers string) types.Scope {
	// 移除多余的空白字符并转换为小写以便比较
	modifiers = strings.ToLower(strings.TrimSpace(modifiers))

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
	// 默认访问修饰符（无修饰符）：包作用域，仅在包内可见
	return types.ScopePackage
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

// parseGenericType 递归提取泛型类型节点中出现的所有类型名（去重，顺序不保证）
// 例如：Map<String, List<Person>> -> ["Map", "String", "List", "Person"]
func parseGenericType(node *sitter.Node, content []byte) []string {
	// 用于去重
	result := make(map[string]struct{})
	// List<? extends Number> numbers = new ArrayList<>();
	// type: generic_type [19, 4] - [19, 26]
	//       type_identifier [19, 4] - [19, 8]
	//       type_arguments [19, 8] - [19, 26]
	//         wildcard [19, 9] - [19, 25]
	//           type_identifier [19, 19] - [19, 25]
	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		for i := uint(0); i < n.ChildCount(); i++ {
			child := n.Child(i)
			kind := types.ToNodeKind(child.Kind())
			switch kind {
			case types.NodeKindTypeIdentifier:
				typeName := child.Utf8Text(content)
				result[typeName] = struct{}{}
			case types.NodeKindGenericType, types.NodeKindTypeArguments, types.NodeKindWildcard:
				walk(child)
			}
		}
	}
	walk(node)

	// 转为切片返回
	var typeNames []string
	for t := range result {
		typeNames = append(typeNames, t)
	}
	return typeNames
}
