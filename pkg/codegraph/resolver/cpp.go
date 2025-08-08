package resolver

import (
	"codebase-indexer/pkg/codegraph/types"
	"context"
	"fmt"
	"strings"

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
			element.BaseElement.Name = content
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
			element.BaseElement.Name = strings.TrimSpace(content)
			element.Declaration.Name = element.BaseElement.Name
		case types.ElementTypeFunctionReturnType:
			typs := findAllTypeIdentifiers(&cap.Node, rc.SourceFile.Content)
			if len(typs) == 0 {
				typs = []string{types.PrimitiveType}
			}
			typs = types.FilterCustomTypes(typs)
			element.Declaration.ReturnType = typs
		case types.ElementTypeFunctionParameters:
			// element.Declaration.Parameters = getFilteredParameters(content)
			parameters := parseCppParameters(&cap.Node, rc.SourceFile.Content)
			for i := range parameters {
				parameters[i].Type = types.FilterCustomTypes(parameters[i].Type)
			}
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
			element.Declaration.ReturnType = getFilteredTypes(content)
		case types.ElementTypeMethodParameters:
			element.Declaration.Parameters = getFilteredParameters(content)
		case types.ElementTypeMethodName:
			element.BaseElement.Name = strings.TrimSpace(content)
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
			element.BaseElement.Name = strings.TrimSpace(content)
		case types.ElementTypeTypedefAlias, types.ElementTypeTypeAliasAlias:
			// typedef只考虑alias
			name := strings.TrimSpace(content)
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
	// var refs = []*Reference{}
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
			element.BaseElement.Name = CleanParam(content)
			if isLocalVariable(&cap.Node) {
				element.BaseElement.Scope = types.ScopeFunction
			} else {
				// 字段目前不算局部变量
				element.BaseElement.Scope = types.ScopeClass
			}
		case types.ElementTypeVariableType, types.ElementTypeFieldType:
			element.VariableType = getFilteredTypes(content)
		case types.ElementTypeEnumConstantName:
			// 枚举的类型不考虑，都是基础类型（有匿名枚举）
			element.BaseElement.Name = CleanParam(content)
			element.VariableType = []string{types.PrimitiveType}
			element.BaseElement.Scope = types.ScopeClass
		}
	}
	elems := []Element{element}
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
			element.BaseElement.Name = strings.TrimSpace(content)
		case types.ElementTypeFunctionOwner, types.ElementTypeCallOwner, types.ElementTypeNewExpressionOwner:
			element.Owner = strings.TrimSpace(content)
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
				element.BaseElement.Name = names[0]
			} else {
				element.BaseElement.Name = content
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
					Name: argContent,
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

	if node == nil {
		return nil
	}
	var params []Parameter
	for i := uint(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
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
			if declaratorNode != nil {
				// 理论上delcs第一个应该是参数名，后面是嵌套的参数，不管
				decls := findAllIdentifiers(declaratorNode, content)
				if len(decls) > 0 {
					param.Name = decls[0]
				}
			}
			params = append(params, param)

		case "variadic_parameter":
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
