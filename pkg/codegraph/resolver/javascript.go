package resolver

import (
	"codebase-indexer/pkg/codegraph/types"
	"codebase-indexer/pkg/codegraph/utils"
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

type JavaScriptResolver struct {
}

var _ ElementResolver = &JavaScriptResolver{}

func (js *JavaScriptResolver) Resolve(ctx context.Context, element Element, rc *ResolveContext) ([]Element, error) {
	return resolve(ctx, js, element, rc)
}

func (js *JavaScriptResolver) resolveImport(ctx context.Context, element *Import, rc *ResolveContext) ([]Element, error) {
	if element.Name == types.EmptyString {
		return nil, fmt.Errorf("import is empty")
	}

	element.FilePaths = []string{}
	importName := element.Name
	pj := rc.ProjectInfo

	elements := []Element{element}

	if pj.IsEmpty() {
		fmt.Println("not support project file list, use default resolve")
		cleanedPath := strings.ReplaceAll(strings.ReplaceAll(importName, "./", ""), "../", "")
		element.FilePaths = []string{cleanedPath}
		return elements, nil
	}

	// 处理相对路径
	if strings.HasPrefix(importName, "./") || strings.HasPrefix(importName, "../") {
		// 计算当前文件相对于 sourceRoot 的路径
		currentRelPath, _ := filepath.Rel(pj.GetSourceRoot(), rc.SourceFile.Path)
		currentDir := utils.ToUnixPath(filepath.Dir(currentRelPath))
		targetPath := utils.ToUnixPath(filepath.Join(currentDir, importName))
		foundPaths := []string{}

		// 尝试不同的文件扩展名
		for _, ext := range []string{".ts", ".tsx", ".js", ".jsx", "/index.ts", "/index.tsx", "/index.js", "/index.jsx"} {
			fullPath := utils.ToUnixPath(filepath.Join(pj.GetSourceRoot(), targetPath+ext))
			if pj.ContainsFileIndex(fullPath) {
				foundPaths = append(foundPaths, fullPath)
			}
		}

		element.FilePaths = foundPaths
		if len(element.FilePaths) > 0 {
			return elements, nil
		}

		return nil, fmt.Errorf("cannot find file which relative import belongs to: %s", importName)
	}

	// 处理项目内绝对路径导入
	foundPaths := []string{}
	for _, relDir := range pj.GetDirs() {
		for _, ext := range []string{".ts", ".tsx", ".js", ".jsx", "/index.ts", "/index.tsx", "/index.js", "/index.jsx"} {
			fullPath := utils.ToUnixPath(filepath.Join(relDir, importName+ext))
			if pj.ContainsFileIndex(fullPath) {
				foundPaths = append(foundPaths, fullPath)
			}
		}
	}

	element.FilePaths = foundPaths
	if len(element.FilePaths) > 0 {
		return elements, nil
	}

	return nil, fmt.Errorf("cannot find file which import belongs to: %s", importName)
}

func (js *JavaScriptResolver) resolvePackage(ctx context.Context, element *Package, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}

func (js *JavaScriptResolver) resolveFunction(ctx context.Context, element *Function, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}

func (js *JavaScriptResolver) resolveMethod(ctx context.Context, element *Method, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}

func (js *JavaScriptResolver) resolveClass(ctx context.Context, element *Class, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}

func (js *JavaScriptResolver) resolveVariable(ctx context.Context, element *Variable, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}

func (js *JavaScriptResolver) resolveInterface(ctx context.Context, element *Interface, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}
