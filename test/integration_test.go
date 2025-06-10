package test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	api "codebase-syncer/api"
	"codebase-syncer/internal/handler"
	"codebase-syncer/internal/scanner"
	"codebase-syncer/internal/scheduler"
	"codebase-syncer/internal/storage"
	"codebase-syncer/internal/syncer"
	"codebase-syncer/internal/utils"
	"codebase-syncer/pkg/logger"
	"codebase-syncer/test/mocks"
)

type IntegrationTestSuite struct {
	suite.Suite
	handler   *handler.GRPCHandler
	scheduler *scheduler.Scheduler
}

var httpSync = new(mocks.MockHTTPSync)
var appInfo = &handler.AppInfo{
	AppName:  "test-app",
	OSName:   "windows",
	ArchName: "amd64",
	Version:  "1.0.0",
}

func (s *IntegrationTestSuite) SetupTest() {
	// 使用真实对象进行测试
	rootPath := os.TempDir()
	logPath, err := utils.GetLogDir(rootPath)
	if err != nil {
		s.T().Fatalf("获取log目录失败: %v", err)
	}
	fmt.Printf("log目录: %s\n", logPath)

	// 初始化缓存目录
	cachePath, err := utils.GetCacheDir(rootPath)
	if err != nil {
		s.T().Fatalf("获取缓存目录失败: %v", err)
	}
	fmt.Printf("缓存目录: %s\n", cachePath)

	// 初始化上报临时目录
	uploadTmpPath, err := utils.GetUploadTmpDir(rootPath)
	if err != nil {
		s.T().Fatalf("获取上报临时目录失败: %v", err)
	}
	fmt.Printf("上报临时目录: %s\n", uploadTmpPath)

	logger, err := logger.NewLogger(logPath, "info")
	if err != nil {
		s.T().Fatalf("初始化日志系统失败: %v", err)
	}
	storageManager, err := storage.NewStorageManager(cachePath, logger)
	if err != nil {
		s.T().Fatalf("初始化存储系统失败: %v", err)
	}
	fileScanner := scanner.NewFileScanner(logger)
	s.handler = handler.NewGRPCHandler(httpSync, storageManager, logger, appInfo)
	s.scheduler = scheduler.NewScheduler(httpSync, fileScanner, storageManager, logger)
}

func (s *IntegrationTestSuite) TestRegisterSync() {
	registerPath := filepath.Join(os.TempDir(), "register-test")
	tests := []struct {
		name    string
		req     *api.RegisterSyncRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: &api.RegisterSyncRequest{
				ClientId:      "client1",
				WorkspacePath: registerPath,
				WorkspaceName: "register-test",
			},
			wantErr: false,
		},
		{
			name: "missing client id",
			req: &api.RegisterSyncRequest{
				WorkspacePath: registerPath,
				WorkspaceName: "register-test",
			},
			wantErr: true,
		},
		{
			name: "empty workspace path",
			req: &api.RegisterSyncRequest{
				ClientId:      "client1",
				WorkspacePath: "",
				WorkspaceName: "register-test",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			resp, err := s.handler.RegisterSync(context.Background(), tt.req)

			if tt.wantErr {
				assert.NoError(t, err)
				assert.Contains(t, resp.Message, "参数错误")
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
			}
		})
	}
}

func (s *IntegrationTestSuite) TestUnregisterSync() {
	workspaceDir := filepath.Join(os.TempDir(), "unregister-test")
	err := os.MkdirAll(workspaceDir, 0755)
	assert.NoError(s.T(), err)
	defer os.RemoveAll(workspaceDir)
	// 1. 先注册工作区
	registerReq := &api.RegisterSyncRequest{
		ClientId:      "test-client",
		WorkspacePath: workspaceDir,
		WorkspaceName: "unregister-test",
	}
	_, err = s.handler.RegisterSync(context.Background(), registerReq)
	assert.NoError(s.T(), err)

	// 2. 正常注销
	req := &api.UnregisterSyncRequest{
		ClientId:      "test-client",
		WorkspacePath: workspaceDir,
		WorkspaceName: "unregister-test",
	}
	_, err = s.handler.UnregisterSync(context.Background(), req)
	assert.NoError(s.T(), err)
}

func (s *IntegrationTestSuite) TestUnregisterSyncInvalidParams() {
	// 测试缺少必要参数的情况
	testCases := []struct {
		name string
		req  *api.UnregisterSyncRequest
	}{
		{
			name: "缺少ClientId",
			req: &api.UnregisterSyncRequest{
				WorkspacePath: "/tmp/test",
				WorkspaceName: "test",
			},
		},
		{
			name: "缺少WorkspacePath",
			req: &api.UnregisterSyncRequest{
				ClientId:      "test-client",
				WorkspaceName: "test",
			},
		},
		{
			name: "缺少WorkspaceName",
			req: &api.UnregisterSyncRequest{
				ClientId:      "test-client",
				WorkspacePath: "/tmp/test",
			},
		},
	}

	for _, tc := range testCases {
		s.T().Run(tc.name, func(t *testing.T) {
			_, err := s.handler.UnregisterSync(context.Background(), tc.req)
			assert.Error(t, err)
		})
	}
}

func (s *IntegrationTestSuite) TestHandlerVersion() {
	tests := []struct {
		name     string
		clientId string
		wantErr  bool
	}{
		{
			name:     "normal case",
			clientId: "client1",
			wantErr:  false,
		},
		{
			name:     "empty client id",
			clientId: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			req := &api.VersionRequest{
				ClientId: tt.clientId,
			}

			resp, err := s.handler.GetVersion(context.Background(), req)

			if tt.wantErr {
				assert.NoError(t, err)
				assert.Contains(t, resp.Message, "参数错误")
			} else {
				assert.NoError(t, err)
				assert.Equal(s.T(), "test-app", resp.Data.AppName)
				assert.Equal(s.T(), "1.0.0", resp.Data.Version)
			}
		})
	}
}

func (s *IntegrationTestSuite) TestTokenSharing() {
	req := &api.ShareAccessTokenRequest{
		ClientId:       "test-client",
		ServerEndpoint: "http://test.server",
		AccessToken:    "test-token",
	}

	resp, err := s.handler.ShareAccessToken(context.Background(), req)
	assert.NoError(s.T(), err)
	assert.True(s.T(), resp.Success)
}

func (s *IntegrationTestSuite) TestFullIntegrationFlow() {
	httpSync.On("SetSyncConfig", mock.Anything).Return()
	httpSync.On("GetSyncConfig", mock.Anything).Return(&syncer.SyncConfig{})
	httpSync.On("FetchServerHashTree", mock.Anything).Return(map[string]string{}, nil)
	httpSync.On("Sync", mock.Anything, mock.Anything).Return(nil)
	// 提前创建工作区目录
	workspaceDir := filepath.Join(os.TempDir(), "test-workspace")
	err := os.MkdirAll(workspaceDir, 0755)
	assert.NoError(s.T(), err)
	// 1. Register workspace
	registerReq := &api.RegisterSyncRequest{
		ClientId:      "test-client",
		WorkspacePath: workspaceDir,
		WorkspaceName: "test-workspace",
	}
	registerResp, err := s.handler.RegisterSync(context.Background(), registerReq)
	assert.NoError(s.T(), err)
	assert.True(s.T(), registerResp.Success)

	// 2. Set token
	tokenReq := &api.ShareAccessTokenRequest{
		ClientId:       "test-client",
		ServerEndpoint: "http://test.server",
		AccessToken:    "test-token",
	}
	_, err = s.handler.ShareAccessToken(context.Background(), tokenReq)
	assert.NoError(s.T(), err)

	// 3. Start scheduler and verify sync
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go s.scheduler.Start(ctx)

	// Wait for scheduler to run
	time.Sleep(1 * time.Second)

	// 4. Unregister workspace
	unregisterReq := &api.UnregisterSyncRequest{
		ClientId:      "test-client",
		WorkspacePath: workspaceDir,
		WorkspaceName: "test-workspace",
	}
	_, err = s.handler.UnregisterSync(context.Background(), unregisterReq)
	assert.NoError(s.T(), err)
}

func (s *IntegrationTestSuite) TestSchedulerOperations() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 测试调度器是否可以正常启动和停止
	go s.scheduler.Start(ctx)
	time.Sleep(100 * time.Millisecond)
}

func TestIntegrationSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
