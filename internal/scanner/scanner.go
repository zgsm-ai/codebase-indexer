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

// 文件状态常量
const (
	FILE_STATUS_ADDED    = "add"
	FILE_STATUS_MODIFIED = "modify"
	FILE_STATUS_DELETED  = "delete"
)

// 同步文件信息
type FileStatus struct {
	Path   string `json:"path"`
	Hash   string `json:"hash"`
	Status string `json:"status"`
}

type ScannerConfig struct {
	IgnorePatterns []string // 忽略规则列表
	MaxFileSizeMB  int      // 文件大小限制，单位MB
}

type ScannerInterface interface {
	SetScannerConfig(config *ScannerConfig)
	CalculateFileHash(filePath string) (string, error)
	ScanDirectory(codebasePath string) (map[string]string, error)
	CalculateFileChanges(local, remote map[string]string) []*FileStatus
}

type FileScanner struct {
	scannerConfig *ScannerConfig
	logger        logger.Logger
}

func NewFileScanner(logger logger.Logger) ScannerInterface {
	defaultScannerConfig := &ScannerConfig{
		IgnorePatterns: storage.DefaultIgnorePatterns,
		MaxFileSizeMB:  storage.DefaultConfigSync.MaxFileSizeMB,
	}
	return &FileScanner{
		scannerConfig: defaultScannerConfig,
		logger:        logger,
	}
}

// SetScannerConfig 设置扫描器配置
func (fs *FileScanner) SetScannerConfig(config *ScannerConfig) {
	if config == nil {
		return
	}
	if len(config.IgnorePatterns) > 0 {
		fs.scannerConfig.IgnorePatterns = config.IgnorePatterns
	}
	if config.MaxFileSizeMB > 0 && config.MaxFileSizeMB <= 10 {
		fs.scannerConfig.MaxFileSizeMB = config.MaxFileSizeMB
	}
}

// 计算文件哈希值
func (fs *FileScanner) CalculateFileHash(filePath string) (string, error) {
	startTime := time.Now()

	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file %s: %v", filePath, err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to calculate hash for file %s: %v", filePath, err)
	}

	hashValue := hex.EncodeToString(hash.Sum(nil))
	fs.logger.Debug("file hash calculated for %s, time taken: %v, hash: %s",
		filePath, time.Since(startTime), hashValue)

	return hashValue, nil
}

// loadIgnoreRules 加载并合并默认忽略规则和.gitignore文件中的规则
func (fs *FileScanner) loadIgnoreRules(codebasePath string) *gitignore.GitIgnore {
	// 首先使用默认规则创建ignore对象
	currentIgnoreRules := fs.scannerConfig.IgnorePatterns
	compiledIgnore := gitignore.CompileIgnoreLines(currentIgnoreRules...)

	// 读取.gitignore文件，并合并
	ignoreFilePath := filepath.Join(codebasePath, ".gitignore")
	if content, err := os.ReadFile(ignoreFilePath); err == nil {
		// 合并.gitignore规则
		var lines []string
		for _, line := range bytes.Split(content, []byte{'\n'}) {
			// 忽略空行和注释行
			trimmedLine := bytes.TrimSpace(line)
			if len(trimmedLine) > 0 && !bytes.HasPrefix(trimmedLine, []byte{'#'}) {
				lines = append(lines, string(trimmedLine))
			}
		}
		if len(lines) > 0 {
			// 将 .gitignore 文件中的规则追加到默认规则之后进行编译
			// 注意：这里的顺序很重要，后添加的规则通常有更高优先级或可以覆盖前面的规则，
			// 具体行为取决于 go-gitignore 库的实现。
			// 通常，更具体的规则（如 !important_file.txt）应该能覆盖更通用的规则（如 *.txt）。
			// go-gitignore 应该能处理标准的 .gitignore 优先级。
			compiledIgnore = gitignore.CompileIgnoreLines(append(currentIgnoreRules, lines...)...)
		}
	} else if !os.IsNotExist(err) {
		fs.logger.Warn("failed to read .gitignore file %s: %v", ignoreFilePath, err)
		// 如果读取失败（非文件不存在错误），则仅使用默认规则
	}
	return compiledIgnore
}

// ScanDirectory 扫描目录并生成哈希树
func (fs *FileScanner) ScanDirectory(codebasePath string) (map[string]string, error) {
	fs.logger.Info("starting directory scan: %s", codebasePath)
	startTime := time.Now()

	hashTree := make(map[string]string)
	var filesScanned int

	ignore := fs.loadIgnoreRules(codebasePath)

	maxFileSizeMB := fs.scannerConfig.MaxFileSizeMB
	maxFileSize := int64(maxFileSizeMB * 1024 * 1024)
	err := filepath.Walk(codebasePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fs.logger.Warn("error accessing file %s: %v", path, err)
			return nil // 继续扫描其他文件
		}

		if info.IsDir() {
			// 对于目录，检查是否应该跳过整个目录
			relPath, _ := filepath.Rel(codebasePath, path)
			// 如果是根目录本身 (relPath is "."), 不要因为 ".*" 规则而跳过它
			if relPath != "." && ignore != nil && ignore.MatchesPath(relPath+"/") {
				fs.logger.Debug("skipping ignored directory: %s", relPath)
				return filepath.SkipDir
			}
			return nil
		}

		// 计算相对路径
		relPath, err := filepath.Rel(codebasePath, path)
		if err != nil {
			fs.logger.Warn("failed to get relative path for file %s: %v", path, err)
			return nil
		}

		// 检查文件是否在.gitignore中被排除
		if ignore != nil && ignore.MatchesPath(relPath) {
			fs.logger.Debug("skipping file excluded by .gitignore: %s", relPath)
			return nil
		}

		// 检查文件大小是否超过最大限制
		if info.Size() >= maxFileSize {
			fs.logger.Debug("skipping file larger than %dMB: %s (size: %.2f MB)", maxFileSizeMB, relPath, float64(info.Size())/1024/1024)
			return nil
		}

		// 计算文件哈希
		hash, err := fs.CalculateFileHash(path)
		if err != nil {
			fs.logger.Warn("error calculating hash for file %s: %v", path, err)
			return nil
		}

		hashTree[relPath] = hash
		filesScanned++

		if filesScanned%100 == 0 {
			fs.logger.Debug("%d files scanned", filesScanned)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan directory: %v", err)
	}

	fs.logger.Info("directory scan completed, %d files scanned, time taken: %v",
		filesScanned, time.Since(startTime))

	return hashTree, nil
}

// 计算文件差异
func (fs *FileScanner) CalculateFileChanges(local, remote map[string]string) []*FileStatus {
	var changes []*FileStatus

	// 检查新增或修改的文件
	for path, localHash := range local {
		if remoteHash, exists := remote[path]; !exists {
			// 新增文件
			changes = append(changes, &FileStatus{
				Path:   path,
				Hash:   localHash,
				Status: FILE_STATUS_ADDED,
			})
		} else if localHash != remoteHash {
			// 修改的文件
			changes = append(changes, &FileStatus{
				Path:   path,
				Hash:   localHash,
				Status: FILE_STATUS_MODIFIED,
			})
		}
	}

	// 检查删除的文件
	for path := range remote {
		if _, exists := local[path]; !exists {
			changes = append(changes, &FileStatus{
				Path:   path,
				Hash:   "",
				Status: FILE_STATUS_DELETED,
			})
		}
	}

	return changes
}
