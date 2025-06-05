package logger

import (
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestNewLogger(t *testing.T) {
	t.Run("创建日志目录成功", func(t *testing.T) {
		tempDir := t.TempDir()
		_, err := NewLogger(tempDir, "debug")
		if err != nil {
			t.Fatalf("创建日志失败: %v", err)
		}
	})

	t.Run("日志级别验证", func(t *testing.T) {
		// 使用observer测试日志级别
		observedCore, observedLogs := observer.New(zapcore.InfoLevel)
		logger := zap.New(observedCore).Sugar()

		logger.Debug("debug message") // 不应记录
		logger.Info("info message")   // 应记录

		logs := observedLogs.All()
		if len(logs) != 1 {
			t.Errorf("期望1条日志记录，实际: %d", len(logs))
		}
		if logs[0].Message != "info message" {
			t.Errorf("日志消息不匹配: %s", logs[0].Message)
		}
	})

	t.Run("无法创建日志目录返回错误", func(t *testing.T) {
		rootDir := t.TempDir()
		// 将 cacheDir 设置为一个文件的路径，而不是目录
		fileAsCacheDirPath := filepath.Join(rootDir, "thisIsAFileNotADirectory")
		if err := os.WriteFile(fileAsCacheDirPath, []byte("I am a file"), 0644); err != nil {
			t.Fatalf("创建用作cacheDir的文件失败: %v", err)
		}

		// 尝试创建日志，应该返回错误
		_, err := NewLogger(fileAsCacheDirPath, "debug")
		if err == nil {
			t.Error("期望返回错误，因为cacheDir是一个文件")
		}
	})

	t.Run("日志目录无效返回错误", func(t *testing.T) {
		_, err := NewLogger("", "warn")
		if err == nil {
			t.Error("应该返回日志目录无效的错误")
		}
	})
}
