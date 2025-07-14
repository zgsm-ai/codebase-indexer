package resolver

import (
	"codebase-indexer/pkg/codegraph/utils"
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

type CResolver struct {
}

var _ ElementResolver = &CResolver{}

func (r *CResolver) Resolve(ctx context.Context, element Element, rc *ResolveContext) error {
	return resolve(ctx, r, element, rc)
}

func (r *CResolver) resolveImport(ctx context.Context, element Import, rc *ResolveContext) error {
	if element.Name == EmptyString {
		return fmt.Errorf("import is empty")
	}

	element.FilePaths = []string{}
	importName := element.Name

	// 处理系统头文件
	if strings.HasPrefix(importName, "<") && strings.HasSuffix(importName, ">") {
		return nil // 系统头文件，不映射到项目文件
	}

	// 移除引号
	headerFile := strings.Trim(importName, "\"")

	pc := rc.ProjectInfo
	if len(pc.fileSet) == 0 {
		fmt.Println("not support project file list, use default resolve")
		element.FilePaths = []string{headerFile}
		return nil
	}

	foundPaths := []string{}

	// 相对路径导入
	if strings.HasPrefix(headerFile, ".") {
		// 计算当前文件相对于 SourceRoot 的路径
		currentRelPath, _ := filepath.Rel(pc.SourceRoot, rc.SourceFile.Path)
		currentDir := utils.ToUnixPath(filepath.Dir(currentRelPath))
		relPath := utils.ToUnixPath(filepath.Join(currentDir, headerFile))
		fullPath := utils.ToUnixPath(filepath.Join(pc.SourceRoot, relPath))
		if pc.containsFileIndex(fullPath) {
			foundPaths = append(foundPaths, fullPath)
		}
	}

	// 在源目录中查找
	for _, relDir := range pc.Dirs {
		fullPath := utils.ToUnixPath(filepath.Join(relDir, headerFile))
		if pc.containsFileIndex(fullPath) {
			foundPaths = append(foundPaths, fullPath)
		}
	}

	element.FilePaths = foundPaths
	if len(element.FilePaths) > 0 {
		return nil
	}

	return fmt.Errorf("cannot find file which import belongs to: %s", importName)
}

func (r *CResolver) resolvePackage(ctx context.Context, element Package, rc *ResolveContext) error {
	//TODO implement me
	panic("implement me")
}

func (r *CResolver) resolveFunction(ctx context.Context, element Function, rc *ResolveContext) error {
	//TODO implement me
	panic("implement me")
}

func (r *CResolver) resolveMethod(ctx context.Context, element Method, rc *ResolveContext) error {
	//TODO implement me
	panic("implement me")
}

func (r *CResolver) resolveClass(ctx context.Context, element Class, rc *ResolveContext) error {
	//TODO implement me
	panic("implement me")
}

func (r *CResolver) resolveVariable(ctx context.Context, element Variable, rc *ResolveContext) error {
	//TODO implement me
	panic("implement me")
}

func (r *CResolver) resolveInterface(ctx context.Context, element Interface, rc *ResolveContext) error {
	//TODO implement me
	panic("implement me")
}
