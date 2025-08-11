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
	return resolve(ctx, ts, element, rc)
}

func (ts *TypeScriptResolver) resolveImport(ctx context.Context, element *Import, rc *ResolveContext) ([]Element, error) {
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
		case types.ElementTypeImport:
			element.Type = types.ElementTypeImport
		case types.ElementTypeImportName:
			element.Name = content
		case types.ElementTypeImportAlias:
			element.Alias = content
		case types.ElementTypeImportSource:
			element.Source = strings.Trim(strings.Trim(content, types.SingleQuote), types.DoubleQuote)
		}
	}
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

func (ts *TypeScriptResolver) resolvePackage(ctx context.Context, element *Package, rc *ResolveContext) ([]Element, error) {
	//不支持包
	panic("not support")
}

func (ts *TypeScriptResolver) resolveFunction(ctx context.Context, element *Function, rc *ResolveContext) ([]Element, error) {
	elements := []Element{element}
	rootCapture := rc.Match.Captures[0]
	updateRootElement(element, &rootCapture, rc.CaptureNames[rootCapture.Index], rc.SourceFile.Content)
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
			parseTypeScriptParameters(element, capture.Node, rc.SourceFile.Content)
		case types.ElementTypeFunctionReturnType:
			returnTypes := parseReturnTypeNode(&capture.Node, rc.SourceFile.Content)
			element.ReturnType = returnTypes
		}
	}
	return elements, nil
}

func (ts *TypeScriptResolver) resolveMethod(ctx context.Context, element *Method, rc *ResolveContext) ([]Element, error) {
	elements := []Element{element}
	rootCap := rc.Match.Captures[0]
	updateRootElement(element, &rootCap, rc.CaptureNames[rootCap.Index], rc.SourceFile.Content)
	for _, capture := range rc.Match.Captures {
		if capture.Node.IsMissing() || capture.Node.IsError() {
			continue
		}
		nodeCaptureName := rc.CaptureNames[capture.Index]
		content := capture.Node.Utf8Text(rc.SourceFile.Content)
		switch types.ToElementType(nodeCaptureName) {
		case types.ElementTypeMethod:
			element.Declaration.Modifier = extractModifiers(content)
		case types.ElementTypeMethodName:
			element.BaseElement.Name = content
			element.Declaration.Name = content
		case types.ElementTypeMethodParameters:
			parseTypeScriptMethodParameters(element, capture.Node, rc.SourceFile.Content)
		case types.ElementTypeMethodReturnType:
			returnTypes := parseReturnTypeNode(&capture.Node, rc.SourceFile.Content)
			element.ReturnType = returnTypes
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
		case types.ElementTypeClass, types.ElementTypeEnum, types.ElementTypeNamespace:
			if isExportStatement(&capture.Node) {
				element.Scope = types.ScopePackage
			} else {
				element.Scope = types.ScopeFile
			}
		case types.ElementTypeClassName, types.ElementTypeEnumName, types.ElementTypeNamespaceName:
			element.BaseElement.Name = CleanParam(content)
		case types.ElementTypeClassExtends:
			element.SuperClasses = append(element.SuperClasses, content)
		case types.ElementTypeClassImplements:
			for i := uint(0); i < capture.Node.ChildCount(); i++ {
				child := capture.Node.Child(i)
				if child != nil && child.Kind() == string(types.NodeKindTypeIdentifier) {
					element.SuperInterfaces = append(element.SuperInterfaces, child.Utf8Text(rc.SourceFile.Content))
				}
			} //枚举字段
		}
	}
	cls, references := parseTypeScriptClassBody(&rootCapure.Node, rc.SourceFile.Content, element.BaseElement.Name, element.Path)
	element.Fields = cls.Fields
	element.Methods = cls.Methods

	// 收集所有引用元素
	for _, ref := range references {
		elements = append(elements, ref)
	}
	return elements, nil
}

func (ts *TypeScriptResolver) resolveVariable(ctx context.Context, element *Variable, rc *ResolveContext) ([]Element, error) {
	if ts.jsResolver == nil {
		ts.jsResolver = &JavaScriptResolver{}
	}

	elements := []Element{}
	rootCapture := rc.Match.Captures[0]
	rootCaptureName := rc.CaptureNames[rootCapture.Index] // 避免未使用警告
	updateRootElement(element, &rootCapture, rootCaptureName, rc.SourceFile.Content)
	// 首先获取变量名和类型信息
	var variableType string
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
			typeStr = strings.TrimPrefix(typeStr, types.Colon)
			typeStr = strings.TrimSpace(typeStr)
			variableType = typeStr

		}
	}
	// 检查是否为解构赋值
	if rootCapture.Node.Kind() == string(types.NodeKindVariableDeclarator) && isDestructuringPattern(&rootCapture.Node) {
		// 使用JS解析器的handleDestructuringWithPath方法
		basicElems, err := ts.jsResolver.handleDestructuringWithPath(&rootCapture.Node, rc.SourceFile.Content, element)
		if err != nil {
			return nil, err
		}
		// 为解构出的变量添加类型信息
		if len(basicElems) > 0 && variableType != types.EmptyString {
			// 使用新方法处理解构类型分配
			elements = ts.processDestructuringWithType(basicElems, variableType)
			return elements, nil
		}
		return basicElems, nil
	}
	if rc.Match != nil && len(rc.Match.Captures) > 0 {
		if rightNode := findRightNode(&rootCapture.Node); rightNode != nil {
			if isRequireImport(rightNode, rc.SourceFile.Content) {
				return []Element{}, nil
			}
			if rightNode.Kind() == string(types.NodeKindArrowFunction) {
				return []Element{}, nil
			}
		}
	}
	for _, capture := range rc.Match.Captures {
		//import函数的处理
		if capture.Node.Kind() == string(types.NodeKindVariableDeclarator) {
			valueNode := capture.Node.ChildByFieldName("value")

			if isImportExpression(valueNode, rc.SourceFile.Content) {
				return []Element{}, nil
			}
		}
		nodeCaptureName := rc.CaptureNames[capture.Index]
		captureType := types.ToElementType(nodeCaptureName)
		content := capture.Node.Utf8Text(rc.SourceFile.Content)
		// 如果不是特殊类型，作为普通变量处理
		switch captureType {
		case types.ElementTypeVariable:
			element.Type = types.ElementTypeVariable
			element.Scope = determineVariableScope(&capture.Node)
		case types.ElementTypeVariableName:
			element.BaseElement.Name = content
			updateElementRange(element, &capture)
		case types.ElementTypeVariableType:
			typeContent := strings.TrimPrefix(content, types.Colon)
			typeContent = strings.TrimSpace(typeContent)
			if isTypeScriptPrimitiveType(typeContent) {
				element.VariableType = []string{types.PrimitiveType}
			} else if isValidReferenceName(typeContent) && isValidReferenceName(typeContent) {
				element.VariableType = []string{typeContent}
				ref := NewReference(element, &capture.Node, typeContent, "")
				// 处理带点号的类型名称（如 typescript.go）
				if strings.Contains(typeContent, types.Dot) {
					parts := strings.Split(typeContent, types.Dot)
					ref.BaseElement.Name = parts[len(parts)-1]
					ref.Owner = parts[len(parts)-2]
				}
				elements = append(elements, ref)
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
			if isExportStatement(&capture.Node) {
				element.Scope = types.ScopePackage
			} else {
				element.Scope = types.ScopeFile
			}
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

	// 使用parseTypeScriptClassBody解析接口体，获取方法
	cls, references := parseTypeScriptClassBody(&rootCapture.Node, rc.SourceFile.Content, element.BaseElement.Name, element.Path)
	for _, method := range cls.Methods {
		element.Methods = append(element.Methods, &method.Declaration)
		elements = append(elements, method)
	}
	for _, ref := range references {
		elements = append(elements, ref)
	}
	return elements, nil
}

func (ts *TypeScriptResolver) resolveCall(ctx context.Context, element *Call, rc *ResolveContext) ([]Element, error) {
	// 处理为import而不是call
	// if isRequireCallCapture(rc) {
	// 	return ts.jsResolver.handleRequireCall(rc)
	// }
	elements := []Element{element}
	rootCapture := rc.Match.Captures[0]
	updateRootElement(element, &rootCapture, rc.CaptureNames[rootCapture.Index], rc.SourceFile.Content)
	for _, capture := range rc.Match.Captures {
		nodeCaptureName := rc.CaptureNames[capture.Index]
		switch types.ToElementType(nodeCaptureName) {
		case types.ElementTypeFunctionCall, types.ElementTypeMethodCall:
			// 处理整个函数调用表达式
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

var primitiveTypesMap = map[string]struct{}{
	// 布尔型
	"boolean": {}, "bool": {},
	// 数值型
	"number": {}, "int": {}, "float": {}, "double": {}, "integer": {}, "bigint": {},
	// 字符串型
	"string": {}, "char": {},
	// 特殊类型
	"null": {}, "undefined": {}, "symbol": {}, "void": {}, "never": {},
	// 通用类型
	"any": {}, "unknown": {}, "object": {}, "Object": {},
	// 数组与元组
	"Array": {}, "[]": {}, "tuple": {}, "Tuple": {},
	// 函数类型
	"Function": {}, "Promise": {},
	// 内置对象类型
	"Date": {}, "RegExp": {}, "Map": {}, "Set": {}, "WeakMap": {}, "WeakSet": {},
}

// 检查TypeScript类型字符串是否包含基本类型
func isTypeScriptPrimitiveType(typeName string) bool {
	// 清理类型名称
	cleanType := strings.TrimPrefix(strings.TrimPrefix(typeName, "[]"), "*")
	// 将输入类型转为小写进行比较
	lowerType := strings.ToLower(cleanType)

	// 检查是否包含任何基本类型名称
	for primType := range primitiveTypesMap {
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
		parts := strings.SplitN(prop, types.Colon, 2)
		if len(parts) < 2 {
			continue
		}

		propName := strings.TrimSpace(parts[0])
		propType := strings.TrimSpace(parts[1])

		// 确保只保留有效的属性名和类型
		if propName != types.EmptyString && propType != types.EmptyString {
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
	if typeAnnotation == types.EmptyString || len(basicElems) == 0 {
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

// parseTypeScriptClassBody 解析TypeScript类体，提取字段和方法
func parseTypeScriptClassBody(node *sitter.Node, content []byte, className string, path string) (*Class, []Element) {
	class := &Class{
		BaseElement: &BaseElement{
			Name:  className,
			Scope: types.ScopeFile,
			Type:  types.ElementTypeClass,
			Path:  path,
		},
		Methods: []*Method{},
		Fields:  []*Field{},
	}
	var references []Element
	var classBodyNode *sitter.Node
	classBodyNode = node.Child(node.ChildCount() - 1)
	if classBodyNode == nil {
		return class, references
	}
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
			method.Path = path
			method.Range = []int32{
				int32(memberNode.StartPosition().Row),
				int32(memberNode.StartPosition().Column),
				int32(memberNode.EndPosition().Row),
				int32(memberNode.EndPosition().Column),
			}
			if method != nil {
				class.Methods = append(class.Methods, method)
			}
		//解析类属性使用
		case types.NodeKindPublicFieldDefinition:
			field, ref := parseTypeScriptFieldNode(memberNode, content)
			if field != nil {
				class.Fields = append(class.Fields, field)
				if ref != nil {
					ref.Path = path
					references = append(references, ref)
				}
			}
		//解析接口方法属性使用
		case types.NodeKindPropertySignature:
			field, ref, method := parseTypeScriptPropertySignatureNode(memberNode, content)
			if field != nil {
				class.Fields = append(class.Fields, field)
				if ref != nil {
					ref.Path = path
					references = append(references, ref)
				}
				if method != nil {
					method.Path = path
					method.Owner = className
					class.Methods = append(class.Methods, method)
				}
			}
		}
	}
	return class, references
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
		methodName := CleanParam(nameNode.Utf8Text(content))
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
			parameter := Parameter{
				Name: patternNode.Utf8Text(content),
				Type: []string{},
			}
			typeContent := strings.TrimPrefix(patternType.Utf8Text(content), types.Colon)
			typeContent = strings.TrimSpace(typeContent)
			if isTypeScriptPrimitiveType(typeContent) {
				parameter.Type = []string{types.PrimitiveType}
			} else {
				parameter.Type = []string{typeContent}
			}
			method.Parameters = append(method.Parameters, parameter)
		}
	}
	// 查找返回类型
	returnNode := node.ChildByFieldName("return_type")
	if returnNode != nil {
		returnContent := returnNode.Utf8Text(content)
		if isTypeScriptPrimitiveType(returnContent) {
			method.ReturnType = []string{types.PrimitiveType}
		} else {
			method.ReturnType = []string{returnContent}
		}
	}
	method.Type = types.ElementTypeMethod
	return method
}
func parseTypeScriptFieldNode(node *sitter.Node, content []byte) (*Field, *Reference) {
	field := &Field{}
	var ref *Reference
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		switch types.ToNodeKind(child.Kind()) {
		case types.NodeKindAccessibilityModifier:
			field.Modifier = child.Utf8Text(content)
		case types.NodeKindPropertyIdentifier:
			field.Name = child.Utf8Text(content)
		case types.NodeKindPrivatePropertyIdentifier:
			fieldName := child.Utf8Text(content)
			if field.Modifier != types.ModifierPrivate {
				// 移除#前缀并设置为私有
				field.Name = strings.TrimPrefix(fieldName, types.Hash)
				field.Modifier = types.ModifierPrivate
			} else {
				field.Name = fieldName
			}
		case types.NodeKindTypeAnnotation:
			typeText := child.Utf8Text(content)
			typeText = strings.TrimPrefix(typeText, types.Colon)
			typeText = strings.TrimSpace(typeText)
			typeText = strings.TrimPrefix(typeText, types.EmailAt)
			field.Type = typeText

			if !isTypeScriptPrimitiveType(typeText) && isValidReferenceName(typeText) {
				ref = &Reference{
					BaseElement: &BaseElement{
						Name: typeText,
						Type: types.ElementTypeReference,
						Range: []int32{
							int32(child.StartPosition().Row),
							int32(child.StartPosition().Column),
							int32(child.EndPosition().Row),
							int32(child.EndPosition().Column),
						},
						Scope: types.ScopeBlock,
					},
				}
				if strings.Contains(typeText, types.Dot) {
					parts := strings.Split(typeText, types.Dot)
					ref.BaseElement.Name = parts[len(parts)-1]
					ref.Owner = parts[len(parts)-2]
				}
			}
		}
	}
	return field, ref
}
func parseTypeScriptPropertySignatureNode(node *sitter.Node, content []byte) (*Field, *Reference, *Method) {
	field := &Field{}
	var ref *Reference
	var method *Method

	nameNode := node.ChildByFieldName("name")
	if nameNode != nil {
		field.Name = CleanParam(nameNode.Utf8Text(content))
	}

	typeNode := node.ChildByFieldName("type")
	if typeNode == nil {
		return field, ref, method
	}

	typeContent := strings.TrimPrefix(typeNode.Utf8Text(content), types.Colon)
	typeContent = strings.TrimSpace(typeContent)
	if isTypeScriptPrimitiveType(typeContent) {
		field.Type = types.PrimitiveType
	} else {
		field.Type = typeContent
	}

	if typeNode.Kind() == string(types.NodeKindTypeAnnotation) {
		for i := uint(0); i < typeNode.ChildCount(); i++ {
			child := typeNode.Child(i)
			if child == nil {
				continue
			}
			switch types.ToNodeKind(child.Kind()) {
			case types.NodeKindTypeIdentifier:
				typeText := parseReturnTypeNode(child, content)[0]
				field.Type = typeText
				if typeText != types.PrimitiveType {
					ref = &Reference{
						BaseElement: &BaseElement{
							Name: typeText,
							Type: types.ElementTypeReference,
							Range: []int32{
								int32(child.StartPosition().Row),
								int32(child.StartPosition().Column),
								int32(child.EndPosition().Row),
								int32(child.EndPosition().Column),
							},
							Scope: types.ScopeBlock,
						},
					}
					if strings.Contains(typeText, types.Dot) {
						parts := strings.SplitN(typeText, types.Dot, 2)
						ref.BaseElement.Name = parts[1]
						ref.Owner = parts[0]
					}
				}
			case types.NodeKindFunctionType:
				method = &Method{
					BaseElement: &BaseElement{
						Name:  field.Name, // Use the field name we already parsed
						Type:  types.ElementTypeMethod,
						Scope: types.ScopeBlock,
						Range: []int32{
							int32(child.StartPosition().Row),
							int32(child.StartPosition().Column),
							int32(child.EndPosition().Row),
							int32(child.EndPosition().Column),
						},
					},
					Declaration: Declaration{
						Name: field.Name,
					},
				}
				paramsNode := child.ChildByFieldName("parameters")
				if paramsNode != nil {
					method.Parameters = parseTypeScriptParamNodes(*paramsNode, content)
				}
				returnTypeNode := child.ChildByFieldName("return_type")
				if returnTypeNode != nil {
					method.ReturnType = parseReturnTypeNode(returnTypeNode, content)
				}
			}
		}
	}

	return field, ref, method
}

// parseTypeScriptParamNodes 解析TypeScript参数节点，返回参数数组
func parseTypeScriptParamNodes(paramsNode sitter.Node, content []byte) []Parameter {
	params := make([]Parameter, 0)

	for i := uint(0); i < paramsNode.ChildCount(); i++ {
		paramNode := paramsNode.Child(i)
		if paramNode == nil || isNodeDelimiter(paramNode) {
			continue
		}
		// 根据参数节点类型进行处理
		switch types.ToNodeKind(paramNode.Kind()) {
		case types.NodeKindOptionalParameter:
			param := parseOptionalParameterNode(paramNode, content)
			if param.Name != types.EmptyString {
				params = append(params, param)
			}
		case types.NodeKindRequiredParameter:
			param := parseRequiredParameterNode(paramNode, content)
			if param.Name != types.EmptyString {
				params = append(params, param)
			}
		case types.NodeKindTypeIdentifier:
			params = append(params, Parameter{
				Name: paramNode.Utf8Text(content),
				Type: []string{paramNode.Utf8Text(content)},
			})
		default:
			// 其他类型的参数节点，尝试作为普通参数处理
			if paramNode.Kind() == string(types.NodeKindIdentifier) {
				// 简单标识符参数
				paramName := paramNode.Utf8Text(content)
				params = append(params, Parameter{
					Name: paramName,
					Type: nil,
				})
			}
		}
	}

	return params
}

// parseOptionalParameterNode 解析可选参数节点
func parseOptionalParameterNode(paramNode *sitter.Node, content []byte) Parameter {
	var paramName string
	var paramType []string

	patternNode := paramNode.ChildByFieldName("pattern")
	if patternNode != nil {
		paramName = patternNode.Utf8Text(content)
	}

	// 获取参数类型
	typeNode := paramNode.ChildByFieldName("type")
	if typeNode != nil {
		paramType = parseReturnTypeNode(typeNode, content)
	}

	return Parameter{
		Name: paramName,
		Type: paramType,
	}
}

// parseRequiredParameterNode 解析必需参数节点
func parseRequiredParameterNode(paramNode *sitter.Node, content []byte) Parameter {
	var paramName string
	var paramType []string

	// 获取参数名称 (可能是标识符或rest_pattern)
	patternNode := paramNode.ChildByFieldName("pattern")
	if patternNode != nil {
		if types.ToNodeKind(patternNode.Kind()) == types.NodeKindRestPattern {
			// 处理剩余参数 (...args)
			restIdNode := patternNode.Child(1)
			if restIdNode == nil {
				// 尝试获取第一个子节点作为标识符
				for j := uint(0); j < patternNode.ChildCount(); j++ {
					child := patternNode.Child(j)
					if child != nil && child.Kind() == types.Identifier {
						restIdNode = child
						break
					}
				}
			}
			if restIdNode != nil {
				paramName = restIdNode.Utf8Text(content)
			} else {
				// 如果找不到标识符节点，使用整个模式节点的文本
				paramName = patternNode.Utf8Text(content)
			}
		} else {
			// 普通参数
			paramName = patternNode.Utf8Text(content)
		}
	}

	// 获取参数类型
	typeNode := paramNode.ChildByFieldName("type")
	if typeNode != nil {
		paramType = parseReturnTypeNode(typeNode, content)
	}

	return Parameter{
		Name: paramName,
		Type: paramType,
	}
}

// parseTypeScriptParameters 解析TypeScript函数参数
func parseTypeScriptParameters(element *Function, paramsNode sitter.Node, content []byte) {
	element.Parameters = parseTypeScriptParamNodes(paramsNode, content)
}

// parseTypeScriptMethodParameters 解析TypeScript方法参数
func parseTypeScriptMethodParameters(element *Method, paramsNode sitter.Node, content []byte) {
	element.Parameters = parseTypeScriptParamNodes(paramsNode, content)
}

// isNodeDelimiter 检查节点是否为分隔符
func isNodeDelimiter(node *sitter.Node) bool {
	kind := node.Kind()
	return kind == "," || kind == "(" || kind == ")" || kind == "{" || kind == "}"
}

// parseReturnTypeNode 解析类型节点
func parseReturnTypeNode(node *sitter.Node, content []byte) []string {
	// 获取类型文本
	typeText := node.Utf8Text(content)
	typeText = strings.TrimPrefix(typeText, types.Colon)
	typeText = strings.TrimSpace(typeText)
	typeText = strings.TrimPrefix(typeText, types.EmailAt)
	if isTypeScriptPrimitiveType(typeText) {
		return []string{types.PrimitiveType}
	}

	return []string{typeText}
}

// 判断reference是否合法的name
func isValidReferenceName(name string) bool {
	// 空字符串不合法
	if name == "" || strings.TrimSpace(name) == "" {
		return false
	}

	// 包含空格不合法
	if strings.Contains(name, " ") || strings.Contains(name, "\t") || strings.Contains(name, "\n") {
		return false
	}

	// 包含特殊符号不合法（保留点号和下划线，因为它们在标识符中是合法的）
	invalidChars := []string{
		"(", ")", "[", "]", "{", "}",
		"<", ">", "=", "+", "-", "*", "/", "%",
		"!", "@", "#", "$", "^", "&", "|", "\\",
		"?", ":", ";", ",", "'", "\"", "`",
		"~",
	}

	for _, char := range invalidChars {
		if strings.Contains(name, char) {
			return false
		}
	}

	// 不能全部是数字
	if strings.TrimSpace(name) != "" {
		allDigits := true
		for _, r := range name {
			if !('0' <= r && r <= '9') {
				allDigits = false
				break
			}
		}
		if allDigits {
			return false
		}
	}

	// 不能以数字开头（标识符规则）
	if len(name) > 0 && '0' <= name[0] && name[0] <= '9' {
		return false
	}

	return true
}
