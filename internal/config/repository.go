package config

import (
	"time"

	gitignore "github.com/sabhiram/go-gitignore"
)

// ScannerConfig holds the configuration for file scanning
type ScannerConfig struct {
	FolderIgnorePatterns []string
	FileIncludePatterns  []string
	MaxFileSizeKB        int // File size limit in KB
	MaxFileCount         int
}

// SyncConfig holds the sync configuration
type SyncConfig struct {
	ClientId  string `json:"machine_id"`
	Token     string `json:"access_token"`
	ServerURL string `json:"base_url"`
}

type CodebaseEnv struct {
	Switch string `json:"switch"`
}

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

// Embedding config
type EmbeddingConfig struct {
	ClientID     string               `json:"clientId"`
	CodebaseName string               `json:"codebaseName"`
	CodebasePath string               `json:"codebasePath"`
	CodebaseId   string               `json:"codebaseId"`
	HashTree     map[string]string    `json:"hashTree"`
	SyncFiles    map[string]string    `json:"syncFiles"`
	SyncIds      map[string]time.Time `json:"syncIds"`
	FailedFiles  map[string]string    `json:"failedFiles"`
}

// Deepwiki configuration
type DeepwikiConfig struct {
	ClientID     string            `json:"clientId"`
	CodebaseName string            `json:"codebaseName"`
	CodebasePath string            `json:"codebasePath"`
	DeepwikiId   string            `json:"deepwikiId"`
	HashTree     map[string]string `json:"hashTree"`
	LastSync     time.Time         `json:"lastSync"`
}

// Ignore config
type IgnoreConfig struct {
	IgnoreRules  *gitignore.GitIgnore
	IncludeRules []string
	MaxFileCount int
	MaxFileSize  int
}
