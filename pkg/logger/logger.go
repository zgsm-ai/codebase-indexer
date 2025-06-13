package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// 日志级别映射
var logLevelMap = map[string]zapcore.Level{
	"debug": zapcore.DebugLevel,
	"info":  zapcore.InfoLevel,
	"warn":  zapcore.WarnLevel,
	"error": zapcore.ErrorLevel,
}

// 日志接口
type Logger interface {
	Debug(format string, args ...any)
	Info(format string, args ...any)
	Warn(format string, args ...any)
	Error(format string, args ...any)
	Fatal(format string, args ...any)
}

// 日志实现
type logger struct {
	log   *zap.Logger
	sugar *zap.SugaredLogger
}

// 创建新日志实例
func NewLogger(logsDir, level string) (Logger, error) {
	// 确保logs目录是有效的可写路径
	if logsDir == "" || strings.Contains(logsDir, "\x00") {
		return nil, fmt.Errorf("日志目录路径无效")
	}

	// 尝试创建目录来验证是否有写入权限
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return nil, fmt.Errorf("无法创建日志目录: %v", err)
	}

	testFile := filepath.Join(logsDir, ".test-write")
	file, err := os.Create(testFile)
	if err != nil {
		return nil, fmt.Errorf("目录不可写: %v", err)
	}
	file.Close()
	os.Remove(testFile)

	// 设置日志输出到文件和控制台
	fileWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   filepath.Join(logsDir, "codebase-syncer.log"),
		MaxSize:    50, // megabytes
		MaxBackups: 0,  //
		MaxAge:     30, // days
		Compress:   true,
		LocalTime:  true,
	})

	// 配置日志编码器
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 设置日志级别
	logLevel, exists := logLevelMap[strings.ToLower(level)]
	if !exists {
		logLevel = zapcore.InfoLevel
	}

	core := zapcore.NewTee(
		zapcore.NewCore(
			zapcore.NewConsoleEncoder(encoderConfig),
			zapcore.AddSync(os.Stdout),
			logLevel,
		),
		zapcore.NewCore(
			zapcore.NewConsoleEncoder(encoderConfig),
			fileWriter,
			logLevel,
		),
	)

	zapLogger := zap.New(core)
	sugar := zapLogger.Sugar()

	return &logger{
		log:   zapLogger,
		sugar: sugar,
	}, nil
}

// 调试级日志
func (l *logger) Debug(format string, args ...any) {
	l.sugar.Debugf(format, args...)
}

// 信息级日志
func (l *logger) Info(format string, args ...any) {
	l.sugar.Infof(format, args...)
}

// 警告级日志
func (l *logger) Warn(format string, args ...any) {
	l.sugar.Warnf(format, args...)
}

// 错误级日志
func (l *logger) Error(format string, args ...any) {
	l.sugar.Errorf(format, args...)
}

// 致命错误日志
func (l *logger) Fatal(format string, args ...any) {
	l.sugar.Fatalf(format, args...)
}
