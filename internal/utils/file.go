// utils/file.go - File handling utilities
package utils

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

// File status constants
const (
	FILE_STATUS_ADDED    = "add"
	FILE_STATUS_MODIFIED = "modify"
	FILE_STATUS_DELETED  = "delete"
	FILE_STATUS_RENAME   = "rename"
)

// File synchronization information
type FileStatus struct {
	Path       string `json:"path"`
	TargetPath string `json:"targetPath"`
	Hash       string `json:"hash"`
	Status     string `json:"status"`
	RequestId  string `json:"requestId"`
}

// FileRenamePair 文件重命名对
type FileRenamePair struct {
	OldFilePath string `json:"oldFilePath"`
	NewFilePath string `json:"newFilePath"`
}

// CalculateFileTimestamp calculates file timestamp (modification time)
func CalculateFileTimestamp(filePath string) (int64, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to get file info for %s: %v", filePath, err)
	}

	timestamp := fileInfo.ModTime().UnixMilli()

	return timestamp, nil
}

// CalculateFileHash calculates file hash value
func CalculateFileHash(filePath string) (string, error) {
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

	return hashValue, nil
}

// Calculate file differences
func CalculateFileChanges(local, remote map[string]string) []*FileStatus {
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

// CalculateFileChangesWithoutDelete compares local and remote files, only recording added and modified files
func CalculateFileChangesWithoutDelete(local, remote map[string]string) []*FileStatus {
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

	return changes
}

// AddFileToZip adds a file to zip archive
func AddFileToZip(zipWriter *zip.Writer, fileRelPath string, basePath string) error {
	file, err := os.Open(filepath.Join(basePath, fileRelPath))
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}

	if runtime.GOOS == "windows" {
		fileRelPath = filepath.ToSlash(fileRelPath)
	}
	header.Name = fileRelPath
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, file)
	return err
}
