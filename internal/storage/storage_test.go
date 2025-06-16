package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockLogger implements a mock logger
type MockLogger struct {
	mock.Mock
}

func (m *MockLogger) Debug(format string, v ...interface{}) {
	fmt.Printf("[MOCK DEBUG] %s\n", fmt.Sprintf(format, v...)) // 输出日志
	m.Called(format, v)
}

func (m *MockLogger) Info(format string, v ...interface{}) {
	fmt.Printf("[MOCK INFO] %s\n", fmt.Sprintf(format, v...)) // 输出日志
	m.Called(format, v)
}

func (m *MockLogger) Warn(format string, v ...interface{}) {
	fmt.Printf("[MOCK WARN] %s\n", fmt.Sprintf(format, v...)) // 输出日志
	m.Called(format, v)
}

func (m *MockLogger) Error(format string, v ...interface{}) {
	fmt.Printf("[MOCK ERROR] %s\n", fmt.Sprintf(format, v...)) // 输出日志
	m.Called(format, v)
}

func (m *MockLogger) Fatal(format string, v ...interface{}) {
	fmt.Printf("[MOCK FATAL] %s\n", fmt.Sprintf(format, v...)) // 输出日志
	m.Called(format, v)
}

func TestNewStorageManager(t *testing.T) {
	logger := &MockLogger{}

	t.Run("create new directory", func(t *testing.T) {
		// 设置临时目录
		tempDir := t.TempDir()
		codebasePath := filepath.Join(tempDir, "codebase")

		// 确保目录不存在
		if _, err := os.Stat(codebasePath); !os.IsNotExist(err) {
			t.Fatalf("test directory should not exist: %v", err)
		}

		sm, err := NewStorageManager(tempDir, logger)
		assert.NoError(t, err)
		require.NotNil(t, sm)

		// 验证目录是否创建
		if _, statErr := os.Stat(codebasePath); os.IsNotExist(statErr) {
			t.Fatalf("codebase directory should be created: %v", statErr)
		}
	})

	t.Run("directory exists", func(t *testing.T) {
		// 预先创建目录
		tempDir := t.TempDir()
		codebasePath := filepath.Join(tempDir, "codebase")
		if err := os.Mkdir(codebasePath, 0755); err != nil {
			t.Fatalf("failed to create test directory: %v", err)
		}

		sm, err := NewStorageManager(tempDir, logger)
		// 验证没有错误
		assert.NoError(t, err)
		require.NotNil(t, sm)
	})

	t.Run("directory creation failed", func(t *testing.T) {
		// 创建一个临时根目录用于测试
		rootDir := t.TempDir()
		// 将 cacheDir 设置为一个文件的路径，而不是目录
		fileAsCacheDirPath := filepath.Join(rootDir, "thisIsAFileNotADirectory")
		if err := os.WriteFile(fileAsCacheDirPath, []byte("I am a file"), 0644); err != nil {
			t.Fatalf("failed to create file as cacheDir: %v", err)
		}

		sm, err := NewStorageManager(fileAsCacheDirPath, logger)

		// 验证返回了错误，并且 sm 为 nil
		assert.Error(t, err)
		if err != nil { // 确保 err 不是 nil 才调用 err.Error()
			assert.Contains(t, err.Error(), "failed to create codebase directory")
		}
		assert.Nil(t, sm)
	})
}

func TestGetCodebaseConfigs(t *testing.T) {
	t.Run("empty config", func(t *testing.T) {
		cm := &StorageManager{
			codebaseConfigs: make(map[string]*CodebaseConfig),
		}

		configs := cm.GetCodebaseConfigs()
		assert.Empty(t, configs)
		assert.NotNil(t, configs) // 确保不是 nil
	})

	t.Run("with config data", func(t *testing.T) {
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

	t.Run("returns reference", func(t *testing.T) {
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
	logger.On("Info", mock.Anything, mock.Anything).Return()

	t.Run("get existing config from memory", func(t *testing.T) {
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

	t.Run("load new config from file", func(t *testing.T) {
		tempDir := t.TempDir()
		file := "test2"
		if err := os.WriteFile(filepath.Join(tempDir, file), []byte(`{"codebaseId": "`+file+`"}`), 0644); err != nil {
			t.Fatalf("failed to create test file %s: %v", file, err)
		}
		cm := &StorageManager{
			codebasePath:    tempDir,
			codebaseConfigs: make(map[string]*CodebaseConfig),
			logger:          logger,
			mutex:           sync.RWMutex{},
		}

		expectedConfig := &CodebaseConfig{CodebaseId: file}

		config, err := cm.GetCodebaseConfig(file)
		assert.NoError(t, err)
		assert.Equal(t, expectedConfig, config)
		assert.Equal(t, expectedConfig, cm.codebaseConfigs[file])

		logger.AssertCalled(t, "Info", "loading codebase file content: %s", mock.Anything)
		logger.AssertCalled(t, "Info", "codebase file loaded successfully, last sync time: %s", mock.Anything)
	})

	t.Run("returns error when config not exists", func(t *testing.T) {
		tempDir := t.TempDir()
		cm := &StorageManager{
			codebasePath:    tempDir,
			codebaseConfigs: make(map[string]*CodebaseConfig),
			logger:          logger,
			mutex:           sync.RWMutex{},
		}

		config, err := cm.GetCodebaseConfig("test3")
		assert.ErrorContains(t, err, "codebase file does not exist")
		assert.Nil(t, config)
		assert.Empty(t, cm.codebaseConfigs)

		logger.AssertCalled(t, "Info", "loading codebase file content: %s", mock.Anything)
	})

	t.Run("concurrent access safe", func(t *testing.T) {
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
				t.Fatalf("failed to create test file %s: %v", file, err)
			}
		}

		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				_, err := cm.GetCodebaseConfig(fmt.Sprintf("test%d", id))
				assert.NoError(t, err)

				logger.AssertCalled(t, "Info", "loading codebase file content: %s", mock.Anything)
				logger.AssertCalled(t, "Info", "codebase file loaded successfully, last sync time: %s", mock.Anything)
			}(i)
		}
		wg.Wait()
	})
}

func TestConfigManager_loadAllConfigs(t *testing.T) {
	logger := &MockLogger{}
	logger.On("Info", mock.Anything, mock.Anything).Return()
	logger.On("Error", mock.Anything, mock.Anything).Return()

	t.Run("directory read failed", func(t *testing.T) {
		// 无权限目录
		cm := &StorageManager{
			codebasePath:    "/root", // Linux下无权限的目录
			codebaseConfigs: make(map[string]*CodebaseConfig),
			logger:          logger,
			mutex:           sync.RWMutex{},
		}

		// 执行
		cm.loadAllConfigs()

		logger.AssertCalled(t, "Error", "failed to read codebase directory: %v", mock.Anything)
	})

	t.Run("no files in directory", func(t *testing.T) {
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

	t.Run("with subdirectories", func(t *testing.T) {
		tempDir := t.TempDir()
		// 创建子目录
		if err := os.Mkdir(filepath.Join(tempDir, "subdir"), 0755); err != nil {
			t.Fatalf("failed to create test subdirectory: %v", err)
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

	t.Run("load config files successfully", func(t *testing.T) {
		tempDir := t.TempDir()
		// 创建测试文件
		testFiles := []string{"config1", "config2"}
		for _, f := range testFiles {
			if err := os.WriteFile(filepath.Join(tempDir, f), []byte(`{"codebaseId": "`+f+`"}`), 0644); err != nil {
				t.Fatalf("failed to create test file %s: %v", f, err)
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

		logger.AssertCalled(t, "Info", "loading codebase file content: %s", mock.Anything)
		logger.AssertCalled(t, "Info", "codebase file loaded successfully, last sync time: %s", mock.Anything)
	})

	t.Run("partial files load failed", func(t *testing.T) {
		tempDir := t.TempDir()
		// 创建测试文件
		testFiles := []string{"good", "bad"}
		for _, f := range testFiles {
			if strings.HasSuffix(f, "bad") {
				if err := os.WriteFile(filepath.Join(tempDir, f), []byte("text"), 0644); err != nil {
					t.Fatalf("failed to create test file %s: %v", f, err)
				}
				continue
			}
			if err := os.WriteFile(filepath.Join(tempDir, f), []byte(`{"codebaseId": "good"}`), 0644); err != nil {
				t.Fatalf("failed to create test file %s: %v", f, err)
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

		logger.AssertCalled(t, "Info", "loading codebase file content: %s", mock.Anything)
		logger.AssertCalled(t, "Info", "codebase file loaded successfully, last sync time: %s", mock.Anything)
		logger.AssertCalled(t, "Error", "failed to load codebase file %s: %v", mock.Anything, mock.Anything)
	})
}

func TestConfigManager_loadCodebaseConfig(t *testing.T) {
	logger := &MockLogger{}
	logger.On("Info", mock.Anything, mock.Anything).Return()

	t.Run("file not exists", func(t *testing.T) {
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
		assert.ErrorContains(t, err, "codebase file does not exist")

		logger.AssertCalled(t, "Info", "loading codebase file content: %s", mock.Anything)
	})

	t.Run("JSON parse failed", func(t *testing.T) {
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
		assert.ErrorContains(t, err, "failed to parse codebase file")

		logger.AssertCalled(t, "Info", "loading codebase file content: %s", mock.Anything)
	})

	t.Run("CodebaseId mismatch", func(t *testing.T) {
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
		assert.ErrorContains(t, err, "codebaseId mismatch")

		logger.AssertCalled(t, "Info", "loading codebase file content: %s", mock.Anything)
	})

	t.Run("load config successfully", func(t *testing.T) {
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

		// 模拟调用
		config, err := cm.loadCodebaseConfig("valid.json")

		// 验证
		assert.NoError(t, err)
		assert.NotNil(t, config)
		assert.Equal(t, "valid.json", config.CodebaseId)

		logger.AssertCalled(t, "Info", "loading codebase file content: %s", mock.Anything)
		logger.AssertCalled(t, "Info", "codebase file loaded successfully, last sync time: %s", mock.Anything)
	})

	t.Run("concurrent read safe", func(t *testing.T) {
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

				logger.AssertCalled(t, "Info", "loading codebase file content: %s", mock.Anything)
				logger.AssertCalled(t, "Info", "codebase file loaded successfully, last sync time: %s", mock.Anything)
			}()
		}
		wg.Wait()
	})
}

func TestSaveCodebaseConfig(t *testing.T) {
	logger := &MockLogger{}
	logger.On("Info", mock.Anything, mock.Anything).Return()

	config := &CodebaseConfig{
		CodebaseId:   "test123",
		CodebasePath: "/test/path",
	}

	tempDir := t.TempDir()
	invalidPath := filepath.Join(tempDir, "invalid", "path")

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
		{
			name: "fail on write file",
			prepare: func() *StorageManager {
				return &StorageManager{
					logger:          logger,
					codebasePath:    invalidPath,
					codebaseConfigs: make(map[string]*CodebaseConfig),
					mutex:           sync.RWMutex{},
				}
			},
			config:      config,
			wantErr:     true,
			expectError: "failed to write config file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := tt.prepare()
			if cm.codebasePath == invalidPath {
				// 创建测试目录，确保创建同名文件时报错
				err := os.MkdirAll(filepath.Join(cm.codebasePath, tt.config.CodebaseId), 0755)
				assert.NoError(t, err)
				defer os.RemoveAll(filepath.Join(cm.codebasePath, tt.config.CodebaseId))
			}
			err := cm.SaveCodebaseConfig(tt.config)
			logger.AssertCalled(t, "Info", "saving codebase config: %s", mock.Anything)

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

				logger.AssertCalled(t, "Info", "codebase config saved successfully, path: %s, codebaseId: %s", mock.Anything, mock.Anything)
			}
		})
	}
}

func TestDeleteCodebaseConfig(t *testing.T) {
	logger := &MockLogger{}
	logger.On("Info", mock.Anything, mock.Anything).Return()

	t.Run("delete config from both memory and file", func(t *testing.T) {
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

		logger.AssertCalled(t, "Info", "codebase config deleted: %s (file and memory)", mock.Anything)
	})

	t.Run("delete from memory only (when file not exists)", func(t *testing.T) {
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

		logger.AssertCalled(t, "Info", "codebase config deleted: %s (memory only)", mock.Anything)
	})

	t.Run("delete from file only (when not in memory)", func(t *testing.T) {
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

		logger.AssertCalled(t, "Info", "codebase file deleted: %s (file only)", mock.Anything)
	})

	t.Run("delete non-existent config", func(t *testing.T) {
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

	t.Run("file deletion failed returns error", func(t *testing.T) {
		tempDir := t.TempDir()
		codebaseId := "test4"
		filePath := filepath.Join(tempDir, codebaseId)

		// 创建并保持打开的文件（模拟删除失败）
		file, err := os.Create(filePath)
		if err != nil {
			t.Fatal("failed to create test file:", err)
		}
		defer func() {
			file.Close()
			os.Remove(filePath)
		}()

		cm := &StorageManager{
			codebasePath: tempDir,
			codebaseConfigs: map[string]*CodebaseConfig{
				codebaseId: {},
			},
			logger: logger,
			mutex:  sync.RWMutex{},
		}

		// 执行删除
		err = cm.DeleteCodebaseConfig(codebaseId)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete codebase file")

		// 验证内存配置未被删除
		assert.NotNil(t, cm.codebaseConfigs[codebaseId])
	})

	t.Run("concurrent deletion safe", func(t *testing.T) {
		tempDir := t.TempDir()
		codebaseCount := 100
		filePaths := make([]string, codebaseCount)
		codebaseConfigs := make(map[string]*CodebaseConfig, codebaseCount)

		// 创建测试文件和内存配置
		for i := 0; i < codebaseCount; i++ {
			codebaseId := fmt.Sprintf("concurrent-%d", i)
			filePath := filepath.Join(tempDir, codebaseId)
			if err := os.WriteFile(filePath, []byte(`{"codebaseId": "`+codebaseId+`"}`), 0644); err != nil {
				t.Fatal("failed to create test file:", err)
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
