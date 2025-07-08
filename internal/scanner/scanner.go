// scanner/scanner.go - File Scanner
package scanner

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"

	"codebase-syncer/internal/storage"
	"codebase-syncer/pkg/logger"

	gitignore "github.com/sabhiram/go-gitignore"
)

// File status constants
const (
	FILE_STATUS_ADDED    = "add"
	FILE_STATUS_MODIFIED = "modify"
	FILE_STATUS_DELETED  = "delete"
)

// File synchronization information
type FileStatus struct {
	Path   string `json:"path"`
	Hash   string `json:"hash"`
	Status string `json:"status"`
}

type ScannerConfig struct {
	FileIgnorePatterns   []string
	FolderIgnorePatterns []string
	MaxFileSizeKB        int // File size limit in KB
}

type ScannerInterface interface {
	SetScannerConfig(config *ScannerConfig)
	GetScannerConfig() *ScannerConfig
	CalculateFileHash(filePath string) (string, error)
	LoadIgnoreRules(codebasePath string) *gitignore.GitIgnore
	LoadFileIgnoreRules(codebasePath string) *gitignore.GitIgnore
	LoadFolderIgnoreRules(codebasePath string) *gitignore.GitIgnore
	ScanDirectory(codebasePath string) (map[string]string, error)
	CalculateFileChanges(local, remote map[string]string) []*FileStatus
}

type FileScanner struct {
	scannerConfig *ScannerConfig
	logger        logger.Logger
	rwMutex       sync.RWMutex
}

func NewFileScanner(logger logger.Logger) ScannerInterface {
	return &FileScanner{
		scannerConfig: defaultScannerConfig(),
		logger:        logger,
	}
}

// defaultScannerConfig returns default scanner configuration
func defaultScannerConfig() *ScannerConfig {
	return &ScannerConfig{
		FileIgnorePatterns:   storage.DefaultConfigSync.FileIgnorePatterns,
		FolderIgnorePatterns: storage.DefaultConfigSync.FolderIgnorePatterns,
		MaxFileSizeKB:        storage.DefaultConfigSync.MaxFileSizeKB,
	}
}

// SetScannerConfig sets the scanner configuration
func (s *FileScanner) SetScannerConfig(config *ScannerConfig) {
	if config == nil {
		return
	}
	s.rwMutex.Lock()
	defer s.rwMutex.Unlock()
	if len(config.FileIgnorePatterns) > 0 {
		s.scannerConfig.FileIgnorePatterns = config.FileIgnorePatterns
	}
	if len(config.FolderIgnorePatterns) > 0 {
		s.scannerConfig.FolderIgnorePatterns = config.FolderIgnorePatterns
	}
	if config.MaxFileSizeKB > 10 && config.MaxFileSizeKB <= 500 {
		s.scannerConfig.MaxFileSizeKB = config.MaxFileSizeKB
	}
}

// GetScannerConfig returns current scanner configuration
func (s *FileScanner) GetScannerConfig() *ScannerConfig {
	s.rwMutex.RLock()
	defer s.rwMutex.RUnlock()
	return s.scannerConfig
}

// CalculateFileHash calculates file hash value
func (s *FileScanner) CalculateFileHash(filePath string) (string, error) {
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
	s.logger.Debug("file hash calculated for %s, time taken: %v, hash: %s",
		filePath, time.Since(startTime), hashValue)

	return hashValue, nil
}

// Load and combine default ignore rules with .gitignore rules
func (s *FileScanner) LoadIgnoreRules(codebasePath string) *gitignore.GitIgnore {
	// First create ignore object with default rules
	fileIngoreRules := s.scannerConfig.FileIgnorePatterns
	folderIgnoreRules := s.scannerConfig.FolderIgnorePatterns
	currentIgnoreRules := append(fileIngoreRules, folderIgnoreRules...)
	compiledIgnore := gitignore.CompileIgnoreLines(currentIgnoreRules...)

	// Read and merge .gitignore file
	gitignoreRules := s.loadGitignore(codebasePath)
	if len(gitignoreRules) > 0 {
		compiledIgnore = gitignore.CompileIgnoreLines(append(gitignoreRules, currentIgnoreRules...)...)
	}

	return compiledIgnore
}

// LoadFileIgnoreRules loads file ignore rules from configuration and merges with .gitignore
func (s *FileScanner) LoadFileIgnoreRules(codebasePath string) *gitignore.GitIgnore {
	// First create ignore object with default rules
	currentIgnoreRules := s.scannerConfig.FileIgnorePatterns
	compiledIgnore := gitignore.CompileIgnoreLines(currentIgnoreRules...)

	// Read and merge .gitignore file
	gitignoreRules := s.loadGitignore(codebasePath)
	if len(gitignoreRules) > 0 {
		compiledIgnore = gitignore.CompileIgnoreLines(append(gitignoreRules, currentIgnoreRules...)...)
	}

	return compiledIgnore
}

// LoadFolderIgnoreRules loads folder ignore rules from configuration and merges with .gitignore
func (s *FileScanner) LoadFolderIgnoreRules(codebasePath string) *gitignore.GitIgnore {
	// First create ignore object with default rules
	currentIgnoreRules := s.scannerConfig.FolderIgnorePatterns
	compiledIgnore := gitignore.CompileIgnoreLines(currentIgnoreRules...)

	// Read and merge .gitignore file
	gitignoreRules := s.loadGitignore(codebasePath)
	if len(gitignoreRules) > 0 {
		compiledIgnore = gitignore.CompileIgnoreLines(append(gitignoreRules, currentIgnoreRules...)...)
	}

	return compiledIgnore
}

// loadGitignore reads .gitignore file and returns list of ignore patterns
func (s *FileScanner) loadGitignore(codebasePath string) []string {
	var ignores []string
	ignoreFilePath := filepath.Join(codebasePath, ".gitignore")
	if content, err := os.ReadFile(ignoreFilePath); err == nil {
		for _, line := range bytes.Split(content, []byte{'\n'}) {
			// Skip empty lines and comments
			trimmedLine := bytes.TrimSpace(line)
			if len(trimmedLine) > 0 && !bytes.HasPrefix(trimmedLine, []byte{'#'}) {
				ignores = append(ignores, string(trimmedLine))
			}
		}
	} else {
		s.logger.Warn("Failed to read .gitignore file: %v", err)
	}
	return ignores
}

// ScanDirectory scans directory and generates hash tree
func (s *FileScanner) ScanDirectory(codebasePath string) (map[string]string, error) {
	s.logger.Info("starting directory scan: %s", codebasePath)
	startTime := time.Now()

	hashTree := make(map[string]string)
	var filesScanned int

	fileIgnore := s.LoadFileIgnoreRules(codebasePath)
	folderIgnore := s.LoadFolderIgnoreRules(codebasePath)

	maxFileSizeKB := s.scannerConfig.MaxFileSizeKB
	maxFileSize := int64(maxFileSizeKB * 1024)
	err := filepath.WalkDir(codebasePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			s.logger.Warn("error accessing file %s: %v", path, err)
			return nil // Continue scanning other files
		}

		// Calculate relative path
		relPath, err := filepath.Rel(codebasePath, path)
		if err != nil {
			s.logger.Warn("failed to get relative path for file %s: %v", path, err)
			return nil
		}

		if d.IsDir() {
			// For directories, check if we should skip entire dir
			// Don't skip root dir (relPath=".") due to ".*" rules
			if relPath != "." && folderIgnore != nil && folderIgnore.MatchesPath(relPath+"/") {
				s.logger.Debug("skipping ignored directory: %s", relPath)
				return fs.SkipDir
			}
			return nil
		}

		// Check if file is excluded by ignore
		if fileIgnore != nil && fileIgnore.MatchesPath(relPath) {
			s.logger.Debug("skipping file excluded by ignore: %s", relPath)
			return nil
		}

		info, err := d.Info()
		if err != nil {
			s.logger.Warn("error getting file info for %s: %v", path, err)
			return nil
		}

		// Verify file size doesn't exceed max limit
		if info.Size() >= maxFileSize {
			s.logger.Debug("skipping file larger than %dKB: %s (size: %.2f KB)", maxFileSizeKB, relPath, float64(info.Size())/1024)
			return nil
		}

		// Calculate file hash
		hash, err := s.CalculateFileHash(path)
		if err != nil {
			s.logger.Warn("error calculating hash for file %s: %v", path, err)
			return nil
		}

		hashTree[relPath] = hash
		filesScanned++

		if filesScanned%100 == 0 {
			s.logger.Debug("%d files scanned", filesScanned)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan directory: %v", err)
	}

	s.logger.Info("directory scan completed, %d files scanned, time taken: %v",
		filesScanned, time.Since(startTime))

	return hashTree, nil
}

// Calculate file differences
func (s *FileScanner) CalculateFileChanges(local, remote map[string]string) []*FileStatus {
	var changes []*FileStatus

	// Check for added or modified files
	for path, localHash := range local {
		if remoteHash, exists := remote[path]; !exists {
			// New file
			changes = append(changes, &FileStatus{
				Path:   path,
				Hash:   localHash,
				Status: FILE_STATUS_ADDED,
			})
		} else if localHash != remoteHash {
			// Modified file
			changes = append(changes, &FileStatus{
				Path:   path,
				Hash:   localHash,
				Status: FILE_STATUS_MODIFIED,
			})
		}
	}

	// Check for deleted files
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
