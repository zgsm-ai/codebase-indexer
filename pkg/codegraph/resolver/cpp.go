package resolver

import (
	"codebase-indexer/pkg/codegraph/types"
	"context"
	"fmt"
	sitter "github.com/tree-sitter/go-tree-sitter"
)

type CppResolver struct {
}

var _ ElementResolver = &CppResolver{}

func (c *CppResolver) Resolve(ctx context.Context, element Element, rc *ResolveContext) ([]Element, error) {
	return resolve(ctx, c, element, rc)
}

func (c *CppResolver) resolveImport(ctx context.Context, element *Import, rc *ResolveContext) ([]Element, error) {
	rootCap := rc.Match.Captures[0]
	updateRootElement(element, &rootCap, rc.CaptureNames[rootCap.Index], rc.SourceFile.Content)
	for _, cap := range rc.Match.Captures {
		captureName := rc.CaptureNames[cap.Index]
		if cap.Node.IsMissing() || cap.Node.IsError() {
			continue
		}
		content := cap.Node.Utf8Text(rc.SourceFile.Content)
		switch types.ToElementType(captureName) {
		case types.ElementTypeImportName:
			// 容错处理，出现空格，语法会报错，但也应该能解析
			element.BaseElement.Name = StripSpaces(content)
		}
	}
	element.BaseElement.Scope = types.ScopeProject
	return []Element{element}, nil
}

func (c *CppResolver) resolvePackage(ctx context.Context, element *Package, rc *ResolveContext) ([]Element, error) {
	// TODO 没有这个概念，不实现
	return nil, fmt.Errorf("not support package")
}

func (c *CppResolver) resolveFunction(ctx context.Context, element *Function, rc *ResolveContext) ([]Element, error) {
	rootCap := rc.Match.Captures[0]
	updateRootElement(element, &rootCap, rc.CaptureNames[rootCap.Index], rc.SourceFile.Content)
	for _, cap := range rc.Match.Captures {
		captureName := rc.CaptureNames[cap.Index]
		if cap.Node.IsMissing() || cap.Node.IsError() {
			continue
		}
		content := cap.Node.Utf8Text(rc.SourceFile.Content)
		switch types.ToElementType(captureName) {
		case types.ElementTypeFunctionName:
			element.BaseElement.Name = StripSpaces(content)
			element.Declaration.Name = element.BaseElement.Name
		case types.ElementTypeFunctionReturnType:
			typs := findAllTypeIdentifiers(&cap.Node, rc.SourceFile.Content)
			if len(typs) == 0 {
				typs = []string{types.PrimitiveType}
			}
			element.Declaration.ReturnType = typs
		case types.ElementTypeFunctionParameters:
			parameters := parseCppParameters(&cap.Node, rc.SourceFile.Content)
			element.Declaration.Parameters = parameters
		}
	}
	element.BaseElement.Scope = types.ScopeProject
	return []Element{element}, nil
}

func (c *CppResolver) resolveMethod(ctx context.Context, element *Method, rc *ResolveContext) ([]Element, error) {
	rootCap := rc.Match.Captures[0]
	updateRootElement(element, &rootCap, rc.CaptureNames[rootCap.Index], rc.SourceFile.Content)
	for _, cap := range rc.Match.Captures {
		captureName := rc.CaptureNames[cap.Index]
		if cap.Node.IsMissing() || cap.Node.IsError() {
			continue
		}
		content := cap.Node.Utf8Text(rc.SourceFile.Content)
		switch types.ToElementType(captureName) {
		case types.ElementTypeMethodReturnType:
			element.Declaration.ReturnType = findAllTypeIdentifiers(&cap.Node, rc.SourceFile.Content)
			if len(element.Declaration.ReturnType) == 0 {
				element.Declaration.ReturnType = []string{types.PrimitiveType}
			}
		case types.ElementTypeMethodParameters:
			element.Declaration.Parameters = parseCppParameters(&cap.Node, rc.SourceFile.Content)
		case types.ElementTypeMethodName:
			element.BaseElement.Name = StripSpaces(content)
			element.Declaration.Name = element.BaseElement.Name
		}
	}
	// 设置owner并且补充默认修饰符
	ownerNode := findMethodOwner(&rootCap.Node)
	var ownerKind types.NodeKind
	if ownerNode != nil {
		element.Owner = extractNodeName(ownerNode, rc.SourceFile.Content)
		ownerKind = types.ToNodeKind(ownerNode.Kind())
	}
	modifier := findAccessSpecifier(&rootCap.Node, rc.SourceFile.Content)
	// 补充作用域
	element.BaseElement.Scope = getScopeFromModifiers(modifier, ownerKind)

	return []Element{element}, nil
}

func (c *CppResolver) resolveClass(ctx context.Context, element *Class, rc *ResolveContext) ([]Element, error) {
	rootCap := rc.Match.Captures[0]
	updateRootElement(element, &rootCap, rc.CaptureNames[rootCap.Index], rc.SourceFile.Content)
	var refs []*Reference
	for _, cap := range rc.Match.Captures {
		captureName := rc.CaptureNames[cap.Index]
		if cap.Node.IsMissing() || cap.Node.IsError() {
			continue
		}
		content := cap.Node.Utf8Text(rc.SourceFile.Content)
		switch types.ToElementType(captureName) {
		case types.ElementTypeClassName, types.ElementTypeStructName, types.ElementTypeEnumName,
			types.ElementTypeUnionName, types.ElementTypeNamespaceName:
			// 枚举类型只考虑name
			element.BaseElement.Name = StripSpaces(content)
		case types.ElementTypeTypedefAlias, types.ElementTypeTypeAliasAlias:
			// typedef只考虑alias
			name := StripSpaces(content)
			// 去除指针引用以及修饰符
			name = CleanParam(name)
			element.BaseElement.Name = name
		case types.ElementTypeClassExtends, types.ElementTypeStructExtends:
			// 不考虑cpp的ns调用，owner暂时无用
			typs := parseBaseClassClause(&cap.Node, rc.SourceFile.Content)
			for _, typ := range typs {
				refs = append(refs, NewReference(element, &cap.Node, typ, types.EmptyString))
				element.SuperClasses = append(element.SuperClasses, typ)
			}
		}
	}
	element.BaseElement.Scope = types.ScopeProject
	elems := []Element{element}
	for _, ref := range refs {
		elems = append(elems, ref)
	}
	return elems, nil
}

func (c *CppResolver) resolveVariable(ctx context.Context, element *Variable, rc *ResolveContext) ([]Element, error) {
	// 字段和变量统一处理
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
		case types.ElementTypeVariableName, types.ElementTypeFieldName:
			// 去除指针引用
			element.BaseElement.Name = CleanParam(content)
			if isLocalVariable(&cap.Node) {
				element.BaseElement.Scope = types.ScopeFunction
			} else {
				// 字段目前不算局部变量
				element.BaseElement.Scope = types.ScopeClass
			}
		case types.ElementTypeVariableType, types.ElementTypeFieldType:
			typs:= findAllTypeIdentifiers(&cap.Node, rc.SourceFile.Content)
			for _, typ := range typs {
				refs = append(refs, NewReference(element, &cap.Node, typ, types.EmptyString))
			}
			element.VariableType = typs
			if len(element.VariableType) == 0 {
				element.VariableType = []string{types.PrimitiveType}
			}			
		case types.ElementTypeEnumConstantName:
			// 枚举的类型不考虑，都是基础类型（有匿名枚举）
			element.BaseElement.Name = StripSpaces(content)
			element.VariableType = []string{types.PrimitiveType}
			element.BaseElement.Scope = types.ScopeClass
		}
	}
	elems := []Element{element}
	for _, ref := range refs {
		elems = append(elems, ref)
	}
	return elems, nil
}

func (c *CppResolver) resolveInterface(ctx context.Context, element *Interface, rc *ResolveContext) ([]Element, error) {
	return nil, fmt.Errorf("cpp not support interface")
}

func (c *CppResolver) resolveCall(ctx context.Context, element *Call, rc *ResolveContext) ([]Element, error) {
	rootCap := rc.Match.Captures[0]
	updateRootElement(element, &rootCap, rc.CaptureNames[rootCap.Index], rc.SourceFile.Content)
	var refs []*Reference
	for _, cap := range rc.Match.Captures {
		captureName := rc.CaptureNames[cap.Index]
		if cap.Node.IsMissing() || cap.Node.IsError() {
			continue
		}
		content := cap.Node.Utf8Text(rc.SourceFile.Content)
		switch types.ToElementType(captureName) {
		case types.ElementTypeFunctionCallName, types.ElementTypeCallName, types.ElementTypeTemplateCallName,
			types.ElementTypeNewExpressionType:
			element.BaseElement.Name = findFirstIdentifier(&cap.Node, rc.SourceFile.Content)
			if element.BaseElement.Name == types.EmptyString {
				// 避免为空
				element.BaseElement.Name = StripSpaces(content)
			}
		case types.ElementTypeFunctionOwner, types.ElementTypeCallOwner, types.ElementTypeNewExpressionOwner:
			element.Owner = StripSpaces(content)
		case types.ElementTypeTemplateCallArgs:
			typs := findAllTypeIdentifiers(&cap.Node, rc.SourceFile.Content)
			if len(typs) != 0 {
				for _, typ := range typs {
					// TODO 可以考虑解析出来命名空间
					refs = append(refs, NewReference(element, &cap.Node, typ, types.EmptyString))
				}
			}
		case types.ElementTypeCompoundLiteralType:
			names := findAllTypeIdentifiers(&cap.Node, rc.SourceFile.Content)
			// (struct MyStruct)
			if len(names) != 0 {
				// 找到第一个类型，作为name
				element.BaseElement.Name = StripSpaces(names[0])
			} else {
				element.BaseElement.Name = StripSpaces(content)
			}
		case types.ElementTypeFunctionArguments, types.ElementTypeCallArguments, types.ElementTypeNewExpressionArgs:
			// 暂时只保留name，参数类型先不考虑
			for i := uint(0); i < cap.Node.NamedChildCount(); i++ {
				arg := cap.Node.NamedChild(i)
				if arg.Kind() == "comment" {
					// 过滤comment
					continue
				}
				argContent := arg.Utf8Text(rc.SourceFile.Content)
				element.Parameters = append(element.Parameters, &Parameter{
					Name: StripSpaces(argContent),
					Type: []string{},
				})
			}
		}
	}
	element.BaseElement.Scope = types.ScopeFunction
	elems := []Element{element}
	for _, ref := range refs {
		elems = append(elems, ref)
	}

	return elems, nil
}
func findAccessSpecifier(node *sitter.Node, content []byte) string {
	// 1. 向上找到 field_declaration_list
	parent := node.Parent()
	for parent != nil && types.ToNodeKind(parent.Kind()) != types.NodeKindFieldList {
		parent = parent.Parent()
	}
	if parent == nil {
		return types.EmptyString // 没找到
	}

	// 2. 在 field_declaration_list 的 children 里，找到 node 前面最近的 access_specifier
	var lastAccess string
	for i := uint(0); i < parent.NamedChildCount(); i++ {
		child := parent.NamedChild(i)
		if child == node {
			break
		}
		if types.ToNodeKind(child.Kind()) == types.NodeKindAccessSpecifier {
			lastAccess = child.Utf8Text(content) // 例如 "public", "private", "protected"
		}
	}
	if lastAccess != types.EmptyString {
		return lastAccess
	}
	// 3. 这里不给默认修饰符
	return types.EmptyString
}

func parseCppParameters(node *sitter.Node, content []byte) []Parameter {

	if node == nil || types.ToNodeKind(node.Kind()) != types.NodeKindParameterList {
		return nil
	}
	var params []Parameter
	for i := uint(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		if child == nil || child.IsMissing() || child.IsError() {
			continue
		}
		childKind := child.Kind()
		switch types.ToNodeKind(childKind) {
		case types.NodeKindParameterDeclaration:
			typs := findAllTypeIdentifiers(child, content)
			if len(typs) == 0 {
				typs = []string{types.PrimitiveType}
			}
			param := Parameter{
				Name: types.EmptyString,
				Type: typs,
			}
			// 可能为nil，即无名参数，只有类型
			declaratorNode := child.ChildByFieldName("declarator")
			// 理论上delcs第一个应该是参数名，后面是嵌套的参数，不管(这里只有一层)
			decls := findAllIdentifiers(declaratorNode, content)
			if len(decls) > 0 {
				param.Name = decls[0]
			}
			params = append(params, param)

		case types.NodeKindVariadicParameter:
			// ...可变参数的情况
			params = append(params, Parameter{
				Name: "...",
				Type: []string{},
			})
		}
	}
	return params
}

func isLocalVariable(node *sitter.Node) bool {
	current := node
	for current != nil {
		kind := current.Kind()
		switch types.ToNodeKind(kind) {
		// cpp、java
		case types.NodeKindFunctionDeclaration, types.NodeKindMethodDeclaration:
			return true
		case types.NodeKindClassDeclaration, types.NodeKindClassSpecifier, types.NodeKindStructSpecifier:
			// 如果在类或结构体内部，但不是局部变量
			return false
		default:
			// 继续向上查找
			current = current.Parent()
		}
	}
	return false
}
// 处理cpp语法中的base_class_clause类型，返回类型列表
func parseBaseClassClause(node *sitter.Node, content []byte) []string {
	if node == nil {
		return nil
	}

	// 如果不是base_class_clause节点，直接返回节点内容
	if types.ToNodeKind(node.Kind()) != types.NodeKindBaseClassClause {
		return []string{node.Utf8Text(content)}
	}

	typs := []string{}

	// 从后往前遍历所有子节点
	for i := int(node.NamedChildCount()) - 1; i >= 0; i-- {
		child := node.NamedChild(uint(i))
		if child == nil || child.Kind() == types.Comma || child.Kind() == types.Colon {
			continue
		}

		// 处理类型节点
		var baseClasses []string

		if types.ToNodeKind(child.Kind()) == types.NodeKindTypeIdentifier {
			// 直接是type_identifier
			baseClasses = []string{child.Utf8Text(content)}
		} else {
			// 不是type_identifier，递归查找所有的type_identifier
			baseClasses = findAllTypeIdentifiers(child, content)
		}

		// 如果找到了类型标识符，添加到结果中
		if len(baseClasses) > 0 {
			typs = append(typs, baseClasses...)
		}
	}

	return typs
}