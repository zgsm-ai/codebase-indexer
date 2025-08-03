package packageclassifier

import (
	"codebase-indexer/pkg/codegraph/workspace"
	"strings"
)

// JavaClassifier Java包分类器
type JavaClassifier struct {
	systemPrefixes map[string]bool
}

// NewJavaClassifier 创建Java分类器
func NewJavaClassifier() *JavaClassifier {
	classifier := &JavaClassifier{
		systemPrefixes: make(map[string]bool),
	}

	// 加载系统包前缀
	defaultPrefixes := []string{
		"java.", "javax.", "jakarta.",
		"org.w3c.", "org.xml.", "org.omg.",
		"org.ietf.", "org.iso.", "org.unicode.",
		"com.sun.", "sun.", "jdk.",
	}

	// 合并默认和配置中的系统包
	for _, prefix := range defaultPrefixes {
		classifier.systemPrefixes[prefix] = true
	}

	return classifier
}

func (j *JavaClassifier) Classify(packageName string, project *workspace.Project) PackageType {
	// 检查是否为系统包
	for prefix := range j.systemPrefixes {
		if strings.HasPrefix(packageName, prefix) {
			return SystemPackage
		}
	}

	// 检查是否为项目内包
	if project != nil && project.JavaPackagePrefix != "" {
		if strings.HasPrefix(packageName, project.JavaPackagePrefix) {
			return ProjectPackage
		}
	}

	// 其他情况视为未知包
	return UnknownPackage
}

// JavaClassifierFactory Java分类器工厂
type JavaClassifierFactory struct{}

func (f *JavaClassifierFactory) CreateClassifier() Classifier {
	return NewJavaClassifier()
}
