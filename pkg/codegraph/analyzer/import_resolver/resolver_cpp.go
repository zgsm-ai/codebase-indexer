package import_resolver

import (
	"codebase-indexer/pkg/codegraph/resolver"
	"codebase-indexer/pkg/codegraph/workspace"
	"context"
	"path/filepath"
	"strings"
)

// CppImportPathResolver C/C++语言路径解析器
type CppImportPathResolver struct {
	searcher *FileSearcher
}

// NewCppImportPathResolver 创建C/C++解析器
func NewCppImportPathResolver(searcher *FileSearcher) *CppImportPathResolver {
	return &CppImportPathResolver{
		searcher: searcher,
	}
}

// ResolveImportPath 解析C/C++导入路径
func (r *CppImportPathResolver) ResolveImportPath(
	ctx context.Context,
	imp *resolver.Import,
	project *workspace.Project,
) ([]string, error) {
	// 使用Source字段（导入源路径），去除引号
	source := imp.Source
	if source == "" {
		source = imp.Name // 降级使用Name
	}
	headerName := strings.Trim(source, `"'<>`)

	// 1. 相对于当前文件查找
	if strings.Contains(headerName, "/") || strings.Contains(headerName, "\\") {
		currentDir := filepath.Dir(imp.Path)
		fullPath := filepath.Join(currentDir, headerName)
		fullPath = filepath.Clean(fullPath)

		if r.searcher.FileExists(fullPath) {
			return []string{fullPath}, nil
		}
	}

	// 2. 在 include 目录中查找
	includeDirs := r.getIncludeDirs()
	for _, includeDir := range includeDirs {
		fullPath := filepath.Join(includeDir, headerName)
		fullPath = filepath.Clean(fullPath)

		if r.searcher.FileExists(fullPath) {
			return []string{fullPath}, nil
		}
	}

	// 3. 递归查找（作为后备）
	files := r.searcher.FindFilesByName(filepath.Base(headerName))
	if len(files) > 0 {
		// 过滤：只返回与路径匹配的文件
		if strings.Contains(headerName, "/") || strings.Contains(headerName, "\\") {
			var filtered []string
			for _, file := range files {
				if strings.Contains(file, headerName) || strings.HasSuffix(file, headerName) {
					filtered = append(filtered, file)
				}
			}
			if len(filtered) > 0 {
				return filtered, nil
			}
		}
		return files, nil
	}

	return nil, nil
}

// getIncludeDirs 获取include目录列表
func (r *CppImportPathResolver) getIncludeDirs() []string {
	return []string{
		".",
		"include",
		"src",
		"lib",
		"libs",
	}
}
