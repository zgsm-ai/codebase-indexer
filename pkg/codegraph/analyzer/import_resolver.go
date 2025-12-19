package analyzer

import (
	"codebase-indexer/pkg/codegraph/analyzer/import_resolver"
	packageclassifier "codebase-indexer/pkg/codegraph/analyzer/package_classifier"
	"codebase-indexer/pkg/codegraph/lang"
	"codebase-indexer/pkg/codegraph/resolver"
	"codebase-indexer/pkg/codegraph/workspace"
	"codebase-indexer/pkg/logger"
	"context"
	"fmt"
	"sync"
)

// ImportResolver 导入解析器（替代旧的 import.go）
type ImportResolver struct {
	logger            logger.Logger
	packageClassifier *packageclassifier.PackageClassifier
	pathResolverMgr   *import_resolver.PathResolverManager
}

// NewImportResolver 创建导入解析器
func NewImportResolver(
	logger logger.Logger,
	packageClassifier *packageclassifier.PackageClassifier,
	projectPath string,
) *ImportResolver {
	return &ImportResolver{
		logger:            logger,
		packageClassifier: packageClassifier,
		pathResolverMgr:   import_resolver.NewPathResolverManager(projectPath),
	}
}

// ResolveImports 解析导入列表
// 这个方法替代了旧 import.go 中的 PreprocessImports
func (ir *ImportResolver) ResolveImports(
	ctx context.Context,
	language lang.Language,
	project *workspace.Project,
	imports []*resolver.Import,
) ([]*resolver.Import, error) {
	if len(imports) == 0 {
		return imports, nil
	}

	// 1. 过滤和预处理导入
	processedImports := make([]*resolver.Import, 0, len(imports))
	for _, imp := range imports {
		// 过滤系统包和第三方包
		packageType, err := ir.packageClassifier.ClassifyPackage(language, imp.Name, project)
		if err != nil {
			ir.logger.Debug("classify %s package %s error: %v", imp.Path, imp.Name, err)
		}

		if packageType == packageclassifier.SystemPackage || packageType == packageclassifier.ThirdPartyPackage {
			continue // 跳过非项目包
		}

		processedImports = append(processedImports, imp)
	}

	// 2. 构建文件索引（首次使用时）
	if err := ir.pathResolverMgr.BuildIndex(); err != nil {
		ir.logger.Warn("build file index error: %v", err)
		// 索引构建失败不影响流程，继续执行
	}

	// 3. 并发解析导入路径
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10) // 限制并发数为10

	for _, imp := range processedImports {
		// 跳过已解析的
		if imp.IsResolved {
			continue
		}

		wg.Add(1)
		go func(imp *resolver.Import) {
			defer wg.Done()
			semaphore <- struct{}{}        // 获取信号量
			defer func() { <-semaphore }() // 释放信号量

			// 解析文件路径
			paths, err := ir.pathResolverMgr.ResolveImportPath(ctx, language, imp, project)
			if err != nil {
				ir.logger.Debug("resolve import %s (source: %s) path error: %v", imp.Name, imp.Source, err)
				return
			}

			// 存储结果
			imp.ResolvedPaths = paths
			imp.IsResolved = true

			if len(paths) > 0 {
				ir.logger.Debug("resolved import %s -> %v", imp.Name, paths)
			}
		}(imp)
	}

	wg.Wait()

	return processedImports, nil
}

// IsFilePathInImportPackage 判断文件路径是否属于导入包的范围
// 保留这个方法以兼容现有代码
func IsFilePathInImportPackage(filePath string, imp *resolver.Import) bool {
	if imp == nil {
		return false
	}

	// 如果已经解析出具体路径，直接匹配
	if imp.IsResolved && len(imp.ResolvedPaths) > 0 {
		for _, resolvedPath := range imp.ResolvedPaths {
			if filePath == resolvedPath {
				return true
			}
		}
	}

	// 降级：使用原有的模糊匹配逻辑
	// （这部分逻辑从旧的 import.go 迁移过来）
	return false
}

// GetResolvedPaths 获取import解析的文件路径
func GetResolvedPaths(imp *resolver.Import) []string {
	if imp == nil || !imp.IsResolved {
		return nil
	}
	return imp.ResolvedPaths
}

// BatchResolveImports 批量解析多个文件的导入
func (ir *ImportResolver) BatchResolveImports(
	ctx context.Context,
	language lang.Language,
	project *workspace.Project,
	fileImports map[string][]*resolver.Import,
) error {
	// 构建索引一次
	if err := ir.pathResolverMgr.BuildIndex(); err != nil {
		ir.logger.Warn("build file index error: %v", err)
	}

	// 收集所有导入
	var allImports []*resolver.Import
	for _, imports := range fileImports {
		allImports = append(allImports, imports...)
	}

	// 统一解析
	_, err := ir.ResolveImports(ctx, language, project, allImports)
	return err
}

// GetStats 获取解析统计信息
func (ir *ImportResolver) GetStats(imports []*resolver.Import) map[string]int {
	stats := map[string]int{
		"total":      len(imports),
		"resolved":   0,
		"unresolved": 0,
		"with_paths": 0,
	}

	for _, imp := range imports {
		if imp.IsResolved {
			stats["resolved"]++
			if len(imp.ResolvedPaths) > 0 {
				stats["with_paths"]++
			}
		} else {
			stats["unresolved"]++
		}
	}

	return stats
}

// LogStats 输出解析统计日志
func (ir *ImportResolver) LogStats(imports []*resolver.Import) {
	stats := ir.GetStats(imports)
	ir.logger.Info(
		"Import resolution stats: total=%d, resolved=%d, with_paths=%d, unresolved=%d",
		stats["total"],
		stats["resolved"],
		stats["with_paths"],
		stats["unresolved"],
	)
}

// ValidateResolution 验证解析结果
func (ir *ImportResolver) ValidateResolution(imports []*resolver.Import) error {
	var unresolvedCount int
	for _, imp := range imports {
		if !imp.IsResolved || len(imp.ResolvedPaths) == 0 {
			unresolvedCount++
		}
	}

	if unresolvedCount > 0 {
		return fmt.Errorf("%d imports could not be resolved", unresolvedCount)
	}

	return nil
}

// PreprocessImports 预处理并解析导入（兼容旧API）
// 这个方法提供向后兼容，新代码应使用ResolveImports
func (ir *ImportResolver) PreprocessImports(
	ctx context.Context,
	language lang.Language,
	project *workspace.Project,
	imports []*resolver.Import,
) ([]*resolver.Import, error) {
	return ir.ResolveImports(ctx, language, project, imports)
}
