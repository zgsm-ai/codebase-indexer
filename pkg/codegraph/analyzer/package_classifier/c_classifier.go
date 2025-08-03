package packageclassifier

import (
	"codebase-indexer/pkg/codegraph/workspace"
	"strings"
)

// CClassifier C包分类器
type CClassifier struct {
	systemHeaders map[string]bool
}

// NewCClassifier 创建C分类器
func NewCClassifier() *CClassifier {
	classifier := &CClassifier{
		systemHeaders: make(map[string]bool),
	}

	// 加载默认系统头文件
	defaultHeaders := []string{
		"stdio.h", "stdlib.h", "string.h", "math.h", "ctype.h", "time.h",
		"stdint.h", "stdbool.h", "assert.h", "limits.h", "errno.h", "signal.h",
		"pthread.h", "sys/stat.h", "fcntl.h", "unistd.h", "stddef.h", "setjmp.h",
		"stdarg.h", "float.h", "iso646.h", "locale.h", "wchar.h", "wctype.h",
	}

	// 合并默认和配置中的系统头文件
	for _, header := range defaultHeaders {
		classifier.systemHeaders[header] = true
	}

	return classifier
}

func (c *CClassifier) Classify(packageName string, project *workspace.Project) PackageType {
	// 移除可能的尖括号或引号
	cleanedName := strings.Trim(packageName, "<>\"'")

	// 检查是否为系统头文件
	if c.systemHeaders[cleanedName] {
		return SystemPackage
	}

	// 检查是否为系统库的其他形式（如带路径的系统头文件）
	systemPathPatterns := []string{
		"/usr/include/", "/usr/local/include/",
		"/Library/Developer/CommandLineTools/usr/include/",
	}
	for _, pattern := range systemPathPatterns {
		if strings.Contains(cleanedName, pattern) {
			return SystemPackage
		}
	}

	// 检查是否为项目内包
	if project != nil && len(project.CppIncludes) > 0 {
		for _, includePath := range project.CppIncludes {
			// 检查包名是否包含项目包含路径
			if strings.Contains(cleanedName, includePath) {
				return ProjectPackage
			}
			// 检查是否为项目相对路径（如 "src/utils.h" 或 "../common.h"）
			if strings.HasPrefix(cleanedName, includePath+"/") ||
			   strings.HasPrefix(cleanedName, "../"+includePath) ||
			   strings.HasPrefix(cleanedName, "./"+includePath) {
				return ProjectPackage
			}
		}
	}

	// 其他情况视为未知包，而不是直接判断为第三方包
	return UnknownPackage
}

// CClassifierFactory C分类器工厂
type CClassifierFactory struct{}

func (f *CClassifierFactory) CreateClassifier() Classifier {
	return NewCClassifier()
}
