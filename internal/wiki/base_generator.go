package wiki

import (
	"codebase-indexer/pkg/logger"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// DocumentType 文档类型枚举
type DocumentType string

const (
	DocTypeWiki      DocumentType = "wiki"
	DocTypeCodeRules DocumentType = "code_rules"
	// 可以扩展更多类型：api_docs, architecture, etc.
)

// DocumentGenerator 文档生成器接口
type DocumentGenerator interface {
	GenerateDocument(ctx context.Context, repoPath string) (*DocumentStructure, error)
	Close() error
}

// BaseGenerator 基础生成器，包含通用逻辑
type BaseGenerator struct {
	llmClient      LLMClient
	processor      *Processor
	config         *SimpleConfig
	performance    *PerformanceStats
	llmStats       *LLMCallStats
	stageStartTime time.Time
	promptManager  *PromptManager
	logger         logger.Logger
	documentType   DocumentType
}

// NewBaseGenerator 创建基础生成器
func NewBaseGenerator(config *SimpleConfig, logger logger.Logger, docType DocumentType) (*BaseGenerator, error) {
	if config == nil {
		config = DefaultSimpleConfig()
	}

	llmClient, err := NewLLMClient(config.APIKey, config.BaseURL, config.Model, logger)
	if err != nil {
		return nil, err
	}

	promptManager := NewPromptManager()

	return &BaseGenerator{
		llmClient:     llmClient,
		processor:     NewProcessor(config, logger),
		config:        config,
		performance:   NewPerformanceStats(),
		llmStats:      NewLLMCallStats(),
		promptManager: promptManager,
		logger:        logger,
		documentType:  docType,
	}, nil
}

// GetDocumentType 获取文档类型
func (g *BaseGenerator) GetDocumentType() DocumentType {
	return g.documentType
}

// Close 关闭生成器
func (g *BaseGenerator) Close() error {
	if g.llmClient != nil {
		return g.llmClient.Close()
	}
	return nil
}

// GenerateDocument 生成代码规则文档
func (g *BaseGenerator) GenerateDocument(ctx context.Context, repoPath string, documentType DocumentType, pageCount string) (*DocumentStructure, error) {
	// 预生成验证 - 检查模板一致性
	if err := g.validateTemplateConsistency(); err != nil {
		return nil, fmt.Errorf("template validation failed: %w", err)
	}
	g.startStage()
	// 执行通用生成逻辑
	repoInfo, err := g.collectRepoInfo(ctx, repoPath)
	if err != nil {
		return nil, fmt.Errorf("collect repo info err:%w", err)
	}
	g.endStage("collect_repo_info")

	g.logger.Info("collect repo %s info cost %d ms, collected %d files", repoPath, g.performance.TotalDuration.Milliseconds(), len(repoInfo.FileMeta))

	g.startStage()

	docStructure, err := g.generateStructure(ctx, repoInfo, documentType, pageCount)
	if err != nil {
		return nil, fmt.Errorf("failed to generate %s structure: %w", string(g.documentType), err)
	}

	g.endStage("structure_generation")
	g.logger.Info("generate %s structure cost %d ms", string(g.documentType), g.performance.TotalDuration.Milliseconds())

	// 生成页面内容
	g.startStage()

	// 这里记录进度较合适
	err = g.generatePages(ctx, docStructure.Pages, repoInfo)
	if err != nil {
		g.outputErrorStats()
		return nil, fmt.Errorf("failed to generate pages content: %w", err)
	}

	g.endStage("content_generation")

	// 完成
	logProgress(g.logger, 1.0, "complete", "document generation completed")
	g.performance.Finish()

	// 输出统计信息
	g.outputFinalStats()

	g.logger.Info("%s document generation completed: %s - %d pages", string(g.documentType), repoPath, len(docStructure.Pages))

	return docStructure, nil
}

// generateStructure 生成Wiki结构
func (g *BaseGenerator) generateStructure(ctx context.Context, repoInfo *RepoInfo,
	docType DocumentType, pageCount string) (*DocumentStructure, error) {

	data := StructPromptData{
		FileTree:       repoInfo.FileTree,
		ReadmeContent:  repoInfo.ReadmeContent,
		OutputLanguage: getOutputLanguage(g.config.Language),
		PageCount:      pageCount,
		ProjectName:    filepath.Base(repoInfo.Path),
	}
	prompt, err := g.promptManager.getStructurePrompt(data, docType)
	if err != nil {
		return nil, fmt.Errorf("failed to get %s structure prompt: %w", string(docType), err)
	}

	// 获取文件路径列表
	var filePaths []string
	for _, fileMeta := range repoInfo.FileMeta {
		filePaths = append(filePaths, fileMeta.Path)
	}

	// 调用LLM生成Wiki结构
	startTime := time.Now()
	xmlResponse, err := g.llmClient.GenerateContent(ctx, prompt, g.config.MaxTokens, g.config.Temperature)
	duration := time.Since(startTime)
	g.recordLLMCall(duration, "structure", err == nil)
	if err != nil {
		return nil, fmt.Errorf("failed to generate %s structure: %w", string(docType), err)
	}

	// 解析XML响应
	wikiStructure, err := g.parseStructureXML(xmlResponse, docType)
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s structure XML: %w", string(docType), err)
	}
	wikiStructure.Title = filepath.Base(repoInfo.Path) + " " + string(docType)
	return wikiStructure, nil
}

// parseStructureXML 解析代码规则结构XML
func (g *BaseGenerator) parseStructureXML(xmlText string, docType DocumentType) (*DocumentStructure, error) {
	g.logger.Debug("Starting to parse %s structure XML, input length: %d", string(docType), len(xmlText))

	xmlParser := NewXMLParser(g.logger)

	// 尝试解析为代码规则结构
	docStructure, err := xmlParser.ParseDocumentStructure(xmlText)
	if err != nil {
		g.logger.Error("Failed to parse %s structure XML: %v", string(docType), err)
		return nil, fmt.Errorf("failed to parse %s structure XML: %w", string(docType), err)
	}
	g.logger.Debug("Successfully parsed %s structure XML, found %d pages", string(docType), len(docStructure.Pages))
	return docStructure, nil
}

// outputFinalStats 输出最终统计信息
func (g *BaseGenerator) outputFinalStats() {
	g.performance.Finish()
	performanceStats := g.getPerformanceStats()
	llmStats := g.getLLMStats()

	performanceMessage := fmt.Sprintf("Performance Stats - Total Duration: %v, File Processing: %v, Structure Generation: %v, Content Generation: %v",
		performanceStats.TotalDuration,
		performanceStats.FileProcessing,
		performanceStats.StructureGeneration,
		performanceStats.ContentGeneration)

	llmMessage := llmStats.String()

	logProgress(g.logger, 1.0, "performance_stats", performanceMessage)
	logProgress(g.logger, 1.0, "llm_stats", llmMessage)

	g.logger.Info(performanceMessage)
	g.logger.Info(llmMessage)
}

// collectRepoInfo 通用生成逻辑
func (g *BaseGenerator) collectRepoInfo(ctx context.Context, repoPath string) (*RepoInfo, error) {
	repoInfo := &RepoInfo{Path: repoPath}
	// 重置性能统计
	g.resetPerformanceStats()

	g.logger.Info("Starting to generate %s document: %s", g.documentType, repoPath)

	callback := logProgress

	callback(g.logger, 0.0, "file_processing", "Starting to process repository files")

	// 1. 处理仓库文件
	g.startStage()
	files, err := g.processor.ProcessRepository(ctx, repoPath)
	g.endStage("file_processing")

	if err != nil {
		callback(g.logger, 0.0, "error", fmt.Sprintf("Failed to process repository: %v", err))
		g.outputErrorStats()
		return repoInfo, fmt.Errorf("failed to process repository: %w", err)
	}

	if len(files) == 0 {
		callback(g.logger, 0.0, "error", "No files found in repository")
		g.outputErrorStats()
		return repoInfo, fmt.Errorf("no files found in repository")
	}

	fileProgress := float64(len(files)) / float64(len(files))
	callback(g.logger, fileProgress, "file_processing_complete", fmt.Sprintf("Successfully processed %d files", len(files)))

	// 2. 生成文件树和README内容
	callback(g.logger, fileProgress, "structure_generation", "Generating file tree and README content")
	g.startStage()
	repoInfo.ReadmeContent = g.loadReadmeContent(ctx, repoPath, files)
	repoInfo.FileTree = g.buildFileTree(files)
	return repoInfo, nil
}

// loadReadmeContent README内容
func (g *BaseGenerator) loadReadmeContent(ctx context.Context, repoPath string, fileMetas []*FileMeta) string {
	var readmeContent string
	for _, fileMeta := range fileMetas {
		fileName := strings.ToLower(filepath.Base(fileMeta.Path))
		if strings.HasPrefix(fileName, "readme") {
			fileContent, err := g.processor.LoadFileContent(fileMeta)
			if err != nil {
				g.logger.Warn("Failed to load README content: %s - %v", fileMeta.Path, err)
				continue
			}
			readmeContent = fileContent.Content
			break
		}
	}

	return readmeContent
}

func (g *BaseGenerator) buildFileTree(fileMetas []*FileMeta) string {
	var fileTreeBuilder strings.Builder
	for _, fileMeta := range fileMetas {
		fileTreeBuilder.WriteString(fileMeta.Path)
		fileTreeBuilder.WriteString("\n")
	}
	fileTree := fileTreeBuilder.String()
	return fileTree
}

func (g *BaseGenerator) generatePages(ctx context.Context, pages []DocumentPage, repoInfo *RepoInfo) error {
	callback := logProgress
	maxConcurrent := g.config.Concurrency
	if maxConcurrent <= 0 {
		maxConcurrent = 1
	}

	var wg sync.WaitGroup
	queue := make(chan *DocumentPage, len(pages))
	errChan := make(chan error, len(pages))

	// 填充队列（使用指针）
	for i := range pages {
		queue <- &pages[i]
	}
	close(queue)

	completedPages := 0
	totalPages := len(pages)

	worker := func() {
		defer wg.Done()
		for page := range queue {
			err := g.generateSinglePageContent(ctx, page, repoInfo)

			if err != nil {
				errChan <- fmt.Errorf("failed to generate content for page %s: %w", page.ID, err)
			} else {
				completedPages++
				progress := float64(completedPages) / float64(totalPages)
				callback(g.logger, progress, "content_generation",
					fmt.Sprintf("Generated page: %s (%d/%d)", page.Title, completedPages, totalPages))
			}
		}
	}

	// 启动工作协程
	for i := 0; i < maxConcurrent; i++ {
		wg.Add(1)
		go worker()
	}

	// 等待完成
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

// generateSinglePageContent 生成单个代码规则页面内容
func (g *BaseGenerator) generateSinglePageContent(ctx context.Context, page *DocumentPage, repoInfo *RepoInfo) error {
	if page.Content != "" {
		g.logger.Debug("Page %s already has content, skipping generation", page.ID)
		return nil
	}

	g.logger.Info("Generating content for page: %s (ID: %s)", page.Title, page.ID)
	g.logger.Debug("Page file paths: %v", page.FilePaths)

	// 加载相关文件内容
	var fileLinks []string
	var fileContents []string

	for _, filePath := range page.FilePaths {
		fileUrl := g.generateFileUrl(filePath, repoInfo.Path)
		fileLinks = append(fileLinks, fmt.Sprintf("- [%s](%s)", filePath, fileUrl))

		// 加载文件实际内容
		content, err := g.loadFileContent(repoInfo.Path, filePath)
		if err != nil {
			g.logger.Warn("Failed to load file content for %s: %v", filePath, err)
			continue
		}
		fileContents = append(fileContents, fmt.Sprintf("### %s\n```\n%s\n```", filePath, content))
	}

	g.logger.Debug("Loaded %d file contents for page %s", len(fileContents), page.ID)

	// 简化提示词数据，专注于真实代码内容
	data := PagePromptData{
		PageTitle:      page.Title,
		FileLinks:      strings.Join(fileLinks, "\n"),
		OutputLanguage: getOutputLanguage(g.config.Language),
		GuidelineCount: "5", // TODO 简化为1条核心规则
		FileTree:       repoInfo.FileTree,
		ReadmeContent:  repoInfo.ReadmeContent,
		FileContents:   strings.Join(fileContents, "\n\n"),
		RelatedFiles:   page.FilePaths,
	}

	// 使用优化后的模板
	promptTemplate := string(g.documentType) + "_page"
	prompt, err := g.promptManager.getPrompt(data, promptTemplate)
	if err != nil {
		g.logger.Error("Failed to generate prompt for page %s: %v", page.ID, err)
		return fmt.Errorf("failed to generate prompt: %w", err)
	}

	g.logger.Debug("Generated prompt for page %s, length: %d", page.ID, len(prompt))

	// 验证提示词内容
	if prompt == "" {
		g.logger.Error("Generated empty prompt for page %s", page.ID)
		return fmt.Errorf("generated empty prompt for page %s", page.ID)
	}

	// 调用LLM生成内容
	startTime := time.Now()
	content, err := g.llmClient.GenerateContent(ctx, prompt, g.config.MaxTokens, g.config.Temperature)
	duration := time.Since(startTime)

	g.recordLLMCall(duration, "content", err == nil)

	if err != nil {
		g.logger.Error("Failed to generate content for page %s: %v", page.ID, err)
		return fmt.Errorf("failed to generate page content: %w", err)
	}

	// 验证生成的内容
	if content == "" {
		g.logger.Error("Generated empty content for page %s", page.ID)
		return fmt.Errorf("generated empty content for page %s", page.ID)
	}

	// 这里直接使用生成的Markdown内容
	cleanedContent := strings.ReplaceAll(content, "```markdown", "")
	cleanedContent = strings.ReplaceAll(cleanedContent, "```", "")
	cleanedContent = strings.TrimSpace(cleanedContent)

	// 验证内容不为空
	if cleanedContent == "" {
		g.logger.Error("Generated empty content for page %s", page.ID)
		return fmt.Errorf("generated empty content for page %s", page.ID)
	}

	// 直接使用生成的Markdown内容
	g.logger.Info("Successfully generated content for page %s (length: %d)", page.ID, len(cleanedContent))
	page.Content = cleanedContent
	return nil
}

// loadFileContent 加载文件内容
func (g *BaseGenerator) loadFileContent(repoPath string, filePath string) (string, error) {
	fullPath := filePath
	if !filepath.IsAbs(filePath) {
		fullPath = filepath.Join(repoPath, filePath)
	}

	// 检查文件是否存在
	info, err := os.Stat(fullPath)
	if err != nil {
		return "", fmt.Errorf("stat file %s err: %v", err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("file is a directory: %s", fullPath)
	}
	if info.Size() > g.config.MaxFileSize {
		return "", fmt.Errorf("file size exceeds limit: %s", fullPath)
	}
	// 读取文件内容
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", fullPath, err)
	}

	// TODO 限制文件大小，避免过大的文件
	if len(content) > 10000 { // 限制为10KB
		content = content[:10000]
	}

	return string(content), nil
}

// generateFileUrl 生成文件URL
func (g *BaseGenerator) generateFileUrl(filePath string, repoPath string) string {
	relPath, err := filepath.Rel(repoPath, filePath)
	if err != nil {
		return filePath
	}
	return relPath
}

// validateTemplateConsistency 验证模板一致性
func (g *BaseGenerator) validateTemplateConsistency() error {
	g.logger.Info("Validating template consistency for code rules generation")

	// 创建XML解析器
	xmlParser := NewXMLParser(g.logger)

	// 验证代码规则结构模板
	sampleStructureXML := `<?xml version="1.0" encoding="UTF-8"?>
<document_structure>
    <title>测试项目编码规范</title>
    <description>测试描述</description>
    <pages>
        <page id="page-1">
            <title>测试页面</title>
            <description>测试页面描述</description>
            <importance>high</importance>
            <relevant_files>
                <file_path>test/file.go</file_path>
            </relevant_files>
            <parent_category>category-1</parent_category>
        </page>
    </pages>
</document_structure>`

	// 测试解析结构XML
	_, err := xmlParser.ParseDocumentStructure(sampleStructureXML)
	if err != nil {
		return fmt.Errorf("code rules structure template validation failed: %w", err)
	}

	g.logger.Info("Template consistency validation passed")
	return nil
}

// Helper methods (same as original)
func (g *BaseGenerator) startStage() {
	g.stageStartTime = time.Now()
}

func (g *BaseGenerator) endStage(stageName string) {
	if !g.stageStartTime.IsZero() {
		duration := time.Since(g.stageStartTime)
		g.performance.AddStageDuration(stageName, duration)
		g.stageStartTime = time.Time{}
	}
}

func (g *BaseGenerator) resetPerformanceStats() {
	g.performance = NewPerformanceStats()
	g.llmStats = NewLLMCallStats()
	g.stageStartTime = time.Time{}
}

func (g *BaseGenerator) recordLLMCall(duration time.Duration, callType string, success bool) {
	g.llmStats.RecordCall(duration, callType, success)
}

func (g *BaseGenerator) outputErrorStats() {
	g.performance.Finish()
	performanceStats := g.getPerformanceStats()
	llmStats := g.getLLMStats()

	performanceMessage := fmt.Sprintf("Performance Stats - Total Duration: %v", performanceStats.TotalDuration)
	llmMessage := llmStats.String()

	logProgress(g.logger, 1.0, "performance_stats", performanceMessage)
	logProgress(g.logger, 1.0, "llm_stats", llmMessage)
}

func (g *BaseGenerator) getPerformanceStats() *PerformanceStats {
	return g.performance
}

func (g *BaseGenerator) getLLMStats() *LLMCallStats {
	return g.llmStats
}
