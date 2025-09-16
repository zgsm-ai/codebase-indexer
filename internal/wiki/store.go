package wiki

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Store 简单的文件存储接口
type Store interface {
	// SaveWiki 保存Wiki到文件
	SaveWiki(repoPath string, wikiStructure *WikiStructure) error

	// LoadWiki 从文件加载Wiki
	LoadWiki(repoPath string) (*WikiStructure, error)

	// DeleteWiki 删除Wiki文件
	DeleteWiki(repoPath string) error

	// WikiExists 检查Wiki文件是否存在
	WikiExists(repoPath string) bool
}

// SimpleFileStore 简单的文件存储实现
type SimpleFileStore struct {
	basePath string // 基础路径，用于存储Wiki文件
}

// NewSimpleFileStore 创建简单的文件存储管理器
func NewSimpleFileStore(basePath string) Store {
	if basePath == "" {
		// 默认使用当前目录下的 .wiki 文件夹
		basePath = ".wiki"
	}

	// 确保基础目录存在
	if err := os.MkdirAll(basePath, 0755); err != nil {
		// 如果创建失败，仍然返回存储实例，但操作可能会失败
		fmt.Printf("Warning: failed to create base directory %s: %v\n", basePath, err)
	}

	return &SimpleFileStore{
		basePath: basePath,
	}
}

// getWikiFilePath 生成Wiki文件路径
func (s *SimpleFileStore) getWikiFilePath(repoPath string) string {
	// 从 repoPath 提取仓库名称
	repoName := filepath.Base(repoPath)

	// 清理仓库名称，移除特殊字符
	repoName = strings.ToLower(strings.ReplaceAll(repoName, "/", "_"))
	repoName = strings.ReplaceAll(repoName, "-", "_")

	// 生成文件名
	filename := fmt.Sprintf("wiki_%s.json", repoName)
	return filepath.Join(s.basePath, filename)
}

// SaveWiki 保存Wiki到JSON文件
func (s *SimpleFileStore) SaveWiki(repoPath string, wikiStructure *WikiStructure) error {
	filePath := s.getWikiFilePath(repoPath)

	// 序列化 WikiStructure 为 JSON
	data, err := json.MarshalIndent(wikiStructure, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal wiki structure: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write wiki file %s: %w", filePath, err)
	}

	return nil
}

// LoadWiki 从JSON文件加载Wiki
func (s *SimpleFileStore) LoadWiki(repoPath string) (*WikiStructure, error) {
	filePath := s.getWikiFilePath(repoPath)

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, nil // 文件不存在，返回 nil
	}

	// 读取文件内容
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read wiki file %s: %w", filePath, err)
	}

	// 反序列化 JSON
	var wikiStructure WikiStructure
	if err := json.Unmarshal(data, &wikiStructure); err != nil {
		return nil, fmt.Errorf("failed to unmarshal wiki structure: %w", err)
	}

	return &wikiStructure, nil
}

// DeleteWiki 删除Wiki文件
func (s *SimpleFileStore) DeleteWiki(repoPath string) error {
	filePath := s.getWikiFilePath(repoPath)

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil // 文件不存在，无需删除
	}

	// 删除文件
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete wiki file %s: %w", filePath, err)
	}

	return nil
}

// WikiExists 检查Wiki文件是否存在
func (s *SimpleFileStore) WikiExists(repoPath string) bool {
	filePath := s.getWikiFilePath(repoPath)

	// 检查文件是否存在
	_, err := os.Stat(filePath)
	return err == nil
}
