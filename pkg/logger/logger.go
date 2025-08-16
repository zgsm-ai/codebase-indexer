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

// Log level mapping
var logLevelMap = map[string]zapcore.Level{
	"debug": zapcore.DebugLevel,
	"info":  zapcore.InfoLevel,
	"warn":  zapcore.WarnLevel,
	"error": zapcore.ErrorLevel,
}

// Logger interface
type Logger interface {
	Debug(format string, args ...any)
	Info(format string, args ...any)
	Warn(format string, args ...any)
	Error(format string, args ...any)
	Fatal(format string, args ...any)
}

// Logger implementation
type logger struct {
	log   *zap.Logger
	sugar *zap.SugaredLogger
}

// NewLogger Create new logger instance
func NewLogger(logsDir, level string) (Logger, error) {
	// Ensure logs directory is valid and writable
	if logsDir == "" || strings.Contains(logsDir, "\x00") {
		return nil, fmt.Errorf("invalid log directory path")
	}

	// Try to create directory to verify write permission
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %v", err)
	}

	testFile := filepath.Join(logsDir, ".test-write")
	file, err := os.Create(testFile)
	if err != nil {
		return nil, fmt.Errorf("directory is not writable: %v", err)
	}
	file.Close()
	os.Remove(testFile)

	// Configure log output to both file and console
	fileWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   filepath.Join(logsDir, "codebase-indexer.log"),
		MaxSize:    10, // megabytes
		MaxBackups: 1,  // backups
		MaxAge:     7,  // days
		Compress:   false,
		LocalTime:  true,
	})

	// Setup log encoder configuration
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000"),
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Set the log level
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

	zapLogger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	sugar := zapLogger.Sugar()

	return &logger{
		log:   zapLogger,
		sugar: sugar,
	}, nil
}

// Debug level log
func (l *logger) Debug(format string, args ...any) {
	l.sugar.Debugf(format, args...)
}

// Info level log
func (l *logger) Info(format string, args ...any) {
	l.sugar.Infof(format, args...)
}

// Warning level log
func (l *logger) Warn(format string, args ...any) {
	l.sugar.Warnf(format, args...)
}

// Error level log
func (l *logger) Error(format string, args ...any) {
	l.sugar.Errorf(format, args...)
}

// Fatal error log
func (l *logger) Fatal(format string, args ...any) {
	l.sugar.Fatalf(format, args...)
}
