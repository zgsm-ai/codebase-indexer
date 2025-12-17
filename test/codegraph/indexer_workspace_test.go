//go:build integration
// +build integration

package codegraph

import (
	"codebase-indexer/pkg/codegraph/types"
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

var testVisitPattern = &types.VisitPattern{ExcludeDirs: []string{".git", ".idea"}, IncludeExts: []string{".go"}}

// TestIndexer_IndexWorkspace 测试索引器的 IndexWorkspace 方法
// 该测试验证索引器能够正确地索引整个工作区，并确保索引的文件数量与实际的文件数量一致
func TestIndexer_IndexWorkspace(t *testing.T) {
	// 设置测试环境
	env, err := setupTestEnvironment()
	assert.NoError(t, err)
	defer teardownTestEnvironment(t, env)

	// 获取测试工作区目录 - 使用项目根目录
	workspaceDir, err := filepath.Abs("../../")
	assert.NoError(t, err)

	if err := initWorkspaceModel(env, workspaceDir); err != nil {
		panic(err)
	}

	// 创建测试索引器
	indexer := createTestIndexer(env, testVisitPattern)

	// 测试 IndexWorkspace - 索引整个工作区
	_, err = indexer.IndexWorkspace(context.Background(), workspaceDir)
	assert.NoError(t, err)

	// 查找工作区中的项目
	projects := env.workspaceReader.FindProjects(env.ctx, workspaceDir, true, testVisitPattern)

	// 验证存储状态 - 确保索引数量与文件数量一致
	validateStorageState(t, env.ctx, env.workspaceReader, env.storage, workspaceDir, projects, testVisitPattern)
}

// TestIndexer_IndexWorkspace_NotExists 测试索引不存在的工作区
func TestIndexer_IndexWorkspace_NotExists(t *testing.T) {
	// 设置测试环境
	env, err := setupTestEnvironment()
	assert.NoError(t, err)
	defer teardownTestEnvironment(t, env)

	// 创建测试索引器
	indexer := createTestIndexer(env, testVisitPattern)

	// 步骤1: 使用不存在的工作区路径
	nonExistentWorkspace := filepath.Join(t.TempDir(), "non_existent_workspace")

	// 步骤2: 尝试索引不存在的工作区，应该返回错误
	_, err = indexer.IndexWorkspace(env.ctx, nonExistentWorkspace)
	assert.ErrorContains(t, err, "not exists")
}
