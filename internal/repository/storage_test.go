package repository

import (
	"codebase-indexer/internal/config"
	"codebase-indexer/test/mocks"
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

func TestNewStorageManager(t *testing.T) {
	logger := &mocks.MockLogger{}

	t.Run("create new directory", func(t *testing.T) {
		// Set up temp directory
		tempDir := t.TempDir()
		codebasePath := filepath.Join(tempDir, "codebase")

		// Make sure directory doesn't exist
		if _, err := os.Stat(codebasePath); !os.IsNotExist(err) {
			t.Fatalf("test directory should not exist: %v", err)
		}

		sm, err := NewStorageManager(tempDir, logger)
		assert.NoError(t, err)
		require.NotNil(t, sm)

		// Verify directory was created
		if _, statErr := os.Stat(codebasePath); os.IsNotExist(statErr) {
			t.Fatalf("codebase directory should be created: %v", statErr)
		}
	})

	t.Run("directory exists", func(t *testing.T) {
		// Pre-create directory
		tempDir := t.TempDir()
		codebasePath := filepath.Join(tempDir, "codebase")
		if err := os.Mkdir(codebasePath, 0755); err != nil {
			t.Fatalf("failed to create test directory: %v", err)
		}

		sm, err := NewStorageManager(tempDir, logger)
		// Verify no error
		assert.NoError(t, err)
		require.NotNil(t, sm)
	})

	t.Run("directory creation failed", func(t *testing.T) {
		// Create temp root dir for testing
		rootDir := t.TempDir()
		// Set cacheDir to a file path instead of directory
		fileAsCacheDirPath := filepath.Join(rootDir, "thisIsAFileNotADirectory")
		if err := os.WriteFile(fileAsCacheDirPath, []byte("I am a file"), 0644); err != nil {
			t.Fatalf("failed to create file as cacheDir: %v", err)
		}

		sm, err := NewStorageManager(fileAsCacheDirPath, logger)

		// Verify error returned and sm is nil
		assert.Error(t, err)
		if err != nil { // Ensure err is not nil before calling err.Error()
			assert.Contains(t, err.Error(), "failed to create codebase directory")
		}
		assert.Nil(t, sm)
	})
}

func TestGetCodebaseConfigs(t *testing.T) {
	t.Run("empty config", func(t *testing.T) {
		cm := &StorageManager{
			codebaseConfigs: make(map[string]*config.CodebaseConfig),
		}

		configs := cm.GetCodebaseConfigs()
		assert.Empty(t, configs)
		assert.NotNil(t, configs) // Ensure not nil
	})

	t.Run("with config data", func(t *testing.T) {
		cm := &StorageManager{
			codebaseConfigs: map[string]*config.CodebaseConfig{
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
			codebaseConfigs: make(map[string]*config.CodebaseConfig),
		}

		configs := cm.GetCodebaseConfigs()
		configs["test"] = &config.CodebaseConfig{} // Modify the returned map

		// Verify if modification affected original data
		assert.NotEmpty(t, cm.codebaseConfigs)
		assert.Equal(t, cm.codebaseConfigs, configs)
	})
}

func TestGetCodebaseConfig(t *testing.T) {
	logger := &mocks.MockLogger{}
	logger.On("Info", mock.Anything, mock.Anything).Return()

	t.Run("get existing config from memory", func(t *testing.T) {
		configs := map[string]*config.CodebaseConfig{
			"test1": {CodebaseId: "test1"},
		}
		cm := &StorageManager{
			codebaseConfigs: configs,
			logger:          logger,
			rwMutex:         sync.RWMutex{},
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
			codebaseConfigs: make(map[string]*config.CodebaseConfig),
			logger:          logger,
			rwMutex:         sync.RWMutex{},
		}

		expectedConfig := &config.CodebaseConfig{CodebaseId: file}

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
			codebaseConfigs: make(map[string]*config.CodebaseConfig),
			logger:          logger,
			rwMutex:         sync.RWMutex{},
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
			codebaseConfigs: make(map[string]*config.CodebaseConfig),
			logger:          logger,
			rwMutex:         sync.RWMutex{},
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
	logger := &mocks.MockLogger{}
	logger.On("Info", mock.Anything, mock.Anything).Return()
	logger.On("Error", mock.Anything, mock.Anything).Return()

	t.Run("directory read failed", func(t *testing.T) {
		// No-permission directory
		cm := &StorageManager{
			codebasePath:    "/root", // No-permission dir on Linux
			codebaseConfigs: make(map[string]*config.CodebaseConfig),
			logger:          logger,
			rwMutex:         sync.RWMutex{},
		}

		// Execute
		cm.loadAllConfigs()

		logger.AssertCalled(t, "Error", "failed to read codebase directory: %v", mock.Anything)
	})

	t.Run("no files in directory", func(t *testing.T) {
		tempDir := t.TempDir()
		cm := &StorageManager{
			codebasePath:    tempDir,
			codebaseConfigs: make(map[string]*config.CodebaseConfig),
			logger:          logger,
			rwMutex:         sync.RWMutex{},
		}

		// Execute
		cm.loadAllConfigs()

		// Verify
		assert.Empty(t, cm.codebaseConfigs)
	})

	t.Run("with subdirectories", func(t *testing.T) {
		tempDir := t.TempDir()
		// Create subdirectory
		if err := os.Mkdir(filepath.Join(tempDir, "subdir"), 0755); err != nil {
			t.Fatalf("failed to create test subdirectory: %v", err)
		}

		cm := &StorageManager{
			codebasePath:    tempDir,
			codebaseConfigs: make(map[string]*config.CodebaseConfig),
			logger:          logger,
			rwMutex:         sync.RWMutex{},
		}

		// Execute
		cm.loadAllConfigs()

		// Verify
		assert.Empty(t, cm.codebaseConfigs)
	})

	t.Run("load config files successfully", func(t *testing.T) {
		tempDir := t.TempDir()
		// Create test files
		testFiles := []string{"config1", "config2"}
		for _, f := range testFiles {
			if err := os.WriteFile(filepath.Join(tempDir, f), []byte(`{"codebaseId": "`+f+`"}`), 0644); err != nil {
				t.Fatalf("failed to create test file %s: %v", f, err)
			}
		}

		cm := &StorageManager{
			codebasePath:    tempDir,
			codebaseConfigs: make(map[string]*config.CodebaseConfig),
			logger:          logger,
			rwMutex:         sync.RWMutex{},
		}

		// Execute
		cm.loadAllConfigs()

		// Verify
		assert.Equal(t, len(testFiles), len(cm.codebaseConfigs))

		logger.AssertCalled(t, "Info", "loading codebase file content: %s", mock.Anything)
		logger.AssertCalled(t, "Info", "codebase file loaded successfully, last sync time: %s", mock.Anything)
	})

	t.Run("partial files load failed", func(t *testing.T) {
		tempDir := t.TempDir()
		// Create test files
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
			codebaseConfigs: make(map[string]*config.CodebaseConfig),
			logger:          logger,
			rwMutex:         sync.RWMutex{},
		}

		// Execute
		cm.loadAllConfigs()

		// Verify
		assert.Equal(t, 1, len(cm.codebaseConfigs))

		logger.AssertCalled(t, "Info", "loading codebase file content: %s", mock.Anything)
		logger.AssertCalled(t, "Info", "codebase file loaded successfully, last sync time: %s", mock.Anything)
		logger.AssertCalled(t, "Error", "failed to load codebase file %s: %v", mock.Anything, mock.Anything)
	})
}

func TestConfigManager_loadCodebaseConfig(t *testing.T) {
	logger := &mocks.MockLogger{}
	logger.On("Info", mock.Anything, mock.Anything).Return()

	t.Run("file not exists", func(t *testing.T) {
		tempDir := t.TempDir()
		cm := &StorageManager{
			codebasePath:    tempDir,
			codebaseConfigs: make(map[string]*config.CodebaseConfig),
			logger:          logger,
			rwMutex:         sync.RWMutex{},
		}

		// Mock call
		config, err := cm.loadCodebaseConfig("nonexistent.json")

		// Verify
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
			codebaseConfigs: make(map[string]*config.CodebaseConfig),
			logger:          logger,
		}

		// Mock call
		config, err := cm.loadCodebaseConfig("invalid")

		// Verify
		assert.Nil(t, config)
		assert.ErrorContains(t, err, "failed to parse codebase file")

		logger.AssertCalled(t, "Info", "loading codebase file content: %s", mock.Anything)
	})

	t.Run("ClientId mismatch", func(t *testing.T) {
		tempDir := t.TempDir()
		filePath := filepath.Join(tempDir, "mismatch")
		testData := `{"codebaseId":"other-id","lastSync":"2025-01-01T00:00:00Z"}`
		if err := os.WriteFile(filePath, []byte(testData), 0644); err != nil {
			t.Fatal(err)
		}

		cm := &StorageManager{
			codebasePath:    tempDir,
			codebaseConfigs: make(map[string]*config.CodebaseConfig),
			logger:          logger,
			rwMutex:         sync.RWMutex{},
		}

		// Mock call
		config, err := cm.loadCodebaseConfig("mismatch")

		// Verify
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
			codebaseConfigs: make(map[string]*config.CodebaseConfig),
			logger:          logger,
			rwMutex:         sync.RWMutex{},
		}

		// Mock call
		config, err := cm.loadCodebaseConfig("valid.json")

		// Verify
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
			codebaseConfigs: make(map[string]*config.CodebaseConfig),
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
	logger := &mocks.MockLogger{}
	logger.On("Info", mock.Anything, mock.Anything).Return()

	codebaseConfig := &config.CodebaseConfig{
		CodebaseId:   "test123",
		CodebasePath: "/test/path",
	}

	tempDir := t.TempDir()
	invalidPath := filepath.Join(tempDir, "invalid", "path")

	tests := []struct {
		name        string
		prepare     func() *StorageManager
		config      *config.CodebaseConfig
		wantErr     bool
		expectError string
	}{
		{
			name: "success save",
			prepare: func() *StorageManager {
				return &StorageManager{
					logger:          logger,
					codebasePath:    tempDir,
					codebaseConfigs: make(map[string]*config.CodebaseConfig),
					rwMutex:         sync.RWMutex{},
				}
			},
			config:  codebaseConfig,
			wantErr: false,
		},
		{
			name: "fail on write file",
			prepare: func() *StorageManager {
				return &StorageManager{
					logger:          logger,
					codebasePath:    invalidPath,
					codebaseConfigs: make(map[string]*config.CodebaseConfig),
					rwMutex:         sync.RWMutex{},
				}
			},
			config:      codebaseConfig,
			wantErr:     true,
			expectError: "failed to write config file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := tt.prepare()
			if cm.codebasePath == invalidPath {
				// Create test directory to trigger file creation error
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

				logger.AssertCalled(t, "Info", "codebase config saved successfully, path: %s", mock.Anything)
			}
		})
	}
}

func TestDeleteCodebaseConfig(t *testing.T) {
	logger := &mocks.MockLogger{}
	logger.On("Info", mock.Anything, mock.Anything).Return()

	t.Run("delete config from both memory and file", func(t *testing.T) {
		tempDir := t.TempDir()
		codebaseId := "test1"
		filePath := filepath.Join(tempDir, codebaseId)

		// Create test file
		if err := os.WriteFile(filePath, []byte{}, 0644); err != nil {
			t.Fatal(err)
		}

		cm := &StorageManager{
			codebasePath: tempDir,
			codebaseConfigs: map[string]*config.CodebaseConfig{
				codebaseId: {},
			},
			logger:  logger,
			rwMutex: sync.RWMutex{},
		}
		// Create test JSON file content
		configData, _ := json.Marshal(&config.CodebaseConfig{
			CodebaseId: codebaseId,
			LastSync:   time.Now(),
		})
		os.WriteFile(filePath, configData, 0644)

		// Execute deletion
		err := cm.DeleteCodebaseConfig(codebaseId)
		assert.NoError(t, err)

		// Verify file was deleted
		_, err = os.Stat(filePath)
		assert.True(t, os.IsNotExist(err))

		// Verify in-memory config was deleted
		assert.Nil(t, cm.codebaseConfigs[codebaseId])

		logger.AssertCalled(t, "Info", "codebase config deleted: %s (file and memory)", mock.Anything)
	})

	t.Run("delete from memory only (when file not exists)", func(t *testing.T) {
		tempDir := t.TempDir()
		codebaseId := "test2"

		cm := &StorageManager{
			codebasePath: tempDir,
			codebaseConfigs: map[string]*config.CodebaseConfig{
				codebaseId: {},
			},
			logger:  logger,
			rwMutex: sync.RWMutex{},
		}

		// Execute deletion
		err := cm.DeleteCodebaseConfig(codebaseId)
		assert.NoError(t, err)

		// Verify in-memory config was deleted
		assert.Nil(t, cm.codebaseConfigs[codebaseId])

		logger.AssertCalled(t, "Info", "codebase config deleted: %s (memory only)", mock.Anything)
	})

	t.Run("delete from file only (when not in memory)", func(t *testing.T) {
		tempDir := t.TempDir()
		codebaseId := "test3"
		filePath := filepath.Join(tempDir, codebaseId)

		// Create test file
		if err := os.WriteFile(filePath, []byte{}, 0644); err != nil {
			t.Fatal(err)
		}

		cm := &StorageManager{
			codebasePath:    tempDir,
			codebaseConfigs: map[string]*config.CodebaseConfig{},
			logger:          logger,
			rwMutex:         sync.RWMutex{},
		}

		// Execute deletion
		err := cm.DeleteCodebaseConfig(codebaseId)
		assert.NoError(t, err)

		// Verify file was deleted
		_, err = os.Stat(filePath)
		assert.True(t, os.IsNotExist(err))

		logger.AssertCalled(t, "Info", "codebase file deleted: %s (file only)", mock.Anything)
	})

	t.Run("delete non-existent config", func(t *testing.T) {
		tempDir := t.TempDir()
		cm := &StorageManager{
			codebasePath:    tempDir,
			codebaseConfigs: map[string]*config.CodebaseConfig{},
			logger:          logger,
			rwMutex:         sync.RWMutex{},
		}

		// Execute deletion
		err := cm.DeleteCodebaseConfig("nonexistent")
		assert.NoError(t, err)
	})

	t.Run("file deletion failed returns error", func(t *testing.T) {
		tempDir := t.TempDir()
		codebaseId := "test4"
		filePath := filepath.Join(tempDir, codebaseId)

		// Create and keep file open (simulates deletion failure)
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
			codebaseConfigs: map[string]*config.CodebaseConfig{
				codebaseId: {},
			},
			logger:  logger,
			rwMutex: sync.RWMutex{},
		}

		// Execute deletion
		err = cm.DeleteCodebaseConfig(codebaseId)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete codebase file")

		// Verify in-memory config was not deleted
		assert.NotNil(t, cm.codebaseConfigs[codebaseId])
	})

	t.Run("concurrent deletion safe", func(t *testing.T) {
		tempDir := t.TempDir()
		codebaseCount := 100
		filePaths := make([]string, codebaseCount)
		codebaseConfigs := make(map[string]*config.CodebaseConfig, codebaseCount)

		// Create test files and in-memory configs
		for i := 0; i < codebaseCount; i++ {
			codebaseId := fmt.Sprintf("concurrent-%d", i)
			filePath := filepath.Join(tempDir, codebaseId)
			if err := os.WriteFile(filePath, []byte(`{"codebaseId": "`+codebaseId+`"}`), 0644); err != nil {
				t.Fatal("failed to create test file:", err)
			}
			filePaths[i] = filePath
			codebaseConfigs[codebaseId] = &config.CodebaseConfig{}
		}

		cm := &StorageManager{
			codebasePath:    tempDir,
			codebaseConfigs: codebaseConfigs,
			logger:          logger,
			rwMutex:         sync.RWMutex{},
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

		// Verify all files were deleted
		for _, filePath := range filePaths {
			_, err := os.Stat(filePath)
			assert.True(t, os.IsNotExist(err))
		}

		// Verify all in-memory configs were deleted
		assert.Empty(t, cm.codebaseConfigs)
	})
}
