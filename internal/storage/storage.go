// storage/storage.go - Configuration and temporary file storage
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

// Codebase configuration
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
	codebaseConfigs map[string]*CodebaseConfig // Stores all codebase configurations
	logger          logger.Logger
	rwMutex         sync.RWMutex
}

// NewStorageManager creates a new configuration manager
func NewStorageManager(cacheDir string, logger logger.Logger) (SotrageInterface, error) {
	// Make sure codebase directory exists
	codebasePath := filepath.Join(cacheDir, "codebase")
	if _, err := os.Stat(codebasePath); os.IsNotExist(err) {
		if err := os.MkdirAll(codebasePath, 0755); err != nil {
			return nil, fmt.Errorf("failed to create codebase directory: %v", err)
		}
	}

	// Initialize codebaseConfigs map
	sm := &StorageManager{
		codebasePath:    codebasePath,
		logger:          logger,
		codebaseConfigs: make(map[string]*CodebaseConfig),
	}

	sm.loadAllConfigs()
	return sm, nil
}

// GetCodebaseConfigs retrieves all project configurations
func (s *StorageManager) GetCodebaseConfigs() map[string]*CodebaseConfig {
	s.rwMutex.RLock()
	defer s.rwMutex.RUnlock()
	return s.codebaseConfigs
}

// GetCodebaseConfig loads codebase configuration
// First checks in memory, if not found then loads from filesystem
func (s *StorageManager) GetCodebaseConfig(codebaseId string) (*CodebaseConfig, error) {
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

// SaveCodebaseConfig saves codebase configuration
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

	// Atomically update in-memory configuration
	s.codebaseConfigs[config.CodebaseId] = config
	s.logger.Info("codebase config saved successfully, path: %s, codebaseId: %s", filePath, config.CodebaseId)
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
