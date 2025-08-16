package types

import (
	"errors"
	"io/fs"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
)

// TreeOption 定义Tree方法的可选参数
type TreeOption func(*TreeOptions)

// TreeOptions 包含Tree方法的可选参数
type TreeOptions struct {
	MaxDepth       int            // 最大递归深度
	ExcludePattern *regexp.Regexp // 排除文件的正则表达式
	IncludePattern *regexp.Regexp // 包含文件的正则表达式
}

// TreeNode 表示目录树中的一个节点，可以是目录或文件
type TreeNode struct {
	FileInfo
	Children []*TreeNode `json:"children,omitempty"` // 子节点（仅目录有）
}

type FileInfo struct {
	Name    string      `json:"name"`  // 节点名称
	Path    string      `json:"path"`  // 节点路径
	Size    int64       `json:"-"`     // 文件大小（仅文件有）
	ModTime time.Time   `json:"-"`     // 修改时间（可选）
	IsDir   bool        `json:"IsDir"` // 是否是目录
	Mode    fs.FileMode `json:"-"`
}

// WalkContext provides context information during directory traversal
type WalkContext struct {
	// Current file or directory being processed
	Path string
	// Relative path from the root directory
	RelativePath string
	// File information
	Info *FileInfo
	// Parent directory path
	ParentPath string
}

// WalkFunc is the type of the function called for each file or directory

type WalkFunc func(walkCtx *WalkContext) error

var SkipDir = errors.New("skip this directory")

type WalkOptions struct {
	IgnoreError  bool
	VisitPattern *VisitPattern
}

type SkipFunc func(fileInfo *FileInfo) (bool, error)

type VisitPattern struct {
	MaxVisitLimit   int
	ExcludeExts     []string
	IncludeExts     []string
	ExcludePrefixes []string
	IncludePrefixes []string
	ExcludeDirs     []string
	IncludeDirs     []string
	SkipFunc        SkipFunc
}

func (v *VisitPattern) ShouldSkip(fileInfo *FileInfo) (bool, error) {
	if fileInfo == nil {
		return false, nil
	}
	isDir := fileInfo.IsDir
	path := fileInfo.Path
	base := filepath.Base(path)
	fileExt := filepath.Ext(base)
	if fileExt != EmptyString && slices.Contains(v.ExcludeExts, fileExt) {
		return true, nil
	}

	if len(v.IncludeExts) > 0 && fileExt != EmptyString && !slices.Contains(v.IncludeExts, fileExt) {
		return true, nil
	}

	if v.SkipFunc != nil {
		skip, err := v.SkipFunc(fileInfo)
		if skip {
			return true, err
		}
	}

	if isDir {
		for _, p := range v.ExcludeDirs {
			if base == p {
				return true, nil
			}
		}

		for _, p := range v.IncludeDirs {
			if base != p {
				return true, nil
			}
		}
	}

	for _, p := range v.ExcludePrefixes {
		if strings.HasPrefix(base, p) {
			return true, nil
		}
	}

	for _, p := range v.IncludePrefixes {
		if !strings.HasPrefix(base, p) {
			return true, nil
		}
	}
	return false, nil
}

type ReadOptions struct {
	StartLine int
	EndLine   int
}
