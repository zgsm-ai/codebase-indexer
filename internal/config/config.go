// config.go - Client configuration management

package config

import (
	"codebase-indexer/internal/utils"
	"encoding/json"
	"fmt"
	"os"
)

type ConfigServer struct {
	RegisterExpireMinutes int `json:"registerExpireMinutes"`
	HashTreeExpireHours   int `json:"hashTreeExpireHours"`
}

type ConfigScan struct {
	MaxFileSizeKB                int      `json:"maxFileSizeKB"`
	MaxFileCount                 int      `json:"maxFileCount"`
	FolderIgnorePatterns         []string `json:"folderIgnorePatterns"`
	FileIncludePatterns          []string `json:"fileIncludePatterns"`
	DeepwikiFolderIgnorePatterns []string `json:"deepwikiFolderIgnorePatterns"`
}

type ConfigSync struct {
	IntervalMinutes         int     `json:"intervalMinutes"`
	MaxRetries              int     `json:"maxRetries"`
	RetryDelaySeconds       int     `json:"retryDelaySeconds"`
	EmbeddingSuccessPercent float32 `json:"embeddingSuccessPercent"`
	CodegraphSuccessPercent float32 `json:"codegraphSuccessPercent"`
}

// Pprof configuration
type ConfigPprof struct {
	Enabled bool   `json:"enabled"`
	Address string `json:"address"`
}

// Client configuration file structure
type ClientConfig struct {
	Server ConfigServer `json:"server"`
	Scan   ConfigScan   `json:"scan"`
	Sync   ConfigSync   `json:"sync"`
	Pprof  ConfigPprof  `json:"pprof"`
}

var DefaultConfigServer = ConfigServer{
	RegisterExpireMinutes: 20, // Default registration validity period in minutes
	HashTreeExpireHours:   24, // Default hash tree validity period in hours
}

var DefaultFileIgnorePatterns = []string{
	// Filter all files and directories starting with dot
	".*",
	// Keep other specific file types
	"*.log", "*.tmp", "*.bak", "*.backup",
	"*.swp", "*.swo", "*.ds_store",
	"*.pyc", "*.class", "*.o",
	"*.exe", "*.dll", "*.so", "*.dylib",
	"*.sqlite", "*.db", "*.cache",
	"*.key", "*.crt", "*.cert", "*.pem",
	// images
	"*.jpg", "*.jpeg", "*.jpe", "*.png", "*.gif", "*.ico", "*.icns", "*.svg", "*.eps",
	"*.bmp", "*.tif", "*.tiff", "*.tga", "*.xpm", "*.webp", "*.heif", "*.heic",
	"*.raw", "*.arw", "*.cr2", "*.cr3", "*.nef", "*.nrw", "*.orf", "*.raf", "*.rw2", "*.rwl", "*.pef", "*.srw", "*.x3f", "*.erf", "*.kdc", "*.3fr", "*.mef", "*.mrw", "*.iiq", "*.gpr", "*.dng", // raw formats
	// video
	"*.mp4", "*.m4v", "*.mkv", "*.webm", "*.mov", "*.avi", "*.wmv", "*.flv",
	// audio
	"*.mp3", "*.wav", "*.m4a", "*.flac", "*.ogg", "*.wma", "*.weba", "*.aac", "*.pcm",
	// compressed
	"*.7z", "*.bz2", "*.gz", "*.gz_", "*.tgz", "*.rar", "*.tar", "*.xz",
	"*.zip", "*.vsix", "*.iso", "*.img", "*.pkg",
	// Fonts
	"*.woff", "*.woff2", "*.otf", "*.ttf", "*.eot",
	// 3d formats
	"*.obj", "*.fbx", "*.stl", "*.3ds", "*.dae", "*.blend", "*.ply",
	"*.glb", "*.gltf", "*.max", "*.c4d", "*.ma", "*.mb", "*.pcd",
	// document
	"*.pdf", "*.ai", "*.ps", "*.indd", // PDF and related formats
	"*.doc", "*.docx", "*.xls", "*.xlsx", "*.ppt", "*.pptx",
	"*.rtf", "*.psd", "*.pbix",
	"*.odt", "*.ods", "*.odp", // OpenDocument formats
}

var DefaultFolderIgnorePatterns = []string{
	// Filter all directories starting with dot
	".*",
	// Keep other specific directories not starting with dot
	"logs/", "temp/", "tmp/", "node_modules/",
	"bin/", "dist/", "build/", "out/",
	"__pycache__/", "venv/", "target/", "vendor/",
	// Keep specific directories starting with dot
	"!.costrict/wiki/",
}

var DefaultFileIncludePatterns = []string{
	".md", ".mdx",
	".json", ".yaml",
}

var DefaultDeepwikiFolderIgnorePatterns = []string{
	// Filter all directories starting with dot
	".*",
	// Keep other specific directories not starting with dot
	"logs/", "temp/", "tmp/", "node_modules/",
	"bin/", "dist/", "build/", "out/",
	"__pycache__/", "venv/", "target/", "vendor/",
}

var DefaultConfigScan = ConfigScan{
	MaxFileSizeKB:                10240,                               // Default maximum file size in KB
	MaxFileCount:                 100000,                              // Default maximum file count
	FolderIgnorePatterns:         DefaultFolderIgnorePatterns,         // Default folder ignore patterns
	FileIncludePatterns:          DefaultFileIncludePatterns,          // Default file include patterns
	DeepwikiFolderIgnorePatterns: DefaultDeepwikiFolderIgnorePatterns, // Default deepwiki folder ignore patterns
}

var DefaultConfigSync = ConfigSync{
	IntervalMinutes:         5,    // Default sync interval in minutes
	MaxRetries:              3,    // Default maximum retry count
	RetryDelaySeconds:       3,    // Default retry delay in seconds
	EmbeddingSuccessPercent: 80.0, // Default embedding success percent
	CodegraphSuccessPercent: 90.0, // Default codegraph success percent
}

// Default pprof configuration
var DefaultConfigPprof = ConfigPprof{
	Enabled: false,            // Default pprof disabled
	Address: "localhost:6060", // Default pprof address
}

// Default client configuration
var DefaultClientConfig = ClientConfig{
	Server: DefaultConfigServer,
	Scan:   DefaultConfigScan,
	Sync:   DefaultConfigSync,
	Pprof:  DefaultConfigPprof,
}

// Global client configuration
var clientConfig ClientConfig

// Value client configuration
func GetClientConfig() ClientConfig {
	return clientConfig
}

// Set client configuration
func SetClientConfig(config ClientConfig) {
	clientConfig = config
}

// AppInfo holds application metadata
type AppInfo struct {
	AppName  string `json:"appName"`
	Version  string `json:"version"`
	OSName   string `json:"osName"`
	ArchName string `json:"archName"`
}

var appInfo AppInfo

func GetAppInfo() AppInfo {
	return appInfo
}

func SetAppInfo(info AppInfo) {
	appInfo = info
}

type AuthInfo struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	ClientId  string `json:"machine_id"`
	Token     string `json:"access_token"`
	ServerURL string `json:"base_url"`
}

// Global auth configuration
var authInfo AuthInfo

// GetAuthInfo gets the current auth configuration
func GetAuthInfo() AuthInfo {
	return authInfo
}

// SetAuthInfo sets the auth configuration
func SetAuthInfo(info AuthInfo) {
	authInfo = info
}

// LoadAuthConfig loads auth configuration from auth.json file
func LoadAuthConfig() error {
	// Get auth.json file path
	authFilePath := utils.AuthJsonFile

	// Check if file exists
	if _, err := os.Stat(authFilePath); os.IsNotExist(err) {
		return fmt.Errorf("auth.json file not found at %s", authFilePath)
	}

	// Read file content
	data, err := os.ReadFile(authFilePath)
	if err != nil {
		return fmt.Errorf("failed to read auth.json file: %w", err)
	}

	// Parse JSON content
	var authConfig AuthInfo
	if err := json.Unmarshal(data, &authConfig); err != nil {
		return fmt.Errorf("failed to parse auth.json: %w", err)
	}

	// Set global auth configuration
	authInfo = authConfig

	return nil
}

// LoadAuthConfigWithPath loads auth configuration from specified path
func LoadAuthConfigWithPath(rootPath string) error {
	// Get auth.json file path using utils function
	authFilePath, err := utils.GetAuthJsonFile(rootPath)
	if err != nil {
		return fmt.Errorf("failed to get auth.json file path: %w", err)
	}

	// Check if file exists
	if _, err := os.Stat(authFilePath); os.IsNotExist(err) {
		return fmt.Errorf("auth.json file not found at %s", authFilePath)
	}

	// Read file content
	data, err := os.ReadFile(authFilePath)
	if err != nil {
		return fmt.Errorf("failed to read auth.json file: %w", err)
	}

	// Parse JSON content
	var authConfig AuthInfo
	if err := json.Unmarshal(data, &authConfig); err != nil {
		return fmt.Errorf("failed to parse auth.json: %w", err)
	}

	// Set global auth configuration
	authInfo = authConfig

	return nil
}
