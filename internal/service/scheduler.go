// scheduler/scheduler.go - Scheduler Manager
package service

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"codebase-indexer/internal/config"
	"codebase-indexer/internal/dto"
	"codebase-indexer/internal/repository"
	"codebase-indexer/internal/utils"
	"codebase-indexer/pkg/logger"
)

type SchedulerConfig struct {
	IntervalMinutes       int // Sync interval in minutes
	RegisterExpireMinutes int // Registration expiration time in minutes
	HashTreeExpireHours   int // Hash tree expiration time in hours
	MaxRetries            int // Maximum retry count
	RetryIntervalSeconds  int // Retry interval in seconds
}

type Scheduler struct {
	httpSync        repository.SyncInterface
	fileScanner     repository.ScannerInterface
	storage         repository.StorageInterface
	schedulerConfig *SchedulerConfig
	logger          logger.Logger
	mutex           sync.Mutex
	rwMutex         sync.RWMutex
	isRunning       bool
	restartCh       chan struct{} // Restart channel
	updateCh        chan struct{} // Config update channel
	currentTicker   *time.Ticker
}

func NewScheduler(httpSync repository.SyncInterface, fileScanner repository.ScannerInterface, storageManager repository.StorageInterface,
	logger logger.Logger) *Scheduler {
	return &Scheduler{
		httpSync:        httpSync,
		fileScanner:     fileScanner,
		storage:         storageManager,
		schedulerConfig: defaultSchedulerConfig(),
		restartCh:       make(chan struct{}),
		updateCh:        make(chan struct{}),
		logger:          logger,
	}
}

// defaultSchedulerConfig Default scheduler configuration
func defaultSchedulerConfig() *SchedulerConfig {
	return &SchedulerConfig{
		IntervalMinutes:       config.DefaultConfigSync.IntervalMinutes,
		RegisterExpireMinutes: config.DefaultConfigServer.RegisterExpireMinutes,
		HashTreeExpireHours:   config.DefaultConfigServer.HashTreeExpireHours,
		MaxRetries:            config.DefaultConfigSync.MaxRetries,
		RetryIntervalSeconds:  config.DefaultConfigSync.RetryDelaySeconds,
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
		s.schedulerConfig.IntervalMinutes = config.IntervalMinutes
	}
	if config.RegisterExpireMinutes > 0 && config.RegisterExpireMinutes <= 60 {
		s.schedulerConfig.RegisterExpireMinutes = config.RegisterExpireMinutes
	}
	if config.HashTreeExpireHours > 0 {
		s.schedulerConfig.HashTreeExpireHours = config.HashTreeExpireHours
	}
	if config.MaxRetries > 1 && config.MaxRetries <= 10 {
		s.schedulerConfig.MaxRetries = config.MaxRetries
	}
	if config.RetryIntervalSeconds > 0 && config.RetryIntervalSeconds <= 30 {
		s.schedulerConfig.RetryIntervalSeconds = config.RetryIntervalSeconds
	}
}

// GetSchedulerConfig Value scheduler configuration
func (s *Scheduler) GetSchedulerConfig() *SchedulerConfig {
	s.rwMutex.RLock()
	defer s.rwMutex.RUnlock()
	return s.schedulerConfig
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

	clientConfig := config.GetClientConfig()
	// Update scheduler configuration
	schedulerConfig := &SchedulerConfig{
		IntervalMinutes:       clientConfig.Sync.IntervalMinutes,
		RegisterExpireMinutes: clientConfig.Server.RegisterExpireMinutes,
		HashTreeExpireHours:   clientConfig.Server.HashTreeExpireHours,
		MaxRetries:            clientConfig.Sync.MaxRetries,
		RetryIntervalSeconds:  clientConfig.Sync.RetryDelaySeconds,
	}
	s.SetSchedulerConfig(schedulerConfig)

	// Update scanner configuration
	scannerConfig := &config.ScannerConfig{
		FolderIgnorePatterns: clientConfig.Scan.FolderIgnorePatterns,
		FileIncludePatterns:  clientConfig.Scan.FileIncludePatterns,
		MaxFileSizeKB:        clientConfig.Scan.MaxFileSizeKB,
		MaxFileCount:         clientConfig.Scan.MaxFileCount,
	}
	s.fileScanner.SetScannerConfig(scannerConfig)
}

// runScheduler Actually run the scheduler loop
func (s *Scheduler) runScheduler(parentCtx context.Context, initial bool) {
	syncInterval := time.Duration(s.schedulerConfig.IntervalMinutes) * time.Minute

	s.logger.Info("starting sync scheduler with interval: %v", syncInterval)

	// Perform immediate sync if this is the initial run
	authInfo := config.GetAuthInfo()
	if initial && authInfo.ClientId != "" && authInfo.Token != "" && authInfo.ServerURL != "" {
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
			authInfo := config.GetAuthInfo()
			if authInfo.ClientId == "" || authInfo.Token == "" || authInfo.ServerURL == "" {
				s.logger.Warn("auth info not properly set, skipping sync")
				continue
			}
			s.performSync()
		}
	}
}

// performSync Perform sync operation
func (s *Scheduler) performSync() {
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

	syncConfigTimeout := time.Duration(s.schedulerConfig.RegisterExpireMinutes) * time.Minute
	codebaseConfigs := s.storage.GetCodebaseConfigs()
	if len(codebaseConfigs) == 0 {
		s.logger.Info("no codebase configs found, skipping sync")
		return
	}

	s.logger.Info("starting sync task")
	startTime := time.Now()
	for _, config := range codebaseConfigs {
		if config.RegisterTime.IsZero() || time.Since(config.RegisterTime) > syncConfigTimeout {
			s.logger.Info("codebase %s registration expired, deleting config, skipping sync", config.CodebaseId)
			if err := s.storage.DeleteCodebaseConfig(config.CodebaseId); err != nil {
				s.logger.Error("failed to delete codebase config: %v", err)
			}
			continue
		}
		_ = s.performSyncForCodebase(config)
	}

	s.logger.Info("sync task completed, total time: %v", time.Since(startTime))
}

// performSyncForCodebase Perform sync task for single codebase
func (s *Scheduler) performSyncForCodebase(config *config.CodebaseConfig) error {
	s.logger.Info("starting sync for codebase: %s", config.CodebaseId)
	nowTime := time.Now()
	ignoreConfig := s.fileScanner.LoadIgnoreConfig(config.CodebasePath)
	localHashTree, err := s.fileScanner.ScanCodebase(ignoreConfig, config.CodebasePath)
	if err != nil {
		s.logger.Error("failed to scan directory (%s): %v", config.CodebasePath, err)
		return fmt.Errorf("scan directory (%s) failed: %v", config.CodebasePath, err)
	}

	// Value codebase hash tree
	var serverHashTree map[string]string
	if len(config.HashTree) > 0 {
		serverHashTree = config.HashTree
	} else {
		serverHashTree = make(map[string]string)
	}

	// Compare hash trees to find changes
	changes := utils.CalculateFileChanges(localHashTree, serverHashTree)
	totalChanges := len(changes)
	if totalChanges == 0 {
		s.logger.Info("no file changes detected, sync completed")
		return nil
	}

	s.logger.Info("detected %d file changes", len(changes))

	// Process all file changes
	// Process file changes in batches of 20
	const batchSize = 20
	var lastErr error

	for i := 0; i < totalChanges; i += batchSize {
		end := i + batchSize
		if end > totalChanges {
			end = totalChanges
		}

		batch := changes[i:end]
		s.logger.Info("processing batch %d/%d, changes %d-%d", (i/batchSize)+1, (totalChanges+batchSize-1)/batchSize, i+1, end)

		if err := s.ProcessFileChanges(config, batch); err != nil {
			s.logger.Error("file changes processing failed for batch %d-%d: %v", i+1, end, err)
			lastErr = fmt.Errorf("file changes processing failed for batch %d-%d: %v", i+1, end, err)
		}
	}

	if lastErr != nil {
		return fmt.Errorf("file changes processing failed for codebase %s: %v", config.CodebaseId, lastErr)
	}

	// Update local hash tree and save configuration
	config.HashTree = localHashTree
	config.LastSync = nowTime
	if err := s.storage.SaveCodebaseConfig(config); err != nil {
		s.logger.Error("failed to save codebase config: %v", err)
		return fmt.Errorf("save codebase config failed: %v", err)
	}

	s.logger.Info("sync completed for codebase: %s, time taken: %v", config.CodebaseId, time.Since(nowTime))
	return nil
}

// PerformSyncForCodebaseWithFilePaths Perform sync for codebase with specified file paths
func (s *Scheduler) PerformSyncForCodebaseWithFilePaths(config *config.CodebaseConfig, filePaths []string) error {
	s.logger.Info("performing sync for codebase: %s, file paths: %v", config.CodebaseId, filePaths)
	nowTime := time.Now()
	localHashTree, err := s.fileScanner.ScanFilePaths(config.CodebasePath, filePaths)
	if err != nil {
		s.logger.Error("failed to scan file paths (%s): %v", config.CodebasePath, err)
		return fmt.Errorf("scan file paths (%s) failed: %v", config.CodebasePath, err)
	}

	// Value codebase hash tree
	var serverHashTree map[string]string
	if len(config.HashTree) > 0 {
		serverHashTree = config.HashTree
	}

	// Compare hash trees to find changes
	changes := utils.CalculateFileChangesWithoutDelete(localHashTree, serverHashTree)
	totalChanges := len(changes)
	if totalChanges == 0 {
		s.logger.Info("no file changes detected, sync completed")
		return nil
	}

	s.logger.Info("detected %d file changes", len(changes))

	// Process file changes in batches of 20
	const batchSize = 20
	var lastErr error

	for i := 0; i < totalChanges; i += batchSize {
		end := i + batchSize
		if end > totalChanges {
			end = totalChanges
		}

		batch := changes[i:end]
		s.logger.Info("processing batch %d/%d, changes %d-%d", (i/batchSize)+1, (totalChanges+batchSize-1)/batchSize, i+1, end)

		if err := s.ProcessFileChanges(config, batch); err != nil {
			s.logger.Error("file changes processing failed for batch %d-%d: %v", i+1, end, err)
			lastErr = fmt.Errorf("file changes processing failed for batch %d-%d: %v", i+1, end, err)
		}
	}

	if lastErr != nil {
		return fmt.Errorf("file changes processing failed for codebase %s: %v", config.CodebaseId, lastErr)
	}

	// Update local hash tree and save configuration
	if len(config.HashTree) == 0 {
		config.HashTree = localHashTree
	} else {
		maps.Copy(config.HashTree, localHashTree)
	}
	config.LastSync = nowTime
	if err := s.storage.SaveCodebaseConfig(config); err != nil {
		s.logger.Error("failed to save codebase config: %v", err)
		return fmt.Errorf("save codebase config failed: %v", err)
	}

	s.logger.Info("sync completed for codebase: %s, time taken: %v", config.CodebaseId, time.Since(nowTime))
	return nil
}

// ProcessFileChanges Process file changes and encapsulate upload logic
func (s *Scheduler) ProcessFileChanges(config *config.CodebaseConfig, changes []*utils.FileStatus) error {
	// Create zip with all changed files (new and modified)
	zipPath, err := s.CreateChangesZip(config, changes)
	if err != nil {
		return fmt.Errorf("failed to create changes zip: %v", err)
	}

	// Upload zip file
	uploadReq := dto.UploadReq{
		ClientId:     config.ClientID,
		CodebasePath: config.CodebasePath,
		CodebaseName: config.CodebaseName,
	}
	err = s.UploadChangesZip(zipPath, uploadReq)
	if err != nil {
		return fmt.Errorf("failed to upload changes zip: %v", err)
	}

	return nil
}

type SyncMetadata struct {
	ClientId     string             `json:"clientId"`
	CodebaseName string             `json:"codebaseName"`
	CodebasePath string             `json:"codebasePath"`
	FileList     []utils.FileStatus `json:"fileList"`
	Timestamp    int64              `json:"timestamp"`
}

// CreateChangesZip Create zip file containing file changes and metadata
func (s *Scheduler) CreateChangesZip(config *config.CodebaseConfig, changes []*utils.FileStatus) (string, error) {
	zipDir := filepath.Join(utils.UploadTmpDir, "zip")
	if err := os.MkdirAll(zipDir, 0755); err != nil {
		return "", err
	}

	zipPath := filepath.Join(zipDir, config.CodebaseId+"-"+time.Now().Format("20060102150405.000000")+".zip")
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return "", err
	}
	// Ensure cleanup of temporary zip file in case of error
	cleanup := func() {
		if err != nil {
			if _, statErr := os.Stat(zipPath); statErr == nil {
				_ = os.Remove(zipPath)
				s.logger.Debug("temp zip file deleted successfully: %s", zipPath)
			}
		}
	}
	defer cleanup()
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Create SyncMetadata
	metadata := &SyncMetadata{
		ClientId:     config.ClientID,
		CodebaseName: config.CodebaseName,
		CodebasePath: config.CodebasePath,
		FileList:     make([]utils.FileStatus, len(changes)),
		Timestamp:    time.Now().Unix(),
	}

	for index, change := range changes {
		if runtime.GOOS == "windows" {
			change.Path = filepath.ToSlash(change.Path)
		}
		metadata.FileList[index] = *change

		// Only add new and modified files to zip
		if change.Status == utils.FILE_STATUS_ADDED || change.Status == utils.FILE_STATUS_MODIFIED {
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

	metadataFilePath := ".shenma_sync/" + time.Now().Format("20060102150405.000000")
	metadataWriter, err := zipWriter.Create(metadataFilePath)
	if err != nil {
		return "", err
	}

	_, err = metadataWriter.Write(metadataJson)
	if err != nil {
		return "", err
	}

	return zipPath, nil
}

func (s *Scheduler) UploadChangesZip(zipPath string, uploadReq dto.UploadReq) error {
	maxRetries := s.schedulerConfig.MaxRetries
	retryDelay := time.Duration(s.schedulerConfig.RetryIntervalSeconds) * time.Second

	s.logger.Info("starting to upload zip file: %s", zipPath)

	var errUpload error
	for i := 0; i < maxRetries; i++ {
		requestId, err := utils.GenerateUUID()
		if err != nil {
			s.logger.Warn("failed to generate upload request ID, using timestamp: %v", err)
			requestId = time.Now().Format("20060102150405.000")
		}
		s.logger.Info("upload request ID: %s", requestId)
		uploadReq.RequestId = requestId
		errUpload = s.httpSync.UploadFile(zipPath, uploadReq)
		if errUpload == nil {
			break
		}
		if !isTimeoutError(errUpload) {
			s.logger.Warn("upload failed with abort retry error")
			break
		}
		s.logger.Warn("failed to upload zip file (attempt %d/%d): %v", i+1, maxRetries, errUpload)
		if i < maxRetries-1 {
			s.logger.Debug("waiting %v before retry...", retryDelay*time.Duration(i+1))
			time.Sleep(retryDelay * time.Duration(i+1))
		}
	}

	// After reporting, try to delete local zip file regardless of success
	if zipPath != "" {
		if err := os.Remove(zipPath); err != nil {
			s.logger.Warn("failed to delete temp zip file: %s, error: %v", zipPath, err)
		}
	}

	if errUpload != nil {
		return errUpload
	}

	return nil
}

// isTimeoutError checks if the error indicates a timeout
func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), "timeout")
}

// SyncForCodebases Batch sync codebases
func (s *Scheduler) SyncForCodebases(ctx context.Context, codebaseConfig []*config.CodebaseConfig) error {
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

	errs := make([]error, 0)
	s.logger.Info("starting sync for codebases")
	startTime := time.Now()
	for _, config := range codebaseConfig {
		if err := ctx.Err(); err != nil {
			return err
		}
		err := s.performSyncForCodebase(config)
		if err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("sync for codebases completed with errors: %v", errs)
	}

	s.logger.Info("sync for codebases completed, total time: %v", time.Since(startTime))
	return nil
}

// CreateSingleFileZip 创建单文件ZIP文件
func (s *Scheduler) CreateSingleFileZip(config *config.CodebaseConfig, fileStatus *utils.FileStatus) (string, error) {
	zipDir := filepath.Join(utils.UploadTmpDir, "zip")
	if err := os.MkdirAll(zipDir, 0755); err != nil {
		return "", err
	}

	zipPath := filepath.Join(zipDir, config.CodebaseId+"-"+time.Now().Format("20060102150405.000000")+".zip")
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return "", err
	}

	// 确保清理临时ZIP文件
	cleanup := func() {
		if err != nil {
			if _, statErr := os.Stat(zipPath); statErr == nil {
				_ = os.Remove(zipPath)
				s.logger.Debug("temp zip file deleted successfully: %s", zipPath)
			}
		}
	}
	defer cleanup()
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// 创建SyncMetadata
	metadata := &SyncMetadata{
		ClientId:     config.ClientID,
		CodebaseName: config.CodebaseName,
		CodebasePath: config.CodebasePath,
		FileList:     make([]utils.FileStatus, 0),
		Timestamp:    time.Now().Unix(),
	}

	if runtime.GOOS == "windows" {
		fileStatus.Path = filepath.ToSlash(fileStatus.Path)
	}
	metadata.FileList = append(metadata.FileList, *fileStatus)

	// 只添加新增和修改的文件到ZIP
	if fileStatus.Status == utils.FILE_STATUS_ADDED || fileStatus.Status == utils.FILE_STATUS_MODIFIED {
		if err := utils.AddFileToZip(zipWriter, fileStatus.Path, config.CodebasePath); err != nil {
			// Continue trying to add other files but log error
			s.logger.Warn("failed to add file to zip: %s, error: %v", fileStatus.Path, err)
		}
	}

	// 添加元数据文件到ZIP
	metadataJson, err := json.Marshal(metadata)
	if err != nil {
		return "", err
	}

	metadataFilePath := ".shenma_sync/" + time.Now().Format("20060102150405.000000")
	metadataWriter, err := zipWriter.Create(metadataFilePath)
	if err != nil {
		return "", err
	}

	_, err = metadataWriter.Write(metadataJson)
	if err != nil {
		return "", err
	}

	return zipPath, nil
}

// CreateFilesZip 创建多文件ZIP文件
func (s *Scheduler) CreateFilesZip(config *config.CodebaseConfig, fileStatus []*utils.FileStatus) (string, error) {
	zipDir := filepath.Join(utils.UploadTmpDir, "zip")
	if err := os.MkdirAll(zipDir, 0755); err != nil {
		return "", err
	}

	zipPath := filepath.Join(zipDir, config.CodebaseId+"-"+time.Now().Format("20060102150405.000000")+".zip")
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return "", err
	}

	// 确保清理临时ZIP文件
	cleanup := func() {
		if err != nil {
			if _, statErr := os.Stat(zipPath); statErr == nil {
				_ = os.Remove(zipPath)
				s.logger.Debug("temp zip file deleted successfully: %s", zipPath)
			}
		}
	}
	defer cleanup()
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// 创建SyncMetadata
	metadata := &SyncMetadata{
		ClientId:     config.ClientID,
		CodebaseName: config.CodebaseName,
		CodebasePath: config.CodebasePath,
		FileList:     make([]utils.FileStatus, 0),
		Timestamp:    time.Now().Unix(),
	}

	// 给FileList设置值，在Windows系统下需要转换路径格式
	for _, f := range fileStatus {
		if runtime.GOOS == "windows" {
			f.Path = filepath.ToSlash(f.Path)
			if f.Status == utils.FILE_STATUS_RENAME {
				f.TargetPath = filepath.ToSlash(f.TargetPath)
			}
		}
		metadata.FileList = append(metadata.FileList, *f)

		// 只添加新增和修改的文件到ZIP
		if f.Status == utils.FILE_STATUS_ADDED || f.Status == utils.FILE_STATUS_MODIFIED {
			if err := utils.AddFileToZip(zipWriter, f.Path, config.CodebasePath); err != nil {
				// Continue trying to add other files but log error
				s.logger.Warn("failed to add file to zip: %s, error: %v", f.Path, err)
			}
		}
	}

	// 添加元数据文件到ZIP
	metadataJson, err := json.Marshal(metadata)
	if err != nil {
		return "", err
	}

	metadataFilePath := ".shenma_sync/" + time.Now().Format("20060102150405.000000")
	metadataWriter, err := zipWriter.Create(metadataFilePath)
	if err != nil {
		return "", err
	}

	_, err = metadataWriter.Write(metadataJson)
	if err != nil {
		return "", err
	}

	return zipPath, nil
}
