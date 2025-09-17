package wiki

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
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
	return g.GenerateDocumentWithProgress(ctx, repoPath, LogProgressCallback)
}

// GenerateDocumentWithProgress 带进度回调的Wiki文档生成
func (g *WikiGenerator) GenerateDocumentWithProgress(ctx context.Context, repoPath string, callback ProgressCallback) (*DocumentStructure, error) {
	// 执行通用生成逻辑
	docStructure, files, fileTree, readmeContent, err := g.CommonGenerateLogic(ctx, repoPath, callback)
	if err != nil {
		return nil, err
	}

	// 生成项目概览页面
	callback(g.logger, 0.5, "overview_generation", "Generating project overview")
	g.startStage()

	selectedFiles := g.selectRepresentativeFiles(files)
	overviewPage, err := g.generateProjectOverview(ctx, repoPath, selectedFiles)
	if err != nil {
		callback(g.logger, 0.5, "error", fmt.Sprintf("Failed to generate project overview: %v", err))
		g.outputErrorStats(callback)
		return nil, fmt.Errorf("failed to generate project overview: %w", err)
	}

	g.endStage("overview_generation")

	// 生成Wiki结构
	callback(g.logger, 0.6, "structure_generation", "Generating Wiki structure")
	g.startStage()

	wikiStructure, err := g.generateWikiStructure(ctx, fileTree, readmeContent, repoPath, files)
	if err != nil {
		callback(g.logger, 0.6, "error", fmt.Sprintf("Failed to generate Wiki structure: %v", err))
		g.outputErrorStats(callback)
		return nil, fmt.Errorf("failed to generate Wiki structure: %w", err)
	}

	g.endStage("structure_generation")

	// 转换结构
	*docStructure = *wikiStructure

	// 重新组织页面结构
	docStructure = g.categorizePagesIntoSections(docStructure.Pages, overviewPage, repoPath)
	docStructure.Description = overviewPage.Content

	// 生成页面内容
	callback(g.logger, 0.7, "content_generation", "Generating Wiki content")
	g.startStage()

	err = g.generatePageContent(ctx, docStructure.Pages, repoPath, files, callback)
	if err != nil {
		callback(g.logger, 0.7, "error", fmt.Sprintf("Failed to generate Wiki content: %v", err))
		g.outputErrorStats(callback)
		return nil, fmt.Errorf("failed to generate Wiki content: %w", err)
	}

	g.endStage("content_generation")

	// 完成
	callback(g.logger, 1.0, "complete", "Wiki generation completed")
	g.performance.Finish()

	// 输出统计信息
	g.outputFinalStats(callback)

	g.logger.Info("Wiki document generation completed: %s - %d pages", repoPath, len(docStructure.Pages))

	return docStructure, nil
}

// selectRepresentativeFiles 选择代表性文件
func (g *WikiGenerator) selectRepresentativeFiles(fileMetas []*FileMeta) []*FileContent {
	var selectedFiles []*FileContent
	fileCount := 0
	maxOverviewFiles := 10

	// 优先选择主要文件类型
	for _, fileMeta := range fileMetas {
		if fileCount >= maxOverviewFiles {
			break
		}

		// 跳过二进制文件和测试文件
		if fileMeta.IsBinary || strings.Contains(fileMeta.Path, "test") {
			continue
		}

		// 优先选择特定类型的文件
		ext := strings.ToLower(fileMeta.Extension)
		if ext == "go" || ext == "js" || ext == "ts" || ext == "py" || ext == "java" ||
			ext == "md" || ext == "json" || ext == "yaml" || ext == "yml" {
			fileContent, err := g.processor.LoadFileContent(fileMeta)
			if err != nil {
				g.logger.Warn("Failed to load file content for %s: %v", fileMeta.Path, err)
				continue
			}
			selectedFiles = append(selectedFiles, fileContent)
			fileCount++
		}
	}

	// 如果没有选到合适的文件，随机选择一些非二进制文件
	if len(selectedFiles) == 0 {
		for _, fileMeta := range fileMetas {
			if fileCount >= maxOverviewFiles {
				break
			}
			if !fileMeta.IsBinary {
				fileContent, err := g.processor.LoadFileContent(fileMeta)
				if err != nil {
					continue
				}
				selectedFiles = append(selectedFiles, fileContent)
				fileCount++
			}
		}
	}

	return selectedFiles
}

// generateProjectOverview 生成项目概览
func (g *WikiGenerator) generateProjectOverview(ctx context.Context, repoPath string, files []*FileContent) (*DocumentPage, error) {
	// 获取文件路径列表
	var filePaths []string
	for _, file := range files {
		filePaths = append(filePaths, file.Info.Path)
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
	data := GenerateWikiPromptData{
		PageTitle:      "Project Overview",
		FileLinks:      fileLinksStr,
		OutputLanguage: outputLang,
	}

	// 生成提示词
	prompt, err := g.promptManager.GetPrompt(data, "wiki_page")
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
		ID:         "overview",
		Title:      "Project Overview",
		Content:    content,
		FilePaths:  filePaths,
		Importance: "high",
	}, nil
}

// generateWikiStructure 生成Wiki结构
func (g *WikiGenerator) generateWikiStructure(ctx context.Context, fileTree string, readmeContent string, repoPath string, fileMetas []*FileMeta) (*DocumentStructure, error) {
	// 获取文件路径列表
	var filePaths []string
	for _, fileMeta := range fileMetas {
		filePaths = append(filePaths, fileMeta.Path)
	}

	// 构建Wiki结构生成提示词
	prompt, err := g.getWikiStructurePrompt(fileTree, readmeContent, filePaths)
	if err != nil {
		return nil, fmt.Errorf("failed to get wiki structure prompt: %w", err)
	}

	// 调用LLM生成Wiki结构
	startTime := time.Now()
	response, err := g.llmClient.GenerateContent(ctx, prompt, g.config.MaxTokens, g.config.Temperature)
	duration := time.Since(startTime)

	g.recordLLMCall(duration, "structure", err == nil)

	if err != nil {
		return nil, fmt.Errorf("failed to generate wiki structure: %w", err)
	}

	// 解析XML响应
	wikiStructure, err := g.parseWikiStructureXML(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse wiki structure XML: %w", err)
	}

	// 设置基本属性
	if wikiStructure.ID == "" {
		wikiStructure.ID = "wiki"
	}
	if wikiStructure.Title == "" {
		wikiStructure.Title = filepath.Base(repoPath) + " Wiki"
	}

	return wikiStructure, nil
}

// getWikiStructurePrompt 获取Wiki结构生成提示词
func (g *WikiGenerator) getWikiStructurePrompt(fileTree string, readmeContent string, filePaths []string) (string, error) {
	outputLang := getOutputLanguage(g.config.Language)

	// 设置页面数量和wiki类型
	pageCount := "1-6"
	wikiType := "comprehensive"

	data := GenerateWikiStructPromptData{
		FileTree:       fileTree,
		ReadmeContent:  readmeContent,
		OutputLanguage: outputLang,
		PageCount:      pageCount,
		WikiType:       wikiType,
	}

	return g.promptManager.GetPrompt(data, "wiki_structure")
}

// parseWikiStructureXML 解析Wiki结构XML
func (g *WikiGenerator) parseWikiStructureXML(xmlText string) (*DocumentStructure, error) {
	// 清理markdown分隔符
	xmlText = strings.ReplaceAll(xmlText, "```xml", "")
	xmlText = strings.ReplaceAll(xmlText, "```", "")

	// 提取XML内容
	xmlMatch := regexp.MustCompile(`<wiki_structure>[\s\S]*?</wiki_structure>`).FindString(xmlText)
	if xmlMatch == "" {
		return nil, fmt.Errorf("no valid XML found in response")
	}

	// 转换为通用文档结构
	docStructure := &DocumentStructure{
		ID:           "wiki",
		Pages:        []DocumentPage{},
		Sections:     []DocumentSection{},
		RootSections: []string{},
		Metadata:     make(map[string]interface{}),
	}

	// 提取title
	titleMatch := regexp.MustCompile(`<title>([^<]+)</title>`).FindStringSubmatch(xmlMatch)
	if len(titleMatch) > 1 {
		docStructure.Title = titleMatch[1]
	}

	// 提取description
	descMatch := regexp.MustCompile(`<description>([^<]+)</description>`).FindStringSubmatch(xmlMatch)
	if len(descMatch) > 1 {
		docStructure.Description = descMatch[1]
	}

	// 提取pages
	pageMatches := regexp.MustCompile(`<page[^>]*>[\s\S]*?</page>`).FindAllString(xmlMatch, -1)
	for _, pageMatch := range pageMatches {
		page := g.parseWikiPageXML(pageMatch)
		if page != nil {
			docStructure.Pages = append(docStructure.Pages, *page)
		}
	}

	// 提取sections
	sectionMatches := regexp.MustCompile(`<section[^>]*>[\s\S]*?</section>`).FindAllString(xmlMatch, -1)
	for _, sectionMatch := range sectionMatches {
		section := g.parseWikiSectionXML(sectionMatch)
		if section != nil {
			docStructure.Sections = append(docStructure.Sections, *section)

			// 检查是否为根section
			sectionID := section.ID
			isReferenced := false
			for _, otherSection := range docStructure.Sections {
				if len(otherSection.Subsections) > 0 {
					for _, subsectionRef := range otherSection.Subsections {
						if subsectionRef == sectionID {
							isReferenced = true
							break
						}
					}
				}
				if isReferenced {
					break
				}
			}

			if !isReferenced {
				docStructure.RootSections = append(docStructure.RootSections, sectionID)
			}
		}
	}

	return docStructure, nil
}

// parseWikiPageXML 解析Wiki页面XML
func (g *WikiGenerator) parseWikiPageXML(xmlText string) *DocumentPage {
	page := &DocumentPage{
		Metadata: make(map[string]interface{}),
	}

	// 提取id
	idMatch := regexp.MustCompile(`id="([^"]+)"`).FindStringSubmatch(xmlText)
	if len(idMatch) > 1 {
		page.ID = idMatch[1]
	}

	// 提取title
	titleMatch := regexp.MustCompile(`<title>([^<]+)</title>`).FindStringSubmatch(xmlText)
	if len(titleMatch) > 1 {
		page.Title = titleMatch[1]
	}

	// 提取importance
	importanceMatch := regexp.MustCompile(`<importance>([^<]+)</importance>`).FindStringSubmatch(xmlText)
	if len(importanceMatch) > 1 {
		page.Importance = importanceMatch[1]
	} else {
		page.Importance = "medium"
	}

	// 提取relevant_files
	filePathMatches := regexp.MustCompile(`<file_path>([^<]+)</file_path>`).FindAllStringSubmatch(xmlText, -1)
	for _, match := range filePathMatches {
		if len(match) > 1 {
			page.FilePaths = append(page.FilePaths, match[1])
		}
	}

	// 提取related_pages
	relatedMatches := regexp.MustCompile(`<related>([^<]+)</related>`).FindAllStringSubmatch(xmlText, -1)
	for _, match := range relatedMatches {
		if len(match) > 1 {
			page.RelatedPages = append(page.RelatedPages, match[1])
		}
	}

	// 提取parent_section
	parentMatch := regexp.MustCompile(`<parent_section>([^<]+)</parent_section>`).FindStringSubmatch(xmlText)
	if len(parentMatch) > 1 {
		page.ParentID = parentMatch[1]
	}

	return page
}

// parseWikiSectionXML 解析Wiki章节XML
func (g *WikiGenerator) parseWikiSectionXML(xmlText string) *DocumentSection {
	section := &DocumentSection{
		Metadata: make(map[string]interface{}),
	}

	// 提取id
	idMatch := regexp.MustCompile(`id="([^"]+)"`).FindStringSubmatch(xmlText)
	if len(idMatch) > 1 {
		section.ID = idMatch[1]
	}

	// 提取title
	titleMatch := regexp.MustCompile(`<title>([^<]+)</title>`).FindStringSubmatch(xmlText)
	if len(titleMatch) > 1 {
		section.Title = titleMatch[1]
	}

	// 提取pages
	pageRefMatches := regexp.MustCompile(`<page_ref>([^<]+)</page_ref>`).FindAllStringSubmatch(xmlText, -1)
	for _, match := range pageRefMatches {
		if len(match) > 1 {
			section.Pages = append(section.Pages, match[1])
		}
	}

	// 提取subsections
	subsectionMatches := regexp.MustCompile(`<section_ref>([^<]+)</section_ref>`).FindAllStringSubmatch(xmlText, -1)
	for _, match := range subsectionMatches {
		if len(match) > 1 {
			section.Subsections = append(section.Subsections, match[1])
		}
	}

	return section
}

// categorizePagesIntoSections 智能分类页面到章节
func (g *WikiGenerator) categorizePagesIntoSections(pages []DocumentPage, overviewPage *DocumentPage, repoPath string) *DocumentStructure {
	// 定义常见类别
	categories := []struct {
		id       string
		title    string
		keywords []string
	}{
		{"overview", "Overview", []string{"overview", "introduction", "about"}},
		{"architecture", "Architecture", []string{"architecture", "structure", "design", "system"}},
		{"features", "Core Features", []string{"feature", "functionality", "core"}},
		{"components", "Components", []string{"component", "module", "widget"}},
		{"api", "API", []string{"api", "endpoint", "service", "server"}},
		{"data", "Data Flow", []string{"data", "flow", "pipeline", "storage"}},
		{"models", "Models", []string{"model", "ai", "ml", "integration"}},
		{"ui", "User Interface", []string{"ui", "interface", "frontend", "page"}},
		{"setup", "Setup & Configuration", []string{"setup", "config", "installation", "deploy"}},
	}

	// 初始化章节和页面集群
	sections := []DocumentSection{}
	rootSections := []string{}
	pageClusters := make(map[string][]DocumentPage)

	// 初始化集群
	for _, category := range categories {
		pageClusters[category.id] = []DocumentPage{}
	}
	pageClusters["other"] = []DocumentPage{}

	// 将页面分配到类别
	allPages := append([]DocumentPage{*overviewPage}, pages...)
	for _, page := range allPages {
		title := strings.ToLower(page.Title)
		assigned := false

		// 尝试找到匹配的类别
		for _, category := range categories {
			for _, keyword := range category.keywords {
				if strings.Contains(title, keyword) {
					pageClusters[category.id] = append(pageClusters[category.id], page)
					assigned = true
					break
				}
			}
			if assigned {
				break
			}
		}

		// 如果没有匹配的类别，放入"other"
		if !assigned {
			pageClusters["other"] = append(pageClusters["other"], page)
		}
	}

	// 构建章节结构
	for _, category := range categories {
		if len(pageClusters[category.id]) > 0 {
			var pageIDs []string
			for _, page := range pageClusters[category.id] {
				pageIDs = append(pageIDs, page.ID)
			}

			section := DocumentSection{
				ID:    category.id,
				Title: category.title,
				Pages: pageIDs,
			}
			sections = append(sections, section)
			rootSections = append(rootSections, category.id)
		}
	}

	// 处理"other"类别
	if len(pageClusters["other"]) > 0 {
		var pageIDs []string
		for _, page := range pageClusters["other"] {
			pageIDs = append(pageIDs, page.ID)
		}

		section := DocumentSection{
			ID:    "other",
			Title: "Other",
			Pages: pageIDs,
		}
		sections = append(sections, section)
		rootSections = append(rootSections, "other")
	}

	// 构建最终的文档结构
	docStructure := &DocumentStructure{
		ID:           "wiki",
		Title:        filepath.Base(repoPath) + " Wiki",
		Description:  overviewPage.Content,
		Pages:        allPages,
		Sections:     sections,
		RootSections: rootSections,
	}

	return docStructure
}

// outputFinalStats 输出最终统计信息
func (g *WikiGenerator) outputFinalStats(callback ProgressCallback) {
	g.performance.Finish()
	performanceStats := g.getPerformanceStats()
	llmStats := g.getLLMStats()

	performanceMessage := fmt.Sprintf("Performance Stats - Total Duration: %v, File Processing: %v, Structure Generation: %v, Content Generation: %v",
		performanceStats.TotalDuration,
		performanceStats.FileProcessing,
		performanceStats.StructureGeneration,
		performanceStats.ContentGeneration)

	llmMessage := llmStats.String()

	callback(g.logger, 1.0, "performance_stats", performanceMessage)
	callback(g.logger, 1.0, "llm_stats", llmMessage)

	g.logger.Info(performanceMessage)
	g.logger.Info(llmMessage)
}
