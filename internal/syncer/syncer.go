// syncer/syncer.go - HTTP同步实现
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
	"time"

	"codebase-syncer/internal/storage"
	"codebase-syncer/pkg/logger"

	"github.com/valyala/fasthttp"
)

// 服务端API路径
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
}

func NewHTTPSync(logger logger.Logger) SyncInterface {
	return &HTTPSync{
		httpClient: &fasthttp.Client{
			ReadTimeout:  60 * time.Second,
			WriteTimeout: 60 * time.Second,
		},
		logger: logger,
	}
}

func (hs *HTTPSync) SetSyncConfig(config *SyncConfig) {
	hs.syncConfig = config
}

func (hs *HTTPSync) GetSyncConfig() *SyncConfig {
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

// 获取服务端哈希树
func (hs *HTTPSync) FetchServerHashTree(codebasePath string) (map[string]string, error) {
	hs.logger.Info("fetching hash tree from server: %s", codebasePath)

	// 准备请求
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

	hs.logger.Debug("sending hash tree request to: %s", url)
	if err := hs.httpClient.Do(req, resp); err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}

	// 处理响应
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

type UploadReq struct {
	ClientId     string `json:"clientId"`
	CodebasePath string `json:"codebasePath"`
	CodebaseName string `json:"codebaseName"`
}

// UploadFile 上传文件到服务器
func (hs *HTTPSync) UploadFile(filePath string, uploadReq *UploadReq) error {
	hs.logger.Info("uploading file: %s", filePath)

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// 添加zip文件
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("failed to create form file: %v", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return fmt.Errorf("failed to copy file content: %v", err)
	}

	// 添加表单字段
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

	hs.logger.Debug("sending file upload request to: %s", url)
	if err := hs.httpClient.Do(req, resp); err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}

	if resp.StatusCode() != fasthttp.StatusOK {
		return fmt.Errorf("upload failed, status: %d, response: %s", resp.StatusCode(), string(resp.Body()))
	}

	hs.logger.Info("file uploaded successfully: %s", filePath)
	return nil
}

// 客户端配置文件URI
const (
	API_GET_CLIENT_CONFIG = "/codebaseSyncer_cli_tools/config.json"
)

// 获取客户端配置文件
func (hs *HTTPSync) GetClientConfig() (storage.ClientConfig, error) {
	hs.logger.Info("fetching client config from server")

	// 准备请求
	url := fmt.Sprintf("%s%s", hs.syncConfig.ServerURL, API_GET_CLIENT_CONFIG)

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

	hs.logger.Debug("sending client config request to: %s", url)
	if err := hs.httpClient.Do(req, resp); err != nil {
		return storage.ClientConfig{}, fmt.Errorf("failed to send request: %v", err)
	}

	// 处理响应
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
