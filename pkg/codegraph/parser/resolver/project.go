package resolver

import (
	"codebase-indexer/pkg/codegraph/parser/utils"
	"path/filepath"
	"strings"
)

// ProjectInfo 项目基础配置信息
type ProjectInfo struct {
	language   Language            // 项目语言
	SourceRoot string              // 源码根路径（如 java 的 src/main/java）
	Dirs       []string            // 源文件目录（相对于 SourceRoot）
	dirToFiles map[string][]string // 目录到文件列表的索引（完整路径）
	fileSet    map[string]struct{} // 文件路径集合（完整路径）
}

func NewProjectInfo(language Language, sourceRoot string, files []string) *ProjectInfo {
	pc := &ProjectInfo{
		language:   language,
		SourceRoot: sourceRoot,
	}
	pc.buildIndex(files)
	return pc
}

// 构建目录和文件索引
func (p *ProjectInfo) buildIndex(files []string) {
	p.dirToFiles = make(map[string][]string)
	p.fileSet = make(map[string]struct{})
	dirSet := make(map[string]struct{})
	if files == nil {
		return
	}
	for _, f := range files {
		dir := utils.ToUnixPath(filepath.Dir(f))
		p.dirToFiles[dir] = append(p.dirToFiles[dir], f)
		p.fileSet[f] = struct{}{}
		dirSet[dir] = struct{}{}
	}

	// 提取相对于 SourceRoot 的目录
	p.Dirs = make([]string, 0, len(dirSet))
	for dir := range dirSet {
		// 计算相对于 SourceRoot 的路径
		p.Dirs = append(p.Dirs, dir)
	}
}

// 辅助函数：查找匹配的文件路径
func (p *ProjectInfo) findMatchingFiles(targetPath string) []string {
	var result []string
	if p.containsFileIndex(targetPath) {
		result = append(result, targetPath)
	}
	return result
}

// 辅助函数：查找目录下所有指定扩展名的文件
func (p *ProjectInfo) findFilesInDirIndex(dir string, ext string) []string {
	var result []string
	files, ok := p.dirToFiles[dir]
	if !ok {
		return result
	}
	for _, f := range files {
		if strings.HasSuffix(f, ext) {
			result = append(result, f)
		}
	}
	return result
}

// 辅助函数：检查文件是否存在于项目文件集合中
func (p *ProjectInfo) containsFileIndex(path string) bool {
	_, ok := p.fileSet[path]
	return ok
}
