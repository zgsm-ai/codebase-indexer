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
			element.Declaration.ReturnType = getFilteredReturnType(content)
		case types.ElementTypeFunctionParameters:
			element.Declaration.Parameters = getFilteredParameters(content)
		}
	}
	element.BaseElement.Scope = types.ScopeProject
	return []Element{element}, nil
}

func (c *CppResolver) resolveMethod(ctx context.Context, element *Method, rc *ResolveContext) ([]Element, error) {
	//TODO 待完成
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
			element.Declaration.ReturnType = getFilteredReturnType(content)
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
	fmt.Println("func return_type", element.Declaration.ReturnType)
	fmt.Println("func name", element.BaseElement.Name)
	fmt.Println("func parameters", element.Declaration.Parameters)
	fmt.Println("func owner", element.Owner)
	fmt.Println("func scope", element.BaseElement.Scope)
	fmt.Println("--------------------------------")

	return []Element{element}, nil
}

func (c *CppResolver) resolveClass(ctx context.Context, element *Class, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	return nil, fmt.Errorf("not support class")
}

func (c *CppResolver) resolveVariable(ctx context.Context, element *Variable, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	return nil, fmt.Errorf("not support class")
}

func (c *CppResolver) resolveInterface(ctx context.Context, element *Interface, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}

func (c *CppResolver) resolveCall(ctx context.Context, element *Call, rc *ResolveContext) ([]Element, error) {
	//TODO 待完成
	rootCap := rc.Match.Captures[0]
	updateRootElement(element, &rootCap, rc.CaptureNames[rootCap.Index], rc.SourceFile.Content)
	for _, cap := range rc.Match.Captures {
		captureName := rc.CaptureNames[cap.Index]
		if cap.Node.IsMissing() || cap.Node.IsError() {
			continue
		}
		content := cap.Node.Utf8Text(rc.SourceFile.Content)
		switch types.ToElementType(captureName) {
		case types.ElementTypeFunctionCall:
			element.BaseElement.Name = strings.TrimSpace(content)
		}
	}
	return []Element{element}, nil
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


