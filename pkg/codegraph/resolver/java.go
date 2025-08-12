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
			element.BaseElement.Name = StripSpaces(content)
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
			continue
		}
		content := cap.Node.Utf8Text(rc.SourceFile.Content)
		switch types.ToElementType(captureName) {
		case types.ElementTypePackageName:
			element.BaseElement.Name = StripSpaces(content)
		}
	}
	element.BaseElement.Scope = types.ScopeProject
	return []Element{element}, nil
}

func (j *JavaResolver) resolveFunction(ctx context.Context, element *Function, rc *ResolveContext) ([]Element, error) {
	// TODO java中不存在单独的函数，暂时不实现
	return nil, fmt.Errorf("java function not supported")
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
			element.BaseElement.Name = StripSpaces(content)
			element.Declaration.Name = element.BaseElement.Name
		case types.ElementTypeMethodReturnType:
			element.Declaration.ReturnType = findAllTypes(&cap.Node, rc.SourceFile.Content)
		case types.ElementTypeMethodParameters:
			element.Declaration.Parameters = ParseParameterList(&cap.Node, rc.SourceFile.Content)
		}
	}
	// 设置owner并且补充默认修饰符
	ownerNode := findMethodOwner(&rootCap.Node)
	var ownerKind types.NodeKind
	if ownerNode != nil {
		owner := extractNodeName(ownerNode, rc.SourceFile.Content)
		element.Owner = StripSpaces(owner)
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
			element.BaseElement.Name = StripSpaces(content)
		case types.ElementTypeClassExtends, types.ElementTypeClassImplements, types.ElementTypeEnumImplements:
			// 枚举的多继承、类的单继承、类的多实现
			// TODO 这里可能存在多个类型，需要额外处理
			typs := ParseSuperTypes(&cap.Node, rc.SourceFile.Content)
			for _, typ := range typs {
				parts := strings.Split(typ, types.Dot)
				// owner 可能是包名，也可能是嵌套类的上层类调用
				owner := StripSpaces(strings.Join(parts[:len(parts)-1], types.Dot))
				parent := StripSpaces(parts[len(parts)-1])
				if owner != types.EmptyString {
					imports = append(imports, &Import{
						BaseElement: &BaseElement{
							Name:  owner,
							Path:  element.BaseElement.Path,
							Scope: types.ScopePackage,
							Type:  types.ElementTypeImport,
							Range: element.BaseElement.Range,
						},
					})
				}

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
	var imports = []*Import{}
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
					Name:  StripSpaces(content), // 暂时用于填充字段
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
			elems[len(elems)-1].BaseElement.Name = StripSpaces(content)
		case types.ElementTypeLocalVariableType, types.ElementTypeFieldType:
			// 左侧的类型声明
			typs := findAllTypes(&cap.Node, rc.SourceFile.Content)
			elems[len(elems)-1].VariableType = typs
			for _, typ := range typs {
				// 得到owner，用点进行分割，取最后一个
				parts := strings.Split(typ, types.Dot)
				owner := StripSpaces(strings.Join(parts[:len(parts)-1], types.Dot))
				realTyp := StripSpaces(parts[len(parts)-1])
				if owner != types.EmptyString {
					// 包名抛一个import
					imports = append(imports, &Import{
						BaseElement: &BaseElement{
							Name:  owner,
							Path:  element.BaseElement.Path,
							Scope: types.ScopePackage,
							Type:  types.ElementTypeImport,
							Range: element.BaseElement.Range,
						},
					})
				}
				// 自定义类型走引用
				refs = append(refs, NewReference(element, &cap.Node, realTyp, owner))

			}
		}
	}
	var elements []Element
	for _, e := range elems {
		elements = append(elements, e)
	}
	for _, r := range refs {
		elements = append(elements, r)
	}
	for _, i := range imports {
		elements = append(elements, i)
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
			element.BaseElement.Name = StripSpaces(content)
		case types.ElementTypeInterfaceModifiers:
			modifier = getElementModifier(content)
		case types.ElementTypeInterfaceExtends:
			// TODO 这里可能存在多个类型，需要额外处理
			typs := ParseSuperTypes(&cap.Node, rc.SourceFile.Content)
			for _, typ := range typs {
				parts := strings.Split(typ, types.Dot)
				// owner 可能是包名，也可能是嵌套类的上层类调用
				owner := StripSpaces(strings.Join(parts[:len(parts)-1], types.Dot))
				parent := StripSpaces(parts[len(parts)-1])
				if owner != types.EmptyString {
					imports = append(imports, &Import{
						BaseElement: &BaseElement{
							Name:  owner,
							Path:  element.BaseElement.Path,
							Scope: types.ScopePackage,
							Type:  types.ElementTypeImport,
							Range: element.BaseElement.Range,
						},
					})
				}
				element.SuperInterfaces = append(element.SuperInterfaces, parent)
				refs = append(refs, NewReference(element, &cap.Node, parent, owner))
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
	rootCap := rc.Match.Captures[0]
	updateRootElement(element, &rootCap, rc.CaptureNames[rootCap.Index], rc.SourceFile.Content)
	var refs []*Reference
	var imports []*Import
	for _, cap := range rc.Match.Captures {
		captureName := rc.CaptureNames[cap.Index]
		if cap.Node.IsMissing() || cap.Node.IsError() {
			continue
		}
		content := cap.Node.Utf8Text(rc.SourceFile.Content)
		switch types.ToElementType(captureName) {
		case types.ElementTypeCallName:
			element.BaseElement.Name = StripSpaces(content)
		case types.ElementTypeClassLiteralType, types.ElementTypeCastExpressionType, types.ElementTypeInstanceofExpressionType,
			types.ElementTypeArrayCreationType, types.ElementTypeNewExpressionType:
			typs := findAllTypes(&cap.Node, rc.SourceFile.Content)
			for i, typ := range typs {
				parts := strings.Split(typ, types.Dot)
				owner := StripSpaces(strings.Join(parts[:len(parts)-1], types.Dot))
				realTyp := StripSpaces(parts[len(parts)-1])
				if owner != types.EmptyString {
					imports = append(imports, &Import{
						BaseElement: &BaseElement{
							Name:  owner,
							Path:  element.BaseElement.Path,
							Scope: types.ScopePackage,
							Type:  types.ElementTypeImport,
							Range: element.BaseElement.Range,
						},
					})
				}
				if i == 0 {
					// 第一个类型作为这个调用的name
					element.BaseElement.Name = realTyp
					element.Owner = owner
					continue
				}
				// 同时剩余的类型都要走引用
				refs = append(refs, NewReference(element, &cap.Node, realTyp, owner))
			}
		case types.ElementTypeCallArguments, types.ElementTypeNewExpressionArgs:
			params := ParseArgumentList(&cap.Node, rc.SourceFile.Content)
			// 只有数量可以用于匹配
			for _, param := range params {
				element.Parameters = append(element.Parameters, &param)
			}
		case types.ElementTypeCallOwner:
			element.Owner = StripSpaces(content)
		}
	}
	element.BaseElement.Scope = types.ScopeFunction
	var elements []Element
	elements = append(elements, element)
	for _, r := range refs {
		elements = append(elements, r)
	}
	for _, i := range imports {
		elements = append(elements, i)
	}
	return elements, nil
}

// ParseArgumentList 解析java方法调用中的参数列表
func ParseArgumentList(node *sitter.Node, content []byte) []Parameter {
	if node == nil {
		return nil
	}
	if types.ToNodeKind(node.Kind()) != types.NodeKindArgumentList {
		return nil
	}
	params := []Parameter{}
	for i := uint(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		if child == nil {
			continue
		}
		// 参数类型没有用
		params = append(params, Parameter{
			Name: child.Utf8Text(content),
			Type: []string{types.PrimitiveType},
		})
	}
	return params
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

// extractNodeName 从类/接口/枚举的声明节点中提取名称
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


// 用于搜索类型里面所有的单个类型，自动过滤了基础数据类型
func findAllTypes(node *sitter.Node, content []byte) []string {
	if node == nil {
		return nil
	}
	kind := types.ToNodeKind(node.Kind())
	if kind == types.NodeKindTypeIdentifier || kind == types.NodeKindScopedTypeIdentifier {
		return []string{node.Utf8Text(content)}
	}

	var typs []string
	for i := uint(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		if child == nil || child.IsMissing() || child.IsError() {
			continue
		}
		childTypes := findAllTypes(child, content)
		typs = append(typs, childTypes...)
	}
	return typs
}

// 用于搜索类型里面第一个单类型，自动过滤了基础数据类型
func findFirstType(node *sitter.Node, content []byte) string {
	if node == nil {
		return types.EmptyString
	}
	kind := types.ToNodeKind(node.Kind())
	if kind == types.NodeKindTypeIdentifier || kind == types.NodeKindScopedTypeIdentifier {
		return node.Utf8Text(content)
	}

	for i := uint(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		if child == nil || child.IsMissing() || child.IsError() {
			continue
		}
		childType := findFirstType(child, content)
		if childType != types.EmptyString {
			return childType
		}
	}
	return types.EmptyString
}

// 解析java语法中的type_list类型，返回类型列表
func parseTypeList(node *sitter.Node, content []byte) []string {
	if node == nil {
		return nil
	}
	if types.ToNodeKind(node.Kind()) != types.NodeKindTypeList {
		return []string{node.Utf8Text(content)}
	}
	typs := []string{}
	for i := uint(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		if child == nil || child.IsMissing() || child.IsError() {
			continue
		}
		switch types.ToNodeKind(child.Kind()) {
		case types.NodeKindScopedTypeIdentifier:
			typs = append(typs, child.Utf8Text(content))
		case types.NodeKindTypeIdentifier:
			typs = append(typs, child.Utf8Text(content))
		default:
			typ := findFirstType(child, content)
			if typ != types.EmptyString {
				typs = append(typs, typ)
			}
		}
	}
	return typs
}

// 解析java中的父类和接口，返回类型列表，并非所有类型，只返回第一个类型
func ParseSuperTypes(node *sitter.Node, content []byte) []string {
	if node == nil {
		return nil
	}
	typs := []string{}
	switch types.ToNodeKind(node.Kind()) {
	case types.NodeKindScopedTypeIdentifier:
		typs = append(typs, node.Utf8Text(content))
	case types.NodeKindTypeIdentifier:
		typs = append(typs, node.Utf8Text(content))
	case types.NodeKindAnnotatedType, types.NodeKindGenericType:
		typ := findFirstType(node, content)
		if typ != types.EmptyString {
			typs = append(typs, typ)
		}
	case types.NodeKindTypeList:
		// 多继承、多实现的情况
		typs = append(typs, parseTypeList(node, content)...)
	}
	return typs
}

// 解析java方法中的参数列表里面的所有类型
func ParseParameterList(node *sitter.Node, content []byte) []Parameter {
	if node == nil || types.ToNodeKind(node.Kind()) != types.NodeKindFormalParameters {
		return nil
	}
	params := []Parameter{}
	for i := uint(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		param, err := parseParameter(child, content)
		if err != nil {
			continue
		}
		params = append(params, param)
	}
	return params
}

// 解析java方法中单个参数类型里面的所有类型
func parseParameter(node *sitter.Node, content []byte) (Parameter, error) {
	if node == nil || types.ToNodeKind(node.Kind()) != types.NodeKindFormalParameter {
		// 一定要校验，否则会panic，可能会接收注释
		return Parameter{}, fmt.Errorf("node is nil or not a formal parameter")
	}
	typs := []string{}
	typChild := node.ChildByFieldName("type")
	if typChild != nil {
		typs = append(typs, findAllTypes(typChild, content)...)
	} else {
		typs = []string{types.PrimitiveType}
	}
	nameChild := node.ChildByFieldName("name")
	if nameChild != nil {
		name := nameChild.Utf8Text(content)
		return Parameter{
			Name: name,
			Type: typs,
		}, nil
	}
	return Parameter{}, fmt.Errorf("name is nil")
}
