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

	"codebase-syncer/pkg/logger"

	"github.com/valyala/fasthttp"
)

// 服务端API路径
const (
	API_UPLOAD_FILE   = "/codebase-indexer/api/v1/files/upload"
	API_GET_HASH_TREE = "/codebase-indexer/api/v1/comparison"
)

type SyncConfig struct {
	ClientId  string
	Token     string
	ServerURL string
}

type HTTPSync struct {
	syncConfig *SyncConfig
	httpClient *fasthttp.Client
	logger     logger.Logger
}

func NewHTTPSync(logger logger.Logger) *HTTPSync {
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

type ComparisonReq struct {
	ClientId     string `json:"clientId"`
	CodebasePath string `json:"codebasePath"`
}

type ComparisonResp struct {
	Code    int                `json:"code"`
	Message string             `json:"message"`
	Data    ComparisonRespData `json:"data"`
}

type ComparisonRespData struct {
	CodebaseTree []TreeItem `json:"codebaseTree"`
}

type TreeItem struct {
	Path string `json:"path"`
	Hash string `json:"hash"`
}

// 获取服务端哈希树
func (hs *HTTPSync) FetchServerHashTree(codebasePath string) (map[string]string, error) {
	hs.logger.Info("从服务器获取哈希树: %s", codebasePath)

	// 准备请求
	url := fmt.Sprintf("%s%s?clientId=%s&codebasePath=%s",
		hs.syncConfig.ServerURL, API_GET_HASH_TREE, hs.syncConfig.ClientId, codebasePath)

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer func() {
		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(resp)
	}()

	req.SetRequestURI(url)
	req.Header.SetMethod("GET")
	req.Header.SetContentType("application/json")
	req.Header.SetCookie("Authorization", "Bearer "+hs.syncConfig.Token)

	hs.logger.Debug("发送获取哈希树请求到: %s", url)
	if err := hs.httpClient.Do(req, resp); err != nil {
		return nil, fmt.Errorf("发送请求失败: %v", err)
	}

	// 处理响应
	if resp.StatusCode() != fasthttp.StatusOK {
		return nil, fmt.Errorf("获取哈希树失败，状态码: %d，响应: %s",
			resp.StatusCode(), string(resp.Body()))
	}

	var responseData ComparisonResp
	if err := json.Unmarshal(resp.Body(), &responseData); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	hashTree := make(map[string]string)
	for _, item := range responseData.Data.CodebaseTree {
		path := item.Path
		if runtime.GOOS == "windows" {
			path = filepath.FromSlash(path)
		}
		hashTree[path] = item.Hash
	}

	hs.logger.Info("成功获取服务器哈希树，包含 %d 个文件", len(hashTree))
	return hashTree, nil
}

type UploadReq struct {
	ClientId     string `json:"clientId"`
	CodebasePath string `json:"codebasePath"`
	CodebaseName string `json:"codebaseName"`
}

// UploadFile 上传文件到服务器
func (hs *HTTPSync) UploadFile(filePath string, uploadReq *UploadReq) error {
	hs.logger.Info("上传文件: %s", filePath)

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("打开zip文件失败: %v", err)
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// 添加zip文件
	part, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("创建表单文件失败: %v", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return fmt.Errorf("复制文件内容失败: %v", err)
	}

	// 添加表单字段
	writer.WriteField("clientId", uploadReq.ClientId)
	writer.WriteField("codebasePath", uploadReq.CodebasePath)
	writer.WriteField("codebaseName", uploadReq.CodebaseName)

	if err := writer.Close(); err != nil {
		return fmt.Errorf("关闭写入器失败: %v", err)
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
	req.Header.SetCookie("Authorization", "Bearer "+hs.syncConfig.Token)
	req.SetBody(body.Bytes())

	hs.logger.Debug("发送文件上传请求到: %s", url)
	if err := hs.httpClient.Do(req, resp); err != nil {
		return fmt.Errorf("发送请求失败: %v", err)
	}

	if resp.StatusCode() != fasthttp.StatusOK {
		return fmt.Errorf("上传失败，状态码: %d，响应: %s", resp.StatusCode(), string(resp.Body()))
	}

	hs.logger.Info("文件上传成功: %s", filePath)
	return nil
}
