package utils

import (
	"github.com/google/uuid"
)

// GenerateUUID 生成一个新的 UUID v7
func GenerateUUID() (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

// GenerateUUIDBytes 生成一个新的 UUID v7 并返回字节切片
func GenerateUUIDBytes() ([]byte, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}
	return id[:], nil
}

// ParseUUID 解析 UUID 字符串
func ParseUUID(uuidStr string) (uuid.UUID, error) {
	return uuid.Parse(uuidStr)
}

// IsValidUUID 检查字符串是否是有效的 UUID
func IsValidUUID(uuidStr string) bool {
	_, err := uuid.Parse(uuidStr)
	return err == nil
}
