package analyzer

import (
	packageclassifier "codebase-indexer/pkg/codegraph/analyzer/package_classifier"
	"codebase-indexer/pkg/codegraph/proto/codegraphpb"
	"codebase-indexer/pkg/codegraph/types"
	"codebase-indexer/pkg/codegraph/workspace"
	"context"
	"path/filepath"
	"strings"

	"codebase-indexer/pkg/codegraph/lang"
	"codebase-indexer/pkg/codegraph/resolver"
)

// PreprocessImports 预处理导入语句，处理 Import.Name Import.Source 两个字段
// 1、python、js、ts 导入相对路径处理，转为绝对路径。
// 2、go同包不同文件处理；大小写处理；import部分移除模块名。
// 3、c/cpp 简单处理，只关心项目内的源码，根据当前文件的using部分，再结合符号名；
// 4、python、ts、go 别名处理；
// 5、作用域。
// 6、将 / 统一转为 .，方便后续处理。
func (da *DependencyAnalyzer) PreprocessImports(ctx context.Context,
	language lang.Language, projectInfo *workspace.Project, imports []*resolver.Import) ([]*resolver.Import, error) {
	processedImports := make([]*resolver.Import, 0, len(imports))
	for _, imp := range imports {
		// TODO 过滤掉标准库、第三方库等非项目的库
		if i := da.processImportByLanguage(imp, language, projectInfo); i != nil {
			processedImports = append(processedImports, i)
		}
	}

	return processedImports, nil
}

// processImportByLanguage 根据语言类型统一处理导入
func (da *DependencyAnalyzer) processImportByLanguage(imp *resolver.Import, language lang.Language,
	project *workspace.Project) *resolver.Import {
	// go项目，只处理 module前缀的，过滤非module前缀的
	packageType, err := da.PackageClassifier.ClassifyPackage(language, imp.Name, project)
	if err != nil {
		da.logger.Debug("classify %s package %s error: %v", imp.Path, imp.Name, err)
	}

	// 过滤掉系统包、第三方包
	if packageType == packageclassifier.SystemPackage || packageType == packageclassifier.ThirdPartyPackage {
		return nil
	}
	// go ，去掉module
	if language == lang.Go && len(project.GoModules) > 0 {
		for _, goModule := range project.GoModules {
			if goModule != types.EmptyString {
				imp.Source = strings.TrimPrefix(imp.Source, goModule+types.Slash)
				imp.Name = strings.TrimPrefix(imp.Name, goModule+types.Slash)
			}
		}
	}

	// 处理相对路径
	if strings.HasPrefix(imp.Source, types.Dot) {
		imp.Source = da.resolveRelativePath(imp.Source, imp.Path)
		imp.Name = da.resolveRelativePath(imp.Name, imp.Path)
	}

	// / 转 .
	imp.Source = da.normalizeImportPath(imp.Source)
	imp.Name = da.normalizeImportPath(imp.Name)

	return imp
}

// resolveRelativePath 统一解析相对路径
func (da *DependencyAnalyzer) resolveRelativePath(importPath, currentFilePath string) string {
	currentDir := filepath.Dir(currentFilePath)

	// 计算上跳层级
	upLevels := strings.Count(importPath, types.ParentDir)
	baseDir := currentDir
	for i := 0; i < upLevels; i++ {
		baseDir = filepath.Dir(baseDir)
	}

	// 处理剩余路径
	relPath := strings.ReplaceAll(importPath, types.CurrentDir, types.EmptyString)
	relPath = strings.ReplaceAll(relPath, types.ParentDir, types.EmptyString)

	return filepath.Join(baseDir, relPath)
}

// normalizeImportPath 标准化导入路径，统一处理点和斜杠; 处理 *
func (da *DependencyAnalyzer) normalizeImportPath(path string) string {
	path = strings.ReplaceAll(path, types.WindowsSeparator, types.Dot)
	path = strings.ReplaceAll(path, types.UnixSeparator, types.Dot)
	path = strings.ReplaceAll(path, types.Star, types.EmptyString)
	return filepath.Clean(path)
}

// IsFilePathInImportPackage 判断文件路径是否属于导入包的范围
func IsFilePathInImportPackage(filePath string, imp *codegraphpb.Import) bool {
	// imp是文件A的导入路径，filePath是文件B的路径
	// 目的是判断文件A是否可导入文件B里面的符号，如果可以，则返回true
	// 转换为.分隔格式（替换所有系统分隔符）
	filePath = strings.ReplaceAll(filePath, types.WindowsSeparator, types.Dot)
	filePath = strings.ReplaceAll(filePath, types.UnixSeparator, types.Dot)

	// 如果满足则说明，filePath(绝对路径)是imp包(相对路径)下面的一个文件，则大概率可以说明文件A可导入文件B里面的符号
	return strings.Contains(filePath, imp.Name) || strings.Contains(filePath, imp.Source)
}
