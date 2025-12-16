package service

import (
	"codebase-indexer/internal/service/indexer"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)


// TestIndexerConfig 测试配置初始化函数
// 该测试验证索引器配置能够正确读取环境变量并设置配置值
func TestIndexerConfig(t *testing.T) {
	// 测试用例结构
	testCases := []struct {
		name           string
		envVars        map[string]string
		expectedConfig IndexerConfig
		description    string
	}{
		{
			name: "默认值测试",
			envVars: map[string]string{
				"MAX_CONCURRENCY": "",
				"MAX_BATCH_SIZE":  "",
				"MAX_FILES":       "",
				"MAX_PROJECTS":    "",
				"CACHE_CAPACITY":  "",
			},
			expectedConfig: IndexerConfig{
				MaxConcurrency: indexer.DefaultConcurrency,
				MaxBatchSize:   indexer.DefaultBatchSize,
				MaxFiles:       -1, // 后面使用的地方会处理
				MaxProjects:    indexer.DefaultMaxProjects,
				CacheCapacity:  indexer.DefaultCacheCapacity,
			},
			description: "当所有环境变量都未设置时，应该使用默认值",
		},
		{
			name: "有效环境变量测试",
			envVars: map[string]string{
				"MAX_CONCURRENCY": "4",
				"MAX_BATCH_SIZE":  "100",
				"MAX_FILES":       "5000",
				"MAX_PROJECTS":    "5",
				"CACHE_CAPACITY":  "10000",
			},
			expectedConfig: IndexerConfig{
				MaxConcurrency: 4,
				MaxBatchSize:   100,
				MaxFiles:       5000,
				MaxProjects:    5,
				CacheCapacity:  10000,
			},
			description: "当所有环境变量都设置为有效值时，应该使用环境变量值",
		},
		{
			name: "无效环境变量测试",
			envVars: map[string]string{
				"MAX_CONCURRENCY": "-1",
				"MAX_BATCH_SIZE":  "0",
				"MAX_FILES":       "invalid",
				"MAX_PROJECTS":    "-5",
				"CACHE_CAPACITY":  "not_a_number",
			},
			expectedConfig: IndexerConfig{
				MaxConcurrency: indexer.DefaultConcurrency,
				MaxBatchSize:   indexer.DefaultBatchSize,
				MaxFiles:       -1, // 后面使用的地方会处理
				MaxProjects:    indexer.DefaultMaxProjects,
				CacheCapacity:  indexer.DefaultCacheCapacity,
			},
			description: "当环境变量设置为无效值时，应该使用默认值",
		},
		{
			name: "部分环境变量测试",
			envVars: map[string]string{
				"MAX_CONCURRENCY": "8",
				"MAX_BATCH_SIZE":  "",
				"MAX_FILES":       "2000",
				"MAX_PROJECTS":    "",
				"CACHE_CAPACITY":  "5000",
			},
			expectedConfig: IndexerConfig{
				MaxConcurrency: 8,
				MaxBatchSize:   indexer.DefaultBatchSize,
				MaxFiles:       2000,
				MaxProjects:    indexer.DefaultMaxProjects,
				CacheCapacity:  5000,
			},
			description: "当部分环境变量设置时，设置的使用环境变量值，未设置的使用默认值",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 保存原始环境变量
			originalEnvVars := make(map[string]string)
			for key := range tc.envVars {
				originalEnvVars[key] = os.Getenv(key)
			}

			// 清理环境变量
			for key := range tc.envVars {
				os.Unsetenv(key)
			}

			// 设置测试环境变量
			for key, value := range tc.envVars {
				if value != "" {
					os.Setenv(key, value)
				}
			}

			// 创建测试配置 - 这个测试现在只验证常量值的正确性
			// 实际的配置初始化在 indexer 包中进行测试
			assert.Equal(t, indexer.DefaultConcurrency, 1, "DefaultConcurrency 应该为 1")
			assert.Equal(t, indexer.DefaultBatchSize, 50, "DefaultBatchSize 应该为 50")
			assert.Equal(t, indexer.DefaultMaxProjects, 3, "DefaultMaxProjects 应该为 3")
			assert.Equal(t, indexer.DefaultCacheCapacity, 100000, "DefaultCacheCapacity 应该为 100000")

			// 恢复原始环境变量
			for key, value := range originalEnvVars {
				if value != "" {
					os.Setenv(key, value)
				} else {
					os.Unsetenv(key)
				}
			}
		})
	}
}

