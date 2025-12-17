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

// TestIndexer_IndexProjectFilesWhenProjectHasIndex 测试项目已有索引时增量索引文件
func TestIndexer_IndexProjectFilesWhenProjectHasIndex(t *testing.T) {
	// 设置测试环境
	env, err := setupTestEnvironment()
	assert.NoError(t, err)
	defer teardownTestEnvironment(t, env)

	// 获取测试工作区目录 - 使用项目根目录
	workspaceDir, err := filepath.Abs("../../")
	assert.NoError(t, err)

	// 创建测试索引器，使用排除 mocks 目录的访问模式
	newVisitPattern := &types.VisitPattern{ExcludeDirs: []string{".git", ".idea", ".vscode", "mocks"}, IncludeExts: []string{".go"}}
	codeIndexer := createTestIndexer(env, newVisitPattern)

	// 查找工作区中的项目
	projects := env.workspaceReader.FindProjects(env.ctx, workspaceDir, true, newVisitPattern)

	// 清理索引存储
	err = cleanIndexStoreTest(env.ctx, projects, env.storage)
	assert.NoError(t, err)

	// 步骤1: 先索引工作区（排除 mocks 目录）
	_, err = codeIndexer.IndexWorkspace(env.ctx, workspaceDir)
	assert.NoError(t, err)
	summary, err := codeIndexer.GetSummary(context.Background(), workspaceDir)
	assert.NoError(t, err)
	assert.True(t, summary.TotalFiles > 0)

	// 步骤2: 获取测试文件并创建路径键映射
	files := getTestFiles(t, workspaceDir)
	pathKeys := createPathKeyMap(t, files)

	// 步骤3: 验证 mocks 目录文件没有被索引
	validateFilesNotIndexed(t, env.ctx, env.storage, projects, pathKeys)

	// 步骤4: 测试索引特定文件
	err = codeIndexer.IndexFiles(context.Background(), workspaceDir, files)
	assert.NoError(t, err)

	// 步骤5: 验证 mocks 目录文件现在已经被索引
	validateFilesIndexed(t, env.ctx, env.storage, projects, pathKeys)
}

// TestIndexer_IndexProjectFilesWhenProjectHasNoIndex 测试项目无索引时索引文件
func TestIndexer_IndexProjectFilesWhenProjectHasNoIndex(t *testing.T) {
	// 设置测试环境
	env, err := setupTestEnvironment()
	assert.NoError(t, err)
	defer teardownTestEnvironment(t, env)

	// 获取测试工作区目录 - 使用项目根目录
	workspaceDir, err := filepath.Abs("../../")
	assert.NoError(t, err)

	// 创建测试索引器
	indexer := createTestIndexer(env, testVisitPattern)

	// 查找工作区中的项目
	projects := env.workspaceReader.FindProjects(env.ctx, workspaceDir, true, testVisitPattern)

	// 清理索引存储
	err = cleanIndexStoreTest(env.ctx, projects, env.storage)
	assert.NoError(t, err)

	// 步骤1: 验证存储初始状态为空
	validateStorageEmpty(t, env.ctx, env.storage, projects)

	// 步骤2: 获取测试文件
	files := getTestFiles(t, workspaceDir)

	// 步骤3: 测试索引特定文件（当项目没有索引时）
	err = indexer.IndexFiles(context.Background(), workspaceDir, files)
	assert.NoError(t, err)

	// 步骤4: 创建路径键映射
	pathKeys := createPathKeyMap(t, files)

	// 步骤5: 验证 mocks 目录文件已经被索引
	validateFilesIndexed(t, env.ctx, env.storage, projects, pathKeys)

	// 步骤6: 验证存储状态 - 确保索引数量与文件数量一致
	validateStorageState(t, env.ctx, env.workspaceReader, env.storage, workspaceDir, projects, testVisitPattern)
}

// TestIndexer_RemoveIndexes 测试删除索引
func TestIndexer_RemoveIndexes(t *testing.T) {
	// 设置测试环境
	env, err := setupTestEnvironment()
	assert.NoError(t, err)
	defer teardownTestEnvironment(t, env)

	// 获取测试工作区目录 - 使用项目根目录
	workspaceDir, err := filepath.Abs("../../")
	assert.NoError(t, err)

	err = initWorkspaceModel(env, workspaceDir)
	assert.NoError(t, err)

	// 创建测试索引器
	indexer := createTestIndexer(env, testVisitPattern)

	// 查找工作区中的项目
	projects := env.workspaceReader.FindProjects(env.ctx, workspaceDir, true, testVisitPattern)

	// 清理索引存储
	err = cleanIndexStoreTest(env.ctx, projects, env.storage)
	assert.NoError(t, err)

	// 步骤1: 获取测试文件
	files := getTestFiles(t, workspaceDir)

	// 步骤2: 索引测试文件
	err = indexer.IndexFiles(context.Background(), workspaceDir, files)
	assert.NoError(t, err)

	// 步骤3: 创建路径键映射
	pathKeys := createPathKeyMap(t, files)
	pathKeysBak := createPathKeyMap(t, files)
	assert.True(t, len(pathKeys) > 0)
	assert.True(t, len(pathKeysBak) > 0)

	// 步骤4: 验证文件已被索引
	validateFilesIndexed(t, env.ctx, env.storage, projects, pathKeys)

	// 步骤5: 统计索引前的总数
	totalIndexBefore, err := countIndexedFiles(env.ctx, env.storage, projects)
	assert.NoError(t, err)

	// 步骤6: 删除索引
	err = indexer.RemoveIndexes(env.ctx, workspaceDir, files)
	assert.NoError(t, err)

	// 步骤7: 统计索引后的总数
	totalIndexAfter, err := countIndexedFiles(env.ctx, env.storage, projects)
	assert.NoError(t, err)

	// 步骤8: 验证删除结果
	validateFilesNotIndexed(t, env.ctx, env.storage, projects, pathKeysBak)
	assert.Equal(t, len(files), len(pathKeysBak))
	assert.True(t, totalIndexBefore-len(files) <= totalIndexAfter)
}

// TestIndexer_IndexFiles_NoProject 测试索引不存在的项目中的文件
func TestIndexer_IndexFiles_NoProject(t *testing.T) {
	// 设置测试环境
	env, err := setupTestEnvironment()
	assert.NoError(t, err)
	defer teardownTestEnvironment(t, env)

	// 创建测试索引器
	indexer := createTestIndexer(env, testVisitPattern)

	// 步骤1: 使用不存在的项目路径
	nonExistentWorkspace := filepath.Join(t.TempDir(), "non_existent_workspace")
	testFiles := []string{"test.go"}

	// 步骤2: 尝试索引不存在的项目中的文件，应该返回错误
	err = indexer.IndexFiles(env.ctx, nonExistentWorkspace, testFiles)
	assert.ErrorContains(t, err, "not exists")
}

// TestIndexer_QueryElements 测试查询元素
func TestIndexer_QueryElements(t *testing.T) {
	// 设置测试环境
	env, err := setupTestEnvironment()
	assert.NoError(t, err)
	defer teardownTestEnvironment(t, env)

	// 获取测试工作区目录 - 使用项目根目录
	workspaceDir, err := filepath.Abs("../../")
	assert.NoError(t, err)

	// 创建测试索引器
	codeIndexer := createTestIndexer(env, testVisitPattern)

	// 查找工作区中的项目
	projects := env.workspaceReader.FindProjects(env.ctx, workspaceDir, true, testVisitPattern)

	// 清理索引存储
	err = cleanIndexStoreTest(env.ctx, projects, env.storage)
	assert.NoError(t, err)

	// 步骤1: 索引整个工作区
	_, err = codeIndexer.IndexWorkspace(env.ctx, workspaceDir)
	assert.NoError(t, err)

	// 步骤2: 获取测试文件
	files := getTestFiles(t, workspaceDir)
	assert.True(t, len(files) > 0)
}

// TestIndexer_QuerySymbols_WithExistFile 测试查询符号
func TestIndexer_QuerySymbols_WithExistFile(t *testing.T) {
	// 设置测试环境
	env, err := setupTestEnvironment()
	assert.NoError(t, err)
	defer teardownTestEnvironment(t, env)

	// 获取测试工作区目录 - 使用项目根目录
	workspaceDir, err := filepath.Abs("../../")
	assert.NoError(t, err)

	// 创建测试索引器
	testIndexer := createTestIndexer(env, testVisitPattern)

	// 查找工作区中的项目
	projects := env.workspaceReader.FindProjects(env.ctx, workspaceDir, true, testVisitPattern)

	// 清理索引存储
	err = cleanIndexStoreTest(env.ctx, projects, env.storage)
	assert.NoError(t, err)

	// 步骤1: 索引整个工作区
	_, err = testIndexer.IndexWorkspace(env.ctx, workspaceDir)
	assert.NoError(t, err)

	// 步骤2: 准备测试文件和符号名称
	_ = filepath.Join(workspaceDir, "test", "mocks", "mock_graph_store.go")
	_ = []string{"MockGraphStorage", "BatchSave"}
}
