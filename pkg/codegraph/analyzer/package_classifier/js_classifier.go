package packageclassifier

import (
	"codebase-indexer/pkg/codegraph/workspace"
	"strings"
)

// JavaScriptClassifier JavaScript包分类器
type JavaScriptClassifier struct {
	systemModules map[string]bool
}

// NewJavaScriptClassifier 创建JavaScript分类器
func NewJavaScriptClassifier() *JavaScriptClassifier {
	classifier := &JavaScriptClassifier{
		systemModules: make(map[string]bool),
	}

	// 加载默认系统模块
	defaultModules := []string{
		"fs", "path", "os", "process", "util", "events", "stream", "http",
		"https", "url", "querystring", "buffer", "crypto", "zlib",
		"child_process", "net", "tls", "dns", "readline", "repl", "vm",
		"assert", "cluster", "console", "constants", "debugger", "dgram",
		"domain", "fs/promises", "http2", "inspector", "module", "perf_hooks",
		"punycode", "stream/promises", "string_decoder", "sys", "timers",
		"timers/promises", "trace_events", "tty", "v8", "worker_threads",
	}

	// 合并默认和配置中的系统模块
	for _, module := range defaultModules {
		classifier.systemModules[module] = true
	}

	return classifier
}

func (js *JavaScriptClassifier) Classify(packageName string, project *workspace.Project) PackageType {
	// 检查是否为核心模块
	if js.systemModules[packageName] {
		return SystemPackage
	}

	// 检查是否为项目内模块（相对路径）
	if strings.HasPrefix(packageName, "./") || strings.HasPrefix(packageName, "../") {
		return ProjectPackage
	}

	// 检查是否为项目内包（通过 project.JsPackages 判断）
	if project != nil && len(project.JsPackages) > 0 {
		for _, jsPackage := range project.JsPackages {
			if packageName == jsPackage || strings.HasPrefix(packageName, jsPackage+"/") {
				return ProjectPackage
			}
		}
	}

	// 其他情况视为未知包，而不是直接判断为第三方包
	return UnknownPackage
}

// JavaScriptClassifierFactory JavaScript分类器工厂
type JavaScriptClassifierFactory struct{}

func (f *JavaScriptClassifierFactory) CreateClassifier() Classifier {
	return NewJavaScriptClassifier()
}

// TypeScriptClassifier TypeScript包分类器
type TypeScriptClassifier struct {
	jsClassifier  Classifier
	systemModules map[string]bool
}

// NewTypeScriptClassifier 创建TypeScript分类器
func NewTypeScriptClassifier() *TypeScriptClassifier {
	classifier := &TypeScriptClassifier{
		jsClassifier:  NewJavaScriptClassifier(),
		systemModules: make(map[string]bool),
	}

	// TypeScript特有的系统模块
	defaultModules := []string{
		"typescript", "@types/node",
	}

	// 合并默认和配置中的系统模块
	for _, module := range defaultModules {
		classifier.systemModules[module] = true
	}

	return classifier
}

func (ts *TypeScriptClassifier) Classify(packageName string, project *workspace.Project) PackageType {
	// 检查TypeScript特有的系统模块
	if ts.systemModules[packageName] {
		return SystemPackage
	}

	// 检查是否为项目内模块（相对路径）
	if strings.HasPrefix(packageName, "./") || strings.HasPrefix(packageName, "../") {
		return ProjectPackage
	}

	// 检查是否为项目内包（通过 project.JsPackages 判断）
	if project != nil && len(project.JsPackages) > 0 {
		for _, jsPackage := range project.JsPackages {
			if packageName == jsPackage || strings.HasPrefix(packageName, jsPackage+"/") {
				return ProjectPackage
			}
		}
	}

	// 其他情况视为未知包，而不是直接判断为第三方包
	return UnknownPackage
}

// TypeScriptClassifierFactory TypeScript分类器工厂
type TypeScriptClassifierFactory struct{}

func (f *TypeScriptClassifierFactory) CreateClassifier() Classifier {
	return NewTypeScriptClassifier()
}
