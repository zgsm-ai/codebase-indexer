package wiki

import (
	"codebase-indexer/pkg/logger"
	"context"
	"fmt"
)

// DocumentManager 文档管理器，提供简化的对外接口，支持多种文档类型
type DocumentManager struct {
	factory  *GeneratorFactory
	exporter *Exporter
	store    DocumentStore
	logger   logger.Logger
	config   *SimpleConfig
}

// NewDocumentManager 创建新的文档管理器
func NewDocumentManager(apiKey, baseURL, model string, logger logger.Logger) (*DocumentManager, error) {
	config := DefaultSimpleConfig()
	config.APIKey = apiKey
	config.BaseURL = baseURL
	config.Model = model
	return NewDocumentManagerWithConfig(config, logger)
}

// NewDocumentManagerWithConfig 使用配置创建文档管理器
func NewDocumentManagerWithConfig(config *SimpleConfig, logger logger.Logger) (*DocumentManager, error) {
	if config == nil {
		config = DefaultSimpleConfig()
	}

	// 创建生成器工厂
	factory := NewGeneratorFactory(logger)

	return &DocumentManager{
		factory:  factory,
		exporter: NewExporter(),
		store:    NewFileStore(config.StoreBasePath),
		logger:   logger,
		config:   config,
	}, nil
}

// GenerateDocument 生成指定类型的文档
// docType: 文档类型 ("wiki" 或 "code_rules")
// repoPath: 本地仓库路径
// 返回: DocumentStructure和错误信息
func (dm *DocumentManager) GenerateDocument(ctx context.Context, docType DocumentType, repoPath string) (*DocumentStructure, error) {
	dm.logger.Info("DocumentManager starting to generate %s document: %s", docType, repoPath)

	// 创建对应类型的生成器
	generator, err := dm.factory.CreateGenerator(docType, dm.config)
	if err != nil {
		dm.logger.Error("Failed to create %s generator: %v", docType, err)
		return nil, fmt.Errorf("failed to create %s generator: %w", docType, err)
	}
	defer generator.Close()

	// 生成文档
	docStructure, err := generator.GenerateDocument(ctx, repoPath)
	if err != nil {
		dm.logger.Error("%s document generation failed: %s", docType, repoPath, err)
		return nil, err
	}

	// 存储生成的文档
	if dm.store != nil {
		dm.logger.Info("Saving generated %s document to storage for repository: %s", docType, repoPath)
		if err := dm.store.SaveDocument(repoPath, docStructure, string(docType)); err != nil {
			return nil, fmt.Errorf("failed to save %s document for repository: %s, err:%w", docType, repoPath, err)
		} else {
			// 获取存储路径用于调试
			if fileStore, ok := dm.store.(*localFileStore); ok {
				filePath := fileStore.getDocumentFilePath(repoPath, string(docType))
				dm.logger.Info("%s document successfully saved to: %s", docType, filePath)
			} else {
				dm.logger.Info("%s document successfully saved to storage for repository: %s", docType, repoPath)
			}
		}
	}

	dm.logger.Info("DocumentManager generated %s document successfully: %s - %d pages", docType, repoPath, len(docStructure.Pages))
	return docStructure, nil
}

// GenerateWiki 生成Wiki文档
// repoPath: 本地仓库路径
// 返回: WikiStructure和错误信息
func (dm *DocumentManager) GenerateWiki(ctx context.Context, repoPath string) (*WikiStructure, error) {
	// 生成文档结构
	docStructure, err := dm.GenerateDocument(ctx, DocTypeWiki, repoPath)
	if err != nil {
		return nil, err
	}

	// 转换为WikiStructure以保持兼容性
	return dm.convertToWikiStructure(docStructure), nil
}

// GenerateCodeRules 生成代码规则文档
// repoPath: 本地仓库路径
// 返回: DocumentStructure和错误信息
func (dm *DocumentManager) GenerateCodeRules(ctx context.Context, repoPath string) (*DocumentStructure, error) {
	return dm.GenerateDocument(ctx, DocTypeCodeRules, repoPath)
}

// ExportDocument 导出文档
// repoPath: 本地仓库路径
// outputPath: 输出路径
// format: 导出格式 ("markdown" 或 "json")
// docType: 文档类型
// ExportMode: 导出模式，仅对 markdown 格式有效 ("single" 或 "multi")
// 返回: 错误信息
func (dm *DocumentManager) ExportDocument(repoPath string, docType DocumentType, options ExportOptions) error {
	// 从 store 中加载文档结构
	if dm.store == nil {
		return fmt.Errorf("store is not initialized")
	}

	docStructure, err := dm.store.LoadDocument(repoPath, string(docType))
	if err != nil {
		return fmt.Errorf("failed to load %s document from store: %w", docType, err)
	}

	if docStructure == nil {
		return fmt.Errorf("no %s document found for repository: %s", docType, repoPath)
	}

	dm.logger.Info("Exporting %s document for repository: %s - %d pages", docType, repoPath, len(docStructure.Pages))

	// 确定 markdown 导出模式，默认为 single
	markdownMode := options.MarkdownMode
	if markdownMode != "" {
		markdownMode = SingleMode
	}

	switch options.Format {
	case MarkdownFormat:
		return dm.exporter.ExportMarkdown(dm.convertToWikiStructure(docStructure), options)
	case JSONFormat:
		return dm.exporter.ExportJSON(dm.convertToWikiStructure(docStructure), options)
	default:
		return fmt.Errorf("unsupported export format: %s", options.Format)
	}
}

// ExportWiki 导出Wiki文档（兼容旧接口）
// repoPath: 本地仓库路径
// outputPath: 输出路径
// format: 导出格式 ("markdown" 或 "json")
// ExportMode: 导出模式，仅对 markdown 格式有效 ("single" 或 "multi")
// 返回: 错误信息
func (dm *DocumentManager) ExportWiki(repoPath string, options ExportOptions) error {
	return dm.ExportDocument(repoPath, DocTypeWiki, options)
}

// ExportCodeRules 导出代码规则文档
// repoPath: 本地仓库路径
// outputPath: 输出路径
// format: 导出格式 ("markdown" 或 "json")
// ExportMode: 导出模式，仅对 markdown 格式有效 ("single" 或 "multi")
// 返回: 错误信息
func (dm *DocumentManager) ExportCodeRules(repoPath string, options ExportOptions) error {
	return dm.ExportDocument(repoPath, DocTypeCodeRules, options)
}

// DeleteDocument 删除文档缓存
// repoPath: 本地仓库路径
// docType: 文档类型
// 返回: 错误信息
func (dm *DocumentManager) DeleteDocument(repoPath string, docType DocumentType) error {
	return dm.store.DeleteDocument(repoPath, string(docType))
}

// DeleteWiki 删除Wiki缓存（兼容旧接口）
// repoPath: 本地仓库路径
// 返回: 错误信息
func (dm *DocumentManager) DeleteWiki(repoPath string) error {
	return dm.DeleteDocument(repoPath, DocTypeWiki)
}

// DeleteCodeRules 删除代码规则文档缓存
// repoPath: 本地仓库路径
// 返回: 错误信息
func (dm *DocumentManager) DeleteCodeRules(repoPath string) error {
	return dm.DeleteDocument(repoPath, DocTypeCodeRules)
}

// ExistsDocument 检查文档是否存在
// repoPath: 本地仓库路径
// docType: 文档类型
// 返回: 是否存在
func (dm *DocumentManager) ExistsDocument(repoPath string, docType DocumentType) bool {
	return dm.store.DocumentExists(repoPath, string(docType))
}

// ExistsWiki 检查Wiki是否存在（兼容旧接口）
// repoPath: 本地仓库路径
// 返回: 是否存在
func (dm *DocumentManager) ExistsWiki(repoPath string) bool {
	return dm.ExistsDocument(repoPath, DocTypeWiki)
}

// ExistsCodeRules 检查代码规则文档是否存在
// repoPath: 本地仓库路径
// 返回: 是否存在
func (dm *DocumentManager) ExistsCodeRules(repoPath string) bool {
	return dm.ExistsDocument(repoPath, DocTypeCodeRules)
}

// GetSupportedDocumentTypes 获取支持的文档类型
func (dm *DocumentManager) GetSupportedDocumentTypes() []DocumentType {
	return dm.factory.GetSupportedTypes()
}

// convertToWikiStructure 将DocumentStructure转换为WikiStructure（用于兼容性）
func (dm *DocumentManager) convertToWikiStructure(docStructure *DocumentStructure) *WikiStructure {
	if docStructure == nil {
		return nil
	}

	// 转换页面
	wikiPages := make([]WikiPage, len(docStructure.Pages))
	for i, page := range docStructure.Pages {
		wikiPages[i] = WikiPage{
			ID:           page.ID,
			Title:        page.Title,
			Content:      page.Content,
			FilePaths:    page.FilePaths,
			Importance:   page.Importance,
			RelatedPages: page.RelatedPages,
			ParentID:     page.ParentID,
		}
	}

	// 转换章节
	wikiSections := make([]WikiSection, len(docStructure.Sections))
	for i, section := range docStructure.Sections {
		wikiSections[i] = WikiSection{
			ID:          section.ID,
			Title:       section.Title,
			Pages:       section.Pages,
			Subsections: section.Subsections,
		}
	}

	return &WikiStructure{
		ID:           docStructure.ID,
		Title:        docStructure.Title,
		Description:  docStructure.Description,
		Pages:        wikiPages,
		Sections:     wikiSections,
		RootSections: docStructure.RootSections,
	}
}

// Close 关闭管理器
func (dm *DocumentManager) Close() error {
	// 清理资源
	dm.factory = nil
	return nil
}
