package resolver

import (
	"codebase-indexer/pkg/codegraph/types"
	"codebase-indexer/pkg/codegraph/utils"
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

type PythonResolver struct {
}

var _ ElementResolver = &PythonResolver{}

func (py *PythonResolver) Resolve(ctx context.Context, element Element, rc *ResolveContext) ([]Element, error) {
	return resolve(ctx, py, element, rc)
}

func (py *PythonResolver) resolveImport(ctx context.Context, element *Import, rc *ResolveContext) ([]Element, error) {
	if element.Name == types.EmptyString {
		return nil, fmt.Errorf("import is empty")
	}

	element.FilePaths = []string{}
	importName := element.Name

	importPath := strings.ReplaceAll(importName, ".", "/")
	pj := rc.ProjectInfo

	elements := []Element{element}

	if pj.IsEmpty() {
		// TODO
		fmt.Println("not support project file list, use default resolve")
		element.FilePaths = []string{importPath}
		return elements, nil
	}

	// 处理相对导入
	if strings.HasPrefix(importName, ".") {
		// 计算当前文件相对于 sourceRoot 的路径
		currentRelPath, _ := filepath.Rel(pj.GetSourceRoot(), rc.SourceFile.Path)
		currentDir := utils.ToUnixPath(filepath.Dir(currentRelPath))
		dots := strings.Count(importName, ".")
		modulePath := strings.TrimPrefix(importName, strings.Repeat(".", dots))

		// 向上移动目录层级
		dir := currentDir
		for i := 0; i < dots-1; i++ {
			dir = utils.ToUnixPath(filepath.Dir(dir))
		}

		// 构建完整路径
		if modulePath != types.EmptyString {
			modulePath = strings.ReplaceAll(modulePath, ".", "/")
			dir = utils.ToUnixPath(filepath.Join(dir, modulePath))
		}

		// 检查是否为包或模块
		for _, ext := range []string{"__init__.py", ".py"} {
			fullPath := utils.ToUnixPath(filepath.Join(pj.GetSourceRoot(), dir, ext))
			if pj.ContainsFileIndex(fullPath) {
				element.FilePaths = append(element.FilePaths, fullPath)
			}
		}

		if len(element.FilePaths) > 0 {
			return elements, nil
		}

		return nil, fmt.Errorf("cannot find file which relative import belongs to: %s", importName)
	}

	// 处理绝对导入
	foundPaths := []string{}

	// 检查是否为包或模块
	for _, ext := range []string{"__init__.py", ".py"} {
		fullPath := utils.ToUnixPath(filepath.Join(importPath, ext))
		if pj.ContainsFileIndex(fullPath) {
			foundPaths = append(foundPaths, fullPath)
		}
		fullPath = utils.ToUnixPath(filepath.Join(importPath + ext))
		if pj.ContainsFileIndex(fullPath) {
			foundPaths = append(foundPaths, fullPath)
		}
	}

	element.FilePaths = foundPaths
	if len(element.FilePaths) > 0 {
		return elements, nil
	}

	return nil, fmt.Errorf("cannot find file which abs import belongs to: %s", importName)
}

func (py *PythonResolver) resolvePackage(ctx context.Context, element *Package, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}

func (py *PythonResolver) resolveFunction(ctx context.Context, element *Function, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}

func (py *PythonResolver) resolveMethod(ctx context.Context, element *Method, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}

func (py *PythonResolver) resolveClass(ctx context.Context, element *Class, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}

func (py *PythonResolver) resolveVariable(ctx context.Context, element *Variable, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}

func (py *PythonResolver) resolveInterface(ctx context.Context, element *Interface, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}

func (py *PythonResolver) resolveCall(ctx context.Context, element *Call, rc *ResolveContext) ([]Element, error) {
	//TODO implement me
	panic("implement me")
}
