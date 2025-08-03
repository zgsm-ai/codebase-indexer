package packageclassifier

import "codebase-indexer/pkg/codegraph/workspace"

// Classifier 包分类器接口
type Classifier interface {
	Classify(packageName string, project *workspace.Project) PackageType
}

// ClassifierFactory 分类器工厂接口
type ClassifierFactory interface {
	CreateClassifier() Classifier
}
