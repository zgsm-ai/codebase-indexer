package import_resolver

import (
	"codebase-indexer/pkg/codegraph/resolver"
	"codebase-indexer/pkg/codegraph/workspace"
	"context"
	"path/filepath"
	"strings"
)

// JavaImportPathResolver Java语言路径解析器
type JavaImportPathResolver struct {
	searcher *FileSearcher
}

// NewJavaImportPathResolver 创建Java解析器
func NewJavaImportPathResolver(searcher *FileSearcher) *JavaImportPathResolver {
	return &JavaImportPathResolver{
		searcher: searcher,
	}
}

// ResolveImportPath 解析Java导入路径
func (r *JavaImportPathResolver) ResolveImportPath(
	ctx context.Context,
	imp *resolver.Import,
	project *workspace.Project,
) ([]string, error) {
	// 使用Source字段（导入源路径）
	source := imp.Source
	if source == "" {
		source = imp.Name // 降级使用Name
	}

	// 1. 处理静态导入: import static pkg.Class.member 或 import static pkg.Class.*
	if strings.HasPrefix(source, "static ") {
		source = strings.TrimPrefix(source, "static ")
		// 提取类名（去掉最后的.member或.*）
		lastDot := strings.LastIndex(source, ".")
		if lastDot > 0 {
			source = source[:lastDot] // 只保留到类名
		}
	}

	// 2. 转换包名为路径
	packagePath := strings.ReplaceAll(source, ".", string(filepath.Separator))

	// 3. 检查是否为通配符导入
	if strings.HasSuffix(source, ".*") {
		return r.resolveWildcardImport(packagePath), nil
	}

	// 4. 普通类导入
	return r.resolveSingleClassImport(packagePath), nil
}

// resolveSingleClassImport 解析单个类导入
func (r *JavaImportPathResolver) resolveSingleClassImport(packagePath string) []string {
	// 1. 首先尝试标准目录结构（快速路径）
	sourceDirs := []string{
		"src/main/java",
		"src/test/java",
		"src",
	}

	var result []string
	for _, srcDir := range sourceDirs {
		fullPath := filepath.Join(srcDir, packagePath+".java")

		if r.searcher.FileExists(fullPath) {
			result = append(result, fullPath)
			return result // 找到第一个就返回
		}
	}

	// 2. 如果标准路径找不到，使用文件索引查找
	// 提取文件名（最后一个路径分隔符后的部分）
	fileName := filepath.Base(packagePath) + ".java"
	matchedPaths := r.searcher.FindFilesByName(fileName)

	// 3. 过滤匹配的路径（检查包路径是否匹配）
	for _, path := range matchedPaths {
		// 将路径分隔符统一为正斜杠进行比较
		normalizedPath := filepath.ToSlash(path)
		normalizedPackage := filepath.ToSlash(packagePath)

		// 检查路径是否以包路径结尾
		if strings.HasSuffix(normalizedPath, normalizedPackage+".java") {
			result = append(result, path)
		}
	}

	return result
}

// resolveWildcardImport 解析通配符导入
func (r *JavaImportPathResolver) resolveWildcardImport(packagePath string) []string {
	// 移除 /*
	packagePath = strings.TrimSuffix(packagePath, "/*")

	// 1. 首先尝试标准目录结构
	sourceDirs := []string{
		"src/main/java",
		"src/test/java",
		"src",
	}

	var result []string
	for _, srcDir := range sourceDirs {
		dirPath := filepath.Join(srcDir, packagePath)

		files, err := r.searcher.ListFilesInDir(dirPath, ".java")
		if err == nil && len(files) > 0 {
			result = append(result, files...)
		}
	}

	// 2. 如果标准路径找不到，尝试查找所有匹配包路径的目录
	if len(result) == 0 {
		// 获取所有 .java 文件
		allFiles := r.searcher.FindFilesByExtension(".java")

		// 过滤出在目标包路径下的文件
		normalizedPackage := filepath.ToSlash(packagePath)
		for _, file := range allFiles {
			normalizedFile := filepath.ToSlash(file)
			// 检查文件是否在目标包目录下（但不在子目录）
			if strings.Contains(normalizedFile, normalizedPackage+"/") {
				// 确保是直接子文件，不是孙子目录的文件
				parts := strings.Split(normalizedFile, normalizedPackage+"/")
				if len(parts) == 2 && !strings.Contains(parts[1], "/") {
					result = append(result, file)
				}
			}
		}
	}

	return result
}
