package wiki

import (
	"bytes"
	"codebase-indexer/pkg/codegraph/types"
	"embed"
	"fmt"
	"text/template"
)

//go:embed templates/*.tmpl
var promptTemplateFS embed.FS

// TemplateType 模板类型
type TemplateType string

const (
	TemplateTypeCodeRulesPage      TemplateType = "code_rules_page"
	TemplateTypeCodeRulesStructure TemplateType = "code_rules_structure"
	TemplateTypeWikiPage           TemplateType = "wiki_page"
	TemplateTypeWikiStructure      TemplateType = "wiki_structure"
)

// languageMap 定义语言代码到显示名称的映射
var languageMap = map[string]string{
	"en":    "English",
	"ja":    "Japanese (日本語)",
	"zh":    "Mandarin Chinese (中文)",
	"zh-tw": "Traditional Chinese (繁體中文)",
	"es":    "Spanish (Español)",
	"kr":    "Korean (한국어)",
	"vi":    "Vietnamese (Tiếng Việt)",
	"pt-br": "Brazilian Portuguese (Português Brasileiro)",
	"fr":    "Français (French)",
	"ru":    "Русский (Russian)",
}

// getOutputLanguage 根据语言代码获取输出语言名称
func getOutputLanguage(language string) string {
	if lang, exists := languageMap[language]; exists {
		return lang
	}
	return languageMap["en"] // 默认英语
}

// PromptManager prompt管理器
type PromptManager struct {
}

// NewPromptManager 创建新的prompt管理器
func NewPromptManager() *PromptManager {
	pm := &PromptManager{}
	return pm
}

const templatesDir = "templates"

// loadTemplate 检查是否存在指定模式
func (pm *PromptManager) loadTemplate(name string) (string, error) {
	file, err := promptTemplateFS.ReadFile(templatesDir + types.Slash + name + ".tmpl")
	return string(file), err
}

// getPrompt 获取Wiki生成提示词，支持模式选择
func (pm *PromptManager) getPrompt(data any, promptName string) (string, error) {
	// 如果未指定模式或模式不存在，使用默认模式
	templateContent, err := pm.loadTemplate(promptName)
	if err != nil {
		return "", err
	}

	// 解析并执行模板
	return pm.executeTemplate(data, promptName, templateContent)
}

func (pm *PromptManager) getStructurePrompt(data any, documentType DocumentType) (string, error) {
	var templateName string
	switch documentType {
	case DocTypeWiki:
		templateName = string(TemplateTypeWikiStructure)
	case DocTypeCodeRules:
		templateName = string(TemplateTypeCodeRulesStructure)
	default:
		return "", fmt.Errorf("unknown document type: %s", documentType)
	}
	return pm.getPrompt(data, templateName)
}

func (pm *PromptManager) getPagePrompt(data any, documentType DocumentType) (string, error) {
	var templateName string
	switch documentType {
	case DocTypeWiki:
		templateName = string(TemplateTypeWikiPage)
	case DocTypeCodeRules:
		templateName = string(TemplateTypeCodeRulesPage)
	default:
		return "", fmt.Errorf("unknown document type: %s", documentType)
	}

	return pm.getPrompt(data, templateName)
}

// executeTemplate 执行模板并返回结果
func (pm *PromptManager) executeTemplate(data any, templateName string, templateContent string) (string, error) {
	// 解析模板
	tmpl, err := template.New(templateName).Parse(templateContent)
	if err != nil {
		// 如果模板解析失败，返回错误信息
		return "", fmt.Errorf("error parsing template: %w", err)
	}

	// 执行模板
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("error executing template: %w", err)
	}

	return buf.String(), nil
}
