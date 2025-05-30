// scanner/scanner.go - 文件扫描器
package scanner

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"codebase-syncer/internal/storage"
	"codebase-syncer/pkg/logger"

	gitignore "github.com/sabhiram/go-gitignore"
)

type FileScanner struct {
	logger logger.Logger
}

func NewFileScanner(logger logger.Logger) *FileScanner {
	return &FileScanner{
		logger: logger,
	}
}

// 计算文件哈希值
func (fs *FileScanner) CalculateFileHash(filePath string) (string, error) {
	startTime := time.Now()

	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("无法打开文件 %s: %v", filePath, err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("无法计算文件 %s 的哈希值: %v", filePath, err)
	}

	hashValue := hex.EncodeToString(hash.Sum(nil))
	fs.logger.Debug("计算文件 %s 的哈希值完成，耗时: %v，哈希值: %s",
		filePath, time.Since(startTime), hashValue)

	return hashValue, nil
}

// 扫描目录并生成哈希树
func (fs *FileScanner) ScanDirectory(codebasePath string) (map[string]string, error) {
	fs.logger.Info("开始扫描目录: %s", codebasePath)
	startTime := time.Now()

	hashTree := make(map[string]string)
	var filesScanned int

	// 默认过滤规则
	defaultIgnore := []string{
		".git/", ".svn/", ".hg/",
		".DS_Store", "*.swp", "*.swo",
		"*.log", "*.tmp", "*.bak", "*.backup",
		"logs/", "temp/", "tmp/", "node_modules/",
		"vendor/", "bin/", "dist/", "build/",
	}

	// 首先使用默认规则创建ignore对象
	ignore := gitignore.CompileIgnoreLines(defaultIgnore...)

	// 读取.gitignore文件，并合并
	ignoreFilePath := filepath.Join(codebasePath, ".gitignore")
	if content, err := os.ReadFile(ignoreFilePath); err == nil {
		// 合并.gitignore规则
		var lines []string
		for _, line := range bytes.Split(content, []byte{'\n'}) {
			if len(line) > 0 && !bytes.HasPrefix(line, []byte{'#'}) {
				lines = append(lines, string(line))
			}
		}
		ignore = gitignore.CompileIgnoreLines(append(defaultIgnore, lines...)...)
	} else if !os.IsNotExist(err) {
		fs.logger.Warn("读取.gitignore文件失败: %v", err)
	}

	err := filepath.Walk(codebasePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fs.logger.Warn("访问文件 %s 时出错: %v", path, err)
			return nil // 继续扫描其他文件
		}

		if info.IsDir() {
			// 对于目录，检查是否应该跳过整个目录
			relPath, _ := filepath.Rel(codebasePath, path)
			if ignore != nil && ignore.MatchesPath(relPath+"/") {
				return filepath.SkipDir
			}
			return nil
		}

		// 计算相对路径
		relPath, err := filepath.Rel(codebasePath, path)
		if err != nil {
			fs.logger.Warn("无法获取文件 %s 的相对路径: %v", path, err)
			return nil
		}

		// 检查文件是否在.gitignore中被排除
		if ignore != nil && ignore.MatchesPath(relPath) {
			fs.logger.Debug("跳过被.gitignore排除的文件: %s", relPath)
			return nil
		}

		// 计算文件哈希
		hash, err := fs.CalculateFileHash(path)
		if err != nil {
			fs.logger.Warn("计算文件 %s 的哈希值时出错: %v", path, err)
			return nil
		}

		hashTree[relPath] = hash
		filesScanned++

		if filesScanned%100 == 0 {
			fs.logger.Debug("已扫描 %d 个文件", filesScanned)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("扫描目录失败: %v", err)
	}

	fs.logger.Info("目录扫描完成，共扫描 %d 个文件，耗时: %v",
		filesScanned, time.Since(startTime))

	return hashTree, nil
}

// 计算文件差异 TODO: 待优化
func (fs *FileScanner) CalculateFileChanges(local, remote map[string]string) []*storage.SyncFile {
	var changes []*storage.SyncFile

	// 检查新增或修改的文件
	for path, localHash := range local {
		if remoteHash, exists := remote[path]; !exists {
			// 新增文件
			changes = append(changes, &storage.SyncFile{
				Path:   path,
				Hash:   localHash,
				Status: storage.FILE_STATUS_ADDED,
			})
		} else if localHash != remoteHash {
			// 修改的文件
			changes = append(changes, &storage.SyncFile{
				Path:   path,
				Hash:   localHash,
				Status: storage.FILE_STATUS_MODIFIED,
			})
		}
	}

	// 检查删除的文件
	for path := range remote {
		if _, exists := local[path]; !exists {
			changes = append(changes, &storage.SyncFile{
				Path:   path,
				Hash:   "",
				Status: storage.FILE_STATUS_DELETED,
			})
		}
	}

	return changes
}
