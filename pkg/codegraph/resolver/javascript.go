package resolver

import (
	"codebase-indexer/pkg/codegraph/types"
	"context"
	"regexp"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// 包级别的正则表达式，只编译一次
var (
	arrayIndexRegex = regexp.MustCompile(`\[[^\]]+\]`)
)

var jsReplacer = strings.NewReplacer(
	"[", "",
	"]", "",
	"*", "",
	"(", "",
	")", "",
)

type JavaScriptResolver struct {
}

var _ ElementResolver = &JavaScriptResolver{}

func (js *JavaScriptResolver) Resolve(ctx context.Context, element Element, rc *ResolveContext) ([]Element, error) {
	return resolve(ctx, js, element, rc)
}

func (js *JavaScriptResolver) resolveImport(ctx context.Context, element *Import, rc *ResolveContext) ([]Element, error) {

	elements := []Element{element}
	rootCapture := rc.Match.Captures[0]
	updateRootElement(element, &rootCapture, rc.CaptureNames[rootCapture.Index], rc.SourceFile.Content)
	for _, capture := range rc.Match.Captures {
		if capture.Node.IsMissing() || capture.Node.IsError() {
			continue
		}
		nodeCaptureName := rc.CaptureNames[capture.Index]
		content := capture.Node.Utf8Text(rc.SourceFile.Content)
		switch types.ToElementType(nodeCaptureName) {
		case types.ElementTypeImportName:
			element.Name = content
		case types.ElementTypeImportAlias:
			element.Alias = content
		case types.ElementTypeImportSource:
			// 先去掉单引号，再去掉双引号
			element.Source = strings.Trim(strings.Trim(content, types.SingleQuote), types.DoubleQuote)
		}
	}

	// 确保有名称：如果还没有名称且有Source，从Source路径中提取最后一个部分
	if element.Name == types.EmptyString && element.Source != types.EmptyString {
		pathParts := strings.Split(element.Source, types.Slash)
		if len(pathParts) > 0 {
			element.Name = pathParts[len(pathParts)-1]
		}
		if strings.Contains(element.Name, types.Dot) {
			element.Name = strings.SplitN(element.Name, types.Dot, 2)[0]
		}
	}
	element.Scope = types.ScopePackage
	return elements, nil
}

func (js *JavaScriptResolver) resolvePackage(ctx context.Context, element *Package, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	//此语言不支持
	panic("not support")
}

func (js *JavaScriptResolver) resolveFunction(ctx context.Context, element *Function, rc *ResolveContext) ([]Element, error) {
	elements := []Element{element}
	rootCapture := rc.Match.Captures[0]
	if isArrowFunctionImport(&rootCapture.Node, rc.SourceFile.Content) {
		return []Element{}, nil
	}
	for _, capture := range rc.Match.Captures {
		if capture.Node.IsMissing() || capture.Node.IsError() {
			continue
		}
		nodeCaptureName := rc.CaptureNames[capture.Index]
		content := capture.Node.Utf8Text(rc.SourceFile.Content)
		switch types.ToElementType(nodeCaptureName) {
		case types.ElementTypeFunction:
			updateRootElement(element, &rootCapture, rc.CaptureNames[rootCapture.Index], rc.SourceFile.Content)
			if isExportStatement(&capture.Node) {
				element.Scope = types.ScopePackage
			} else {
				element.Scope = types.ScopeFile
			}
			element.Declaration.Modifier = extractModifiers(content)
		case types.ElementTypeFunctionName:
			element.BaseElement.Name = content
			element.Declaration.Name = content
		case types.ElementTypeFunctionParameters:
			parseJavaScriptParameters(element, capture.Node, rc.SourceFile.Content)
		}
	}
	return elements, nil
}

func (js *JavaScriptResolver) resolveMethod(ctx context.Context, element *Method, rc *ResolveContext) ([]Element, error) {
	elements := []Element{element}
	rootCap := rc.Match.Captures[0]
	for _, capture := range rc.Match.Captures {
		if capture.Node.IsMissing() || capture.Node.IsError() {
			continue
		}
		nodeCaptureName := rc.CaptureNames[capture.Index]
		content := capture.Node.Utf8Text(rc.SourceFile.Content)
		switch types.ToElementType(nodeCaptureName) {
		case types.ElementTypeMethod:
			updateRootElement(element, &rootCap, rc.CaptureNames[rootCap.Index], rc.SourceFile.Content)
			element.Declaration.Modifier = extractModifiers(content)
		case types.ElementTypeMethodName:
			element.BaseElement.Name = content
			element.Declaration.Name = content
		case types.ElementTypeMethodParameters:
			parseJavaScriptParameters(element, capture.Node, rc.SourceFile.Content)
		}
	}
	ownerNode := findMethodOwner(&rootCap.Node)
	var ownerKind types.NodeKind
	if ownerNode != nil {
		element.Owner = extractNodeName(ownerNode, rc.SourceFile.Content)
		ownerKind = types.ToNodeKind(ownerNode.Kind())
	}
	// 补充作用域
	element.BaseElement.Scope = getScopeFromModifiers(element.Declaration.Modifier, ownerKind)
	return elements, nil
}

func (js *JavaScriptResolver) resolveClass(ctx context.Context, element *Class, rc *ResolveContext) ([]Element, error) {
	elements := []Element{element}
	element.Fields = []*Field{}
	element.Methods = []*Method{}
	element.SuperClasses = []string{}
	rootCapure := rc.Match.Captures[0]
	captureName := rc.CaptureNames[rootCapure.Index]
	updateRootElement(element, &rootCapure, captureName, rc.SourceFile.Content)
	for _, capture := range rc.Match.Captures {
		nodeCaptureName := rc.CaptureNames[capture.Index]
		content := capture.Node.Utf8Text(rc.SourceFile.Content)
		switch types.ToElementType(nodeCaptureName) {
		case types.ElementTypeClass:
			element.Type = types.ElementTypeClass
			if isExportStatement(&capture.Node) {
				element.Scope = types.ScopePackage
			} else {
				element.Scope = types.ScopeFile
			}
		case types.ElementTypeClassName:
			element.BaseElement.Name = content
		case types.ElementTypeClassExtends:
			Node := capture.Node
			content = Node.Child(1).Utf8Text(rc.SourceFile.Content)
			element.SuperClasses = append(element.SuperClasses, content)
		}
	}
	cls, references := parseJavaScriptClassBody(&rootCapure.Node, rc.SourceFile.Content, element.BaseElement.Name, rc.SourceFile.Path)
	element.Fields = cls.Fields
	element.Methods = cls.Methods
	// 收集所有引用元素
	elements = append(elements, references...)
	return elements, nil
}

func (js *JavaScriptResolver) resolveVariable(ctx context.Context, element *Variable, rc *ResolveContext) ([]Element, error) {
	/*
		JavaScript中变量都没有类型，Variable元素的VariableType为空
	*/
	elements := []Element{}
	rootCapure := rc.Match.Captures[0]
	rootCaptureName := rc.CaptureNames[rootCapure.Index]
	updateRootElement(element, &rootCapure, rootCaptureName, rc.SourceFile.Content)
	// 检查是否为解构赋值
	if isDestructuringPattern(&rootCapure.Node) {
		// 添加handleDestructuringWithPath函数，传递元素路径
		return js.handleDestructuringWithPath(&rootCapure.Node, rc.SourceFile.Content, element)
	}
	// 检查是否为import导入和箭头函数
	if rc.Match != nil && len(rc.Match.Captures) > 0 {
		if rightNode := findRightNode(&rootCapure.Node); rightNode != nil {
			if isRequireImport(rightNode, rc.SourceFile.Content) {
				return []Element{}, nil
			}
			if rightNode.Kind() == string(types.NodeKindArrowFunction) {
				return []Element{}, nil
			}
		}
	}
	for _, capture := range rc.Match.Captures {
		nodeCaptureName := rc.CaptureNames[capture.Index]
		captureType := types.ToElementType(nodeCaptureName)
		if capture.Node.Kind() == string(types.NodeKindVariableDeclarator) {
			valueNode := capture.Node.ChildByFieldName("value")
			if isImportExpression(valueNode, rc.SourceFile.Content) {
				return []Element{}, nil
			}
		}
		// 如果不是特殊类型，作为普通变量处理
		switch captureType {
		case types.ElementTypeVariable:
			element.Type = types.ElementTypeVariable
			// 根据父节点判断变量作用域
			element.Scope = determineVariableScope(&capture.Node)
		case types.ElementTypeVariableName:
			element.BaseElement.Name = capture.Node.Utf8Text(rc.SourceFile.Content)
		}
	}
	elements = append(elements, element)
	return elements, nil
}

func (js *JavaScriptResolver) resolveInterface(ctx context.Context, element *Interface, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("not support")
}

func (js *JavaScriptResolver) resolveCall(ctx context.Context, element *Call, rc *ResolveContext) ([]Element, error) {
	// 处理为import而不是call
	// if isRequireCallCapture(rc) {
	// 	return js.handleRequireCall(element, rc)
	// }
	elements := []Element{element}
	rootCapture := rc.Match.Captures[0]
	updateRootElement(element, &rootCapture, rc.CaptureNames[rootCapture.Index], rc.SourceFile.Content)
	for _, capture := range rc.Match.Captures {
		nodeCaptureName := rc.CaptureNames[capture.Index]
		switch types.ToElementType(nodeCaptureName) {
		case types.ElementTypeFunctionCall, types.ElementTypeMethodCall:
			funcNode := capture.Node.ChildByFieldName("function")
			if funcNode != nil {
				switch types.ToNodeKind(funcNode.Kind()) {
				case types.NodeKindFuncLiteral:
					return nil, nil
				case types.NodeKindIdentifier:
					element.BaseElement.Name = funcNode.Utf8Text(rc.SourceFile.Content)
				case types.NodeKindSelectorExpression, types.NodeKindMemberExpression:
					extractMemberExpressionPath(funcNode, element, rc.SourceFile.Content)
				}
			}
		case types.ElementTypeFunctionArguments, types.ElementTypeCallArguments:
			processArguments(element, capture.Node)
		case types.ElementTypeStructCall:
			refPathMap := extractReferencePath(&capture.Node, rc.SourceFile.Content)
			element.BaseElement.Name = refPathMap["property"]
			element.Owner = refPathMap["object"]
		}
	}
	if element.Scope == types.EmptyString {
		element.Scope = types.ScopeFunction
	}

	return elements, nil
}

type ParameterSetter interface {
	SetParameters([]Parameter)
}

// 为 Function 和 Method 实现这个接口
func (f *Function) SetParameters(params []Parameter) {
	f.Declaration.Parameters = params
}

func (m *Method) SetParameters(params []Parameter) {
	m.Declaration.Parameters = params
}

// 通用的参数解析函数
func parseJavaScriptParameters(element ParameterSetter, paramsNode sitter.Node, content []byte) {
	parameters := make([]Parameter, 0)
	for i := uint(0); i < paramsNode.ChildCount(); i++ {
		child := paramsNode.Child(i)
		if child != nil && child.Kind() == types.Identifier {
			paramName := child.Utf8Text(content)
			parameters = append(parameters, Parameter{
				Name: paramName,
				Type: nil, // JavaScript作为动态语言，参数类型通常无法从语法中直接获取
			})
		}
	}
	element.SetParameters(parameters)
}

// extractModifiers 从函数或方法声明中提取修饰符
// elementType: 元素类型，如"function"或"method"
func extractModifiers(content string) string {
	// JavaScript中函数和方法的可能修饰符
	modifiers := []string{"async", "static", "get", "set", "*"}
	result := types.EmptyString

	// 按空格分割函数声明
	for _, mod := range modifiers {
		if containsModifier(content, mod) {
			if result != types.EmptyString {
				result += types.Space
			}
			result += mod
		}
	}

	return result
}

// containsModifier 判断字符串是否包含指定的修饰符
func containsModifier(content string, modifier string) bool {
	// 生成器函数特殊处理
	if modifier == types.Star {
		// 只在函数声明的开始部分查找生成器修饰符
		// 检查 "function*" 或对象方法中的 "* methodName"
		if strings.Contains(content, "function*") {
			return true
		}

		// 对于对象方法: * methodName() { ... }
		// 确保 * 出现在行首或大括号后，且后面跟着标识符
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "* ") {
				// 确保 * 后面跟着的是标识符而不是运算符
				rest := strings.TrimSpace(trimmed[2:])
				if len(rest) > 0 && (rest[0] >= 'a' && rest[0] <= 'z' || rest[0] >= 'A' && rest[0] <= 'Z' || rest[0] == '_' || rest[0] == '$') {
					return true
				}
			}
		}
		return false
	}

	// 其他修饰符需要确保是单独的单词，且不在等号右边
	// 分割成等号左右两部分，只在左边查找修饰符
	parts := strings.SplitN(content, "=", 2)
	searchArea := parts[0] // 只在等号左边查找修饰符

	words := strings.Fields(searchArea)
	for _, word := range words {
		// 清理可能的标点符号
		cleanWord := strings.Trim(word, "(){}[],;:")
		if cleanWord == modifier {
			return true
		}
	}
	return false
}

// parseJavaScriptClassBody 解析JavaScript类体，提取字段和方法
func parseJavaScriptClassBody(node *sitter.Node, content []byte, className string, path string) (*Class, []Element) {
	class := &Class{
		BaseElement: &BaseElement{
			Name:  className,
			Scope: types.ScopeFile,
			Type:  types.ElementTypeClass,
		},
		Methods: []*Method{},
		Fields:  []*Field{},
	}

	// 收集引用元素
	var references []Element

	// 查找class_body节点
	var classBodyNode *sitter.Node
	// 类声明节点的最后一个子节点通常是类体
	classBodyNode = node.Child(node.ChildCount() - 1)
	if classBodyNode == nil {
		return class, references
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
			method.Owner = className
			method.Path = path
			if method != nil {
				class.Methods = append(class.Methods, method)
			}
		case types.NodeKindFieldDefinition:
			// 处理字段
			field, ref := parseJavaScriptFieldNode(memberNode, content, path)
			if field != nil {
				class.Fields = append(class.Fields, field)
				// 如果存在引用元素，添加到引用列表中
				if ref != nil {
					references = append(references, ref)
				}
			}
		}
	}

	return class, references
}

// parseJavaScriptMethodNode 解析JavaScript方法节点
func parseJavaScriptMethodNode(node *sitter.Node, content []byte, className string) *Method {
	method := &Method{}
	method.Owner = className

	// 设置默认作用域和修饰符
	method.BaseElement = &BaseElement{
		Scope: types.ScopeFile,
	}
	method.Declaration = &Declaration{
		Modifier: types.ModifierPublic, // JavaScript默认为public
	}

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
		method.Declaration.Parameters = make([]Parameter, 0)
		for j := uint(0); j < paramsNode.ChildCount(); j++ {
			paramChild := paramsNode.Child(j)
			if paramChild != nil && paramChild.Kind() == types.Identifier {
				paramName := paramChild.Utf8Text(content)
				method.Declaration.Parameters = append(method.Declaration.Parameters, Parameter{
					Name: paramName,
					Type: nil,
				})
			}
		}
	}

	method.Type = types.ElementTypeMethod

	// 检查修饰符
	methodContent := node.Utf8Text(content)
	extractedModifiers := extractModifiers(methodContent)
	if extractedModifiers != types.EmptyString {
		method.Declaration.Modifier = extractedModifiers + types.Space + method.Declaration.Modifier
	}

	return method
}

// parseJavaScriptFieldNode 解析JavaScript字段节点
func parseJavaScriptFieldNode(node *sitter.Node, content []byte, path string) (*Field, *Reference) {
	field := &Field{}
	var ref *Reference

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
			return nil, nil
		}
	}

	fieldName := nameNode.Utf8Text(content)
	isPrivate := strings.HasPrefix(fieldName, types.Hash)

	// 处理私有字段
	if isPrivate {
		fieldName = strings.TrimPrefix(fieldName, types.Hash)
		field.Modifier = types.ModifierPrivate
	} else {
		field.Modifier = types.ModifierPublic
	}

	field.Name = fieldName

	// 查找字段值（可能是引用类型）
	valueNode := node.ChildByFieldName("value")
	if valueNode != nil {
		// 处理字段值，检查是否为引用类型
		fieldType := ""
		nodeKind := valueNode.Kind()

		// 根据值节点的类型确定字段类型
		switch nodeKind {
		case string(types.NodeKindNewExpression):
			// 从new表达式中提取类型
			constructorNode := valueNode.Child(1)
			if constructorNode != nil {
				fieldType = constructorNode.Utf8Text(content)

				// 检查是否是成员表达式
				if constructorNode.Kind() == string(types.NodeKindMemberExpression) {
					// 创建引用元素
					ref = &Reference{
						BaseElement: &BaseElement{
							Type: types.ElementTypeReference,
							Path: path,
							Range: []int32{
								int32(constructorNode.StartPosition().Row),
								int32(constructorNode.StartPosition().Column),
								int32(constructorNode.EndPosition().Row),
								int32(constructorNode.EndPosition().Column),
							},
						},
					}

					// 处理成员表达式
					objectNode := constructorNode.ChildByFieldName("object")
					propertyNode := constructorNode.ChildByFieldName("property")
					if objectNode != nil && propertyNode != nil {
						ref.Owner = objectNode.Utf8Text(content)
						ref.BaseElement.Name = propertyNode.Utf8Text(content)
						fieldType = ref.Owner + types.Dot + ref.BaseElement.Name
					} else {
						ref.BaseElement.Name = fieldType
					}
					ref.BaseElement.Scope = types.ScopeBlock
				}
			}
		case string(types.NodeKindIdentifier):
			// 标识符可能是类型引用
			fieldType = valueNode.Utf8Text(content)

			// 如果不是基本类型，创建引用元素
			if !isJavaScriptPrimitiveType(fieldType) {
				ref = &Reference{
					BaseElement: &BaseElement{
						Name: fieldType,
						Type: types.ElementTypeReference,
						Path: path,
						Range: []int32{
							int32(valueNode.StartPosition().Row),
							int32(valueNode.StartPosition().Column),
							int32(valueNode.EndPosition().Row),
							int32(valueNode.EndPosition().Column),
						},
					},
				}
				ref.BaseElement.Scope = types.ScopeBlock
			}
		case string(types.NodeKindMemberExpression):
			// 成员表达式是引用类型
			objectNode := valueNode.ChildByFieldName("object")
			propertyNode := valueNode.ChildByFieldName("property")

			if objectNode != nil && propertyNode != nil {
				// 创建引用元素
				ref = &Reference{
					BaseElement: &BaseElement{
						Type: types.ElementTypeReference,
						Path: path,
						Range: []int32{
							int32(valueNode.StartPosition().Row),
							int32(valueNode.StartPosition().Column),
							int32(valueNode.EndPosition().Row),
							int32(valueNode.EndPosition().Column),
						},
					},
					Owner: objectNode.Utf8Text(content),
				}
				ref.BaseElement.Scope = types.ScopeBlock
				ref.BaseElement.Name = propertyNode.Utf8Text(content)
				fieldType = ref.Owner + types.Dot + ref.BaseElement.Name
			} else {
				fieldType = valueNode.Utf8Text(content)
			}
		}
		field.Type = fieldType
	} else {
		field.Type = "any" // JavaScript中字段类型默认为any
	}

	return field, ref
}

// isJavaScriptPrimitiveType 检查类型是否为JavaScript基本数据类型
func isJavaScriptPrimitiveType(typeName string) bool {
	primitiveTypes := []string{
		"string", "number", "boolean", "null", "undefined",
		"Symbol", "BigInt", "any", "void", "object", "array",
		"function", "regexp", "date", "promise", "map", "set", "true", "false",
	}

	typeName = strings.ToLower(typeName)
	for _, t := range primitiveTypes {
		if typeName == t {
			return true
		}
	}
	return false
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

// handleDestructuringWithPath 带路径的解构赋值处理
func (js *JavaScriptResolver) handleDestructuringWithPath(node *sitter.Node, content []byte, element *Variable) ([]Element, error) {
	elements := []Element{}

	// 获取左侧解构模式和右侧引用值
	nameNode := node.ChildByFieldName("name")
	valueNode := findRightNode(node)
	if nameNode == nil || valueNode == nil {
		return elements, nil
	}
	// 获取作用域
	scope := determineVariableScope(node)
	// 处理左侧变量
	nodeKind := types.ToNodeKind(nameNode.Kind())
	if nodeKind == types.NodeKindArrayPattern || nodeKind == types.NodeKindObjectPattern {
		for i := uint(0); i < nameNode.ChildCount(); i++ {
			identifierNode := nameNode.Child(i)
			varName := types.EmptyString
			if identifierNode.Kind() == string(types.Identifier) || identifierNode.Kind() == string(types.NodeKindShorthandPropertyIdentifierPattern) {
				varName = CleanParam(identifierNode.Utf8Text(content))
			} else if identifierNode.Kind() == string(types.NodeKindPairPattern) {
				valueNode := identifierNode.ChildByFieldName("value")
				if valueNode.Kind() == string(types.Identifier) {
					elements = append(elements, createVariableElement(valueNode.Utf8Text(content), scope, element))
				}
				for i := uint(0); i < valueNode.ChildCount(); i++ {
					childNode := valueNode.Child(i)
					if childNode.Kind() == string(types.NodeKindShorthandPropertyIdentifierPattern) || childNode.Kind() == string(types.Identifier) {
						elements = append(elements, createVariableElement(childNode.Utf8Text(content), scope, element))
					}
				}
				continue
			} else if identifierNode.Kind() == string(types.NodeKindRestPattern) {
				idNode := identifierNode.Child(1)
				if idNode != nil {
					varName = CleanParam(idNode.Utf8Text(content))
				}
			}
			if varName != types.EmptyString {
				varElement := createVariableElement(string(varName), scope, element)
				elements = append(elements, varElement)
			}
		}
	}

	return elements, nil
}

// createVariableElement 创建变量元素
func createVariableElement(name string, scope types.Scope, element *Variable) *Variable {
	variable := &Variable{
		BaseElement: &BaseElement{
			Name:  name,
			Path:  element.BaseElement.Path,
			Scope: scope,
			Type:  types.ElementTypeVariable,
			Range: element.BaseElement.Range,
		},
		VariableType: nil,
	}
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

// extractReferencePath 递归提取 member_expression 的 object 路径和 property 名称
func extractReferencePath(node *sitter.Node, content []byte) map[string]string {
	result := map[string]string{"object": "", "property": ""}

	// 如果是标识符，直接返回名称
	if node.Kind() == string(types.NodeKindIdentifier) {
		result["property"] = node.Utf8Text(content)
		return result
	}

	// 如果是成员表达式，提取对象和属性
	if node.Kind() == string(types.NodeKindMemberExpression) {
		objectNode := node.ChildByFieldName("object")
		propertyNode := node.ChildByFieldName("property")

		if objectNode != nil && propertyNode != nil {
			// 获取属性名（最右侧部分）
			propertyText := propertyNode.Utf8Text(content)
			result["property"] = propertyText

			// 获取对象部分（左侧部分）
			// 如果对象是另一个成员表达式，则需要递归处理
			if objectNode.Kind() == string(types.NodeKindMemberExpression) {
				// 对于嵌套的成员表达式，递归处理
				subResult := extractReferencePath(objectNode, content)
				if subResult["object"] != types.EmptyString {
					result["object"] = subResult["object"] + types.Dot + subResult["property"]
				} else {
					result["object"] = subResult["property"]
				}
			} else {
				// 简单对象，直接获取文本
				result["object"] = objectNode.Utf8Text(content)
			}

			return result
		}
	}

	// 如果是new表达式，获取构造函数名称
	if node.Kind() == string(types.NodeKindNewExpression) {
		constructorNode := node.Child(1) // new之后的第一个子节点通常是构造函数
		if constructorNode != nil {
			// 检查构造函数是否是成员表达式
			if constructorNode.Kind() == string(types.NodeKindMemberExpression) {
				return extractReferencePath(constructorNode, content)
			} else {
				result["property"] = constructorNode.Utf8Text(content)
				return result
			}
		}
	}

	if node.Kind() == string(types.NodeKindTypeIdentifier) {
		result["property"] = node.Utf8Text(content)
		return result
	}

	if node.Kind() == string(types.NodeKindQualifiedType) {
		propertyNode := node.ChildByFieldName("name")
		if propertyNode != nil {
			result["property"] = propertyNode.Utf8Text(content)
		}
		objectNode := node.ChildByFieldName("package")
		if objectNode != nil {
			result["object"] = objectNode.Utf8Text(content)
		}
	}

	if node.Kind() == string(types.NodeKindSliceType) || node.Kind() == string(types.NodeKindPointType) || node.Kind() == string(types.NodeKindArrayType) {
		// 检查是否包含泛型子节点
		for i := uint(0); i < node.ChildCount(); i++ {
			child := node.Child(i)
			if child.Kind() == string(types.NodeKindGenericType) {
				// 处理泛型类型：获取基础类型标识符
				result = extractReferencePath(child, content)
				return result
			}
		}
		text := node.Utf8Text(content)

		// 清除[]、*字符和数组索引如[100]、[queueCapacity]等
		cleanText := jsReplacer.Replace(text)
		cleanText = arrayIndexRegex.ReplaceAllString(cleanText, "")
		result["property"] = cleanText
		// 根据.分割，后面为property，前面为object
		if strings.Contains(cleanText, types.Dot) {
			parts := strings.Split(cleanText, types.Dot)
			if len(parts) >= 2 {
				result["object"] = strings.Join(parts[:len(parts)-1], types.Dot)
				result["property"] = parts[len(parts)-1]
			}
		}
	}

	// 处理泛型类型，如 MyType<T, U> 或 *MyType<T, U>
	if node.Kind() == string(types.NodeKindGenericType) {
		// 获取基础类型标识符
		typeIdNode := node.ChildByFieldName("type")
		if typeIdNode != nil {
			typeName := typeIdNode.Utf8Text(content)
			// 根据.分割，后面为property，前面为object
			if strings.Contains(typeName, types.Dot) {
				parts := strings.Split(typeName, types.Dot)
				if len(parts) >= 2 {
					result["object"] = strings.Join(parts[:len(parts)-1], types.Dot)
					result["property"] = parts[len(parts)-1]
				}
			} else {
				result["property"] = typeName
			}
		}
	}

	// 处理数组
	return result
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
			if funcName == types.EmptyString {
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
	if funcName != types.EmptyString {
		call.BaseElement.Name = funcName
	}

	// 设置对象路径作为所有者
	if len(objPath) > 0 {
		call.Owner = strings.Join(objPath, types.Dot)
	}
}

// processArguments 处理JavaScript函数调用的参数
func processArguments(element *Call, argsNode sitter.Node) {
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

		// 创建参数对象
		param := &Parameter{
			Name: "", // 使用参数值作为名称
			Type: nil,
		}

		// 添加参数
		element.Parameters = append(element.Parameters, param)
	}
}

// isRequireCallCapture 检查捕获是否为require函数调用
func isRequireCallCapture(rc *ResolveContext) bool {
	if rc.Match == nil || len(rc.Match.Captures) == 0 {
		return false
	}

	rootCapture := rc.Match.Captures[0]
	if rootCapture.Node.Kind() != string(types.NodeKindCallExpression) {
		return false
	}

	funcNode := rootCapture.Node.ChildByFieldName("function")
	if funcNode == nil {
		return false
	}

	return funcNode.Kind() == string(types.NodeKindIdentifier) && funcNode.Utf8Text(rc.SourceFile.Content) == "require"
}

// handleRequireCall 将require函数调用处理为import
func (js *JavaScriptResolver) handleRequireCall(element *Call, rc *ResolveContext) ([]Element, error) {
	// 创建import元素
	importElement := &Import{
		BaseElement: &BaseElement{
			Type:  types.ElementTypeImport,
			Scope: types.ScopeFile,
			Path:  element.Path,
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
				importElement.Source = strings.Trim(argNode.Utf8Text(rc.SourceFile.Content), "'\"")
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

		if parent.Kind() == string(types.NodeKindVariableDeclarator) {
			// 找到变量声明，获取变量名
			nameNode := parent.ChildByFieldName("name")
			if nameNode != nil {
				importElement.Name = nameNode.Utf8Text(rc.SourceFile.Content)
				importElement.BaseElement.Name = importElement.Name
				break
			}
		}

		currentNode = parent
	}

	// 设置范围
	updateElementRange(importElement, &rootCapture)

	return []Element{importElement}, nil
}

// 检查节点是否为import表达式（适配多种类型）
func isImportExpression(valueNode *sitter.Node, content []byte) bool {
	if valueNode == nil {
		return false
	}

	// 情况1: await import(...)
	if valueNode.Kind() == string(types.NodeKindAwaitExpression) {
		// 尝试使用ChildByFieldName获取call_expression
		callNode := valueNode.ChildByFieldName("expression")
		if callNode == nil {
			// 如果ChildByFieldName失败，尝试遍历所有子节点查找call_expression
			for i := uint(0); i < valueNode.ChildCount(); i++ {
				childNode := valueNode.Child(i)
				if childNode != nil && childNode.Kind() == string(types.NodeKindCallExpression) {
					callNode = childNode
					break
				}
			}
		}

		if callNode != nil {
			funcNode := callNode.ChildByFieldName("function")
			if funcNode != nil && funcNode.Utf8Text(content) == "import" {
				return true
			}
		}
		return false
	}

	// 情况2: 直接import(...)
	if valueNode.Kind() == string(types.NodeKindCallExpression) {
		funcNode := valueNode.ChildByFieldName("function")
		if funcNode != nil && funcNode.Utf8Text(content) == "import" {
			return true
		}
		return false
	}

	// 情况3: 递归检查复杂表达式中的import调用
	// 例如: Promise.resolve().then(() => import(...))
	var findImportCall func(node *sitter.Node) bool
	findImportCall = func(node *sitter.Node) bool {
		if node == nil {
			return false
		}

		// 检查当前节点
		if node.Kind() == string(types.NodeKindCallExpression) {
			funcNode := node.ChildByFieldName("function")
			if funcNode != nil && funcNode.Utf8Text(content) == "import" {
				return true
			}
		}

		// 递归检查所有子节点
		for i := uint(0); i < node.ChildCount(); i++ {
			if findImportCall(node.Child(i)) {
				return true
			}
		}

		return false
	}

	return findImportCall(valueNode)
}

// isRequireImport 检查节点是否为require导入
func isRequireImport(node *sitter.Node, content []byte) bool {
	if node == nil {
		return false
	}
	node = node.ChildByFieldName("function")
	if node == nil {
		return false
	}
	// if node.Kind() == string(types.NodeKindIdentifier) && node.Utf8Text(content) == "require" {
	// 	return true
	// }
	return false
}

// isArrowFunctionImport 检查节点是否为箭头函数导入
func isArrowFunctionImport(node *sitter.Node, content []byte) bool {
	if node == nil {
		return false
	}
	// 1. 获取value字段 (arrow_function)
	valueNode := node.ChildByFieldName("value")
	if valueNode == nil {
		return false
	}
	// 2. 检查是否为箭头函数
	if valueNode.Kind() != string(types.NodeKindArrowFunction) {
		return false
	}
	// 3. 获取body字段 (call_expression)
	bodyNode := valueNode.ChildByFieldName("body")
	if bodyNode == nil {
		return false
	}
	// 4. 获取function字段并检查是否为import
	funcNode := bodyNode.ChildByFieldName("function")
	if funcNode == nil {
		return false
	}
	return funcNode.Utf8Text(content) == "import"
}

// isExportStatement 检查节点是否为export语句
func isExportStatement(node *sitter.Node) bool {
	if node == nil {
		return false
	}
	return node.Kind() == string(types.NodeKindExportStatement)
}
