package wiki

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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
		// 只提供基本的目录分类，不做复杂的业务推断
		KeyDirectories: g.identifyKeyDirectories(filePaths),
	}

	templateName := "code_rules_structure"
	return g.promptManager.GetPrompt(data, templateName)
}

// analyzeRuntimePatternsFromPaths 从文件路径分析运行时行为模式
func (g *CodeRulesGenerator) analyzeRuntimePatternsFromPaths(filePaths []string) []string {
	var patterns []string
	patternMap := make(map[string]bool)

	for _, path := range filePaths {
		// 基于文件路径和命名识别运行时模式
		if strings.Contains(path, "goroutine") || strings.Contains(path, "async") {
			patternMap["并发处理"] = true
		}
		if strings.Contains(path, "pool") || strings.Contains(path, "Pool") {
			patternMap["资源池化"] = true
		}
		if strings.Contains(path, "queue") || strings.Contains(path, "Queue") {
			patternMap["队列处理"] = true
		}
		if strings.Contains(path, "scheduler") || strings.Contains(path, "cron") {
			patternMap["调度机制"] = true
		}
		if strings.Contains(path, "worker") || strings.Contains(path, "job") {
			patternMap["工作线程"] = true
		}
		if strings.Contains(path, "stream") || strings.Contains(path, "Stream") {
			patternMap["流式处理"] = true
		}
		if strings.Contains(path, "event") || strings.Contains(path, "Event") {
			patternMap["事件驱动"] = true
		}
	}

	for pattern := range patternMap {
		patterns = append(patterns, pattern)
	}
	return patterns
}

// analyzeBusinessRulesFromPaths 从文件路径分析业务规则模式
func (g *CodeRulesGenerator) analyzeBusinessRulesFromPaths(filePaths []string) []string {
	var rules []string
	ruleMap := make(map[string]bool)

	for _, path := range filePaths {
		// 基于文件路径识别业务规则
		if strings.Contains(path, "validation") || strings.Contains(path, "validate") {
			ruleMap["数据验证规则"] = true
		}
		if strings.Contains(path, "rule") || strings.Contains(path, "Rule") {
			ruleMap["业务规则引擎"] = true
		}
		if strings.Contains(path, "workflow") || strings.Contains(path, "flow") {
			ruleMap["工作流规则"] = true
		}
		if strings.Contains(path, "state") || strings.Contains(path, "status") {
			ruleMap["状态转换规则"] = true
		}
		if strings.Contains(path, "limit") || strings.Contains(path, "quota") {
			ruleMap["限流配额规则"] = true
		}
		if strings.Contains(path, "permission") || strings.Contains(path, "auth") {
			ruleMap["权限规则"] = true
		}
	}

	for rule := range ruleMap {
		rules = append(rules, rule)
	}
	return rules
}

// analyzeCodeTemplatesFromPaths 从文件路径分析代码模板模式
func (g *CodeRulesGenerator) analyzeCodeTemplatesFromPaths(filePaths []string) []string {
	var templates []string
	templateMap := make(map[string]bool)

	for _, path := range filePaths {
		// 基于文件路径识别代码模板
		if strings.Contains(path, "template") || strings.Contains(path, "Template") {
			templateMap["代码模板"] = true
		}
		if strings.Contains(path, "generator") || strings.Contains(path, "generate") {
			templateMap["代码生成器"] = true
		}
		if strings.Contains(path, "builder") || strings.Contains(path, "Builder") {
			templateMap["建造者模式"] = true
		}
		if strings.Contains(path, "factory") || strings.Contains(path, "Factory") {
			templateMap["工厂模式"] = true
		}
		if strings.Contains(path, "prototype") || strings.Contains(path, "Prototype") {
			templateMap["原型模式"] = true
		}
		if strings.Contains(path, "singleton") || strings.Contains(path, "Singleton") {
			templateMap["单例模式"] = true
		}
	}

	for template := range templateMap {
		templates = append(templates, template)
	}
	return templates
}

// identifyKeyDirectories 识别关键目录结构 - 简化版本
func (g *CodeRulesGenerator) identifyKeyDirectories(filePaths []string) map[string][]string {
	directoryMap := make(map[string][]string)

	for _, path := range filePaths {
		// 基于目录结构识别技术架构层次
		if strings.HasPrefix(path, "cmd/") {
			directoryMap["cmd"] = append(directoryMap["cmd"], path)
		}
		if strings.HasPrefix(path, "internal/") {
			directoryMap["internal"] = append(directoryMap["internal"], path)
		}
		if strings.HasPrefix(path, "pkg/") {
			directoryMap["pkg"] = append(directoryMap["pkg"], path)
		}
		if strings.HasPrefix(path, "api/") {
			directoryMap["api"] = append(directoryMap["api"], path)
		}
		if strings.HasPrefix(path, "test/") {
			directoryMap["test"] = append(directoryMap["test"], path)
		}
		if strings.Contains(path, "config") {
			directoryMap["config"] = append(directoryMap["config"], path)
		}
		if strings.Contains(path, "errs") || strings.Contains(path, "errors") {
			directoryMap["error-handling"] = append(directoryMap["error-handling"], path)
		}
		if strings.Contains(path, "_test.go") {
			directoryMap["testing"] = append(directoryMap["testing"], path)
		}
	}

	return directoryMap
}

// extractTechStack 提取技术栈
func (g *CodeRulesGenerator) extractTechStack(filePaths []string) []string {
	var techStack []string
	techMap := make(map[string]bool)

	for _, path := range filePaths {
		// 基于文件扩展和路径识别技术栈
		if strings.HasSuffix(path, ".proto") {
			techMap["gRPC/Protobuf"] = true
		}
		if strings.Contains(path, "docker") || strings.Contains(path, "Dockerfile") {
			techMap["Docker"] = true
		}
		if strings.Contains(path, "kubernetes") || strings.Contains(path, "k8s") {
			techMap["Kubernetes"] = true
		}
		if strings.Contains(path, "redis") {
			techMap["Redis"] = true
		}
		if strings.Contains(path, "mysql") || strings.Contains(path, "postgres") {
			techMap["关系型数据库"] = true
		}
		if strings.Contains(path, "mongo") {
			techMap["MongoDB"] = true
		}
		if strings.Contains(path, "elastic") {
			techMap["Elasticsearch"] = true
		}
		if strings.Contains(path, "kafka") || strings.Contains(path, "rabbitmq") {
			techMap["消息队列"] = true
		}
	}

	for tech := range techMap {
		techStack = append(techStack, tech)
	}
	return techStack
}

// extractKeyPatterns 提取关键模式
func (g *CodeRulesGenerator) extractKeyPatterns(filePaths []string) []string {
	var patterns []string
	patternMap := make(map[string]bool)

	for _, path := range filePaths {
		// 基于文件路径识别关键模式
		if strings.Contains(path, "repository") || strings.Contains(path, "dao") {
			patternMap["Repository模式"] = true
		}
		if strings.Contains(path, "service") && strings.Contains(path, "impl") {
			patternMap["Service接口+实现"] = true
		}
		if strings.Contains(path, "middleware") {
			patternMap["中间件模式"] = true
		}
		if strings.Contains(path, "interceptor") {
			patternMap["拦截器模式"] = true
		}
		if strings.Contains(path, "strategy") || strings.Contains(path, "algorithm") {
			patternMap["策略模式"] = true
		}
		if strings.Contains(path, "factory") {
			patternMap["工厂模式"] = true
		}
		if strings.Contains(path, "observer") || strings.Contains(path, "event") {
			patternMap["观察者模式"] = true
		}
	}

	for pattern := range patternMap {
		patterns = append(patterns, pattern)
	}
	return patterns
}

// extractExternalDependencies 提取外部依赖
func (g *CodeRulesGenerator) extractExternalDependencies(filePaths []string) []string {
	var deps []string
	depMap := make(map[string]bool)

	for _, path := range filePaths {
		// 基于文件内容识别外部依赖（简化版本）
		if strings.Contains(path, "third-party") || strings.Contains(path, "external") {
			depMap["第三方服务"] = true
		}
		if strings.Contains(path, "payment") {
			depMap["支付服务"] = true
		}
		if strings.Contains(path, "sms") || strings.Contains(path, "notification") {
			depMap["通知服务"] = true
		}
		if strings.Contains(path, "oauth") || strings.Contains(path, "auth") {
			depMap["认证服务"] = true
		}
	}

	for dep := range depMap {
		deps = append(deps, dep)
	}
	return deps
}

// truncateString 截断字符串用于日志显示
func (g *CodeRulesGenerator) truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
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

// parseCodeRulesPageXML 解析新的页面结构XML（已废弃，使用XMLParser替代）
func (g *CodeRulesGenerator) parseCodeRulesPageXML(xmlText string) *DocumentPage {
	// 使用新的XML解析器
	xmlParser := NewXMLParser(g.logger)
	page := xmlParser.parsePageXML(xmlText)
	return page
}

// parseCodeRulesSectionXML 解析代码规则章节XML
func (g *CodeRulesGenerator) parseCodeRulesSectionXML(xmlText string) *DocumentSection {
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

	// 提取guideline_ref作为pages
	guidelineRefMatches := regexp.MustCompile(`<guideline_ref>([^<]+)</guideline_ref>`).FindAllStringSubmatch(xmlText, -1)
	for _, match := range guidelineRefMatches {
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

// parseCodeRulesGuidelineXML 解析代码规则指南XML（转换为页面）（已废弃，使用XMLParser替代）
func (g *CodeRulesGenerator) parseCodeRulesGuidelineXML(xmlText string) *DocumentPage {
	// 由于新的简化XML解析器不再处理复杂的guideline内容，
	// 这里直接返回一个简单的页面结构
	page := &DocumentPage{
		Metadata: make(map[string]interface{}),
	}

	// 从XML中提取基本信息
	xmlParser := NewXMLParser(g.logger)
	cleanedXML := xmlParser.ExtractXMLContent(xmlText)

	// 提取标题
	if match := regexp.MustCompile(`<title>([^<]+)</title>`).FindStringSubmatch(cleanedXML); len(match) > 1 {
		page.Title = strings.TrimSpace(match[1])
	}

	// 提取重要性
	if match := regexp.MustCompile(`<importance>([^<]+)</importance>`).FindStringSubmatch(cleanedXML); len(match) > 1 {
		page.Importance = strings.TrimSpace(match[1])
	} else {
		page.Importance = "medium"
	}

	// 提取描述
	if match := regexp.MustCompile(`<description>([^<]+)</description>`).FindStringSubmatch(cleanedXML); len(match) > 1 {
		page.Description = strings.TrimSpace(match[1])
	}

	// 提取文件路径
	filePathMatches := regexp.MustCompile(`<file_path>([^<]+)</file_path>`).FindAllStringSubmatch(cleanedXML, -1)
	for _, match := range filePathMatches {
		if len(match) > 1 {
			page.FilePaths = append(page.FilePaths, strings.TrimSpace(match[1]))
		}
	}

	// 存储完整的指南内容在metadata中
	page.Metadata["guideline_xml"] = cleanedXML

	return page
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
		fileUrl := g.generateFileUrl(filePath, repoPath)
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
	fullPath := filepath.Join(repoPath, filePath)

	// 检查文件是否存在
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return "", fmt.Errorf("file not found: %s", fullPath)
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

// analyzeBusinessContext 分析业务上下文
func (g *CodeRulesGenerator) analyzeBusinessContext(fileContents []string) []string {
	var businessDomains []string
	domainMap := make(map[string]bool)

	// 分析文件内容中的业务关键词
	businessKeywords := []string{"订单", "支付", "用户", "商品", "库存", "交易", "账务", "通知"}

	for _, content := range fileContents {
		for _, keyword := range businessKeywords {
			if strings.Contains(content, keyword) {
				domainMap[keyword] = true
			}
		}
	}

	for domain := range domainMap {
		businessDomains = append(businessDomains, domain)
	}

	if len(businessDomains) == 0 {
		businessDomains = append(businessDomains, "通用业务处理")
	}

	return businessDomains
}

// getCodeRulesPagePrompt 获取代码规则页面生成提示词 - 简化版本
func (g *CodeRulesGenerator) getCodeRulesPagePrompt(data GenerateCodeRulesPromptData) (string, error) {
	templateName := "code_rules_page"
	return g.promptManager.GetPrompt(data, templateName)
}

// analyzeConfigurationFiles 分析配置文件
func (g *CodeRulesGenerator) analyzeConfigurationFiles(filePaths []string) []string {
	var configs []string
	configMap := make(map[string]bool)

	for _, path := range filePaths {
		if strings.Contains(path, "config") || strings.Contains(path, "Config") {
			configMap["配置文件"] = true
		}
		if strings.Contains(path, ".yaml") || strings.Contains(path, ".yml") {
			configMap["YAML配置"] = true
		}
		if strings.Contains(path, ".json") {
			configMap["JSON配置"] = true
		}
		if strings.Contains(path, ".toml") {
			configMap["TOML配置"] = true
		}
		if strings.Contains(path, ".env") {
			configMap["环境变量"] = true
		}
	}

	for config := range configMap {
		configs = append(configs, config)
	}
	return configs
}

// analyzeTestFiles 分析测试文件
func (g *CodeRulesGenerator) analyzeTestFiles(filePaths []string) []string {
	var tests []string
	testMap := make(map[string]bool)

	for _, path := range filePaths {
		if strings.Contains(path, "_test.go") {
			testMap["单元测试"] = true
		}
		if strings.Contains(path, "integration") || strings.Contains(path, "e2e") {
			testMap["集成测试"] = true
		}
		if strings.Contains(path, "mock") || strings.Contains(path, "Mock") {
			testMap["Mock测试"] = true
		}
		if strings.Contains(path, "benchmark") || strings.Contains(path, "Benchmark") {
			testMap["性能测试"] = true
		}
	}

	for test := range testMap {
		tests = append(tests, test)
	}
	return tests
}

// analyzeRuntimePatterns 分析运行时行为模式
func (g *CodeRulesGenerator) analyzeRuntimePatterns(fileContents []string) []string {
	var patterns []string
	patternMap := make(map[string]bool)

	for _, content := range fileContents {
		// 分析代码内容中的运行时行为
		if strings.Contains(content, "go func") || strings.Contains(content, "goroutine") {
			patternMap["并发处理"] = true
		}
		if strings.Contains(content, "sync.Mutex") || strings.Contains(content, "sync.RWMutex") {
			patternMap["锁机制"] = true
		}
		if strings.Contains(content, "channel") || strings.Contains(content, "make(chan") {
			patternMap["通道通信"] = true
		}
		if strings.Contains(content, "context.WithTimeout") || strings.Contains(content, "context.WithCancel") {
			patternMap["上下文控制"] = true
		}
		if strings.Contains(content, "defer") {
			patternMap["延迟执行"] = true
		}
		if strings.Contains(content, "select") {
			patternMap["多路复用"] = true
		}
		if strings.Contains(content, "atomic") {
			patternMap["原子操作"] = true
		}
		if strings.Contains(content, "once") || strings.Contains(content, "sync.Once") {
			patternMap["单次执行"] = true
		}
	}

	for pattern := range patternMap {
		patterns = append(patterns, pattern)
	}
	return patterns
}

// analyzeBusinessSemantics 分析业务语义
func (g *CodeRulesGenerator) analyzeBusinessSemantics(fileContents []string) []string {
	var semantics []string
	semanticMap := make(map[string]bool)

	for _, content := range fileContents {
		// 深度分析业务语义
		if strings.Contains(content, "entity") || strings.Contains(content, "Entity") {
			semanticMap["实体模型"] = true
		}
		if strings.Contains(content, "aggregate") || strings.Contains(content, "Aggregate") {
			semanticMap["聚合根"] = true
		}
		if strings.Contains(content, "value object") || strings.Contains(content, "ValueObject") {
			semanticMap["值对象"] = true
		}
		if strings.Contains(content, "domain event") || strings.Contains(content, "DomainEvent") {
			semanticMap["领域事件"] = true
		}
		if strings.Contains(content, "repository") || strings.Contains(content, "Repository") {
			semanticMap["仓储模式"] = true
		}
		if strings.Contains(content, "service") && strings.Contains(content, "domain") {
			semanticMap["领域服务"] = true
		}
		if strings.Contains(content, "specification") || strings.Contains(content, "Specification") {
			semanticMap["规格模式"] = true
		}
		if strings.Contains(content, "factory") && strings.Contains(content, "domain") {
			semanticMap["领域工厂"] = true
		}
	}

	for semantic := range semanticMap {
		semantics = append(semantics, semantic)
	}
	return semantics
}

// analyzeCodeTemplates 分析代码模板
func (g *CodeRulesGenerator) analyzeCodeTemplates(fileContents []string) []string {
	var templates []string
	templateMap := make(map[string]bool)

	for _, content := range fileContents {
		// 分析代码中的模板模式
		if strings.Contains(content, "interface") && strings.Contains(content, "type") {
			templateMap["接口定义模板"] = true
		}
		if strings.Contains(content, "struct") && strings.Contains(content, "type") {
			templateMap["结构体定义模板"] = true
		}
		if strings.Contains(content, "func") && strings.Contains(content, "error") {
			templateMap["错误处理函数模板"] = true
		}
		if strings.Contains(content, "New") && strings.Contains(content, "func") {
			templateMap["构造函数模板"] = true
		}
		if strings.Contains(content, "Handle") || strings.Contains(content, "Process") {
			templateMap["处理函数模板"] = true
		}
		if strings.Contains(content, "Validate") || strings.Contains(content, "Check") {
			templateMap["验证函数模板"] = true
		}
		if strings.Contains(content, "Convert") || strings.Contains(content, "Transform") {
			templateMap["转换函数模板"] = true
		}
		if strings.Contains(content, "Mock") && strings.Contains(content, "struct") {
			templateMap["Mock结构模板"] = true
		}
	}

	for template := range templateMap {
		templates = append(templates, template)
	}
	return templates
}

// analyzeDependencyNetwork 分析依赖网络
func (g *CodeRulesGenerator) analyzeDependencyNetwork(filePaths []string) []string {
	var dependencies []string
	depMap := make(map[string]bool)

	// 简化的依赖分析，基于文件路径和命名约定
	for _, path := range filePaths {
		if strings.Contains(path, "import") || strings.Contains(path, "require") {
			depMap["导入依赖"] = true
		}
		if strings.Contains(path, "vendor") || strings.Contains(path, "third_party") {
			depMap["第三方依赖"] = true
		}
		if strings.Contains(path, "internal") {
			depMap["内部依赖"] = true
		}
		if strings.Contains(path, "api") || strings.Contains(path, "proto") {
			depMap["接口依赖"] = true
		}
		if strings.Contains(path, "model") || strings.Contains(path, "entity") {
			depMap["模型依赖"] = true
		}
		if strings.Contains(path, "service") && !strings.Contains(path, "inter") {
			depMap["服务依赖"] = true
		}
	}

	for dep := range depMap {
		dependencies = append(dependencies, dep)
	}
	return dependencies
}

// generateFileUrl 生成文件URL
func (g *CodeRulesGenerator) generateFileUrl(filePath string, repoPath string) string {
	// 简单的文件URL生成逻辑，可以根据需要扩展
	return fmt.Sprintf("/file/%s", filePath)
}
