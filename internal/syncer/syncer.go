// syncer/syncer.go - HTTP sync implementation
package syncer

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

	"codebase-indexer/internal/storage"
	"codebase-indexer/internal/utils"
	"codebase-indexer/pkg/logger"

	"github.com/valyala/fasthttp"
)

// Server API paths
const (
	API_UPLOAD_FILE       = "/codebase-indexer/api/v1/files/upload"
	API_GET_CODEBASE_HASH = "/codebase-indexer/api/v1/codebases/hash"
)

type SyncConfig struct {
	ClientId  string
	Token     string
	ServerURL string
}

type SyncInterface interface {
	SetSyncConfig(config *SyncConfig)
	GetSyncConfig() *SyncConfig
	FetchServerHashTree(codebasePath string) (map[string]string, error)
	UploadFile(codebasePath string, uploadRe *UploadReq) error
	GetClientConfig() (storage.ClientConfig, error)
}

type HTTPSync struct {
	syncConfig *SyncConfig
	httpClient *fasthttp.Client
	logger     logger.Logger
	rwMutex    sync.RWMutex
}

func NewHTTPSync(syncConfig *SyncConfig, logger logger.Logger) SyncInterface {
	return &HTTPSync{
		syncConfig: syncConfig,
		httpClient: &fasthttp.Client{
			MaxIdleConnDuration: 90 * time.Second,
			ReadTimeout:         60 * time.Second,
			WriteTimeout:        utils.BaseWriteTimeoutSeconds * time.Second,
			MaxConnsPerHost:     500,
		},
		logger: logger,
	}
}

// Calculate dynamic timeout (in seconds)
func (hs *HTTPSync) calculateTimeout(fileSize int64) time.Duration {
	fileSizeMB := float64(fileSize) / (1024 * 1024)
	baseTimeout := utils.BaseWriteTimeoutSeconds * time.Second

	// Files ≤10MB use fixed 60s timeout
	if fileSizeMB <= 10 {
		return baseTimeout
	}

	// Files >10MB: 60s + (file size MB - 10)*5s
	totalTimeout := baseTimeout + time.Duration(fileSizeMB-10)*5*time.Second

	// Maximum does not exceed 10 minutes
	if totalTimeout > 600*time.Second {
		return 600 * time.Second
	}
	return totalTimeout
}

func (hs *HTTPSync) SetSyncConfig(config *SyncConfig) {
	hs.rwMutex.Lock()
	defer hs.rwMutex.Unlock()
	hs.syncConfig = config
}

func (hs *HTTPSync) GetSyncConfig() *SyncConfig {
	hs.rwMutex.RLock()
	defer hs.rwMutex.RUnlock()
	return hs.syncConfig
}

type CodebaseHashReq struct {
	ClientId     string `json:"clientId"`
	CodebasePath string `json:"codebasePath"`
}

type CodebaseHashResp struct {
	Code    int                  `json:"code"`
	Message string               `json:"message"`
	Data    CodebaseHashRespData `json:"data"`
}

type CodebaseHashRespData struct {
	List []HashItem `json:"list"`
}

type HashItem struct {
	Path string `json:"path"`
	Hash string `json:"hash"`
}

// Fetch server hash tree
func (hs *HTTPSync) FetchServerHashTree(codebasePath string) (map[string]string, error) {
	hs.logger.Info("fetching hash tree from server: %s", codebasePath)

	// Check if config fields are empty
	if hs.syncConfig == nil || hs.syncConfig.ServerURL == "" || hs.syncConfig.ClientId == "" || hs.syncConfig.Token == "" {
		return nil, fmt.Errorf("sync config is not properly set, please check clientId, serverURL and token")
	}

	// Prepare the request
	url := fmt.Sprintf("%s%s?clientId=%s&codebasePath=%s",
		hs.syncConfig.ServerURL, API_GET_CODEBASE_HASH, hs.syncConfig.ClientId, codebasePath)

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer func() {
		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(resp)
	}()

	req.SetRequestURI(url)
	req.Header.SetMethod("GET")
	req.Header.SetContentType("application/json")
	req.Header.Set("Authorization", "Bearer "+hs.syncConfig.Token)

	hs.logger.Info("sending hash tree request to: %s", url)
	if err := hs.httpClient.Do(req, resp); err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}

	// Process the response
	if resp.StatusCode() != fasthttp.StatusOK {
		return nil, fmt.Errorf("failed to get hash tree, status: %d, response: %s",
			resp.StatusCode(), string(resp.Body()))
	}

	var responseData CodebaseHashResp
	if err := json.Unmarshal(resp.Body(), &responseData); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}

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

type UploadReq struct {
	ClientId     string `json:"clientId"`
	CodebasePath string `json:"codebasePath"`
	CodebaseName string `json:"codebaseName"`
}

// UploadFile uploads file to server
func (hs *HTTPSync) UploadFile(filePath string, uploadReq *UploadReq) error {
	mu := sync.Mutex{}
	hs.logger.Info("uploading file: %s", filePath)

	// Check if config fields are empty
	if hs.syncConfig == nil || hs.syncConfig.ServerURL == "" || hs.syncConfig.ClientId == "" || hs.syncConfig.Token == "" {
		return fmt.Errorf("sync config is not properly set, please check clientId, serverURL and token")
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
	if fileSize > 100*1024*1024 {
		return fmt.Errorf("file size exceeds 100MB")
	}

	// 设置动态超时
	timeout := hs.calculateTimeout(fileSize)
	mu.Lock()
	hs.httpClient.WriteTimeout = timeout
	mu.Unlock()

	body := &bytes.Buffer{}
	counter := &writeCounter{}
	startTime := time.Now()
	defer func() {
		mu.Lock()
		duration := time.Since(startTime)
		mu.Unlock()
		hs.logger.Info("upload stats - file: %s, size: %d bytes, uploaded: %d bytes (%.1f%%), duration: %v, speed: %.2f KB/s",
			filePath, fileSize, counter.n, float64(counter.n)/float64(fileSize)*100, duration, float64(counter.n)/1024/duration.Seconds())
	}()
	writer := multipart.NewWriter(body)

	// Add zip file
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("failed to create form file: %v", err)
	}

	if _, err := io.Copy(io.MultiWriter(part, counter), file); err != nil {
		return fmt.Errorf("failed to copy file content: %v", err)
	}

	// Add form fields
	writer.WriteField("clientId", uploadReq.ClientId)
	writer.WriteField("codebasePath", uploadReq.CodebasePath)
	writer.WriteField("codebaseName", uploadReq.CodebaseName)

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %v", err)
	}

	url := fmt.Sprintf("%s%s", hs.syncConfig.ServerURL, API_UPLOAD_FILE)

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer func() {
		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(resp)
	}()

	req.SetRequestURI(url)
	req.Header.SetMethod("POST")
	req.Header.SetContentType(writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+hs.syncConfig.Token)
	req.SetBody(body.Bytes())

	hs.logger.Info("sending file upload request to: %s", url)
	if err := hs.httpClient.Do(req, resp); err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}

	if resp.StatusCode() != fasthttp.StatusOK {
		return fmt.Errorf("upload failed, status: %d, response: %s", resp.StatusCode(), string(resp.Body()))
	}

	hs.logger.Info("file uploaded successfully: %s", filePath)
	return nil
}

// Client config file URI
const (
	API_GET_CLIENT_CONFIG = "/shenma/api/v1/config/%scodebase-indexer-config.json"
)

// Get client configuration
func (hs *HTTPSync) GetClientConfig() (storage.ClientConfig, error) {
	hs.logger.Info("fetching client config from server")

	// Check if config fields are empty
	if hs.syncConfig == nil || hs.syncConfig.ServerURL == "" || hs.syncConfig.ClientId == "" || hs.syncConfig.Token == "" {
		return storage.ClientConfig{}, fmt.Errorf("sync config is not properly set, please check clientId, serverURL and token")
	}

	uri := fmt.Sprintf(API_GET_CLIENT_CONFIG, "")
	appInfo := storage.GetAppInfo()
	if appInfo.Version != "" {
		uri = fmt.Sprintf(API_GET_CLIENT_CONFIG, appInfo.Version+"/")
	}

	// Prepare the request
	url := fmt.Sprintf("%s%s", hs.syncConfig.ServerURL, uri)

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer func() {
		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(resp)
	}()

	req.SetRequestURI(url)
	req.Header.SetMethod("GET")
	req.Header.SetContentType("application/json")
	req.Header.Set("Authorization", "Bearer "+hs.syncConfig.Token)

	hs.logger.Info("sending client config request to: %s", url)
	if err := hs.httpClient.Do(req, resp); err != nil {
		return storage.ClientConfig{}, fmt.Errorf("failed to send request: %v", err)
	}

	// Process the response
	if resp.StatusCode() != fasthttp.StatusOK {
		return storage.ClientConfig{}, fmt.Errorf("failed to get client config, status: %d, response: %s",
			resp.StatusCode(), string(resp.Body()))
	}

	var clientConfig storage.ClientConfig
	if err := json.Unmarshal(resp.Body(), &clientConfig); err != nil {
		return storage.ClientConfig{}, fmt.Errorf("failed to parse response: %v", err)
	}

	hs.logger.Info("client config fetched successfully")
	return clientConfig, nil
}
