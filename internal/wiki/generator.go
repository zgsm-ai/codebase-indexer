package wiki

import (
	"codebase-indexer/pkg/logger"
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
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
	GenerateDocumentWithProgress(ctx context.Context, repoPath string, callback ProgressCallback) (*DocumentStructure, error)
	GetDocumentType() DocumentType
	Close() error
}

// DocumentStructure 通用文档结构
type DocumentStructure struct {
	ID           string
	Title        string
	Description  string
	Pages        []DocumentPage
	Sections     []DocumentSection
	RootSections []string
	Metadata     map[string]interface{} // 扩展元数据
}

// DocumentPage 文档页面
type DocumentPage struct {
	ID           string
	Title        string
	Content      string
	FilePaths    []string
	Importance   string
	RelatedPages []string
	ParentID     string
	Metadata     map[string]interface{} // 扩展元数据
	Description  string
}

// DocumentSection 文档章节
type DocumentSection struct {
	ID          string
	Title       string
	Pages       []string
	Subsections []string
	Metadata    map[string]interface{} // 扩展元数据
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

// CommonGenerateLogic 通用生成逻辑
func (g *BaseGenerator) CommonGenerateLogic(ctx context.Context, repoPath string, callback ProgressCallback) (*DocumentStructure, []*FileMeta, string, string, error) {
	// 重置性能统计
	g.resetPerformanceStats()

	g.logger.Info("Starting to generate %s document: %s", g.documentType, repoPath)

	if callback == nil {
		callback = LogProgressCallback
	}

	callback(g.logger, 0.0, "file_processing", "Starting to process repository files")

	// 1. 处理仓库文件
	g.startStage()
	files, err := g.processor.ProcessRepository(ctx, repoPath)
	g.endStage("file_processing")

	if err != nil {
		callback(g.logger, 0.0, "error", fmt.Sprintf("Failed to process repository: %v", err))
		g.outputErrorStats(callback)
		return nil, nil, "", "", fmt.Errorf("failed to process repository: %w", err)
	}

	if len(files) == 0 {
		callback(g.logger, 0.0, "error", "No files found in repository")
		g.outputErrorStats(callback)
		return nil, nil, "", "", fmt.Errorf("no files found in repository")
	}

	fileProgress := float64(len(files)) / float64(len(files))
	callback(g.logger, fileProgress, "file_processing_complete", fmt.Sprintf("Successfully processed %d files", len(files)))

	// 2. 生成文件树和README内容
	callback(g.logger, fileProgress, "structure_generation", "Generating file tree and README content")
	g.startStage()
	fileTree, readmeContent, err := g.generateFileTreeAndReadme(ctx, repoPath, files)
	if err != nil {
		callback(g.logger, fileProgress, "error", fmt.Sprintf("Failed to generate file tree and README: %v", err))
		g.outputErrorStats(callback)
		return nil, nil, "", "", fmt.Errorf("failed to generate file tree and readme: %w", err)
	}

	return &DocumentStructure{}, files, fileTree, readmeContent, nil
}

// generateFileTreeAndReadme 生成文件树和README内容
func (g *BaseGenerator) generateFileTreeAndReadme(ctx context.Context, repoPath string, fileMetas []*FileMeta) (string, string, error) {
	var fileTreeBuilder strings.Builder
	for _, fileMeta := range fileMetas {
		fileTreeBuilder.WriteString(fileMeta.Path)
		fileTreeBuilder.WriteString("\n")
	}
	fileTree := fileTreeBuilder.String()

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

	return fileTree, readmeContent, nil
}

// generateDocumentStructure 生成文档结构（由子类实现具体逻辑）
func (g *BaseGenerator) generateDocumentStructure(ctx context.Context, fileTree string, readmeContent string, repoPath string, fileMetas []*FileMeta) (*DocumentStructure, error) {
	// 基础实现，子类应该重写此方法
	return &DocumentStructure{
		ID:    string(g.documentType),
		Title: fmt.Sprintf("%s Documentation", filepath.Base(repoPath)),
		Pages: []DocumentPage{},
	}, nil
}

// generatePageContent 生成页面内容
func (g *BaseGenerator) generatePageContent(ctx context.Context, pages []DocumentPage, repoPath string, fileMetas []*FileMeta, callback ProgressCallback) error {
	if callback == nil {
		callback = LogProgressCallback
	}

	maxConcurrent := g.config.Concurrency
	if maxConcurrent <= 0 {
		maxConcurrent = 1
	}

	completedPages := 0
	totalPages := len(pages)

	for i, page := range pages {
		err := g.generateSinglePageContent(ctx, &page, repoPath, fileMetas)
		pages[i] = page
		if err != nil {
			g.logger.Error("failed to generate content for page %s: %v", page.ID, err)
		} else {
			completedPages++
			progress := float64(completedPages) / float64(totalPages)
			callback(g.logger, progress, "content_generation",
				fmt.Sprintf("Generated page: %s (%d/%d)", page.Title, completedPages, totalPages))
		}
	}

	return nil
}

// generateSinglePageContent 生成单个页面内容
func (g *BaseGenerator) generateSinglePageContent(ctx context.Context, page *DocumentPage, repoPath string, fileMetas []*FileMeta) error {
	if page.Content != "" {
		return nil
	}

	// 加载相关文件内容
	var fileLinks []string
	for _, filePath := range page.FilePaths {
		fileUrl := g.generateFileUrl(filePath, repoPath)
		fileLinks = append(fileLinks, fmt.Sprintf("- [%s](%s)", filePath, fileUrl))
	}

	// 生成提示词
	promptTemplate := string(g.documentType) + "_page"

	var prompt string
	var err error

	// 根据文档类型使用不同的数据结构
	switch g.documentType {
	case DocTypeCodeRules:
		data := GenerateCodeRulesPromptData{
			PageTitle:      page.Title,
			FileLinks:      strings.Join(fileLinks, "\n"),
			OutputLanguage: getOutputLanguage(g.config.Language),
			GuidelineCount: "15-25", // 默认值
		}
		prompt, err = g.promptManager.GetPrompt(data, promptTemplate)
	default:
		data := GenerateWikiPromptData{
			PageTitle:      page.Title,
			FileLinks:      strings.Join(fileLinks, "\n"),
			OutputLanguage: getOutputLanguage(g.config.Language),
		}
		prompt, err = g.promptManager.GetPrompt(data, promptTemplate)
	}
	if err != nil {
		return fmt.Errorf("failed to generate prompt: %w", err)
	}

	// 调用LLM生成内容
	startTime := time.Now()
	content, err := g.llmClient.GenerateContent(ctx, prompt, g.config.MaxTokens, g.config.Temperature)
	duration := time.Since(startTime)

	g.recordLLMCall(duration, "content", err == nil)

	if err != nil {
		return fmt.Errorf("failed to generate page content: %w", err)
	}

	page.Content = content
	return nil
}

// generateFileUrl 生成文件URL
func (g *BaseGenerator) generateFileUrl(filePath string, repoPath string) string {
	relPath, err := filepath.Rel(repoPath, filePath)
	if err != nil {
		return filePath
	}
	return relPath
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

func (g *BaseGenerator) outputErrorStats(callback ProgressCallback) {
	g.performance.Finish()
	performanceStats := g.getPerformanceStats()
	llmStats := g.getLLMStats()

	performanceMessage := fmt.Sprintf("Performance Stats - Total Duration: %v", performanceStats.TotalDuration)
	llmMessage := llmStats.String()

	callback(g.logger, 1.0, "performance_stats", performanceMessage)
	callback(g.logger, 1.0, "llm_stats", llmMessage)
}

func (g *BaseGenerator) getPerformanceStats() *PerformanceStats {
	return g.performance
}

func (g *BaseGenerator) getLLMStats() *LLMCallStats {
	return g.llmStats
}

// ParseXMLResponse 通用XML解析方法
func (g *BaseGenerator) ParseXMLResponse(xmlText string, rootTag string, parseFunc func(string) interface{}) (interface{}, error) {
	// 清理markdown分隔符
	xmlText = strings.ReplaceAll(xmlText, "```xml", "")
	xmlText = strings.ReplaceAll(xmlText, "```", "")

	// 提取XML内容
	xmlMatch := regexp.MustCompile(`<` + rootTag + `>[\s\S]*?</` + rootTag + `>`).FindString(xmlText)
	if xmlMatch == "" {
		return nil, fmt.Errorf("no valid XML found in response")
	}

	result := parseFunc(xmlMatch)
	if result == nil {
		return nil, fmt.Errorf("failed to parse XML content")
	}

	return result, nil
}
