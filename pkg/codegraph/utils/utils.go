package utils

import (
	"context"
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
