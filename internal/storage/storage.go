// storage/storage.go - 配置和临时文件存储
package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"codebase-syncer/pkg/logger"
)

// codebase配置
type CodebaseConfig struct {
	ClientID     string            `json:"clientId"`
	CodebaseName string            `json:"codebaseName"`
	CodebasePath string            `json:"codebasePath"`
	CodebaseId   string            `json:"codebaseId"`
	HashTree     map[string]string `json:"hashTree"`
	LastSync     time.Time         `json:"lastSync"`
	RegisterTime time.Time         `json:"registerTime"`
}

type SotrageInterface interface {
	GetCodebaseConfigs() map[string]*CodebaseConfig
	GetCodebaseConfig(codebaseId string) (*CodebaseConfig, error)
	SaveCodebaseConfig(config *CodebaseConfig) error
	DeleteCodebaseConfig(codebaseId string) error
}

type StorageManager struct {
	codebasePath    string
	codebaseConfigs map[string]*CodebaseConfig // 存储所有codebase 配置
	logger          logger.Logger
	rwMutex         sync.RWMutex
}

// NewStorageManager 创建一个新的配置管理器
func NewStorageManager(cacheDir string, logger logger.Logger) (SotrageInterface, error) {
	// 确保codebase目录存在
	codebasePath := filepath.Join(cacheDir, "codebase")
	if _, err := os.Stat(codebasePath); os.IsNotExist(err) {
		if err := os.MkdirAll(codebasePath, 0755); err != nil {
			return nil, fmt.Errorf("failed to create codebase directory: %v", err)
		}
	}

	// 初始化 codebaseConfigs map
	sm := &StorageManager{
		codebasePath:    codebasePath,
		logger:          logger,
		codebaseConfigs: make(map[string]*CodebaseConfig),
	}

	sm.loadAllConfigs()
	return sm, nil
}

// GetCodebaseConfigs 获取所有项目配置
func (s *StorageManager) GetCodebaseConfigs() map[string]*CodebaseConfig {
	s.rwMutex.RLock()
	defer s.rwMutex.RUnlock()
	return s.codebaseConfigs
}

// GetCodebaseConfig 加载codebase 配置
// 优先从内存配置中查找，找不到再从文件系统加载
func (s *StorageManager) GetCodebaseConfig(codebaseId string) (*CodebaseConfig, error) {
	s.rwMutex.RLock()
	config, exists := s.codebaseConfigs[codebaseId]
	s.rwMutex.RUnlock()

	if exists {
		return config, nil
	}

	// 内存中没有查到，尝试从文件加载
	config, err := s.loadCodebaseConfig(codebaseId)
	if err != nil {
		return nil, err
	}

	s.rwMutex.Lock()
	s.codebaseConfigs[codebaseId] = config
	s.rwMutex.Unlock()

	return config, nil
}

// 加载所有codebase 配置文件
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

// loadCodebaseConfig 加载codebase 配置文件
func (s *StorageManager) loadCodebaseConfig(codebaseId string) (*CodebaseConfig, error) {
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

	var config CodebaseConfig
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

// SaveCodebaseConfig 保存codebase 配置
func (s *StorageManager) SaveCodebaseConfig(config *CodebaseConfig) error {
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

	// 原子性更新内存配置
	s.codebaseConfigs[config.CodebaseId] = config
	s.logger.Info("codebase config saved successfully, path: %s, codebaseId: %s", filePath, config.CodebaseId)
	return nil
}

// DeleteCodebaseConfig 删除codebase 配置
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

	// 文件删除成功后才删除内存中的配置
	if exists {
		delete(s.codebaseConfigs, codebaseId)
		s.logger.Info("codebase config deleted: %s (file and memory)", filePath)
	} else {
		s.logger.Info("codebase file deleted: %s (file only)", filePath)
	}
	return nil
}
