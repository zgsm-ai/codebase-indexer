package types

import (
	"errors"
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
	Name    string    `json:"name"`  // 节点名称
	Path    string    `json:"path"`  // 节点路径
	Size    int64     `json:"-"`     // 文件大小（仅文件有）
	ModTime time.Time `json:"-"`     // 修改时间（可选）
	IsDir   bool      `json:"IsDir"` // 是否是目录
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

	// 1. 排除指定扩展名的文件
	if fileExt != EmptyString && slices.Contains(v.ExcludeExts, fileExt) {
		return true, nil
	}

	// 2. 仅包含指定扩展名的文件（非目录）
	if len(v.IncludeExts) > 0 && !isDir {
		if fileExt == EmptyString { // 无扩展名的文件直接排除
			return true, nil
		}
		if !slices.Contains(v.IncludeExts, fileExt) { // 扩展名不在包含列表中
			return true, nil
		}
	}

	// 3. 自定义跳过函数（优先级：用户自定义逻辑高于内置规则）
	if v.SkipFunc != nil {
		skip, err := v.SkipFunc(fileInfo)
		if skip {
			return true, err
		}
	}

	// 4. 目录特定规则（基于完整路径判断，解决父/子目录区分问题）
	if isDir {
		// 4.1 排除指定路径的目录（支持完整路径或特定层级目录）
		for _, excludePath := range v.ExcludeDirs {
			// 匹配规则：目录路径完全一致，或为目标目录的子目录（可选）
			if base == excludePath {
				return true, nil
			}
		}

		// 4.2 仅包含指定路径的目录（支持完整路径或特定层级目录）
		if len(v.IncludeDirs) > 0 {
			found := false
			for _, includePath := range v.IncludeDirs {
				// 匹配规则：目录路径完全一致，或为目标目录的子目录（可选）
				if base == includePath {
					found = true
					break
				}
			}
			if !found { // 不在包含列表中则跳过
				return true, nil
			}
		}
	}

	// 5. 排除指定前缀的文件/目录（基于名称）
	for _, p := range v.ExcludePrefixes {
		if strings.HasPrefix(base, p) {
			return true, nil
		}
	}

	// 6. 仅包含指定前缀的文件/目录（基于名称）
	if len(v.IncludePrefixes) > 0 {
		found := false
		for _, p := range v.IncludePrefixes {
			if strings.HasPrefix(base, p) {
				found = true
				break
			}
		}
		if !found {
			return true, nil
		}
	}

	return false, nil
}

type ReadOptions struct {
	StartLine int
	EndLine   int
}
