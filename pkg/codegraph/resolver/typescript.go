package resolver

import (
	"codebase-indexer/pkg/codegraph/types"
	"context"
	"strings"
)

type TypeScriptResolver struct {
}

var _ ElementResolver = &TypeScriptResolver{}

func (ts *TypeScriptResolver) Resolve(ctx context.Context, element Element, rc *ResolveContext) ([]Element, error) {
	/*
		特殊处理：require调用应该被解析为import，而不是call或variable
		后面提取出来作为公共函数
	*/
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
							importElement.Alias = string(nameNode.Utf8Text(rc.SourceFile.Content))
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
			element.Content = []byte(content)
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
	return []Element{element}, nil
}

func (ts *TypeScriptResolver) resolveFunction(ctx context.Context, element *Function, rc *ResolveContext) ([]Element, error) {
	return []Element{element}, nil
}

func (ts *TypeScriptResolver) resolveMethod(ctx context.Context, element *Method, rc *ResolveContext) ([]Element, error) {
	return []Element{element}, nil
}

func (ts *TypeScriptResolver) resolveClass(ctx context.Context, element *Class, rc *ResolveContext) ([]Element, error) {
	return []Element{element}, nil
}

func (ts *TypeScriptResolver) resolveVariable(ctx context.Context, element *Variable, rc *ResolveContext) ([]Element, error) {
	return []Element{element}, nil
}

func (ts *TypeScriptResolver) resolveInterface(ctx context.Context, element *Interface, rc *ResolveContext) ([]Element, error) {
	return []Element{element}, nil
}

func (ts *TypeScriptResolver) resolveCall(ctx context.Context, element *Call, rc *ResolveContext) ([]Element, error) {
	return []Element{element}, nil
}
