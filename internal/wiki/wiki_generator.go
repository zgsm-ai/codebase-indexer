package wiki

import (
	"context"
	"fmt"
	"strings"
	"time"

	"codebase-indexer/pkg/logger"
)

// WikiGenerator 兼容原版的Wiki生成器（基于新的基础架构）
type WikiGenerator struct {
	*BaseGenerator
}

// NewWikiGenerator 创建Wiki生成器V2
func NewWikiGenerator(config *SimpleConfig, logger logger.Logger) (DocumentGenerator, error) {
	baseGen, err := NewBaseGenerator(config, logger, DocTypeWiki)
	if err != nil {
		return nil, err
	}

	return &WikiGenerator{
		BaseGenerator: baseGen,
	}, nil
}

// GenerateDocument 生成Wiki文档
func (g *WikiGenerator) GenerateDocument(ctx context.Context, repoPath string) (*DocumentStructure, error) {
	docStructure, err := g.BaseGenerator.GenerateDocument(ctx, repoPath, DocTypeWiki, "6-15")
	if err != nil {
		return nil, err
	}
	repoInfo, err := g.collectRepoInfo(ctx, repoPath)
	if err != nil {
		return nil, err
	}
	// 生成项目概览页面
	g.logger.Info("start to generate project overview for repo %s", repoPath)

	overviewPage, err := g.generateProjectOverview(ctx, repoInfo)
	if err != nil {
		return docStructure, fmt.Errorf("failed to generate project overview: %w", err)
	}
	docStructure.Description = overviewPage.Content
	docStructure.Pages = append(append([]DocumentPage(nil), *overviewPage), docStructure.Pages...)
	g.logger.Info("generate project overview for repo %s successfully", repoPath)
	return docStructure, nil
}

// generateProjectOverview 生成项目概览
func (g *WikiGenerator) generateProjectOverview(ctx context.Context, repoInfo *RepoInfo) (*DocumentPage, error) {
	// 获取文件路径列表
	var filePaths []string
	for _, file := range repoInfo.FileMeta {
		filePaths = append(filePaths, file.Path)
	}

	// 构建文件链接列表
	var fileLinks []string
	for _, path := range filePaths {
		fileLinks = append(fileLinks, fmt.Sprintf("- [%s](%s)", path, path))
	}
	fileLinksStr := strings.Join(fileLinks, "\n")

	// 获取输出语言
	outputLang := getOutputLanguage(g.config.Language)

	// 准备模板数据
	data := PagePromptData{
		PageTitle:      "Project Overview",
		FileLinks:      fileLinksStr,
		OutputLanguage: outputLang,
		ReadmeContent:  repoInfo.ReadmeContent,
	}

	// 生成提示词
	prompt, err := g.promptManager.getPrompt(data, "wiki_page")
	if err != nil {
		return nil, fmt.Errorf("failed to generate prompt: %w", err)
	}

	// 调用LLM生成内容
	startTime := time.Now()
	content, err := g.llmClient.GenerateContent(ctx, prompt, g.config.MaxTokens, g.config.Temperature)
	duration := time.Since(startTime)

	g.recordLLMCall(duration, "content", err == nil)

	if err != nil {
		return nil, fmt.Errorf("failed to generate project overview: %w", err)
	}

	return &DocumentPage{
		ID:         "project-overview",
		Title:      "Project Overview",
		Content:    content,
		FilePaths:  filePaths,
		Importance: "high",
	}, nil
}

//
//// categorizePagesIntoSections 智能分类页面到章节
//func (g *WikiGenerator) categorizePagesIntoSections(pages []DocumentPage, overviewPage *DocumentPage, repoPath string) *DocumentStructure {
//	// 定义常见类别
//	categories := []struct {
//		id       string
//		title    string
//		keywords []string
//	}{
//		{"overview", "Overview", []string{"overview", "introduction", "about"}},
//		{"architecture", "Architecture", []string{"architecture", "structure", "design", "system"}},
//		{"features", "Core Features", []string{"feature", "functionality", "core"}},
//		{"components", "Components", []string{"component", "module", "widget"}},
//		{"api", "API", []string{"api", "endpoint", "service", "server"}},
//		{"data", "Data Flow", []string{"data", "flow", "pipeline", "storage"}},
//		{"models", "Models", []string{"model", "ai", "ml", "integration"}},
//		{"ui", "User Interface", []string{"ui", "interface", "frontend", "page"}},
//		{"setup", "Setup & Configuration", []string{"setup", "config", "installation", "deploy"}},
//	}
//
//	// 初始化章节和页面集群
//	sections := []DocumentSection{}
//	rootSections := []string{}
//	pageClusters := make(map[string][]DocumentPage)
//
//	// 初始化集群
//	for _, category := range categories {
//		pageClusters[category.id] = []DocumentPage{}
//	}
//	pageClusters["other"] = []DocumentPage{}
//
//	// 将页面分配到类别
//	allPages := append([]DocumentPage{*overviewPage}, pages...)
//	for _, page := range allPages {
//		title := strings.ToLower(page.Title)
//		assigned := false
//
//		// 尝试找到匹配的类别
//		for _, category := range categories {
//			for _, keyword := range category.keywords {
//				if strings.Contains(title, keyword) {
//					pageClusters[category.id] = append(pageClusters[category.id], page)
//					assigned = true
//					break
//				}
//			}
//			if assigned {
//				break
//			}
//		}
//
//		// 如果没有匹配的类别，放入"other"
//		if !assigned {
//			pageClusters["other"] = append(pageClusters["other"], page)
//		}
//	}
//
//	// 构建章节结构
//	for _, category := range categories {
//		if len(pageClusters[category.id]) > 0 {
//			var pageIDs []string
//			for _, page := range pageClusters[category.id] {
//				pageIDs = append(pageIDs, page.ID)
//			}
//
//			section := DocumentSection{
//				ID:    category.id,
//				Title: category.title,
//				Pages: pageIDs,
//			}
//			sections = append(sections, section)
//			rootSections = append(rootSections, category.id)
//		}
//	}
//
//	// 处理"other"类别
//	if len(pageClusters["other"]) > 0 {
//		var pageIDs []string
//		for _, page := range pageClusters["other"] {
//			pageIDs = append(pageIDs, page.ID)
//		}
//
//		section := DocumentSection{
//			ID:    "other",
//			Title: "Other",
//			Pages: pageIDs,
//		}
//		sections = append(sections, section)
//		rootSections = append(rootSections, "other")
//	}
//
//	// 构建最终的文档结构
//	docStructure := &DocumentStructure{
//		ID:           "wiki",
//		Title:        filepath.Base(repoPath) + " Wiki",
//		Description:  overviewPage.Content,
//		Pages:        allPages,
//		Sections:     sections,
//		RootSections: rootSections,
//	}
//
//	return docStructure
//}
