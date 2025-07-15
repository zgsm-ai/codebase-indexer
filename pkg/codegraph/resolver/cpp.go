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
	if element.Name == types.EmptyString {
		return nil, fmt.Errorf("import is empty")
	}

	element.FilePaths = []string{}
	importName := element.Name

	elements := []Element{element}
	// 处理系统头文件
	if strings.HasPrefix(importName, "<") && strings.HasSuffix(importName, ">") {
		return elements, nil // 系统头文件，不映射到项目文件
	}

	// 移除引号
	headerFile := strings.Trim(importName, types.DoubleQuote)

	pj := rc.ProjectInfo
	if pj.IsEmpty() {
		fmt.Println("not support project file list, use default resolve")
		element.FilePaths = []string{headerFile}
		return elements, nil
	}

	foundPaths := []string{}

	// 相对路径导入
	if strings.HasPrefix(headerFile, types.Dot) {
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

	return nil, fmt.Errorf("cannot find file which import belongs to: %s", importName)
}

func (c *CppResolver) resolvePackage(ctx context.Context, element *Package, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}

func (c *CppResolver) resolveFunction(ctx context.Context, element *Function, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
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
