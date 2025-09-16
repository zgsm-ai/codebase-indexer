package wiki

import (
	"codebase-indexer/test/mocks"
	"os"
	"path/filepath"
	"testing"
)

// TestWikiManagerSimpleInterface 测试简化后的WikiManager接口
func TestWikiManagerSimpleInterface(t *testing.T) {
	// 创建测试目录
	testDir := t.TempDir()
	logger := &mocks.MockLogger{}
	// 创建简单的测试文件
	testContent := `package main

import "fmt"

func main() {
	fmt.Println("Hello, Simplified Wiki!")
}`

	if err := os.WriteFile(filepath.Join(testDir, "main.go"), []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// 创建Wiki管理器
	apiKey := "test-key"
	baseURL := "https://api.openai.com/v1"
	model := "gpt-3.5-turbo"
	manager, err := NewWikiManager(apiKey, baseURL, model, logger)
	if err != nil {
		t.Fatalf("Failed to create Wiki manager: %v", err)
	}

	// 测试DeleteWiki方法
	err = manager.DeleteWiki(testDir)
	if err != nil {
		t.Fatalf("Failed to delete Wiki: %v", err)
	}

	t.Log("Simplified WikiManager interface test passed")
}

// TestWikiManagerWithConfig 测试使用配置创建WikiManager
func TestWikiManagerWithConfig(t *testing.T) {
	// 创建测试目录
	testDir := t.TempDir()
	logger := &mocks.MockLogger{}
	// 创建配置
	config := DefaultSimpleConfig()
	config.APIKey = "test-key"
	config.BaseURL = "https://api.openai.com/v1"
	config.Model = "gpt-3.5-turbo"

	// 使用配置创建Wiki管理器
	manager, err := NewWikiManagerWithConfig(config, logger)
	if err != nil {
		t.Fatalf("Failed to create Wiki manager with config: %v", err)
	}

	// 测试DeleteWiki方法
	err = manager.DeleteWiki(testDir)
	if err != nil {
		t.Fatalf("Failed to delete Wiki: %v", err)
	}

	t.Log("WikiManager creation with config test passed")
}
