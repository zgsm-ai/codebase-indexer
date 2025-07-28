package utils

import (
	"errors"
	"path/filepath"
	"strings"
)

// ValidateCodebaseID 验证代码库ID格式
func ValidateCodebaseID(id string) bool {
	if id == "" {
		return false
	}
	if len(id) < 3 || len(id) > 50 {
		return false
	}
	// 只允许字母、数字、连字符和下划线
	for _, char := range id {
		if !(char >= 'a' && char <= 'z') && !(char >= 'A' && char <= 'Z') &&
			!(char >= '0' && char <= '9') && char != '-' && char != '_' {
			return false
		}
	}
	return true
}

// SanitizePath 清理文件路径，防止目录遍历攻击
func SanitizePath(path string) (string, error) {
	if path == "" {
		return "", nil
	}

	// 清理路径
	cleanPath := filepath.Clean(path)

	// 检查是否包含相对路径
	if strings.Contains(cleanPath, "..") {
		return "", ErrInvalidPath
	}

	// 确保路径以/开头
	if !strings.HasPrefix(cleanPath, "/") {
		cleanPath = "/" + cleanPath
	}

	return cleanPath, nil
}

// ValidateFilePath 验证文件路径
func ValidateFilePath(path string) bool {
	if path == "" {
		return false
	}

	// 检查路径长度
	if len(path) > 1024 {
		return false
	}

	// 检查是否包含非法字符
	illegalChars := []string{"\x00", "..", "~", "\\"}
	for _, char := range illegalChars {
		if strings.Contains(path, char) {
			return false
		}
	}

	return true
}

// ValidatePageParams 验证分页参数
func ValidatePageParams(page, size int) (int, int) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	return page, size
}

// ValidateTreeDepth 验证目录树深度
func ValidateTreeDepth(depth int) int {
	if depth < 1 {
		depth = 1
	}
	if depth > 10 {
		depth = 10
	}
	return depth
}

// ValidateSymbolName 验证符号名称
func ValidateSymbolName(symbol string) bool {
	if symbol == "" {
		return false
	}
	if len(symbol) > 255 {
		return false
	}
	return true
}

// ValidateLanguage 验证编程语言
func ValidateLanguage(language string) bool {
	validLanguages := map[string]bool{
		"go":          true,
		"javascript":  true,
		"typescript":  true,
		"python":      true,
		"java":        true,
		"c":           true,
		"cpp":         true,
		"csharp":      true,
		"php":         true,
		"ruby":        true,
		"rust":        true,
		"kotlin":      true,
		"scala":       true,
		"swift":       true,
		"objective-c": true,
	}

	if language == "" {
		return true // 允许空值
	}

	return validLanguages[strings.ToLower(language)]
}

// Error definitions
var (
	ErrInvalidPath = errors.New("invalid file path")
)
