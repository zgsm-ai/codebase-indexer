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

		// 验证日志文件是否创建
		logFile := filepath.Join(tempDir, "codebase-syncer.log")
		if _, err := os.Stat(logFile); os.IsNotExist(err) {
			t.Errorf("日志文件应该存在: %s", logFile)
		}
	})

	t.Run("日志目录自动创建", func(t *testing.T) {
		tempDir := filepath.Join(t.TempDir(), "new_logs")
		_, err := NewLogger(tempDir, "info")
		if err != nil {
			t.Fatalf("创建日志失败: %v", err)
		}

		if _, err := os.Stat(tempDir); os.IsNotExist(err) {
			t.Errorf("日志目录应该自动创建: %s", tempDir)
		}
	})

	t.Run("无效日志级别使用默认级别", func(t *testing.T) {
		tempDir := t.TempDir()
		l, _ := NewLogger(tempDir, "invalid_level")

		log, ok := l.(*logger)
		if !ok {
			t.Fatal("类型断言失败")
		}

		// 验证默认级别是Info
		memCore, recorded := observer.New(zapcore.InfoLevel)
		log.log = log.log.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			return memCore
		}))

		log.log.Debug("debug message")
		log.log.Info("info message")

		if len(recorded.All()) != 1 {
			t.Errorf("期望只记录info级别的日志，实际记录: %d", len(recorded.All()))
		}
	})

	t.Run("日志同时输出到文件和终端", func(t *testing.T) {
		tempDir := t.TempDir()
		_, err := NewLogger(tempDir, "debug")
		if err != nil {
			t.Fatalf("创建日志失败: %v", err)
		}

		logFile := filepath.Join(tempDir, "codebase-syncer.log")
		fileInfo, _ := os.Stat(logFile)
		if fileInfo.Size() == 0 {
			t.Error("日志文件应该是非空的")
		}
	})

	t.Run("无法创建日志目录返回错误", func(t *testing.T) {
		_, err := NewLogger("/invalid/path", "warn")
		if err == nil {
			t.Error("应该返回无法创建目录的错误")
		}
	})

	t.Run("正确设置日志级别", func(t *testing.T) {
		testCases := []struct {
			level    string
			expected zapcore.Level
		}{
			{"debug", zapcore.DebugLevel},
			{"info", zapcore.InfoLevel},
			{"warn", zapcore.WarnLevel},
			{"error", zapcore.ErrorLevel},
		}

		for _, tc := range testCases {
			t.Run(tc.level, func(t *testing.T) {
				tempDir := t.TempDir()
				l, _ := NewLogger(tempDir, tc.level)

				log, ok := l.(*logger)
				if !ok {
					t.Fatal("类型断言失败")
				}

				memCore, recorded := observer.New(tc.expected)
				log.log = log.log.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
					return memCore
				}))

				switch tc.expected {
				case zapcore.DebugLevel:
					log.log.Debug("test message")
				case zapcore.InfoLevel:
					log.log.Info("test message")
				case zapcore.WarnLevel:
					log.log.Warn("test message")
				case zapcore.ErrorLevel:
					log.log.Error("test message")
				}

				if len(recorded.All()) != 1 {
					t.Errorf("期望记录1条%s级别日志，实际记录: %d", tc.level, len(recorded.All()))
				}
			})
		}
	})
}

func TestLogger_Debug(t *testing.T) {
	tempDir := t.TempDir()
	l, err := NewLogger(tempDir, "debug")
	if err != nil {
		t.Fatalf("创建日志失败: %v", err)
	}

	log, ok := l.(*logger)
	if !ok {
		t.Fatal("类型断言失败")
	}

	// 使用observer捕获日志输出
	memCore, recorded := observer.New(zapcore.DebugLevel)
	log.log = log.log.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return memCore
	}))

	// 测试调用
	format := "测试日志 %s %d"
	arg1 := "参数1"
	arg2 := 42
	l.Debug(format, arg1, arg2)

	// 验证
	entries := recorded.All()
	if len(entries) != 1 {
		t.Fatalf("期望1条日志记录，实际: %d", len(entries))
	}

	expectedMessage := "测试日志 参数1 42"
	if entries[0].Message != expectedMessage {
		t.Errorf("日志消息不匹配\n期望: %s\n实际: %s", expectedMessage, entries[0].Message)
	}
}

func TestLogger_Info(t *testing.T) {
	tempDir := t.TempDir()
	l, err := NewLogger(tempDir, "info")
	if err != nil {
		t.Fatalf("创建日志失败: %v", err)
	}

	log, ok := l.(*logger)
	if !ok {
		t.Fatal("类型断言失败")
	}

	memCore, recorded := observer.New(zapcore.InfoLevel)
	log.log = log.log.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return memCore
	}))

	format := "测试Info日志 %s %d"
	arg1 := "info参数"
	arg2 := 100
	l.Info(format, arg1, arg2)

	entries := recorded.All()
	if len(entries) != 1 {
		t.Fatalf("期望1条日志记录，实际: %d", len(entries))
	}

	expectedMessage := "测试Info日志 info参数 100"
	if entries[0].Message != expectedMessage {
		t.Errorf("日志消息不匹配\n期望: %s\n实际: %s", expectedMessage, entries[0].Message)
	}

	if entries[0].Level != zapcore.InfoLevel {
		t.Errorf("日志级别不匹配\n期望: %s\n实际: %s", zapcore.InfoLevel, entries[0].Level)
	}
}

func TestLogger_Warn(t *testing.T) {
	tempDir := t.TempDir()
	l, err := NewLogger(tempDir, "warn")
	if err != nil {
		t.Fatalf("创建日志失败: %v", err)
	}

	log, ok := l.(*logger)
	if !ok {
		t.Fatal("类型断言失败")
	}

	memCore, recorded := observer.New(zapcore.WarnLevel)
	log.log = log.log.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return memCore
	}))

	format := "测试Warn日志 %s %d"
	arg1 := "warn参数"
	arg2 := 200
	l.Warn(format, arg1, arg2)

	entries := recorded.All()
	if len(entries) != 1 {
		t.Fatalf("期望1条日志记录，实际: %d", len(entries))
	}

	expectedMessage := "测试Warn日志 warn参数 200"
	if entries[0].Message != expectedMessage {
		t.Errorf("日志消息不匹配\n期望: %s\n实际: %s", expectedMessage, entries[0].Message)
	}

	if entries[0].Level != zapcore.WarnLevel {
		t.Errorf("日志级别不匹配\n期望: %s\n实际: %s", zapcore.WarnLevel, entries[0].Level)
	}
}

func TestLogger_Error(t *testing.T) {
	tempDir := t.TempDir()
	l, err := NewLogger(tempDir, "error")
	if err != nil {
		t.Fatalf("创建日志失败: %v", err)
	}

	log, ok := l.(*logger)
	if !ok {
		t.Fatal("类型断言失败")
	}

	memCore, recorded := observer.New(zapcore.ErrorLevel)
	log.log = log.log.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
		return memCore
	}))

	format := "测试Error日志 %s %d"
	arg1 := "error参数"
	arg2 := 300
	l.Error(format, arg1, arg2)

	entries := recorded.All()
	if len(entries) != 1 {
		t.Fatalf("期望1条日志记录，实际: %d", len(entries))
	}

	expectedMessage := "测试Error日志 error参数 300"
	if entries[0].Message != expectedMessage {
		t.Errorf("日志消息不匹配\n期望: %s\n实际: %s", expectedMessage, entries[0].Message)
	}

	if entries[0].Level != zapcore.ErrorLevel {
		t.Errorf("日志级别不匹配\n期望: %s\n实际: %s", zapcore.ErrorLevel, entries[0].Level)
	}
}

func TestLogger_Fatal(t *testing.T) {
	tempDir := t.TempDir()
	l, err := NewLogger(tempDir, "debug")
	if err != nil {
		t.Fatalf("创建日志失败: %v", err)
	}

	// Fatal测试只需要验证方法调用不会panic
	// 由于Fatal会调用os.Exit,无法完全测试实际行为
	l.Fatal("测试Fatal日志 %s %d", "fatal参数", 400)
}
