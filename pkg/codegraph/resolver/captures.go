package resolver

import (
	"codebase-indexer/pkg/codegraph/types"
	"strings"
)

const (
	dotName       = ".name"
	dotArguments  = ".arguments"
	dotParameters = ".parameters"
	dotOwner      = ".owner"
	dotSource     = ".source"
	dotAlias      = ".alias"
)

// 函数工厂：生成检查字符串是否以特定后缀结尾的函数
func createSuffixChecker(suffix string) func(string) bool {
	return func(captureName string) bool {
		return strings.HasSuffix(captureName, suffix)
	}
}

// 使用工厂函数创建检查器
var (
	IsNameCapture       = createSuffixChecker(dotName)
	IsParametersCapture = createSuffixChecker(dotParameters)
	IsArgumentsCapture  = createSuffixChecker(dotArguments)
	IsOwnerCapture      = createSuffixChecker(dotOwner)
	IsSourceCapture     = createSuffixChecker(dotSource)
	IsAliasCapture      = createSuffixChecker(dotAlias)
)

// IsElementNameCapture 名称捕获
func IsElementNameCapture(elementType types.ElementType, captureName string) bool {
	return IsNameCapture(captureName) &&
		captureName == string(elementType)+dotName
}
