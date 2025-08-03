package packageclassifier

import (
	"codebase-indexer/pkg/codegraph/lang"
	"codebase-indexer/pkg/codegraph/workspace"
	"fmt"
)

// PackageClassifier 包分类器主结构体
type PackageClassifier struct {
	classifiers map[lang.Language]Classifier
	factories   map[lang.Language]ClassifierFactory
}

// NewPackageClassifier 创建新的包分类器
func NewPackageClassifier() *PackageClassifier {
	classifier := &PackageClassifier{
		classifiers: make(map[lang.Language]Classifier),
		factories:   make(map[lang.Language]ClassifierFactory),
	}

	// 注册所有分类器工厂
	classifier.RegisterFactory(lang.Java, &JavaClassifierFactory{})
	classifier.RegisterFactory(lang.Python, &PythonClassifierFactory{})
	classifier.RegisterFactory(lang.Go, &GoClassifierFactory{})
	classifier.RegisterFactory(lang.C, &CClassifierFactory{})
	classifier.RegisterFactory(lang.CPP, &CppClassifierFactory{})
	classifier.RegisterFactory(lang.JavaScript, &JavaScriptClassifierFactory{})
	classifier.RegisterFactory(lang.TypeScript, &TypeScriptClassifierFactory{})

	return classifier
}

// RegisterFactory 注册分类器工厂
func (pc *PackageClassifier) RegisterFactory(language lang.Language, factory ClassifierFactory) {
	pc.factories[language] = factory
}

// GetClassifier 获取指定语言的分类器
func (pc *PackageClassifier) GetClassifier(language lang.Language) (Classifier, error) {

	// 检查缓存的分类器
	if classifier, exists := pc.classifiers[language]; exists {
		return classifier, nil
	}

	// 检查是否有对应的工厂
	factory, exists := pc.factories[language]
	if !exists {
		return nil, fmt.Errorf("unsupported language: %s", language)
	}

	// 创建分类器
	classifier := factory.CreateClassifier()
	pc.classifiers[language] = classifier

	return classifier, nil
}

// ClassifyPackage 分类包
func (pc *PackageClassifier) ClassifyPackage(language lang.Language, packageName string,
	project *workspace.Project) (PackageType, error) {

	// 获取分类器
	classifier, err := pc.GetClassifier(language)
	if err != nil {
		return UnknownPackage, err
	}

	// 分类包
	result := classifier.Classify(packageName, project)

	return result, nil
}
