// utils/path.go - 路径处理
package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	AppRootDir   = "./.zgsm"
	LogsDir      = "./.zgsm/logs"
	CacheDir     = "./.zgsm/cache"
	UploadTmpDir = "./.zgsm/tmp"
)

// GetRootDir 获取跨平台的根目录
// 返回类似 Windows: %USERPROFILE%/.zgsm, Linux/macOS: ~/.zgsm 的路径
func GetRootDir(appName string) (string, error) {
	var rootDir string

	switch runtime.GOOS {
	case "windows":
		// Windows: 使用 %USERPROFILE% 或 %APPDATA%
		if userProfile := os.Getenv("USERPROFILE"); userProfile != "" {
			rootDir = filepath.Join(userProfile, "."+appName)
		} else if appData := os.Getenv("APPDATA"); appData != "" {
			rootDir = filepath.Join(appData, appName)
		} else {
			// 备用方案：使用当前用户目录
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			rootDir = filepath.Join(homeDir, "."+appName)
		}
	case "darwin":
		// macOS: 使用 ~/Library/Application Support/ 或 ~/.appname
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		// 可以选择使用标准的 macOS 应用支持目录
		// rootDir = filepath.Join(homeDir, "Library", "Application Support", appName)
		// 或者使用简单的隐藏目录
		rootDir = filepath.Join(homeDir, "."+appName)
	default:
		// Linux 和其他 Unix-like 系统
		// XDG Base Directory Specification 标准
		// XDG_CONFIG_HOME: 用户配置文件的基础目录
		// - 如果设置了 XDG_CONFIG_HOME，通常是 ~/.config
		// - 如果未设置，默认为 ~/.config
		// - 最终路径示例: ~/.config/appname 或 /home/用户名/.config/appname
		if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
			// 用户自定义了 XDG_CONFIG_HOME，使用该路径
			// 例如: XDG_CONFIG_HOME=/custom/config -> /custom/config/appname
			rootDir = filepath.Join(xdgConfig, appName)
		} else {
			// 未设置 XDG_CONFIG_HOME，使用传统的隐藏目录方式
			// 例如: ~/.appname 或 /home/用户名/.appname
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			rootDir = filepath.Join(homeDir, "."+appName)
		}
	}

	// 确保配置目录存在
	if err := os.MkdirAll(rootDir, 0755); err != nil {
		return "", err
	}

	AppRootDir = rootDir

	return rootDir, nil
}

func GetLogDir(rootPath string) (string, error) {
	if _, err := os.Stat(rootPath); os.IsNotExist(err) {
		return "", fmt.Errorf("root path %s does not exist", rootPath)
	}

	logPath := filepath.Join(rootPath, "logs")
	// 确保配置目录存在
	if err := os.MkdirAll(logPath, 0755); err != nil {
		return "", err
	}

	LogsDir = logPath

	return logPath, nil
}

func GetCacheDir(rootPath string) (string, error) {
	if _, err := os.Stat(rootPath); os.IsNotExist(err) {
		return "", fmt.Errorf("root path %s does not exist", rootPath)
	}

	cachePath := filepath.Join(rootPath, "cache")
	// 确保配置目录存在
	if err := os.MkdirAll(cachePath, 0755); err != nil {
		return "", err
	}

	CacheDir = cachePath

	return cachePath, nil
}

func GetUploadTmpDir(rootPath string) (string, error) {
	if _, err := os.Stat(rootPath); os.IsNotExist(err) {
		return "", fmt.Errorf("root path %s does not exist", rootPath)
	}

	tmpPath := filepath.Join(rootPath, "tmp")
	// 确保配置目录存在
	if err := os.MkdirAll(tmpPath, 0755); err != nil {
		return "", err
	}

	UploadTmpDir = tmpPath

	return tmpPath, nil
}

func CleanUploadTmpDir() error {
	return os.RemoveAll(UploadTmpDir)
}

// 将Windows路径转为Unix路径
func WindowsPathToUnix(path string) string {
	// 转换路径分隔符
	unixPath := filepath.ToSlash(path)
	// 处理Windows盘符(D:\)转换为Unix风格(/d/)
	if len(unixPath) > 1 && unixPath[1] == ':' {
		drive := string(unixPath[0])
		return "/" + strings.ToLower(drive) + unixPath[2:]
	}
	return unixPath
}

func UnixPathToWindows(path string) string {
	// 处理Unix风格路径(/d/)转换为Windows盘符(D:\)
	if len(path) > 1 && path[0] == '/' {
		drive := string(path[1])
		if len(path) > 2 && path[2] == '/' {
			return strings.ToUpper(drive) + ":" + filepath.FromSlash(path[2:])
		}
	}
	return filepath.FromSlash(path)
}
