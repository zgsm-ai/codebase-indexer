package wiki

import (
	"codebase-indexer/pkg/logger"
	"context"
)

// CodeRulesGenerator 代码规则文档生成器
type CodeRulesGenerator struct {
	*BaseGenerator
}

// NewCodeRulesGenerator 创建代码规则生成器
func NewCodeRulesGenerator(config *SimpleConfig, logger logger.Logger) (DocumentGenerator, error) {
	baseGen, err := NewBaseGenerator(config, logger, DocTypeCodeRules)
	if err != nil {
		return nil, err
	}

	return &CodeRulesGenerator{
		BaseGenerator: baseGen,
	}, nil
}

// GenerateDocument 生成代码规则文档
func (g *CodeRulesGenerator) GenerateDocument(ctx context.Context, repoPath string) (*DocumentStructure, error) {
	return g.BaseGenerator.GenerateDocument(ctx, repoPath, DocTypeCodeRules, "3-8") // TODO 根据项目规模推断
}
