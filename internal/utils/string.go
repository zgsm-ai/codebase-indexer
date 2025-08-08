package utils

import (
	"crypto/md5"
	"fmt"
	"path/filepath"
)

// GenerateCodebaseID 生成代码库唯一ID
func GenerateCodebaseID(path string) string {
	name := filepath.Base(path)
	// 使用MD5哈希生成唯一ID，结合名称和路径
	return fmt.Sprintf("%s_%x", name, md5.Sum([]byte(path)))
}

// GenerateCodebaseEmbeddingID 生成代码库嵌入唯一ID
func GenerateCodebaseEmbeddingID(path string) string {
	name := filepath.Base(path)
	// 使用MD5哈希生成唯一ID，结合名称和路径
	return fmt.Sprintf("%s_%x_embedding", name, md5.Sum([]byte(path)))
}
