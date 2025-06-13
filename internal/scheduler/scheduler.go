// scheduler/scheduler.go - 调度管理器
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

type SchedulerConfig struct {
	IntervalMinutes       int // 同步间隔，单位：分钟
	RegisterExpireMinutes int // 注册过期时间，单位：分钟
	MaxRetries            int // 最大重试次数
	RetryIntervalSeconds  int // 重试间隔，单位：秒
}

type Scheduler struct {
	httpSync         syncer.SyncInterface
	fileScanner      scanner.ScannerInterface
	storage          storage.SotrageInterface
	sechedulerConfig *SchedulerConfig
	logger           logger.Logger
	mutex            sync.Mutex
	isRunning        bool
	restartCh        chan struct{} // 重启通道
	updateCh         chan struct{} // 更新配置通道
	currentTicker    *time.Ticker
}

func NewScheduler(httpSync syncer.SyncInterface, fileScanner scanner.ScannerInterface, storageManager storage.SotrageInterface,
	logger logger.Logger) *Scheduler {
	defaultSchedulerConfig := &SchedulerConfig{
		IntervalMinutes:       storage.DefaultConfigSync.IntervalMinutes,
		RegisterExpireMinutes: storage.DefaultConfigServer.RegisterExpireMinutes,
		MaxRetries:            storage.DefaultConfigSync.MaxRetries,
		RetryIntervalSeconds:  storage.DefaultConfigSync.RetryDelaySeconds,
	}
	return &Scheduler{
		httpSync:         httpSync,
		fileScanner:      fileScanner,
		storage:          storageManager,
		sechedulerConfig: defaultSchedulerConfig,
		restartCh:        make(chan struct{}),
		updateCh:         make(chan struct{}),
		logger:           logger,
	}
}

// SetSchedulerConfig 设置调度器配置
func (s *Scheduler) SetSchedulerConfig(config *SchedulerConfig) {
	if config == nil {
		return
	}
	if config.IntervalMinutes > 0 && config.IntervalMinutes <= 30 {
		s.sechedulerConfig.IntervalMinutes = config.IntervalMinutes
	}
	if config.RegisterExpireMinutes > 0 && config.RegisterExpireMinutes <= 60 {
		s.sechedulerConfig.RegisterExpireMinutes = config.RegisterExpireMinutes
	}
	if config.MaxRetries > 1 && config.MaxRetries <= 10 {
		s.sechedulerConfig.MaxRetries = config.MaxRetries
	}
	if config.RetryIntervalSeconds > 0 && config.RetryIntervalSeconds <= 30 {
		s.sechedulerConfig.RetryIntervalSeconds = config.RetryIntervalSeconds
	}
}

// 启动调度器
func (s *Scheduler) Start(ctx context.Context) {
	go s.runScheduler(ctx, true)
}

// runScheduler 实际运行调度器循环
func (s *Scheduler) runScheduler(parentCtx context.Context, initial bool) {
	syncInterval := time.Duration(s.sechedulerConfig.IntervalMinutes) * time.Minute

	s.logger.Info("启动同步调度器，间隔: %v", syncInterval)

	// 立即执行一次同步
	if initial && s.httpSync.GetSyncConfig() != nil {
		s.performSync()
	}

	// 设置定时器
	s.currentTicker = time.NewTicker(syncInterval)
	defer s.currentTicker.Stop()

	for {
		select {
		case <-parentCtx.Done():
			s.logger.Info("同步调度器已停止")
			return
		case <-s.restartCh:
			s.logger.Info("收到重启信号，重启调度器")
			return
		case <-s.updateCh:
			s.logger.Info("收到更新配置信号，等待更新配置")
			time.Sleep(500 * time.Millisecond)
			continue
		case <-s.currentTicker.C:
			if s.httpSync.GetSyncConfig() == nil {
				s.logger.Warn("未配置同步配置，跳过同步")
				continue
			}
			s.performSync()
		}
	}
}

// Restart 重启调度器
func (s *Scheduler) Restart(ctx context.Context) {
	s.logger.Info("准备重启调度器")

	s.restartCh <- struct{}{}
	s.logger.Info("调度器重启信号已发送")
	time.Sleep(100 * time.Millisecond) // 等待调度器重启

	go s.runScheduler(ctx, false)
}

// Update 更新调度器配置
func (s *Scheduler) Update(ctx context.Context) {
	s.logger.Info("准备更新调度器")

	s.updateCh <- struct{}{}
	s.logger.Info("调度器更新配置信号已发送")
	time.Sleep(100 * time.Millisecond) // 等待调度器更新

	config := storage.GetClientConfig()
	// 更新scheduler配置
	schedulerConfig := &SchedulerConfig{
		IntervalMinutes:       config.Sync.IntervalMinutes,
		RegisterExpireMinutes: config.Server.RegisterExpireMinutes,
		MaxRetries:            config.Sync.MaxRetries,
		RetryIntervalSeconds:  config.Sync.RetryDelaySeconds,
	}
	s.SetSchedulerConfig(schedulerConfig)

	// 更新scanner配置
	scannerConfig := &scanner.ScannerConfig{
		IgnorePatterns: config.Sync.IgnorePatterns,
		MaxFileSizeMB:  config.Sync.MaxFileSizeMB,
	}
	s.fileScanner.SetScannerConfig(scannerConfig)
}

// performSync 执行同步
func (s *Scheduler) performSync() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// 防止同时执行多个同步任务
	if s.isRunning {
		s.logger.Info("已有同步任务正在执行，跳过本次同步")
		return
	}

	s.isRunning = true
	defer func() {
		s.isRunning = false
	}()

	s.logger.Info("开始执行同步任务")
	startTime := time.Now()

	syncConfigTimeout := time.Duration(s.sechedulerConfig.RegisterExpireMinutes) * time.Minute
	codebaseConfigs := s.storage.GetCodebaseConfigs()
	for _, config := range codebaseConfigs {
		if config.RegisterTime.IsZero() || time.Since(config.RegisterTime) > syncConfigTimeout {
			s.logger.Info("codebase %s 注册已过期，删除配置，跳过同步", config.CodebaseId)
			if err := s.storage.DeleteCodebaseConfig(config.CodebaseId); err != nil {
				s.logger.Error("删除codebase配置失败: %v", err)
			}
			continue
		}
		s.performSyncForCodebase(config)
	}

	s.logger.Info("同步任务完成，总耗时: %v", time.Since(startTime))
}

// performSyncForCodebase 执行单个codebase 的同步任务
func (s *Scheduler) performSyncForCodebase(config *storage.CodebaseConfig) {
	s.logger.Info("开始执行同步任务，codebase: %s", config.CodebaseId)
	startTime := time.Now()
	localHashTree, err := s.fileScanner.ScanDirectory(config.CodebasePath)
	if err != nil {
		s.logger.Error("扫描本地目录(%s)失败: %v", config.CodebasePath, err)
		return
	}

	// 获取服务器哈希树
	var serverHashTree map[string]string
	if len(config.HashTree) > 0 {
		serverHashTree = config.HashTree
	} else {
		s.logger.Info("本地哈希树为空，从服务器获取")
		serverHashTree, err = s.httpSync.FetchServerHashTree(config.CodebasePath)
		if err != nil {
			s.logger.Warn("从服务器获取哈希树失败: %v", err)
			// 没有服务器哈希树，使用空哈希树进行全量同步
			serverHashTree = make(map[string]string)
		}
	}

	// 比较哈希树，找出变更
	changes := s.fileScanner.CalculateFileChanges(localHashTree, serverHashTree)
	if len(changes) == 0 {
		s.logger.Info("未检测到文件变更，同步完成")
		return
	}

	s.logger.Info("检测到 %d 个文件变更", len(changes))

	// 处理所有文件变更
	if err := s.processFileChanges(config, changes); err != nil {
		s.logger.Error("同步任务失败，处理文件变更失败: %v", err)
		return
	}

	// 更新本地哈希树并保存配置
	config.HashTree = localHashTree
	config.LastSync = time.Now()
	if err := s.storage.SaveCodebaseConfig(config); err != nil {
		s.logger.Error("保存codebase 配置失败: %v", err)
	}

	s.logger.Info("同步任务完成，codebase: %s, 耗时: %v", config.CodebaseId, time.Since(startTime))
}

type SyncMetadata struct {
	ClientId     string            `json:"clientId"`
	CodebaseName string            `json:"codebaseName"`
	CodebasePath string            `json:"codebasePath"`
	FileList     map[string]string `json:"fileList"`
	Timestamp    int64             `json:"timestamp"`
}

// processFileChanges 处理文件变更，将上传逻辑封装
func (s *Scheduler) processFileChanges(config *storage.CodebaseConfig, changes []*scanner.FileStatus) error {
	// 创建包含所有变更（新增和修改的文件）的zip文件
	zipPath, err := s.createChangesZip(config, changes)
	if err != nil {
		return fmt.Errorf("创建zip文件失败: %v", err)
	}

	// 上传zip文件
	uploadReq := &syncer.UploadReq{
		ClientId:     config.ClientID,
		CodebasePath: config.CodebasePath,
		CodebaseName: config.CodebaseName,
	}
	err = s.uploadChangesZip(zipPath, uploadReq)
	if err != nil {
		return fmt.Errorf("上传zip文件失败: %v", err)
	}

	return nil
}

// createChangesZip 创建包含文件变更和元数据的zip文件
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

	// 创建SyncMetadata
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

		// 只将新增和修改的文件添加到zip包
		if change.Status == scanner.FILE_STATUS_ADDED || change.Status == scanner.FILE_STATUS_MODIFIED {
			if err := utils.AddFileToZip(zipWriter, change.Path, config.CodebasePath); err != nil {
				// 继续尝试添加其他文件，但记录错误
				s.logger.Warn("添加文件到zip失败: %s, 错误: %v", change.Path, err)
			}
		}
	}

	// 添加metadata文件到zip
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

	s.logger.Info("开始上报zip文件: %s", zipPath)

	var errUpload error
	for i := 0; i < maxRetries; i++ {
		errUpload = s.httpSync.UploadFile(zipPath, uploadReq)
		if errUpload == nil {
			s.logger.Info("zip文件上报成功")
			break
		}
		if strings.Contains(errUpload.Error(), "429") || strings.Contains(errUpload.Error(), "503") {
			s.logger.Warn("上传文件被限流，退出重试")
			break
		}
		s.logger.Warn("上报zip文件失败 (尝试 %d/%d): %v", i+1, maxRetries, errUpload)
		if i < maxRetries-1 {
			s.logger.Info("等待 %v 后重试...", retryDelay*time.Duration(i+1))
			time.Sleep(retryDelay * time.Duration(i+1))
		}
	}

	// 上报结束后，无论成功与否，都尝试删除本地的zip文件
	if zipPath != "" {
		if err := os.Remove(zipPath); err != nil {
			s.logger.Warn("删除临时zip文件失败: %s, 错误: %v", zipPath, err)
		} else {
			s.logger.Info("成功删除临时zip文件: %s", zipPath)
		}
	}

	if errUpload != nil {
		return errUpload
	}

	return nil
}
