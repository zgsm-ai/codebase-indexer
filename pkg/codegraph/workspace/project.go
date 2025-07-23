package workspace

import (
	"codebase-indexer/pkg/codegraph/lang"
)

// ProjectInfo 项目基础配置信息
type ProjectInfo struct {
	language lang.Language // 项目语言
	Path     string
	Name     string
}
func NewProjectInfo(language lang.Language, path string) *ProjectInfo {
	return &ProjectInfo{
		language: language,
		Path:     path,
		Name:     path,
	}
}
