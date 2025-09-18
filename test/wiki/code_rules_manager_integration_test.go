package wiki

import (
	"codebase-indexer/internal/wiki"
	"codebase-indexer/pkg/logger"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var codeRulesLogger, codeRulesErr = logger.NewLogger("/tmp/logs", "debug", "codebase-indexer-code-rules-test")

// TestCodeRulesManagerGenerateCodeRulesIntegration 测试DocumentManager的GenerateCodeRules方法集成测试
func TestCodeRulesManagerGenerateCodeRulesIntegration(t *testing.T) {
	if codeRulesErr != nil {
		t.Fatalf("init log err:%v", codeRulesErr)
	}
	// 获取项目根目录路径
	repoPath, err := filepath.Abs("../../")
	if err != nil {
		t.Fatalf("Failed to get project root path: %v", err)
	}

	// 创建Document管理器
	apiKey := os.Getenv("API_KEY")
	baseURL := os.Getenv("BASE_URL")
	model := os.Getenv("MODEL")
	manager, err := wiki.NewDocumentManager(apiKey, baseURL, model, codeRulesLogger)
	if err != nil {
		t.Fatalf("Failed to create Document manager: %v", err)
	}

	// 创建上下文，设置超时时间
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
	defer cancel()

	// 测试GenerateCodeRules方法
	codeRulesStructure, err := manager.GenerateCodeRules(ctx, repoPath)
	if err != nil {
		t.Fatalf("Failed to generate CodeRules: %v", err)
	}
	assert.NotNil(t, codeRulesStructure)
}

// TestCodeRulesManagerExportCodeRulesIntegration 测试DocumentManager的ExportCodeRules方法集成测试
func TestCodeRulesManagerExportCodeRulesIntegration(t *testing.T) {
	// 获取项目根目录路径
	repoPath, err := filepath.Abs("../../")
	if err != nil {
		t.Fatalf("Failed to get project root path: %v", err)
	}

	// 创建Document管理器
	apiKey := "test" // 使用测试密钥
	baseURL := "https://openai.com"
	model := "gpt-4o"
	manager, err := wiki.NewDocumentManager(apiKey, baseURL, model, codeRulesLogger)
	if err != nil {
		t.Fatalf("Failed to create Document manager: %v", err)
	}

	outputBase := filepath.Join(repoPath, "test", "wiki", ".documents")
	// 创建临时输出目录
	outputDir := filepath.Join(outputBase, "output")
	multiOutputDir := filepath.Join(outputBase, "multi_output")

	// 测试单文件Markdown导出
	t.Log("Testing single file markdown export")

	err = manager.ExportCodeRules(repoPath, wiki.ExportOptions{OutputPath: outputDir, Format: "markdown", MarkdownMode: "single"})
	if err != nil {
		t.Fatalf("Failed to export single Markdown: %v", err)
	}

	// 测试多文件Markdown导出
	t.Log("Testing multi file markdown export")

	err = manager.ExportCodeRules(repoPath, wiki.ExportOptions{OutputPath: multiOutputDir, Format: "markdown", MarkdownMode: "multi"})
	if err != nil {
		t.Fatalf("Failed to export multi Markdown: %v", err)
	}

	// 测试JSON导出
	t.Log("Testing JSON export")

	err = manager.ExportCodeRules(repoPath, wiki.ExportOptions{OutputPath: outputDir, Format: "json", MarkdownMode: "single"})
	if err != nil {
		t.Fatalf("Failed to export JSON: %v", err)
	}

	// 测试不支持的格式
	t.Log("Testing unsupported format")

	err = manager.ExportCodeRules(repoPath, wiki.ExportOptions{OutputPath: outputDir, Format: "unsupported", MarkdownMode: ""})
	if err == nil {
		t.Error("Should return unsupported format error")
	}

	t.Log("CodeRules export test passed")
}
