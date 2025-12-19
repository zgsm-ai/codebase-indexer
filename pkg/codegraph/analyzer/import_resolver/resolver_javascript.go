package import_resolver

import (
	"codebase-indexer/pkg/codegraph/resolver"
	"codebase-indexer/pkg/codegraph/workspace"
	"context"
	"path/filepath"
	"strings"
)

// JSImportPathResolver JavaScript/TypeScript语言路径解析器
type JSImportPathResolver struct {
	searcher    *FileSearcher
	pathAliases map[string]string // 路径别名映射：别名 -> 实际路径
}

// NewJSImportPathResolver 创建JS/TS解析器
func NewJSImportPathResolver(searcher *FileSearcher) *JSImportPathResolver {
	return &JSImportPathResolver{
		searcher:    searcher,
		pathAliases: make(map[string]string),
	}
}

// SetPathAliases 设置路径别名（从tsconfig.json或jsconfig.json读取）
func (r *JSImportPathResolver) SetPathAliases(aliases map[string]string) {
	r.pathAliases = aliases
}

// ResolveImportPath 解析JS/TS导入路径
func (r *JSImportPathResolver) ResolveImportPath(
	ctx context.Context,
	imp *resolver.Import,
	project *workspace.Project,
) ([]string, error) {
	source := imp.Source

	// 1. 检查路径别名（如 @/components, ~/utils）
	if resolved, ok := r.resolvePathAlias(source); ok {
		source = resolved
	}

	// 2. 相对路径解析
	if strings.HasPrefix(source, "./") || strings.HasPrefix(source, "../") {
		return r.resolveRelativePath(imp, project), nil
	}

	// 3. 绝对路径（项目内）
	// node_modules 的包已被 classifier 过滤，这里只处理项目内的绝对路径
	return r.resolveAbsolutePath(source), nil
}

// resolvePathAlias 解析路径别名
func (r *JSImportPathResolver) resolvePathAlias(source string) (string, bool) {
	// 检查是否匹配路径别名
	for alias, realPath := range r.pathAliases {
		if strings.HasPrefix(source, alias) {
			// 替换别名为实际路径
			resolved := strings.Replace(source, alias, realPath, 1)
			return resolved, true
		}
	}
	return "", false
}

// resolveRelativePath 解析相对路径
func (r *JSImportPathResolver) resolveRelativePath(imp *resolver.Import, project *workspace.Project) []string {
	// 获取当前文件的相对路径
	currentFilePath := imp.Path

	// 如果是绝对路径，转换为相对路径
	if filepath.IsAbs(currentFilePath) {
		relPath, err := filepath.Rel(project.Path, currentFilePath)
		if err != nil {
			return nil
		}
		currentFilePath = relPath
	}

	// 获取当前文件所在目录
	currentDir := filepath.Dir(currentFilePath)

	// 拼接相对导入路径
	basePath := filepath.Join(currentDir, imp.Source)
	basePath = filepath.Clean(basePath)

	// 尝试多种扩展名
	extensions := []string{"", ".js", ".jsx", ".ts", ".tsx", ".mjs", ".json"}
	var candidates []string

	for _, ext := range extensions {
		// 直接文件
		candidates = append(candidates, basePath+ext)
		// index 文件
		if ext != "" {
			candidates = append(candidates, filepath.Join(basePath, "index"+ext))
		}
	}

	// 验证存在
	var result []string
	for _, path := range candidates {
		if r.searcher.FileExists(path) {
			result = append(result, path)
			// 找到第一个就返回（避免重复）
			break
		}
	}

	return result
}

// resolveAbsolutePath 解析绝对路径
func (r *JSImportPathResolver) resolveAbsolutePath(source string) []string {
	// 1. 首先尝试直接路径
	extensions := []string{"", ".js", ".jsx", ".ts", ".tsx", ".mjs", ".json"}
	var candidates []string

	for _, ext := range extensions {
		candidates = append(candidates, source+ext)
		if ext != "" {
			candidates = append(candidates, filepath.Join(source, "index"+ext))
		}
	}

	// 验证存在
	var result []string
	for _, path := range candidates {
		if r.searcher.FileExists(path) {
			result = append(result, path)
			break
		}
	}

	// 2. 如果直接路径找不到，使用文件索引查找
	if len(result) == 0 {
		// 提取文件名（最后一个路径分隔符后的部分）
		fileName := filepath.Base(source)

		// 尝试每种扩展名
		for _, ext := range extensions {
			if ext == "" {
				continue
			}
			searchName := fileName + ext
			matchedPaths := r.searcher.FindFilesByName(searchName)

			// 过滤匹配的路径（检查路径是否以 source 结尾）
			for _, path := range matchedPaths {
				// 将路径分隔符统一为正斜杠进行比较
				normalizedPath := filepath.ToSlash(path)
				normalizedSource := filepath.ToSlash(source)

				// 检查路径是否以 source 结尾
				if strings.HasSuffix(normalizedPath, normalizedSource+ext) ||
					strings.HasSuffix(normalizedPath, normalizedSource+"/index"+ext) {
					result = append(result, path)
				}
			}

			if len(result) > 0 {
				break
			}
		}
	}

	return result
}
