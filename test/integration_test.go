package test

import (
	"context"
	"crypto/md5"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	api "codebase-indexer/api"
	"codebase-indexer/internal/config"
	"codebase-indexer/internal/handler"
	"codebase-indexer/internal/repository"
	"codebase-indexer/internal/service"
	"codebase-indexer/internal/utils"
	"codebase-indexer/pkg/logger"
	"codebase-indexer/test/mocks"
)

type IntegrationTestSuite struct {
	suite.Suite
	handler   *handler.GRPCHandler
	scheduler *service.Scheduler
}

var httpSync = new(mocks.MockHTTPSync)
var appInfo = config.AppInfo{
	AppName:  "test-app",
	OSName:   "windows",
	ArchName: "amd64",
	Version:  "1.0.0",
}

func (s *IntegrationTestSuite) SetupTest() {
	// Use real objects for testing
	rootPath := os.TempDir()
	logPath, err := utils.GetLogDir(rootPath)
	if err != nil {
		s.T().Fatalf("failed to get log directory: %v", err)
	}
	fmt.Printf("log directory: %s\n", logPath)

	// Initialize cache directory
	cachePath, err := utils.GetCacheDir(rootPath, appInfo.AppName)
	if err != nil {
		s.T().Fatalf("failed to get cache directory: %v", err)
	}
	fmt.Printf("cache directory: %s\n", cachePath)

	// Initialize upload temporary directory
	uploadTmpPath, err := utils.GetCacheUploadTmpDir(cachePath)
	if err != nil {
		s.T().Fatalf("failed to get upload temp directory: %v", err)
	}
	fmt.Printf("upload temp directory: %s\n", uploadTmpPath)

	config.SetAppInfo(appInfo)

	logger, err := logger.NewLogger(logPath, "info", "codebase-indexer")
	if err != nil {
		s.T().Fatalf("failed to initialize logger: %v", err)
	}
	storageManager, err := repository.NewStorageManager(cachePath, logger)
	if err != nil {
		s.T().Fatalf("failed to initialize storage system: %v", err)
	}
	fileScanner := repository.NewFileScanner(logger)
	s.scheduler = service.NewScheduler(httpSync, fileScanner, storageManager, logger)
	s.handler = handler.NewGRPCHandler(httpSync, fileScanner, storageManager, s.scheduler, logger)
}

func (s *IntegrationTestSuite) TestRegisterSync() {
	registerPath := filepath.Join(os.TempDir(), "register-test")
	tests := []struct {
		name    string
		req     *api.RegisterSyncRequest
		wantErr bool
	}{
		{
			name: "register valid request",
			req: &api.RegisterSyncRequest{
				ClientId:      "client1",
				WorkspacePath: registerPath,
				WorkspaceName: "register-test",
			},
			wantErr: false,
		},
		{
			name: "register missing client id",
			req: &api.RegisterSyncRequest{
				WorkspacePath: registerPath,
				WorkspaceName: "register-test",
			},
			wantErr: true,
		},
		{
			name: "register empty workspace path",
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
				assert.Contains(t, resp.Message, "invalid parameters")
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
			}
		})
	}
}

func (s *IntegrationTestSuite) TestSyncCodebases() {
	// Prepare workspace directory
	workspaceDir := filepath.Join(os.TempDir(), "sync-codebases-test")
	err := os.MkdirAll(workspaceDir, 0755)
	assert.NoError(s.T(), err)
	defer os.RemoveAll(workspaceDir)

	httpSync.On("GetSyncConfig").Return(&config.SyncConfig{ClientId: "client1"}, nil)
	httpSync.On("FetchServerHashTree", mock.Anything).Return(map[string]string{}, nil)
	httpSync.On("UploadFile", mock.Anything, mock.Anything).Return(nil)

	tests := []struct {
		name    string
		req     *api.SyncCodebaseRequest
		wantErr bool
	}{
		{
			name: "sync valid request",
			req: &api.SyncCodebaseRequest{
				ClientId:      "client1",
				WorkspacePath: workspaceDir,
				WorkspaceName: "sync-codebases-test",
			},
			wantErr: false,
		},
		{
			name: "sync missing client id",
			req: &api.SyncCodebaseRequest{
				WorkspacePath: workspaceDir,
				WorkspaceName: "sync-codebases-test",
			},
			wantErr: true,
		},
		{
			name: "sync empty workspace path",
			req: &api.SyncCodebaseRequest{
				ClientId:      "client1",
				WorkspacePath: "",
				WorkspaceName: "sync-codebases-test",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			resp, err := s.handler.SyncCodebase(context.Background(), tt.req)

			if tt.wantErr {
				assert.NoError(t, err)
				assert.Contains(t, resp.Message, "invalid parameters")
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
	// 1. Register workspace first
	registerReq := &api.RegisterSyncRequest{
		ClientId:      "test-client",
		WorkspacePath: workspaceDir,
		WorkspaceName: "unregister-test",
	}
	_, err = s.handler.RegisterSync(context.Background(), registerReq)
	assert.NoError(s.T(), err)

	// 2. Normal unregistration
	req := &api.UnregisterSyncRequest{
		ClientId:      "test-client",
		WorkspacePath: workspaceDir,
		WorkspaceName: "unregister-test",
	}
	_, err = s.handler.UnregisterSync(context.Background(), req)
	assert.NoError(s.T(), err)
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
				assert.Contains(t, resp.Message, "invalid parameters")
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

func (s *IntegrationTestSuite) TestCheckIgnoreFile() {
	// Prepare workspace directory
	workspaceDir := filepath.Join(os.TempDir(), "check-ignore-test")
	err := os.MkdirAll(workspaceDir, 0755)
	assert.NoError(s.T(), err)
	defer os.RemoveAll(workspaceDir)

	// Create test files
	normalFile := filepath.Join(workspaceDir, "normal.txt")
	err = os.WriteFile(normalFile, []byte("normal content"), 0644)
	assert.NoError(s.T(), err)

	largeFile := filepath.Join(workspaceDir, "large.txt")
	err = os.WriteFile(largeFile, make([]byte, 110*1024), 0644) // 110KB
	assert.NoError(s.T(), err)

	ignoredFile := filepath.Join(workspaceDir, "ignored.txt")
	err = os.WriteFile(ignoredFile, []byte("ignored content"), 0644)
	assert.NoError(s.T(), err)

	// Create .gitignore file
	gitignoreFile := filepath.Join(workspaceDir, ".gitignore")
	err = os.WriteFile(gitignoreFile, []byte("ignored.txt\n"), 0644)
	assert.NoError(s.T(), err)

	tests := []struct {
		name        string
		req         *api.CheckIgnoreFileRequest
		wantCode    string
		wantMessage string
	}{
		{
			name: "invalid parameters",
			req: &api.CheckIgnoreFileRequest{
				ClientId:      "",
				WorkspacePath: "",
				WorkspaceName: "",
				FilePaths:     []string{},
			},
			wantCode:    "0001",
			wantMessage: "invalid parameters",
		},
		{
			name: "file size exceeded",
			req: &api.CheckIgnoreFileRequest{
				ClientId:      "test-client",
				WorkspacePath: workspaceDir,
				WorkspaceName: "check-ignore-test",
				FilePaths:     []string{largeFile},
			},
			wantCode:    "2001",
			wantMessage: "file size exceeded limit",
		},
		{
			name: "ignored file found",
			req: &api.CheckIgnoreFileRequest{
				ClientId:      "test-client",
				WorkspacePath: workspaceDir,
				WorkspaceName: "check-ignore-test",
				FilePaths:     []string{ignoredFile},
			},
			wantCode:    "2002",
			wantMessage: "ignore file found",
		},
		{
			name: "normal case",
			req: &api.CheckIgnoreFileRequest{
				ClientId:      "test-client",
				WorkspacePath: workspaceDir,
				WorkspaceName: "check-ignore-test",
				FilePaths:     []string{normalFile},
			},
			wantCode:    "0",
			wantMessage: "no ignored files found",
		},
	}

	for _, tt := range tests {
		s.T().Run(tt.name, func(t *testing.T) {
			resp, err := s.handler.CheckIgnoreFile(context.Background(), tt.req)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantCode, resp.Code)
			if tt.wantMessage != "" {
				assert.Contains(t, resp.Message, tt.wantMessage)
			}
		})
	}
}

func (s *IntegrationTestSuite) TestFullIntegrationFlow() {
	httpSync.On("SetSyncConfig", mock.Anything).Return()
	httpSync.On("GetSyncConfig", mock.Anything).Return(&config.SyncConfig{})
	httpSync.On("FetchServerHashTree", mock.Anything).Return(map[string]string{}, nil)
	httpSync.On("Sync", mock.Anything, mock.Anything).Return(nil)
	// Create workspace directory in advance
	workspaceDir := filepath.Join(os.TempDir(), "test-workspace")
	err := os.MkdirAll(workspaceDir, 0755)
	assert.NoError(s.T(), err)
	defer os.RemoveAll(workspaceDir)
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

	// Test if scheduler can start and stop normally
	go s.scheduler.Start(ctx)
	time.Sleep(100 * time.Millisecond)
}

func (s *IntegrationTestSuite) TestSyncForCodebases() {
	// Prepare test data
	ctx := context.Background()
	workspaceDir := filepath.Join(os.TempDir(), "sync-test")
	workspaceDir2 := filepath.Join(os.TempDir(), "sync-test2")
	err := os.MkdirAll(workspaceDir, 0755)
	assert.NoError(s.T(), err)
	err = os.MkdirAll(workspaceDir2, 0755)
	assert.NoError(s.T(), err)
	defer os.RemoveAll(workspaceDir)
	defer os.RemoveAll(workspaceDir2)

	// 1. Register test workspace
	registerReq := &api.RegisterSyncRequest{
		ClientId:      "test-client",
		WorkspacePath: workspaceDir,
		WorkspaceName: "sync-test",
	}
	_, err = s.handler.RegisterSync(ctx, registerReq)
	assert.NoError(s.T(), err)
	// Register second workspace
	registerReq2 := &api.RegisterSyncRequest{
		ClientId:      "test-client",
		WorkspacePath: workspaceDir2,
		WorkspaceName: "sync-test2",
	}
	_, err = s.handler.RegisterSync(ctx, registerReq2)
	assert.NoError(s.T(), err)

	// 2. Value codebase configuration
	codebaseConfigs := []*config.CodebaseConfig{
		{
			ClientID:     "test-client",
			CodebaseId:   fmt.Sprintf("%s_%x", "sync-test", md5.Sum([]byte(workspaceDir))),
			CodebaseName: "sync-test",
			CodebasePath: workspaceDir,
		},
		{
			ClientID:     "test-client",
			CodebaseId:   fmt.Sprintf("%s_%x", "sync-test2", md5.Sum([]byte(workspaceDir2))),
			CodebaseName: "sync-test2",
			CodebasePath: workspaceDir2,
		},
	}

	// 3. Test batch sync
	err = s.scheduler.SyncForCodebases(ctx, codebaseConfigs)
	assert.NoError(s.T(), err)

	// 4. Test empty configuration
	err = s.scheduler.SyncForCodebases(ctx, []*config.CodebaseConfig{})
	assert.NoError(s.T(), err)
}

func (s *IntegrationTestSuite) TestSyncForCodebasesWithContextCancellation() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	workspaceDir := filepath.Join(os.TempDir(), "sync-cancel-test")
	err := os.MkdirAll(workspaceDir, 0755)
	assert.NoError(s.T(), err)
	defer os.RemoveAll(workspaceDir)

	registerReq := &api.RegisterSyncRequest{
		ClientId:      "test-client",
		WorkspacePath: workspaceDir,
		WorkspaceName: "sync-cancel-test",
	}
	_, err = s.handler.RegisterSync(ctx, registerReq)
	assert.NoError(s.T(), err)

	codebaseConfigs := []*config.CodebaseConfig{
		{
			ClientID:     "test-client",
			CodebaseId:   fmt.Sprintf("%s_%x", "sync-cancel-test", md5.Sum([]byte(workspaceDir))),
			CodebaseName: "sync-cancel-test",
			CodebasePath: workspaceDir,
		},
	}

	// Expected cancellation
	err = s.scheduler.SyncForCodebases(ctx, codebaseConfigs)
	assert.Error(s.T(), err)
	assert.Contains(s.T(), err.Error(), "context canceled")
}

func TestIntegrationSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
