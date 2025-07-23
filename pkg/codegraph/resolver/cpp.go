package resolver

import (
	"codebase-indexer/pkg/codegraph/types"
	"context"
	"fmt"
	"strings"
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
			element.Name = strings.TrimSpace(content)
		}
	}

	importName := element.Name

	elements := []Element{element}

	// 处理系统头文件
	if strings.HasPrefix(importName, "<") && strings.HasSuffix(importName, ">") {
		return elements, nil // 系统头文件，不映射到项目文件
	}
	return elements, nil
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
			element.Declaration.ReturnType = parseLocalVariableType(&cap.Node, rc.SourceFile.Content)
		case types.ElementTypeFunctionParameters:
			element.Declaration.Parameters = getFilteredParameters(content)
		}
	}
	return []Element{element}, nil
}

func (c *CppResolver) resolveMethod(ctx context.Context, element *Method, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}

func (c *CppResolver) resolveClass(ctx context.Context, element *Class, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}

func (c *CppResolver) resolveVariable(ctx context.Context, element *Variable, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}

func (c *CppResolver) resolveInterface(ctx context.Context, element *Interface, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}

func (c *CppResolver) resolveCall(ctx context.Context, element *Call, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}
