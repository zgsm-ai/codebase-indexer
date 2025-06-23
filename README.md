# codebase-syncer

## Project Introduction
A codebase sync client tool for synchronizing local code changes to remote servers. Implements incremental sync using file hash comparison, supports scheduled sync and manual sync, remote configuration management, and multi-platform operation.

## Features

- File hash comparison: Uses SHA256 algorithm to compare file differences
- Incremental sync: Only uploads changed files
- Configuration management: Supports both local and server-side configurations
- Multi-platform support: Windows/Linux/macOS

## Architecture Design

### Core Modules

1. **Scanner** - File scanner
   - Recursively scans code directories
   - Calculates file hashes 
   - Applies .gitignore rules
   - Detects file changes

2. **Syncer** - Synchronizer
   - HTTP file upload
   - Gets server file hash tree  
   - Manages incremental sync

3. **Storage** - Storage management
   - Local cache management
   - Configuration persistence
   - File metadata storage

4. **Scheduler** - Task scheduling
   - Triggers scheduled scans
   - Manages sync task queue
   - Retry mechanism

5. **Daemon** - Daemon process  
   - gRPC service management
   - Graceful start/stop control
   - System signal handling

### Interface Definitions

1. **gRPC Interfaces** (defined in api/codebase_syncer.proto)
   - Register project sync (RegisterSync)
   - Sync codebase (SyncCodebase)
   - Unregister project sync (UnregisterSync)
   - Share access token (ShareAccessToken)
   - Get version info (GetVersion)

2. **HTTP Interfaces** (defined in internal/syncer/syncer.go)
   - Fetch server hash tree (FetchServerHashTree)
   - Upload file to server (UploadFile)
   - Get client config (GetClientConfig)

## Usage Instructions

### Startup Parameters

```sh
./codebase-syncer \
  --appname myapp \          # Application name
  --grpc localhost:51353 \   # gRPC service address
  --loglevel info \          # Log level
  --clientid CLIENT_ID \     # Client ID
  --server SERVER_URL \      # Server URL
  --token AUTH_TOKEN \       # Authentication token
```

### Configuration File

Example configuration:
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

### Build & Run

Build project:
```sh
./scripts/build.sh ${os} ${arch} ${version}
```

Package all:
```sh
./scripts/package_all.sh ${version}
```

## Development Guide

### Dependency Management

```sh
go mod tidy
```

### Code Generation

Update gRPC protocol:
```sh
protoc --go_out=api --go-grpc_out=api api/codebase_syncer.proto
```

### Testing

Unit tests:
```sh
go test ./...
```

Integration tests:  
```sh
go test -tags=integration ./test