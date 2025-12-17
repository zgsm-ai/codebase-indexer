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
	ConnMaxIdleTime   time.Duration `json:"connMaxIdleTime"`   // 连接最大空闲时间
	EnableWAL         bool          `json:"enableWAL"`         // 启用WAL模式
	EnableForeignKeys bool          `json:"enableForeignKeys"` // 启用外键约束
	// 分批删除配置
	BatchDeleteSize  int           `json:"batchDeleteSize"`  // 分批删除的批次大小
	BatchDeleteDelay time.Duration `json:"batchDeleteDelay"` // 分批删除之间的延迟
}

// DefaultDatabaseConfig 默认数据库配置
func DefaultDatabaseConfig() *DatabaseConfig {
	return &DatabaseConfig{
		DataDir:           utils.DbDir,
		DatabaseName:      "codebase_indexer.db",
		MaxOpenConns:      1, // SQLite单写设计，强制单连接避免SQLITE_BUSY
		MaxIdleConns:      1,
		ConnMaxLifetime:   0, // 不限制，避免频繁重建连接
		ConnMaxIdleTime:   0, // 不关闭空闲连接
		EnableWAL:         true,
		EnableForeignKeys: true,
		BatchDeleteSize:   1000,                 // 默认每批删除1000条记录
		BatchDeleteDelay:  5 * time.Millisecond, // 默认批次间延迟5毫秒
	}
}
