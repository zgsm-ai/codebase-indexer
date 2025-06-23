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
	t.Run("Successfully create log directory", func(t *testing.T) {
		tempDir := t.TempDir()
		_, err := NewLogger(tempDir, "debug")
		if err != nil {
			t.Fatalf("Failed to create log: %v", err)
		}
	})

	t.Run("Log level validation", func(t *testing.T) {
		// Use observer to test log levels
		observedCore, observedLogs := observer.New(zapcore.InfoLevel)
		logger := zap.New(observedCore).Sugar()

		logger.Debug("debug message") // Should not be recorded
		logger.Info("info message")   // Should be recorded

		logs := observedLogs.All()
		if len(logs) != 1 {
			t.Errorf("Expected 1 log record, got: %d", len(logs))
		}
		if logs[0].Message != "info message" {
			t.Errorf("Log message mismatch: %s", logs[0].Message)
		}
	})

	t.Run("Failed to create log directory returns error", func(t *testing.T) {
		rootDir := t.TempDir()
		// Set cacheDir as file path instead of directory
		fileAsCacheDirPath := filepath.Join(rootDir, "thisIsAFileNotADirectory")
		if err := os.WriteFile(fileAsCacheDirPath, []byte("I am a file"), 0644); err != nil {
			t.Fatalf("Failed to create file for cacheDir: %v", err)
		}

		// Try to create log, should return error
		_, err := NewLogger(fileAsCacheDirPath, "debug")
		if err == nil {
			t.Error("Expected error since cacheDir is a file")
		}
	})

	t.Run("Invalid log directory returns error", func(t *testing.T) {
		_, err := NewLogger("", "warn")
		if err == nil {
			t.Error("Should return invalid log directory error")
		}
	})
}
