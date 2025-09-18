package wiki

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"codebase-indexer/pkg/logger"
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
	return g.GenerateDocumentWithProgress(ctx, repoPath, LogProgressCallback)
}

// GenerateDocumentWithProgress 带进度回调的代码规则文档生成
func (g *CodeRulesGenerator) GenerateDocumentWithProgress(ctx context.Context, repoPath string, callback ProgressCallback) (*DocumentStructure, error) {
	// 预生成验证 - 检查模板一致性
	callback(g.logger, 0.1, "pre_validation", "Validating template consistency")
	g.startStage()

	if err := g.validateTemplateConsistency(); err != nil {
		callback(g.logger, 0.1, "error", fmt.Sprintf("Template validation failed: %v", err))
		g.outputErrorStats(callback)
		return nil, fmt.Errorf("template validation failed: %w", err)
	}

	g.endStage("pre_validation")

	// 执行通用生成逻辑
	docStructure, files, fileTree, readmeContent, err := g.CommonGenerateLogic(ctx, repoPath, callback)
	if err != nil {
		return nil, err
	}

	// 生成代码规则结构
	callback(g.logger, 0.5, "structure_generation", "Generating code rules structure")
	g.startStage()

	codeRulesStructure, err := g.generateCodeRulesStructure(ctx, fileTree, readmeContent, repoPath, files)
	if err != nil {
		callback(g.logger, 0.5, "error", fmt.Sprintf("Failed to generate code rules structure: %v", err))
		g.outputErrorStats(callback)
		return nil, fmt.Errorf("failed to generate code rules structure: %w", err)
	}

	g.endStage("structure_generation")

	// 转换通用结构
	*docStructure = *codeRulesStructure

	// 生成页面内容
	callback(g.logger, 0.6, "content_generation", "Generating code rules content")
	g.startStage()

	err = g.generateCodeRulesPageContent(ctx, docStructure.Pages, repoPath, files, fileTree, readmeContent, callback)
	if err != nil {
		callback(g.logger, 0.6, "error", fmt.Sprintf("Failed to generate code rules content: %v", err))
		g.outputErrorStats(callback)
		return nil, fmt.Errorf("failed to generate code rules content: %w", err)
	}

	g.endStage("content_generation")

	// 完成
	callback(g.logger, 1.0, "complete", "Code rules generation completed")
	g.performance.Finish()

	// 输出统计信息
	g.outputFinalStats(callback)

	g.logger.Info("Code rules document generation completed: %s - %d guidelines", repoPath, len(docStructure.Pages))

	return docStructure, nil
}

// generateCodeRulesStructure 生成代码规则结构
func (g *CodeRulesGenerator) generateCodeRulesStructure(ctx context.Context, fileTree string, readmeContent string, repoPath string, fileMetas []*FileMeta) (*DocumentStructure, error) {
	// 获取文件路径列表
	var filePaths []string
	for _, fileMeta := range fileMetas {
		filePaths = append(filePaths, fileMeta.Path)
	}

	// 构建代码规则结构生成提示词
	prompt, err := g.getCodeRulesStructurePrompt(fileTree, readmeContent, repoPath, filePaths)
	if err != nil {
		return nil, fmt.Errorf("failed to get code rules structure prompt: %w", err)
	}

	// 调用LLM生成代码规则结构
	startTime := time.Now()
	response, err := g.llmClient.GenerateContent(ctx, prompt, g.config.MaxTokens, g.config.Temperature)
	duration := time.Since(startTime)

	g.recordLLMCall(duration, "structure", err == nil)

	if err != nil {
		return nil, fmt.Errorf("failed to generate code rules structure: %w", err)
	}

	// 解析XML响应
	codeRulesStructure, err := g.parseCodeRulesStructureXML(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse code rules structure XML: %w", err)
	}

	// 设置基本属性
	if codeRulesStructure.ID == "" {
		codeRulesStructure.ID = "code_rules"
	}
	if codeRulesStructure.Title == "" {
		codeRulesStructure.Title = filepath.Base(repoPath) + " Coding Rules"
	}

	return codeRulesStructure, nil
}

// getCodeRulesStructurePrompt 获取代码规则结构生成提示词 - 简化版本
func (g *CodeRulesGenerator) getCodeRulesStructurePrompt(fileTree string, readmeContent string, repoPath string, filePaths []string) (string, error) {
	outputLang := getOutputLanguage(g.config.Language)

	// 简化分析，只识别基本目录结构 TODO 根据项目规模推断
	pageCount := "6"

	// 简化数据结构，专注于目录结构识别
	data := GenerateCodeRulesStructPromptData{
		FileTree:       fileTree,
		ReadmeContent:  readmeContent,
		OutputLanguage: outputLang,
		PageCount:      pageCount,
		ProjectName:    filepath.Base(repoPath),
	}

	templateName := "code_rules_structure"
	return g.promptManager.GetPrompt(data, templateName)
}

// parseCodeRulesStructureXML 解析代码规则结构XML
func (g *CodeRulesGenerator) parseCodeRulesStructureXML(xmlText string) (*DocumentStructure, error) {
	g.logger.Debug("Starting to parse code rules structure XML, input length: %d", len(xmlText))

	// 使用新的XML解析器
	xmlParser := NewXMLParser(g.logger)

	// 尝试解析为代码规则结构
	docStructure, err := xmlParser.ParseCodeRulesStructure(xmlText)
	if err != nil {
		g.logger.Error("Failed to parse code rules structure XML: %v", err)
		return nil, fmt.Errorf("failed to parse code rules structure XML: %w", err)
	}

	// 设置默认值
	if docStructure.ID == "" {
		docStructure.ID = "code_rules"
	}
	if len(docStructure.Sections) == 0 {
		docStructure.Sections = []DocumentSection{}
	}
	if len(docStructure.RootSections) == 0 {
		docStructure.RootSections = []string{}
	}
	if docStructure.Metadata == nil {
		docStructure.Metadata = make(map[string]interface{})
	}

	g.logger.Debug("Successfully parsed code rules structure XML, found %d pages", len(docStructure.Pages))
	return docStructure, nil
}

// outputFinalStats 输出最终统计信息
func (g *CodeRulesGenerator) outputFinalStats(callback ProgressCallback) {
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

// generateCodeRulesPageContent 生成代码规则页面内容
// loadRelevantFileContents 加载相关文件的真实内容
func (g *CodeRulesGenerator) loadRelevantFileContents(filePaths []string, fileMetas []*FileMeta) (string, error) {
	var contents []string

	for _, filePath := range filePaths {
		// 在fileMetas中查找对应的文件
		for _, fileMeta := range fileMetas {
			if fileMeta.Path == filePath {
				// 读取文件内容
				content, err := os.ReadFile(filePath)
				if err != nil {
					g.logger.Warn("Failed to read file %s: %v", filePath, err)
					continue
				}
				contents = append(contents, fmt.Sprintf("## File: %s\n```go\n%s\n```\n", filePath, string(content)))
				break
			}
		}
	}

	return strings.Join(contents, "\n"), nil
}

func (g *CodeRulesGenerator) generateCodeRulesPageContent(ctx context.Context, pages []DocumentPage, repoPath string, fileMetas []*FileMeta, fileTree string, readmeContent string, callback ProgressCallback) error {
	if callback == nil {
		callback = LogProgressCallback
	}

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
			err := g.generateCodeRulesSinglePageContent(ctx, page, repoPath, fileMetas, fileTree, readmeContent)

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

// generateCodeRulesSinglePageContent 生成单个代码规则页面内容
func (g *CodeRulesGenerator) generateCodeRulesSinglePageContent(ctx context.Context, page *DocumentPage, repoPath string, fileMetas []*FileMeta, fileTree string, readmeContent string) error {
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
		fileUrl := g.generateFileUrl(filePath)
		fileLinks = append(fileLinks, fmt.Sprintf("- [%s](%s)", filePath, fileUrl))

		// 加载文件实际内容
		content, err := g.loadFileContent(repoPath, filePath)
		if err != nil {
			g.logger.Warn("Failed to load file content for %s: %v", filePath, err)
			continue
		}
		fileContents = append(fileContents, fmt.Sprintf("### %s\n```\n%s\n```", filePath, content))
	}

	g.logger.Debug("Loaded %d file contents for page %s", len(fileContents), page.ID)

	// 简化提示词数据，专注于真实代码内容
	data := GenerateCodeRulesPromptData{
		PageTitle:      page.Title,
		FileLinks:      strings.Join(fileLinks, "\n"),
		OutputLanguage: getOutputLanguage(g.config.Language),
		GuidelineCount: "1", // 简化为1条核心规则
		FileTree:       fileTree,
		ReadmeContent:  readmeContent,
		FileContents:   strings.Join(fileContents, "\n\n"),
		RelatedFiles:   page.FilePaths,
	}

	// 使用优化后的模板
	promptTemplate := string(g.documentType) + "_page"
	prompt, err := g.promptManager.GetPrompt(data, promptTemplate)
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

	// 由于新的简化XML解析器不再处理复杂的guideline内容，
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

// validateTemplateConsistency 验证模板一致性
func (g *CodeRulesGenerator) validateTemplateConsistency() error {
	g.logger.Info("Validating template consistency for code rules generation")

	// 创建XML解析器
	xmlParser := NewXMLParser(g.logger)

	// 验证代码规则结构模板
	sampleStructureXML := `<?xml version="1.0" encoding="UTF-8"?>
<code_rules_structure>
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
</code_rules_structure>`

	// 测试解析结构XML
	_, err := xmlParser.ParseCodeRulesStructure(sampleStructureXML)
	if err != nil {
		return fmt.Errorf("code rules structure template validation failed: %w", err)
	}

	g.logger.Info("Template consistency validation passed")
	return nil
}

// loadFileContent 加载文件内容
func (g *CodeRulesGenerator) loadFileContent(repoPath string, filePath string) (string, error) {
	fullPath := filePath
	if !filepath.IsAbs(filePath) {
		fullPath = filepath.Join(repoPath, filePath)
	}
	g.logger.Debug("Loading file content for %s", filePath)

	// 检查文件是否存在
	info, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("file not found: %s", fullPath)
	}
	if info != nil && info.IsDir() {
		return "", fmt.Errorf("file is a directory: %s", fullPath)
	}

	// 读取文件内容
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", fullPath, err)
	}

	// 限制文件大小，避免过大的文件
	if len(content) > 10000 { // 限制为10KB
		content = content[:10000]
	}

	return string(content), nil
}

// generateFileUrl 生成文件URL
func (g *CodeRulesGenerator) generateFileUrl(filePath string) string {
	// 简单的文件URL生成逻辑，可以根据需要扩展
	return fmt.Sprintf("/file/%s", filePath)
}
