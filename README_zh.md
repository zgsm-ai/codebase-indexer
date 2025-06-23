# codebase-syncer

## 项目介绍
codebase-syncer 是一个代码库同步客户端工具，用于将本地代码变更同步到远程服务端。通过文件哈希值比对实现增量同步，支持定时同步和主动同步，支持远程配置管理，支持多平台运行。

## 功能特性

- 文件哈希比对：使用SHA256算法比对文件差异
- 增量同步：仅上传变更文件
- 配置管理：支持本地和服务端配置
- 多平台支持：Windows/Linux/macOS

## 架构设计

### 核心模块

1. **Scanner** - 文件扫描器
   - 递归扫描代码目录
   - 计算文件哈希值
   - 应用.gitignore规则
   - 检测文件变更

2. **Syncer** - 同步器
   - HTTP文件上传
   - 获取服务端文件哈希树
   - 增量同步管理

3. **Storage** - 存储管理
   - 本地缓存管理
   - 配置持久化
   - 文件元数据存储

4. **Scheduler** - 任务调度
   - 定时扫描触发
   - 同步任务队列管理
   - 重试机制

5. **Daemon** - 守护进程
   - gRPC服务管理
   - 优雅启停控制
   - 系统信号处理

### 接口定义

1. **gRPC接口** (定义在api/codebase_syncer.proto)
   - 注册项目同步 (RegisterSync)
   - 同步项目 (SyncCodebase)
   - 注销项目同步 (UnregisterSync)
   - 共享AccessToken (ShareAccessToken)
   - 获取应用版本信息 (GetVersion)

2. **HTTP接口** (定义在internal/syncer/syncer.go)
   - 获取服务端文件哈希树 (FetchServerHashTree)
   - 上传文件到服务器 (UploadFile)
   - 获取客户端配置文件 (GetClientConfig)

## 使用说明

### 启动参数

```sh
./codebase-syncer \
  --appname myapp \          # 应用名称
  --grpc localhost:51353 \   # gRPC服务地址
  --loglevel info \          # 日志级别
  --clientid CLIENT_ID \     # 客户端ID
  --server SERVER_URL \      # 服务端地址
  --token AUTH_TOKEN \       # 认证令牌
```

### 配置文件

示例配置：
```json
{
  "server": {
    "registerExpireMinutes": 30,
    "hashTreeExpireHours": 24
  },
  "sync": {
    "intervalMinutes": 3,
    "maxFileSizeMB": 1,
    "maxRetries": 3,
    "retryDelaySeconds": 5,
    "ignorePatterns": [
     ".*",
     "*.swp", "*.swo",
     "*.pyc", "*.class", "*.o", "*.obj",
     "*.log", "*.tmp", "*.bak", "*.backup",
     "*.exe", "*.dll", "*.so", "*.dylib",
     "logs/", "temp/", "tmp/", "node_modules/",
     "bin/", "dist/", "build/",
     "__pycache__/", "venv/", "target/"
    ]
  }
}

```

### 构建运行

构建项目：
```sh
./scripts/build.sh ${os} ${arch} ${version}
```

一键打包：
```sh
./scripts/package_all.sh ${version}
```

## 开发指南

### 依赖管理

```sh
go mod tidy
```

### 代码生成

更新gRPC协议：
```sh
protoc --go_out=api --go-grpc_out=api api/codebase_syncer.proto
```

### 测试运行

单元测试：
```sh
go test ./...
```

集成测试：
```sh
go test -tags=integration ./test
```