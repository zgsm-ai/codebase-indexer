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
	GetEmbeddingConfigs() map[string]*config.EmbeddingConfig
	GetEmbeddingConfig(embeddingId string) (*config.EmbeddingConfig, error)
	SaveEmbeddingConfig(config *config.EmbeddingConfig) error
	DeleteEmbeddingConfig(embeddingId string) error
}

type EmbeddingFileRepo struct {
	embeddingPath    string
	embeddingConfigs map[string]*config.EmbeddingConfig // Stores all embedding configurations
	logger           logger.Logger
	rwMutex          sync.RWMutex
}

// NewEmbeddingFileRepo creates a new configuration manager
func NewEmbeddingFileRepo(embeddingDir string, logger logger.Logger) (EmbeddingFileRepository, error) {
	if embeddingDir == "" || strings.Contains(embeddingDir, "\x00") {
		return nil, fmt.Errorf("invalid embedding directory path")
	}

	// Try to create directory to verify write permission
	if err := os.MkdirAll(embeddingDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create embedding directory: %v", err)
	}

	// Initialize embeddingConfigs map
	sm := &EmbeddingFileRepo{
		embeddingPath:    embeddingDir,
		logger:           logger,
		embeddingConfigs: make(map[string]*config.EmbeddingConfig),
	}

	sm.loadAllConfigs()
	return sm, nil
}

// GetEmbeddingConfigs retrieves all project configurations
func (s *EmbeddingFileRepo) GetEmbeddingConfigs() map[string]*config.EmbeddingConfig {
	s.rwMutex.RLock()
	defer s.rwMutex.RUnlock()
	return s.embeddingConfigs
}

// GetEmbeddingConfig loads embedding configuration
// First checks in memory, if not found then loads from filesystem
func (s *EmbeddingFileRepo) GetEmbeddingConfig(embeddingId string) (*config.EmbeddingConfig, error) {
	s.rwMutex.RLock()
	config, exists := s.embeddingConfigs[embeddingId]
	s.rwMutex.RUnlock()

	if exists {
		return config, nil
	}

	// Not found in memory, try loading from file
	config, err := s.loadEmbeddingConfig(embeddingId)
	if err != nil {
		return nil, err
	}

	s.rwMutex.Lock()
	s.embeddingConfigs[embeddingId] = config
	s.rwMutex.Unlock()

	return config, nil
}

// Load all embedding configuration files
func (s *EmbeddingFileRepo) loadAllConfigs() {
	files, err := os.ReadDir(s.embeddingPath)
	if err != nil {
		s.logger.Error("failed to read embedding directory: %v", err)
		return
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		config, err := s.loadEmbeddingConfig(file.Name())
		if err != nil {
			s.logger.Error("failed to load embedding file %s: %v", file.Name(), err)
			continue
		}
		s.embeddingConfigs[file.Name()] = config
	}
}

// loadEmbeddingConfig loads a embedding configuration file
func (s *EmbeddingFileRepo) loadEmbeddingConfig(embeddingId string) (*config.EmbeddingConfig, error) {
	s.logger.Info("loading embedding file content: %s", embeddingId)

	s.rwMutex.RLock()
	defer s.rwMutex.RUnlock()

	filePath := filepath.Join(s.embeddingPath, embeddingId)

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("embedding file does not exist: %s", filePath)
		}
		return nil, fmt.Errorf("failed to read embedding file: %v", err)
	}

	var config config.EmbeddingConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse embedding file: %v", err)
	}

	if config.CodebaseId != embeddingId {
		return nil, fmt.Errorf("embedding Id mismatch: expected %s, got %s",
			embeddingId, config.CodebaseId)
	}

	s.logger.Info("embedding file loaded successfully, path: %s", filePath)

	return &config, nil
}

// SaveEmbeddingConfig saves embedding configuration
func (s *EmbeddingFileRepo) SaveEmbeddingConfig(config *config.EmbeddingConfig) error {
	if config == nil {
		return fmt.Errorf("embedding config is empty: %v", config)
	}
	s.logger.Info("saving embedding config: %s", config.CodebasePath)

	s.rwMutex.Lock()
	defer s.rwMutex.Unlock()

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize config: %v", err)
	}

	filePath := filepath.Join(s.embeddingPath, config.CodebaseId)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	// Atomically update in-memory configuration
	s.embeddingConfigs[config.CodebaseId] = config
	s.logger.Info("embedding config saved successfully, path: %s", filePath)
	return nil
}

// DeleteEmbeddingConfig deletes embedding configuration
func (s *EmbeddingFileRepo) DeleteEmbeddingConfig(embeddingId string) error {
	s.logger.Info("deleting embedding config: %s", embeddingId)

	filePath := filepath.Join(s.embeddingPath, embeddingId)

	s.rwMutex.Lock()
	defer s.rwMutex.Unlock()

	exists := s.embeddingConfigs[embeddingId] != nil

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		if exists {
			delete(s.embeddingConfigs, embeddingId)
			s.logger.Info("embedding config deleted: %s (memory only)", embeddingId)
		}
		return nil
	}

	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete embedding file: %v", err)
	}

	// Only delete in-memory config after file deletion succeeds
	if exists {
		delete(s.embeddingConfigs, embeddingId)
		s.logger.Info("embedding config deleted: %s (file and memory)", filePath)
	} else {
		s.logger.Info("embedding file deleted: %s (file only)", filePath)
	}
	return nil
}
