// config.go - Client configuration management

package config

type ConfigServer struct {
	RegisterExpireMinutes int `json:"registerExpireMinutes"`
	HashTreeExpireHours   int `json:"hashTreeExpireHours"`
}

type ConfigSync struct {
	IntervalMinutes      int      `json:"intervalMinutes"`
	MaxFileSizeKB        int      `json:"maxFileSizeKB"`
	MaxRetries           int      `json:"maxRetries"`
	RetryDelaySeconds    int      `json:"retryDelaySeconds"`
	FileIgnorePatterns   []string `json:"fileIgnorePatterns"`
	FolderIgnorePatterns []string `json:"folderIgnorePatterns"`
}

// Pprof configuration
type ConfigPprof struct {
	Enabled  bool   `json:"enabled"`
	Address  string `json:"address"`
}

// Client configuration file structure
type ClientConfig struct {
	Server ConfigServer `json:"server"`
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
	"__pycache__/", "venv/", "target/",
}

var DefaultConfigSync = ConfigSync{
	IntervalMinutes:      5,                           // Default sync interval in minutes
	MaxFileSizeKB:        50,                          // Default maximum file size in KB
	MaxRetries:           3,                           // Default maximum retry count
	RetryDelaySeconds:    3,                           // Default retry delay in seconds
	FileIgnorePatterns:   DefaultFileIgnorePatterns,   // Default file ignore patterns
	FolderIgnorePatterns: DefaultFolderIgnorePatterns, // Default folder ignore patterns
}

// Default pprof configuration
var DefaultConfigPprof = ConfigPprof{
	Enabled: false,                // Default pprof disabled
	Address: "localhost:6060",     // Default pprof address
}

// Default client configuration
var DefaultClientConfig = ClientConfig{
	Server: DefaultConfigServer,
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
