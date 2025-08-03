package packageclassifier

import (
	"codebase-indexer/pkg/codegraph/types"
	"codebase-indexer/pkg/codegraph/workspace"
	"strings"
)

// GoClassifier Go包分类器
type GoClassifier struct {
	systemPackages map[string]bool
}

// NewGoClassifier 创建Go分类器
func NewGoClassifier() *GoClassifier {
	classifier := &GoClassifier{
		systemPackages: make(map[string]bool),
	}

	// 加载默认系统包
	defaultPackages := []string{
		"fmt", "os", "io", "bufio", "strings", "strconv", "time",
		"encoding/json", "sync", "net/http", "database/sql",
		"reflect", "errors", "context", "flag", "log", "math",
		"sort", "unicode", "bytes", "crypto", "encoding/base64",
		"encoding/csv", "encoding/xml", "hash", "html", "image",
		"index/suffixarray", "io/ioutil", "net", "os/exec", "path",
		"regexp", "runtime", "syscall", "testing", "text/template",
		"unicode/utf8", "unsafe",
	}

	// 合并默认和配置中的系统包
	for _, pkg := range defaultPackages {
		classifier.systemPackages[pkg] = true
	}

	return classifier
}

func (g *GoClassifier) Classify(packageName string, project *workspace.Project) PackageType {
	// 检查是否为系统包
	// 系统包是 Go 标准库中的包，如 fmt, os, io 等
	if g.systemPackages[packageName] {
		return SystemPackage
	}

	// 检查项目模块信息
	goModule := project.GoModules
	if goModule != types.EmptyString {
		// 如果包名以项目模块名开头，则为项目包
		// 例如：项目模块为 "github.com/example/myapp"，则 "github.com/example/myapp/utils" 是项目包
		if strings.HasPrefix(packageName, goModule+types.Slash) {
			return ProjectPackage
		}
	}

	// 检查是否为项目内包（相对路径）
	// 相对路径的包通常是项目内部的包，如 "./utils", "../common" 等
	if strings.HasPrefix(packageName, "./") || strings.HasPrefix(packageName, "../") {
		return ProjectPackage
	}

	// 对于不匹配系统包和项目包的情况，返回 UnknownPackage
	// 这样可以避免将不确定的包错误地归类为第三方包
	// 例如：一些特殊的包名或者无法识别的包路径
	return UnknownPackage
}

// GoClassifierFactory Go分类器工厂
type GoClassifierFactory struct{}

func (f *GoClassifierFactory) CreateClassifier() Classifier {
	return NewGoClassifier()
}
