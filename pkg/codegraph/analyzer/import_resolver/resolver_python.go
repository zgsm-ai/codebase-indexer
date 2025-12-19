package import_resolver

import (
	"codebase-indexer/pkg/codegraph/resolver"
	"codebase-indexer/pkg/codegraph/workspace"
	"context"
	"path/filepath"
	"strings"
)

// PythonImportPathResolver Python语言路径解析器
type PythonImportPathResolver struct {
	searcher *FileSearcher
}

// NewPythonImportPathResolver 创建Python解析器
func NewPythonImportPathResolver(searcher *FileSearcher) *PythonImportPathResolver {
	return &PythonImportPathResolver{
		searcher: searcher,
	}
}

// ResolveImportPath 解析Python导入路径
func (r *PythonImportPathResolver) ResolveImportPath(
	ctx context.Context,
	imp *resolver.Import,
	project *workspace.Project,
) ([]string, error) {
	var candidates []string

	// 1. 处理相对导入
	if strings.HasPrefix(imp.Source, ".") {
		candidates = r.resolveRelativeImport(imp)
	} else {
		// 2. 处理绝对导入
		candidates = r.resolveAbsoluteImport(imp)
	}

	// 3. 验证并返回存在的文件
	var result []string
	for _, path := range candidates {
		if r.searcher.FileExists(path) {
			result = append(result, path)
		}
	}

	return result, nil
}

// resolveRelativeImport 解析相对导入
func (r *PythonImportPathResolver) resolveRelativeImport(imp *resolver.Import) []string {
	// 计算相对路径层级（点的数量）
	level := 0
	source := imp.Source
	for strings.HasPrefix(source, ".") {
		level++
		source = strings.TrimPrefix(source, ".")
	}

	// 基于当前文件位置向上跳转
	currentDir := filepath.Dir(imp.Path)
	for i := 1; i < level; i++ { // 从1开始，因为第一个点表示当前目录
		currentDir = filepath.Dir(currentDir)
	}

	// 构建模块路径
	if source != "" {
		modulePath := strings.ReplaceAll(source, ".", string(filepath.Separator))
		currentDir = filepath.Join(currentDir, modulePath)
	}

	// 生成候选路径
	return []string{
		currentDir + ".py",
		filepath.Join(currentDir, "__init__.py"),
	}
}

// resolveAbsoluteImport 解析绝对导入
func (r *PythonImportPathResolver) resolveAbsoluteImport(imp *resolver.Import) []string {
	// 将包名转为路径
	modulePath := strings.ReplaceAll(imp.Source, ".", string(filepath.Separator))

	return []string{
		modulePath + ".py",
		filepath.Join(modulePath, "__init__.py"),
	}
}
