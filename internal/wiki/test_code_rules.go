package wiki

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	loggerpkg "codebase-indexer/pkg/logger"
)

// TestCodeRulesGeneration 测试代码规则生成功能
func TestCodeRulesGeneration(t *testing.T) {
	// 创建测试用的临时目录
	testDir := "./test_repo"
	err := os.MkdirAll(testDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}
	defer os.RemoveAll(testDir)

	// 创建一些测试文件
	testFiles := map[string]string{
		"README.md": `# Test Project
A sample Go project for testing code rules generation.

## Tech Stack
- Go 1.19
- Gin framework
- PostgreSQL
- Docker`,

		"go.mod": `module test-project

go 1.19

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/lib/pq v1.10.9
)`,

		"main.go": `package main

import (
	"log"
	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()
	router.GET("/health", healthCheck)
	log.Fatal(router.Run(":8080"))
}

func healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"status": "healthy",
	})
}`,

		"internal/service/user_service.go": `package service

import "test-project/internal/model"

type UserService struct {
	repo UserRepository
}

func NewUserService(repo UserRepository) *UserService {
	return &UserService{repo: repo}
}

func (s *UserService) GetUser(id string) (*model.User, error) {
	return s.repo.FindByID(id)
}`,

		"internal/model/user.go": `package model

type User struct {
	ID    string ` + "`json:\"id\"`" + `
	Name  string ` + "`json:\"name\"`" + `
	Email string ` + "`json:\"email\"`" + `
}`,
	}

	// 写入测试文件
	for path, content := range testFiles {
		fullPath := filepath.Join(testDir, path)
		dir := filepath.Dir(fullPath)
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}

		err = os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to write test file %s: %v", path, err)
		}
	}

	// 创建logger
	logger, err := loggerpkg.NewLogger("./logs", "debug", "test")
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// 创建配置
	config := &SimpleConfig{
		APIKey:         "", // 可以留空用于测试
		BaseURL:        "",
		Model:          "",
		Language:       "zh",
		MaxTokens:      4096,
		Temperature:    0.7,
		MaxFiles:       100,
		MaxFileSize:    1024 * 1024, // 1MB
		OutputDir:      "./output",
		Concurrency:    2,
		PromptTemplate: "code_rules",
	}

	// 创建生成器工厂
	factory := NewGeneratorFactory(logger)

	// 创建代码规则生成器
	generator, err := factory.CreateGenerator(DocTypeCodeRules, config)
	if err != nil {
		t.Fatalf("Failed to create code rules generator: %v", err)
	}
	defer generator.Close()

	// 生成代码规则文档
	ctx := context.Background()
	docStructure, err := generator.GenerateDocument(ctx, testDir)
	if err != nil {
		t.Fatalf("Failed to generate code rules document: %v", err)
	}

	// 验证结果
	if docStructure == nil {
		t.Fatal("Document structure is nil")
	}

	if docStructure.Title == "" {
		t.Error("Document title is empty")
	}

	if len(docStructure.Pages) == 0 {
		t.Error("No pages generated")
	}

	if len(docStructure.Sections) == 0 {
		t.Error("No sections generated")
	}

	// 打印结果摘要
	fmt.Printf("=== Code Rules Generation Test Results ===\n")
	fmt.Printf("Title: %s\n", docStructure.Title)
	fmt.Printf("Description: %s\n", docStructure.Description)
	fmt.Printf("Total Pages: %d\n", len(docStructure.Pages))
	fmt.Printf("Total Sections: %d\n", len(docStructure.Sections))
	fmt.Printf("Root Sections: %v\n", docStructure.RootSections)

	fmt.Printf("\n=== Generated Guidelines ===\n")
	for i, page := range docStructure.Pages {
		fmt.Printf("%d. %s (Importance: %s)\n", i+1, page.Title, page.Importance)
		if len(page.FilePaths) > 0 {
			fmt.Printf("   Related Files: %v\n", page.FilePaths)
		}
	}

	fmt.Printf("\n=== Document Structure ===\n")
	for _, section := range docStructure.Sections {
		fmt.Printf("Section: %s (%s)\n", section.Title, section.ID)
		fmt.Printf("  Pages: %v\n", section.Pages)
		if len(section.Subsections) > 0 {
			fmt.Printf("  Subsections: %v\n", section.Subsections)
		}
	}

	// 验证生成的内容
	fmt.Printf("\n=== Sample Page Content ===\n")
	if len(docStructure.Pages) > 0 {
		samplePage := docStructure.Pages[0]
		if samplePage.Content != "" {
			fmt.Printf("First page content preview (first 200 chars):\n%s...\n",
				truncateString(samplePage.Content, 200))
		} else {
			t.Error("Sample page content is empty")
		}
	}

	fmt.Printf("\n=== Test Completed Successfully ===\n")
}

// TestGeneratorFactory 测试生成器工厂
func TestGeneratorFactory(t *testing.T) {
	logger, err := loggerpkg.NewLogger("./logs", "debug", "test")
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	factory := NewGeneratorFactory(logger)

	// 测试支持的类型
	supportedTypes := factory.GetSupportedTypes()
	if len(supportedTypes) != 2 {
		t.Errorf("Expected 2 supported types, got %d", len(supportedTypes))
	}

	// 验证支持的类型
	expectedTypes := []DocumentType{DocTypeWiki, DocTypeCodeRules}
	for _, expectedType := range expectedTypes {
		found := false
		for _, supportedType := range supportedTypes {
			if supportedType == expectedType {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected type %s not found in supported types", expectedType)
		}
	}

	// 测试创建生成器
	config := DefaultSimpleConfig()

	// 测试Wiki生成器
	wikiGen, err := factory.CreateGenerator(DocTypeWiki, config)
	if err != nil {
		t.Errorf("Failed to create Wiki generator: %v", err)
	}
	if wikiGen.GetDocumentType() != DocTypeWiki {
		t.Errorf("Expected Wiki generator, got %s", wikiGen.GetDocumentType())
	}

	// 测试代码规则生成器
	codeRulesGen, err := factory.CreateGenerator(DocTypeCodeRules, config)
	if err != nil {
		t.Errorf("Failed to create CodeRules generator: %v", err)
	}
	if codeRulesGen.GetDocumentType() != DocTypeCodeRules {
		t.Errorf("Expected CodeRules generator, got %s", codeRulesGen.GetDocumentType())
	}

	// 测试不支持的类型
	_, err = factory.CreateGenerator("unsupported", config)
	if err == nil {
		t.Error("Expected error for unsupported document type")
	}

	fmt.Println("Generator factory test completed successfully")
}

// truncateString 截断字符串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
