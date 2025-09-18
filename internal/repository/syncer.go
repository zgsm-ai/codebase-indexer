// syncer/syncer.go - HTTP sync implementation
package repository

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"codebase-indexer/internal/config"
	"codebase-indexer/internal/dto"
	"codebase-indexer/internal/utils"
	"codebase-indexer/pkg/logger"
)

// Server API paths
const (
	API_UPLOAD_TOKEN      = "/codebase-embedder/api/v1/files/token"
	API_UPLOAD_FILE       = "/codebase-embedder/api/v1/files/upload"
	API_FILE_STATUS       = "/codebase-embedder/api/v1/files/status"
	API_GET_CODEBASE_HASH = "/codebase-embedder/api/v1/codebases/hash"
	API_DELETE_EMBEDDING  = "/codebase-embedder/api/v1/embeddings"
)

type SyncInterface interface {
	SetSyncConfig(config *config.SyncConfig)
	GetSyncConfig() *config.SyncConfig
	FetchServerHashTree(codebasePath string) (map[string]string, error)
	UploadFile(filePath string, uploadReq dto.UploadReq) error
	GetClientConfig() (config.ClientConfig, error)
	FetchUploadToken(req dto.UploadTokenReq) (*dto.UploadTokenResp, error)
	FetchFileStatus(req dto.FileStatusReq) (*dto.FileStatusResp, error)
	DeleteEmbedding(req dto.DeleteEmbeddingReq) (*dto.DeleteEmbeddingResp, error)
}

type HTTPSync struct {
	syncConfig *config.SyncConfig
	httpClient *utils.HTTPClient
	logger     logger.Logger
	rwMutex    sync.RWMutex
}

func NewHTTPSync(syncConfig *config.SyncConfig, logger logger.Logger) SyncInterface {
	return &HTTPSync{
		syncConfig: syncConfig,
		httpClient: utils.NewHTTPClient(),
		logger:     logger,
	}
}

// Calculate dynamic timeout (in seconds)
func (hs *HTTPSync) calculateTimeout(fileSize int64) time.Duration {
	fileSizeMB := float64(fileSize) / (1024 * 1024)
	baseTimeout := utils.BaseWriteTimeoutSeconds * time.Second

	// FileMeta ≤5MB use fixed 60s timeout
	if fileSizeMB <= 5 {
		return baseTimeout
	}

	// FileMeta >5MB: 60s + (file size MB - 5)*5s
	totalTimeout := baseTimeout + time.Duration(fileSizeMB-5)*5*time.Second

	// Maximum does not exceed 10 minutes
	if totalTimeout > 600*time.Second {
		return 600 * time.Second
	}
	return totalTimeout
}

// ValidateSyncConfig 验证同步配置
func (hs *HTTPSync) ValidateSyncConfig(authInfo config.AuthInfo) error {
	if authInfo.ServerURL == "" {
		return fmt.Errorf("serverURL is empty")
	}

	if authInfo.ClientId == "" {
		return fmt.Errorf("clientId is empty")
	}

	if authInfo.Token == "" {
		return fmt.Errorf("token is empty")
	}

	return nil
}

func (hs *HTTPSync) SetSyncConfig(config *config.SyncConfig) {
	hs.rwMutex.Lock()
	defer hs.rwMutex.Unlock()
	hs.syncConfig = config
}

func (hs *HTTPSync) GetSyncConfig() *config.SyncConfig {
	hs.rwMutex.RLock()
	defer hs.rwMutex.RUnlock()
	return hs.syncConfig
}

// Fetch server hash tree
func (hs *HTTPSync) FetchServerHashTree(codebasePath string) (map[string]string, error) {
	hs.logger.Info("fetching hash tree from server: %s", codebasePath)

	// 验证配置
	authInfo := config.GetAuthInfo()
	if err := hs.ValidateSyncConfig(authInfo); err != nil {
		return nil, err
	}

	// 构建请求URL
	url := fmt.Sprintf("%s%s", authInfo.ServerURL, API_GET_CODEBASE_HASH)

	// 构建查询参数
	queryParams := map[string]string{
		"clientId":     authInfo.ClientId,
		"codebasePath": codebasePath,
	}

	// 执行请求
	var responseData dto.CodebaseHashResp
	hs.logger.Info("sending HTTP %s request to: %s", "GET", url)
	startTime := time.Now()
	if err := hs.httpClient.DoGetRequest(url, queryParams, authInfo.Token, &responseData); err != nil {
		return nil, err
	}
	duration := time.Since(startTime)
	hs.logger.Info("HTTP %s %s completed in %v, status: %d", "GET", url, duration, responseData.Code)

	// 处理响应数据
	hashTree := make(map[string]string)
	for _, item := range responseData.Data.List {
		path := item.Path
		if runtime.GOOS == "windows" {
			path = filepath.FromSlash(path)
		}
		hashTree[path] = item.Hash
	}

	hs.logger.Info("successfully fetched server hash tree, contains %d files", len(hashTree))
	return hashTree, nil
}

type writeCounter struct {
	n int64
}

func (wc *writeCounter) Write(p []byte) (int, error) {
	wc.n += int64(len(p))
	return len(p), nil
}

// UploadFile uploads file to server
func (hs *HTTPSync) UploadFile(filePath string, uploadReq dto.UploadReq) error {
	hs.logger.Info("uploading file: %s", filePath)

	// 验证配置
	authInfo := config.GetAuthInfo()
	if err := hs.ValidateSyncConfig(authInfo); err != nil {
		return err
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to get file info: %v", err)
	}
	fileSize := fileInfo.Size()
	// TODO: Temporarily hardcode file size limit, will be changed to remote configuration management in the future
	if fileSize >= 100*1024*1024 {
		return fmt.Errorf("file size exceeds 100MB")
	}

	// 设置动态超时
	timeout := hs.calculateTimeout(fileSize)

	counter := &writeCounter{}
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		hs.logger.Info("upload stats - file: %s, size: %d bytes, uploaded: %d bytes (%.1f%%), duration: %v, speed: %.2f KB/s",
			filePath, fileSize, counter.n, float64(counter.n)/float64(fileSize)*100, duration, float64(counter.n)/1024/duration.Seconds())
	}()

	// 构建multipart表单数据
	formData := &utils.MultipartFormData{
		Files: map[string]*utils.MultipartFile{
			"file": {
				FileName: filepath.Base(filePath),
				Reader:   file, // 直接使用文件读取器
			},
		},
		Fields: map[string]string{
			"clientId":     uploadReq.ClientId,
			"codebasePath": uploadReq.CodebasePath,
			"codebaseName": uploadReq.CodebaseName,
			"uploadToken":  uploadReq.UploadToken,
		},
	}

	// 执行上传请求
	url := fmt.Sprintf("%s%s", authInfo.ServerURL, API_UPLOAD_FILE)

	// 创建带有超时的HTTP请求
	httpReq := &utils.HTTPRequest{
		Method:      "POST",
		URL:         url,
		Timeout:     timeout,
		ContentType: "multipart/form-data",
		Headers: map[string]string{
			"X-Request-ID": uploadReq.RequestId,
		},
	}

	// 使用自定义执行方法处理multipart请求
	hs.logger.Info("sending HTTP %s request to: %s", "POST", url)
	if err := hs.executeMultipartUpload(httpReq, formData, file, counter, authInfo.Token); err != nil {
		return err
	}

	hs.logger.Info("file uploaded successfully: %s", filePath)
	return nil
}

// executeMultipartUpload 执行multipart上传
func (hs *HTTPSync) executeMultipartUpload(httpReq *utils.HTTPRequest, formData *utils.MultipartFormData, file io.Reader, counter *writeCounter, token string) error {
	// 创建multipart表单
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// 添加文件
	for fieldName, fileData := range formData.Files {
		part, err := writer.CreateFormFile(fieldName, fileData.FileName)
		if err != nil {
			return fmt.Errorf("failed to create form file: %v", err)
		}

		// 使用多写入器同时写入计数器和表单
		multiWriter := io.MultiWriter(part, counter)
		if _, err := io.Copy(multiWriter, file); err != nil {
			return fmt.Errorf("failed to copy file content: %v", err)
		}
	}

	// 添加普通字段
	for fieldName, value := range formData.Fields {
		if err := writer.WriteField(fieldName, value); err != nil {
			return fmt.Errorf("failed to write field: %v", err)
		}
	}

	// 关闭writer
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %v", err)
	}

	// 设置请求体和内容类型
	httpReq.Body = body.Bytes()
	httpReq.ContentType = writer.FormDataContentType()

	// 执行请求
	resp, err := hs.httpClient.DoHTTPRequest(httpReq, token)
	if err != nil {
		return err
	}

	// 检查响应状态
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("upload failed, status: %d, response: %s", resp.StatusCode, string(resp.Body))
	}

	return nil
}

// Client config file URI
const (
	API_GET_CLIENT_CONFIG = "/costrict/codebase-indexer/config/%scodebase-indexer-config.json"
)

// Value client configuration
func (hs *HTTPSync) GetClientConfig() (config.ClientConfig, error) {
	hs.logger.Info("fetching client config from server")

	// 验证配置
	authInfo := config.GetAuthInfo()
	if err := hs.ValidateSyncConfig(authInfo); err != nil {
		return config.ClientConfig{}, err
	}

	// 构建请求URL
	uri := fmt.Sprintf(API_GET_CLIENT_CONFIG, "")
	appInfo := config.GetAppInfo()
	if appInfo.Version != "" {
		uri = fmt.Sprintf(API_GET_CLIENT_CONFIG, appInfo.Version+"/")
	}

	url := fmt.Sprintf("%s%s", authInfo.ServerURL, uri)

	// 执行请求
	var clientConfig config.ClientConfig
	hs.logger.Info("sending HTTP %s request to: %s", "GET", url)
	startTime := time.Now()
	if err := hs.httpClient.DoGetRequest(url, nil, authInfo.Token, &clientConfig); err != nil {
		return config.ClientConfig{}, err
	}
	duration := time.Since(startTime)
	hs.logger.Info("HTTP %s %s completed in %v", "GET", url, duration)

	return clientConfig, nil
}

// FetchUploadToken fetches upload token from server
func (hs *HTTPSync) FetchUploadToken(req dto.UploadTokenReq) (*dto.UploadTokenResp, error) {
	hs.logger.Info("fetching upload token from server")

	// 验证配置
	authInfo := config.GetAuthInfo()
	if err := hs.ValidateSyncConfig(authInfo); err != nil {
		return nil, err
	}

	// 构建请求URL
	url := fmt.Sprintf("%s%s", authInfo.ServerURL, API_UPLOAD_TOKEN)

	// 执行JSON请求
	var responseData dto.UploadTokenResp
	hs.logger.Info("sending HTTP %s request to: %s", "GET", url)
	startTime := time.Now()
	if err := hs.httpClient.DoJSONRequest("POST", url, req, authInfo.Token, &responseData); err != nil {
		return nil, err
	}
	duration := time.Since(startTime)
	hs.logger.Info("HTTP %s %s completed in %v, status: %d", "POST", url, duration, responseData.Code)

	return &responseData, nil
}

// FetchFileStatus fetches file status from server
func (hs *HTTPSync) FetchFileStatus(req dto.FileStatusReq) (*dto.FileStatusResp, error) {
	hs.logger.Info("fetching file status from server")

	// 验证配置
	authInfo := config.GetAuthInfo()
	if err := hs.ValidateSyncConfig(authInfo); err != nil {
		return nil, err
	}

	// 构建请求URL
	url := fmt.Sprintf("%s%s", authInfo.ServerURL, API_FILE_STATUS)

	// 执行JSON请求
	var responseData dto.FileStatusResp
	hs.logger.Info("sending HTTP %s request to: %s", "POST", url)
	startTime := time.Now()
	if err := hs.httpClient.DoJSONRequest("POST", url, req, authInfo.Token, &responseData); err != nil {
		return nil, err
	}
	duration := time.Since(startTime)
	hs.logger.Info("HTTP %s %s completed in %v, status: %d", "POST", url, duration, responseData.Code)

	return &responseData, nil
}

// DeleteEmbedding deletes embedding from server
func (hs *HTTPSync) DeleteEmbedding(req dto.DeleteEmbeddingReq) (*dto.DeleteEmbeddingResp, error) {
	hs.logger.Info("deleting embedding from server")

	// 验证配置
	authInfo := config.GetAuthInfo()
	if err := hs.ValidateSyncConfig(authInfo); err != nil {
		return nil, err
	}

	// 构建请求URL
	url := fmt.Sprintf("%s%s", authInfo.ServerURL, API_DELETE_EMBEDDING)

	// 构建查询参数
	queryParams := map[string]string{
		"clientId":     req.ClientId,
		"codebasePath": req.CodebasePath,
	}

	// 如果有文件路径参数，添加到查询参数中
	if len(req.FilePaths) > 0 {
		// 对于多个文件路径，可能需要特殊处理，这里简单地将第一个文件路径作为参数
		// 实际使用中可能需要根据API规范调整
		queryParams["filePaths"] = req.FilePaths[0]
	}

	// 创建HTTP请求
	httpReq := &utils.HTTPRequest{
		Method:      "DELETE",
		URL:         url,
		QueryParams: queryParams,
		Headers:     map[string]string{},
	}

	// 执行请求
	hs.logger.Info("sending HTTP %s request to: %s", "DELETE", url)
	startTime := time.Now()
	resp, err := hs.httpClient.DoHTTPRequest(httpReq, authInfo.Token)
	if err != nil {
		return nil, err
	}
	duration := time.Since(startTime)

	// 检查响应状态
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("delete embedding failed, status: %d, response: %s", resp.StatusCode, string(resp.Body))
	}

	// 解析响应数据
	var responseData dto.DeleteEmbeddingResp
	if err := json.Unmarshal(resp.Body, &responseData); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

	hs.logger.Info("HTTP %s %s completed in %v, status: %d", "DELETE", url, duration, responseData.Code)
	return &responseData, nil
}
