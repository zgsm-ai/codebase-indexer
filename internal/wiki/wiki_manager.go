package wiki

import (
	"codebase-indexer/pkg/logger"
	"context"
	"fmt"
)

// WikiManager Wiki管理器，提供简化的对外接口
type WikiManager struct {
	generator *Generator
	exporter  *Exporter
	store     Store
	logger    logger.Logger
}

// NewWikiManager 创建新的Wiki管理器
func NewWikiManager(apiKey, baseURL, model string, logger logger.Logger) (*WikiManager, error) {
	config := DefaultSimpleConfig()
	config.APIKey = apiKey
	config.BaseURL = baseURL
	config.Model = model
	return NewWikiManagerWithConfig(config, logger)
}

// NewWikiManagerWithConfig 使用配置创建Wiki管理器
func NewWikiManagerWithConfig(config *SimpleConfig, logger logger.Logger) (*WikiManager, error) {
	if config == nil {
		config = DefaultSimpleConfig()
	}

	generator, err := NewGenerator(config, logger)
	if err != nil {
		return nil, err
	}

	return &WikiManager{
		generator: generator,
		exporter:  NewExporter(),
		store:     NewSimpleFileStore(config.StoreBasePath),
		logger:    logger,
	}, nil
}

// GenerateWiki 生成Wiki文档
// repoPath: 本地仓库路径
// 返回: WikiStructure和错误信息
func (wm *WikiManager) GenerateWiki(ctx context.Context, repoPath string) (*WikiStructure, error) {
	wm.logger.Info("WikiManager starting to generate Wiki: %s", repoPath)

	// 生成新的wiki
	wikiStructure, err := wm.generator.GenerateWiki(ctx, repoPath)
	if err != nil {
		wm.logger.Error("Wiki generation failed: %s", repoPath, err)
		return nil, err
	}

	// 存储生成的wiki
	if wm.store != nil {
		wm.logger.Info("Saving generated wiki to storage for repository: %s", repoPath)
		if err := wm.store.SaveWiki(repoPath, wikiStructure); err != nil {
			return nil, fmt.Errorf("failed to save wiki for repository: %s, err:%w", repoPath, err)
		} else {
			// 获取存储路径用于调试
			if fileStore, ok := wm.store.(*SimpleFileStore); ok {
				filePath := fileStore.getWikiFilePath(repoPath)
				wm.logger.Info("Wiki successfully saved to: %s", filePath)
			} else {
				wm.logger.Info("Wiki successfully saved to storage for repository: %s", repoPath)
			}
		}
	}

	wm.logger.Info("WikiManager generated Wiki successfully: %s - %d pages", repoPath, len(wikiStructure.Pages))
	return wikiStructure, nil
}

// ExportWiki 导出Wiki文档
// repoPath: 本地仓库路径
// outputPath: 输出路径
// format: 导出格式 ("markdown" 或 "json")
// mode: 导出模式，仅对 markdown 格式有效 ("single" 或 "multi")
// 返回: 错误信息
func (wm *WikiManager) ExportWiki(repoPath, outputPath, format string, mode ...string) error {
	// 从 store 中加载 wiki 结构
	if wm.store == nil {
		return fmt.Errorf("store is not initialized")
	}

	wikiStructure, err := wm.store.LoadWiki(repoPath)
	if err != nil {
		return fmt.Errorf("failed to load wiki from store: %w", err)
	}

	if wikiStructure == nil {
		return fmt.Errorf("no wiki found for repository: %s", repoPath)
	}

	wm.logger.Info("Exporting wiki for repository: %s - %d pages", repoPath, len(wikiStructure.Pages))

	// 确定 markdown 导出模式，默认为 single
	markdownMode := "single"
	if len(mode) > 0 && mode[0] != "" {
		markdownMode = mode[0]
	}

	switch format {
	case "markdown":
		return wm.exporter.ExportMarkdown(wikiStructure, outputPath, markdownMode)
	case "json":
		return wm.exporter.ExportJSON(wikiStructure, outputPath)
	default:
		return fmt.Errorf("unsupported export format: %s", format)
	}
}

// DeleteWiki 删除Wiki缓存
// repoPath: 本地仓库路径
// 返回: 错误信息
func (wm *WikiManager) DeleteWiki(repoPath string) error {
	return wm.store.DeleteWiki(repoPath)
}

func (wm *WikiManager) ExistsWiki(ctx context.Context, path string) bool {
	return wm.store.WikiExists(path)
}
