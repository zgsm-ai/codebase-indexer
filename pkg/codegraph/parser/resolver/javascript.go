package resolver

import (
	"codebase-indexer/pkg/codegraph/parser/utils"
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

type JavaScriptResolver struct {
}

var _ ElementResolver = &JavaScriptResolver{}

func (js *JavaScriptResolver) Resolve(ctx context.Context, element Element, rc *ResolveContext) error {
	return resolve(ctx, js, element, rc)
}

func (js *JavaScriptResolver) resolveImport(ctx context.Context, element Import, rc *ResolveContext) error {
	if element.Name == EmptyString {
		return fmt.Errorf("import is empty")
	}

	element.FilePaths = []string{}
	importName := element.Name
	pc := rc.ProjectInfo
	if len(pc.fileSet) == 0 {
		fmt.Println("not support project file list, use default resolve")
		cleanedPath := strings.ReplaceAll(strings.ReplaceAll(importName, "./", ""), "../", "")
		element.FilePaths = []string{cleanedPath}
		return nil
	}

	// 处理相对路径
	if strings.HasPrefix(importName, "./") || strings.HasPrefix(importName, "../") {
		// 计算当前文件相对于 SourceRoot 的路径
		currentRelPath, _ := filepath.Rel(pc.SourceRoot, rc.SourceFile.Path)
		currentDir := utils.ToUnixPath(filepath.Dir(currentRelPath))
		targetPath := utils.ToUnixPath(filepath.Join(currentDir, importName))
		foundPaths := []string{}

		// 尝试不同的文件扩展名
		for _, ext := range []string{".ts", ".tsx", ".js", ".jsx", "/index.ts", "/index.tsx", "/index.js", "/index.jsx"} {
			fullPath := utils.ToUnixPath(filepath.Join(pc.SourceRoot, targetPath+ext))
			if pc.containsFileIndex(fullPath) {
				foundPaths = append(foundPaths, fullPath)
			}
		}

		element.FilePaths = foundPaths
		if len(element.FilePaths) > 0 {
			return nil
		}

		return fmt.Errorf("cannot find file which relative import belongs to: %s", importName)
	}

	// 处理项目内绝对路径导入
	foundPaths := []string{}
	for _, relDir := range pc.Dirs {
		for _, ext := range []string{".ts", ".tsx", ".js", ".jsx", "/index.ts", "/index.tsx", "/index.js", "/index.jsx"} {
			fullPath := utils.ToUnixPath(filepath.Join(relDir, importName+ext))
			if pc.containsFileIndex(fullPath) {
				foundPaths = append(foundPaths, fullPath)
			}
		}
	}

	element.FilePaths = foundPaths
	if len(element.FilePaths) > 0 {
		return nil
	}

	return fmt.Errorf("cannot find file which import belongs to: %s", importName)
}

func (js *JavaScriptResolver) resolvePackage(ctx context.Context, element Package, rc *ResolveContext) error {
	//TODO implement me
	panic("implement me")
}

func (js *JavaScriptResolver) resolveFunction(ctx context.Context, element Function, rc *ResolveContext) error {
	//TODO implement me
	panic("implement me")
}

func (js *JavaScriptResolver) resolveMethod(ctx context.Context, element Method, rc *ResolveContext) error {
	//TODO implement me
	panic("implement me")
}

func (js *JavaScriptResolver) resolveClass(ctx context.Context, element Class, rc *ResolveContext) error {
	//TODO implement me
	panic("implement me")
}

func (js *JavaScriptResolver) resolveVariable(ctx context.Context, element Variable, rc *ResolveContext) error {
	//TODO implement me
	panic("implement me")
}

func (js *JavaScriptResolver) resolveInterface(ctx context.Context, element Interface, rc *ResolveContext) error {
	//TODO implement me
	panic("implement me")
}
