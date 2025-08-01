package analyzer

import (
	"codebase-indexer/pkg/codegraph/parser"
	"codebase-indexer/pkg/codegraph/types"
	"codebase-indexer/pkg/codegraph/workspace"
	"context"
	"path/filepath"
	"strings"

	"codebase-indexer/pkg/codegraph/lang"
	"codebase-indexer/pkg/codegraph/resolver"
)

// preprocessImport 预处理导入语句，处理 Import.Name Import.Source 两个字段
// 1、python、js、ts 导入相对路径处理，转为绝对路径。
// 2、go同包不同文件处理；大小写处理；import部分移除模块名。
// 3、c/cpp 简单处理，只关心项目内的源码，根据当前文件的using部分，再结合符号名；
// 4、python、ts、go 别名处理；
// 5、作用域。
// 6、将 . 统一转为 /，方便后续处理。
func (da *DependencyAnalyzer) preprocessImport(ctx context.Context,
	projectInfo *workspace.Project, fileElementTables []*parser.FileElementTable) error {
	for _, p := range fileElementTables {
		imports := make([]*resolver.Import, 0, len(p.Imports))
		for _, imp := range p.Imports {
			// TODO 过滤掉标准库、第三方库等非项目的库
			if i := da.processImportByLanguage(imp, p.Path, p.Language, projectInfo); i != nil {
				imports = append(imports, i)
			}
		}
		p.Imports = imports
	}
	return nil
}

// processImportByLanguage 根据语言类型统一处理导入
func (da *DependencyAnalyzer) processImportByLanguage(imp *resolver.Import, currentFilePath string,
	language lang.Language, project *workspace.Project) *resolver.Import {
	// go项目，只处理 module前缀的，过滤非module前缀的
	if language == lang.Go {
		goModule := project.GoModule
		if goModule == types.EmptyString {
			da.logger.Debug("process_import go module not recognized in project %s", project.Path)
		} else {
			// 如果不以package 开头，则跳过
			if !strings.HasPrefix(imp.Source, goModule) && !strings.HasPrefix(imp.Name, goModule) {
				return nil
			}
			imp.Source = strings.TrimPrefix(imp.Source, goModule+types.Slash)
			imp.Name = strings.TrimPrefix(imp.Name, goModule+types.Slash)
		}
	}

	// TODO 排除其它语言的系统或第三方的包，比如 java.* 开头的包

	// 处理相对路径
	if strings.HasPrefix(imp.Source, types.Dot) {
		imp.Source = da.resolveRelativePath(imp.Source, currentFilePath, language)
		imp.Name = da.resolveRelativePath(imp.Name, currentFilePath, language)
	}

	// . 转 /
	imp.Source = da.normalizeImportPath(imp.Source)
	imp.Name = da.normalizeImportPath(imp.Name)

	return imp
}

// resolveRelativePath 统一解析相对路径
func (da *DependencyAnalyzer) resolveRelativePath(importPath, currentFilePath string, language lang.Language) string {
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

// normalizeImportPath 标准化导入路径，统一处理点和斜杠
func (da *DependencyAnalyzer) normalizeImportPath(path string) string {
	path = strings.ReplaceAll(path, types.Dot, types.Slash)
	return filepath.Clean(path)
}
