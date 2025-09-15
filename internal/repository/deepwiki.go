package repository

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"codebase-indexer/internal/config"
	"codebase-indexer/pkg/logger"
)

type DeepwikiFileRepository interface {
	GetDeepwikiConfigs() map[string]*config.DeepwikiConfig
	GetDeepwikiConfig(deepwikiId string) (*config.DeepwikiConfig, error)
	SaveDeepwikiConfig(config *config.DeepwikiConfig) error
	DeleteDeepwikiConfig(deepwikiId string) error
}

type DeepwikiFileRepo struct {
	deepwikiPath    string
	deepwikiConfigs map[string]*config.DeepwikiConfig // Stores all deepwiki configurations
	logger          logger.Logger
	rwMutex         sync.RWMutex
}

// NewDeepwikiFileRepo creates a new configuration manager
func NewDeepwikiFileRepo(deepwikiDir string, logger logger.Logger) (DeepwikiFileRepository, error) {
	if deepwikiDir == "" || strings.Contains(deepwikiDir, "\x00") {
		return nil, fmt.Errorf("invalid deepwiki directory path")
	}

	// Try to create directory to verify write permission
	if err := os.MkdirAll(deepwikiDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create deepwiki directory: %v", err)
	}

	// Initialize deepwikiConfigs map
	sm := &DeepwikiFileRepo{
		deepwikiPath:    deepwikiDir,
		logger:          logger,
		deepwikiConfigs: make(map[string]*config.DeepwikiConfig),
	}

	sm.loadAllConfigs()
	return sm, nil
}

// GetDeepwikiConfigs retrieves all project configurations
func (s *DeepwikiFileRepo) GetDeepwikiConfigs() map[string]*config.DeepwikiConfig {
	s.rwMutex.RLock()
	defer s.rwMutex.RUnlock()
	return s.deepwikiConfigs
}

// GetDeepwikiConfig loads deepwiki configuration
// First checks in memory, if not found then loads from filesystem
func (s *DeepwikiFileRepo) GetDeepwikiConfig(deepwikiId string) (*config.DeepwikiConfig, error) {
	s.rwMutex.RLock()
	config, exists := s.deepwikiConfigs[deepwikiId]
	s.rwMutex.RUnlock()

	if exists {
		return config, nil
	}

	// Not found in memory, try loading from file
	config, err := s.loadDeepwikiConfig(deepwikiId)
	if err != nil {
		return nil, err
	}

	s.rwMutex.Lock()
	s.deepwikiConfigs[deepwikiId] = config
	s.rwMutex.Unlock()

	return config, nil
}

// Load all deepwiki configuration files
func (s *DeepwikiFileRepo) loadAllConfigs() {
	files, err := os.ReadDir(s.deepwikiPath)
	if err != nil {
		s.logger.Error("failed to read deepwiki directory: %v", err)
		return
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		config, err := s.loadDeepwikiConfig(file.Name())
		if err != nil {
			s.logger.Error("failed to load deepwiki file %s: %v", file.Name(), err)
			continue
		}
		s.deepwikiConfigs[file.Name()] = config
	}
}

// loadDeepwikiConfig loads a deepwiki configuration file
func (s *DeepwikiFileRepo) loadDeepwikiConfig(deepwikiId string) (*config.DeepwikiConfig, error) {
	s.logger.Info("loading deepwiki file content: %s", deepwikiId)

	s.rwMutex.RLock()
	defer s.rwMutex.RUnlock()

	filePath := filepath.Join(s.deepwikiPath, deepwikiId)

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("deepwiki file does not exist: %s", filePath)
		}
		return nil, fmt.Errorf("failed to read deepwiki file: %v", err)
	}

	var config config.DeepwikiConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse deepwiki file: %v", err)
	}

	if config.DeepwikiId != deepwikiId {
		return nil, fmt.Errorf("deepwiki Id mismatch: expected %s, got %s",
			deepwikiId, config.DeepwikiId)
	}

	s.logger.Info("deepwiki file loaded successfully, path: %s", filePath)

	return &config, nil
}

// SaveDeepwikiConfig saves deepwiki configuration
func (s *DeepwikiFileRepo) SaveDeepwikiConfig(config *config.DeepwikiConfig) error {
	if config == nil {
		return fmt.Errorf("deepwiki config is empty: %v", config)
	}
	s.logger.Info("saving deepwiki config: %s", config.CodebasePath)

	s.rwMutex.Lock()
	defer s.rwMutex.Unlock()

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize config: %v", err)
	}

	filePath := filepath.Join(s.deepwikiPath, config.DeepwikiId)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	// Atomically update in-memory configuration
	s.deepwikiConfigs[config.DeepwikiId] = config
	s.logger.Info("deepwiki config saved successfully, path: %s", filePath)
	return nil
}

// DeleteDeepwikiConfig deletes deepwiki configuration
func (s *DeepwikiFileRepo) DeleteDeepwikiConfig(deepwikiId string) error {
	s.logger.Info("deleting deepwiki config: %s", deepwikiId)

	filePath := filepath.Join(s.deepwikiPath, deepwikiId)

	s.rwMutex.Lock()
	defer s.rwMutex.Unlock()

	exists := s.deepwikiConfigs[deepwikiId] != nil

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		if exists {
			delete(s.deepwikiConfigs, deepwikiId)
			s.logger.Info("deepwiki config deleted: %s (memory only)", deepwikiId)
		}
		return nil
	}

	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete deepwiki file: %v", err)
	}

	// Only delete in-memory config after file deletion succeeds
	if exists {
		delete(s.deepwikiConfigs, deepwikiId)
		s.logger.Info("deepwiki config deleted: %s (file and memory)", filePath)
	} else {
		s.logger.Info("deepwiki file deleted: %s (file only)", filePath)
	}
	return nil
}
