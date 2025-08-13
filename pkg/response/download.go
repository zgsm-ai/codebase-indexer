package response

import (
	"net/http"
	"path/filepath"
	"sync"

	"github.com/gin-gonic/gin"
)

// Downloader 用于维护分片下载状态
type Downloader struct {
	c        *gin.Context
	fileName string
	isFirst  bool
	mu       sync.Mutex
}

// NewDownloader 创建一个新的分片下载器
// c: Gin上下文
// fileName: 下载的文件名
func NewDownloader(c *gin.Context, fileName string) *Downloader {
	return &Downloader{
		c:        c,
		fileName: fileName,
		isFirst:  true,
	}
}

// Write 写入一部分数据（可多次调用）
func (d *Downloader) Write(data []byte) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// 第一次写入时设置响应头
	if d.isFirst {
		d.c.Header("Content-Type", "application/octet-stream")
		d.c.Header("Content-Disposition", "attachment; filename="+filepath.Base(d.fileName))
		d.c.Header("Transfer-Encoding", "chunked") // 使用分块传输编码
		d.c.Writer.WriteHeader(http.StatusOK)
		d.isFirst = false
	}

	// 写入当前分片数据
	if _, err := d.c.Writer.Write(data); err != nil {
		return err
	}

	// 刷新缓冲区
	d.c.Writer.Flush()
	return nil
}

// Finish 完成下载（可选，用于最后清理）
func (d *Downloader) Finish() {
	// 分块传输在最后会自动添加结束标记
	// 这里可以添加额外的清理逻辑
}
