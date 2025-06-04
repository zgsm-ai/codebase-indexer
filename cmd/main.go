// cmd/main.go - 程序入口
package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	interval = 1 * time.Minute // 同步间隔
)

func main() {
	// 解析命令行参数
	appName := flag.String("appname", "zgsm", "应用名称")
	grpcServer := flag.String("grpc", "localhost:50051", "gRPC服务器地址")
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
	logger.Info("客户端启动中...")

	// 初始化各模块
	storageManager := storage.NewStorageManager(utils.CacheDir, logger)
	fileScanner := scanner.NewFileScanner(logger)
	httpSync := syncer.NewHTTPSync(logger)
	grpcHandler := handler.NewGRPCHandler(httpSync, storageManager, logger)
	syncScheduler := scheduler.NewScheduler(interval, httpSync, fileScanner, storageManager, logger)

	// 启动gRPC服务端
	lis, err := net.Listen("tcp", *grpcServer)
	if err != nil {
		logger.Fatal("failed to listen: %v", err)
		return
	}
	s := grpc.NewServer()
	api.RegisterSyncServiceServer(s, grpcHandler)
	go func() {
		logger.Info("启动gRPC服务端，监听地址: %s", *grpcServer)
		if err := s.Serve(lis); err != nil {
			logger.Fatal("failed to serve: %v", err)
			return
		}
	}()

	// 启动同步守护进程
	daemon := daemon.NewDaemon(syncScheduler, grpcHandler, logger)
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
