// scheduler/scheduler.go - 调度管理器
package scheduler

import (
	"archive/zip"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"codebase-syncer/internal/scanner"
	"codebase-syncer/internal/storage"
	"codebase-syncer/internal/syncer"
	"codebase-syncer/pkg/logger"
	"codebase-syncer/pkg/utils"
)

type Scheduler struct {
	syncInterval time.Duration
	httpSync     *syncer.HTTPSync
	fileScanner  *scanner.FileScanner
	storage      *storage.ConfigManager
	logger       logger.Logger
	mutex        sync.Mutex
	isRunning    bool
}

func NewScheduler(syncInterval time.Duration, httpSync *syncer.HTTPSync,
	fileScanner *scanner.FileScanner, storage *storage.ConfigManager,
	logger logger.Logger) *Scheduler {
	return &Scheduler{
		syncInterval: syncInterval,
		httpSync:     httpSync,
		fileScanner:  fileScanner,
		storage:      storage,
		logger:       logger,
	}
}

// 启动调度器
func (s *Scheduler) Start(ctx context.Context) {
	s.logger.Info("启动同步调度器，间隔: %v", s.syncInterval)

	// 立即执行一次同步
	if s.httpSync.GetSyncConfig() == nil {
		s.logger.Warn("未配置同步配置，跳过同步")
	} else {
		s.performSync()
	}

	// 设置定时器
	ticker := time.NewTicker(s.syncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("同步调度器已停止")
			return
		case <-ticker.C:
			if s.httpSync.GetSyncConfig() == nil {
				s.logger.Warn("未配置同步配置，跳过同步")
				continue
			}
			s.performSync()
		}
	}
}

// 设置同步间隔
func (s *Scheduler) SetSyncInterval(interval time.Duration) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.syncInterval = interval
	s.logger.Info("同步间隔已更新为: %v", interval)
}

// 执行同步
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

	projectConfigs := s.storage.GetConfigs()

	for _, config := range projectConfigs {
		s.performSyncForProject(config)
	}

	s.logger.Info("同步任务完成，总耗时: %v", time.Since(startTime))
}

func (s *Scheduler) performSyncForProject(config *storage.ProjectConfig) {
	s.logger.Info("开始执行同步任务，项目: %s", config.CodebaseId)
	startTime := time.Now()
	localHashTree, err := s.fileScanner.ScanDirectory(config.CodebasePath)
	if err != nil {
		s.logger.Error("扫描本地目录失败: %v", err)
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
			s.logger.Error("从服务器获取哈希树失败: %v", err)
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
	s.processFileChanges(config, changes)

	// 更新本地哈希树并保存配置
	config.HashTree = localHashTree
	config.LastSync = time.Now()
	if err := s.storage.SaveProjectConfig(config); err != nil {
		s.logger.Error("保存项目配置失败: %v", err)
	}

	s.logger.Info("同步任务完成，项目: %s, 耗时: %v", config.CodebaseId, time.Since(startTime))
}

type SyncMetadata struct {
	ClientId     string            `json:"clientId"`
	CodebaseName string            `json:"codebaseName"`
	CodebasePath string            `json:"codebasePath"`
	FileList     map[string]string `json:"fileList"`
	Timestamp    int64             `json:"timestamp"`
}

// processFileChanges 处理文件变更，将上传逻辑封装
func (s *Scheduler) processFileChanges(config *storage.ProjectConfig, changes []*storage.SyncFile) {
	// 创建SyncMetadata
	metadata := &SyncMetadata{
		ClientId:     config.ClientID,
		CodebaseName: config.CodebaseName,
		CodebasePath: config.CodebasePath,
		FileList:     make(map[string]string),
		Timestamp:    time.Now().Unix(),
	}

	// 按状态分类变更
	var addedFiles, modifiedFiles, deletedFiles []*storage.SyncFile
	for _, change := range changes {
		switch change.Status {
		case storage.FILE_STATUS_ADDED:
			addedFiles = append(addedFiles, change)
		case storage.FILE_STATUS_MODIFIED:
			modifiedFiles = append(modifiedFiles, change)
		case storage.FILE_STATUS_DELETED:
			deletedFiles = append(deletedFiles, change)
		}
	}

	uploadReq := &syncer.UploadReq{
		ClientId:     config.ClientID,
		CodebasePath: config.CodebasePath,
		CodebaseName: config.CodebaseName,
	}
	// 创建临时zip文件
	zipDir := filepath.Join(utils.UploadTmpDir, "zip")
	if err := os.MkdirAll(zipDir, 0755); err != nil {
		s.logger.Error("创建zip目录失败: %v", err)
		return
	}

	zipPath := filepath.Join(zipDir, config.CodebaseId+"-"+time.Now().Format("20060102150405")+".zip")
	zipFile, err := os.Create(zipPath)
	if err != nil {
		s.logger.Error("创建临时zip文件失败: %v", err)
		return
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// 上传新增文件
	for _, file := range addedFiles {
		s.logger.Info("上传新增文件: %s (项目: %s)", file.Path, config.CodebaseId)
		if err := s.httpSync.UploadFile(file, uploadReq); err != nil {
			s.logger.Error("上传新增文件失败: %s, 错误: %v", file.Path, err)
		}
		// 记录到metadata
		filePath := file.Path
		if runtime.GOOS == "windows" {
			filePath = filepath.ToSlash(filePath)
		}
		metadata.FileList[filePath] = file.Status

		// 将文件添加到zip
		if err := addFileToZip(zipWriter, file.Path, config.CodebasePath); err != nil {
			s.logger.Error("添加文件到zip失败: %s, 错误: %v", file.Path, err)
		}
	}

	// 上传修改文件
	for _, file := range modifiedFiles {
		s.logger.Info("上传修改文件: %s (项目: %s)", file.Path, config.CodebaseId)
		if err := s.httpSync.UploadFile(file, uploadReq); err != nil {
			s.logger.Error("上传修改文件失败: %s, 错误: %v", file.Path, err)
		}
		// 记录到metadata
		filePath := file.Path
		if runtime.GOOS == "windows" {
			filePath = filepath.ToSlash(filePath)
		}
		metadata.FileList[filePath] = file.Status

		// 将文件添加到zip
		if err := addFileToZip(zipWriter, file.Path, config.CodebasePath); err != nil {
			s.logger.Error("添加文件到zip失败: %s, 错误: %v", file.Path, err)
		}
	}

	// 处理删除文件
	for _, file := range deletedFiles {
		s.logger.Info("通知服务器删除文件: %s (项目: %s)", file.Path, config.CodebaseId)
		if err := s.httpSync.UploadFile(file, uploadReq); err != nil {
			s.logger.Error("通知服务器删除文件失败: %s, 错误: %v", file.Path, err)
		}
		// 记录到metadata
		filePath := file.Path
		if runtime.GOOS == "windows" {
			filePath = filepath.ToSlash(filePath)
		}
		metadata.FileList[filePath] = file.Status
	}

	// 添加metadata文件到zip
	metadataJson, err := json.Marshal(metadata)
	if err != nil {
		s.logger.Error("序列化metadata失败: %v", err)
		return
	}

	metadataPath := ".sync_metadata/" + time.Now().Format("20060102150405")
	metadataWriter, err := zipWriter.Create(metadataPath)
	if err != nil {
		s.logger.Error("创建metadata文件失败: %v", err)
		return
	}

	if _, err := metadataWriter.Write(metadataJson); err != nil {
		s.logger.Error("写入metadata文件失败: %v", err)
		return
	}

	// 确保所有数据写入zip文件
	zipWriter.Close()

	s.logger.Info("开始上报zip文件: %s", zipPath)
	if err := s.httpSync.UploadZipFile(zipPath, uploadReq); err != nil {
		s.logger.Error("上报zip文件失败: %v", err)
		return
	}
	s.logger.Info("zip文件上报成功")
}

// addFileToZip 将文件添加到zip中
func addFileToZip(zipWriter *zip.Writer, filePath string, basePath string) error {
	file, err := os.Open(filepath.Join(basePath, filePath))
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}

	if runtime.GOOS == "windows" {
		filePath = filepath.ToSlash(filePath)
	}
	header.Name = filePath
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, file)
	return err
}
