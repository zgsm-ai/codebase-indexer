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

type EmbeddingFileRepository interface {
	GetCodebaseEmbeddingConfigs() map[string]*config.CodebaseEmbeddingConfig
	GetCodebaseEmbeddingConfig(codebaseId string) (*config.CodebaseEmbeddingConfig, error)
	SaveCodebaseEmbeddingConfig(config *config.CodebaseEmbeddingConfig) error
	DeleteCodebaseEmbeddingConfig(codebaseId string) error
}

type EmbeddingFileRepo struct {
	codebasePath    string
	codebaseConfigs map[string]*config.CodebaseEmbeddingConfig // Stores all codebase configurations
	logger          logger.Logger
	rwMutex         sync.RWMutex
}

// NewEmbeddingFileRepo creates a new configuration manager
func NewEmbeddingFileRepo(codebaseDir string, logger logger.Logger) (EmbeddingFileRepository, error) {
	if codebaseDir == "" || strings.Contains(codebaseDir, "\x00") {
		return nil, fmt.Errorf("invalid codebase directory path")
	}

	// Try to create directory to verify write permission
	if err := os.MkdirAll(codebaseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create codebase directory: %v", err)
	}

	// Initialize codebaseConfigs map
	sm := &EmbeddingFileRepo{
		codebasePath:    codebaseDir,
		logger:          logger,
		codebaseConfigs: make(map[string]*config.CodebaseEmbeddingConfig),
	}

	sm.loadAllConfigs()
	return sm, nil
}

// GetCodebaseEmbeddingConfigs retrieves all project configurations
func (s *EmbeddingFileRepo) GetCodebaseEmbeddingConfigs() map[string]*config.CodebaseEmbeddingConfig {
	s.rwMutex.RLock()
	defer s.rwMutex.RUnlock()
	return s.codebaseConfigs
}

// GetCodebaseEmbeddingConfig loads codebase configuration
// First checks in memory, if not found then loads from filesystem
func (s *EmbeddingFileRepo) GetCodebaseEmbeddingConfig(codebaseId string) (*config.CodebaseEmbeddingConfig, error) {
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
func (s *EmbeddingFileRepo) loadAllConfigs() {
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
func (s *EmbeddingFileRepo) loadCodebaseConfig(codebaseId string) (*config.CodebaseEmbeddingConfig, error) {
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

	var config config.CodebaseEmbeddingConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse codebase file: %v", err)
	}

	if config.CodebaseId != codebaseId {
		return nil, fmt.Errorf("codebaseId mismatch: expected %s, got %s",
			codebaseId, config.CodebaseId)
	}

	s.logger.Info("codebase file loaded successfully")

	return &config, nil
}

// SaveCodebaseEmbeddingConfig saves codebase configuration
func (s *EmbeddingFileRepo) SaveCodebaseEmbeddingConfig(config *config.CodebaseEmbeddingConfig) error {
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

// DeleteCodebaseEmbeddingConfig deletes codebase configuration
func (s *EmbeddingFileRepo) DeleteCodebaseEmbeddingConfig(codebaseId string) error {
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
