package wiki

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Exporter Wiki导出器
type Exporter struct{}

// NewExporter 创建新的导出器
func NewExporter() *Exporter {
	return &Exporter{}
}

// ExportMarkdown 导出为Markdown格式，支持两种模式：
// mode: "single" - 单个文件模式（默认）
// mode: "multi" - 多文件/目录模式，使用 index.md 进行路径引用
// customFilename: 自定义文件名（可选，仅对single模式有效）
func (e *Exporter) ExportMarkdown(wikiStructure *WikiStructure, outputPath string, mode string, customFilename string) error {
	if mode == "multi" {
		return e.exportMarkdownMulti(wikiStructure, outputPath)
	}
	// 默认使用单文件模式
	return e.exportMarkdownSingle(wikiStructure, outputPath, customFilename)
}

// exportMarkdownSingle 单文件模式导出
func (e *Exporter) exportMarkdownSingle(wikiStructure *WikiStructure, outputPath string, customFilename string) error {
	// 确保输出目录存在
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// 生成markdown内容
	content := e.generateMarkdownExport(wikiStructure)

	// 确定文件名
	var filename string
	if len(customFilename) > 0 && customFilename != "" {
		// 使用自定义文件名
		filename = customFilename
		// 确保有.md扩展名
		if !strings.HasSuffix(filename, ".md") {
			filename = filename + ".md"
		}
	} else {
		// 生成默认文件名，与 store 中的文件名保持一致
		repoName := "wiki"
		if wikiStructure.Title != "" {
			repoName = strings.ToLower(strings.ReplaceAll(wikiStructure.Title, " ", "_"))
			repoName = strings.ReplaceAll(repoName, "/", "_")
			repoName = strings.ReplaceAll(repoName, "-", "_")
			// 如果标题已经包含 "wiki"，就不再添加
			if !strings.Contains(repoName, "wiki") {
				repoName = repoName + "_wiki"
			}
		}
		filename = fmt.Sprintf("%s.md", repoName)
	}

	// 写入文件
	outputPath = filepath.Join(outputPath, filename)
	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write markdown file: %w", err)
	}

	return nil
}

// exportMarkdownMulti 多文件/目录模式导出
func (e *Exporter) exportMarkdownMulti(wikiStructure *WikiStructure, outputPath string) error {
	// 创建主目录
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// 生成 index.md 主索引文件
	indexContent := e.generateIndexMarkdown(wikiStructure)
	indexPath := filepath.Join(outputPath, "index.md")
	if err := os.WriteFile(indexPath, []byte(indexContent), 0644); err != nil {
		return fmt.Errorf("failed to write index file: %w", err)
	}

	// 为每个页面创建单独的 markdown 文件
	for i := range wikiStructure.Pages {
		page := &wikiStructure.Pages[i]
		pageContent := e.generatePageMarkdown(page, wikiStructure)

		// 使用页面ID作为文件名，确保唯一性
		pageFilename := fmt.Sprintf("%s.md", page.ID)
		pagePath := filepath.Join(outputPath, pageFilename)

		if err := os.WriteFile(pagePath, []byte(pageContent), 0644); err != nil {
			return fmt.Errorf("failed to write page file %s: %w", pageFilename, err)
		}
	}

	return nil
}

// generateMarkdownExport 生成Markdown导出 - 与api/api.py保持一致
func (e *Exporter) generateMarkdownExport(wikiStructure *WikiStructure) string {
	var builder strings.Builder

	// 开始与api/api.py保持一致的markdown导出
	builder.WriteString(fmt.Sprintf("# %s\n\n", wikiStructure.Title))
	builder.WriteString(fmt.Sprintf("Generated on: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	// 添加目录
	builder.WriteString("## Table of Contents\n\n")
	for _, page := range wikiStructure.Pages {
		builder.WriteString(fmt.Sprintf("- [%s](#%s)\n", page.Title, page.ID))
	}
	builder.WriteString("\n")

	// 添加每个页面
	for _, page := range wikiStructure.Pages {
		builder.WriteString(fmt.Sprintf("<a id='%s'></a>\n\n", page.ID))

		// 检查页面内容是否已经包含标题，避免重复
		pageContent := page.Content
		if strings.HasPrefix(strings.TrimSpace(pageContent), "#") {
			// 如果内容已经以标题开始，不再添加额外的标题
			builder.WriteString(fmt.Sprintf("%s\n\n", pageContent))
		} else {
			// 如果内容没有标题，添加一个
			builder.WriteString(fmt.Sprintf("## %s\n\n", page.Title))
			builder.WriteString(fmt.Sprintf("%s\n\n", pageContent))
		}

		// 添加相关页面
		if len(page.RelatedPages) > 0 {
			builder.WriteString("### Related Pages\n\n")
			var relatedTitles []string
			for _, relatedID := range page.RelatedPages {
				// 查找相关页面的标题
				for _, relatedPage := range wikiStructure.Pages {
					if relatedPage.ID == relatedID {
						relatedTitles = append(relatedTitles, fmt.Sprintf("[%s](#%s)", relatedPage.Title, relatedID))
						break
					}
				}
			}
			if len(relatedTitles) > 0 {
				builder.WriteString("Related topics: " + strings.Join(relatedTitles, ", ") + "\n\n")
			}
		}

		builder.WriteString("---\n\n")
	}

	return builder.String()
}

// generateIndexMarkdown 生成索引页面的 markdown 内容
func (e *Exporter) generateIndexMarkdown(wikiStructure *WikiStructure) string {
	var builder strings.Builder

	// 标题和基本信息 - 避免重复添加"Wiki 文档"后缀
	cleanTitle := wikiStructure.Title
	if strings.HasSuffix(cleanTitle, " Wiki") || strings.HasSuffix(cleanTitle, " 文档") {
		cleanTitle = strings.TrimSuffix(cleanTitle, " Wiki")
		cleanTitle = strings.TrimSuffix(cleanTitle, " 文档")
	}

	builder.WriteString(fmt.Sprintf("# %s\n\n", cleanTitle))
	builder.WriteString(fmt.Sprintf("生成时间: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))
	builder.WriteString(fmt.Sprintf("总页面数: %d\n\n", len(wikiStructure.Pages)))

	// 页面索引
	builder.WriteString("## 页面索引\n\n")
	for _, page := range wikiStructure.Pages {
		// 创建指向单独页面文件的链接
		pageFile := fmt.Sprintf("%s.md", page.ID)
		builder.WriteString(fmt.Sprintf("- [%s](%s)\n", page.Title, pageFile))
	}

	return builder.String()
}

// generatePageMarkdown 生成单个页面的 markdown 内容
func (e *Exporter) generatePageMarkdown(page *WikiPage, wikiStructure *WikiStructure) string {
	var builder strings.Builder

	// 检查页面内容是否已经包含标题，避免重复
	pageContent := strings.TrimSpace(page.Content)
	if strings.HasPrefix(pageContent, "#") {
		// 如果内容已经以标题开始，直接使用内容
		builder.WriteString(pageContent)
		builder.WriteString("\n")
	} else {
		// 如果内容没有标题，添加标题
		builder.WriteString(fmt.Sprintf("# %s\n\n", page.Title))
		builder.WriteString(pageContent)
		builder.WriteString("\n")
	}

	// 相关页面链接
	if len(page.RelatedPages) > 0 {
		builder.WriteString("## 相关页面\n\n")
		for _, relatedID := range page.RelatedPages {
			// 查找相关页面的标题
			for _, relatedPage := range wikiStructure.Pages {
				if relatedPage.ID == relatedID {
					relatedFile := fmt.Sprintf("%s.md", relatedID)
					builder.WriteString(fmt.Sprintf("- [%s](%s)\n", relatedPage.Title, relatedFile))
					break
				}
			}
		}
		builder.WriteString("\n")
	}

	// 返回主页面的链接
	builder.WriteString("[← 返回主索引](index.md)\n\n")

	return builder.String()
}

// generateAnchor 生成锚点
func (e *Exporter) generateAnchor(title string) string {
	anchor := strings.ToLower(title)
	anchor = strings.ReplaceAll(anchor, " ", "-")
	anchor = strings.ReplaceAll(anchor, "_", "-")

	var result strings.Builder
	for _, r := range anchor {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}

	return result.String()
}

// ExportJSON 导出为JSON格式 - 与api/api.py保持一致
func (e *Exporter) ExportJSON(wikiStructure *WikiStructure, outputPath string) error {
	// 确保输出目录存在
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// 构建导出数据 - 与api/api.py完全一致
	exportData := map[string]interface{}{
		"metadata": map[string]interface{}{
			"repository":   wikiStructure.Title,
			"generated_at": time.Now().Format(time.RFC3339),
			"page_count":   len(wikiStructure.Pages),
		},
		"pages": wikiStructure.Pages,
	}

	// 转换为JSON
	jsonData, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	// 生成文件名，与 store 中的文件名保持一致
	repoName := "wiki"
	if wikiStructure.Title != "" {
		repoName = strings.ToLower(strings.ReplaceAll(wikiStructure.Title, " ", "_"))
		repoName = strings.ReplaceAll(repoName, "/", "_")
		repoName = strings.ReplaceAll(repoName, "-", "_")
		// 如果标题已经包含 "wiki"，就不再添加
		if !strings.Contains(repoName, "wiki") {
			repoName = repoName + "_wiki"
		}
	}
	filename := fmt.Sprintf("%s.json", repoName)

	// 写入文件
	outputPath = filepath.Join(outputPath, filename)
	if err := os.WriteFile(outputPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write JSON file: %w", err)
	}

	return nil
}
