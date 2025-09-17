package wiki

import (
	"codebase-indexer/pkg/logger"
	"fmt"
)

// GeneratorFactory 文档生成器工厂
type GeneratorFactory struct {
	generators map[DocumentType]func(*SimpleConfig, logger.Logger) (DocumentGenerator, error)
	logger     logger.Logger
}

// NewGeneratorFactory 创建新的生成器工厂
func NewGeneratorFactory(log logger.Logger) *GeneratorFactory {
	factory := &GeneratorFactory{
		generators: make(map[DocumentType]func(*SimpleConfig, logger.Logger) (DocumentGenerator, error)),
		logger:     log,
	}

	// 注册默认的生成器
	factory.RegisterGenerator(DocTypeWiki, NewWikiGenerator)
	factory.RegisterGenerator(DocTypeCodeRules, NewCodeRulesGenerator)

	return factory
}

// RegisterGenerator 注册文档生成器
func (f *GeneratorFactory) RegisterGenerator(docType DocumentType, creator func(*SimpleConfig, logger.Logger) (DocumentGenerator, error)) {
	f.generators[docType] = creator
	f.logger.Info("Registered document generator: %s", docType)
}

// CreateGenerator 创建指定类型的文档生成器
func (f *GeneratorFactory) CreateGenerator(docType DocumentType, config *SimpleConfig) (DocumentGenerator, error) {
	creator, exists := f.generators[docType]
	if !exists {
		return nil, fmt.Errorf("unsupported document type: %s", docType)
	}

	return creator(config, f.logger)
}

// GetSupportedTypes 获取支持的文档类型
func (f *GeneratorFactory) GetSupportedTypes() []DocumentType {
	types := make([]DocumentType, 0, len(f.generators))
	for docType := range f.generators {
		types = append(types, docType)
	}
	return types
}
