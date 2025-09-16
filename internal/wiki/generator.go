package wiki

import (
	"codebase-indexer/pkg/logger"
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Generator Wiki生成器
type Generator struct {
	llmClient      LLMClient
	processor      *Processor
	config         *SimpleConfig
	performance    *PerformanceStats
	llmStats       *LLMCallStats
	stageStartTime time.Time
	promptManager  *PromptManager
	logger         logger.Logger
}

// NewGenerator 创建新的Wiki生成器
func NewGenerator(config *SimpleConfig, logger logger.Logger) (*Generator, error) {
	if config == nil {
		config = DefaultSimpleConfig()
	}

	llmClient, err := NewLLMClient(config.APIKey, config.BaseURL, config.Model, logger)
	if err != nil {
		return nil, err
	}

	// 获取全局prompt管理器
	promptManager := NewPromptManager()

	return &Generator{
		llmClient:     llmClient,
		processor:     NewProcessor(config, logger),
		config:        config,
		performance:   NewPerformanceStats(),
		llmStats:      NewLLMCallStats(),
		promptManager: promptManager,
		logger:        logger,
	}, nil
}

// GenerateWiki 生成Wiki文档 - 完整版本，与参考实现保持一致
func (g *Generator) GenerateWiki(ctx context.Context, repoPath string) (*WikiStructure, error) {
	// 使用默认的日志进度回调
	return g.GenerateWikiWithProgress(ctx, repoPath, LogProgressCallback)
}

// GenerateWikiWithProgress 带进度回调的Wiki文档生成
func (g *Generator) GenerateWikiWithProgress(ctx context.Context, repoPath string, callback ProgressCallback) (*WikiStructure, error) {
	// 如果没有提供回调，使用默认的日志回调
	if callback == nil {
		callback = LogProgressCallback
	}

	// 重置性能统计并记录开始时间
	g.resetPerformanceStats()

	g.logger.Info("Starting to generate Wiki document: %s", repoPath)

	// 发送开始事件
	callback(g.logger, 0.0, "file_processing", "Starting to process repository files")

	// 1. 处理仓库文件
	g.startStage()

	files, err := g.processor.ProcessRepository(ctx, repoPath)

	g.endStage("file_processing")
	if err != nil {
		callback(g.logger, 0.0, "error", fmt.Sprintf("Failed to process repository: %v", err))
		g.outputErrorStats(callback)
		return nil, fmt.Errorf("failed to process repository: %w", err)
	}

	if len(files) == 0 {
		callback(g.logger, 0.0, "error", "No files found in repository")
		g.outputErrorStats(callback)
		return nil, fmt.Errorf("no files found in repository")
	}

	// 发送文件处理完成事件 - 使用实际文件数计算进度
	fileProgress := float64(len(files)) / float64(len(files)) // 文件处理完成，进度100%
	callback(g.logger, fileProgress, "file_processing_complete", fmt.Sprintf("Successfully processed %d files", len(files)))

	// 2. 生成文件树和README内容
	callback(g.logger, fileProgress, "structure_generation", "Generating file tree and README content")

	g.startStage()
	fileTree, readmeContent, err := g.generateFileTreeAndReadme(ctx, repoPath, files)
	if err != nil {
		callback(g.logger, fileProgress, "error", fmt.Sprintf("Failed to generate file tree and README: %v", err))
		g.outputErrorStats(callback)
		return nil, fmt.Errorf("failed to generate file tree and readme: %w", err)
	}

	// 3. 动态确定Wiki结构（基于LLM分析）
	callback(g.logger, fileProgress, "structure_generation", "Analyzing project structure and generating Wiki framework")

	wikiStructure, err := g.determineWikiStructure(ctx, fileTree, readmeContent, repoPath, files)

	g.endStage("structure_generation")

	if err != nil {
		callback(g.logger, fileProgress, "error", fmt.Sprintf("Failed to determine Wiki structure: %v", err))
		g.outputErrorStats(callback)
		return nil, fmt.Errorf("failed to determine wiki structure: %w", err)
	}

	// 3.1. 如果是comprehensive模式，使用categorizePagesIntoSections重新组织页面结构

	callback(g.logger, fileProgress, "structure_organization", "Organizing pages into sections")

	// 延迟加载：只加载需要的文件内容用于生成项目概览
	// 选择代表性的文件（最多10个）来生成概览，避免加载所有文件
	var selectedFiles []*FileContent
	fileCount := 0
	maxOverviewFiles := 10

	// 遍历文件
	for _, fileMeta := range files {
		// 优先选择主要文件类型
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
		for _, fileMeta := range files {
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

	// 生成项目概览页面
	overviewPage, err := g.generateProjectOverview(ctx, repoPath, selectedFiles)
	if err != nil {
		callback(g.logger, fileProgress, "error", fmt.Sprintf("Failed to generate project overview: %v", err))
		g.outputErrorStats(callback)
		return nil, fmt.Errorf("failed to generate project overview: %w", err)
	}

	// 使用categorizePagesIntoSections重新组织页面结构
	wikiStructure = g.categorizePagesIntoSections(wikiStructure.Pages, overviewPage, repoPath)

	// 更新wiki结构的描述为项目概览内容
	wikiStructure.Description = overviewPage.Content

	callback(g.logger, fileProgress, "structure_organization_complete", "Pages organized into sections")

	// 发送结构生成完成事件 - 使用文件数作为基础进度
	callback(g.logger, fileProgress, "structure_generation_complete", fmt.Sprintf("Generated Wiki structure: %d pages", len(wikiStructure.Pages)))

	// 基于文件处理的进度
	totalFiles := len(files)
	callback(g.logger, float64(totalFiles)/float64(totalFiles), "content_generation", "Starting to generate page content")

	g.startStage()
	// 生成页面内容
	if err := g.generatePageContentWithProgress(ctx, wikiStructure.Pages, repoPath, files, callback); err != nil {
		callback(g.logger, float64(totalFiles)/float64(totalFiles), "error", fmt.Sprintf("Failed to generate page content: %v", err))
		g.outputErrorStats(callback)
		return nil, fmt.Errorf("failed to generate page content: %w", err)
	}
	g.endStage("content_generation")

	// 发送完成事件
	callback(g.logger, 1.0, "complete", "Wiki document generation completed")

	// 完成性能统计
	g.performance.Finish()

	// 输出性能统计信息
	performanceStats := g.getPerformanceStats()
	llmStats := g.getLLMStats()

	performanceMessage := fmt.Sprintf("Performance Stats - Total Duration: %v, File Processing: %v, Structure Generation: %v, Content Generation: %v, Average Duration: %v, Stage Count: %d",
		performanceStats.TotalDuration,
		performanceStats.FileProcessing,
		performanceStats.StructureGeneration,
		performanceStats.ContentGeneration,
		performanceStats.GetAverageDuration(),
		performanceStats.StageCount)

	llmMessage := llmStats.String()

	// 通过 callback 输出性能统计信息
	callback(g.logger, 1.0, "performance_stats", performanceMessage)
	callback(g.logger, 1.0, "llm_stats", llmMessage)

	g.logger.Info(performanceMessage)
	g.logger.Info(llmMessage)

	g.logger.Info("Wiki document generation completed: %s - %d pages", repoPath, len(wikiStructure.Pages))

	return wikiStructure, nil
}

// generateProjectOverview 生成项目概览
func (g *Generator) generateProjectOverview(ctx context.Context, repoPath string, files []*FileContent) (*WikiPage, error) {
	// 获取文件路径列表
	var filePaths []string
	for _, file := range files {
		filePaths = append(filePaths, file.Info.Path)
	}

	// 构建文件链接列表 - 与参考实现完全一致
	var fileLinks []string
	for _, path := range filePaths {
		// 使用简化的文件URL生成，与React版本保持一致
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

	promptTemplate := g.config.PromptTemplate + "_page"
	// 生成提示词，使用配置中的模式
	prompt, err := g.promptManager.GetPrompt(data, promptTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to generate prompt: %w", err)
	}
	// 调用LLM生成内容
	startTime := time.Now()
	content, err := g.llmClient.GenerateContent(ctx, prompt, g.config.MaxTokens, g.config.Temperature)
	duration := time.Since(startTime)

	// 记录LLM调用统计
	g.recordLLMCall(duration, "content", err == nil)

	if err != nil {
		return nil, fmt.Errorf("failed to generate project overview: %w", err)
	}

	return &WikiPage{
		ID:         "overview",
		Title:      "Project Overview",
		Content:    content,
		FilePaths:  filePaths,
		Importance: "high",
	}, nil
}

// categorizePagesIntoSections 智能分类页面到章节，与React版本1718-1728行保持一致
func (g *Generator) categorizePagesIntoSections(pages []WikiPage, overviewPage *WikiPage, repoPath string) *WikiStructure {
	// 定义常见类别，与React版本完全一致
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
	sections := []WikiSection{}
	rootSections := []string{}
	pageClusters := make(map[string][]WikiPage)

	// 初始化集群
	for _, category := range categories {
		pageClusters[category.id] = []WikiPage{}
	}
	pageClusters["other"] = []WikiPage{}

	// 将页面分配到类别
	allPages := append([]WikiPage{*overviewPage}, pages...)
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

			section := WikiSection{
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

		section := WikiSection{
			ID:    "other",
			Title: "Other",
			Pages: pageIDs,
		}
		sections = append(sections, section)
		rootSections = append(rootSections, "other")
	}

	// 构建最终的Wiki结构
	wikiStructure := &WikiStructure{
		ID:           "wiki",
		Title:        filepath.Base(repoPath) + " Wiki",
		Description:  overviewPage.Content,
		Pages:        allPages,
		Sections:     sections,
		RootSections: rootSections,
	}

	return wikiStructure
}

// generateFileUrl 生成文件URL，支持不同平台
func (g *Generator) generateFileUrl(filePath string, repoPath string) string {
	// 本地仓库模式，直接返回相对文件路径
	// 移除仓库路径前缀，只返回相对路径
	relPath, err := filepath.Rel(repoPath, filePath)
	if err != nil {
		return filePath
	}
	return relPath
}

// generateFileTreeAndReadme 生成文件树和README内容
func (g *Generator) generateFileTreeAndReadme(ctx context.Context, repoPath string, fileMetas []*FileMeta) (string, string, error) {
	// 生成文件树字符串
	var fileTreeBuilder strings.Builder
	for _, fileMeta := range fileMetas {
		fileTreeBuilder.WriteString(fileMeta.Path)
		fileTreeBuilder.WriteString("\n")
	}
	fileTree := fileTreeBuilder.String()

	// TODO 查找README文件并延迟加载其内容
	var readmeContent string
	for _, fileMeta := range fileMetas {
		fileName := strings.ToLower(filepath.Base(fileMeta.Path))
		if strings.HasPrefix(fileName, "readme") {
			// 延迟加载README文件内容
			fileContent, err := g.processor.LoadFileContent(fileMeta)
			if err != nil {
				g.logger.Warn("Failed to load README content: %s - %v", fileMeta.Path, err)
				continue
			}
			readmeContent = fileContent.Content
			break
		}
	}

	return fileTree, readmeContent, nil
}

// determineWikiStructure 动态确定Wiki结构，与React版本保持一致
func (g *Generator) determineWikiStructure(ctx context.Context, fileTree string, readmeContent string, repoPath string, fileMetas []*FileMeta) (*WikiStructure, error) {
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

	// 记录LLM调用统计
	g.recordLLMCall(duration, "structure", err == nil)

	if err != nil {
		return nil, fmt.Errorf("failed to generate wiki structure: %w", err)
	}

	// 解析XML响应
	wikiStructure, err := g.parseWikiStructureXML(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse wiki structure XML: %w", err)
	}

	// 设置Wiki ID和标题
	if wikiStructure.ID == "" {
		wikiStructure.ID = "wiki"
	}
	if wikiStructure.Title == "" {
		wikiStructure.Title = filepath.Base(repoPath) + " Wiki"
	}

	return wikiStructure, nil
}

// getWikiStructurePrompt 获取Wiki结构生成提示词，与React版本保持一致
func (g *Generator) getWikiStructurePrompt(fileTree string, readmeContent string, filePaths []string) (string, error) {
	// 根据语言设置输出语言
	outputLang := getOutputLanguage(g.config.Language)

	promptTemplate := g.config.PromptTemplate + "_structure"
	// 设置页面数量
	pageCount := "8-12"
	// 设置wiki类型
	wikiType := "comprehensive"

	data := GenerateWikiStructPromptData{
		FileTree:       fileTree,
		ReadmeContent:  readmeContent,
		OutputLanguage: outputLang,
		PageCount:      pageCount,
		WikiType:       wikiType,
	}
	return g.promptManager.GetPrompt(data, promptTemplate)
}

// parseWikiStructureXML 解析Wiki结构XML，与React版本保持一致
func (g *Generator) parseWikiStructureXML(xmlText string) (*WikiStructure, error) {
	// 清理markdown分隔符
	xmlText = strings.ReplaceAll(xmlText, "```xml", "")
	xmlText = strings.ReplaceAll(xmlText, "```", "")

	// 提取XML内容
	xmlMatch := regexp.MustCompile(`<wiki_structure>[\s\S]*?</wiki_structure>`).FindString(xmlText)
	if xmlMatch == "" {
		return nil, fmt.Errorf("no valid XML found in response")
	}

	// 简化的XML解析（在实际项目中可以使用更完整的XML解析器）
	wikiStructure := &WikiStructure{
		ID:           "wiki",
		Pages:        []WikiPage{},
		Sections:     []WikiSection{},
		RootSections: []string{},
	}

	// 提取title
	titleMatch := regexp.MustCompile(`<title>([^<]+)</title>`).FindStringSubmatch(xmlMatch)
	if len(titleMatch) > 1 {
		wikiStructure.Title = titleMatch[1]
	}

	// 提取description
	descMatch := regexp.MustCompile(`<description>([^<]+)</description>`).FindStringSubmatch(xmlMatch)
	if len(descMatch) > 1 {
		wikiStructure.Description = descMatch[1]
	}

	// 提取pages
	pageMatches := regexp.MustCompile(`<page[^>]*>[\s\S]*?</page>`).FindAllString(xmlMatch, -1)
	for _, pageMatch := range pageMatches {
		page := g.parseWikiPageXML(pageMatch)
		if page != nil {
			wikiStructure.Pages = append(wikiStructure.Pages, *page)
		}
	}

	// 提取sections
	sectionMatches := regexp.MustCompile(`<section[^>]*>[\s\S]*?</section>`).FindAllString(xmlMatch, -1)
	for _, sectionMatch := range sectionMatches {
		section := g.parseWikiSectionXML(sectionMatch)
		if section != nil {
			wikiStructure.Sections = append(wikiStructure.Sections, *section)

			// 检查是否为根section
			sectionID := section.ID
			isReferenced := false
			for _, otherSection := range wikiStructure.Sections {
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
				wikiStructure.RootSections = append(wikiStructure.RootSections, sectionID)
			}
		}
	}

	return wikiStructure, nil
}

// parseWikiPageXML 解析Wiki页面XML
func (g *Generator) parseWikiPageXML(xmlText string) *WikiPage {
	page := &WikiPage{}

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
func (g *Generator) parseWikiSectionXML(xmlText string) *WikiSection {
	section := &WikiSection{}

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

// generatePageContentWithConcurrency 并发生成页面内容
func (g *Generator) generatePageContentWithConcurrency(ctx context.Context, pages []WikiPage, repoPath string, fileMetas []*FileMeta) error {
	// 使用配置中的并发数
	maxConcurrent := g.config.Concurrency // 简化并发数
	if maxConcurrent <= 0 {
		maxConcurrent = 1
	}

	// 创建活动请求映射
	activeRequests := make(map[string]bool)
	var mu sync.Mutex

	// 创建错误通道
	errChan := make(chan error, len(pages))

	// 创建等待组
	var wg sync.WaitGroup

	// 处理队列
	queue := make(chan WikiPage, len(pages))
	for _, page := range pages {
		queue <- page
	}
	close(queue)

	// 工作函数
	worker := func() {
		defer wg.Done()
		for page := range queue {
			// 检查是否已经在处理中
			mu.Lock()
			if activeRequests[page.ID] {
				mu.Unlock()
				continue
			}
			activeRequests[page.ID] = true
			mu.Unlock()

			// 生成页面内容（带重试）
			var err error
			if true { // 简化重试逻辑
				err = g.generateSinglePageContentWithRetry(ctx, &page, repoPath, fileMetas)
			} else {
				err = g.generateSinglePageContent(ctx, &page, repoPath, fileMetas)
			}

			if err != nil {
				errChan <- fmt.Errorf("failed to generate content for page %s: %w", page.ID, err)
			}

			// 标记为完成
			mu.Lock()
			delete(activeRequests, page.ID)
			mu.Unlock()
		}
	}

	// 启动工作协程
	for i := 0; i < maxConcurrent; i++ {
		wg.Add(1)
		go worker()
	}

	// 等待所有工作完成
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// 检查错误
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// generateSinglePageContent 生成单个页面内容（延迟加载版本）
func (g *Generator) generateSinglePageContent(ctx context.Context, page *WikiPage, repoPath string, fileMetas []*FileMeta) error {
	// 如果内容已经存在，跳过
	if page.Content != "" {
		return nil
	}

	// 延迟加载：只加载页面相关的文件内容
	var fileLinks []string
	var loadedFileContents []string

	// 限制同时处理的文件数量，避免内存溢出
	maxFilesPerPage := 5
	processedFiles := 0

	for _, filePath := range page.FilePaths {
		if processedFiles >= maxFilesPerPage {
			// 如果文件过多，添加提示信息
			fileLinks = append(fileLinks, fmt.Sprintf("- ... and %d more files", len(page.FilePaths)-maxFilesPerPage))
			break
		}

		// 查找对应的文件元数据
		for _, fileMeta := range fileMetas {
			if fileMeta.Path == filePath {
				// 延迟加载文件内容
				fileContent, err := g.processor.LoadFileContent(fileMeta)
				if err != nil {
					g.logger.Warn("Failed to load file content for %s: %v", filePath, err)
					// 即使加载失败，也添加文件链接
					fileUrl := g.generateFileUrl(filePath, repoPath)
					fileLinks = append(fileLinks, fmt.Sprintf("- [%s](%s) (content loading failed)", filePath, fileUrl))
				} else {
					fileUrl := g.generateFileUrl(filePath, repoPath)
					fileLinks = append(fileLinks, fmt.Sprintf("- [%s](%s)", filePath, fileUrl))
					// 添加文件内容摘要（限制长度避免提示词过长）
					contentSummary := fileContent.Content
					if len(contentSummary) > 500 {
						contentSummary = contentSummary[:500] + "..."
					}
					loadedFileContents = append(loadedFileContents, fmt.Sprintf("File: %s\n```%s\n%s\n```",
						filePath, fileContent.Info.Language, contentSummary))
				}
				processedFiles++
				break
			}
		}
	}

	// 生成提示词（使用原有的generatePagePrompt方法）
	prompt, err := g.generatePagePrompt(page.Title, fileLinks, g.config.Language)
	if err != nil {
		return err
	}
	// 调用LLM生成内容
	startTime := time.Now()
	content, err := g.llmClient.GenerateContent(ctx, prompt, g.config.MaxTokens, g.config.Temperature)
	duration := time.Since(startTime)

	// 记录LLM调用统计
	g.recordLLMCall(duration, "content", err == nil)

	if err != nil {
		return fmt.Errorf("failed to generate content for page %s: %w", page.ID, err)
	}

	// 清理markdown分隔符
	content = strings.ReplaceAll(content, "```markdown", "")
	content = strings.ReplaceAll(content, "```", "")

	// 更新页面内容
	page.Content = content

	return nil
}

// generatePageContentWithProgress 带进度回调的页面内容生成
func (g *Generator) generatePageContentWithProgress(ctx context.Context, pages []WikiPage, repoPath string, fileMetas []*FileMeta, callback ProgressCallback) error {
	// 使用配置中的并发数
	maxConcurrent := g.config.Concurrency
	if maxConcurrent <= 0 {
		maxConcurrent = 1
	}

	// 创建活动请求映射
	activeRequests := make(map[string]bool)
	var mu sync.Mutex

	// 创建错误通道
	errChan := make(chan error, len(pages))

	// 创建等待组
	var wg sync.WaitGroup

	// 处理队列
	queue := make(chan WikiPage, len(pages))
	for _, page := range pages {
		queue <- page
	}
	close(queue)

	completedPages := 0
	successPages := 0
	failedPages := 0
	totalPages := len(pages)

	g.logger.Info("Starting to generate page content: total %d pages", totalPages)

	// 工作函数
	worker := func() {
		defer wg.Done()
		for page := range queue {
			// 检查是否已经在处理中
			mu.Lock()
			if activeRequests[page.ID] {
				mu.Unlock()
				continue
			}
			activeRequests[page.ID] = true
			mu.Unlock()

			// 生成页面内容（带重试）
			var err error
			if true { // 简化重试逻辑
				err = g.generateSinglePageContentWithRetry(ctx, &page, repoPath, fileMetas)
			} else {
				err = g.generateSinglePageContent(ctx, &page, repoPath, fileMetas)
			}

			mu.Lock()
			if err != nil {
				failedPages++
				errChan <- fmt.Errorf("failed to generate content for page %s: %w", page.ID, err)
				g.logger.Warn("Page generation failed: %s - %v", page.Title, err)
			} else {
				successPages++
				completedPages++
				g.logger.Info("Page generation succeeded: %s (%d/%d)", page.Title, completedPages, totalPages)

				// 发送进度更新 - 使用实际处理进度：已处理页面数 / 总页面数
				progress := float64(completedPages) / float64(totalPages)
				callback(g.logger, progress, "content_generation", fmt.Sprintf("Generating page: %s (%d/%d)", page.Title, completedPages, totalPages))
			}

			// 标记为完成
			delete(activeRequests, page.ID)
			mu.Unlock()
		}
	}

	// 启动工作协程
	for i := 0; i < maxConcurrent; i++ {
		wg.Add(1)
		go worker()
	}

	// 等待所有工作完成
	go func() {
		wg.Wait()
		close(errChan)
	}()

	// 检查错误
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	// 打印最终统计信息
	g.logger.Info("Page generation completed: total_pages=%d, success=%d, failed=%d", totalPages, successPages, failedPages)

	return nil
}

// generateSinglePageContentWithRetry 带重试机制的单个页面内容生成
func (g *Generator) generateSinglePageContentWithRetry(ctx context.Context, page *WikiPage, repoPath string, fileMetas []*FileMeta) error {
	if page.Content != "" {
		return nil
	}

	var lastErr error

	for attempt := 0; attempt <= 3; attempt++ { // 简化重试次数
		if attempt > 0 {
			// 简单的延迟重试
			time.Sleep(time.Duration(attempt) * time.Second)
		}

		err := g.generateSinglePageContent(ctx, page, repoPath, fileMetas)
		if err == nil {
			return nil
		}

		lastErr = err
	}

	return fmt.Errorf("still failed after %d retries: %w", 3, lastErr)
}

// startStage 开始阶段计时
func (g *Generator) startStage() {
	g.stageStartTime = time.Now()
}

// endStage 结束阶段计时并记录
func (g *Generator) endStage(stageName string) {
	if !g.stageStartTime.IsZero() {
		duration := time.Since(g.stageStartTime)
		g.performance.AddStageDuration(stageName, duration)
		g.stageStartTime = time.Time{} // 重置开始时间
	}
}

// getPerformanceStats 获取性能统计信息
func (g *Generator) getPerformanceStats() *PerformanceStats {
	return g.performance
}

// resetPerformanceStats 重置性能统计
func (g *Generator) resetPerformanceStats() {
	g.performance = NewPerformanceStats()
	g.llmStats = NewLLMCallStats()
	g.stageStartTime = time.Time{}
}

// recordLLMCall 记录LLM调用
func (g *Generator) recordLLMCall(duration time.Duration, callType string, success bool) {
	g.llmStats.RecordCall(duration, callType, success)
}

// getLLMStats 获取LLM调用统计
func (g *Generator) getLLMStats() *LLMCallStats {
	return g.llmStats
}

// outputErrorStats 在出错时输出统计信息
func (g *Generator) outputErrorStats(callback ProgressCallback) {
	// 完成性能统计
	g.performance.Finish()

	// 获取统计信息
	performanceStats := g.getPerformanceStats()
	llmStats := g.getLLMStats()

	// 构建性能统计消息
	performanceMessage := fmt.Sprintf("Performance Stats - Total Duration: %v, File Processing: %v, Structure Generation: %v, Content Generation: %v, Average Duration: %v, Stage Count: %d",
		performanceStats.TotalDuration,
		performanceStats.FileProcessing,
		performanceStats.StructureGeneration,
		performanceStats.ContentGeneration,
		performanceStats.GetAverageDuration(),
		performanceStats.StageCount)

	llmMessage := llmStats.String()

	// 通过 callback 输出统计信息
	callback(g.logger, 1.0, "performance_stats", performanceMessage)
	callback(g.logger, 1.0, "llm_stats", llmMessage)

	// 通过日志输出
	g.logger.Info("=== Performance Statistics Report on Error ===")
	g.logger.Info(performanceMessage)
	g.logger.Info(llmMessage)
	g.logger.Info("========================")
}

// generatePagePrompt 生成页面提示词（使用文件链接）
func (g *Generator) generatePagePrompt(pageTitle string, fileLinks []string, language string) (string, error) {
	fileLinksStr := strings.Join(fileLinks, "\n")
	// 准备模板数据
	data := GenerateWikiPromptData{
		PageTitle:      pageTitle,
		FileLinks:      fileLinksStr,
		OutputLanguage: language,
	}
	promptTemplate := g.config.PromptTemplate + "_page"

	// 使用配置中的模式
	return g.promptManager.GetPrompt(data, promptTemplate)
}

// Close 关闭生成器
func (g *Generator) Close() error {
	if g.llmClient != nil {
		return g.llmClient.Close()
	}
	return nil
}
