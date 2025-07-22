package resolver

import (
	"codebase-indexer/pkg/codegraph/types"
	"context"
	"fmt"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type JavaScriptResolver struct {
}

var _ ElementResolver = &JavaScriptResolver{}

func (js *JavaScriptResolver) Resolve(ctx context.Context, element Element, rc *ResolveContext) ([]Element, error) {
	// 特殊处理：require调用应该被解析为import，而不是call或variable
	if rc.Match != nil && len(rc.Match.Captures) > 0 {
		rootCapture := rc.Match.Captures[0]
		// 检查是否是call_expression且函数是require
		if rootCapture.Node.Kind() == string(types.NodeKindCallExpression) {
			funcNode := rootCapture.Node.ChildByFieldName("function")
			if funcNode != nil && funcNode.Kind() == string(types.NodeKindIdentifier) &&
				string(funcNode.Utf8Text(rc.SourceFile.Content)) == "require" {
				// 如果是variable元素，跳过处理（会在call中处理为import）
				if _, isVar := element.(*Variable); isVar {
					return []Element{}, nil
				}

				// 如果是call元素，转换为import处理
				if _, isCall := element.(*Call); isCall {
					importElement := &Import{
						BaseElement: &BaseElement{
							Type:  types.ElementTypeImport,
							Scope: types.ScopeFile,
						},
					}

					// 获取模块路径
					argsNode := rootCapture.Node.ChildByFieldName("arguments")
					if argsNode != nil {
						for i := uint(0); i < argsNode.ChildCount(); i++ {
							argNode := argsNode.Child(i)
							if argNode != nil && argNode.Kind() == "string" {
								importElement.Source = strings.Trim(string(argNode.Utf8Text(rc.SourceFile.Content)), "'\"")
								break
							}
						}
					}

					// 查找变量名
					var parent = rootCapture.Node.Parent()
					if parent != nil && parent.Kind() == string(types.NodeKindVariableDeclarator) {
						nameNode := parent.ChildByFieldName("name")
						if nameNode != nil {
							importElement.Name = string(nameNode.Utf8Text(rc.SourceFile.Content))
							importElement.BaseElement.Name = importElement.Name
							importElement.Content = []byte(importElement.Name)
						}
					}

					// 设置范围
					importElement.SetRange([]int32{
						int32(rootCapture.Node.StartPosition().Row),
						int32(rootCapture.Node.StartPosition().Column),
						int32(rootCapture.Node.EndPosition().Row),
						int32(rootCapture.Node.EndPosition().Column),
					})

					return []Element{importElement}, nil
				}
			}
		}
	}

	// 常规解析流程
	return resolve(ctx, js, element, rc)
}

func (js *JavaScriptResolver) resolveImport(ctx context.Context, element *Import, rc *ResolveContext) ([]Element, error) {
	pj := rc.ProjectInfo

	elements := []Element{element}
	if pj.IsEmpty() {
		fmt.Println("not support project file list, use default resolve")
		return elements, nil
	}
	for _, capture := range rc.Match.Captures {
		if capture.Node.IsMissing() || capture.Node.IsError() {
			continue
		}
		nodeCaptureName := rc.CaptureNames[capture.Index]
		content := capture.Node.Utf8Text(rc.SourceFile.Content)
		updateRootElement(element, &capture, nodeCaptureName, rc.SourceFile.Content)
		switch types.ToElementType(nodeCaptureName) {
		case types.ElementTypeImport:
			element.Type = types.ElementTypeImport
		case types.ElementTypeImportName:
			element.Name = content
			element.Content = []byte(content)
		case types.ElementTypeImportAlias:
			element.Alias = content
		case types.ElementTypeImportSource:
			element.Source = strings.Trim(content, "'")
		}
	}

	return elements, nil
}

func (js *JavaScriptResolver) resolvePackage(ctx context.Context, element *Package, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	//此语言不支持
	panic("not support")
}

func (js *JavaScriptResolver) resolveFunction(ctx context.Context, element *Function, rc *ResolveContext) ([]Element, error) {
	elements := []Element{element}
	for _, capture := range rc.Match.Captures {
		if capture.Node.IsMissing() || capture.Node.IsError() {
			continue
		}
		nodeCaptureName := rc.CaptureNames[capture.Index]
		content := capture.Node.Utf8Text(rc.SourceFile.Content)

		// 调试类型映射
		mappedType := types.ToElementType(nodeCaptureName)
		fmt.Printf("处理捕获节点: %s, 映射类型: %v, 内容: %q\n", nodeCaptureName, mappedType, content)

		updateRootElement(element, &capture, nodeCaptureName, rc.SourceFile.Content)

		switch types.ToElementType(nodeCaptureName) {
		case types.ElementTypeFunction:
			element.Type = types.ElementTypeFunction
			element.Scope = types.ScopeFile
			// 检查是否包含修饰符
			element.Declaration.Modifier = extractModifiers(content, "function")
		case types.ElementTypeFunctionName:
			element.BaseElement.Name = content
			element.Declaration.Name = content
			element.Content = []byte(content)
		case types.ElementTypeFunctionParameters:
			parseJavaScriptParameters(element, capture.Node, rc.SourceFile.Content)
		}
	}
	return elements, nil
}

// parseJavaScriptParameters 解析JavaScript函数参数
func parseJavaScriptParameters(element *Function, paramsNode sitter.Node, content []byte) {
	element.Parameters = make([]Parameter, 0)
	for i := uint(0); i < paramsNode.ChildCount(); i++ {
		child := paramsNode.Child(i)
		if child != nil && child.Kind() == types.Identifier {
			paramNode := child
			paramName := paramNode.Utf8Text(content)
			element.Parameters = append(element.Parameters, Parameter{
				Name: paramName,
				Type: nil,
			})
		}
	}
}

// extractModifiers 从函数或方法声明中提取修饰符
// elementType: 元素类型，如"function"或"method"
func extractModifiers(content string, elementType string) string {
	// JavaScript中函数和方法的可能修饰符
	modifiers := []string{"async", "static", "get", "set", "*"}
	result := ""

	// 按空格分割函数声明
	for _, mod := range modifiers {
		if containsModifier(content, mod) {
			if result != "" {
				result += " "
			}
			result += mod
		}
	}

	return result
}

// containsModifier 判断字符串是否包含指定的修饰符
func containsModifier(content string, modifier string) bool {
	// 生成器函数特殊处理
	if modifier == "*" {
		// 检查是否包含 "function*" 或 "* "
		return strings.Contains(content, "function*") || strings.Contains(content, "* ")
	}

	// 其他修饰符需要确保是单独的单词
	words := strings.Fields(content)
	for _, word := range words {
		if word == modifier {
			return true
		}
	}
	return false
}

func (js *JavaScriptResolver) resolveMethod(ctx context.Context, element *Method, rc *ResolveContext) ([]Element, error) {
	rootCap := rc.Match.Captures[0]
	updateRootElement(element, &rootCap, rc.CaptureNames[rootCap.Index], rc.SourceFile.Content)
	elements := []Element{element}
	for _, capture := range rc.Match.Captures {
		if capture.Node.IsMissing() || capture.Node.IsError() {
			continue
		}
		nodeCaptureName := rc.CaptureNames[capture.Index]
		content := capture.Node.Utf8Text(rc.SourceFile.Content)
		updateRootElement(element, &capture, nodeCaptureName, rc.SourceFile.Content)

		switch types.ToElementType(nodeCaptureName) {
		case types.ElementTypeMethod:
			element.Type = types.ElementTypeMethod
			// 检查是否包含修饰符
			element.Declaration.Modifier = extractModifiers(content, "method")
		case types.ElementTypeMethodName:
			element.BaseElement.Name = content
			element.Declaration.Name = content
			element.Content = []byte(content)
		case types.ElementTypeMethodParameters:
			parseJavaScriptMethodParameters(element, capture.Node, rc.SourceFile.Content)
		}
	}
	// 获取方法所属的类，原方法在java.go中
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
	return elements, nil
}

// parseJavaScriptMethodParameters 解析JavaScript方法参数
func parseJavaScriptMethodParameters(element *Method, paramsNode sitter.Node, content []byte) {
	element.Parameters = make([]Parameter, 0)
	for i := uint(0); i < paramsNode.ChildCount(); i++ {
		child := paramsNode.Child(i)
		if child != nil && child.Kind() == types.Identifier {
			paramNode := child
			paramName := paramNode.Utf8Text(content)
			element.Parameters = append(element.Parameters, Parameter{
				Name: paramName,
				Type: nil, // JavaScript作为动态语言，参数类型通常无法从语法中直接获取
			})
		}
	}
}

func (js *JavaScriptResolver) resolveClass(ctx context.Context, element *Class, rc *ResolveContext) ([]Element, error) {
	elements := []Element{element}
	rootCapure := rc.Match.Captures[0]
	captureName := rc.CaptureNames[rootCapure.Index]
	updateRootElement(element, &rootCapure, captureName, rc.SourceFile.Content)
	// 初始化字段和方法数组
	element.Fields = []*Field{}
	element.Methods = []*Method{}
	element.SuperClasses = []string{}

	for _, capture := range rc.Match.Captures {
		if capture.Node.IsMissing() || capture.Node.IsError() {
			continue
		}
		nodeCaptureName := rc.CaptureNames[capture.Index]
		content := capture.Node.Utf8Text(rc.SourceFile.Content)
		updateRootElement(element, &capture, nodeCaptureName, rc.SourceFile.Content)

		switch types.ToElementType(nodeCaptureName) {
		case types.ElementTypeClass:
			element.Type = types.ElementTypeClass
			element.Scope = types.ScopeFile
			// 解析类体中的所有成员
		case types.ElementTypeClassName:
			element.BaseElement.Name = content
			element.Content = []byte(content)
		case types.ElementTypeClassExtends:
			//获取继承的类名
			Node := capture.Node
			content = Node.Child(1).Utf8Text(rc.SourceFile.Content)
			// 处理类继承关系
			element.SuperClasses = append(element.SuperClasses, content)
		}
	}
	parseJavaScriptClassBody(&rootCapure.Node, rc.SourceFile.Content, element)
	return elements, nil
}

// parseJavaScriptClassBody 解析JavaScript类体，提取字段和方法
func parseJavaScriptClassBody(node *sitter.Node, content []byte, class *Class) {
	// 查找class_body节点
	var classBodyNode *sitter.Node

	// 类声明节点的最后一个子节点通常是类体
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child != nil && child.Kind() == string(types.NodeKindClassBody) {
			classBodyNode = child
			break
		}
	}

	if classBodyNode == nil {
		return
	}

	// 遍历类体中的所有成员
	for i := uint(0); i < classBodyNode.ChildCount(); i++ {
		memberNode := classBodyNode.Child(i)
		if memberNode == nil {
			continue
		}

		kind := memberNode.Kind()
		switch types.ToNodeKind(kind) {
		case types.NodeKindMethodDefinition:
			// 处理方法
			method := parseJavaScriptMethodNode(memberNode, content, class.BaseElement.Name)
			if method != nil {
				class.Methods = append(class.Methods, method)
			}
		case types.NodeKindFieldDefinition:
			// 处理字段
			field := parseJavaScriptFieldNode(memberNode, content)
			if field != nil {
				class.Fields = append(class.Fields, field)
			}
		}
	}
}

// parseJavaScriptMethodNode 解析JavaScript方法节点
func parseJavaScriptMethodNode(node *sitter.Node, content []byte, className string) *Method {
	method := &Method{}
	method.Owner = className

	// 设置默认作用域和修饰符
	method.BaseElement = &BaseElement{
		Scope: types.ScopeFile,
	}
	method.Declaration.Modifier = types.ModifierPublic // JavaScript默认为public

	// 查找方法名
	nameNode := node.ChildByFieldName("name")
	if nameNode != nil {
		methodName := nameNode.Utf8Text(content)
		method.BaseElement.Name = methodName
		method.Declaration.Name = methodName
	}

	// 查找方法参数
	paramsNode := node.ChildByFieldName("parameters")
	if paramsNode != nil {
		method.Parameters = make([]Parameter, 0)
		for j := uint(0); j < paramsNode.ChildCount(); j++ {
			paramChild := paramsNode.Child(j)
			if paramChild != nil && paramChild.Kind() == types.Identifier {
				paramName := paramChild.Utf8Text(content)
				method.Parameters = append(method.Parameters, Parameter{
					Name: paramName,
					Type: nil,
				})
			}
		}
	}

	// 检查是否是构造函数
	if nameNode != nil && nameNode.Utf8Text(content) == "constructor" {
		method.Type = types.ElementTypeConstructor
	} else {
		method.Type = types.ElementTypeMethod
	}

	// 检查修饰符
	if strings.Contains(node.Utf8Text(content), "static") {
		method.Declaration.Modifier = "static " + method.Declaration.Modifier
	}
	if strings.Contains(node.Utf8Text(content), "async") {
		method.Declaration.Modifier = "async " + method.Declaration.Modifier
	}

	return method
}

// parseJavaScriptFieldNode 解析JavaScript字段节点
func parseJavaScriptFieldNode(node *sitter.Node, content []byte) *Field {
	field := &Field{}

	// 查找字段名
	nameNode := node.ChildByFieldName("property")
	if nameNode == nil {

		// 尝试查找property_identifier子节点
		for i := uint(0); i < node.ChildCount(); i++ {
			child := node.Child(i)
			if child != nil && (child.Kind() == string(types.NodeKindPropertyIdentifier) || child.Kind() == string(types.NodeKindPrivatePropertyIdentifier)) {
				nameNode = child
				break
			}
		}

		if nameNode == nil {
			return nil
		}
	}

	fieldName := nameNode.Utf8Text(content)
	isPrivate := strings.HasPrefix(fieldName, "#")

	// 处理私有字段
	if isPrivate {
		fieldName = strings.TrimPrefix(fieldName, "#")
		field.Modifier = types.ModifierPrivate
	} else {
		field.Modifier = types.ModifierPublic
	}

	field.Name = fieldName
	field.Type = "" // JavaScript中字段类型默认为any

	// 检查是否有static修饰符
	// if strings.Contains(node.Utf8Text(content), types.Static) {
	// 	if field.Modifier == types.EmptyString {
	// 		field.Modifier = types.Static
	// 	} else {
	// 		field.Modifier = types.Static + " " + field.Modifier
	// 	}
	// }

	return field
}

func (js *JavaScriptResolver) resolveVariable(ctx context.Context, element *Variable, rc *ResolveContext) ([]Element, error) {
	/*
		JavaScript中变量都没有类型，Variable元素的VariableType为空
	*/
	elements := []Element{}
	// 变量名
	var variableName string

	// 检查节点内容
	rootCapure := rc.Match.Captures[0]
	_ = rc.CaptureNames[rootCapure.Index] // 避免未使用警告

	// 首先获取变量名
	for _, capture := range rc.Match.Captures {
		if capture.Node.IsMissing() || capture.Node.IsError() {
			continue
		}
		nodeCaptureName := rc.CaptureNames[capture.Index]
		content := capture.Node.Utf8Text(rc.SourceFile.Content)

		if types.ToElementType(nodeCaptureName) == types.ElementTypeVariableName {
			variableName = string(content)
			break
		}
	}

	// 使用一个集合跟踪已处理的节点，避免重复处理
	processedNodes := make(map[uint32]bool)

	// 检查是否为箭头函数
	for _, capture := range rc.Match.Captures {
		if capture.Node.IsMissing() || capture.Node.IsError() {
			continue
		}

		// 避免重复处理同一个节点
		if processedNodes[capture.Index] {
			continue
		}
		processedNodes[capture.Index] = true

		nodeCaptureName := rc.CaptureNames[capture.Index]
		captureType := types.ToElementType(nodeCaptureName)
		content := capture.Node.Utf8Text(rc.SourceFile.Content)

		// 检查是否为解构赋值
		if isDestructuringPattern(&capture.Node) {
			return handleDestructuring(&capture.Node, rc.SourceFile.Content)
		}

		if captureType == types.ElementTypeVariable || captureType == types.ElementTypeGlobalVariable {
			// 检查赋值右侧是否为箭头函数
			rightNode := findRightNode(&capture.Node)

			// 检查是否为require函数调用
			if rightNode != nil && isRequireCall(rightNode, rc.SourceFile.Content) {
				// 如果是require调用，跳过变量处理，后续会在resolveCall中处理为import
				return []Element{}, nil
			}

			if rightNode != nil && isArrowFunction(rightNode) {
				// 直接创建函数元素
				functionElement := &Function{
					BaseElement: &BaseElement{
						Name:  variableName,
						Scope: types.ScopeFile,
						Type:  types.ElementTypeFunction,
					},
					Declaration: Declaration{
						Name:       variableName,
						Parameters: []Parameter{},
						ReturnType: []string{},
						Modifier:   types.Arrow,
					},
				}

				// 提取参数
				parseArrowFunctionParameters(functionElement, rightNode, rc.SourceFile.Content)

				// 设置范围
				functionElement.SetRange([]int32{
					int32(rightNode.StartPosition().Row),
					int32(rightNode.StartPosition().Column),
					int32(rightNode.EndPosition().Row),
					int32(rightNode.EndPosition().Column),
				})

				// 直接返回函数元素
				elements = append(elements, functionElement)
				return elements, nil
			} else if rightNode != nil && isClassOrStructReference(rightNode) {
				// 处理引用
				refElement := &Reference{
					BaseElement: &BaseElement{
						Name:  variableName,
						Scope: types.ScopeFile,
						Type:  types.ElementTypeReference,
					},
					Owner: variableName,
				}

				// 存储引用路径
				refPath := extractReferencePath(rightNode, rc.SourceFile.Content)
				refElement.Content = []byte(refPath)

				// 设置范围
				refElement.SetRange([]int32{
					int32(rightNode.StartPosition().Row),
					int32(rightNode.StartPosition().Column),
					int32(rightNode.EndPosition().Row),
					int32(rightNode.EndPosition().Column),
				})

				// 返回引用元素
				elements = append(elements, refElement)
				return elements, nil
			}
		}

		// 如果不是特殊类型，作为普通变量处理
		switch types.ToElementType(nodeCaptureName) {
		case types.ElementTypeVariable:
			element.Type = types.ElementTypeVariable

			// 根据父节点判断变量作用域
			element.Scope = determineVariableScope(&capture.Node)
			element.VariableType = []string{types.PrimitiveType}
		case types.ElementTypeVariableName:
			element.BaseElement.Name = variableName
			element.SetRange([]int32{
				int32(capture.Node.StartPosition().Row),
				int32(capture.Node.StartPosition().Column),
				int32(capture.Node.EndPosition().Row),
				int32(capture.Node.EndPosition().Column),
			})
		case types.ElementTypeVariableValue:
			element.Content = []byte(content)
		}
	}

	// 只添加一次元素
	if len(elements) == 0 && element.BaseElement.Name != "" {
		elements = append(elements, element)
	}

	return elements, nil
}

// isDestructuringPattern 检查是否为解构赋值模式
func isDestructuringPattern(node *sitter.Node) bool {
	// 检查是否是变量声明节点
	if node.Kind() != string(types.NodeKindVariableDeclarator) {
		return false
	}

	// 获取name字段，检查是否为解构模式
	nameNode := node.ChildByFieldName("name")
	if nameNode == nil {
		return false
	}

	nodeKind := nameNode.Kind()
	// 检查是否为数组或对象解构模式
	return nodeKind == string(types.NodeKindArrayPattern) || nodeKind == string(types.NodeKindObjectPattern)
}

// handleDestructuring 处理解构赋值
func handleDestructuring(node *sitter.Node, content []byte) ([]Element, error) {
	elements := []Element{}

	// 获取左侧解构模式和右侧引用值
	nameNode := node.ChildByFieldName("name")
	valueNode := findRightNode(node)

	if nameNode == nil || valueNode == nil {
		return elements, nil
	}

	// 获取作用域
	scope := determineVariableScope(node)

	// 处理右侧引用
	var refPath string
	if isClassOrStructReference(valueNode) {
		refPath = extractReferencePath(valueNode, content)

		// 创建引用元素
		refElement := &Reference{
			BaseElement: &BaseElement{
				Name:  string(valueNode.Utf8Text(content)),
				Scope: types.ScopeFile,
				Type:  types.ElementTypeReference,
			},
			Owner: string(valueNode.Utf8Text(content)),
		}

		// 设置内容
		refElement.Content = []byte("ref:" + refPath)

		// 设置范围
		refElement.SetRange([]int32{
			int32(valueNode.StartPosition().Row),
			int32(valueNode.StartPosition().Column),
			int32(valueNode.EndPosition().Row),
			int32(valueNode.EndPosition().Column),
		})

		elements = append(elements, refElement)
	}

	// 处理左侧变量
	nodeKind := types.ToNodeKind(nameNode.Kind())
	if nodeKind == types.NodeKindArrayPattern || nodeKind == types.NodeKindObjectPattern {
		for i := uint(0); i < nameNode.ChildCount(); i++ {
			identifierNode := nameNode.Child(i)
			if identifierNode.Kind() == string(types.Identifier) || identifierNode.Kind() == string(types.NodeKindShorthandPropertyIdentifierPattern) {
				varName := identifierNode.Utf8Text(content)
				varElement := createVariableElement(string(varName), identifierNode, scope)
				elements = append(elements, varElement)
			}
		}
	}

	return elements, nil
}

// createVariableElement 创建变量元素
func createVariableElement(name string, node *sitter.Node, scope types.Scope) *Variable {
	variable := &Variable{
		BaseElement: &BaseElement{
			Name:  name,
			Scope: scope,
			Type:  types.ElementTypeVariable,
		},
		VariableType: nil,
	}

	// 设置范围
	variable.SetRange([]int32{
		int32(node.StartPosition().Row),
		int32(node.StartPosition().Column),
		int32(node.EndPosition().Row),
		int32(node.EndPosition().Column),
	})

	return variable
}

// determineVariableScope 根据节点的上下文确定变量的作用域
func determineVariableScope(node *sitter.Node) types.Scope {
	// 检查父节点
	var current *sitter.Node = node
	maxDepth := 5 // 限制向上查找的层数，防止无限循环

	// 特殊处理：对于variable_declarator节点，找到其父节点(通常是declaration)
	if node.Kind() == string(types.NodeKindVariableDeclarator) {
		parent := node.Parent()
		if parent != nil {
			// 查找声明的类型：let/const是词法(块级)作用域，var是函数作用域
			if parent.Kind() == string(types.NodeKindLexicalDeclaration) {
				return types.ScopeBlock
			} else if parent.Kind() == string(types.NodeKindVariableDeclaration) {
				// 对于var声明，需要向上查找第一个函数作用域或文件作用域
				current = parent // 从变量声明的父节点开始向上查找
			}
		}
	}

	// 从当前节点开始，逐级向上查找作用域容器
	for i := 0; i < maxDepth; i++ {
		// 先检查当前节点
		if i == 0 && current != node {
			// 跳过当前节点检查，因为已经在特殊处理中检查过
		} else {
			// 检查当前节点类型
			switch {
			case isBlockScopeContainer(current.Kind()):
				return types.ScopeBlock
			case isFunctionScopeContainer(current.Kind()):
				return types.ScopeFunction
			case isClassScopeContainer(current.Kind()):
				return types.ScopeClass
			case isFileScopeContainer(current.Kind()):
				return types.ScopeFile
			}
		}

		// 获取父节点
		parent := current.Parent()
		if parent == nil {
			break
		}

		current = parent
	}

	// 默认为文件作用域
	return types.ScopeFile
}

// isBlockScopeContainer 判断节点类型是否为块级作用域容器
func isBlockScopeContainer(nodeKind string) bool {
	// 检查是否为代码块等
	blockScopeContainers := []string{
		"statement_block",
		"for_statement",
		"for_in_statement",
		"for_of_statement",
		"while_statement",
		"if_statement",
		"else_clause",
		"try_statement",
		"catch_clause",
		"block",
		"lexical_declaration",  // let/const声明
		"variable_declaration", // var声明
	}

	for _, containerKind := range blockScopeContainers {
		if nodeKind == containerKind {
			return true
		}
	}

	return false
}

// isFunctionScopeContainer 判断节点类型是否为函数作用域容器
func isFunctionScopeContainer(nodeKind string) bool {
	// 检查是否为函数相关
	functionScopeContainers := []string{
		"function_declaration",
		"function",
		"arrow_function",
		"generator_function",
		"generator_function_declaration",
		"async_function",
		"async_function_declaration",
		"function_expression",
		"method_definition",
	}

	for _, containerKind := range functionScopeContainers {
		if nodeKind == containerKind {
			return true
		}
	}

	return false
}

// isClassScopeContainer 判断节点类型是否为类作用域容器
func isClassScopeContainer(nodeKind string) bool {
	// 检查是否为类相关
	classScopeContainers := []string{
		"class_declaration",
		"class_body",
		"class",
		"class_expression",
	}

	for _, containerKind := range classScopeContainers {
		if nodeKind == containerKind {
			return true
		}
	}

	return false
}

// isFileScopeContainer 判断节点类型是否为文件级作用域容器
func isFileScopeContainer(nodeKind string) bool {
	// 检查是否为顶层作用域
	fileScopeContainers := []string{
		"program",
		"source_file",
		"module",
		"script",
	}

	for _, containerKind := range fileScopeContainers {
		if nodeKind == containerKind {
			return true
		}
	}

	return false
}

// findRightNode 查找赋值右侧节点
func findRightNode(node *sitter.Node) *sitter.Node {
	// 查找赋值右侧
	rightNode := node.ChildByFieldName("value")
	if rightNode == nil {
		// 尝试查找变量声明中的第三个子节点（通常是赋值右侧）
		if node.ChildCount() >= 3 {
			rightNode = node.Child(2)
		}
	}
	return rightNode
}

// parseArrowFunctionParameters 解析箭头函数参数
func parseArrowFunctionParameters(functionElement *Function, node *sitter.Node, content []byte) {
	// 查找formal_parameters节点
	paramsNode := node.ChildByFieldName("parameters")
	if paramsNode == nil {
		return
	}

	// 遍历所有子节点，查找identifier类型的节点作为参数
	for i := uint(0); i < paramsNode.ChildCount(); i++ {
		child := paramsNode.Child(i)
		if child != nil && child.Kind() == types.Identifier {
			// 提取参数名
			paramName := child.Utf8Text(content)

			// 添加到参数列表
			functionElement.Parameters = append(functionElement.Parameters, Parameter{
				Name: string(paramName),
				Type: nil, // JavaScript是动态类型语言，参数类型通常不显式声明
			})
		}
	}
}

// extractReferencePath 提取引用的路径
func extractReferencePath(node *sitter.Node, content []byte) string {
	// 如果是标识符，直接返回名称
	if node.Kind() == string(types.NodeKindIdentifier) {
		return string(node.Utf8Text(content))
	}

	// 如果是成员表达式，尝试构建完整路径
	if node.Kind() == string(types.NodeKindMemberExpression) {
		objectNode := node.ChildByFieldName("object")
		propertyNode := node.ChildByFieldName("property")

		if objectNode != nil && propertyNode != nil {
			objectText := objectNode.Utf8Text(content)
			propertyText := propertyNode.Utf8Text(content)
			return string(objectText) + "." + string(propertyText)
		}
	}

	// 如果是new表达式，获取构造函数名称
	if node.Kind() == string(types.NodeKindNewExpression) {
		constructorNode := node.Child(1) // new之后的第一个子节点通常是构造函数
		if constructorNode != nil {
			return string(constructorNode.Utf8Text(content))
		}
	}

	// 默认返回节点文本
	return string(node.Utf8Text(content))
}

// isArrowFunction 判断节点是否为箭头函数
func isArrowFunction(node *sitter.Node) bool {
	// 检查节点类型
	nodeKind := node.Kind()
	if nodeKind == string(types.NodeKindArrowFunction) {
		return true
	}

	// 如果是变量赋值表达式，检查其内容
	return strings.Contains(nodeKind, "function") ||
		strings.Contains(nodeKind, "arrow")
}

// isClassOrStructReference 判断节点是否为类或结构体引用
func isClassOrStructReference(node *sitter.Node) bool {
	nodeKind := node.Kind()

	// 检查常见的引用类型
	if nodeKind == string(types.NodeKindIdentifier) || nodeKind == string(types.NodeKindMemberExpression) {
		return true
	}

	// 检查是否为new表达式
	if nodeKind == string(types.NodeKindNewExpression) {
		return true
	}

	// 检查对象表达式 (可能是结构体字面量)
	if nodeKind == string(types.NodeKindObject) {
		return false // 这是字面量，不是引用
	}

	return false
}

func (js *JavaScriptResolver) resolveInterface(ctx context.Context, element *Interface, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("not support")
}

func (js *JavaScriptResolver) resolveCall(ctx context.Context, element *Call, rc *ResolveContext) ([]Element, error) {
	// 检查是否为require调用
	if isRequireCallCapture(rc) {
		// 处理为import而不是call
		return js.handleRequireCall(ctx, rc)
	}

	elements := []Element{element}

	// 如果没有匹配信息，直接返回
	if rc.Match == nil || rc.Match.Captures == nil || len(rc.Match.Captures) == 0 {
		return elements, nil
	}

	// 设置默认类型
	element.Type = types.ElementTypeFunctionCall
	// 处理所有捕获节点
	for _, capture := range rc.Match.Captures {
		// 跳过无效节点
		if capture.Node.IsMissing() || capture.Node.IsError() {
			continue
		}

		nodeCaptureName := rc.CaptureNames[capture.Index]
		updateRootElement(element, &capture, nodeCaptureName, rc.SourceFile.Content)
		switch types.ToElementType(nodeCaptureName) {
		case types.ElementTypeFunctionCall:
			// 处理整个函数调用表达式
			funcNode := capture.Node.ChildByFieldName("function")
			if funcNode != nil {
				// 检查是否为匿名函数立即调用模式（IIFE）
				switch types.ToNodeKind(funcNode.Kind()) {
				case types.NodeKindFuncLiteral:
					return nil, nil

				// 正常函数调用处理
				case types.NodeKindIdentifier:
					// 简单函数调用
					element.BaseElement.Name = funcNode.Utf8Text(rc.SourceFile.Content)
				case types.NodeKindSelectorExpression, types.NodeKindMemberExpression:
					// 处理成员表达式，支持多层链式调用
					extractMemberExpressionPath(funcNode, element, rc.SourceFile.Content)
				}
			}
		case types.ElementTypeFunctionArguments:
			// 专门处理参数列表
			processArguments(element, capture.Node, rc.SourceFile.Content)
		case types.ElementTypeCallName:
			// 从成员表达式中提取函数名和所有者
			if types.ToNodeKind(capture.Node.Kind()) == types.NodeKindMemberExpression {
				extractCallNameAndOwner(&capture.Node, element, rc.SourceFile.Content)
			} else {
				// 如果不是成员表达式，直接获取函数名
				element.BaseElement.Name = capture.Node.Utf8Text(rc.SourceFile.Content)
			}
		}
	}
	return elements, nil
}

// extractMemberExpressionPath 递归提取成员表达式的完整路径
func extractMemberExpressionPath(node *sitter.Node, call *Call, content []byte) {
	if node == nil {
		return
	}

	// 提取函数名和对象路径
	var funcName string
	var objPath []string

	// 递归处理成员表达式
	current := node
	for {
		// 对象和属性
		objectNode := current.ChildByFieldName("object")
		propertyNode := current.ChildByFieldName("property")

		if propertyNode != nil {
			// 最底层的属性是函数名
			if funcName == "" {
				funcName = propertyNode.Utf8Text(content)
			} else {
				// 中间层的属性是路径的一部分
				objPath = append([]string{propertyNode.Utf8Text(content)}, objPath...)
			}
		}

		if objectNode == nil {
			break
		}

		// 检查对象是否还是成员表达式
		if objectNode.Kind() == string(types.NodeKindMemberExpression) {
			current = objectNode
			continue
		}

		// 处理最顶层对象
		objPath = append([]string{objectNode.Utf8Text(content)}, objPath...)
		break
	}

	// 设置函数名
	if funcName != "" {
		call.BaseElement.Name = funcName
	}

	// 设置对象路径作为所有者
	if len(objPath) > 0 {
		call.Owner = strings.Join(objPath, ".")
	}
}

// processArguments 处理JavaScript函数调用的参数
func processArguments(element *Call, argsNode sitter.Node, content []byte) {
	// 初始化参数列表
	if element.Parameters == nil {
		element.Parameters = []*Parameter{}
	}

	// 遍历所有参数
	for i := uint(0); i < argsNode.ChildCount(); i++ {
		argNode := argsNode.Child(i)
		if argNode == nil || argNode.IsError() || argNode.IsMissing() {
			continue
		}

		// 过滤掉逗号等分隔符
		if argNode.Kind() == "," || argNode.Kind() == "(" || argNode.Kind() == ")" {
			continue
		}
		argName := argNode.Utf8Text(content)
		// 创建参数对象
		param := &Parameter{
			Name: argName, // JavaScript参数通常没有命名
			Type: nil,
		}

		// 添加参数
		element.Parameters = append(element.Parameters, param)
	}
}

// extractCallNameAndOwner 从成员表达式中提取函数名和所有者
func extractCallNameAndOwner(node *sitter.Node, call *Call, content []byte) {
	if node == nil {
		return
	}

	nodeText := string(node.Utf8Text(content))

	// 处理诸如 "o.next" 或 "a.b.c" 这样的表达式
	if strings.Contains(nodeText, ".") {
		parts := strings.Split(nodeText, ".")
		if len(parts) > 0 {
			// 最后一部分是函数名
			call.BaseElement.Name = parts[len(parts)-1]

			// 前面的部分组成所有者
			if len(parts) > 1 {
				call.Owner = strings.Join(parts[:len(parts)-1], ".")
			}
		}
	} else {
		// 如果没有点，整个文本就是函数名
		call.BaseElement.Name = nodeText
	}
}

// isRequireCall 检查节点是否为require函数调用
func isRequireCall(node *sitter.Node, content []byte) bool {
	// 检查是否为函数调用
	if node.Kind() != "call_expression" {
		return false
	}

	// 获取函数名
	funcNode := node.ChildByFieldName("function")
	if funcNode == nil {
		return false
	}

	// 检查是否为标识符且名称为"require"
	return funcNode.Kind() == "identifier" && string(funcNode.Utf8Text(content)) == "require"
}

// isRequireCallCapture 检查捕获是否为require函数调用
func isRequireCallCapture(rc *ResolveContext) bool {
	if rc.Match == nil || len(rc.Match.Captures) == 0 {
		return false
	}

	// 检查第一个捕获是否为函数调用
	rootCapture := rc.Match.Captures[0]
	if rootCapture.Node.Kind() != "call_expression" {
		return false
	}

	// 获取函数名
	funcNode := rootCapture.Node.ChildByFieldName("function")
	if funcNode == nil {
		return false
	}

	// 检查是否为"require"
	return funcNode.Kind() == "identifier" && string(funcNode.Utf8Text(rc.SourceFile.Content)) == "require"
}

// handleRequireCall 将require函数调用处理为import
func (js *JavaScriptResolver) handleRequireCall(ctx context.Context, rc *ResolveContext) ([]Element, error) {
	// 创建import元素
	importElement := &Import{
		BaseElement: &BaseElement{
			Type:  types.ElementTypeImport,
			Scope: types.ScopeFile,
		},
	}

	rootCapture := rc.Match.Captures[0]

	// 查找require调用的参数(模块路径)
	argsNode := rootCapture.Node.ChildByFieldName("arguments")
	if argsNode != nil {
		for i := uint(0); i < argsNode.ChildCount(); i++ {
			argNode := argsNode.Child(i)
			if argNode != nil && argNode.Kind() == "string" {
				// 去除引号
				importElement.Source = strings.Trim(string(argNode.Utf8Text(rc.SourceFile.Content)), "'\"")
				break
			}
		}
	}

	// 查找变量赋值语句来获取导入名称
	// 向上查找父节点，直到找到variable_declarator
	var currentNode = &rootCapture.Node
	for i := 0; i < 3; i++ { // 限制向上查找的层数
		parent := currentNode.Parent()
		if parent == nil {
			break
		}

		if parent.Kind() == "variable_declarator" {
			// 找到变量声明，获取变量名
			nameNode := parent.ChildByFieldName("name")
			if nameNode != nil {
				importElement.Name = string(nameNode.Utf8Text(rc.SourceFile.Content))
				importElement.BaseElement.Name = importElement.Name
				importElement.Content = []byte(importElement.Name)
				break
			}
		}

		currentNode = parent
	}

	// 设置范围
	importElement.SetRange([]int32{
		int32(rootCapture.Node.StartPosition().Row),
		int32(rootCapture.Node.StartPosition().Column),
		int32(rootCapture.Node.EndPosition().Row),
		int32(rootCapture.Node.EndPosition().Column),
	})

	// 添加CommonJS标记
	if importElement.Content != nil {
		importElement.Content = append(importElement.Content, []byte(" (CommonJS)")...)
	}

	return []Element{importElement}, nil
}
