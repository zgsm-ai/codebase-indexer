package config

import (
	"codebase-indexer/internal/utils"
	"time"
)

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	DataDir           string        `json:"dataDir"`           // 数据库文件存储目录
	DatabaseName      string        `json:"databaseName"`      // 数据库文件名
	MaxOpenConns      int           `json:"maxOpenConns"`      // 最大打开连接数
	MaxIdleConns      int           `json:"maxIdleConns"`      // 最大空闲连接数
	ConnMaxLifetime   time.Duration `json:"connMaxLifetime"`   // 连接最大生命周期
	EnableWAL         bool          `json:"enableWAL"`         // 启用WAL模式
	EnableForeignKeys bool          `json:"enableForeignKeys"` // 启用外键约束
}

// DefaultDatabaseConfig 默认数据库配置
func DefaultDatabaseConfig() *DatabaseConfig {
	return &DatabaseConfig{
		DataDir:           utils.DbDir,
		DatabaseName:      "codebase_indexer.db",
		MaxOpenConns:      25,
		MaxIdleConns:      10,
		ConnMaxLifetime:   5 * time.Minute,
		EnableWAL:         true,
		EnableForeignKeys: true,
	}
}
