package wiki

import (
	"codebase-indexer/internal/wiki"
	"codebase-indexer/pkg/logger"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var newLogger, err = logger.NewLogger("/tmp/logs", "debug", "codebase-indexer-test")

// TestWikiManagerGenerateWikiIntegration 测试WikiManager的GenerateWiki方法集成测试
func TestWikiManagerGenerateWikiIntegration(t *testing.T) {
	if err != nil {
		t.Fatalf("init log err:%v", err)
	}
	// 获取项目根目录路径
	repoPath, err := filepath.Abs("../")
	if err != nil {
		t.Fatalf("Failed to get project root path: %v", err)
	}

	// 创建Wiki管理器
	apiKey := os.Getenv("API_KEY") // 使用测试密钥
	baseURL := os.Getenv("BASE_URL")
	model := os.Getenv("MODEL")
	manager, err := wiki.NewDocumentManager(apiKey, baseURL, model, newLogger)
	if err != nil {
		t.Fatalf("Failed to create Wiki manager: %v", err)
	}

	// 创建上下文，设置超时时间
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// 测试GenerateWiki方法
	wikiStructure, err := manager.GenerateWiki(ctx, repoPath)
	if err != nil {
		t.Fatalf("Failed to generate Wiki: %v", err)
	}

	// 验证Wiki结构
	if wikiStructure == nil {
		t.Fatal("Wiki structure is empty")
	}

	if wikiStructure.Title == "" {
		t.Error("Wiki title is empty")
	}

	if len(wikiStructure.Pages) == 0 {
		t.Error("Wiki pages are empty")
	}

	// 验证项目概览页面
	overviewFound := false
	for _, page := range wikiStructure.Pages {
		if page.ID == "overview" {
			overviewFound = true
			if page.Content == "" {
				t.Error("Project overview page content is empty")
			}
			break
		}
	}

	if !overviewFound {
		t.Error("Project overview page not found")
	}

	t.Logf("Wiki generated successfully, contains %d pages", len(wikiStructure.Pages))
	t.Logf("Wiki title: %s", wikiStructure.Title)
}

// TestWikiManagerExportWikiIntegration 测试WikiManager的ExportWiki方法集成测试
func TestWikiManagerExportWikiIntegration(t *testing.T) {
	// 获取项目根目录路径
	repoPath, err := filepath.Abs("../")
	if err != nil {
		t.Fatalf("Failed to get project root path: %v", err)
	}

	// 创建Wiki管理器
	apiKey := "test" // 使用测试密钥
	baseURL := "https://openai.com"
	model := "gpt-4o"
	manager, err := wiki.NewDocumentManager(apiKey, baseURL, model, newLogger)
	if err != nil {
		t.Fatalf("Failed to create Wiki manager: %v", err)
	}
	outputBase := filepath.Join(repoPath, "wiki", ".documents")
	// 创建临时输出目录
	outputDir := filepath.Join(outputBase, "output")
	multiOutputDir := filepath.Join(outputBase, "multi_output")

	// 测试单文件Markdown导出
	t.Log("Testing single file markdown export")
	err = manager.ExportWiki(repoPath, outputDir, "markdown", "single", "")
	if err != nil {
		t.Fatalf("Failed to export single Markdown: %v", err)
	}

	// 测试多文件Markdown导出
	t.Log("Testing multi file markdown export")
	err = manager.ExportWiki(repoPath, multiOutputDir, "markdown", "multi", "")
	if err != nil {
		t.Fatalf("Failed to export multi Markdown: %v", err)
	}

	// 测试JSON导出
	t.Log("Testing JSON export")
	err = manager.ExportWiki(repoPath, outputDir, "json", "single", "")
	if err != nil {
		t.Fatalf("Failed to export JSON: %v", err)
	}

	// 测试不支持的格式
	t.Log("Testing unsupported format")
	err = manager.ExportWiki(repoPath, outputDir, "json", "single", "")
	if err == nil {
		t.Error("Should return unsupported format error")
	}

	t.Log("Wiki export test passed")
}

// TestWikiManagerDeleteWikiIntegration 测试WikiManager的DeleteWiki方法集成测试
func TestWikiManagerDeleteWikiIntegration(t *testing.T) {
	// 获取项目根目录路径
	repoPath, err := filepath.Abs("../")
	if err != nil {
		t.Fatalf("Failed to get project root path: %v", err)
	}

	// 创建Wiki管理器
	apiKey := "test-key" // 使用测试密钥
	baseURL := "https://api.openai.com/v1"
	model := "gpt-3.5-turbo"
	manager, err := wiki.NewDocumentManager(apiKey, baseURL, model, newLogger)
	if err != nil {
		t.Fatalf("Failed to create Wiki manager: %v", err)
	}

	// 测试DeleteWiki方法
	err = manager.DeleteWiki(repoPath)
	if err != nil {
		t.Fatalf("Failed to delete Wiki: %v", err)
	}

	// 再次删除应该不会出错（幂等操作）
	err = manager.DeleteWiki(repoPath)
	if err != nil {
		t.Fatalf("Failed to delete Wiki again: %v", err)
	}

	t.Log("Wiki deletion test passed")
}

// TestWikiManagerFullWorkflowIntegration 测试WikiManager的完整工作流程集成测试
func TestWikiManagerFullWorkflowIntegration(t *testing.T) {
	// 获取项目根目录路径
	repoPath, err := filepath.Abs("./")
	if err != nil {
		t.Fatalf("Failed to get project root path: %v", err)
	}

	// 创建临时输出目录
	outputDir := t.TempDir()

	// 创建Wiki管理器
	apiKey := "test-key" // 使用测试密钥
	baseURL := "https://api.openai.com/v1"
	model := "gpt-3.5-turbo"
	manager, err := wiki.NewDocumentManager(apiKey, baseURL, model, newLogger)
	if err != nil {
		t.Fatalf("Failed to create Wiki manager: %v", err)
	}

	// 创建上下文，设置超时时间
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	// 1. 生成Wiki
	t.Log("Step 1: Generate Wiki")
	wikiStructure, err := manager.GenerateWiki(ctx, repoPath)
	if err != nil {
		t.Fatalf("Failed to generate Wiki: %v", err)
	}

	// 验证生成的Wiki结构
	if wikiStructure == nil || len(wikiStructure.Pages) == 0 {
		t.Fatal("Generated Wiki structure is invalid")
	}

	t.Logf("Successfully generated Wiki, contains %d pages", len(wikiStructure.Pages))

	// 2. 导出Wiki为Markdown格式
	t.Log("Step 2: Export Wiki as Markdown format")
	err = manager.ExportWiki(repoPath, outputDir, "markdown", "single", "")
	if err != nil {
		t.Fatalf("Failed to export Markdown: %v", err)
	}

	// 验证Markdown文件
	readmePath := filepath.Join(outputDir, "README.md")
	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		t.Error("README.md file not generated")
	}

	// 3. 导出Wiki为JSON格式
	t.Log("Step 3: Export Wiki as JSON format")
	err = manager.ExportWiki(repoPath, outputDir, "json", "single", "")
	if err != nil {
		t.Fatalf("Failed to export JSON: %v", err)
	}

	// 验证JSON文件
	jsonPath := filepath.Join(outputDir, "wiki.json")
	if _, err := os.Stat(jsonPath); os.IsNotExist(err) {
		t.Error("wiki.json file not generated")
	}

	// 4. 删除Wiki缓存
	t.Log("Step 4: Delete Wiki cache")
	err = manager.DeleteWiki(repoPath)
	if err != nil {
		t.Fatalf("Failed to delete Wiki: %v", err)
	}

	t.Log("Full workflow test passed")
}

// TestWikiManagerWithConfigIntegration 测试使用配置创建WikiManager的集成测试
func TestWikiManagerWithConfigIntegration(t *testing.T) {
	// 获取项目根目录路径
	repoPath, err := filepath.Abs("./")
	if err != nil {
		t.Fatalf("Failed to get project root path: %v", err)
	}

	// 创建配置
	config := wiki.DefaultSimpleConfig()
	config.APIKey = "test-key"
	config.BaseURL = "https://api.openai.com/v1"
	config.Model = "gpt-3.5-turbo"
	config.Language = "zh-CN"
	config.MaxTokens = 4096
	config.Temperature = 0.7

	// 使用配置创建Wiki管理器
	manager, err := wiki.NewDocumentManagerWithConfig(config, newLogger)
	if err != nil {
		t.Fatalf("Failed to create Wiki manager with config: %v", err)
	}

	// 创建上下文，设置超时时间
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// 测试GenerateWiki方法
	wikiStructure, err := manager.GenerateWiki(ctx, repoPath)
	if err != nil {
		t.Fatalf("Failed to generate Wiki: %v", err)
	}

	// 验证Wiki结构
	if wikiStructure == nil {
		t.Fatal("Wiki structure is empty")
	}

	if len(wikiStructure.Pages) == 0 {
		t.Error("Wiki pages are empty")
	}

	// 测试DeleteWiki方法
	err = manager.DeleteWiki(repoPath)
	if err != nil {
		t.Fatalf("Failed to delete Wiki: %v", err)
	}

	t.Log("WikiManager integration test with config passed")
}

// TestWikiManagerErrorHandlingIntegration 测试WikiManager错误处理的集成测试
func TestWikiManagerErrorHandlingIntegration(t *testing.T) {
	// 获取不存在的仓库路径
	nonExistentPath := "/non/existent/path"

	// 创建Wiki管理器
	apiKey := "test-key" // 使用测试密钥
	baseURL := "https://api.openai.com/v1"
	model := "gpt-3.5-turbo"
	manager, err := wiki.NewDocumentManager(apiKey, baseURL, model, newLogger)
	if err != nil {
		t.Fatalf("Failed to create Wiki manager: %v", err)
	}

	// 创建上下文，设置超时时间
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 测试对不存在路径生成Wiki
	_, err = manager.GenerateWiki(ctx, nonExistentPath)
	if err == nil {
		t.Error("Should return error because path does not exist")
	}

	// 测试对不存在路径删除Wiki（应该不会出错）
	err = manager.DeleteWiki(nonExistentPath)
	if err != nil {
		t.Fatalf("Failed to delete non-existent Wiki: %v", err)
	}

	t.Log("Error handling integration test passed")
}
