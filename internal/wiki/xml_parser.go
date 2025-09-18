package wiki

import (
	"codebase-indexer/pkg/codegraph/types"
	"fmt"
	"regexp"
	"strings"

	"codebase-indexer/pkg/logger"
)

// XMLParser XML解析器 - 简化版本，只处理结构模板
type XMLParser struct {
	logger logger.Logger
}

// NewXMLParser 创建XML解析器
func NewXMLParser(logger logger.Logger) *XMLParser {
	return &XMLParser{
		logger: logger,
	}
}

// ExtractXMLContent 提取XML内容，移除markdown代码块标记
func (p *XMLParser) ExtractXMLContent(content string) string {
	// 清理markdown分隔符
	content = strings.ReplaceAll(content, "```xml", "")
	content = strings.ReplaceAll(content, "```", "")
	return strings.TrimSpace(content)
}

// ParseDocumentStructure 解析Wiki结构XML
func (p *XMLParser) ParseDocumentStructure(xmlText string) (*DocumentStructure, error) {
	cleanedXML := p.ExtractXMLContent(xmlText)

	// 提取根元素
	xmlMatch := regexp.MustCompile(`<document_structure>[\s\S]*?</document_structure>`).FindString(cleanedXML)
	if xmlMatch == "" {
		return nil, fmt.Errorf("no valid document_structure XML found")
	}

	docStructure := &DocumentStructure{
		Pages:        []DocumentPage{},
		Sections:     []DocumentSection{},
		RootSections: []string{},
		Metadata:     make(map[string]interface{}),
	}

	// 提取基本信息
	docStructure.Title = p.extractField(xmlMatch, "title")
	docStructure.Description = p.extractField(xmlMatch, "description")

	// 解析页面
	pageMatches := regexp.MustCompile(`<page[^>]*>[\s\S]*?</page>`).FindAllString(xmlMatch, -1)
	for _, pageMatch := range pageMatches {
		page := p.parsePageXML(pageMatch)
		if page != nil {
			docStructure.Pages = append(docStructure.Pages, *page)
		}
	}

	// 解析章节
	sectionMatches := regexp.MustCompile(`<section[^>]*>[\s\S]*?</section>`).FindAllString(xmlMatch, -1)
	for _, sectionMatch := range sectionMatches {
		section := p.parseSectionXML(sectionMatch)
		if section != nil {
			docStructure.Sections = append(docStructure.Sections, *section)
		}
	}

	return docStructure, nil
}

// parsePageXML 解析页面XML
func (p *XMLParser) parsePageXML(xmlText string) *DocumentPage {
	page := &DocumentPage{
		Metadata: make(map[string]interface{}),
	}

	// 提取属性
	page.ID = p.extractAttribute(xmlText, "page", "id")
	page.Title = p.extractField(xmlText, "title")
	page.Description = p.extractField(xmlText, "description")
	page.Importance = p.extractField(xmlText, "importance")
	page.ParentID = p.extractField(xmlText, "parent_category")
	page.ParentID = p.extractField(xmlText, "parent_section") // 兼容wiki结构

	// 提取文件路径
	page.FilePaths = p.extractFields(xmlText, "file_path")

	// 提取相关页面
	page.RelatedPages = p.extractFields(xmlText, "related")

	return page
}

// parseSectionXML 解析章节XML
func (p *XMLParser) parseSectionXML(xmlText string) *DocumentSection {
	section := &DocumentSection{
		Metadata: make(map[string]interface{}),
	}

	section.ID = p.extractAttribute(xmlText, "section", "id")
	section.Title = p.extractField(xmlText, "title")

	// 提取页面引用
	pageRefs := p.extractFields(xmlText, "page_ref")
	for _, pageRef := range pageRefs {
		section.Pages = append(section.Pages, pageRef)
	}

	// 提取子章节引用
	subsectionRefs := p.extractFields(xmlText, "section_ref")
	for _, subsectionRef := range subsectionRefs {
		section.Subsections = append(section.Subsections, subsectionRef)
	}

	return section
}

// extractField 提取字段值
func (p *XMLParser) extractField(xmlText, fieldName string) string {
	pattern := fmt.Sprintf(`<%s>([^<]+)</%s>`, fieldName, fieldName)
	if match := regexp.MustCompile(pattern).FindStringSubmatch(xmlText); len(match) > 1 {
		return strings.TrimSpace(match[1])
	}
	return ""
}

// extractFields 提取所有字段值
func (p *XMLParser) extractFields(xmlText, fieldName string) []string {
	var results []string
	pattern := fmt.Sprintf(`<%s>([^<]+)</%s>`, fieldName, fieldName)
	matches := regexp.MustCompile(pattern).FindAllStringSubmatch(xmlText, -1)
	for _, match := range matches {
		if len(match) > 1 {
			results = append(results, strings.TrimSpace(match[1]))
		}
	}
	return results
}

// extractAttribute 提取属性值
func (p *XMLParser) extractAttribute(xmlText, tagName, attrName string) string {
	pattern := fmt.Sprintf(`<%s[^>]*%s\s*=\s*["']([^"']*)["'][^>]*>`, tagName, attrName)
	if match := regexp.MustCompile(pattern).FindStringSubmatch(xmlText); len(match) > 1 {
		return strings.TrimSpace(match[1])
	}
	return types.EmptyString
}
