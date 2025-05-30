package logger

import (
	"codebase-syncer/pkg/utils"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	Debug(format string, args ...interface{})
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
	Fatal(format string, args ...interface{})
}

// 日志实现
type logger struct {
	log   *zap.Logger
	sugar *zap.SugaredLogger
}

// 创建新日志实例
func NewLogger(level string) Logger {
	// 生成按日期命名的日志文件
	currentDate := time.Now().Format("20060102")
	logFileName := filepath.Join(utils.LogsDir, fmt.Sprintf("codebase-syncer-%s.log", currentDate))

	// 设置日志输出到文件和控制台
	fileWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   logFileName,
		MaxSize:    100, // megabytes
		MaxBackups: 0,   //
		MaxAge:     30,  // days
		Compress:   true,
		LocalTime:  true,
	})

	// 配置日志编码器
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000"),
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
			zapcore.NewJSONEncoder(encoderConfig),
			zapcore.AddSync(os.Stdout),
			logLevel,
		),
		zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderConfig),
			fileWriter,
			logLevel,
		),
	)

	zapLogger := zap.New(core, zap.AddCaller())
	sugar := zapLogger.Sugar()

	return &logger{
		log:   zapLogger,
		sugar: sugar,
	}
}

// 调试级日志
func (l *logger) Debug(format string, args ...interface{}) {
	l.sugar.Debugf(format, args...)
}

// 信息级日志
func (l *logger) Info(format string, args ...interface{}) {
	l.sugar.Infof(format, args...)
}

// 警告级日志
func (l *logger) Warn(format string, args ...interface{}) {
	l.sugar.Warnf(format, args...)
}

// 错误级日志
func (l *logger) Error(format string, args ...interface{}) {
	l.sugar.Errorf(format, args...)
}

// 致命错误日志
func (l *logger) Fatal(format string, args ...interface{}) {
	l.sugar.Fatalf(format, args...)
}
