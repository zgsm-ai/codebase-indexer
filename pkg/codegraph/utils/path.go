package utils

import (
	"codebase-indexer/pkg/codegraph/types"
	"context"
	"errors"
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
	// 转换为 Unix 风格路径（统一使用 / 分隔符）
	// 1. 替换所有 Windows 风格的 \ 为 /
	slashed := strings.ReplaceAll(rawPath, types.WindowsSeparator, types.Slash)
	// 2. 清理路径（去掉多余 /、.、..）
	unixPath := path.Clean(slashed)
	return unixPath
}

// PathEqual 比较路径是否相等，/ \ 转为 /
func PathEqual(a, b string) bool {
	return filepath.ToSlash(a) == filepath.ToSlash(b)
}

// IsSubdir 判断sub绝对路径是否是parent绝对路径的子目录
// 注意：parent和sub都必须是绝对路径
func IsSubdir(parent, sub string) bool {
	// 确保路径已清理（处理.和..）
	parent = ToUnixPath(parent)
	sub = ToUnixPath(sub)
	if !strings.HasSuffix(parent, types.Slash) {
		parent = parent + types.Slash
	}
	if !strings.HasSuffix(sub, types.Slash) {
		sub = sub + types.Slash
	}
	return strings.HasPrefix(sub, parent) && sub != parent
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

// FindLongestExistingPath 从路径末尾向上查找最长的存在路径（文件或目录均可）
// 特殊处理：若最终仅根目录存在且输入路径不是根目录，则返回错误
func FindLongestExistingPath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", errors.New("invalid absolute path: " + err.Error())
	}
	current := filepath.Clean(absPath)
	originalPath := current // 保存原始路径用于根目录判断

	for {
		// 检查当前路径是否存在（文件或目录均可）
		if _, err := os.Stat(current); err == nil {
			// 若存在的是根目录，且原始路径不是根目录，则视为不存在
			if isRoot(current) && current != originalPath {
				return "", errors.New("path and its parents not exist")
			}
			return current, nil
		}

		parent := filepath.Dir(current)
		if parent == current { // 到达根目录仍未找到
			return "", errors.New("path and its parents not exist")
		}
		current = parent
	}
}

// 判断路径是否为根目录（跨系统）
func isRoot(path string) bool {
	clean := filepath.Clean(path)
	return filepath.Dir(clean) == clean
}
