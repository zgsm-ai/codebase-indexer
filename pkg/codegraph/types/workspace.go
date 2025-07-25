package types

import (
	"errors"
	"io"
	"io/fs"
	"time"
)

type FileInfo struct {
	Name    string    `json:"language"`          // 节点名称
	Path    string    `json:"path"`              // 节点路径
	Size    int64     `json:"size,omitempty"`    // 文件大小（仅文件有）
	ModTime time.Time `json:"modTime,omitempty"` // 修改时间（可选）
	IsDir   bool      `json:"IsDir"`             // 是否是目录
	Mode    fs.FileMode
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

type WalkFunc func(walkCtx *WalkContext, reader io.ReadCloser) error

var SkipDir = errors.New("skip this directory")

type WalkOptions struct {
	IgnoreError     bool
	ExcludeExts     []string
	IncludeExts     []string
	ExcludePrefixes []string
	IncludePrefixes []string
	ExcludeDirs     []string
	IncludeDirs     []string
}

type ReadOptions struct {
	StartLine int
	EndLine   int
}
