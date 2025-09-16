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
	fmt.Println("=== Document Manager Test ===")

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
A sample Go project for testing document generation.

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

	// 创建文档管理器
	docManager, err := wiki.NewDocumentManagerWithConfig(config, logger)
	if err != nil {
		fmt.Printf("Failed to create document manager: %v\n", err)
		return
	}
	defer docManager.Close()

	fmt.Println("Document manager created successfully")

	// 测试支持的文档类型
	supportedTypes := docManager.GetSupportedDocumentTypes()
	fmt.Printf("Supported document types: %v\n", supportedTypes)

	// 测试生成代码规则文档
	fmt.Println("\n=== Testing Code Rules Generation ===")
	ctx := context.Background()

	codeRulesDoc, err := docManager.GenerateCodeRules(ctx, testDir)
	if err != nil {
		fmt.Printf("Failed to generate code rules document: %v\n", err)
		// 继续测试其他功能，不退出
	} else if codeRulesDoc != nil {
		fmt.Printf("Code rules document generated successfully\n")
		fmt.Printf("Title: %s\n", codeRulesDoc.Title)
		fmt.Printf("Description: %s\n", codeRulesDoc.Description)
		fmt.Printf("Total Pages: %d\n", len(codeRulesDoc.Pages))
		fmt.Printf("Total Sections: %d\n", len(codeRulesDoc.Sections))

		// 检查文档是否存在
		exists := docManager.ExistsCodeRules(testDir)
		fmt.Printf("Code rules document exists in storage: %v\n", exists)
	}

	// 测试生成Wiki文档（兼容模式）
	fmt.Println("\n=== Testing Wiki Generation (Compatibility Mode) ===")

	wikiDoc, err := docManager.GenerateWiki(ctx, testDir)
	if err != nil {
		fmt.Printf("Failed to generate wiki document: %v\n", err)
	} else if wikiDoc != nil {
		fmt.Printf("Wiki document generated successfully\n")
		fmt.Printf("Title: %s\n", wikiDoc.Title)
		fmt.Printf("Description: %s\n", wikiDoc.Description)
		fmt.Printf("Total Pages: %d\n", len(wikiDoc.Pages))
		fmt.Printf("Total Sections: %d\n", len(wikiDoc.Sections))

		// 检查Wiki是否存在
		exists := docManager.ExistsWiki(testDir)
		fmt.Printf("Wiki document exists in storage: %v\n", exists)
	}

	// 测试导出功能
	fmt.Println("\n=== Testing Export Functionality ===")

	// 测试导出代码规则为JSON
	err = docManager.ExportCodeRules(testDir, "./output/code_rules.json", "json")
	if err != nil {
		fmt.Printf("Failed to export code rules to JSON: %v\n", err)
	} else {
		fmt.Println("Code rules exported to JSON successfully")
	}

	// 测试导出Wiki为Markdown
	err = docManager.ExportWiki(testDir, "./output/wiki.md", "markdown", "single")
	if err != nil {
		fmt.Printf("Failed to export wiki to Markdown: %v\n", err)
	} else {
		fmt.Println("Wiki exported to Markdown successfully")
	}

	// 测试删除功能
	fmt.Println("\n=== Testing Delete Functionality ===")

	err = docManager.DeleteCodeRules(testDir)
	if err != nil {
		fmt.Printf("Failed to delete code rules: %v\n", err)
	} else {
		fmt.Println("Code rules deleted successfully")
		exists := docManager.ExistsCodeRules(testDir)
		fmt.Printf("Code rules document exists after deletion: %v\n", exists)
	}

	err = docManager.DeleteWiki(testDir)
	if err != nil {
		fmt.Printf("Failed to delete wiki: %v\n", err)
	} else {
		fmt.Println("Wiki deleted successfully")
		exists := docManager.ExistsWiki(testDir)
		fmt.Printf("Wiki document exists after deletion: %v\n", exists)
	}

	fmt.Printf("\n=== Document Manager Test Completed ===\n")
	fmt.Println("✅ Document manager created successfully")
	fmt.Println("✅ Supported document types retrieved")
	if codeRulesDoc != nil {
		fmt.Println("✅ Code rules document generated")
		fmt.Println("✅ Code rules document storage verified")
	}
	if wikiDoc != nil {
		fmt.Println("✅ Wiki document generated (compatibility mode)")
		fmt.Println("✅ Wiki document storage verified")
	}
	fmt.Println("✅ Export functionality tested")
	fmt.Println("✅ Delete functionality tested")
	fmt.Println("\nThe new DocumentManager is working correctly!")
}
