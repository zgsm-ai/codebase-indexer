package resolver

import (
	"context"
	"fmt"
	"golang.org/x/tools/go/packages"
	"strings"
)

type GoResolver struct {
}

var _ ElementResolver = &GoResolver{}

func (r *GoResolver) Resolve(ctx context.Context, element Element, rc *ResolveContext) error {
	return resolve(ctx, r, element, rc)
}

func (r *GoResolver) resolveImport(ctx context.Context, element Import, rc *ResolveContext) error {
	if element.Name == EmptyString {
		return fmt.Errorf("import is empty")
	}

	element.FilePaths = []string{}
	importName := element.Name

	// 标准库，直接排除
	if yes, _ := r.isStandardLibrary(importName); yes {
		fmt.Printf("import_resolver import %s is stantdard lib, skip\n", importName)
		return nil
	}
	pc := rc.ProjectInfo
	// 移除mod，如果有
	relPath := importName
	if strings.HasPrefix(importName, pc.SourceRoot) {
		relPath = strings.TrimPrefix(importName, pc.SourceRoot+"/")
	}

	if len(pc.fileSet) == 0 {
		fmt.Println("not support project file list, use default resolve")
		element.FilePaths = []string{relPath}
		return nil
	}

	// 尝试匹配 .go 文件
	relPathWithExt := relPath + ".go"
	if pc.containsFileIndex(relPathWithExt) {
		element.FilePaths = []string{relPathWithExt}
		return nil
	}

	// 匹配包目录下所有 .go 文件

	filesInDir := pc.findFilesInDirIndex(relPath, ".go")
	if len(filesInDir) > 0 {
		element.FilePaths = append(element.FilePaths, filesInDir...)
	}

	if len(element.FilePaths) > 0 {
		return nil
	}

	return fmt.Errorf("cannot find file which import belongs to: %s", importName)
}

func (r *GoResolver) resolvePackage(ctx context.Context, element Package, rc *ResolveContext) error {
	//TODO implement me
	panic("implement me")
}

func (r *GoResolver) resolveFunction(ctx context.Context, element Function, rc *ResolveContext) error {
	//TODO implement me
	panic("implement me")
}

func (r *GoResolver) resolveMethod(ctx context.Context, element Method, rc *ResolveContext) error {
	//TODO implement me
	panic("implement me")
}

func (r *GoResolver) resolveClass(ctx context.Context, element Class, rc *ResolveContext) error {
	//TODO implement me
	panic("implement me")
}

func (r *GoResolver) resolveVariable(ctx context.Context, element Variable, rc *ResolveContext) error {
	//TODO implement me
	panic("implement me")
}

func (r *GoResolver) resolveInterface(ctx context.Context, element Interface, rc *ResolveContext) error {
	//TODO implement me
	panic("implement me")
}

func (g *GoResolver) isStandardLibrary(pkgPath string) (bool, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName,
	}

	pkgs, err := packages.Load(cfg, pkgPath)
	if err != nil {
		return false, fmt.Errorf("import_resolver load package: %v", err)
	}

	if len(pkgs) == 0 {
		return false, fmt.Errorf("import_resolver package not found: %s", pkgPath)
	}

	// 标准库包的PkgPath以"internal/"或非模块路径开头
	return !strings.Contains(pkgs[0].PkgPath, "."), nil
}
