package handler

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"codebase-syncer/internal/storage"
	"codebase-syncer/internal/syncer"
	"codebase-syncer/test/mocks"
)

var mockLogger = &mocks.MockLogger{}
var appInfo = &AppInfo{
	AppName:  "test-app",
	OSName:   "windows",
	ArchName: "amd64",
	Version:  "1.0.0",
}

func TestNewGRPCHandler(t *testing.T) {
	// 创建测试所需对象
	httpSync := &syncer.HTTPSync{}
	storageManager := &storage.StorageManager{}

	h := NewGRPCHandler(httpSync, storageManager, mockLogger, appInfo)
	assert.NotNil(t, h)
}

func TestIsGitRepository(t *testing.T) {
	// 创建临时目录
	tmpDir := t.TempDir()

	// 创建.git目录
	err := os.Mkdir(filepath.Join(tmpDir, ".git"), 0755)
	assert.NoError(t, err)

	httpSync := &syncer.HTTPSync{}
	storageManager := &storage.StorageManager{}
	h := NewGRPCHandler(httpSync, storageManager, mockLogger, appInfo)

	// 测试有效git仓库
	assert.True(t, h.isGitRepository(tmpDir))

	// 测试无效路径
	assert.False(t, h.isGitRepository(filepath.Join(tmpDir, "nonexistent")))

	// 测试非git目录
	nonGitDir := filepath.Join(tmpDir, "not-git")
	err = os.Mkdir(nonGitDir, 0755)
	assert.NoError(t, err)
	assert.False(t, h.isGitRepository(nonGitDir))
}

func TestFindCodebasePathsToRegister(t *testing.T) {
	mockLogger.On("Info", mock.Anything, mock.Anything).Return()
	mockLogger.On("Error", mock.Anything, mock.Anything).Return()
	// 创建测试目录结构
	baseDir := t.TempDir()

	// 创建子目录结构
	subDir1 := filepath.Join(baseDir, "repo1")
	subDir2 := filepath.Join(baseDir, "repo2")
	nonRepoDir := filepath.Join(baseDir, "notrepo")

	os.Mkdir(subDir1, 0755)
	os.Mkdir(subDir2, 0755)
	os.Mkdir(nonRepoDir, 0755)
	os.Mkdir(filepath.Join(subDir1, ".git"), 0755)
	os.Mkdir(filepath.Join(subDir2, ".git"), 0755)

	httpSync := &syncer.HTTPSync{}
	storageManager := &storage.StorageManager{}
	h := NewGRPCHandler(httpSync, storageManager, mockLogger, appInfo)

	// 测试查找codebase路径
	configs, err := h.findCodebasePathsToRegister(baseDir, "test-name")
	assert.NoError(t, err)
	assert.Len(t, configs, 2) // 应该找到两个git仓库

	// 验证返回的配置
	for _, config := range configs {
		switch config.CodebaseName {
		case "repo1":
			assert.Equal(t, subDir1, config.CodebasePath)
		case "repo2":
			assert.Equal(t, subDir2, config.CodebasePath)
		}
	}

	// 测试无效路径
	_, err = h.findCodebasePathsToRegister(filepath.Join(baseDir, "nonexistent"), "test-name")
	assert.Error(t, err)
}
