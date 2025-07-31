// utils/file.go - File handling utilities
package utils

import (
	"archive/zip"
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
)

// File synchronization information
type FileStatus struct {
	Path   string `json:"path"`
	Hash   string `json:"hash"`
	Status string `json:"status"`
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
