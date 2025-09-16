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

	err = g.generatePageContent(ctx, docStructure.Pages, repoPath, files, callback)
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
	prompt, err := g.getCodeRulesStructurePrompt(fileTree, readmeContent, filePaths)
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

// getCodeRulesStructurePrompt 获取代码规则结构生成提示词
func (g *CodeRulesGenerator) getCodeRulesStructurePrompt(fileTree string, readmeContent string, filePaths []string) (string, error) {
	outputLang := getOutputLanguage(g.config.Language)

	// 设置页面数量和指南数量
	pageCount := "8-12"
	guidelineCount := "15-25"

	data := GenerateCodeRulesStructPromptData{
		FileTree:       fileTree,
		ReadmeContent:  readmeContent,
		OutputLanguage: outputLang,
		PageCount:      pageCount,
		GuidelineCount: guidelineCount,
	}

	return g.promptManager.GetPrompt(data, "code_rules_structure")
}

// parseCodeRulesStructureXML 解析代码规则结构XML
func (g *CodeRulesGenerator) parseCodeRulesStructureXML(xmlText string) (*DocumentStructure, error) {
	// 清理markdown分隔符
	xmlText = strings.ReplaceAll(xmlText, "```xml", "")
	xmlText = strings.ReplaceAll(xmlText, "```", "")

	// 提取XML内容
	xmlMatch := regexp.MustCompile(`<coding_rules>[\s\S]*?</coding_rules>`).FindString(xmlText)
	if xmlMatch == "" {
		return nil, fmt.Errorf("no valid code rules XML found in response")
	}

	// 创建文档结构
	docStructure := &DocumentStructure{
		ID:           "code_rules",
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

	// 提取sections
	sectionMatches := regexp.MustCompile(`<section[^>]*>[\s\S]*?</section>`).FindAllString(xmlMatch, -1)
	for _, sectionMatch := range sectionMatches {
		section := g.parseCodeRulesSectionXML(sectionMatch)
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

	// 提取guidelines作为页面
	guidelineMatches := regexp.MustCompile(`<guideline[^>]*>[\s\S]*?</guideline>`).FindAllString(xmlMatch, -1)
	for _, guidelineMatch := range guidelineMatches {
		page := g.parseCodeRulesGuidelineXML(guidelineMatch)
		if page != nil {
			docStructure.Pages = append(docStructure.Pages, *page)
		}
	}

	return docStructure, nil
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

// parseCodeRulesGuidelineXML 解析代码规则指南XML（转换为页面）
func (g *CodeRulesGenerator) parseCodeRulesGuidelineXML(xmlText string) *DocumentPage {
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

	// 提取related_guidelines
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

	// 存储完整的指南内容在metadata中，用于后续页面生成
	page.Metadata["guideline_xml"] = xmlText

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
