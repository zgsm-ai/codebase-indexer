package resolver

import (
	"codebase-indexer/pkg/codegraph/types"
	"codebase-indexer/pkg/codegraph/utils"
	"context"
	"fmt"
	"path/filepath"
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

	element.FilePaths = []string{}
	importName := element.Name

	elements := []Element{element}

	// 处理系统头文件
	if strings.HasPrefix(importName, "<") && strings.HasSuffix(importName, ">") {
		return elements, nil // 系统头文件，不映射到项目文件
	}

	// 移除引号
	headerFile := strings.Trim(importName, "\"")

	pj := rc.ProjectInfo
	if pj.IsEmpty() {
		fmt.Println("not support project file list, use default resolve")
		element.FilePaths = []string{headerFile}
		return elements, nil
	}

	foundPaths := []string{}

	// 相对路径导入
	if strings.HasPrefix(headerFile, ".") {
		// 计算当前文件相对于 sourceRoot 的路径
		currentRelPath, _ := filepath.Rel(pj.GetSourceRoot(), rc.SourceFile.Path)
		currentDir := utils.ToUnixPath(filepath.Dir(currentRelPath))
		relPath := utils.ToUnixPath(filepath.Join(currentDir, headerFile))
		fullPath := utils.ToUnixPath(filepath.Join(pj.GetSourceRoot(), relPath))
		if pj.ContainsFileIndex(fullPath) {
			foundPaths = append(foundPaths, fullPath)
		}
	}

	// 在源目录中查找
	for _, relDir := range pj.GetDirs() {
		fullPath := utils.ToUnixPath(filepath.Join(relDir, headerFile))
		if pj.ContainsFileIndex(fullPath) {
			foundPaths = append(foundPaths, fullPath)
		}
	}

	element.FilePaths = foundPaths
	if len(element.FilePaths) > 0 {
		return elements, nil
	}
	return elements, nil
	// return nil, fmt.Errorf("cannot find file which import belongs to: %s", importName)
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
