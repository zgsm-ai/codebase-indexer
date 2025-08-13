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

// UniqueStringSlice 删除重复的字符串
func UniqueStringSlice(slice []string) []string {
	uniqueSlice := make([]string, 0, len(slice))
	uniqueMap := make(map[string]struct{})
	for _, str := range slice {
		if _, ok := uniqueMap[str]; !ok {
			uniqueMap[str] = struct{}{}
			uniqueSlice = append(uniqueSlice, str)
		}
	}
	return uniqueSlice
}

// StringSlice2Map 将字符串切片转换为map
func StringSlice2Map(slice []string) map[string]struct{} {
	uniqueMap := make(map[string]struct{})
	for _, str := range slice {
		uniqueMap[str] = struct{}{}
	}
	return uniqueMap
}
