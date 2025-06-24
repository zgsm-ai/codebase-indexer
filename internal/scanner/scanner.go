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
	IgnorePatterns []string // List of ignore patterns
	MaxFileSizeMB  int      // File size limit in MB
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
		IgnorePatterns: storage.DefaultIgnorePatterns,
		MaxFileSizeMB:  storage.DefaultConfigSync.MaxFileSizeMB,
	}
}

// SetScannerConfig sets the scanner configuration
func (s *FileScanner) SetScannerConfig(config *ScannerConfig) {
	if config == nil {
		return
	}
	s.rwMutex.Lock()
	defer s.rwMutex.Unlock()
	if len(config.IgnorePatterns) > 0 {
		s.scannerConfig.IgnorePatterns = config.IgnorePatterns
	}
	if config.MaxFileSizeMB > 0 && config.MaxFileSizeMB <= 10 {
		s.scannerConfig.MaxFileSizeMB = config.MaxFileSizeMB
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
func (s *FileScanner) loadIgnoreRules(codebasePath string) *gitignore.GitIgnore {
	// First create ignore object with default rules
	currentIgnoreRules := s.scannerConfig.IgnorePatterns
	compiledIgnore := gitignore.CompileIgnoreLines(currentIgnoreRules...)

	// Read and merge .gitignore file
	ignoreFilePath := filepath.Join(codebasePath, ".gitignore")
	if content, err := os.ReadFile(ignoreFilePath); err == nil {
		// Merge .gitignore rules
		var lines []string
		for _, line := range bytes.Split(content, []byte{'\n'}) {
			// Skip empty lines and comments
			trimmedLine := bytes.TrimSpace(line)
			if len(trimmedLine) > 0 && !bytes.HasPrefix(trimmedLine, []byte{'#'}) {
				lines = append(lines, string(trimmedLine))
			}
		}
		if len(lines) > 0 {
			// Append .gitignore rules after default rules
			// Note: Order matters here - later rules have higher priority
			// More specific rules (e.g. !important_file.txt) override general ones (e.g. *.txt)
			// go-gitignore handles standard .gitignore priority rules
			compiledIgnore = gitignore.CompileIgnoreLines(append(currentIgnoreRules, lines...)...)
		}
	} else if !os.IsNotExist(err) {
		s.logger.Warn("failed to read .gitignore file %s: %v", ignoreFilePath, err)
		// If read fails (not a file-not-exist error), use default rules only
	}
	return compiledIgnore
}

// ScanDirectory scans directory and generates hash tree
func (s *FileScanner) ScanDirectory(codebasePath string) (map[string]string, error) {
	s.logger.Info("starting directory scan: %s", codebasePath)
	startTime := time.Now()

	hashTree := make(map[string]string)
	var filesScanned int

	ignore := s.loadIgnoreRules(codebasePath)

	maxFileSizeMB := s.scannerConfig.MaxFileSizeMB
	maxFileSize := int64(maxFileSizeMB * 1024 * 1024)
	err := filepath.WalkDir(codebasePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			s.logger.Warn("error accessing file %s: %v", path, err)
			return nil // Continue scanning other files
		}

		if d.IsDir() {
			// For directories, check if we should skip entire dir
			relPath, _ := filepath.Rel(codebasePath, path)
			// Don't skip root dir (relPath=".") due to ".*" rules
			if relPath != "." && ignore != nil && ignore.MatchesPath(relPath+"/") {
				s.logger.Debug("skipping ignored directory: %s", relPath)
				return fs.SkipDir
			}
			return nil
		}

		info, err := d.Info()
		if err != nil {
			s.logger.Warn("error getting file info for %s: %v", path, err)
			return nil
		}

		// Calculate relative path
		relPath, err := filepath.Rel(codebasePath, path)
		if err != nil {
			s.logger.Warn("failed to get relative path for file %s: %v", path, err)
			return nil
		}

		// Check if file is excluded by .gitignore
		if ignore != nil && ignore.MatchesPath(relPath) {
			s.logger.Debug("skipping file excluded by .gitignore: %s", relPath)
			return nil
		}

		// Verify file size doesn't exceed max limit
		if info.Size() >= maxFileSize {
			s.logger.Debug("skipping file larger than %dMB: %s (size: %.2f MB)", maxFileSizeMB, relPath, float64(info.Size())/1024/1024)
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
