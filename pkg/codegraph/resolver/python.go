package resolver

import (
	"codebase-indexer/pkg/codegraph/types"
	"context"
	"fmt"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

type PythonResolver struct {
}

var _ ElementResolver = &PythonResolver{}

func (py *PythonResolver) Resolve(ctx context.Context, element Element, rc *ResolveContext) ([]Element, error) {
	return resolve(ctx, py, element, rc)
}

func (py *PythonResolver) resolveImport(ctx context.Context, element *Import, rc *ResolveContext) ([]Element, error) {
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
			element.BaseElement.Name = content
		case types.ElementTypeImportSource:
			element.Source = content
		case types.ElementTypeImportAlias:
			element.Alias = content
		}
	}
	element.BaseElement.Scope = types.ScopePackage
	return []Element{element}, nil
}

func (py *PythonResolver) resolvePackage(ctx context.Context, element *Package, rc *ResolveContext) ([]Element, error) {
	// python 不支持 package 类型
	return nil, fmt.Errorf("python not support package")
}

func (py *PythonResolver) resolveFunction(ctx context.Context, element *Function, rc *ResolveContext) ([]Element, error) {
	rootCap := rc.Match.Captures[0]
	updateRootElement(element, &rootCap, rc.CaptureNames[rootCap.Index], rc.SourceFile.Content)
	if element.Declaration == nil {
		element.Declaration = &Declaration{}
	}
	for _, cap := range rc.Match.Captures {
		captureName := rc.CaptureNames[cap.Index]
		if cap.Node.IsMissing() || cap.Node.IsError() {
			continue
		}
		content := cap.Node.Utf8Text(rc.SourceFile.Content)
		switch types.ToElementType(captureName) {
		case types.ElementTypeFunctionName:
			element.BaseElement.Name = content
			element.Declaration.Name = content
		case types.ElementTypeFunctionParameters:
			element.Declaration.Parameters = parsePyFuncParams(&cap.Node, rc.SourceFile.Content)
		case types.ElementTypeFunctionReturnType:
			element.Declaration.ReturnType = collectPyTypeIdentifiers(&cap.Node, rc.SourceFile.Content)
		}
	}
	element.BaseElement.Scope = types.ScopePackage
	return []Element{element}, nil
}

func (py *PythonResolver) resolveMethod(ctx context.Context, element *Method, rc *ResolveContext) ([]Element, error) {
	rootCap := rc.Match.Captures[0]
	updateRootElement(element, &rootCap, rc.CaptureNames[rootCap.Index], rc.SourceFile.Content)
	if element.Declaration == nil {
		element.Declaration = &Declaration{}
	}
	for _, cap := range rc.Match.Captures {
		captureName := rc.CaptureNames[cap.Index]
		if cap.Node.IsMissing() || cap.Node.IsError() {
			continue
		}
		content := cap.Node.Utf8Text(rc.SourceFile.Content)
		switch types.ToElementType(captureName) {
		case types.ElementTypeMethodName:
			element.BaseElement.Name = content
			element.Declaration.Name = content
		case types.ElementTypeMethodParameters:
			element.Declaration.Parameters = parsePyFuncParams(&cap.Node, rc.SourceFile.Content)
			if len(element.Declaration.Parameters) > 0 && element.Declaration.Parameters[0].Name == "self" {
				element.Declaration.Parameters = element.Declaration.Parameters[1:]
			}
		case types.ElementTypeMethodReturnType:
			element.Declaration.ReturnType = collectPyTypeIdentifiers(&cap.Node, rc.SourceFile.Content)
		}
	}
	pnode := findMethodOwner(&rootCap.Node)
	if pnode != nil {
		element.Owner = extractNodeName(pnode, rc.SourceFile.Content)
	}
	element.BaseElement.Scope = types.ScopeClass
	return []Element{element}, nil
}

func (py *PythonResolver) resolveClass(ctx context.Context, element *Class, rc *ResolveContext) ([]Element, error) {
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
		case types.ElementTypeClassName:
			element.BaseElement.Name = content
		case types.ElementTypeClassExtends:
			element.SuperClasses = collectPyTypeIdentifiers(&cap.Node, rc.SourceFile.Content)
			for _, typ := range element.SuperClasses {
				refs = append(refs, NewReference(element, &cap.Node, typ, types.EmptyString))
			}
		}
	}
	element.Scope = types.ScopePackage
	elements := []Element{element}
	for _, r := range refs {
		elements = append(elements, r)
	}
	return elements, nil
}

func (py *PythonResolver) resolveVariable(ctx context.Context, element *Variable, rc *ResolveContext) ([]Element, error) {
	rootCap := rc.Match.Captures[0]
	var refs []*Reference
	updateRootElement(element, &rootCap, rc.CaptureNames[rootCap.Index], rc.SourceFile.Content)
	for _, cap := range rc.Match.Captures {
		captureName := rc.CaptureNames[cap.Index]
		if cap.Node.IsMissing() || cap.Node.IsError() {
			continue
		}
		content := cap.Node.Utf8Text(rc.SourceFile.Content)
		switch types.ToElementType(captureName) {
		case types.ElementTypeVariableName:
			element.BaseElement.Name = content
		case types.ElementTypeVariableType:
			element.VariableType = collectPyTypeIdentifiers(&cap.Node, rc.SourceFile.Content)
			for _, typ := range element.VariableType {
				refs = append(refs, NewReference(element, &cap.Node, typ, types.EmptyString))
			}
			// TODO 类的字段，用scm来做
		}
	}
	elements := []Element{element}
	for _, r := range refs {
		elements = append(elements, r)
	}
	return elements, nil
}

func (py *PythonResolver) resolveInterface(ctx context.Context, element *Interface, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	return nil, fmt.Errorf("not supported interface yet")
}

func (py *PythonResolver) resolveCall(ctx context.Context, element *Call, rc *ResolveContext) ([]Element, error) {
	rootCap := rc.Match.Captures[0]
	updateRootElement(element, &rootCap, rc.CaptureNames[rootCap.Index], rc.SourceFile.Content)
	for _, cap := range rc.Match.Captures {
		captureName := rc.CaptureNames[cap.Index]
		if cap.Node.IsMissing() || cap.Node.IsError() {
			continue
		}
		switch types.ToElementType(captureName) {
		case types.ElementTypeFunctionCallName:
			element.BaseElement.Name = cap.Node.Utf8Text(rc.SourceFile.Content)
		case types.ElementTypeFunctionArguments:
			args := getArgs(&cap.Node, rc.SourceFile.Content)
			for _, arg := range args {
				element.Parameters = append(element.Parameters, &Parameter{
					Name: arg,
				})
			}
		}
	}
	element.BaseElement.Scope = types.ScopeFunction
	elements := []Element{element}
	return elements, nil
}
func getArgs(node *sitter.Node, content []byte) []string {
	if node == nil {
		return nil
	}
	var args []string
	for i := uint(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		if child.IsMissing() || child.IsError() {
			continue
		}
		args = append(args, child.Utf8Text(content))
	}
	return args
}

// 解析python函数参数，可能包含name和type，也可能只包含其中一个，也可能是可变参数
func parsePyFuncParams(node *sitter.Node, content []byte) []Parameter {
	if node == nil {
		return nil
	}
	var params []Parameter
	for i := uint(0); i < node.NamedChildCount(); i++ {
		child := node.NamedChild(i)
		if child.IsMissing() || child.IsError() {
			continue
		}
		// fmt.Println("child", child.Kind())
		switch types.ToNodeKind(child.Kind()) {
		case types.NodeKindListSplatPattern:
			name := child.Utf8Text(content)
			name = strings.ReplaceAll(name, "*", "...")
			params = append(params, Parameter{
				Name: name,
			})
		case types.NodeKindDictSplatPattern:
			name := child.Utf8Text(content)
			name = strings.ReplaceAll(name, "**", "...")
			params = append(params, Parameter{
				Name: name,
			})
		case types.NodeKindDefaultParameter:
			name := child.ChildByFieldName("name").Utf8Text(content)
			params = append(params, Parameter{
				Name: name,
			})
		case types.NodeKindTypedParameter:
			name := child.Child(0).Utf8Text(content)
			name = strings.ReplaceAll(name, "**", "...")
			name = strings.ReplaceAll(name, "*", "...")
			typs := findAllIdentifiers(child.ChildByFieldName("type"), content)
			params = append(params, Parameter{
				Name: name,
				Type: typs,
			})
		case types.NodeKindTypedDefaultParameter:
			name := child.ChildByFieldName("name").Utf8Text(content)
			name = strings.ReplaceAll(name, "**", "...")
			name = strings.ReplaceAll(name, "*", "...")
			typs := findAllIdentifiers(child.ChildByFieldName("type"), content)
			if child.ChildByFieldName("value") != nil {
				valueTyps := collectPyTypeIdentifiers(child.ChildByFieldName("value"), content)
				typs = append(typs, valueTyps...)
			}
			params = append(params, Parameter{
				Name: name,
				Type: typs,
			})
		}
	}
	// fmt.Println("----------------------")
	return params
}

// collectPyTypeIdentifiers: 解析传参为类型的情况，也可处理单个复合参数解析为多个基础参数的情况
func collectPyTypeIdentifiers(node *sitter.Node, content []byte) []string {
	if node == nil {
		return nil
	}
	var results []string
	var walk func(n *sitter.Node)
	walk = func(n *sitter.Node) {
		if n == nil || n.IsMissing() || n.IsError() {
			return
		}
		switch types.ToNodeKind(n.Kind()) {
		case types.NodeKindIdentifier:
			results = append(results, n.Utf8Text(content))
		case types.NodeKindAttribute:
			// 只取最末尾的 attribute 字段
			attr := n.ChildByFieldName("attribute")
			if attr != nil {
				walk(attr)
			}
		case types.NodeKindSubscript:
			// 这个还可以给解析返回值或单个参数类型使用
			// 分别递归 value 和 subscript 字段
			// 捕获所有 subscript（可能有多个参数）
			for i := uint(0); i < n.NamedChildCount(); i++ {
				child := n.NamedChild(i)
				if child == nil || child.IsMissing() || child.IsError() {
					continue
				}
				walk(child)
			}
		case types.NodeKindKeywordArgument:
			name := n.ChildByFieldName("name").Utf8Text(content)
			if name != "metaclass" {
				return
			}
			value := n.ChildByFieldName("value")
			walk(value)
		case types.NodeKindGenericType:
			for i := uint(0); i < n.NamedChildCount(); i++ {
				child := n.NamedChild(i)
				if child == nil || child.IsMissing() || child.IsError() {
					continue
				}
				walk(child)
			}
		default:
			for i := uint(0); i < n.ChildCount(); i++ {
				walk(n.Child(i))
			}
		}
	}
	walk(node)
	return results
}
