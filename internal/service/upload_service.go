package service

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"codebase-indexer/internal/config"
	"codebase-indexer/internal/dto"
	"codebase-indexer/internal/repository"
	"codebase-indexer/internal/utils"
	"codebase-indexer/pkg/logger"
)

// UploadService 文件上传服务接口
type UploadService interface {
	// UploadFileWithRetry 带重试的文件上传
	UploadFileWithRetry(workspacePath string, filePath string, maxRetries int) (string, error)
	// DeleteFileWithRetry 带重试的文件删除
	DeleteFileWithRetry(workspacePath string, filePath string, maxRetries int) (string, error)
	// UploadFilesWithRetry 批量带重试的文件上传
	UploadFilesWithRetry(workspacePath string, filePaths []string, maxRetries int) ([]string, error)
	// DeleteFilesWithRetry 批量带重试的文件删除
	DeleteFilesWithRetry(workspacePath string, filePaths []string, maxRetries int) ([]string, error)
	// RenameFileWithRetry 带重试的文件重命名
	RenameFileWithRetry(workspacePath string, oldFilePath string, newFilePath string, maxRetries int) (string, error)
	// RenameFilesWithRetry 批量带重试的文件重命名
	RenameFilesWithRetry(workspacePath string, renamePairs []utils.FileRenamePair, maxRetries int) ([]string, error)
}

// UploadConfig 上传配置
type UploadConfig struct {
	MaxRetries      int           `json:"maxRetries"`      // 最大重试次数
	BaseRetryDelay  time.Duration `json:"baseRetryDelay"`  // 基础重试延迟
	FileSizeLimitMB int           `json:"fileSizeLimitMB"` // 文件大小限制(MB)
	Timeout         time.Duration `json:"timeout"`         // 上传超时时间
	EnableRetry     bool          `json:"enableRetry"`     // 是否启用重试
}

// DefaultUploadConfig 默认上传配置
var DefaultUploadConfig = UploadConfig{
	MaxRetries:      3,
	BaseRetryDelay:  1 * time.Second,
	FileSizeLimitMB: 100,
	Timeout:         300 * time.Second,
	EnableRetry:     true,
}

// uploadService 文件上传服务实现
type uploadService struct {
	scheduler *Scheduler
	syncer    repository.SyncInterface
	logger    logger.Logger
	config    *config.SyncConfig
	uploadCfg *UploadConfig
}

// NewUploadService 创建文件上传服务
func NewUploadService(
	scheduler *Scheduler,
	syncer repository.SyncInterface,
	logger logger.Logger,
	config *config.SyncConfig,
) UploadService {
	// 复制默认配置
	uploadCfg := DefaultUploadConfig
	return &uploadService{
		scheduler: scheduler,
		syncer:    syncer,
		logger:    logger,
		config:    config,
		uploadCfg: &uploadCfg,
	}
}

// SetUploadConfig 设置上传配置
func (us *uploadService) SetUploadConfig(cfg *UploadConfig) {
	if cfg == nil {
		return
	}
	us.uploadCfg = cfg
}

// GetUploadConfig 获取上传配置
func (us *uploadService) GetUploadConfig() *UploadConfig {
	return us.uploadCfg
}

// UploadFileWithRetry 带重试的文件上传算法
func (us *uploadService) UploadFileWithRetry(workspacePath string, filePath string, maxRetries int) (string, error) {
	if !us.uploadCfg.EnableRetry {
		// 如果禁用重试，直接上传一次
		return us.uploadSingleFile(workspacePath, filePath)
	}

	// 使用配置中的最大重试次数或传入的参数
	actualMaxRetries := us.uploadCfg.MaxRetries
	if maxRetries > 0 {
		actualMaxRetries = maxRetries
	}

	var lastErr error

	for attempt := 1; attempt <= actualMaxRetries; attempt++ {
		us.logger.Info("uploading file %s (attempt %d/%d)", filePath, attempt, actualMaxRetries)

		requestId, err := us.uploadSingleFile(workspacePath, filePath)
		if err == nil {
			us.logger.Info("file %s uploaded successfully", filePath)
			return requestId, nil
		}

		lastErr = err
		us.logger.Warn("failed to upload file %s (attempt %d/%d): %v", filePath, attempt, actualMaxRetries, err)

		if attempt < actualMaxRetries {
			// 检查是否为可重试错误
			if !us.isRetryableError(err) {
				us.logger.Error("non-retryable error occurred for file %s: %v", filePath, err)
				break
			}

			// 指数退避
			delay := us.uploadCfg.BaseRetryDelay * time.Duration(math.Pow(2, float64(attempt-1)))
			us.logger.Info("waiting %v before retry...", delay)
			time.Sleep(delay)
		}
	}

	return "", fmt.Errorf("failed to upload file %s after %d attempts, last error: %w", filePath, actualMaxRetries, lastErr)
}

func (us *uploadService) DeleteFileWithRetry(workspacePath string, filePath string, maxRetries int) (string, error) {
	fileStatus := &utils.FileStatus{
		Path:   filePath,
		Status: utils.FILE_STATUS_DELETED,
	}

	// 6. 获取上传令牌
	tokenReq := dto.UploadTokenReq{
		ClientId:     us.config.ClientId,
		CodebasePath: workspacePath,
		CodebaseName: filepath.Base(workspacePath),
	}

	tokenResp, err := us.syncer.FetchUploadToken(tokenReq)
	if err != nil {
		return "", fmt.Errorf("failed to fetch upload token: %w", err)
	}

	// 4. 创建临时的 codebase 配置
	codebaseConfig := &config.CodebaseConfig{
		ClientID:     us.config.ClientId,
		CodebaseId:   filepath.Base(workspacePath),
		CodebasePath: workspacePath,
		CodebaseName: filepath.Base(workspacePath),
		RegisterTime: time.Now(),
	}

	// 5. 创建ZIP文件
	zipPath, err := us.scheduler.CreateSingleFileZip(codebaseConfig, fileStatus)
	if err != nil {
		return "", fmt.Errorf("failed to create zip file: %w", err)
	}

	// 清理临时文件
	defer func() {
		if zipPath != "" {
			if err := os.Remove(zipPath); err != nil {
				us.logger.Warn("failed to delete temp zip file %s: %v", zipPath, err)
			} else {
				us.logger.Info("temp zip file deleted successfully: %s", zipPath)
			}
		}
	}()

	// 7. 上传文件
	requestId, err := utils.GenerateUUID()
	if err != nil {
		us.logger.Warn("failed to generate delete sync ID, using timestamp: %v", err)
		requestId = time.Now().Format("20060102150405000")
	}
	uploadReq := dto.UploadReq{
		ClientId:     us.config.ClientId,
		CodebasePath: workspacePath,
		CodebaseName: filepath.Base(workspacePath),
		RequestId:    requestId,
		UploadToken:  tokenResp.Data.Token,
	}

	err = us.syncer.UploadFile(zipPath, uploadReq)
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	us.logger.Info("file %s uploaded successfully", filePath)
	return requestId, nil
}

// uploadSingleFile 单文件上传算法
func (us *uploadService) uploadSingleFile(workspacePath string, filePath string) (string, error) {
	// 1. 验证文件路径
	fullPath := filepath.Join(workspacePath, filePath)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return "", fmt.Errorf("file does not exist: %s", fullPath)
	}

	// TODO: 获取workspacePath的函数codebaseConfig

	// 2. 检查文件大小
	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to get file info: %w", err)
	}

	fileSizeMB := float64(fileInfo.Size()) / (1024 * 1024)
	if fileSizeMB > float64(us.uploadCfg.FileSizeLimitMB) {
		return "", fmt.Errorf("file size %.2fMB exceeds limit %dMB", fileSizeMB, us.uploadCfg.FileSizeLimitMB)
	}

	// TODO：获取文件哈希值

	// 6. 获取上传令牌
	tokenReq := dto.UploadTokenReq{
		ClientId:     us.config.ClientId,
		CodebasePath: workspacePath,
		CodebaseName: filepath.Base(workspacePath),
	}
	tokenResp, err := us.syncer.FetchUploadToken(tokenReq)
	if err != nil {
		return "", fmt.Errorf("failed to fetch upload token: %w", err)
	}

	// 3. 创建文件变更对象
	fileStatus := &utils.FileStatus{
		Path:   filePath,
		Status: utils.FILE_STATUS_MODIFIED,
	}

	// 4. 创建临时的 codebase 配置
	codebaseConfig := &config.CodebaseConfig{
		ClientID:     us.config.ClientId,
		CodebaseId:   filepath.Base(workspacePath),
		CodebasePath: workspacePath,
		CodebaseName: filepath.Base(workspacePath),
		RegisterTime: time.Now(),
	}

	// 5. 创建ZIP文件
	zipPath, err := us.scheduler.CreateSingleFileZip(codebaseConfig, fileStatus)
	if err != nil {
		return "", fmt.Errorf("failed to create zip file: %w", err)
	}

	// 清理临时文件
	defer func() {
		if zipPath != "" {
			if err := os.Remove(zipPath); err != nil {
				us.logger.Warn("failed to delete temp zip file %s: %v", zipPath, err)
			} else {
				us.logger.Info("temp zip file deleted successfully: %s", zipPath)
			}
		}
	}()

	// 7. 上传文件
	requestId, err := utils.GenerateUUID()
	if err != nil {
		us.logger.Warn("failed to generate upload request ID, using timestamp: %v", err)
		requestId = time.Now().Format("20060102150405.000")
	}
	us.logger.Info("upload request ID: %s", requestId)
	uploadReq := dto.UploadReq{
		ClientId:     us.config.ClientId,
		CodebasePath: workspacePath,
		CodebaseName: filepath.Base(workspacePath),
		RequestId:    requestId,
		UploadToken:  tokenResp.Data.Token,
	}

	err = us.syncer.UploadFile(zipPath, uploadReq)
	if err != nil {
		return "", fmt.Errorf("failed to upload file: %w", err)
	}

	// TODO: 上传成功后，将文件hash保存到codebaseConfig中

	us.logger.Info("file %s uploaded successfully", filePath)
	return requestId, nil
}

// isRetryableError 检查错误是否可重试
func (us *uploadService) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()

	// 网络相关错误可重试
	if strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "network is unreachable") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "no such host") {
		return true
	}

	// 服务器错误可重试
	if strings.Contains(errStr, "500") ||
		strings.Contains(errStr, "502") ||
		strings.Contains(errStr, "503") ||
		strings.Contains(errStr, "504") {
		return true
	}

	// 临时性错误可重试
	if strings.Contains(errStr, "temporary") ||
		strings.Contains(errStr, "temporarily") ||
		strings.Contains(errStr, "too many requests") ||
		strings.Contains(errStr, "rate limit") {
		return true
	}

	return false
}

// retryWithExponentialBackoff 重试机制实现
func (us *uploadService) retryWithExponentialBackoff(
	operation func() error,
	maxRetries int,
	baseDelay time.Duration,
) error {
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := operation()
		if err == nil {
			return nil
		}

		lastErr = err

		// 检查是否可重试
		if !us.isRetryableError(err) {
			return fmt.Errorf("non-retryable error: %w", err)
		}

		// 最后一次尝试不再等待
		if attempt == maxRetries {
			break
		}

		// 指数退避
		delay := baseDelay * time.Duration(math.Pow(2, float64(attempt-1)))
		us.logger.Info("waiting %v before retry (attempt %d/%d)", delay, attempt, maxRetries)
		time.Sleep(delay)
	}

	return fmt.Errorf("failed after %d attempts, last error: %w", maxRetries, lastErr)
}

// UploadFilesWithRetry 批量带重试的文件上传
func (us *uploadService) UploadFilesWithRetry(workspacePath string, filePaths []string, maxRetries int) ([]string, error) {
	us.logger.Info("starting batch upload for %d files in workspace %s", len(filePaths), workspacePath)

	if len(filePaths) == 0 {
		return []string{}, nil
	}

	// 使用配置中的最大重试次数或传入的参数
	actualMaxRetries := us.uploadCfg.MaxRetries
	if maxRetries > 0 {
		actualMaxRetries = maxRetries
	}

	var requestIds []string
	var uploadErrors []error

	for _, filePath := range filePaths {
		us.logger.Info("uploading file %s", filePath)

		requestId, err := us.UploadFileWithRetry(workspacePath, filePath, actualMaxRetries)
		if err != nil {
			us.logger.Error("failed to upload file %s: %v", filePath, err)
			uploadErrors = append(uploadErrors, fmt.Errorf("failed to upload file %s: %w", filePath, err))
			continue
		}

		requestIds = append(requestIds, requestId)
		us.logger.Info("file %s uploaded successfully with request ID: %s", filePath, requestId)
	}

	// 如果有上传错误，返回汇总错误
	if len(uploadErrors) > 0 {
		return requestIds, fmt.Errorf("batch upload completed with %d errors out of %d files. First error: %w",
			len(uploadErrors), len(filePaths), uploadErrors[0])
	}

	us.logger.Info("batch upload completed successfully for %d files", len(filePaths))
	return requestIds, nil
}

// DeleteFilesWithRetry 批量带重试的文件删除
func (us *uploadService) DeleteFilesWithRetry(workspacePath string, filePaths []string, maxRetries int) ([]string, error) {
	us.logger.Info("starting batch delete for %d files in workspace %s", len(filePaths), workspacePath)

	if len(filePaths) == 0 {
		return []string{}, nil
	}

	// 使用配置中的最大重试次数或传入的参数
	actualMaxRetries := us.uploadCfg.MaxRetries
	if maxRetries > 0 {
		actualMaxRetries = maxRetries
	}

	var requestIds []string
	var deleteErrors []error

	for _, filePath := range filePaths {
		us.logger.Info("deleting file %s", filePath)

		requestId, err := us.DeleteFileWithRetry(workspacePath, filePath, actualMaxRetries)
		if err != nil {
			us.logger.Error("failed to delete file %s: %v", filePath, err)
			deleteErrors = append(deleteErrors, fmt.Errorf("failed to delete file %s: %w", filePath, err))
			continue
		}

		requestIds = append(requestIds, requestId)
		us.logger.Info("file %s deleted successfully with request ID: %s", filePath, requestId)
	}

	// 如果有删除错误，返回汇总错误
	if len(deleteErrors) > 0 {
		return requestIds, fmt.Errorf("batch delete completed with %d errors out of %d files. First error: %w",
			len(deleteErrors), len(filePaths), deleteErrors[0])
	}

	us.logger.Info("batch delete completed successfully for %d files", len(filePaths))
	return requestIds, nil
}

// RenameFileWithRetry 带重试的文件重命名
func (us *uploadService) RenameFileWithRetry(workspacePath string, oldFilePath string, newFilePath string, maxRetries int) (string, error) {
	us.logger.Info("starting rename operation from %s to %s in workspace %s", oldFilePath, newFilePath, workspacePath)

	if !us.uploadCfg.EnableRetry {
		// 如果禁用重试，直接重命名一次
		return us.renameSingleFile(workspacePath, oldFilePath, newFilePath)
	}

	// 使用配置中的最大重试次数或传入的参数
	actualMaxRetries := us.uploadCfg.MaxRetries
	if maxRetries > 0 {
		actualMaxRetries = maxRetries
	}

	var lastErr error

	for attempt := 1; attempt <= actualMaxRetries; attempt++ {
		us.logger.Info("renaming file %s to %s (attempt %d/%d)", oldFilePath, newFilePath, attempt, actualMaxRetries)

		requestId, err := us.renameSingleFile(workspacePath, oldFilePath, newFilePath)
		if err == nil {
			us.logger.Info("file %s renamed to %s successfully", oldFilePath, newFilePath)
			return requestId, nil
		}

		lastErr = err
		us.logger.Warn("failed to rename file %s to %s (attempt %d/%d): %v", oldFilePath, newFilePath, attempt, actualMaxRetries, err)

		if attempt < actualMaxRetries {
			// 检查是否为可重试错误
			if !us.isRetryableError(err) {
				us.logger.Error("non-retryable error occurred for renaming file %s to %s: %v", oldFilePath, newFilePath, err)
				break
			}

			// 指数退避
			delay := us.uploadCfg.BaseRetryDelay * time.Duration(math.Pow(2, float64(attempt-1)))
			us.logger.Info("waiting %v before retry...", delay)
			time.Sleep(delay)
		}
	}

	return "", fmt.Errorf("failed to rename file %s to %s after %d attempts, last error: %w", oldFilePath, newFilePath, actualMaxRetries, lastErr)
}

// renameSingleFile 单文件重命名算法
func (us *uploadService) renameSingleFile(workspacePath string, oldFilePath string, newFilePath string) (string, error) {
	// 2. 验证新文件路径的目录是否存在
	newFullPath := filepath.Join(workspacePath, newFilePath)
	newDir := filepath.Dir(newFullPath)
	if _, err := os.Stat(newDir); os.IsNotExist(err) {
		return "", fmt.Errorf("target directory does not exist: %s", newDir)
	}

	// 3. 检查文件大小
	fileInfo, err := os.Stat(newFullPath)
	if err != nil {
		return "", fmt.Errorf("failed to get file info: %w", err)
	}

	fileSizeMB := float64(fileInfo.Size()) / (1024 * 1024)
	if fileSizeMB > float64(us.uploadCfg.FileSizeLimitMB) {
		return "", fmt.Errorf("file size %.2fMB exceeds limit %dMB", fileSizeMB, us.uploadCfg.FileSizeLimitMB)
	}

	// 4. 创建文件重命名对象
	fileStatus := &utils.FileStatus{
		Path:       oldFilePath,
		TargetPath: newFilePath,
		Status:     utils.FILE_STATUS_RENAME,
	}

	// 5. 创建临时的 codebase 配置
	codebaseConfig := &config.CodebaseConfig{
		ClientID:     us.config.ClientId,
		CodebaseId:   filepath.Base(workspacePath),
		CodebasePath: workspacePath,
		CodebaseName: filepath.Base(workspacePath),
		RegisterTime: time.Now(),
	}

	// 6. 创建ZIP文件
	zipPath, err := us.scheduler.CreateSingleFileZip(codebaseConfig, fileStatus)
	if err != nil {
		return "", fmt.Errorf("failed to create zip file: %w", err)
	}

	// 清理临时文件
	defer func() {
		if zipPath != "" {
			if err := os.Remove(zipPath); err != nil {
				us.logger.Warn("failed to delete temp zip file %s: %v", zipPath, err)
			} else {
				us.logger.Info("temp zip file deleted successfully: %s", zipPath)
			}
		}
	}()

	// 7. 获取上传令牌
	tokenReq := dto.UploadTokenReq{
		ClientId:     us.config.ClientId,
		CodebasePath: workspacePath,
		CodebaseName: filepath.Base(workspacePath),
	}

	tokenResp, err := us.syncer.FetchUploadToken(tokenReq)
	if err != nil {
		return "", fmt.Errorf("failed to fetch upload token: %w", err)
	}

	// 8. 上传文件
	requestId, err := utils.GenerateUUID()
	if err != nil {
		us.logger.Warn("failed to generate rename request ID, using timestamp: %v", err)
		requestId = time.Now().Format("20060102150405000")
	}
	uploadReq := dto.UploadReq{
		ClientId:     us.config.ClientId,
		CodebasePath: workspacePath,
		CodebaseName: filepath.Base(workspacePath),
		RequestId:    requestId,
		UploadToken:  tokenResp.Data.Token,
	}

	// 使用令牌上传文件
	originalToken := us.config.Token
	us.config.Token = tokenResp.Data.Token
	defer func() {
		us.config.Token = originalToken
	}()

	err = us.syncer.UploadFile(zipPath, uploadReq)
	if err != nil {
		return "", fmt.Errorf("failed to upload rename file: %w", err)
	}

	us.logger.Info("file %s renamed to %s successfully", oldFilePath, newFilePath)
	return requestId, nil
}

// RenameFilesWithRetry 批量带重试的文件重命名
func (us *uploadService) RenameFilesWithRetry(workspacePath string, renamePairs []utils.FileRenamePair, maxRetries int) ([]string, error) {
	us.logger.Info("starting batch rename for %d files in workspace %s", len(renamePairs), workspacePath)

	if len(renamePairs) == 0 {
		return []string{}, nil
	}

	// 使用配置中的最大重试次数或传入的参数
	actualMaxRetries := us.uploadCfg.MaxRetries
	if maxRetries > 0 {
		actualMaxRetries = maxRetries
	}

	var requestIds []string
	var renameErrors []error

	for _, renamePair := range renamePairs {
		us.logger.Info("renaming file %s to %s", renamePair.OldFilePath, renamePair.NewFilePath)

		requestId, err := us.RenameFileWithRetry(workspacePath, renamePair.OldFilePath, renamePair.NewFilePath, actualMaxRetries)
		if err != nil {
			us.logger.Error("failed to rename file %s to %s: %v", renamePair.OldFilePath, renamePair.NewFilePath, err)
			renameErrors = append(renameErrors, fmt.Errorf("failed to rename file %s to %s: %w", renamePair.OldFilePath, renamePair.NewFilePath, err))
			continue
		}

		requestIds = append(requestIds, requestId)
		us.logger.Info("file %s renamed to %s successfully with request ID: %s", renamePair.OldFilePath, renamePair.NewFilePath, requestId)
	}

	// 如果有重命名错误，返回汇总错误
	if len(renameErrors) > 0 {
		return requestIds, fmt.Errorf("batch rename completed with %d errors out of %d files. First error: %w",
			len(renameErrors), len(renamePairs), renameErrors[0])
	}

	us.logger.Info("batch rename completed successfully for %d files", len(renamePairs))
	return requestIds, nil
}
