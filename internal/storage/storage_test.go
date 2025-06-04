package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockLogger 是一个 mock logger 实现
type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Debug(format string, v ...interface{}) {
	fmt.Printf("[MOCK DEBUG] %s\n", fmt.Sprintf(format, v...)) // 输出日志
	// m.Called(format, v)
}

func (m *MockLogger) Info(format string, v ...interface{}) {
	fmt.Printf("[MOCK INFO] %s\n", fmt.Sprintf(format, v...)) // 输出日志
	// m.Called(format, v)
}

func (m *MockLogger) Warn(format string, v ...interface{}) {
	fmt.Printf("[MOCK WARN] %s\n", fmt.Sprintf(format, v...)) // 输出日志
	// m.Called(format, v)
}

func (m *MockLogger) Error(format string, v ...interface{}) {
	fmt.Printf("[MOCK ERROR] %s\n", fmt.Sprintf(format, v...)) // 输出日志
	// m.Called(format, v)
}

func (m *MockLogger) Fatal(format string, v ...interface{}) {
	fmt.Printf("[MOCK FATAL] %s\n", fmt.Sprintf(format, v...)) // 输出日志
	// m.Called(format, v)
}

func TestNewStorageManager(t *testing.T) {
	t.Run("创建新目录", func(t *testing.T) {
		// 设置临时目录
		tempDir := t.TempDir()
		codebasePath := filepath.Join(tempDir, "codebase")

		// 确保目录不存在
		if _, err := os.Stat(codebasePath); !os.IsNotExist(err) {
			t.Fatalf("测试目录应该不存在: %v", err)
		}

		logger := &MockLogger{}
		// logger.On("Fatal", mock.Anything, mock.Anything).Return()

		sm := NewStorageManager(tempDir, logger)

		// 验证目录是否创建
		if _, err := os.Stat(codebasePath); os.IsNotExist(err) {
			t.Fatalf("应该创建了 codebase 目录: %v", err)
		}

		// 验证结构体字段
		assert.Equal(t, codebasePath, sm.codebasePath)
		assert.Equal(t, logger, sm.logger)
		assert.NotNil(t, sm.codebaseConfigs)
		assert.Empty(t, sm.codebaseConfigs)
	})

	t.Run("目录已存在", func(t *testing.T) {
		tempDir := t.TempDir()
		codebasePath := filepath.Join(tempDir, "codebase")

		// 预先创建目录
		if err := os.Mkdir(codebasePath, 0755); err != nil {
			t.Fatalf("无法创建测试目录: %v", err)
		}

		logger := &MockLogger{}
		sm := NewStorageManager(tempDir, logger)

		// 验证没有调用 Fatal
		// logger.AssertNotCalled(t, "Fatal", mock.Anything, mock.Anything)

		// 验证结构体字段
		assert.Equal(t, codebasePath, sm.codebasePath)
		assert.Equal(t, logger, sm.logger)
		assert.NotNil(t, sm.codebaseConfigs)
		assert.Empty(t, sm.codebaseConfigs)
	})

	t.Run("目录创建失败", func(t *testing.T) {
		// 使用系统根目录应该会触发权限错误
		logger := &MockLogger{}
		// logger.On("Fatal", "无法创建codebase目录: %v", mock.Anything).Return()

		sm := NewStorageManager("/", logger)

		// 验证 Fatal 被调用
		// logger.AssertCalled(t, "Fatal", "无法创建codebase目录: %v", mock.Anything)

		// 即使在错误情况下也应该返回有效的 StorageManager
		assert.NotNil(t, sm)
	})
}

func TestGetCodebaseConfigs(t *testing.T) {
	t.Run("空配置", func(t *testing.T) {
		cm := &StorageManager{
			codebaseConfigs: make(map[string]*CodebaseConfig),
		}

		configs := cm.GetCodebaseConfigs()
		assert.Empty(t, configs)
		assert.NotNil(t, configs) // 确保不是 nil
	})

	t.Run("有配置数据", func(t *testing.T) {
		cm := &StorageManager{
			codebaseConfigs: map[string]*CodebaseConfig{
				"test1": {},
				"test2": {},
			},
		}

		configs := cm.GetCodebaseConfigs()
		assert.Equal(t, 2, len(configs))
		assert.Equal(t, cm.codebaseConfigs, configs)
	})

	t.Run("返回的是引用", func(t *testing.T) {
		cm := &StorageManager{
			codebaseConfigs: make(map[string]*CodebaseConfig),
		}

		configs := cm.GetCodebaseConfigs()
		configs["test"] = &CodebaseConfig{} // 修改返回的 map

		// 验证修改是否影响了原始数据
		assert.NotEmpty(t, cm.codebaseConfigs)
		assert.Equal(t, cm.codebaseConfigs, configs)
	})
}

func TestGetCodebaseConfig(t *testing.T) {
	logger := &MockLogger{}

	t.Run("从内存获取现有配置", func(t *testing.T) {
		configs := map[string]*CodebaseConfig{
			"test1": {CodebaseId: "test1"},
		}
		cm := &StorageManager{
			codebaseConfigs: configs,
			logger:          logger,
			mutex:           sync.RWMutex{},
		}

		config, err := cm.GetCodebaseConfig("test1")
		assert.NoError(t, err)
		assert.Same(t, cm.codebaseConfigs["test1"], config)
	})

	t.Run("从文件加载新配置", func(t *testing.T) {
		tempDir := t.TempDir()
		file := "test2"
		if err := os.WriteFile(filepath.Join(tempDir, file), []byte(`{"codebaseId": "`+file+`"}`), 0644); err != nil {
			t.Fatalf("无法创建测试文件 %s: %v", file, err)
		}
		cm := &StorageManager{
			codebasePath:    tempDir,
			codebaseConfigs: make(map[string]*CodebaseConfig),
			logger:          logger,
			mutex:           sync.RWMutex{},
		}

		expectedConfig := &CodebaseConfig{CodebaseId: file}
		// logger.On("Info", "加载codebase文件内容: %s", []interface{}{"test2"}).Return()
		// logger.On("Info", "成功加载codebase文件，上次同步时间: %s", []interface{}{"0001-01-01T00:00:00Z"}).Return()

		config, err := cm.GetCodebaseConfig("test2")
		assert.NoError(t, err)
		assert.Equal(t, expectedConfig, config)
		assert.Equal(t, expectedConfig, cm.codebaseConfigs["test2"])
	})

	t.Run("配置不存在返回错误", func(t *testing.T) {
		tempDir := t.TempDir()
		cm := &StorageManager{
			codebasePath:    tempDir,
			codebaseConfigs: make(map[string]*CodebaseConfig),
			logger:          logger,
			mutex:           sync.RWMutex{},
		}

		// logger.On("Info", "加载codebase文件内容: %s", []interface{}{"test3"}).Return()
		// logger.On("Info", "成功加载codebase文件，上次同步时间: %s", []interface{}{"0001-01-01T00:00:00Z"}).Return()
		config, err := cm.GetCodebaseConfig("test3")
		assert.ErrorContains(t, err, "codebase文件不存在")
		assert.Nil(t, config)
		assert.Empty(t, cm.codebaseConfigs)
	})

	t.Run("并发访问安全", func(t *testing.T) {
		tempDir := t.TempDir()
		cm := &StorageManager{
			codebasePath:    tempDir,
			codebaseConfigs: make(map[string]*CodebaseConfig),
			logger:          logger,
			mutex:           sync.RWMutex{},
		}

		for i := 0; i < 100; i++ {
			file := fmt.Sprintf("test%d", i)
			if err := os.WriteFile(filepath.Join(tempDir, file), []byte(`{"codebaseId": "`+file+`"}`), 0644); err != nil {
				t.Fatalf("无法创建测试文件 %s: %v", file, err)
			}
		}

		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				_, err := cm.GetCodebaseConfig(fmt.Sprintf("test%d", id))
				assert.NoError(t, err)
			}(i)
		}
		wg.Wait()
	})
}

func TestConfigManager_loadAllConfigs(t *testing.T) {
	logger := &MockLogger{}

	t.Run("目录读取失败", func(t *testing.T) {
		logger := &MockLogger{}
		// logger.On("Error", "读取codebase目录失败: %v", mock.Anything).Return()

		// 无权限目录
		cm := &StorageManager{
			codebasePath:    "/root", // Linux下无权限的目录
			codebaseConfigs: make(map[string]*CodebaseConfig),
			logger:          logger,
			mutex:           sync.RWMutex{},
		}

		// 执行
		cm.loadAllConfigs()

		// 验证
		// logger.AssertCalled(t, "Error", "读取codebase目录失败: %v", mock.Anything)
	})

	t.Run("目录中没有文件", func(t *testing.T) {
		tempDir := t.TempDir()
		cm := &StorageManager{
			codebasePath:    tempDir,
			codebaseConfigs: make(map[string]*CodebaseConfig),
			logger:          logger,
			mutex:           sync.RWMutex{},
		}

		// 执行
		cm.loadAllConfigs()

		// 验证
		assert.Empty(t, cm.codebaseConfigs)
	})

	t.Run("目录中有子目录", func(t *testing.T) {
		tempDir := t.TempDir()
		// 创建子目录
		if err := os.Mkdir(filepath.Join(tempDir, "subdir"), 0755); err != nil {
			t.Fatalf("无法创建测试子目录: %v", err)
		}

		cm := &StorageManager{
			codebasePath:    tempDir,
			codebaseConfigs: make(map[string]*CodebaseConfig),
			logger:          logger,
			mutex:           sync.RWMutex{},
		}

		// 执行
		cm.loadAllConfigs()

		// 验证
		assert.Empty(t, cm.codebaseConfigs)
	})

	t.Run("成功加载配置文件", func(t *testing.T) {
		tempDir := t.TempDir()
		// 创建测试文件
		testFiles := []string{"config1", "config2"}
		for _, f := range testFiles {
			if err := os.WriteFile(filepath.Join(tempDir, f), []byte(`{"codebaseId": "`+f+`"}`), 0644); err != nil {
				t.Fatalf("无法创建测试文件 %s: %v", f, err)
			}
		}

		cm := &StorageManager{
			codebasePath:    tempDir,
			codebaseConfigs: make(map[string]*CodebaseConfig),
			logger:          logger,
			mutex:           sync.RWMutex{},
		}

		// 执行
		cm.loadAllConfigs()

		// 验证
		assert.Equal(t, len(testFiles), len(cm.codebaseConfigs))
	})

	t.Run("部分文件加载失败", func(t *testing.T) {
		logger := &MockLogger{}
		// logger.On("Error", "加载codebase文件 %s 失败: %v", mock.Anything, mock.Anything).Return()

		tempDir := t.TempDir()
		// 创建测试文件
		testFiles := []string{"good", "bad"}
		for _, f := range testFiles {
			if strings.HasSuffix(f, "bad") {
				if err := os.WriteFile(filepath.Join(tempDir, f), []byte("text"), 0644); err != nil {
					t.Fatalf("无法创建测试文件 %s: %v", f, err)
				}
				continue
			}
			if err := os.WriteFile(filepath.Join(tempDir, f), []byte(`{"codebaseId": "good"}`), 0644); err != nil {
				t.Fatalf("无法创建测试文件 %s: %v", f, err)
			}
		}

		cm := &StorageManager{
			codebasePath:    tempDir,
			codebaseConfigs: make(map[string]*CodebaseConfig),
			logger:          logger,
			mutex:           sync.RWMutex{},
		}

		// 执行
		cm.loadAllConfigs()

		// 验证
		assert.Equal(t, 1, len(cm.codebaseConfigs))
		// logger.AssertCalled(t, "Error", "加载codebase文件 %s 失败: %v", mock.Anything, mock.Anything)
	})
}

func TestConfigManager_loadCodebaseConfig(t *testing.T) {

	t.Run("文件不存在", func(t *testing.T) {
		logger := &MockLogger{}
		tempDir := t.TempDir()
		cm := &StorageManager{
			codebasePath:    tempDir,
			codebaseConfigs: make(map[string]*CodebaseConfig),
			logger:          logger,
			mutex:           sync.RWMutex{},
		}

		// 模拟调用
		config, err := cm.loadCodebaseConfig("nonexistent.json")

		// 验证
		assert.Nil(t, config)
		assert.ErrorContains(t, err, "codebase文件不存在")
		// logger.AssertCalled(t, "Info", "加载codebase文件内容: %s", mock.Anything)
	})

	t.Run("JSON解析失败", func(t *testing.T) {
		logger := &MockLogger{}
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "invalid")
		if err := os.WriteFile(filePath, []byte("{invalid json}"), 0644); err != nil {
			t.Fatal(err)
		}

		cm := &StorageManager{
			codebasePath:    tempDir,
			codebaseConfigs: make(map[string]*CodebaseConfig),
			logger:          logger,
		}

		// 模拟调用
		config, err := cm.loadCodebaseConfig("invalid")

		// 验证
		assert.Nil(t, config)
		assert.ErrorContains(t, err, "解析codebase文件失败")
		// logger.AssertCalled(t, "Info", "加载codebase文件内容: %s", mock.Anything)
	})

	t.Run("CodebaseId不匹配", func(t *testing.T) {
		logger := &MockLogger{}
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "mismatch")
		testData := `{"codebaseId":"other-id","lastSync":"2025-01-01T00:00:00Z"}`
		if err := os.WriteFile(filePath, []byte(testData), 0644); err != nil {
			t.Fatal(err)
		}

		cm := &StorageManager{
			codebasePath:    tempDir,
			codebaseConfigs: make(map[string]*CodebaseConfig),
			logger:          logger,
			mutex:           sync.RWMutex{},
		}

		// 模拟调用
		config, err := cm.loadCodebaseConfig("mismatch")

		// 验证
		assert.Nil(t, config)
		assert.ErrorContains(t, err, "codebase目录中的coebase文件ID不匹配")
		// logger.AssertCalled(t, "Info", "加载codebase文件内容: %s", mock.Anything)
	})

	t.Run("成功加载配置", func(t *testing.T) {
		logger := &MockLogger{}
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "valid.json")
		testData := `{"codebaseId":"valid.json","lastSync":"2025-01-01T00:00:00Z"}`
		if err := os.WriteFile(filePath, []byte(testData), 0644); err != nil {
			t.Fatal(err)
		}

		cm := &StorageManager{
			codebasePath:    tempDir,
			codebaseConfigs: make(map[string]*CodebaseConfig),
			logger:          logger,
			mutex:           sync.RWMutex{},
		}

		// logger.On("Info", "成功加载codebase文件，上次同步时间: %s", mock.Anything).Return()

		// 模拟调用
		config, err := cm.loadCodebaseConfig("valid.json")

		// 验证
		assert.NoError(t, err)
		assert.NotNil(t, config)
		assert.Equal(t, "valid.json", config.CodebaseId)
		// logger.AssertCalled(t, "Info", "加载codebase文件内容: %s", mock.Anything)
		// logger.AssertCalled(t, "Info", "成功加载codebase文件，上次同步时间: %s", mock.Anything)
	})

	t.Run("并发读取安全", func(t *testing.T) {
		logger := &MockLogger{}
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "concurrent.json")
		testData := `{"codebaseId":"concurrent.json","lastSync":"2025-01-01T00:00:00Z"}`
		if err := os.WriteFile(filePath, []byte(testData), 0644); err != nil {
			t.Fatal(err)
		}

		cm := &StorageManager{
			codebasePath:    tempDir,
			codebaseConfigs: make(map[string]*CodebaseConfig),
			logger:          logger,
		}

		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := cm.loadCodebaseConfig("concurrent.json")
				assert.NoError(t, err)
			}()
		}
		wg.Wait()
	})
}

func TestSaveCodebaseConfig(t *testing.T) {
	logger := &MockLogger{}
	config := &CodebaseConfig{
		CodebaseId:   "test123",
		CodebasePath: "/test/path",
	}

	tempDir := t.TempDir()

	tests := []struct {
		name        string
		prepare     func() *StorageManager
		config      *CodebaseConfig
		wantErr     bool
		expectError string
	}{
		{
			name: "success save",
			prepare: func() *StorageManager {
				return &StorageManager{
					logger:          logger,
					codebasePath:    tempDir,
					codebaseConfigs: make(map[string]*CodebaseConfig),
					mutex:           sync.RWMutex{},
				}
			},
			config:  config,
			wantErr: false,
		},
		// {
		// 	name: "fail on json marshal",
		// 	prepare: func() *StorageManager {
		// 		return &StorageManager{
		// 			logger:          logger,
		// 			codebasePath:    tempDir,
		// 			codebaseConfigs: make(map[string]*CodebaseConfig),
		// 			mutex:           sync.RWMutex{},
		// 		}
		// 	},
		// 	config:      nil,
		// 	wantErr:     true,
		// 	expectError: "序列化配置失败",
		// },
		{
			name: "fail on write file",
			prepare: func() *StorageManager {
				return &StorageManager{
					logger:          logger,
					codebasePath:    "/invalid/path",
					codebaseConfigs: make(map[string]*CodebaseConfig),
					mutex:           sync.RWMutex{},
				}
			},
			config:      config,
			wantErr:     true,
			expectError: "写入配置文件失败",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := tt.prepare()
			err := cm.SaveCodebaseConfig(tt.config)

			if (err != nil) != tt.wantErr {
				t.Errorf("SaveCodebaseConfig() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && tt.expectError != "" && !strings.Contains(err.Error(), tt.expectError) {
				t.Errorf("SaveCodebaseConfig() error = %v, want contains %q", err, tt.expectError)
			}

			if !tt.wantErr {
				filePath := filepath.Join(tempDir, tt.config.CodebaseId)
				if _, err := os.Stat(filePath); os.IsNotExist(err) {
					t.Errorf("file not created: %v", filePath)
				}

				if cm.codebaseConfigs[tt.config.CodebaseId] == nil {
					t.Errorf("memory config not saved")
				}
			}
		})
	}
}

func TestDeleteCodebaseConfig(t *testing.T) {
	logger := &MockLogger{}
	// logger.On("Info", mock.Anything, mock.Anything).Return()

	t.Run("删除内存和文件中的配置", func(t *testing.T) {
		tempDir := t.TempDir()
		codebaseId := "test1"
		filePath := filepath.Join(tempDir, codebaseId)

		// 创建测试文件
		if err := os.WriteFile(filePath, []byte{}, 0644); err != nil {
			t.Fatal(err)
		}

		cm := &StorageManager{
			codebasePath: tempDir,
			codebaseConfigs: map[string]*CodebaseConfig{
				codebaseId: {},
			},
			logger: logger,
			mutex:  sync.RWMutex{},
		}
		// 创建测试JSON文件内容
		configData, _ := json.Marshal(&CodebaseConfig{
			CodebaseId: codebaseId,
			LastSync:   time.Now(),
		})
		os.WriteFile(filePath, configData, 0644)

		// 执行删除
		err := cm.DeleteCodebaseConfig(codebaseId)
		assert.NoError(t, err)

		// 验证文件已删除
		_, err = os.Stat(filePath)
		assert.True(t, os.IsNotExist(err))

		// 验证内存中的配置已删除
		assert.Nil(t, cm.codebaseConfigs[codebaseId])

		// 验证日志调用
		// logger.AssertCalled(t, "Info", "codebase配置已删除: %s (文件+内存)", mock.Anything)
	})

	t.Run("仅删除内存中的配置（当文件不存在时）", func(t *testing.T) {
		tempDir := t.TempDir()
		codebaseId := "test2"

		cm := &StorageManager{
			codebasePath: tempDir,
			codebaseConfigs: map[string]*CodebaseConfig{
				codebaseId: {},
			},
			logger: logger,
			mutex:  sync.RWMutex{},
		}

		// 执行删除
		err := cm.DeleteCodebaseConfig(codebaseId)
		assert.NoError(t, err)

		// 验证内存中的配置已删除
		assert.Nil(t, cm.codebaseConfigs[codebaseId])

		// 验证日志调用
		// logger.AssertCalled(t, "Info", "仅内存中的codebase配置已删除: %s", codebaseId)
	})

	t.Run("仅删除文件中的配置（当内存中不存在时）", func(t *testing.T) {
		tempDir := t.TempDir()
		codebaseId := "test3"
		filePath := filepath.Join(tempDir, codebaseId)

		// 创建测试文件
		if err := os.WriteFile(filePath, []byte{}, 0644); err != nil {
			t.Fatal(err)
		}

		cm := &StorageManager{
			codebasePath:    tempDir,
			codebaseConfigs: map[string]*CodebaseConfig{},
			logger:          logger,
			mutex:           sync.RWMutex{},
		}

		// 执行删除
		err := cm.DeleteCodebaseConfig(codebaseId)
		assert.NoError(t, err)

		// 验证文件已删除
		_, err = os.Stat(filePath)
		assert.True(t, os.IsNotExist(err))

		// 验证日志调用
		// logger.AssertCalled(t, "Info", "codebase文件已删除: %s (仅文件)", mock.Anything)
	})

	t.Run("删除不存在的配置", func(t *testing.T) {
		tempDir := t.TempDir()
		cm := &StorageManager{
			codebasePath:    tempDir,
			codebaseConfigs: map[string]*CodebaseConfig{},
			logger:          logger,
			mutex:           sync.RWMutex{},
		}

		// 执行删除
		err := cm.DeleteCodebaseConfig("nonexistent")
		assert.NoError(t, err)
	})

	t.Run("文件删除失败返回错误", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("跳过Windows下的文件权限测试")
		}

		tempDir := t.TempDir()
		codebaseId := "test4"
		filePath := filepath.Join(tempDir, codebaseId)

		// 创建不可删除的目录（模拟删除失败）
		if err := os.Mkdir(filePath, 0755); err != nil {
			t.Fatal(err)
		}

		cm := &StorageManager{
			codebasePath: tempDir,
			codebaseConfigs: map[string]*CodebaseConfig{
				codebaseId: {},
			},
			logger: logger,
			mutex:  sync.RWMutex{},
		}

		// 执行删除
		err := cm.DeleteCodebaseConfig(codebaseId)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "删除配置文件失败")

		// 验证内存配置未被删除
		assert.NotNil(t, cm.codebaseConfigs[codebaseId])
	})

	t.Run("并发删除安全", func(t *testing.T) {
		tempDir := t.TempDir()
		codebaseCount := 100
		filePaths := make([]string, codebaseCount)
		codebaseConfigs := make(map[string]*CodebaseConfig, codebaseCount)

		// 创建测试文件和内存配置
		for i := 0; i < codebaseCount; i++ {
			codebaseId := fmt.Sprintf("concurrent-%d", i)
			filePath := filepath.Join(tempDir, codebaseId)
			if err := os.WriteFile(filePath, []byte(`{"codebaseId": "`+codebaseId+`"}`), 0644); err != nil {
				t.Fatal(err)
			}
			filePaths[i] = filePath
			codebaseConfigs[codebaseId] = &CodebaseConfig{}
		}

		cm := &StorageManager{
			codebasePath:    tempDir,
			codebaseConfigs: codebaseConfigs,
			logger:          logger,
			mutex:           sync.RWMutex{},
		}

		var wg sync.WaitGroup
		for i := 0; i < codebaseCount; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				err := cm.DeleteCodebaseConfig(fmt.Sprintf("concurrent-%d", id))
				assert.NoError(t, err)
			}(i)
		}
		wg.Wait()

		// 验证所有文件已被删除
		for _, filePath := range filePaths {
			_, err := os.Stat(filePath)
			assert.True(t, os.IsNotExist(err))
		}

		// 验证内存中的配置已全部删除
		assert.Empty(t, cm.codebaseConfigs)
	})
}
