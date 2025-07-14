package resolver

import (
	"codebase-indexer/pkg/codegraph/utils"
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

type JavaResolver struct {
}

var _ ElementResolver = &JavaResolver{}

func (j *JavaResolver) Resolve(ctx context.Context, element Element, rc *ResolveContext) error {
	//TODO implement me
	panic("implement me")
}

func (j *JavaResolver) resolveImport(ctx context.Context, element Import, rc *ResolveContext) error {
	if element.Name == EmptyString {
		return fmt.Errorf("import is empty")
	}

	element.FilePaths = []string{}
	importName := element.Name

	// 处理类导入
	classPath := strings.ReplaceAll(importName, ".", "/") + ".java"
	pc := rc.ProjectInfo
	fullPath := utils.ToUnixPath(filepath.Join(pc.SourceRoot, classPath))

	if len(pc.fileSet) == 0 {
		// TODO logger
		fmt.Println("not support project file list, use default resolve")
		element.FilePaths = []string{fullPath}
		return nil
	}

	// 处理静态导入
	if strings.HasPrefix(importName, "static ") {
		importName = strings.TrimPrefix(importName, "static ")
	}

	// 处理包导入
	if strings.HasSuffix(importName, ".*") {
		pkgPath := strings.ReplaceAll(strings.TrimSuffix(importName, ".*"), ".", "/")
		fullPkgPath := utils.ToUnixPath(filepath.Join(pc.SourceRoot, pkgPath))
		files := pc.findFilesInDirIndex(fullPkgPath, ".java")
		element.FilePaths = files
		if len(element.FilePaths) == 0 {
			return fmt.Errorf("cannot find file which package belongs to: %s", importName)
		}
		return nil
	}

	element.FilePaths = pc.findMatchingFiles(fullPath)

	if len(element.FilePaths) == 0 {
		return fmt.Errorf("cannot find file which import belongs to: %s", importName)
	}

	return nil
}

func (j *JavaResolver) resolvePackage(ctx context.Context, element Package, rc *ResolveContext) error {
	//TODO implement me
	panic("implement me")
}

func (j *JavaResolver) resolveFunction(ctx context.Context, element Function, rc *ResolveContext) error {
	//TODO implement me
	panic("implement me")
}

func (j *JavaResolver) resolveMethod(ctx context.Context, element Method, rc *ResolveContext) error {
	//TODO implement me
	panic("implement me")
}

func (j *JavaResolver) resolveClass(ctx context.Context, element Class, rc *ResolveContext) error {
	//TODO implement me
	panic("implement me")
}

func (j *JavaResolver) resolveVariable(ctx context.Context, element Variable, rc *ResolveContext) error {
	//TODO implement me
	panic("implement me")
}

func (j *JavaResolver) resolveInterface(ctx context.Context, element Interface, rc *ResolveContext) error {
	//TODO implement me
	panic("implement me")
}
