// scanner/scanner.go - File Scanner
package repository

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"codebase-indexer/internal/config"
	"codebase-indexer/internal/utils"
	"codebase-indexer/pkg/codegraph/lang"
	"codebase-indexer/pkg/codegraph/types"
	"codebase-indexer/pkg/logger"

	gitignore "github.com/sabhiram/go-gitignore"
)

type ScannerInterface interface {
	SetScannerConfig(config *config.ScannerConfig)
	GetScannerConfig() *config.ScannerConfig
	CalculateFileHash(filePath string) (string, error)
	LoadIgnoreRules(codebasePath string) *gitignore.GitIgnore
	LoadFileIgnoreRules(codebasePath string) *gitignore.GitIgnore
	LoadFolderIgnoreRules(codebasePath string) *gitignore.GitIgnore
	LoadIncludeFiles() []string
	ScanCodebase(codebasePath string) (map[string]string, error)
	ScanFilePaths(codebasePath string, filePaths []string) (map[string]string, error)
	ScanDirectory(codebasePath, dirPath string) (map[string]string, error)
	ScanFile(codebasePath, filePath string) (string, error)
	IsIgnoreFile(codebasePath, filePath string) (bool, error)
	CalculateFileChanges(local, remote map[string]string) []*utils.FileStatus
	CalculateFileChangesWithoutDelete(local, remote map[string]string) []*utils.FileStatus
}

type FileScanner struct {
	scannerConfig *config.ScannerConfig
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
func defaultScannerConfig() *config.ScannerConfig {
	return &config.ScannerConfig{
		FileIgnorePatterns:   config.DefaultConfigSync.FileIgnorePatterns,
		FolderIgnorePatterns: config.DefaultConfigSync.FolderIgnorePatterns,
		FileIncludePatterns:  config.DefaultConfigSync.FileIncludePatterns,
		MaxFileSizeKB:        config.DefaultConfigSync.MaxFileSizeKB,
		MaxFileCount:         config.DefaultConfigSync.MaxFileCount,
	}
}

// SetScannerConfig sets the scanner configuration
func (s *FileScanner) SetScannerConfig(config *config.ScannerConfig) {
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
	if len(config.FileIncludePatterns) > 0 {
		s.scannerConfig.FileIncludePatterns = config.FileIncludePatterns
	}
	if config.MaxFileSizeKB > 10 && config.MaxFileSizeKB <= 500 {
		s.scannerConfig.MaxFileSizeKB = config.MaxFileSizeKB
	}
	if config.MaxFileCount < 100 && config.MaxFileCount > 100000 {
		s.scannerConfig.MaxFileCount = config.MaxFileCount
	}
}

// GetScannerConfig returns current scanner configuration
func (s *FileScanner) GetScannerConfig() *config.ScannerConfig {
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

type ignoreStu struct {
	ignoreRules  *gitignore.GitIgnore
	includeRules []string
	maxFileCount int
	maxFileSize  int
}

func (s *FileScanner) loadIngoreRules(codebasePath string) ignoreStu {
	return ignoreStu{
		ignoreRules:  s.LoadIgnoreRules(codebasePath),
		includeRules: s.LoadIncludeFiles(),
		maxFileCount: s.scannerConfig.MaxFileCount,
		maxFileSize:  s.scannerConfig.MaxFileSizeKB,
	}
}

// skipAll skipDir
func (s *FileScanner) checkIgnoreRules(ignoreRules ignoreStu, filePath *types.FileInfo) (bool, error) {
	return false, nil
}

// Load and combine default ignore rules with .gitignore rules
func (s *FileScanner) LoadIgnoreRules(codebasePath string) *gitignore.GitIgnore {
	// First create ignore object with default rules
	// fileIngoreRules := s.scannerConfig.FileIgnorePatterns
	currentIgnoreRules := s.scannerConfig.FolderIgnorePatterns

	// Read and merge .gitignore file
	gitignoreRules := s.loadGitignore(codebasePath)
	if len(gitignoreRules) > 0 {
		currentIgnoreRules = append(currentIgnoreRules, gitignoreRules...)
	}

	// Read and merge .coignore file
	coignoreRules := s.loadCoignore(codebasePath)
	if len(coignoreRules) > 0 {
		currentIgnoreRules = append(currentIgnoreRules, coignoreRules...)
	}

	// Remove duplicate rules
	uniqueRules := utils.UniqueStringSlice(currentIgnoreRules)

	compiledIgnore := gitignore.CompileIgnoreLines(uniqueRules...)

	return compiledIgnore
}

// LoadFileIgnoreRules loads file ignore rules from configuration and merges with .gitignore
func (s *FileScanner) LoadFileIgnoreRules(codebasePath string) *gitignore.GitIgnore {
	// First create ignore object with default rules
	currentIgnoreRules := s.scannerConfig.FileIgnorePatterns

	// Read and merge .gitignore file
	gitignoreRules := s.loadGitignore(codebasePath)
	if len(gitignoreRules) > 0 {
		currentIgnoreRules = append(currentIgnoreRules, gitignoreRules...)
	}

	// Read and merge .coignore file
	coignoreRules := s.loadCoignore(codebasePath)
	if len(coignoreRules) > 0 {
		currentIgnoreRules = append(currentIgnoreRules, coignoreRules...)
	}

	// Remove duplicate rules
	uniqueRules := utils.UniqueStringSlice(currentIgnoreRules)

	compiledIgnore := gitignore.CompileIgnoreLines(uniqueRules...)

	return compiledIgnore
}

// LoadFolderIgnoreRules loads folder ignore rules from configuration and merges with .gitignore
func (s *FileScanner) LoadFolderIgnoreRules(codebasePath string) *gitignore.GitIgnore {
	// First create ignore object with default rules
	currentIgnoreRules := s.scannerConfig.FolderIgnorePatterns

	// Read and merge .gitignore file
	gitignoreRules := s.loadGitignore(codebasePath)
	if len(gitignoreRules) > 0 {
		currentIgnoreRules = append(currentIgnoreRules, gitignoreRules...)
	}

	// Read and merge .coignore file
	coignoreRules := s.loadCoignore(codebasePath)
	if len(coignoreRules) > 0 {
		currentIgnoreRules = append(currentIgnoreRules, coignoreRules...)
	}

	// Remove duplicate rules
	uniqueRules := utils.UniqueStringSlice(currentIgnoreRules)

	compiledIgnore := gitignore.CompileIgnoreLines(uniqueRules...)

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

func (s *FileScanner) loadCoignore(codebasePath string) []string {
	var ignores []string
	ignoreFilePath := filepath.Join(codebasePath, ".coignore")
	if content, err := os.ReadFile(ignoreFilePath); err == nil {
		for _, line := range bytes.Split(content, []byte{'\n'}) {
			// Skip empty lines and comments
			trimmedLine := bytes.TrimSpace(line)
			if len(trimmedLine) > 0 && !bytes.HasPrefix(trimmedLine, []byte{'#'}) {
				ignores = append(ignores, string(trimmedLine))
			}
		}
	} else {
		s.logger.Warn("Failed to read .coignore file: %v", err)
	}
	return ignores
}

// LoadIncludeFiles returns the list of file extensions to include during scanning
func (s *FileScanner) LoadIncludeFiles() []string {
	includeFiles := s.scannerConfig.FileIncludePatterns

	treeSitterParsers := lang.GetTreeSitterParsers()
	for _, l := range treeSitterParsers {
		if len(includeFiles) == 0 {
			includeFiles = l.SupportedExts
		} else {
			includeFiles = append(includeFiles, l.SupportedExts...)
		}
	}

	return includeFiles
}

// ScanCodebase scans codebase directory and generates hash tree
func (s *FileScanner) ScanCodebase(codebasePath string) (map[string]string, error) {
	s.logger.Info("starting codebase scan: %s", codebasePath)
	startTime := time.Now()

	hashTree := make(map[string]string)
	var filesScanned int

	// fileIgnore := s.LoadFileIgnoreRules(codebasePath)
	// folderIgnore := s.LoadFolderIgnoreRules(codebasePath)
	ignore := s.LoadIgnoreRules(codebasePath)
	fileInclude := s.LoadIncludeFiles()
	fileIncludeMap := utils.StringSlice2Map(fileInclude)

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
			if relPath != "." && ignore != nil && ignore.MatchesPath(relPath+"/") {
				s.logger.Debug("skipping ignored directory: %s", relPath)
				return fs.SkipDir
			}
			return nil
		}

		// Check if file is excluded by ignore
		if ignore != nil && ignore.MatchesPath(relPath) {
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

		// Verify file extension is supported
		if len(fileIncludeMap) > 0 {
			fileExt := filepath.Ext(path)
			if _, ok := fileIncludeMap[fileExt]; !ok {
				s.logger.Debug("skipping file with unsupported extension: %s", relPath)
				return nil
			}
		}

		// Calculate file hash
		hash, err := utils.CalculateFileTimestamp(path)
		if err != nil {
			s.logger.Warn("error calculating hash for file %s: %v", path, err)
			return nil
		}

		filesScanned++
		if filesScanned > s.scannerConfig.MaxFileCount {
			return fmt.Errorf("reached maximum file count limit: %d", filesScanned)
		}

		hashTree[relPath] = strconv.FormatInt(hash, 10)

		return nil
	})

	if err != nil {
		// 检查是否是达到文件数上限的错误
		if err.Error() == fmt.Sprintf("reached maximum file count limit: %d", filesScanned) {
			s.logger.Warn("reached maximum file count limit: %d, stopping scan, time taken: %v", filesScanned, time.Since(startTime))
			return hashTree, nil
		}
		return nil, fmt.Errorf("failed to scan codebase: %v", err)
	}

	s.logger.Info("codebase scan completed, %d files scanned, time taken: %v",
		filesScanned, time.Since(startTime))

	return hashTree, nil
}

// ScanFilePaths scans file paths and generates hash tree
func (s *FileScanner) ScanFilePaths(codebasePath string, filePaths []string) (map[string]string, error) {
	s.logger.Info("starting file paths scan for codebase: %s", codebasePath)
	filesHashTree := make(map[string]string)
	for _, filePath := range filePaths {
		// Check if the file is in this codebase
		relPath, err := filepath.Rel(codebasePath, filePath)
		if err != nil {
			s.logger.Debug("file path %s is not in codebase %s: %v", filePath, codebasePath, err)
			continue
		}

		// Check file size and ignore rules
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			s.logger.Warn("failed to get file info: %s, %v", filePath, err)
			continue
		}

		// If directory
		if fileInfo.IsDir() {
			dirHashTree, err := s.ScanDirectory(codebasePath, filePath)
			if err != nil {
				s.logger.Warn("failed to scan directory: %s, %v", filePath, err)
				continue
			}
			maps.Copy(filesHashTree, dirHashTree)
		} else {
			fileHash, err := s.ScanFile(codebasePath, filePath)
			if err != nil {
				s.logger.Warn("failed to scan file: %s, %v", filePath, err)
				continue
			}
			filesHashTree[relPath] = fileHash
		}
	}
	s.logger.Info("file paths scan completed, scanned %d files", len(filesHashTree))

	return filesHashTree, nil
}

// ScanDirectory scans directory and generates hash tree
func (s *FileScanner) ScanDirectory(codebasePath, dirPath string) (map[string]string, error) {
	s.logger.Info("starting directory scan: %s", dirPath)
	startTime := time.Now()

	hashTree := make(map[string]string)
	var filesScanned int

	// fileIgnore := s.LoadFileIgnoreRules(codebasePath)
	// folderIgnore := s.LoadFolderIgnoreRules(codebasePath)
	ignore := s.LoadIgnoreRules(codebasePath)
	fileInclude := s.LoadIncludeFiles()
	fileIncludeMap := utils.StringSlice2Map(fileInclude)

	maxFileSizeKB := s.scannerConfig.MaxFileSizeKB
	maxFileSize := int64(maxFileSizeKB * 1024)
	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
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
			if relPath != "." && ignore != nil && ignore.MatchesPath(relPath+"/") {
				s.logger.Debug("skipping ignored directory: %s", relPath)
				return fs.SkipDir
			}
			return nil
		}

		// Check if file is excluded by ignore
		if ignore != nil && ignore.MatchesPath(relPath) {
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

		if len(fileIncludeMap) > 0 {
			if _, ok := fileIncludeMap[relPath]; !ok {
				s.logger.Debug("skipping file not included: %s", relPath)
				return nil
			}
		}

		// Calculate file hash
		hash, err := utils.CalculateFileTimestamp(path)
		if err != nil {
			s.logger.Warn("error calculating hash for file %s: %v", path, err)
			return nil
		}

		filesScanned++
		if filesScanned > s.scannerConfig.MaxFileCount {
			return fmt.Errorf("reached maximum file count limit: %d", filesScanned)
		}

		hashTree[relPath] = strconv.FormatInt(hash, 10)

		return nil
	})

	if err != nil {
		// 检查是否是达到文件数上限的错误
		if err.Error() == fmt.Sprintf("reached maximum file count limit: %d", filesScanned) {
			s.logger.Warn("reached maximum file count limit: %d, stopping scan, time taken: %v", filesScanned, time.Since(startTime))
			return hashTree, nil
		}
		return nil, fmt.Errorf("failed to scan directory: %v", err)
	}

	s.logger.Info("directory scan completed, %d files scanned, time taken: %v",
		filesScanned, time.Since(startTime))

	return hashTree, nil
}

// ScanFile scans file and generates hash tree
func (s *FileScanner) ScanFile(codebasePath, filePath string) (string, error) {
	s.logger.Info("starting file scan: %s", filePath)
	startTime := time.Now()

	// fileIgnore := s.LoadFileIgnoreRules(codebasePath)
	ignore := s.LoadIgnoreRules(codebasePath)
	fileInclude := s.LoadIncludeFiles()
	fileIncludeMap := utils.StringSlice2Map(fileInclude)
	maxFileSizeKB := s.scannerConfig.MaxFileSizeKB
	maxFileSize := int64(maxFileSizeKB * 1024)
	relPath, err := filepath.Rel(codebasePath, filePath)
	if err != nil {
		return "", fmt.Errorf("failed to get relative path: %v", err)
	}
	if ignore != nil && ignore.MatchesPath(relPath) {
		return "", fmt.Errorf("file excluded by ignore")
	}
	info, err := os.Stat(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to get file info: %v", err)
	}
	if info.Size() >= maxFileSize {
		return "", fmt.Errorf("file larger than %dKB(size: %.2f KB)", maxFileSizeKB, float64(info.Size())/1024)
	}
	if len(fileIncludeMap) > 0 {
		if _, ok := fileIncludeMap[relPath]; !ok {
			return "", fmt.Errorf("file not included: %s", relPath)
		}
	}
	hash, err := utils.CalculateFileTimestamp(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to scan file: %v", err)
	}

	s.logger.Info("file scan completed, time taken: %v",
		time.Since(startTime))

	return strconv.FormatInt(hash, 10), nil
}

func (s *FileScanner) IsIgnoreFile(codebasePath, filePath string) (bool, error) {
	maxFileSizeKB := s.scannerConfig.MaxFileSizeKB
	maxFileSize := int64(maxFileSizeKB * 1024)
	ignore := s.LoadIgnoreRules(codebasePath)
	if ignore == nil {
		return false, fmt.Errorf("ignore rules not loaded")
	}
	fileInclude := s.LoadIncludeFiles()
	fileIncludeMap := utils.StringSlice2Map(fileInclude)

	relPath, err := filepath.Rel(codebasePath, filePath)
	if err != nil {
		return false, fmt.Errorf("failed to get relative path: %v", err)
	}
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return false, fmt.Errorf("failed to get file info: %v", err)
	}

	// If directory, append "/" and skip size check
	checkPath := relPath
	if fileInfo.IsDir() {
		checkPath = relPath + "/"
	} else if fileInfo.Size() > maxFileSize {
		// For regular files, check size limit
		fileSizeKB := float64(fileInfo.Size()) / 1024
		s.logger.Info("file size exceeded limit: %s (%.2fKB)", filePath, fileSizeKB)
		return true, nil
	}

	if ignore.MatchesPath(checkPath) {
		s.logger.Info("ignore file found: %s in codebase %s", checkPath, codebasePath)
		return true, nil
	}

	if len(fileIncludeMap) > 0 {
		if _, ok := fileIncludeMap[filePath]; ok {
			s.logger.Info("file included: %s in codebase %s", filePath, codebasePath)
			return false, nil
		} else {
			s.logger.Info("file not included: %s in codebase %s", filePath, codebasePath)
			return true, nil
		}
	}

	return false, fmt.Errorf("file not ignored")
}

// Calculate file differences
func (s *FileScanner) CalculateFileChanges(local, remote map[string]string) []*utils.FileStatus {
	var changes []*utils.FileStatus

	// Check for added or modified files
	for path, localHash := range local {
		if remoteHash, exists := remote[path]; !exists {
			// New file
			changes = append(changes, &utils.FileStatus{
				Path:   path,
				Hash:   localHash,
				Status: utils.FILE_STATUS_ADDED,
			})
		} else if localHash != remoteHash {
			// Modified file
			changes = append(changes, &utils.FileStatus{
				Path:   path,
				Hash:   localHash,
				Status: utils.FILE_STATUS_MODIFIED,
			})
		}
	}

	// Check for deleted files
	for path := range remote {
		if _, exists := local[path]; !exists {
			changes = append(changes, &utils.FileStatus{
				Path:   path,
				Hash:   "",
				Status: utils.FILE_STATUS_DELETED,
			})
		}
	}

	return changes
}

// CalculateFileChangesWithoutDelete compares local and remote files, only recording added and modified files
func (s *FileScanner) CalculateFileChangesWithoutDelete(local, remote map[string]string) []*utils.FileStatus {
	var changes []*utils.FileStatus

	// Check for added or modified files
	for path, localHash := range local {
		if remoteHash, exists := remote[path]; !exists {
			// New file
			changes = append(changes, &utils.FileStatus{
				Path:   path,
				Hash:   localHash,
				Status: utils.FILE_STATUS_ADDED,
			})
		} else if localHash != remoteHash {
			// Modified file
			changes = append(changes, &utils.FileStatus{
				Path:   path,
				Hash:   localHash,
				Status: utils.FILE_STATUS_MODIFIED,
			})
		}
	}

	return changes
}
