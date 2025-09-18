package wiki

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DocumentStore 增强的文档存储接口，支持多种文档类型
type DocumentStore interface {
	// SaveDocument 保存文档到文件
	SaveDocument(repoPath string, docStructure *DocumentStructure, docType string) error

	// LoadDocument 从文件加载文档
	LoadDocument(repoPath string, docType string) (*DocumentStructure, error)

	// DeleteDocument 删除文档文件
	DeleteDocument(repoPath string, docType string) error

	// DocumentExists 检查文档文件是否存在
	DocumentExists(repoPath string, docType string) bool
}

// localFileStore 增强的文件存储实现，支持多种文档类型
type localFileStore struct {
	basePath string // 基础路径，用于存储文档文件
}

// NewFileStore 创建增强的文件存储管理器
func NewFileStore(basePath string) DocumentStore {
	if basePath == "" {
		// 默认使用当前目录下的 .documents 文件夹
		basePath = ".documents"
	}

	// 确保基础目录存在
	if err := os.MkdirAll(basePath, 0755); err != nil {
		// 如果创建失败，仍然返回存储实例，但操作可能会失败
		fmt.Printf("Warning: failed to create base directory %s: %v\n", basePath, err)
	}

	return &localFileStore{
		basePath: basePath,
	}
}

// getDocumentFilePath 生成文档文件路径
func (s *localFileStore) getDocumentFilePath(repoPath string, docType string) string {
	// 从 repoPath 提取仓库名称
	repoName := filepath.Base(repoPath)

	// 清理仓库名称，移除特殊字符
	repoName = strings.ToLower(strings.ReplaceAll(repoName, "/", "_"))
	repoName = strings.ReplaceAll(repoName, "-", "_")

	// 生成文件名
	filename := fmt.Sprintf("%s_%s.json", docType, repoName)
	return filepath.Join(s.basePath, filename)
}

// SaveDocument 保存文档到JSON文件
func (s *localFileStore) SaveDocument(repoPath string, docStructure *DocumentStructure, docType string) error {
	filePath := s.getDocumentFilePath(repoPath, docType)

	// 序列化 DocumentStructure 为 JSON
	data, err := json.MarshalIndent(docStructure, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal document structure: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write document file %s: %w", filePath, err)
	}

	return nil
}

// LoadDocument 从JSON文件加载文档
func (s *localFileStore) LoadDocument(repoPath string, docType string) (*DocumentStructure, error) {
	filePath := s.getDocumentFilePath(repoPath, docType)

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, nil // 文件不存在，返回 nil
	}

	// 读取文件内容
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read document file %s: %w", filePath, err)
	}

	// 反序列化 JSON
	var docStructure DocumentStructure
	if err := json.Unmarshal(data, &docStructure); err != nil {
		return nil, fmt.Errorf("failed to unmarshal document structure: %w", err)
	}

	return &docStructure, nil
}

// DeleteDocument 删除文档文件
func (s *localFileStore) DeleteDocument(repoPath string, docType string) error {
	filePath := s.getDocumentFilePath(repoPath, docType)

	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil // 文件不存在，无需删除
	}

	// 删除文件
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete document file %s: %w", filePath, err)
	}

	return nil
}

// DocumentExists 检查文档文件是否存在
func (s *localFileStore) DocumentExists(repoPath string, docType string) bool {
	filePath := s.getDocumentFilePath(repoPath, docType)

	// 检查文件是否存在
	_, err := os.Stat(filePath)
	return err == nil
}
