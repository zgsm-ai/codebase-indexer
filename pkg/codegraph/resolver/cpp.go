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
			element.Declaration.ReturnType = getFilteredTypes(content)
		case types.ElementTypeFunctionParameters:
			element.Declaration.Parameters = getFilteredParameters(content)
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
	//TODO implement me
	return nil, fmt.Errorf("not support class")
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
			element.BaseElement.Name = CleanParam(content)
		case types.ElementTypeVariableType, types.ElementTypeFieldType:
			element.VariableType = getFilteredTypes(content)
		case types.ElementTypeVariableValue, types.ElementTypeFieldValue:
			// 有可能是字面量，也有可能是类和结构体的创建，和方法调用
			// 只能处理一个 new 的创建
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
				Owner: "", // 待定
			}
			refs = append(refs, ref)
		}
	}
	element.BaseElement.Scope = types.ScopeFunction
	elems := []Element{element}
	for _, ref := range refs {
		elems = append(elems, ref)
	}
	return elems, nil
}

func (c *CppResolver) resolveInterface(ctx context.Context, element *Interface, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	return nil, fmt.Errorf("not support interface")
}

func (c *CppResolver) resolveCall(ctx context.Context, element *Call, rc *ResolveContext) ([]Element, error) {
	rootCap := rc.Match.Captures[0]
	updateRootElement(element, &rootCap, rc.CaptureNames[rootCap.Index], rc.SourceFile.Content)
	for _, cap := range rc.Match.Captures {
		captureName := rc.CaptureNames[cap.Index]
		if cap.Node.IsMissing() || cap.Node.IsError() {
			continue
		}
		content := cap.Node.Utf8Text(rc.SourceFile.Content)
		switch types.ToElementType(captureName) {
		case types.ElementTypeFunctionCallName, types.ElementTypeCallName:
			element.BaseElement.Name = strings.TrimSpace(content)
		case types.ElementTypeFunctionOwner, types.ElementTypeCallOwner:
			element.Owner = strings.TrimSpace(content)
		case types.ElementTypeFunctionArguments, types.ElementTypeCallArguments:
			// 暂时只保留name，参数类型先不考虑
			for i := uint(0); i < cap.Node.NamedChildCount(); i++ {
				arg := cap.Node.NamedChild(i)
				argContent := arg.Utf8Text(rc.SourceFile.Content)
				element.Parameters = append(element.Parameters, &Parameter{
					Name: argContent,
					Type: []string{},
				})
			}
		}
	}
	element.BaseElement.Scope = types.ScopeBlock
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
