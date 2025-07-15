package resolver

import (
	"codebase-indexer/pkg/codegraph/types"
	"codebase-indexer/pkg/codegraph/utils"
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

type JavaResolver struct {
}

var _ ElementResolver = &JavaResolver{}

func (j *JavaResolver) Resolve(ctx context.Context, element Element, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}

func (j *JavaResolver) resolveImport(ctx context.Context, element *Import, rc *ResolveContext) ([]Element, error) {
	if element.Name == types.EmptyString {
		return nil, fmt.Errorf("import is empty")
	}

	element.FilePaths = []string{}
	importName := element.Name

	// 处理类导入
	classPath := strings.ReplaceAll(importName, ".", "/") + ".java"
	pj := rc.ProjectInfo
	fullPath := utils.ToUnixPath(filepath.Join(pj.GetSourceRoot(), classPath))

	elements := []Element{element}

	if pj.IsEmpty() {
		// TODO logger
		fmt.Println("not support project file list, use default resolve")
		element.FilePaths = []string{fullPath}
		return elements, nil
	}

	// 处理静态导入
	if strings.HasPrefix(importName, "static ") {
		importName = strings.TrimPrefix(importName, "static ")
	}

	// 处理包导入
	if strings.HasSuffix(importName, ".*") {
		pkgPath := strings.ReplaceAll(strings.TrimSuffix(importName, ".*"), ".", "/")
		fullPkgPath := utils.ToUnixPath(filepath.Join(pj.GetSourceRoot(), pkgPath))
		files := pj.FindFilesInDirIndex(fullPkgPath, ".java")
		element.FilePaths = files
		if len(element.FilePaths) == 0 {
			return nil, fmt.Errorf("cannot find file which package belongs to: %s", importName)
		}
		return elements, nil
	}

	element.FilePaths = pj.FindMatchingFiles(fullPath)

	if len(element.FilePaths) == 0 {
		return nil, fmt.Errorf("cannot find file which import belongs to: %s", importName)
	}

	return elements, nil
}

func (j *JavaResolver) resolvePackage(ctx context.Context, element *Package, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}

func (j *JavaResolver) resolveFunction(ctx context.Context, element *Function, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}

func (j *JavaResolver) resolveMethod(ctx context.Context, element *Method, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}

func (j *JavaResolver) resolveClass(ctx context.Context, element *Class, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}

func (j *JavaResolver) resolveVariable(ctx context.Context, element *Variable, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}

func (j *JavaResolver) resolveInterface(ctx context.Context, element *Interface, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}
