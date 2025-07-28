package resolver

import (
	"codebase-indexer/pkg/codegraph/types"
	"context"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type TypeScriptResolver struct {
	jsResolver *JavaScriptResolver // 用于复用JavaScript解析器功能
}

var _ ElementResolver = &TypeScriptResolver{}

func (ts *TypeScriptResolver) Resolve(ctx context.Context, element Element, rc *ResolveContext) ([]Element, error) {
	// 初始化JS解析器（如果尚未初始化）
	if ts.jsResolver == nil {
		ts.jsResolver = &JavaScriptResolver{}
	}

	/*
		特殊处理：require调用应该被解析为import，而不是call或variable
		后面提取出来作为公共函数
	*/
	if rc.Match != nil && len(rc.Match.Captures) > 0 {
		rootCapture := rc.Match.Captures[0]
		// 检查是否是call_expression且函数是require
		if rootCapture.Node.Kind() == string(types.NodeKindCallExpression) {
			funcNode := rootCapture.Node.ChildByFieldName("function")
			if funcNode != nil && (funcNode.Kind() == string(types.NodeKindIdentifier) &&
				string(funcNode.Utf8Text(rc.SourceFile.Content)) == "require") {
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
							if argNode != nil && argNode.Kind() == string(types.NodeKindString) {
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
						}
					}

					// 设置范围
					updateElementRange(importElement, &rootCapture)

					return []Element{importElement}, nil
				}
			}
		}
	}
	// 常规解析流程
	return resolve(ctx, ts, element, rc)
}

func (ts *TypeScriptResolver) resolveImport(ctx context.Context, element *Import, rc *ResolveContext) ([]Element, error) {
	elements := []Element{element}

	for _, capture := range rc.Match.Captures {
		if capture.Node.IsMissing() || capture.Node.IsError() {
			continue
		}
		nodeCaptureName := rc.CaptureNames[capture.Index]
		content := capture.Node.Utf8Text(rc.SourceFile.Content)
		switch types.ToElementType(nodeCaptureName) {
		case types.ElementTypeImport:
			element.Type = types.ElementTypeImport
		case types.ElementTypeImportName:
			element.Name = content
			updateElementRange(element, &capture)
		case types.ElementTypeImportAlias:
			element.Alias = content
		case types.ElementTypeImportSource:
			element.Source = strings.Trim(strings.Trim(content, "'"), "\"")
		}
	}
	return elements, nil
}

func (ts *TypeScriptResolver) resolvePackage(ctx context.Context, element *Package, rc *ResolveContext) ([]Element, error) {
	//不支持包
	panic("not support")
}

func (ts *TypeScriptResolver) resolveFunction(ctx context.Context, element *Function, rc *ResolveContext) ([]Element, error) {
	return []Element{element}, nil
}

func (ts *TypeScriptResolver) resolveMethod(ctx context.Context, element *Method, rc *ResolveContext) ([]Element, error) {
	return []Element{element}, nil
}

func (ts *TypeScriptResolver) resolveClass(ctx context.Context, element *Class, rc *ResolveContext) ([]Element, error) {
	elements := []Element{element}
	element.Fields = []*Field{}
	element.Methods = []*Method{}
	element.SuperClasses = []string{}
	rootCapure := rc.Match.Captures[0]
	captureName := rc.CaptureNames[rootCapure.Index]
	updateRootElement(element, &rootCapure, captureName, rc.SourceFile.Content)

	for _, capture := range rc.Match.Captures {
		if capture.Node.IsMissing() || capture.Node.IsError() {
			continue
		}
		nodeCaptureName := rc.CaptureNames[capture.Index]
		content := capture.Node.Utf8Text(rc.SourceFile.Content)
		switch types.ToElementType(nodeCaptureName) {
		case types.ElementTypeClass:
			element.Type = types.ElementTypeClass
			element.Scope = types.ScopeFile
			// 解析类体中的所有成员
		case types.ElementTypeClassName:
			element.BaseElement.Name = content
		case types.ElementTypeClassExtends:
			element.SuperClasses = append(element.SuperClasses, content)
		case types.ElementTypeClassImplements:
			for i := uint(0); i < capture.Node.ChildCount(); i++ {
				child := capture.Node.Child(i)
				if child != nil && child.Kind() == string(types.NodeKindTypeIdentifier) {
					element.SuperInterfaces = append(element.SuperInterfaces, child.Utf8Text(rc.SourceFile.Content))
				}
			}
		}
	}
	cls := parseTypeScriptClassBody(&rootCapure.Node, rc.SourceFile.Content, element.BaseElement.Name)
	element.Fields = cls.Fields
	element.Methods = cls.Methods
	return elements, nil
}

func (ts *TypeScriptResolver) resolveVariable(ctx context.Context, element *Variable, rc *ResolveContext) ([]Element, error) {
	// 初始化JS解析器（如果尚未初始化）
	if ts.jsResolver == nil {
		ts.jsResolver = &JavaScriptResolver{}
	}

	elements := []Element{}
	rootCapture := rc.Match.Captures[0]
	_ = rc.CaptureNames[rootCapture.Index] // 避免未使用警告

	// 首先获取变量名和类型信息
	var variableType string
	// 第一遍扫描，收集类型类型
	for _, capture := range rc.Match.Captures {
		if capture.Node.IsMissing() || capture.Node.IsError() {
			continue
		}
		nodeCaptureName := rc.CaptureNames[capture.Index]
		content := capture.Node.Utf8Text(rc.SourceFile.Content)
		captureType := types.ToElementType(nodeCaptureName)

		switch captureType {
		case types.ElementTypeVariableType:
			// 收集类型信息
			typeStr := string(content)
			// 移除TypeScript类型声明中的冒号前缀
			typeStr = strings.TrimPrefix(typeStr, ":")
			typeStr = strings.TrimSpace(typeStr)
			variableType = typeStr

		}
	}

	// 检查是否为解构赋值
	if rootCapture.Node.Kind() == string(types.NodeKindVariableDeclarator) && isDestructuringPattern(&rootCapture.Node) {
		// 使用JS解析器的handleDestructuringWithPath方法
		basicElems, err := ts.jsResolver.handleDestructuringWithPath(&rootCapture.Node, rc.SourceFile.Content, element.Path)
		if err != nil {
			return nil, err
		}

		// 为解构出的变量添加类型信息
		if len(basicElems) > 0 && variableType != "" {
			// 使用新方法处理解构类型分配
			elements = ts.processDestructuringWithType(basicElems, variableType)
			return elements, nil
		}
		return basicElems, nil
	}

	// 使用一个集合跟踪已处理的节点，避免重复处理
	processedNodes := make(map[uint32]bool)

	for _, capture := range rc.Match.Captures {
		if capture.Node.IsMissing() || capture.Node.IsError() {
			continue
		}

		// 避免重复处理同一个节点
		if processedNodes[capture.Index] {
			continue
		}

		//import函数的处理
		if capture.Node.Kind() == string(types.NodeKindVariableDeclarator) {
			valueNode := capture.Node.ChildByFieldName("value")

			if isImportExpression(valueNode, rc.SourceFile.Content) {
				return []Element{}, nil
			}
		}

		processedNodes[capture.Index] = true

		nodeCaptureName := rc.CaptureNames[capture.Index]
		captureType := types.ToElementType(nodeCaptureName)
		content := capture.Node.Utf8Text(rc.SourceFile.Content)

		if captureType == types.ElementTypeVariable || captureType == types.ElementTypeGlobalVariable {
			// 检查赋值右侧是否为箭头函数
			rightNode := findRightNode(&capture.Node)

			// 检查是否为require函数调用
			if rightNode != nil && isRequireCall(rightNode, rc.SourceFile.Content) {
				// 如果是require调用，跳过变量处理，后续会在resolveCall中处理为import
				return []Element{}, nil
			}
			arrowFunction := isArrowFunction(rightNode)
			if rightNode != nil && arrowFunction != types.EmptyString {
				// 直接创建函数元素
				functionElement := &Function{
					BaseElement: &BaseElement{
						Name:  "",
						Path:  element.Path,
						Scope: types.ScopeFile,
						Type:  types.ElementTypeFunction,
					},
					Declaration: Declaration{
						Name:       "",
						Parameters: []Parameter{},
						ReturnType: []string{},
						Modifier:   arrowFunction,
					},
				}
				// 提取参数和返回类型
				parseTypeScriptArrowFunctionParameters(functionElement, rightNode, rc.SourceFile.Content)
				// 设置范围
				updateElementRange(functionElement, &capture)
				// 直接返回函数元素
				elements = append(elements, functionElement)
			} else if rightNode != nil && isClassOrStructReference(rightNode) {
				// 存储引用路径
				refPathMap := extractReferencePath(rightNode, rc.SourceFile.Content)
				// 处理引用
				refElement := &Reference{
					BaseElement: &BaseElement{
						Name:  refPathMap["property"],
						Path:  element.Path,
						Scope: types.ScopeFile,
						Type:  types.ElementTypeReference,
					},
					Owner: refPathMap["object"],
				}
				// 设置范围
				updateElementRange(refElement, &capture)
				// 返回引用元素
				elements = append(elements, refElement)
			}
		}
		// 如果不是特殊类型，作为普通变量处理
		switch captureType {
		case types.ElementTypeVariable:
			element.Type = types.ElementTypeVariable
			// 根据父节点判断变量作用域
			element.Scope = determineVariableScope(&capture.Node)
		case types.ElementTypeVariableName:
			element.BaseElement.Name = content
			updateElementRange(element, &capture)
		case types.ElementTypeVariableValue:
			// element.Content = []byte(content)
		case types.ElementTypeVariableType:
			// 去掉类型声明中的冒号前缀
			typeContent := strings.TrimPrefix(content, ":")
			typeContent = strings.TrimSpace(typeContent)

			if isTypeScriptPrimitiveType(typeContent) {
				element.VariableType = []string{types.PrimitiveType}
			} else {
				element.VariableType = []string{typeContent}
			}

		}
	}
	elements = append(elements, element)
	return elements, nil
}
func (ts *TypeScriptResolver) resolveInterface(ctx context.Context, element *Interface, rc *ResolveContext) ([]Element, error) {
	elements := []Element{element}
	rootCapture := rc.Match.Captures[0]
	captureName := rc.CaptureNames[rootCapture.Index]
	updateRootElement(element, &rootCapture, captureName, rc.SourceFile.Content)

	// 初始化方法数组和继承接口数组
	element.Methods = []*Declaration{}
	element.SuperInterfaces = []string{}

	for _, capture := range rc.Match.Captures {
		if capture.Node.IsMissing() || capture.Node.IsError() {
			continue
		}
		nodeCaptureName := rc.CaptureNames[capture.Index]
		content := capture.Node.Utf8Text(rc.SourceFile.Content)
		switch types.ToElementType(nodeCaptureName) {
		case types.ElementTypeInterface:
			element.Type = types.ElementTypeInterface
		case types.ElementTypeInterfaceName:
			element.BaseElement.Name = content
		case types.ElementTypeInterfaceExtends:
			// 查找extends_type_clause节点中的所有type子节点
			extendsNode := &capture.Node

			// 遍历所有子节点，查找type标识符
			for i := uint(0); i < extendsNode.ChildCount(); i++ {
				typeNode := extendsNode.Child(i)
				if typeNode != nil && typeNode.Kind() == string(types.NodeKindTypeIdentifier) {
					// 获取接口名称并添加到SuperInterfaces
					interfaceName := typeNode.Utf8Text(rc.SourceFile.Content)
					element.SuperInterfaces = append(element.SuperInterfaces, string(interfaceName))
				}
			}
		}
	}

	// 使用parseJavaScriptClassBody解析接口体，获取方法
	cls := parseTypeScriptClassBody(&rootCapture.Node, rc.SourceFile.Content, element.BaseElement.Name)
	// 将Method转换为Declaration
	for _, method := range cls.Methods {
		element.Methods = append(element.Methods, &method.Declaration)
	}
	return elements, nil
}

func (ts *TypeScriptResolver) resolveCall(ctx context.Context, element *Call, rc *ResolveContext) ([]Element, error) {
	return []Element{element}, nil
}

// 检查TypeScript类型字符串是否包含基本类型
func isTypeScriptPrimitiveType(typeName string) bool {
	// 清理类型名称
	cleanType := strings.TrimPrefix(strings.TrimPrefix(typeName, "[]"), "*")

	// TypeScript基本数据类型列表
	primitiveTypes := []string{
		// 布尔型
		"boolean", "bool", "true", "false",

		// 数值型
		"number", "int", "float", "double", "integer", "bigint", "number",

		// 字符串型
		"string", "char",

		// 特殊类型
		"null", "undefined", "symbol", "void", "never",

		// 通用类型
		"any", "unknown", "object", "Object",

		// 数组与元组
		"Array", "[]", "tuple", "Tuple",

		// 函数类型
		"Function", "=>", "Promise",

		// 内置对象类型
		"Date", "RegExp", "Map", "Set", "WeakMap", "WeakSet",
	}

	// 将输入类型转为小写进行比较
	lowerType := strings.ToLower(cleanType)

	// 检查是否包含任何基本类型名称
	for _, primType := range primitiveTypes {
		if strings.Contains(lowerType, strings.ToLower(primType)) {
			return true
		}
	}

	return false
}

// 从对象类型字符串中提取属性名及其类型
func extractPropertyTypes(typeStr string) map[string]string {
	// 结果映射: 属性名 -> 类型
	result := make(map[string]string)

	// 检查是否是对象类型
	if !strings.Contains(typeStr, ":") || !strings.Contains(typeStr, ";") {
		return result
	}

	// 去掉可能的前后缀
	cleanType := strings.TrimSpace(typeStr)
	cleanType = strings.TrimPrefix(cleanType, "{")
	cleanType = strings.TrimSuffix(cleanType, "}")
	cleanType = strings.TrimSpace(cleanType)

	// 按分号分割各属性定义
	properties := strings.Split(cleanType, ";")
	for _, prop := range properties {
		prop = strings.TrimSpace(prop)
		if prop == "" {
			continue
		}

		// 按冒号分割属性名和类型
		parts := strings.SplitN(prop, ":", 2)
		if len(parts) < 2 {
			continue
		}

		propName := strings.TrimSpace(parts[0])
		propType := strings.TrimSpace(parts[1])

		// 确保只保留有效的属性名和类型
		if propName != "" && propType != "" {
			result[propName] = propType
		}
	}

	return result
}

// 处理TypeScript中解构变量的类型分配
func (ts *TypeScriptResolver) processDestructuringWithType(
	basicElems []Element,
	typeAnnotation string) []Element {

	// 如果类型注解为空或元素为空，直接返回
	if typeAnnotation == "" || len(basicElems) == 0 {
		return basicElems
	}

	// 解析对象类型中的属性类型映射
	propertyTypes := extractPropertyTypes(typeAnnotation)

	// 如果没有解析出属性类型，使用默认处理
	if len(propertyTypes) == 0 {
		// 检查整体类型是否为基本类型
		for _, elem := range basicElems {
			if v, ok := elem.(*Variable); ok {
				if isTypeScriptPrimitiveType(typeAnnotation) {
					v.VariableType = []string{types.PrimitiveType}
				} else {
					v.VariableType = []string{typeAnnotation}
				}
			}
		}
		return basicElems
	}

	// 为每个变量分配其在对象类型中对应的类型
	for _, elem := range basicElems {
		if v, ok := elem.(*Variable); ok {
			varName := v.GetName()

			// 查找该变量名对应的类型
			if propType, exists := propertyTypes[varName]; exists {
				// 检查属性类型是否为基本类型
				if isTypeScriptPrimitiveType(propType) {
					v.VariableType = []string{types.PrimitiveType}
				} else {
					v.VariableType = []string{propType}
				}
			} else {
				// 如果找不到对应属性类型，使用默认类型
				v.VariableType = []string{types.PrimitiveType}
			}
		}
	}

	return basicElems
}

// 解析TypeScript箭头函数参数及其类型
func parseTypeScriptArrowFunctionParameters(functionElement *Function, node *sitter.Node, content []byte) {
	// 查找parameters节点
	parametersNode := node.ChildByFieldName("parameters")
	if parametersNode == nil {
		return
	}

	// 遍历所有命名子节点查找参数
	for i := uint(0); i < parametersNode.NamedChildCount(); i++ {
		paramNode := parametersNode.NamedChild(i)
		if paramNode == nil {
			continue
		}

		// 获取参数类型
		paramName := ""
		paramType := ""
		// 检查参数节点类型
		switch types.ToNodeKind(paramNode.Kind()) {
		case types.NodeKindRequiredParameter:
			// 获取参数名称 - 从pattern子节点提取
			patternNode := paramNode.ChildByFieldName("pattern")
			patternType := paramNode.ChildByFieldName("type")
			if patternNode != nil {
				paramName = string(patternNode.Utf8Text(content))
			}
			if patternType != nil {
				paramType = string(patternType.Utf8Text(content))
			}
		case types.NodeKindIdentifier:
			// 简单参数名称，没有类型注解
			paramName = string(paramNode.Utf8Text(content))
		case types.NodeKindRestParameter:
			// 处理rest参数 (...args)
			patternNode := paramNode.ChildByFieldName("pattern")
			if patternNode != nil {
				paramName = "..." + string(patternNode.Utf8Text(content))
			}
		case types.NodeKindOptionalParameter:
			// 处理可选参数 (name?)
			patternNode := paramNode.ChildByFieldName("pattern")
			if patternNode != nil {
				paramName = string(patternNode.Utf8Text(content)) + "?"
			}
		}

		// 添加参数到函数定义
		if paramName != "" {
			param := Parameter{
				Name: paramName,
				Type: nil,
			}
			if isTypeScriptPrimitiveType(paramType) {
				param.Type = []string{types.PrimitiveType}
			} else {
				param.Type = []string{paramType}
			}
			// 添加到函数参数列表
			functionElement.Declaration.Parameters = append(functionElement.Declaration.Parameters, param)
		}
	}

	// 查找返回类型注解
	ReturnNode := node.ChildByFieldName("return_type")
	if ReturnNode != nil {
		content := string(ReturnNode.Utf8Text(content))
		// 去掉冒号和前导空格
		content = strings.TrimSpace(content)
		if strings.HasPrefix(content, ":") {
			content = strings.TrimSpace(content[1:]) // 去掉冒号后再 Trim
		}

		if isTypeScriptPrimitiveType(content) {
			functionElement.ReturnType = []string{types.PrimitiveType}
		} else {
			functionElement.ReturnType = []string{content}
		}
	}
}

// 检查节点是否为import表达式（适配多种类型）
func isImportExpression(valueNode *sitter.Node, content []byte) bool {
	if valueNode == nil {
		return false
	}

	// 情况1: await import(...)
	if valueNode.Kind() == "await_expression" {
		// 尝试使用ChildByFieldName获取call_expression
		callNode := valueNode.ChildByFieldName("expression")
		if callNode == nil {
			// 如果ChildByFieldName失败，尝试遍历所有子节点查找call_expression
			for i := uint(0); i < valueNode.ChildCount(); i++ {
				childNode := valueNode.Child(i)
				if childNode != nil && childNode.Kind() == "call_expression" {
					callNode = childNode
					break
				}
			}
		}

		if callNode != nil {
			funcNode := callNode.ChildByFieldName("function")
			if funcNode != nil && string(funcNode.Utf8Text(content)) == "import" {
				return true
			}
		}
		return false
	}

	// 情况2: 直接import(...)
	if valueNode.Kind() == "call_expression" {
		funcNode := valueNode.ChildByFieldName("function")
		if funcNode != nil && string(funcNode.Utf8Text(content)) == "import" {
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
		if node.Kind() == "call_expression" {
			funcNode := node.ChildByFieldName("function")
			if funcNode != nil && string(funcNode.Utf8Text(content)) == "import" {
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

// parseTypeScriptClassBody 解析TypeScript类体，提取字段和方法
func parseTypeScriptClassBody(node *sitter.Node, content []byte, className string) *Class {
	class := &Class{
		BaseElement: &BaseElement{
			Name:  className,
			Scope: types.ScopeFile,
			Type:  types.ElementTypeClass,
		},
		Methods: []*Method{},
		Fields:  []*Field{},
	}
	// 查找class_body节点
	var classBodyNode *sitter.Node
	// 类声明节点的最后一个子节点通常是类体
	classBodyNode = node.Child(node.ChildCount() - 1)
	if classBodyNode == nil {
		return class
	}
	// 遍历类体中的所有成员
	for i := uint(0); i < classBodyNode.ChildCount(); i++ {
		memberNode := classBodyNode.Child(i)
		if memberNode == nil {
			continue
		}

		kind := memberNode.Kind()
		switch types.ToNodeKind(kind) {
		case types.NodeKindMethodDefinition, types.NodeKindMethodSignature:
			method := parseTypeScriptMethodNode(memberNode, content, class.BaseElement.Name)
			method.Owner = className
			if method != nil {
				class.Methods = append(class.Methods, method)
			}
		case types.NodeKindPublicFieldDefinition:
			field := parseTypeScriptFieldNode(memberNode, content)
			if field != nil {
				class.Fields = append(class.Fields, field)
			}
		}
	}
	return class
}

// parseTypeScriptMethodNode 解析TypeScript方法节点
func parseTypeScriptMethodNode(node *sitter.Node, content []byte, className string) *Method {
	method := &Method{}
	method.Owner = className

	// 设置默认作用域和修饰符
	method.BaseElement = &BaseElement{
		Scope: types.ScopeFile,
	}
	modifierNode := node.ChildByFieldName("accessibility_modifier")
	if modifierNode != nil {
		method.Declaration.Modifier = modifierNode.Utf8Text(content)
	} else {
		method.Declaration.Modifier = types.ModifierPublic
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
		method.Parameters = make([]Parameter, 0)
		for j := uint(0); j < paramsNode.ChildCount(); j++ {
			paramChild := paramsNode.Child(j)
			patternNode := paramChild.ChildByFieldName("pattern")
			patternType := paramChild.ChildByFieldName("type")
			if patternNode == nil || patternType == nil {
				continue
			}
			if patternType != nil {
				typeContent := strings.TrimPrefix(string(patternType.Utf8Text(content)), ":")
				typeContent = strings.TrimSpace(typeContent)
				if isTypeScriptPrimitiveType(typeContent) {
					method.Parameters = append(method.Parameters, Parameter{
						Name: string(patternNode.Utf8Text(content)),
						Type: []string{types.PrimitiveType},
					})
				} else {
					method.Parameters = append(method.Parameters, Parameter{
						Name: string(patternNode.Utf8Text(content)),
						Type: []string{typeContent},
					})
				}
			} else {
				// 没有类型注解，添加无类型参数
				method.Parameters = append(method.Parameters, Parameter{
					Name: string(patternNode.Utf8Text(content)),
					Type: []string{},
				})
			}
		}
	}
	// 查找返回类型
	returnNode := node.ChildByFieldName("return_type")
	if returnNode != nil {
		returnContent := string(returnNode.Utf8Text(content))
		if isTypeScriptPrimitiveType(returnContent) {
			method.ReturnType = []string{types.PrimitiveType}
		} else {
			method.ReturnType = []string{returnContent}
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
func parseTypeScriptFieldNode(node *sitter.Node, content []byte) *Field {
	field := &Field{}
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		switch types.ToNodeKind(child.Kind()) {
		case types.NodeKindAccessibilityModifier:
			// 处理访问修饰符 (public, private, protected)
			field.Modifier = child.Utf8Text(content)
		case types.NodeKindPropertyIdentifier:
			// 处理普通属性标识符
			field.Name = child.Utf8Text(content)
		case types.NodeKindPrivatePropertyIdentifier:
			// 处理私有属性标识符 (#name)
			fieldName := child.Utf8Text(content)
			// 如果已经有private修饰符，不需要再处理#前缀
			if field.Modifier != types.ModifierPrivate {
				// 移除#前缀并设置为私有
				field.Name = strings.TrimPrefix(fieldName, "#")
				field.Modifier = types.ModifierPrivate
			} else {
				field.Name = fieldName
			}
		case types.NodeKindTypeAnnotation:
			typeText := child.Utf8Text(content)
			// 去掉类型声明中的冒号前缀
			typeText = strings.TrimPrefix(typeText, ":")
			typeText = strings.TrimSpace(typeText)
			// 判断是否为基本类型
			if isTypeScriptPrimitiveType(typeText) {
				field.Type = types.PrimitiveType
			} else {
				field.Type = typeText
			}
		}
	}

	return field
}
