// cmd/main.go - 程序入口
package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	api "codebase-syncer/api" // 已生成的protobuf包
	"codebase-syncer/internal/daemon"
	"codebase-syncer/internal/handler"
	"codebase-syncer/internal/scanner"
	"codebase-syncer/internal/scheduler"
	"codebase-syncer/internal/storage"
	"codebase-syncer/internal/syncer"
	"codebase-syncer/internal/utils"
	"codebase-syncer/pkg/logger"

	"google.golang.org/grpc"
)

var (
	// version will be set by the linker during build
	osName   string
	archName string
	version  string
)

func main() {
	if osName != "" {
		fmt.Printf("OS: %s\n", osName)
	}
	if archName != "" {
		fmt.Printf("Arch: %s\n", archName)
	}
	if version != "" {
		fmt.Printf("Version: %s\n", version)
	}

	// 解析命令行参数
	appName := flag.String("appname", "zgsm", "应用名称")
	grpcServer := flag.String("grpc", "localhost:51353", "gRPC服务器地址")
	logLevel := flag.String("loglevel", "info", "日志级别 (debug, info, warn, error)")
	flag.Parse()

	// 初始化目录
	if err := initDir(*appName); err != nil {
		fmt.Printf("初始化目录失败: %v\n", err)
		return
	}

	// 初始化日志系统
	logger, err := logger.NewLogger(utils.LogsDir, *logLevel)
	if err != nil {
		fmt.Printf("初始化日志系统失败: %v\n", err)
		return
	}
	logger.Info("OS: %s, Arch: %s, App: %s, Version: %s, 启动中...", osName, archName, *appName, version)

	// 初始化各模块
	storageManager, err := storage.NewStorageManager(utils.CacheDir, logger)
	if err != nil {
		logger.Fatal("初始化存储管理器失败: %v", err)
		return
	}
	fileScanner := scanner.NewFileScanner(logger)
	httpSync := syncer.NewHTTPSync(logger)
	appInfo := &handler.AppInfo{AppName: *appName, ArchName: archName, OSName: osName, Version: version}
	grpcHandler := handler.NewGRPCHandler(httpSync, storageManager, logger, appInfo)
	syncScheduler := scheduler.NewScheduler(httpSync, fileScanner, storageManager, logger)

	// 初始化gRPC服务端
	lis, err := net.Listen("tcp", *grpcServer)
	if err != nil {
		logger.Fatal("failed to listen: %v", err)
		return
	}
	s := grpc.NewServer()
	api.RegisterSyncServiceServer(s, grpcHandler)

	// 启动守护进程
	daemon := daemon.NewDaemon(syncScheduler, s, lis, httpSync, logger)
	go daemon.Start()

	// 处理系统信号，优雅退出
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	<-signals

	logger.Info("接收到退出信号，正在优雅关闭...")
	daemon.Stop()
	logger.Info("客户端已成功关闭")
}

// initDir 初始化目录
func initDir(appName string) error {
	// 初始化根目录
	rootPath, err := utils.GetRootDir(appName)
	if err != nil {
		fmt.Printf("获取根目录失败: %v\n", err)
		return fmt.Errorf("获取根目录失败: %v", err)
	}
	fmt.Printf("根目录: %s\n", rootPath)

	// 初始化日志目录
	logPath, err := utils.GetLogDir(rootPath)
	if err != nil {
		fmt.Printf("获取log目录失败: %v\n", err)
		return fmt.Errorf("获取log目录失败: %v", err)
	}
	fmt.Printf("log目录: %s\n", logPath)

	// 初始化缓存目录
	cachePath, err := utils.GetCacheDir(rootPath)
	if err != nil {
		fmt.Printf("获取缓存目录失败: %v\n", err)
		return fmt.Errorf("获取缓存目录失败: %v", err)
	}
	fmt.Printf("缓存目录: %s\n", cachePath)

	// 初始化上报临时目录
	uploadTmpPath, err := utils.GetUploadTmpDir(rootPath)
	if err != nil {
		fmt.Printf("获取上报临时目录失败: %v\n", err)
		return fmt.Errorf("获取上报临时目录失败: %v", err)
	}
	fmt.Printf("上报临时目录: %s\n", uploadTmpPath)

	return nil
}
