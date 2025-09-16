package wiki

import (
	"fmt"
	"net/url"
	"os"
)

// SimpleValidator 简单的验证器
type SimpleValidator struct{}

// NewSimpleValidator 创建新的简单验证器
func NewSimpleValidator() *SimpleValidator {
	return &SimpleValidator{}
}

// ValidateConfig 验证简化配置
func (v *SimpleValidator) ValidateConfig(config *SimpleConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	// 验证API密钥
	if config.APIKey == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	// 验证BaseURL
	if err := v.validateBaseURL(config.BaseURL); err != nil {
		return fmt.Errorf("invalid base URL: %w", err)
	}

	// 验证模型名称
	if config.Model == "" {
		return fmt.Errorf("model name cannot be empty")
	}

	// 验证语言设置
	if err := v.validateLanguage(config.Language); err != nil {
		return fmt.Errorf("invalid language: %w", err)
	}

	// 验证最大令牌数
	if config.MaxTokens <= 0 || config.MaxTokens > 100000 {
		return fmt.Errorf("max tokens must be between 1 and 100000")
	}

	// 验证温度参数
	if config.Temperature < 0.0 || config.Temperature > 2.0 {
		return fmt.Errorf("temperature must be between 0.0 and 2.0")
	}

	// 验证最大文件数
	if config.MaxFiles <= 0 || config.MaxFiles > 10000 {
		return fmt.Errorf("max files must be between 1 and 10000")
	}

	// 验证最大文件大小
	if config.MaxFileSize <= 0 || config.MaxFileSize > 100*1024*1024 {
		return fmt.Errorf("max file size must be between 1 and 100MB")
	}

	// 验证输出目录
	if config.OutputDir == "" {
		return fmt.Errorf("output directory cannot be empty")
	}

	return nil
}

// validateBaseURL 验证基础URL
func (v *SimpleValidator) validateBaseURL(baseURL string) error {
	if baseURL == "" {
		return fmt.Errorf("base URL cannot be empty")
	}

	// 解析URL
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	// 检查协议
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("URL scheme must be http or https")
	}

	// 检查主机
	if parsedURL.Host == "" {
		return fmt.Errorf("URL host cannot be empty")
	}

	return nil
}

// validateLanguage 验证语言设置
func (v *SimpleValidator) validateLanguage(language string) error {
	if language == "" {
		return fmt.Errorf("language cannot be empty")
	}

	// 支持的语言列表
	supportedLanguages := map[string]bool{
		"en":    true,
		"zh":    true,
		"zh-CN": true,
		"zh-tw": true,
		"ja":    true,
		"es":    true,
		"kr":    true,
		"vi":    true,
		"pt-br": true,
		"fr":    true,
		"ru":    true,
	}

	if !supportedLanguages[language] {
		return fmt.Errorf("unsupported language: %s", language)
	}

	return nil
}

// ValidateRepositoryPath 验证仓库路径
func (v *SimpleValidator) ValidateRepositoryPath(repoPath string) error {
	if repoPath == "" {
		return fmt.Errorf("repository path cannot be empty")
	}

	// 检查路径是否存在
	if _, err := os.Stat(repoPath); err != nil {
		return fmt.Errorf("repository path does not exist: %w", err)
	}

	return nil
}

// ValidateExportFormat 验证导出格式
func (v *SimpleValidator) ValidateExportFormat(format string) error {
	if format == "" {
		return fmt.Errorf("export format cannot be empty")
	}

	supportedFormats := map[string]bool{
		"markdown": true,
		"json":     true,
	}

	if !supportedFormats[format] {
		return fmt.Errorf("unsupported export format: %s", format)
	}

	return nil
}

// DefaultValidator 默认验证器
var DefaultValidator = NewSimpleValidator()
