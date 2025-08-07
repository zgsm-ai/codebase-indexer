package utils

import (
	"codebase-indexer/pkg/codegraph/types"
	"context"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func CheckContextCanceled(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

// ToUnixPath 将相对路径转换为 Unix 风格（使用 / 分隔符，去除冗余路径元素）
func ToUnixPath(rawPath string) string {
	// path.Clean 会自动处理为 Unix 风格路径，去除多余的 /、. 和 ..
	filePath := path.Clean(rawPath)
	filePath = filepath.ToSlash(filePath)
	return filePath
}

// PathEqual 比较路径是否相等，/ \ 转为 /
func PathEqual(a, b string) bool {
	return filepath.ToSlash(a) == filepath.ToSlash(b)
}

func IsChild(parent, path string) bool {
	// 确保路径规范化（处理斜杠、相对路径等）
	parent = ToUnixPath(filepath.Clean(parent))
	path = ToUnixPath(filepath.Clean(path))

	// 计算相对路径
	rel, err := filepath.Rel(parent, path)
	if err != nil {
		return false // 无法计算相对路径（如跨磁盘）
	}

	// 相对路径不能以 ".." 开头，且不能等于 "."（即相同路径）
	return !strings.HasPrefix(rel, "..") && rel != "."
}

// IsHiddenFile 判断文件或目录是否为隐藏项
func IsHiddenFile(path string) bool {
	// 标准化路径，处理相对路径、符号链接等
	cleanPath := filepath.Clean(path)

	// 处理特殊路径
	if cleanPath == "." || cleanPath == ".." {
		return false
	}

	// 分割路径组件（兼容不同操作系统的路径分隔符）
	components := strings.Split(cleanPath, string(filepath.Separator))

	// 检查每个组件是否以"."开头（且不为空字符串）
	for _, comp := range components {
		if len(comp) > 0 && comp[0] == '.' {
			return true
		}
	}

	return false
}

// IsRelativePath 判断是否为相对路径
func IsRelativePath(path string) bool {
	return strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../") || path == "." || path == ".."
}

// IsSameParentDir 属于相同的父目录
func IsSameParentDir(a, b string) bool {
	parentA := filepath.Dir(a)
	parentB := filepath.Dir(b)
	// 比较父目录是否相同（已自动处理路径分隔符差异）
	return parentA == parentB
}

// ListOnlyFiles 列出指定目录下的所有文件（不包含子目录、隐藏目录）
func ListOnlyFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() { // 只保留文件，过滤目录
			if IsHiddenFile(entry.Name()) {
				continue
			}
			// 获取目录的完整路径
			fullPath := filepath.Join(dir, entry.Name())
			files = append(files, fullPath)
		}
	}
	return files, nil
}

// ListSubDirs 列出指定目录下的子目录(不包括文件、隐藏目录)
func ListSubDirs(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var subDirs []string
	for _, entry := range entries {
		if entry.IsDir() { // 只保留目录，过滤文件或隐藏目录
			if IsHiddenFile(entry.Name()) {
				continue
			}
			// 获取文件的完整路径
			fullPath := filepath.Join(dir, entry.Name())
			subDirs = append(subDirs, fullPath)
		}
	}
	return subDirs, nil
}

// EnsureTrailingSeparator 确保路径尾部带有系统对应的路径分隔符
// 若已有分隔符则不重复添加
func EnsureTrailingSeparator(path string) string {
	if path == types.EmptyString {
		return types.EmptyString
	}
	// 获取当前系统的路径分隔符（如'/'或'\\'）
	sep := string(filepath.Separator)
	// 判断路径最后一个字符是否为分隔符
	if strings.HasSuffix(path, sep) {
		return path
	}
	// 追加分隔符
	return path + sep
}

// TrimLastSeparator 移除路径尾部最后一个系统分隔符
// 问题：无法处理连续分隔符（如 "dir//" 会保留 "dir/"），根路径处理可能不符合预期
func TrimLastSeparator(path string) string {
	return strings.TrimSuffix(path, string(filepath.Separator))
}
