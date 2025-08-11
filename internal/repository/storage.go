// storage/storage.go - Configuration and temporary file storage
package repository

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"codebase-indexer/internal/config"
	"codebase-indexer/internal/dto"
	"codebase-indexer/internal/utils"
	"codebase-indexer/pkg/logger"
)

type StorageInterface interface {
	GetCodebaseConfigs() map[string]*config.CodebaseConfig
	GetCodebaseConfig(codebaseId string) (*config.CodebaseConfig, error)
	SaveCodebaseConfig(config *config.CodebaseConfig) error
	DeleteCodebaseConfig(codebaseId string) error
	GetCodebaseEnv() *config.CodebaseEnv
	SaveCodebaseEnv(codebaseEnv *config.CodebaseEnv) error
}

type StorageManager struct {
	codebasePath    string
	codebaseConfigs map[string]*config.CodebaseConfig // Stores all codebase configurations
	codebaseEnvPath string
	codebaseEnv     *config.CodebaseEnv
	logger          logger.Logger
	rwMutex         sync.RWMutex
}

// NewStorageManager creates a new configuration manager
func NewStorageManager(workspaceDir string, logger logger.Logger) (StorageInterface, error) {
	if workspaceDir == "" || strings.Contains(workspaceDir, "\x00") {
		return nil, fmt.Errorf("invalid codebase directory path")
	}

	// Try to create directory to verify write permission
	if err := os.MkdirAll(workspaceDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create codebase directory: %v", err)
	}

	// Initialize codebaseConfigs map
	sm := &StorageManager{
		codebasePath:    workspaceDir,
		codebaseEnvPath: utils.EnvFile,
		logger:          logger,
		codebaseConfigs: make(map[string]*config.CodebaseConfig),
	}

	sm.loadAllConfigs()
	sm.loadCodebaseEnv()
	return sm, nil
}

// GetCodebaseConfigs retrieves all project configurations
func (s *StorageManager) GetCodebaseConfigs() map[string]*config.CodebaseConfig {
	s.rwMutex.RLock()
	defer s.rwMutex.RUnlock()
	return s.codebaseConfigs
}

// GetCodebaseConfig loads codebase configuration
// First checks in memory, if not found then loads from filesystem
func (s *StorageManager) GetCodebaseConfig(codebaseId string) (*config.CodebaseConfig, error) {
	s.rwMutex.RLock()
	config, exists := s.codebaseConfigs[codebaseId]
	s.rwMutex.RUnlock()

	if exists {
		return config, nil
	}

	// Not found in memory, try loading from file
	config, err := s.loadCodebaseConfig(codebaseId)
	if err != nil {
		return nil, err
	}

	s.rwMutex.Lock()
	s.codebaseConfigs[codebaseId] = config
	s.rwMutex.Unlock()

	return config, nil
}

func (s *StorageManager) GetCodebaseEnv() *config.CodebaseEnv {
	s.rwMutex.RLock()
	defer s.rwMutex.RUnlock()

	return s.codebaseEnv
}

func (s *StorageManager) SaveCodebaseEnv(codebaseEnv *config.CodebaseEnv) error {
	if codebaseEnv == nil {
		return fmt.Errorf("codebase env is empty: %v", codebaseEnv)
	}
	s.logger.Info("saving codebase env: %s", codebaseEnv)

	s.rwMutex.Lock()
	defer s.rwMutex.Unlock()

	data, err := json.MarshalIndent(codebaseEnv, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize codebase env: %v", err)
	}

	if err := os.WriteFile(s.codebaseEnvPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write codebase env file: %v", err)
	}

	// Atomically update in-memory configuration
	s.codebaseEnv = codebaseEnv
	s.logger.Info("codebase env saved successfully, path: %s", s.codebaseEnvPath)
	return nil
}

func (s *StorageManager) loadCodebaseEnv() {
	data, err := os.ReadFile(s.codebaseEnvPath)
	if err != nil {
		if os.IsNotExist(err) {
			// 文件不存在，创建默认配置
			s.logger.Info("codebase env file not found, creating with default values")
			s.codebaseEnv = &config.CodebaseEnv{
				Switch: dto.SwitchOn, // 默认开启索引功能
			}

			// 将默认配置写入文件
			defaultData, err := json.MarshalIndent(s.codebaseEnv, "", "  ")
			if err != nil {
				s.logger.Error("failed to marshal default codebase env: %v", err)
				return
			}

			if err := os.WriteFile(s.codebaseEnvPath, defaultData, 0644); err != nil {
				s.logger.Error("failed to create default codebase env file: %v", err)
				return
			}

			s.logger.Info("default codebase env file created successfully")
			return
		}
		s.logger.Error("failed to read codebase env file: %v", err)
		return
	}
	if err := json.Unmarshal(data, &s.codebaseEnv); err != nil {
		s.logger.Error("failed to parse codebase env file: %v", err)
		return
	}
}

// Load all codebase configuration files
func (s *StorageManager) loadAllConfigs() {
	files, err := os.ReadDir(s.codebasePath)
	if err != nil {
		s.logger.Error("failed to read codebase directory: %v", err)
		return
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		config, err := s.loadCodebaseConfig(file.Name())
		if err != nil {
			s.logger.Error("failed to load codebase file %s: %v", file.Name(), err)
			continue
		}
		s.codebaseConfigs[file.Name()] = config
	}
}

// loadCodebaseConfig loads a codebase configuration file
func (s *StorageManager) loadCodebaseConfig(codebaseId string) (*config.CodebaseConfig, error) {
	s.logger.Info("loading codebase file content: %s", codebaseId)

	s.rwMutex.RLock()
	defer s.rwMutex.RUnlock()

	filePath := filepath.Join(s.codebasePath, codebaseId)

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("codebase file does not exist: %s", filePath)
		}
		return nil, fmt.Errorf("failed to read codebase file: %v", err)
	}

	var config config.CodebaseConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse codebase file: %v", err)
	}

	if config.CodebaseId != codebaseId {
		return nil, fmt.Errorf("codebaseId mismatch: expected %s, got %s",
			codebaseId, config.CodebaseId)
	}

	s.logger.Info("codebase file loaded successfully, last sync time: %s",
		config.LastSync.Format(time.RFC3339))

	return &config, nil
}

// SaveCodebaseConfig saves codebase configuration
func (s *StorageManager) SaveCodebaseConfig(config *config.CodebaseConfig) error {
	if config == nil {
		return fmt.Errorf("codebase config is empty: %v", config)
	}
	s.logger.Info("saving codebase config: %s", config.CodebasePath)

	s.rwMutex.Lock()
	defer s.rwMutex.Unlock()

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize config: %v", err)
	}

	filePath := filepath.Join(s.codebasePath, config.CodebaseId)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	// Atomically update in-memory configuration
	s.codebaseConfigs[config.CodebaseId] = config
	s.logger.Info("codebase config saved successfully, path: %s", filePath)
	return nil
}

// DeleteCodebaseConfig deletes codebase configuration
func (s *StorageManager) DeleteCodebaseConfig(codebaseId string) error {
	s.logger.Info("deleting codebase config: %s", codebaseId)

	filePath := filepath.Join(s.codebasePath, codebaseId)

	s.rwMutex.Lock()
	defer s.rwMutex.Unlock()

	exists := s.codebaseConfigs[codebaseId] != nil

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		if exists {
			delete(s.codebaseConfigs, codebaseId)
			s.logger.Info("codebase config deleted: %s (memory only)", codebaseId)
		}
		return nil
	}

	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete codebase file: %v", err)
	}

	// Only delete in-memory config after file deletion succeeds
	if exists {
		delete(s.codebaseConfigs, codebaseId)
		s.logger.Info("codebase config deleted: %s (file and memory)", filePath)
	} else {
		s.logger.Info("codebase file deleted: %s (file only)", filePath)
	}
	return nil
}
