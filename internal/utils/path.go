// utils/path.go - Path handling utilities
package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	AppRootDir            = "./.zgsm"
	LogsDir               = "./.zgsm/logs"
	CacheDir              = "./.zgsm/cache"
	UploadTmpDir          = "./.zgsm/tmp"
	DbDir                 = "./.zgsm/cache/db"
	WorkspaceDir          = "./.zgsm/cache/workspace"
	WorkspaceEmbeddingDir = "./.zgsm/cache/workspaceEmbedding"
	IndexDir              = "./.zgsm/index"
)

// GetRootDir gets cross-platform root directory
// Returns paths like Windows: %USERPROFILE%/.zgsm, Linux/macOS: ~/.zgsm
func GetRootDir(appName string) (string, error) {
	var rootDir string

	switch runtime.GOOS {
	case "windows":
		// Windows: Use %USERPROFILE% or %APPDATA%
		if userProfile := os.Getenv("USERPROFILE"); userProfile != "" {
			rootDir = filepath.Join(userProfile, "."+appName)
		} else if appData := os.Getenv("APPDATA"); appData != "" {
			rootDir = filepath.Join(appData, appName)
		} else {
			// Fallback: Use current user home directory
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			rootDir = filepath.Join(homeDir, "."+appName)
		}
	case "darwin":
		// macOS: Use ~/Library/Application Support/ or ~/.appname
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		// Option 1: Use standard macOS application support directory
		// rootDir = filepath.Join(homeDir, "Library", "Application Support", appName)
		// Option 2: Use simple hidden directory
		rootDir = filepath.Join(homeDir, "."+appName)
	default:
		// Linux and other Unix-like systems
		// XDG Base Directory Specification standard
		// XDG_CONFIG_HOME: Base directory for user config files
		// - If XDG_CONFIG_HOME is set, typically ~/.config
		// - If not set, defaults to ~/.config
		// - Example paths: ~/.config/appname or /home/username/.config/appname
		if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
			// User customized XDG_CONFIG_HOME, use this path
			// Example: XDG_CONFIG_HOME=/custom/config -> /custom/config/appname
			rootDir = filepath.Join(xdgConfig, appName)
		} else {
			// XDG_CONFIG_HOME not set, use traditional hidden directory
			// Example: ~/.appname or /home/username/.appname
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			rootDir = filepath.Join(homeDir, "."+appName)
		}
	}

	// Ensure config directory exists
	if err := os.MkdirAll(rootDir, 0755); err != nil {
		return "", err
	}

	AppRootDir = rootDir

	return rootDir, nil
}

// GetLogDir gets log directory
func GetLogDir(rootPath string) (string, error) {
	if _, err := os.Stat(rootPath); os.IsNotExist(err) {
		return "", fmt.Errorf("root path %s does not exist", rootPath)
	}

	logPath := filepath.Join(rootPath, "logs")
	// Ensure config directory exists
	if err := os.MkdirAll(logPath, 0755); err != nil {
		return "", err
	}

	LogsDir = logPath

	return logPath, nil
}

// GetCacheDir gets cache directory
func GetCacheDir(rootPath string) (string, error) {
	if _, err := os.Stat(rootPath); os.IsNotExist(err) {
		return "", fmt.Errorf("root path %s does not exist", rootPath)
	}

	cachePath := filepath.Join(rootPath, "cache")
	// Ensure config directory exists
	if err := os.MkdirAll(cachePath, 0755); err != nil {
		return "", err
	}

	CacheDir = cachePath

	return cachePath, nil
}

// GetUploadTmpDir gets temporary upload directory
func GetUploadTmpDir(rootPath string) (string, error) {
	if _, err := os.Stat(rootPath); os.IsNotExist(err) {
		return "", fmt.Errorf("root path %s does not exist", rootPath)
	}

	tmpPath := filepath.Join(rootPath, "tmp")
	// Ensure config directory exists
	if err := os.MkdirAll(tmpPath, 0755); err != nil {
		return "", err
	}

	UploadTmpDir = tmpPath

	return tmpPath, nil
}

func GetCacheDbDir(cachePath string) (string, error) {
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		return "", fmt.Errorf("cache path %s does not exist", cachePath)
	}

	dbPath := filepath.Join(cachePath, "db")
	// Ensure config directory exists
	if err := os.MkdirAll(dbPath, 0755); err != nil {
		return "", err
	}

	DbDir = dbPath

	return dbPath, nil
}

func GetCacheWorkspaceDir(cachePath string) (string, error) {
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		return "", fmt.Errorf("cache path %s does not exist", cachePath)
	}

	workspacePath := filepath.Join(cachePath, "workspace")

	// Ensure config directory exists
	if err := os.MkdirAll(workspacePath, 0755); err != nil {
		return "", err
	}

	WorkspaceDir = workspacePath

	return workspacePath, nil
}

func GetCacheWorkspaceEmbeddingDir(cachePath string) (string, error) {
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		return "", fmt.Errorf("cache path %s does not exist", cachePath)
	}

	workspaceEmbeddingPath := filepath.Join(cachePath, "workspaceEmbedding")

	// Ensure config directory exists
	if err := os.MkdirAll(workspaceEmbeddingPath, 0755); err != nil {
		return "", err
	}

	WorkspaceEmbeddingDir = workspaceEmbeddingPath

	return workspaceEmbeddingPath, nil
}

func GetIndexDir(rootPath string) (string, error) {
	if _, err := os.Stat(rootPath); os.IsNotExist(err) {
		return "", fmt.Errorf("root path %s does not exist", rootPath)
	}

	indexPath := filepath.Join(rootPath, "index")

	// Ensure config directory exists
	if err := os.MkdirAll(indexPath, 0755); err != nil {
		return "", err
	}

	IndexDir = indexPath

	return indexPath, nil
}

// CleanUploadTmpDir cleans temporary upload directory
func CleanUploadTmpDir() error {
	return os.RemoveAll(UploadTmpDir)
}

// Convert Windows path to Unix path
func WindowsAbsolutePathToUnix(path string) string {
	// Convert path separators
	unixPath := filepath.ToSlash(path)
	// Handle Windows drive letters (D:\) converting to Unix style (/d/)
	if len(unixPath) > 1 && unixPath[1] == ':' {
		drive := string(unixPath[0])
		return "/" + strings.ToLower(drive) + unixPath[2:]
	}
	return unixPath
}

// Convert Unix path to Windows path
func UnixAbsolutePathToWindows(path string) string {
	// Handle Unix style paths (/d/) converting to Windows drive letters (D:\)
	if len(path) > 1 && path[0] == '/' {
		drive := string(path[1])
		if len(path) > 2 && path[2] == '/' {
			return strings.ToUpper(drive) + ":" + filepath.FromSlash(path[2:])
		}
	}
	return filepath.FromSlash(path)
}
