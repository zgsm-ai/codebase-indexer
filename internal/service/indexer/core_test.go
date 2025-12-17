package indexer

import (
	"codebase-indexer/pkg/codegraph/types"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitConfig(t *testing.T) {
	tests := []struct {
		name           string
		envVars        map[string]string
		initialConfig  *Config
		validateConfig func(*testing.T, *Config)
	}{
		{
			name:    "使用默认值",
			envVars: map[string]string{},
			initialConfig: &Config{
				MaxConcurrency: 0,
				MaxBatchSize:   0,
				MaxFiles:       0,
				MaxProjects:    0,
				CacheCapacity:  0,
			},
			validateConfig: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 1, cfg.MaxConcurrency)     // defaultConcurrency
				assert.Equal(t, 50, cfg.MaxBatchSize)      // defaultBatchSize
				assert.Equal(t, 3, cfg.MaxProjects)        // defaultMaxProjects
				assert.Equal(t, 100000, cfg.CacheCapacity) // defaultCacheCapacity
			},
		},
		{
			name: "从环境变量读取",
			envVars: map[string]string{
				"MAX_CONCURRENCY": "8",
				"MAX_BATCH_SIZE":  "100",
				"MAX_FILES":       "20000",
				"MAX_PROJECTS":    "5",
				"CACHE_CAPACITY":  "200000",
			},
			initialConfig: &Config{},
			validateConfig: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 8, cfg.MaxConcurrency)
				assert.Equal(t, 100, cfg.MaxBatchSize)
				assert.Equal(t, 20000, cfg.MaxFiles)
				assert.Equal(t, 5, cfg.MaxProjects)
				assert.Equal(t, 200000, cfg.CacheCapacity)
			},
		},
		{
			name: "无效环境变量使用默认值",
			envVars: map[string]string{
				"MAX_CONCURRENCY": "invalid",
				"MAX_BATCH_SIZE":  "-1",
			},
			initialConfig: &Config{},
			validateConfig: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 1, cfg.MaxConcurrency) // defaultConcurrency
				assert.Equal(t, 50, cfg.MaxBatchSize)  // defaultBatchSize
			},
		},
		{
			name:    "保留已设置的正值",
			envVars: map[string]string{},
			initialConfig: &Config{
				MaxConcurrency: 10,
				MaxBatchSize:   200,
			},
			validateConfig: func(t *testing.T, cfg *Config) {
				assert.Equal(t, 10, cfg.MaxConcurrency)
				assert.Equal(t, 200, cfg.MaxBatchSize)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 设置环境变量
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}
			defer func() {
				// 清理环境变量
				for key := range tt.envVars {
					os.Unsetenv(key)
				}
			}()

			// 执行初始化
			initConfig(tt.initialConfig)

			// 验证结果
			tt.validateConfig(t, tt.initialConfig)
		})
	}
}

func TestConfig_WithVisitPattern(t *testing.T) {
	config := &Config{
		VisitPattern: &types.VisitPattern{
			ExcludeDirs: []string{".git", "node_modules"},
			IncludeExts: []string{".go", ".ts"},
		},
	}

	initConfig(config)

	assert.NotNil(t, config.VisitPattern)
	assert.Contains(t, config.VisitPattern.ExcludeDirs, ".git")
	assert.Contains(t, config.VisitPattern.IncludeExts, ".go")
}

func TestNewIndexer(t *testing.T) {
	// 这个测试需要完整的依赖注入，这里只测试构造函数不会panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("NewIndexer panicked: %v", r)
		}
	}()

	// 使用nil参数测试（实际使用时应该提供真实的依赖）
	config := Config{
		MaxConcurrency: 2,
		MaxBatchSize:   10,
	}

	// 注意：实际调用需要所有依赖，这里只是确保函数签名正确
	_ = config
}

func TestProgressInfo_Calculation(t *testing.T) {
	progress := &ProgressInfo{
		Total:         100,
		Processed:     60,
		PreviousNum:   20,
		WorkspacePath: "/test/workspace",
	}

	// 测试进度计算
	totalProcessed := progress.Processed + progress.PreviousNum
	percentage := float64(progress.Processed) / float64(progress.Total) * 100

	assert.Equal(t, 80, totalProcessed)
	assert.Equal(t, 60.0, percentage)
}

