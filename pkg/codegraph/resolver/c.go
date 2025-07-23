package resolver

import (
	"codebase-indexer/pkg/codegraph/types"
	"context"
	"fmt"
	"strings"
)

type CResolver struct {
}

var _ ElementResolver = &CResolver{}

func (r *CResolver) Resolve(ctx context.Context, element Element, rc *ResolveContext) ([]Element, error) {
	return resolve(ctx, r, element, rc)
}

func (r *CResolver) resolveImport(ctx context.Context, element *Import, rc *ResolveContext) ([]Element, error) {
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
			element.Name = content
		}
	}

	importName := element.Name

	elements := []Element{element}

	// 处理系统头文件
	if strings.HasPrefix(importName, "<") && strings.HasSuffix(importName, ">") {
		return elements, nil // 系统头文件，不映射到项目文件
	}

	return nil, fmt.Errorf("cannot find file which import belongs to: %s", importName)
}

func (r *CResolver) resolvePackage(ctx context.Context, element *Package, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}

func (r *CResolver) resolveFunction(ctx context.Context, element *Function, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}

func (r *CResolver) resolveMethod(ctx context.Context, element *Method, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}

func (r *CResolver) resolveClass(ctx context.Context, element *Class, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}

func (r *CResolver) resolveVariable(ctx context.Context, element *Variable, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}

func (r *CResolver) resolveInterface(ctx context.Context, element *Interface, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}

func (r *CResolver) resolveCall(ctx context.Context, element *Call, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}
