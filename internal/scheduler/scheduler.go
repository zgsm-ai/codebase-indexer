// scheduler/scheduler.go - Scheduler Manager
package scheduler

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"codebase-syncer/internal/scanner"
	"codebase-syncer/internal/storage"
	"codebase-syncer/internal/syncer"
	"codebase-syncer/internal/utils"
	"codebase-syncer/pkg/logger"
)

const (
	statusUnauthorized       = "401" // HTTP 401 Unauthorized
	statusTooManyRequests    = "429" // HTTP 429 Too Many Requests
	statusServiceUnavailable = "503" // HTTP 503 Service Unavailable
)

type SchedulerConfig struct {
	IntervalMinutes       int // Sync interval in minutes
	RegisterExpireMinutes int // Registration expiration time in minutes
	HashTreeExpireHours   int // Hash tree expiration time in hours
	MaxRetries            int // Maximum retry count
	RetryIntervalSeconds  int // Retry interval in seconds
}

type Scheduler struct {
	httpSync         syncer.SyncInterface
	fileScanner      scanner.ScannerInterface
	storage          storage.SotrageInterface
	sechedulerConfig *SchedulerConfig
	logger           logger.Logger
	mutex            sync.Mutex
	rwMutex          sync.RWMutex
	isRunning        bool
	restartCh        chan struct{} // Restart channel
	updateCh         chan struct{} // Config update channel
	currentTicker    *time.Ticker
}

func NewScheduler(httpSync syncer.SyncInterface, fileScanner scanner.ScannerInterface, storageManager storage.SotrageInterface,
	logger logger.Logger) *Scheduler {
	return &Scheduler{
		httpSync:         httpSync,
		fileScanner:      fileScanner,
		storage:          storageManager,
		sechedulerConfig: defaultSchedulerConfig(),
		restartCh:        make(chan struct{}),
		updateCh:         make(chan struct{}),
		logger:           logger,
	}
}

// defaultSchedulerConfig Default scheduler configuration
func defaultSchedulerConfig() *SchedulerConfig {
	return &SchedulerConfig{
		IntervalMinutes:       storage.DefaultConfigSync.IntervalMinutes,
		RegisterExpireMinutes: storage.DefaultConfigServer.RegisterExpireMinutes,
		HashTreeExpireHours:   storage.DefaultConfigServer.HashTreeExpireHours,
		MaxRetries:            storage.DefaultConfigSync.MaxRetries,
		RetryIntervalSeconds:  storage.DefaultConfigSync.RetryDelaySeconds,
	}
}

// SetSchedulerConfig Set scheduler configuration
func (s *Scheduler) SetSchedulerConfig(config *SchedulerConfig) {
	if config == nil {
		return
	}
	s.rwMutex.Lock()
	defer s.rwMutex.Unlock()
	if config.IntervalMinutes > 0 && config.IntervalMinutes <= 30 {
		s.sechedulerConfig.IntervalMinutes = config.IntervalMinutes
	}
	if config.RegisterExpireMinutes > 0 && config.RegisterExpireMinutes <= 60 {
		s.sechedulerConfig.RegisterExpireMinutes = config.RegisterExpireMinutes
	}
	if config.HashTreeExpireHours > 0 {
		s.sechedulerConfig.HashTreeExpireHours = config.HashTreeExpireHours
	}
	if config.MaxRetries > 1 && config.MaxRetries <= 10 {
		s.sechedulerConfig.MaxRetries = config.MaxRetries
	}
	if config.RetryIntervalSeconds > 0 && config.RetryIntervalSeconds <= 30 {
		s.sechedulerConfig.RetryIntervalSeconds = config.RetryIntervalSeconds
	}
}

// GetSchedulerConfig Get scheduler configuration
func (s *Scheduler) GetSchedulerConfig() *SchedulerConfig {
	s.rwMutex.RLock()
	defer s.rwMutex.RUnlock()
	return s.sechedulerConfig
}

// Start Start the scheduler
func (s *Scheduler) Start(ctx context.Context) {
	go s.runScheduler(ctx, true)
}

// Restart Restart the scheduler
func (s *Scheduler) Restart(ctx context.Context) {
	s.logger.Info("preparing to restart scheduler")

	s.restartCh <- struct{}{}
	s.logger.Info("scheduler restart signal sent")
	time.Sleep(100 * time.Millisecond) // Wait for scheduler restart

	go s.runScheduler(ctx, false)
}

// LoadConfig Update scheduler configuration
func (s *Scheduler) LoadConfig(ctx context.Context) {
	s.logger.Info("preparing to load scheduler config")

	s.updateCh <- struct{}{}
	s.logger.Info("scheduler config load signal sent")
	time.Sleep(100 * time.Millisecond) // Wait for scheduler update

	config := storage.GetClientConfig()
	// Update scheduler configuration
	schedulerConfig := &SchedulerConfig{
		IntervalMinutes:       config.Sync.IntervalMinutes,
		RegisterExpireMinutes: config.Server.RegisterExpireMinutes,
		HashTreeExpireHours:   config.Server.HashTreeExpireHours,
		MaxRetries:            config.Sync.MaxRetries,
		RetryIntervalSeconds:  config.Sync.RetryDelaySeconds,
	}
	s.SetSchedulerConfig(schedulerConfig)

	// Update scanner configuration
	scannerConfig := &scanner.ScannerConfig{
		IgnorePatterns: config.Sync.IgnorePatterns,
		MaxFileSizeMB:  config.Sync.MaxFileSizeMB,
	}
	s.fileScanner.SetScannerConfig(scannerConfig)
}

// runScheduler Actually run the scheduler loop
func (s *Scheduler) runScheduler(parentCtx context.Context, initial bool) {
	syncInterval := time.Duration(s.sechedulerConfig.IntervalMinutes) * time.Minute

	s.logger.Info("starting sync scheduler with interval: %v", syncInterval)

	// Perform immediate sync if this is the initial run
	if initial && s.httpSync.GetSyncConfig() != nil {
		s.performSync()
	}

	// Setup ticker
	s.currentTicker = time.NewTicker(syncInterval)
	defer s.currentTicker.Stop()

	for {
		select {
		case <-parentCtx.Done():
			s.logger.Info("sync scheduler stopped")
			return
		case <-s.restartCh:
			s.logger.Info("received restart signal, restarting scheduler")
			return
		case <-s.updateCh:
			s.logger.Info("received config update signal, waiting for update")
			time.Sleep(500 * time.Millisecond)
			continue
		case <-s.currentTicker.C:
			if s.httpSync.GetSyncConfig() == nil {
				s.logger.Warn("sync config not found, skipping sync")
				continue
			}
			s.performSync()
		}
	}
}

// performSync Perform sync operation
func (s *Scheduler) performSync() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Prevent multiple sync tasks from running concurrently
	if s.isRunning {
		s.logger.Info("sync task already running, skipping this run")
		return
	}

	// Mark as running
	s.isRunning = true
	defer func() {
		s.isRunning = false
	}()

	s.logger.Info("starting sync task")
	startTime := time.Now()

	syncConfigTimeout := time.Duration(s.sechedulerConfig.RegisterExpireMinutes) * time.Minute
	codebaseConfigs := s.storage.GetCodebaseConfigs()
	for _, config := range codebaseConfigs {
		if config.RegisterTime.IsZero() || time.Since(config.RegisterTime) > syncConfigTimeout {
			s.logger.Info("codebase %s registration expired, deleting config, skipping sync", config.CodebaseId)
			if err := s.storage.DeleteCodebaseConfig(config.CodebaseId); err != nil {
				s.logger.Error("failed to delete codebase config: %v", err)
			}
			continue
		}
		s.performSyncForCodebase(config)
	}

	s.logger.Info("sync task completed, total time: %v", time.Since(startTime))
}

// performSyncForCodebase Perform sync task for single codebase
func (s *Scheduler) performSyncForCodebase(config *storage.CodebaseConfig) {
	s.logger.Info("starting sync for codebase: %s", config.CodebaseId)
	nowTime := time.Now()
	localHashTree, err := s.fileScanner.ScanDirectory(config.CodebasePath)
	if err != nil {
		s.logger.Error("failed to scan local directory (%s): %v", config.CodebasePath, err)
		return
	}

	// Get codebase hash tree
	var serverHashTree map[string]string
	if len(config.HashTree) > 0 && config.LastSync.Add(time.Duration(s.sechedulerConfig.HashTreeExpireHours)*time.Hour).After(nowTime) {
		serverHashTree = config.HashTree
	} else {
		s.logger.Info("local hash tree empty, fetching from server")
		serverHashTree, err = s.httpSync.FetchServerHashTree(config.CodebasePath)
		if err != nil {
			s.logger.Warn("failed to get hash tree from server: %v", err)
			// No server hash tree available, use empty hash tree for full sync
			serverHashTree = make(map[string]string)
		} else {
			// Update codebase hash tree
			s.logger.Info("fetched server hash tree successfully, updating codebase config")
			config.HashTree = serverHashTree
			config.LastSync = nowTime
			if err := s.storage.SaveCodebaseConfig(config); err != nil {
				s.logger.Error("failed to save codebase config: %v", err)
			}
		}
	}

	// Compare hash trees to find changes
	changes := s.fileScanner.CalculateFileChanges(localHashTree, serverHashTree)
	if len(changes) == 0 {
		s.logger.Info("no file changes detected, sync completed")
		return
	}

	s.logger.Info("detected %d file changes", len(changes))

	// Process all file changes
	if err := s.processFileChanges(config, changes); err != nil {
		s.logger.Error("file changes processing failed: %v", err)
		return
	}

	// Update local hash tree and save configuration
	config.HashTree = localHashTree
	config.LastSync = nowTime
	if err := s.storage.SaveCodebaseConfig(config); err != nil {
		s.logger.Error("failed to save codebase config: %v", err)
	}

	s.logger.Info("sync completed for codebase: %s, time taken: %v", config.CodebaseId, time.Since(nowTime))
}

// processFileChanges Process file changes and encapsulate upload logic
func (s *Scheduler) processFileChanges(config *storage.CodebaseConfig, changes []*scanner.FileStatus) error {
	// Create zip with all changed files (new and modified)
	zipPath, err := s.createChangesZip(config, changes)
	if err != nil {
		return fmt.Errorf("failed to create changes zip: %v", err)
	}

	// Upload zip file
	uploadReq := &syncer.UploadReq{
		ClientId:     config.ClientID,
		CodebasePath: config.CodebasePath,
		CodebaseName: config.CodebaseName,
	}
	err = s.uploadChangesZip(zipPath, uploadReq)
	if err != nil {
		return fmt.Errorf("failed to upload changes zip: %v", err)
	}

	return nil
}

type SyncMetadata struct {
	ClientId     string            `json:"clientId"`
	CodebaseName string            `json:"codebaseName"`
	CodebasePath string            `json:"codebasePath"`
	FileList     map[string]string `json:"fileList"`
	Timestamp    int64             `json:"timestamp"`
}

// createChangesZip Create zip file containing file changes and metadata
func (s *Scheduler) createChangesZip(config *storage.CodebaseConfig, changes []*scanner.FileStatus) (string, error) {
	zipDir := filepath.Join(utils.UploadTmpDir, "zip")
	if err := os.MkdirAll(zipDir, 0755); err != nil {
		return "", err
	}

	zipPath := filepath.Join(zipDir, config.CodebaseId+"-"+time.Now().Format("20060102150405")+".zip")
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return "", err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Create SyncMetadata
	metadata := &SyncMetadata{
		ClientId:     config.ClientID,
		CodebaseName: config.CodebaseName,
		CodebasePath: config.CodebasePath,
		FileList:     make(map[string]string),
		Timestamp:    time.Now().Unix(),
	}

	for _, change := range changes {
		filePath := change.Path
		if runtime.GOOS == "windows" {
			filePath = filepath.ToSlash(filePath)
		}
		metadata.FileList[filePath] = change.Status

		// Only add new and modified files to zip
		if change.Status == scanner.FILE_STATUS_ADDED || change.Status == scanner.FILE_STATUS_MODIFIED {
			if err := utils.AddFileToZip(zipWriter, change.Path, config.CodebasePath); err != nil {
				// Continue trying to add other files but log error
				s.logger.Warn("failed to add file to zip: %s, error: %v", change.Path, err)
			}
		}
	}

	// Add metadata file to zip
	metadataJson, err := json.Marshal(metadata)
	if err != nil {
		return "", err
	}

	metadataFilePath := ".shenma_sync/" + time.Now().Format("20060102150405")
	metadataWriter, err := zipWriter.Create(metadataFilePath)
	if err != nil {
		return "", err
	}

	if _, err := metadataWriter.Write(metadataJson); err != nil {
		return "", err
	}

	return zipPath, nil
}

func (s *Scheduler) uploadChangesZip(zipPath string, uploadReq *syncer.UploadReq) error {
	maxRetries := s.sechedulerConfig.MaxRetries
	retryDelay := time.Duration(s.sechedulerConfig.RetryIntervalSeconds) * time.Second

	s.logger.Info("starting to upload zip file: %s", zipPath)

	var errUpload error
	for i := 0; i < maxRetries; i++ {
		errUpload = s.httpSync.UploadFile(zipPath, uploadReq)
		if errUpload == nil {
			s.logger.Info("zip file uploaded successfully")
			break
		}
		if isAbortRetryError(errUpload) {
			s.logger.Warn("upload failed with abort retry error")
			break
		}
		s.logger.Warn("failed to upload zip file (attempt %d/%d): %v", i+1, maxRetries, errUpload)
		if i < maxRetries-1 {
			s.logger.Info("waiting %v before retry...", retryDelay*time.Duration(i+1))
			time.Sleep(retryDelay * time.Duration(i+1))
		}
	}

	// After reporting, try to delete local zip file regardless of success
	if zipPath != "" {
		if err := os.Remove(zipPath); err != nil {
			s.logger.Warn("failed to delete temp zip file: %s, error: %v", zipPath, err)
		} else {
			s.logger.Info("temp zip file deleted successfully: %s", zipPath)
		}
	}

	if errUpload != nil {
		return errUpload
	}

	return nil
}

// isAbortRetryError checks if the error indicates we should abort retrying
func isAbortRetryError(err error) bool {
	if err == nil {
		return false
	}

	errorStr := err.Error()
	return strings.Contains(errorStr, statusUnauthorized) ||
		strings.Contains(errorStr, statusTooManyRequests) ||
		strings.Contains(errorStr, statusServiceUnavailable)
}

// SyncForCodebases Batch sync codebases
func (s *Scheduler) SyncForCodebases(ctx context.Context, codebaseConfig []*storage.CodebaseConfig) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Prevent multiple sync tasks running concurrently
	if s.isRunning {
		s.logger.Info("sync task already running, skipping this sync")
		return nil
	}

	// Mark as running
	s.isRunning = true
	defer func() {
		s.isRunning = false
	}()

	// Check if context was cancelled
	if err := ctx.Err(); err != nil {
		return err
	}

	s.logger.Info("starting sync for codebases")
	startTime := time.Now()
	for _, config := range codebaseConfig {
		if err := ctx.Err(); err != nil {
			return err
		}
		s.performSyncForCodebase(config)
	}

	s.logger.Info("sync for codebases completed, total time: %v", time.Since(startTime))
	return nil
}
