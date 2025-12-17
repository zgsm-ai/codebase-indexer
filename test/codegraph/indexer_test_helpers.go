//go:build integration
// +build integration

package codegraph

import (
	"codebase-indexer/pkg/codegraph/lang"
	"codebase-indexer/pkg/codegraph/store"
	"codebase-indexer/pkg/codegraph/types"
	"codebase-indexer/pkg/codegraph/utils"
	"codebase-indexer/pkg/codegraph/workspace"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// getSupportedExtByLanguageTestHelper returns supported extensions for a specific language
func getSupportedExtByLanguageTestHelper(language lang.Language) []string {
	parser, err := lang.GetSitterParserByLanguage(language)
	if err != nil {
		panic(err)
	}
	return parser.SupportedExts
}

// getAllSupportedExtTestHelper returns all supported extensions from all parsers
func getAllSupportedExtTestHelper() []string {
	parsers := lang.GetTreeSitterParsers()
	ext := make([]string, 0, len(parsers))
	for _, parser := range parsers {
		ext = append(ext, parser.SupportedExts...)
	}
	return ext
}

// countGoFiles 统计工作区中的 Go 文件数量
func countGoFiles(ctx context.Context, workspaceReader workspace.WorkspaceReader, workspaceDir string, visitPattern *types.VisitPattern) (int, error) {
	var goCount int
	err := workspaceReader.WalkFile(ctx, workspaceDir, func(walkCtx *types.WalkContext) error {
		if walkCtx.Info.IsDir {
			return nil
		}
		if strings.HasSuffix(walkCtx.Path, ".go") {
			goCount++
		}
		return nil
	}, types.WalkOptions{IgnoreError: true, VisitPattern: visitPattern})
	return goCount, err
}

// countIndexedFiles 统计已索引的文件数量
func countIndexedFiles(ctx context.Context, storage store.GraphStorage, projects []*workspace.Project) (int, error) {
	var indexSize int
	for _, p := range projects {
		iter := storage.Iter(ctx, p.Uuid)
		for iter.Next() {
			key := iter.Key()
			if store.IsElementPathKey(key) {
				indexSize++
			}
		}
		err := iter.Close()
		if err != nil {
			return 0, err
		}
	}
	return indexSize, nil
}

// validateStorageState 验证存储状态，确保索引数量与文件数量一致
func validateStorageState(t *testing.T, ctx context.Context, workspaceReader workspace.WorkspaceReader,
	storage store.GraphStorage, workspaceDir string,
	projects []*workspace.Project, visitPattern *types.VisitPattern) {
	// 统计 Go 文件数量
	goCount, err := countGoFiles(ctx, workspaceReader, workspaceDir, visitPattern)
	assert.NoError(t, err)

	// 统计索引数量
	indexSize, err := countIndexedFiles(ctx, storage, projects)
	assert.NoError(t, err)

	// 验证 80% 解析成功
	assert.True(t, float64(indexSize) > float64(goCount)*0.8)

	// 记录存储大小
	for _, p := range projects {
		t.Logf("=> storage size: %d", storage.Size(ctx, p.Uuid, store.PathKeySystemPrefix))
	}
}

// cleanIndexStoreTest 清理索引存储
func cleanIndexStoreTest(ctx context.Context, projects []*workspace.Project, storage store.GraphStorage) error {
	for _, p := range projects {
		if err := storage.DeleteAll(ctx, p.Uuid); err != nil {
			return err
		}
		if storage.Size(ctx, p.Uuid, types.EmptyString) > 0 {
			return fmt.Errorf("clean workspace index failed, size not equal 0")
		}
	}
	return nil
}

// getTestFiles 获取测试用的文件列表
func getTestFiles(t *testing.T, workspaceDir string) []string {
	filePath := filepath.Join(workspaceDir, "test", "mocks")
	files, err := utils.ListOnlyFiles(filePath)
	assert.NoError(t, err)
	return files
}

// createPathKeyMap 创建文件路径键的映射
func createPathKeyMap(t *testing.T, files []string) map[string]any {
	pathKeys := make(map[string]any)
	for _, f := range files {
		key, err := store.ElementPathKey{Language: lang.Go, Path: f}.Get()
		assert.NoError(t, err)
		pathKeys[key] = nil
	}
	return pathKeys
}

// validateFilesNotIndexed 验证指定文件没有被索引
func validateFilesNotIndexed(t *testing.T, ctx context.Context, storage store.GraphStorage, projects []*workspace.Project, pathKeys map[string]any) {
	for _, p := range projects {
		iter := storage.Iter(ctx, p.Uuid)
		for iter.Next() {
			key := iter.Key()
			if !store.IsElementPathKey(key) {
				continue
			}
			_, ok := pathKeys[key]
			assert.False(t, ok, "File should not be indexed: %s", key)
		}
		err := iter.Close()
		assert.NoError(t, err)
	}
}

// validateFilesIndexed 验证指定文件已经被索引
func validateFilesIndexed(t *testing.T, ctx context.Context, storage store.GraphStorage,
	projects []*workspace.Project, pathKeys map[string]any) {
	for _, p := range projects {
		iter := storage.Iter(ctx, p.Uuid)
		for iter.Next() {
			key := iter.Key()
			if !store.IsElementPathKey(key) {
				continue
			}
			delete(pathKeys, key)
		}
		err := iter.Close()
		assert.NoError(t, err)
	}
	assert.True(t, len(pathKeys) == 0, "All files should be indexed")
}

// validateStorageEmpty 验证存储为空
func validateStorageEmpty(t *testing.T, ctx context.Context, storage store.GraphStorage, projects []*workspace.Project) {
	for _, p := range projects {
		size := storage.Size(ctx, p.Uuid, types.EmptyString)
		assert.Equal(t, 0, size, "Storage should be empty for project: %s", p.Uuid)
	}
}

// printCallGraphToFile 将调用链以层次结构打印到文件
func printCallGraphToFile(t *testing.T, nodes []*types.RelationNode, filename string) {
	output, err := os.Create(filename)
	assert.NoError(t, err)
	defer output.Close()

	fmt.Fprintf(output, "调用链分析结果 (生成时间: %s)\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(output, "===========================================\n\n")

	for i, node := range nodes {
		fmt.Fprintf(output, "根节点 %d:\n", i+1)
		printNodeRecursive(output, node, 0)
		fmt.Fprintf(output, "\n")
	}
}

// printNodeRecursive 递归打印节点及其子节点
func printNodeRecursive(output *os.File, node *types.RelationNode, depth int) {
	indent := strings.Repeat("  ", depth)
	fmt.Fprintf(output, "%s├─ %s [%s]\n", indent, node.SymbolName, node.NodeType)
	fmt.Fprintf(output, "%s   文件: %s\n", indent, node.FilePath)
	if node.Position.StartLine > 0 || node.Position.EndLine > 0 {
		fmt.Fprintf(output, "%s   位置: 行%d-%d, 列%d-%d\n", indent,
			node.Position.StartLine, node.Position.EndLine,
			node.Position.StartColumn, node.Position.EndColumn)
	}
	if node.Content != "" {
		// 只显示内容的前50个字符
		content := strings.ReplaceAll(node.Content, "\n", "\\n")
		if len(content) > 50 {
			content = content[:50] + "..."
		}
		fmt.Fprintf(output, "%s   内容: %s\n", indent, content)
	}
	fmt.Fprintf(output, "%s   子调用数: %d\n", indent, len(node.Children))

	for _, child := range node.Children {
		printNodeRecursive(output, child, depth+1)
	}
}
