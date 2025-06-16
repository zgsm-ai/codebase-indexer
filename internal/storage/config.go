// config.go - 客户端配置管理

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

// 客户端配置文件结构
type ClientConfig struct {
	Server ConfigServer `json:"server"`
	Sync   ConfigSync   `json:"sync"`
}

var DefaultConfigServer = ConfigServer{
	RegisterExpireMinutes: 30, // 默认注册有效期30分钟
	HashTreeExpireHours:   24, // 默认哈希树有效期24小时
}

var DefaultIgnorePatterns = []string{
	// 过滤所有以点开头的文件和目录
	".*",
	// 保留其他非点开头的特定文件类型和目录
	"*.swp", "*.swo",
	"*.pyc", "*.class", "*.o", "*.obj",
	"*.log", "*.tmp", "*.bak", "*.backup",
	"logs/", "temp/", "tmp/", "node_modules/",
	"vendor/", "bin/", "dist/", "build/",
	"__pycache__/", "venv/", "target/",
}

var DefaultConfigSync = ConfigSync{
	IntervalMinutes:   5,                     // 默认同步间隔5分钟
	MaxFileSizeMB:     1,                     // 默认最大文件大小1MB
	MaxRetries:        3,                     // 默认最大重试次数3次
	RetryDelaySeconds: 5,                     // 默认重试间隔5秒
	IgnorePatterns:    DefaultIgnorePatterns, // 默认忽略模式
}

// 默认客户端配置
var DefaultClientConfig = ClientConfig{
	Server: DefaultConfigServer,
	Sync:   DefaultConfigSync,
}

// 全局客户端配置
var clientConfig ClientConfig

// 获取客户端配置
func GetClientConfig() ClientConfig {
	return clientConfig
}

// 设置客户端配置
func SetClientConfig(config ClientConfig) {
	clientConfig = config
}
