package packageclassifier

import (
	"codebase-indexer/pkg/codegraph/workspace"
	"strings"
)

// CppClassifier C++包分类器
type CppClassifier struct {
	systemHeaders map[string]bool
}

// NewCppClassifier 创建C++分类器
func NewCppClassifier() *CppClassifier {
	classifier := &CppClassifier{
		systemHeaders: make(map[string]bool),
	}

	// 加载默认系统头文件
	defaultHeaders := []string{
		"iostream", "vector", "string", "algorithm", "memory", "map", "set",
		"unordered_map", "unordered_set", "tuple", "functional", "thread",
		"mutex", "condition_variable", "future", "chrono", "regex", "fstream",
		"sstream", "iomanip", "array", "bitset", "deque", "forward_list",
		"list", "queue", "stack", "valarray", "complex", "exception",
		"initializer_list", "ios", "iosfwd", "istream", "limits", "locale",
		"ostream", "stdexcept", "streambuf", "typeinfo", "utility", "atomic",
		"cfenv", "codecvt", "csetjmp", "csignal", "cstdarg", "cstddef",
		"cstdint", "cstdio", "cstdlib", "cstring", "ctgmath", "ctime", "cwchar",
		"cwctype", "execution", "filesystem", "random", "ratio", "scoped_allocator",
		"shared_mutex", "strstream", "system_error", "type_traits",
	}

	// 合并默认和配置中的系统头文件
	for _, header := range defaultHeaders {
		classifier.systemHeaders[header] = true
	}

	return classifier
}

func (cpp *CppClassifier) Classify(packageName string, project *workspace.Project) PackageType {
	// 移除可能的尖括号或引号
	cleanedName := strings.Trim(packageName, "<>\"'")

	// C++17及以上可能没有.h扩展名
	cleanedName = strings.TrimSuffix(cleanedName, ".h")

	// 检查是否为C++系统头文件
	if cpp.systemHeaders[cleanedName] {
		return SystemPackage
	}

	// 检查是否为C标准库（在C++中使用）
	cSystemHeaders := map[string]bool{
		"stdio": true, "stdlib": true, "string": true, "math": true,
		"ctype": true, "time": true, "stdint": true, "stdbool": true,
	}
	if cSystemHeaders[cleanedName] {
		return SystemPackage
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

// CppClassifierFactory C++分类器工厂
type CppClassifierFactory struct{}

func (f *CppClassifierFactory) CreateClassifier() Classifier {
	return NewCppClassifier()
}
