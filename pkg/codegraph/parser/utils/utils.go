package utils

import (
	"context"
	"path"
	"path/filepath"
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
