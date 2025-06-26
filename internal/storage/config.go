// config.go - Client configuration management

package storage

type ConfigServer struct {
	RegisterExpireMinutes int `json:"registerExpireMinutes"`
	HashTreeExpireHours   int `json:"hashTreeExpireHours"`
}

type ConfigSync struct {
	IntervalMinutes   int      `json:"intervalMinutes"`
	MaxFileSizeMB     int      `json:"maxFileSizeMB"`
	MaxRetries        int      `json:"maxRetries"`
	RetryDelaySeconds int      `json:"retryDelaySeconds"`
	IgnorePatterns    []string `json:"ignorePatterns"`
}

// Client configuration file structure
type ClientConfig struct {
	Server ConfigServer `json:"server"`
	Sync   ConfigSync   `json:"sync"`
}

var DefaultConfigServer = ConfigServer{
	RegisterExpireMinutes: 30, // Default registration validity period in minutes
	HashTreeExpireHours:   24, // Default hash tree validity period in hours
}

var DefaultIgnorePatterns = []string{
	// Filter all files and directories starting with dot
	".*",
	// Keep other specific file types and directories not starting with dot
	"*.swp", "*.swo",
	"*.pyc", "*.class", "*.o", "*.obj",
	"*.log", "*.tmp", "*.bak", "*.backup",
	"*.exe", "*.dll", "*.so", "*.dylib",
	"*.zip", "*.tar", "*.gz", "*.rar",
	// "*.pdf", "*.doc", "*.docx", "*.xls", "*.xlsx", "*.ppt", "*.pptx",
	"*.jpg", "*.jpeg", "*.png", "*.gif", "*.ico", "*.svg",
	"*.mp3", "*.mp4", "*.wav", "*.ogg", "*.flac", "*.aac", "*.wma", "*.m4a",
	"*.sqlite", "*.db", "*.key", "*.crt", "*.cert", "*.pem",
	"logs/", "temp/", "tmp/", "node_modules/",
	"bin/", "dist/", "build/",
	"__pycache__/", "venv/", "target/",
}

var DefaultConfigSync = ConfigSync{
	IntervalMinutes:   5,                     // Default sync interval in minutes
	MaxFileSizeMB:     1,                     // Default maximum file size in MB
	MaxRetries:        3,                     // Default maximum retry count
	RetryDelaySeconds: 3,                     // Default retry delay in seconds
	IgnorePatterns:    DefaultIgnorePatterns, // Default ignore patterns
}

// Default client configuration
var DefaultClientConfig = ClientConfig{
	Server: DefaultConfigServer,
	Sync:   DefaultConfigSync,
}

// Global client configuration
var clientConfig ClientConfig

// Get client configuration
func GetClientConfig() ClientConfig {
	return clientConfig
}

// Set client configuration
func SetClientConfig(config ClientConfig) {
	clientConfig = config
}
