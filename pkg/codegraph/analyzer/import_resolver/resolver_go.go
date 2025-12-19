package import_resolver

import (
	"codebase-indexer/pkg/codegraph/resolver"
	"codebase-indexer/pkg/codegraph/workspace"
	"context"
	"path/filepath"
	"strings"
)

// GoImportPathResolver Go语言路径解析器
type GoImportPathResolver struct {
	searcher *FileSearcher
}

// NewGoImportPathResolver 创建Go解析器
func NewGoImportPathResolver(searcher *FileSearcher) *GoImportPathResolver {
	return &GoImportPathResolver{
		searcher: searcher,
	}
}

// ResolveImportPath 解析Go导入路径
func (r *GoImportPathResolver) ResolveImportPath(
	ctx context.Context,
	imp *resolver.Import,
	project *workspace.Project,
) ([]string, error) {
	// 1. 去除模块前缀
	importPath := imp.Source
	for _, module := range project.GoModules {
		if module != "" && strings.HasPrefix(importPath, module) {
			importPath = strings.TrimPrefix(importPath, module+"/")
			break
		}
	}

	// 2. 优先查找vendor目录
	vendorPath := filepath.Join("vendor", importPath)
	files, err := r.searcher.ListFilesInDir(vendorPath, ".go")
	if err == nil && len(files) > 0 {
		// 过滤测试文件
		result := r.filterTestFiles(files)
		if len(result) > 0 {
			return result, nil
		}
	}

	// 3. 查找项目目录
	dirPath := filepath.Clean(importPath)
	files, err = r.searcher.ListFilesInDir(dirPath, ".go")
	if err != nil {
		return nil, err
	}

	// 4. 过滤测试文件
	return r.filterTestFiles(files), nil
}

// filterTestFiles 过滤测试文件
func (r *GoImportPathResolver) filterTestFiles(files []string) []string {
	var result []string
	for _, file := range files {
		if !strings.HasSuffix(file, "_test.go") {
			result = append(result, file)
		}
	}
	return result
}
