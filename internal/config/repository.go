package config

import "time"

// ScannerConfig holds the configuration for file scanning
type ScannerConfig struct {
	FileIgnorePatterns   []string
	FolderIgnorePatterns []string
	MaxFileSizeKB        int // File size limit in KB
}

// SyncConfig holds the sync configuration
type SyncConfig struct {
	ClientId  string
	Token     string
	ServerURL string
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

type CodebaseEmbeddingConfig struct {
	ClientID     string            `json:"clientId"`
	CodebaseName string            `json:"codebaseName"`
	CodebasePath string            `json:"codebasePath"`
	CodebaseId   string            `json:"codebaseId"`
	HashTree     map[string]string `json:"hashTree"`
	SyncFiles    map[string]string `json:"syncFiles"`
	SyncIds      []string          `json:"syncIds"`
	FailedFiles  map[string]string `json:"failedFiles"`
}
