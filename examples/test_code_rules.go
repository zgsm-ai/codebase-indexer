//go:build ignore
// +build ignore

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"codebase-indexer/internal/wiki"
	loggerpkg "codebase-indexer/pkg/logger"
)

func main() {
	fmt.Println("=== Code Rules Generation Test ===")

	// 创建测试用的临时目录
	testDir := "./test_repo"
	err := os.MkdirAll(testDir, 0755)
	if err != nil {
		fmt.Printf("Failed to create test directory: %v\n", err)
		return
	}
	defer os.RemoveAll(testDir)

	// 创建一些测试文件
	testFiles := map[string]string{
		"README.md": `# Test Go Project
A sample Go project for testing code rules generation.

## Technology Stack
- Go 1.19
- Gin web framework
- PostgreSQL database
- Docker containerization`,

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
			fmt.Printf("Failed to create directory %s: %v\n", dir, err)
			return
		}

		err = os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			fmt.Printf("Failed to write test file %s: %v\n", path, err)
			return
		}
	}

	// 创建logger
	logger, err := loggerpkg.NewLogger("./logs", "debug", "test")
	if err != nil {
		fmt.Printf("Failed to create logger: %v\n", err)
		return
	}

	// 创建配置
	config := &wiki.SimpleConfig{
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
	factory := wiki.NewGeneratorFactory(logger)

	// 创建代码规则生成器
	generator, err := factory.CreateGenerator(wiki.DocTypeCodeRules, config)
	if err != nil {
		fmt.Printf("Failed to create code rules generator: %v\n", err)
		return
	}
	defer generator.Close()

	fmt.Println("Generating code rules document...")

	// 生成代码规则文档
	ctx := context.Background()
	docStructure, err := generator.GenerateDocument(ctx, testDir)
	if err != nil {
		fmt.Printf("Failed to generate code rules document: %v\n", err)
		return
	}

	// 验证结果
	if docStructure == nil {
		fmt.Println("Error: Document structure is nil")
		return
	}

	if docStructure.Title == "" {
		fmt.Println("Warning: Document title is empty")
	}

	if len(docStructure.Pages) == 0 {
		fmt.Println("Warning: No pages generated")
	}

	if len(docStructure.Sections) == 0 {
		fmt.Println("Warning: No sections generated")
	}

	// 打印结果摘要
	fmt.Printf("\n=== Code Rules Generation Results ===\n")
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
			fmt.Printf("First page content preview (first 500 chars):\n%s...\n",
				truncateString(samplePage.Content, 500))
		} else {
			fmt.Println("Warning: Sample page content is empty")
		}
	}

	fmt.Printf("\n=== Test Completed Successfully ===\n")
}

// truncateString 截断字符串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
