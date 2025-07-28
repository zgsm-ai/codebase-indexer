package service

import (
	"context"
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"codebase-indexer/internal/storage"
	"codebase-indexer/pkg/logger"
)

// CodebaseService 处理代码库相关的业务逻辑
type CodebaseService interface {
	// FindCodebasePaths 查找指定路径下的代码库配置
	FindCodebasePaths(ctx context.Context, basePath, baseName string) ([]storage.CodebaseConfig, error)

	// IsGitRepository 检查路径是否为Git仓库
	IsGitRepository(ctx context.Context, path string) bool

	// GenerateCodebaseID 生成代码库唯一ID
	GenerateCodebaseID(name, path string) string
}

// NewCodebaseService 创建新的代码库服务
func NewCodebaseService(logger logger.Logger) CodebaseService {
	return &codebaseService{
		logger: logger,
	}
}

type codebaseService struct {
	logger logger.Logger
}

// FindCodebasePaths 查找指定路径下的代码库配置
func (s *codebaseService) FindCodebasePaths(ctx context.Context, basePath, baseName string) ([]storage.CodebaseConfig, error) {
	var configs []storage.CodebaseConfig

	if s.IsGitRepository(ctx, basePath) {
		s.logger.Info("path %s is a git repository", basePath)
		configs = append(configs, storage.CodebaseConfig{
			CodebasePath: basePath,
			CodebaseName: baseName,
		})
		return configs, nil
	}

	subDirs, err := os.ReadDir(basePath)
	if err != nil {
		s.logger.Error("failed to read directory %s: %v", basePath, err)
		return nil, fmt.Errorf("failed to read directory %s: %w", basePath, err)
	}

	foundSubRepo := false
	for _, entry := range subDirs {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			subDirPath := filepath.Join(basePath, entry.Name())
			if s.IsGitRepository(ctx, subDirPath) {
				configs = append(configs, storage.CodebaseConfig{
					CodebasePath: subDirPath,
					CodebaseName: entry.Name(),
				})
				foundSubRepo = true
			}
		}
	}

	if !foundSubRepo {
		configs = append(configs, storage.CodebaseConfig{
			CodebasePath: basePath,
			CodebaseName: baseName,
		})
	}

	return configs, nil
}

// IsGitRepository 检查路径是否为Git仓库
func (s *codebaseService) IsGitRepository(ctx context.Context, path string) bool {
	gitPath := filepath.Join(path, ".git")
	if _, err := os.Stat(gitPath); err == nil {
		return true
	}

	// 检查是否为子模块（.git文件）
	gitFile := filepath.Join(path, ".git")
	if info, err := os.Stat(gitFile); err == nil && !info.IsDir() {
		return true
	}

	return false
}

// GenerateCodebaseID 生成代码库唯一ID
func (s *codebaseService) GenerateCodebaseID(name, path string) string {
	// 使用MD5哈希生成唯一ID，结合名称和路径
	data := fmt.Sprintf("%s:%s", name, path)
	hash := md5.Sum([]byte(data))
	return fmt.Sprintf("%x", hash)
}
