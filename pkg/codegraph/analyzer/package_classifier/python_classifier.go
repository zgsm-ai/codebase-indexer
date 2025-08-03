package packageclassifier

import (
	"codebase-indexer/pkg/codegraph/workspace"
	"strings"
)

// PythonClassifier Python包分类器
type PythonClassifier struct {
	systemPackages map[string]bool
}

// NewPythonClassifier 创建Python分类器
func NewPythonClassifier() *PythonClassifier {
	classifier := &PythonClassifier{
		systemPackages: make(map[string]bool),
	}

	// 加载默认系统包
	defaultPackages := []string{
		"os", "sys", "io", "math", "json", "datetime", "re", "collections",
		"itertools", "random", "logging", "unittest", "pathlib", "subprocess",
		"threading", "multiprocessing", "socket", "http", "urllib", "csv",
		"xml", "email", "hashlib", "base64", "struct", "pickle", "copy",
		"enum", "functools", "operator", "dataclasses", "typing", "asyncio",
		"contextlib", "glob", "tempfile", "shutil",
	}

	// 合并默认和配置中的系统包
	for _, pkg := range defaultPackages {
		classifier.systemPackages[pkg] = true
	}

	return classifier
}

func (p *PythonClassifier) Classify(packageName string, project *workspace.Project) PackageType {
	// 分割子包
	parts := strings.Split(packageName, ".")
	rootPackage := parts[0]

	// 检查是否为系统包
	if p.systemPackages[rootPackage] {
		return SystemPackage
	}

	// 相对导入（项目内包）
	if strings.HasPrefix(packageName, ".") {
		return ProjectPackage
	}

	// 检查是否为项目内包
	if project != nil && len(project.PythonPackages) > 0 {
		for _, projPkg := range project.PythonPackages {
			// 检查根包是否匹配项目包列表中的包
			if rootPackage == projPkg {
				return ProjectPackage
			}
			// 检查完整包名是否以项目包名开头（支持子包）
			if strings.HasPrefix(packageName, projPkg+".") {
				return ProjectPackage
			}
		}
	}

	// 其他情况视为未知包，而不是直接判断为第三方包
	return UnknownPackage
}

// PythonClassifierFactory Python分类器工厂
type PythonClassifierFactory struct{}

func (f *PythonClassifierFactory) CreateClassifier() Classifier {
	return NewPythonClassifier()
}
