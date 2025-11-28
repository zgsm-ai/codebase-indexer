package service

import (
	"codebase-indexer/internal/model"
	"codebase-indexer/pkg/codegraph/types"
	"codebase-indexer/pkg/codegraph/workspace"
	"codebase-indexer/test/mocks"
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestCodegraphProcessor_ProcessActiveWorkspaces(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// 创建 mock 对象
	mockWorkspaceRepo := mocks.NewMockWorkspaceRepository(ctrl)
	mockWorkspaceReader := mocks.NewMockWorkspaceReader(ctrl)
	mockIndexer := mocks.NewMockIndexer(ctrl)
	mockEventRepo := mocks.NewMockEventRepository(ctrl)

	// 创建测试实例
	processor := &CodegraphProcessor{
		workspaceRepo:   mockWorkspaceRepo,
		workspaceReader: mockWorkspaceReader,
		indexer:         mockIndexer,
		eventRepo:       mockEventRepo,
	}

	tests := []struct {
		name           string
		setupMocks     func()
		expectedResult []*model.Workspace
		expectedError  error
	}{
		{
			name: "成功获取活跃工作区",
			setupMocks: func() {
				// 模拟返回包含活跃和非活跃工作区的列表
				workspaces := []*model.Workspace{
					{ID: 1, WorkspaceName: "Active1", WorkspacePath: "/path/active1", Active: model.True},
					{ID: 2, WorkspaceName: "Inactive1", WorkspacePath: "/path/inactive1", Active: "false"},
					{ID: 3, WorkspaceName: "Active2", WorkspacePath: "/path/active2", Active: model.True},
				}
				mockWorkspaceRepo.EXPECT().GetActiveWorkspaces().Return(workspaces, nil)
			},
			expectedResult: []*model.Workspace{
				{ID: 1, WorkspaceName: "Active1", WorkspacePath: "/path/active1", Active: model.True},
				{ID: 3, WorkspaceName: "Active2", WorkspacePath: "/path/active2", Active: model.True},
			},
			expectedError: nil,
		},
		{
			name: "仓库返回空列表",
			setupMocks: func() {
				mockWorkspaceRepo.EXPECT().GetActiveWorkspaces().Return([]*model.Workspace{}, nil)
			},
			expectedResult: []*model.Workspace{},
			expectedError:  nil,
		},
		{
			name: "仓库返回错误",
			setupMocks: func() {
				mockWorkspaceRepo.EXPECT().GetActiveWorkspaces().Return(nil, errors.New("database error"))
			},
			expectedResult: nil,
			expectedError:  errors.New("failed to get active workspaces: database error"),
		},
		{
			name: "没有活跃工作区",
			setupMocks: func() {
				workspaces := []*model.Workspace{
					{ID: 1, WorkspaceName: "Inactive1", WorkspacePath: "/path/inactive1", Active: "false"},
					{ID: 2, WorkspaceName: "Inactive2", WorkspacePath: "/path/inactive2", Active: "false"},
				}
				mockWorkspaceRepo.EXPECT().GetActiveWorkspaces().Return(workspaces, nil)
			},
			expectedResult: []*model.Workspace{},
			expectedError:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			result, err := processor.ProcessActiveWorkspaces(context.Background())

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Equal(t, tt.expectedError.Error(), err.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, len(tt.expectedResult), len(result))
			for i, expectedWorkspace := range tt.expectedResult {
				assert.Equal(t, expectedWorkspace.ID, result[i].ID)
				assert.Equal(t, expectedWorkspace.WorkspaceName, result[i].WorkspaceName)
				assert.Equal(t, expectedWorkspace.WorkspacePath, result[i].WorkspacePath)
				assert.Equal(t, expectedWorkspace.Active, result[i].Active)
			}
		})
	}
}

func TestCodegraphProcessor_ProcessAddFileEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// 创建 mock 对象
	mockWorkspaceRepo := mocks.NewMockWorkspaceRepository(ctrl)
	mockLogger := &mocks.MockLogger{}
	mockWorkspaceReader := mocks.NewMockWorkspaceReader(ctrl)
	mockIndexer := mocks.NewMockIndexer(ctrl)
	mockEventRepo := mocks.NewMockEventRepository(ctrl)

	// 创建测试实例
	processor := &CodegraphProcessor{
		workspaceRepo:   mockWorkspaceRepo,
		logger:          mockLogger,
		workspaceReader: mockWorkspaceReader,
		indexer:         mockIndexer,
		eventRepo:       mockEventRepo,
	}

	tests := []struct {
		name        string
		event       *model.Event
		setupMocks  func()
		expectError bool
		errorMsg    string
	}{
		{
			name: "成功处理添加文件事件",
			event: &model.Event{
				ID:              1,
				WorkspacePath:   "/workspace",
				EventType:       model.EventTypeAddFile,
				SourceFilePath:  "/workspace/file.go",
				CodegraphStatus: model.CodegraphStatusInit,
			},
			setupMocks: func() {
				// 文件存在且不是目录
				fileInfo := &types.FileInfo{
					Name:  "file.go",
					Path:  "/workspace/file.go",
					IsDir: false,
				}
				mockWorkspaceReader.EXPECT().Stat("/workspace/file.go").Return(fileInfo, nil)

				// 索引文件成功
				mockIndexer.EXPECT().IndexFiles(gomock.Any(), "/workspace", []string{"/workspace/file.go"}).Return(nil)

				// 更新事件状态为成功
				mockEventRepo.EXPECT().UpdateEvent(gomock.Any()).Return(nil)
			},
			expectError: false,
		},
		{
			name: "文件不存在",
			event: &model.Event{
				ID:              2,
				WorkspacePath:   "/workspace",
				EventType:       model.EventTypeAddFile,
				SourceFilePath:  "/workspace/notfound.go",
				CodegraphStatus: model.CodegraphStatusInit,
			},
			setupMocks: func() {
				// 设置 mock logger 预期
				mockLogger.On("Error", "codegraph failed to process add event, file %s not exists.", []interface{}{"/workspace/notfound.go"}).Return()

				// 文件不存在
				mockWorkspaceReader.EXPECT().Stat("/workspace/notfound.go").Return(nil, workspace.ErrPathNotExists)

				// 更新事件状态为失败
				mockEventRepo.EXPECT().UpdateEvent(gomock.Any()).Return(nil)
			},
			expectError: true,
			errorMsg:    "no such file or directory",
		},
		{
			name: "文件是目录",
			event: &model.Event{
				ID:              3,
				WorkspacePath:   "/workspace",
				EventType:       model.EventTypeAddFile,
				SourceFilePath:  "/workspace/directory",
				CodegraphStatus: model.CodegraphStatusInit,
			},
			setupMocks: func() {
				// 设置 mock logger 预期
				mockLogger.On("Error", "codegraph add event, file %s is dir, not process.", []interface{}{"/workspace/directory"}).Return()

				// 文件是目录
				fileInfo := &types.FileInfo{
					Name:  "directory",
					Path:  "/workspace/directory",
					IsDir: true,
				}
				mockWorkspaceReader.EXPECT().Stat("/workspace/directory").Return(fileInfo, nil)

				// 更新事件状态（不应该失败，因为是目录所以跳过处理）
				mockEventRepo.EXPECT().UpdateEvent(gomock.Any()).Return(nil)
			},
			expectError: false,
		},
		{
			name: "索引文件失败",
			event: &model.Event{
				ID:              4,
				WorkspacePath:   "/workspace",
				EventType:       model.EventTypeAddFile,
				SourceFilePath:  "/workspace/error.go",
				CodegraphStatus: model.CodegraphStatusInit,
			},
			setupMocks: func() {
				// 文件存在且不是目录
				fileInfo := &types.FileInfo{
					Name:  "error.go",
					Path:  "/workspace/error.go",
					IsDir: false,
				}
				mockWorkspaceReader.EXPECT().Stat("/workspace/error.go").Return(fileInfo, nil)

				// 索引文件失败
				indexErr := errors.New("index failed")
				mockIndexer.EXPECT().IndexFiles(gomock.Any(), "/workspace", []string{"/workspace/error.go"}).Return(indexErr)

				// 更新事件状态为失败
				mockEventRepo.EXPECT().UpdateEvent(gomock.Any()).Return(nil)
			},
			expectError: true,
			errorMsg:    "index failed",
		},
		{
			name: "更新事件状态失败",
			event: &model.Event{
				ID:              5,
				WorkspacePath:   "/workspace",
				EventType:       model.EventTypeAddFile,
				SourceFilePath:  "/workspace/updatefail.go",
				CodegraphStatus: model.CodegraphStatusInit,
			},
			setupMocks: func() {
				// 文件存在且不是目录
				fileInfo := &types.FileInfo{
					Name:  "updatefail.go",
					Path:  "/workspace/updatefail.go",
					IsDir: false,
				}
				mockWorkspaceReader.EXPECT().Stat("/workspace/updatefail.go").Return(fileInfo, nil)

				// 索引文件成功
				mockIndexer.EXPECT().IndexFiles(gomock.Any(), "/workspace", []string{"/workspace/updatefail.go"}).Return(nil)

				// 更新事件状态失败
				updateErr := errors.New("update failed")
				mockEventRepo.EXPECT().UpdateEvent(gomock.Any()).Return(updateErr)
			},
			expectError: true,
			errorMsg:    "failed to update success processed event. update err: update failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			err := processor.ProcessAddFileEvent(context.Background(), tt.event)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCodegraphProcessor_ProcessModifyFileEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// 创建 mock 对象
	mockWorkspaceRepo := mocks.NewMockWorkspaceRepository(ctrl)
	mockLogger := &mocks.MockLogger{}
	mockWorkspaceReader := mocks.NewMockWorkspaceReader(ctrl)
	mockIndexer := mocks.NewMockIndexer(ctrl)
	mockEventRepo := mocks.NewMockEventRepository(ctrl)

	// 创建测试实例
	processor := &CodegraphProcessor{
		workspaceRepo:   mockWorkspaceRepo,
		logger:          mockLogger,
		workspaceReader: mockWorkspaceReader,
		indexer:         mockIndexer,
		eventRepo:       mockEventRepo,
	}

	tests := []struct {
		name        string
		event       *model.Event
		setupMocks  func()
		expectError bool
		errorMsg    string
	}{
		{
			name: "成功处理修改文件事件",
			event: &model.Event{
				ID:              1,
				WorkspacePath:   "/workspace",
				EventType:       model.EventTypeModifyFile,
				SourceFilePath:  "/workspace/file.go",
				CodegraphStatus: model.CodegraphStatusInit,
			},
			setupMocks: func() {
				// 文件存在且不是目录
				fileInfo := &types.FileInfo{
					Name:  "file.go",
					Path:  "/workspace/file.go",
					IsDir: false,
				}
				mockWorkspaceReader.EXPECT().Stat("/workspace/file.go").Return(fileInfo, nil)

				// 重新索引文件成功
				mockIndexer.EXPECT().IndexFiles(gomock.Any(), "/workspace", []string{"/workspace/file.go"}).Return(nil)

				// 更新事件状态为成功
				mockEventRepo.EXPECT().UpdateEvent(gomock.Any()).Return(nil)
			},
			expectError: false,
		},
		{
			name: "文件不存在",
			event: &model.Event{
				ID:              2,
				WorkspacePath:   "/workspace",
				EventType:       model.EventTypeModifyFile,
				SourceFilePath:  "/workspace/notfound.go",
				CodegraphStatus: model.CodegraphStatusInit,
			},
			setupMocks: func() {
				// 设置 mock logger 预期
				mockLogger.On("Error", "codegraph failed to process modify event, file %s not exists", []interface{}{"/workspace/notfound.go"}).Return()

				// 文件不存在
				mockWorkspaceReader.EXPECT().Stat("/workspace/notfound.go").Return(nil, workspace.ErrPathNotExists)

				// 更新事件状态为失败
				mockEventRepo.EXPECT().UpdateEvent(gomock.Any()).Return(nil)
			},
			expectError: true,
			errorMsg:    "no such file or directory",
		},
		{
			name: "文件是目录",
			event: &model.Event{
				ID:              3,
				WorkspacePath:   "/workspace",
				EventType:       model.EventTypeModifyFile,
				SourceFilePath:  "/workspace/directory",
				CodegraphStatus: model.CodegraphStatusInit,
			},
			setupMocks: func() {
				// 设置 mock logger 预期
				mockLogger.On("Error", "codegraph modify event, file %s is dir, not process.", []interface{}{"/workspace/directory"}).Return()

				// 文件是目录
				fileInfo := &types.FileInfo{
					Name:  "directory",
					Path:  "/workspace/directory",
					IsDir: true,
				}
				mockWorkspaceReader.EXPECT().Stat("/workspace/directory").Return(fileInfo, nil)

				// 更新事件状态（不应该失败，因为是目录所以跳过处理）
				mockEventRepo.EXPECT().UpdateEvent(gomock.Any()).Return(nil)
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			err := processor.ProcessModifyFileEvent(context.Background(), tt.event)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCodegraphProcessor_ProcessDeleteFileEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// 创建 mock 对象
	mockWorkspaceRepo := mocks.NewMockWorkspaceRepository(ctrl)
	mockLogger := &mocks.MockLogger{}
	mockWorkspaceReader := mocks.NewMockWorkspaceReader(ctrl)
	mockIndexer := mocks.NewMockIndexer(ctrl)
	// mockgen -source=internal/repository/event.go -destination=test/mocks/mock_event_repository.go -package=mocks
	mockEventRepo := mocks.NewMockEventRepository(ctrl)

	// 创建测试实例
	processor := &CodegraphProcessor{
		workspaceRepo:   mockWorkspaceRepo,
		logger:          mockLogger,
		workspaceReader: mockWorkspaceReader,
		indexer:         mockIndexer,
		eventRepo:       mockEventRepo,
	}

	tests := []struct {
		name        string
		event       *model.Event
		setupMocks  func()
		expectError bool
		errorMsg    string
	}{
		{
			name: "成功处理删除文件事件",
			event: &model.Event{
				ID:              1,
				WorkspacePath:   "/workspace",
				EventType:       model.EventTypeDeleteFile,
				SourceFilePath:  "/workspace/file.go",
				CodegraphStatus: model.CodegraphStatusInit,
			},
			setupMocks: func() {
				// 删除索引成功
				mockIndexer.EXPECT().RemoveIndexes(gomock.Any(), "/workspace", []string{"/workspace/file.go"}).Return(nil)

				// 更新事件状态为成功
				mockEventRepo.EXPECT().UpdateEvent(gomock.Any()).Return(nil)
			},
			expectError: false,
		},
		{
			name: "删除索引失败",
			event: &model.Event{
				ID:              2,
				WorkspacePath:   "/workspace",
				EventType:       model.EventTypeDeleteFile,
				SourceFilePath:  "/workspace/error.go",
				CodegraphStatus: model.CodegraphStatusInit,
			},
			setupMocks: func() {
				// 删除索引失败
				deleteErr := errors.New("delete failed")
				mockIndexer.EXPECT().RemoveIndexes(gomock.Any(), "/workspace", []string{"/workspace/error.go"}).Return(deleteErr)

				// 更新事件状态为失败
				mockEventRepo.EXPECT().UpdateEvent(gomock.Any()).Return(nil)
			},
			expectError: true,
			errorMsg:    "delete failed",
		},
		{
			name: "更新事件状态失败",
			event: &model.Event{
				ID:              3,
				WorkspacePath:   "/workspace",
				EventType:       model.EventTypeDeleteFile,
				SourceFilePath:  "/workspace/updatefail.go",
				CodegraphStatus: model.CodegraphStatusInit,
			},
			setupMocks: func() {
				// 删除索引成功
				mockIndexer.EXPECT().RemoveIndexes(gomock.Any(), "/workspace", []string{"/workspace/updatefail.go"}).Return(nil)

				// 更新事件状态失败
				updateErr := errors.New("update failed")
				mockEventRepo.EXPECT().UpdateEvent(gomock.Any()).Return(updateErr)
			},
			expectError: true,
			errorMsg:    "failed to update success processed event. update err: update failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			err := processor.ProcessDeleteFileEvent(context.Background(), tt.event)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCodegraphProcessor_ProcessRenameFileEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// 创建 mock 对象
	mockWorkspaceRepo := mocks.NewMockWorkspaceRepository(ctrl)
	mockLogger := &mocks.MockLogger{}
	mockWorkspaceReader := mocks.NewMockWorkspaceReader(ctrl)
	mockIndexer := mocks.NewMockIndexer(ctrl)
	mockEventRepo := mocks.NewMockEventRepository(ctrl)

	// 创建测试实例
	processor := &CodegraphProcessor{
		workspaceRepo:   mockWorkspaceRepo,
		logger:          mockLogger,
		workspaceReader: mockWorkspaceReader,
		indexer:         mockIndexer,
		eventRepo:       mockEventRepo,
	}

	tests := []struct {
		name        string
		event       *model.Event
		setupMocks  func()
		expectError bool
		errorMsg    string
	}{
		{
			name: "成功处理重命名文件事件",
			event: &model.Event{
				ID:              1,
				WorkspacePath:   "/workspace",
				EventType:       model.EventTypeRenameFile,
				SourceFilePath:  "/workspace/old.go",
				TargetFilePath:  "/workspace/new.go",
				CodegraphStatus: model.CodegraphStatusInit,
			},
			setupMocks: func() {
				// 重命名索引成功
				mockIndexer.EXPECT().RenameIndexes(gomock.Any(), "/workspace", "/workspace/old.go", "/workspace/new.go").Return(nil)

				// 更新事件状态为成功
				mockEventRepo.EXPECT().UpdateEvent(gomock.Any()).Return(nil)
			},
			expectError: false,
		},
		{
			name: "重命名索引失败",
			event: &model.Event{
				ID:              2,
				WorkspacePath:   "/workspace",
				EventType:       model.EventTypeRenameFile,
				SourceFilePath:  "/workspace/old.go",
				TargetFilePath:  "/workspace/new.go",
				CodegraphStatus: model.CodegraphStatusInit,
			},
			setupMocks: func() {
				// 重命名索引失败
				renameErr := errors.New("rename failed")
				mockIndexer.EXPECT().RenameIndexes(gomock.Any(), "/workspace", "/workspace/old.go", "/workspace/new.go").Return(renameErr)

				// 更新事件状态为失败
				mockEventRepo.EXPECT().UpdateEvent(gomock.Any()).Return(nil)
			},
			expectError: true,
			errorMsg:    "rename failed",
		},
		{
			name: "更新事件状态失败",
			event: &model.Event{
				ID:              3,
				WorkspacePath:   "/workspace",
				EventType:       model.EventTypeRenameFile,
				SourceFilePath:  "/workspace/old.go",
				TargetFilePath:  "/workspace/new.go",
				CodegraphStatus: model.CodegraphStatusInit,
			},
			setupMocks: func() {
				// 重命名索引成功
				mockIndexer.EXPECT().RenameIndexes(gomock.Any(), "/workspace", "/workspace/old.go", "/workspace/new.go").Return(nil)

				// 更新事件状态失败
				updateErr := errors.New("update failed")
				mockEventRepo.EXPECT().UpdateEvent(gomock.Any()).Return(updateErr)
			},
			expectError: true,
			errorMsg:    "failed to update success processed event. update err: update failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			err := processor.ProcessRenameFileEvent(context.Background(), tt.event)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCodegraphProcessor_ProcessOpenWorkspaceEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// 创建 mock 对象
	mockWorkspaceRepo := mocks.NewMockWorkspaceRepository(ctrl)
	mockLogger := &mocks.MockLogger{}
	mockWorkspaceReader := mocks.NewMockWorkspaceReader(ctrl)
	mockIndexer := mocks.NewMockIndexer(ctrl)
	mockEventRepo := mocks.NewMockEventRepository(ctrl)

	// 创建测试实例
	processor := &CodegraphProcessor{
		workspaceRepo:   mockWorkspaceRepo,
		logger:          mockLogger,
		workspaceReader: mockWorkspaceReader,
		indexer:         mockIndexer,
		eventRepo:       mockEventRepo,
	}

	tests := []struct {
		name        string
		event       *model.Event
		setupMocks  func()
		expectError bool
		errorMsg    string
	}{
		{
			name: "成功处理打开工作区事件",
			event: &model.Event{
				ID:              1,
				WorkspacePath:   "/workspace",
				EventType:       model.EventTypeOpenWorkspace,
				CodegraphStatus: model.CodegraphStatusInit,
			},
			setupMocks: func() {
				// 工作区存在且是目录
				fileInfo := &types.FileInfo{
					Name:  "workspace",
					Path:  "/workspace",
					IsDir: true,
				}
				mockWorkspaceReader.EXPECT().Stat("/workspace").Return(fileInfo, nil)

				// 更新 codegraph 信息
				mockWorkspaceRepo.EXPECT().UpdateCodegraphInfo("/workspace", 0, gomock.Any()).Return(nil)

				// 索引工作区成功
				mockIndexer.EXPECT().IndexWorkspace(gomock.Any(), "/workspace").Return(&types.IndexTaskMetrics{}, nil)

				// 更新事件状态为成功
				mockEventRepo.EXPECT().UpdateEvent(gomock.Any()).Return(nil)
			},
			expectError: false,
		},
		{
			name: "工作区不存在",
			event: &model.Event{
				ID:              2,
				WorkspacePath:   "/notfound",
				EventType:       model.EventTypeOpenWorkspace,
				CodegraphStatus: model.CodegraphStatusInit,
			},
			setupMocks: func() {
				// 设置 mock logger 预期
				mockLogger.On("Error", "codegraph failed to process open_workspace event event, workspace %s not exists", []interface{}{"/notfound"}).Return()

				// 工作区不存在
				mockWorkspaceReader.EXPECT().Stat("/notfound").Return(nil, workspace.ErrPathNotExists)

				// 更新事件状态为失败
				mockEventRepo.EXPECT().UpdateEvent(gomock.Any()).Return(nil)
			},
			expectError: true,
			errorMsg:    "no such file or directory",
		},
		{
			name: "工作区是文件而不是目录",
			event: &model.Event{
				ID:              3,
				WorkspacePath:   "/workspace/file.txt",
				EventType:       model.EventTypeOpenWorkspace,
				CodegraphStatus: model.CodegraphStatusInit,
			},
			setupMocks: func() {
				// 设置 mock logger 预期
				mockLogger.On("Error", "codegraph open_workspace event, %s is file, not process.", []interface{}{"/workspace/file.txt"}).Return()

				// 工作区是文件
				fileInfo := &types.FileInfo{
					Name:  "file.txt",
					Path:  "/workspace/file.txt",
					IsDir: false,
				}
				mockWorkspaceReader.EXPECT().Stat("/workspace/file.txt").Return(fileInfo, nil)

				// 更新事件状态（不应该失败，因为是文件所以跳过处理）
				mockEventRepo.EXPECT().UpdateEvent(gomock.Any()).Return(nil)
			},
			expectError: false,
		},
		{
			name: "更新 codegraph 信息失败",
			event: &model.Event{
				ID:              4,
				WorkspacePath:   "/workspace",
				EventType:       model.EventTypeOpenWorkspace,
				CodegraphStatus: model.CodegraphStatusInit,
			},
			setupMocks: func() {
				// 工作区存在且是目录
				fileInfo := &types.FileInfo{
					Name:  "workspace",
					Path:  "/workspace",
					IsDir: true,
				}
				mockWorkspaceReader.EXPECT().Stat("/workspace").Return(fileInfo, nil)

				// 更新 codegraph 信息失败
				updateErr := errors.New("update codegraph info failed")
				mockWorkspaceRepo.EXPECT().UpdateCodegraphInfo("/workspace", 0, gomock.Any()).Return(updateErr)

				// 设置 mock logger 预期
				mockLogger.On("Error", "codegraph failed to process open_workspace event event, workspace %s reset successful file num failed, err:%v", []interface{}{"/workspace", updateErr}).Return()

				// 更新事件状态为失败
				mockEventRepo.EXPECT().UpdateEvent(gomock.Any()).Return(nil)
			},
			expectError: true,
			errorMsg:    "update codegraph info failed",
		},
		{
			name: "索引工作区失败",
			event: &model.Event{
				ID:              5,
				WorkspacePath:   "/workspace",
				EventType:       model.EventTypeOpenWorkspace,
				CodegraphStatus: model.CodegraphStatusInit,
			},
			setupMocks: func() {
				// 工作区存在且是目录
				fileInfo := &types.FileInfo{
					Name:  "workspace",
					Path:  "/workspace",
					IsDir: true,
				}
				mockWorkspaceReader.EXPECT().Stat("/workspace").Return(fileInfo, nil)

				// 更新 codegraph 信息成功
				mockWorkspaceRepo.EXPECT().UpdateCodegraphInfo("/workspace", 0, gomock.Any()).Return(nil)

				// 索引工作区失败
				indexErr := errors.New("index workspace failed")
				mockIndexer.EXPECT().IndexWorkspace(gomock.Any(), "/workspace").Return(nil, indexErr)

				// 更新事件状态为失败
				mockEventRepo.EXPECT().UpdateEvent(gomock.Any()).Return(nil)
			},
			expectError: true,
			errorMsg:    "index workspace failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			err := processor.ProcessOpenWorkspaceEvent(context.Background(), tt.event)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCodegraphProcessor_ProcessRebuildWorkspaceEvent(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// 创建 mock 对象
	mockWorkspaceRepo := mocks.NewMockWorkspaceRepository(ctrl)
	mockLogger := &mocks.MockLogger{}
	mockWorkspaceReader := mocks.NewMockWorkspaceReader(ctrl)
	mockIndexer := mocks.NewMockIndexer(ctrl)
	mockEventRepo := mocks.NewMockEventRepository(ctrl)

	// 创建测试实例
	processor := &CodegraphProcessor{
		workspaceRepo:   mockWorkspaceRepo,
		logger:          mockLogger,
		workspaceReader: mockWorkspaceReader,
		indexer:         mockIndexer,
		eventRepo:       mockEventRepo,
	}

	tests := []struct {
		name        string
		event       *model.Event
		setupMocks  func()
		expectError bool
		errorMsg    string
	}{
		{
			name: "成功处理重建工作区事件",
			event: &model.Event{
				ID:              1,
				WorkspacePath:   "/workspace",
				EventType:       model.EventTypeRebuildWorkspace,
				CodegraphStatus: model.CodegraphStatusInit,
			},
			setupMocks: func() {
				// 删除所有索引成功
				mockIndexer.EXPECT().RemoveAllIndexes(gomock.Any(), "/workspace").Return(nil)

				// 处理打开工作区事件成功
				// 工作区存在且是目录
				fileInfo := &types.FileInfo{
					Name:  "workspace",
					Path:  "/workspace",
					IsDir: true,
				}
				mockWorkspaceReader.EXPECT().Stat("/workspace").Return(fileInfo, nil)

				// 更新 codegraph 信息
				mockWorkspaceRepo.EXPECT().UpdateCodegraphInfo("/workspace", 0, gomock.Any()).Return(nil)

				// 索引工作区成功
				mockIndexer.EXPECT().IndexWorkspace(gomock.Any(), "/workspace").Return(&types.IndexTaskMetrics{}, nil)

				// 更新事件状态为成功
				mockEventRepo.EXPECT().UpdateEvent(gomock.Any()).Return(nil)
			},
			expectError: false,
		},
		{
			name: "删除所有索引失败",
			event: &model.Event{
				ID:              2,
				WorkspacePath:   "/workspace",
				EventType:       model.EventTypeRebuildWorkspace,
				CodegraphStatus: model.CodegraphStatusInit,
			},
			setupMocks: func() {
				// 删除所有索引失败
				removeErr := errors.New("remove all indexes failed")
				mockIndexer.EXPECT().RemoveAllIndexes(gomock.Any(), "/workspace").Return(removeErr)
			},
			expectError: true,
			errorMsg:    "remove all indexes failed",
		},
		{
			name: "处理打开工作区事件失败",
			event: &model.Event{
				ID:              3,
				WorkspacePath:   "/workspace",
				EventType:       model.EventTypeRebuildWorkspace,
				CodegraphStatus: model.CodegraphStatusInit,
			},
			setupMocks: func() {
				// 删除所有索引成功
				mockIndexer.EXPECT().RemoveAllIndexes(gomock.Any(), "/workspace").Return(nil)

				// 处理打开工作区事件失败
				// 工作区不存在
				mockWorkspaceReader.EXPECT().Stat("/workspace").Return(nil, workspace.ErrPathNotExists)

				// 设置 mock logger 预期
				mockLogger.On("Error", "codegraph failed to process open_workspace event event, workspace %s not exists", []interface{}{"/workspace"}).Return()

				// 更新事件状态为失败
				mockEventRepo.EXPECT().UpdateEvent(gomock.Any()).Return(nil)
			},
			expectError: true,
			errorMsg:    "no such file or directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			err := processor.ProcessRebuildWorkspaceEvent(context.Background(), tt.event)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCodegraphProcessor_updateEventStatusFinally(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// 创建 mock 对象
	mockEventRepo := mocks.NewMockEventRepository(ctrl)

	// 创建测试实例
	processor := &CodegraphProcessor{
		eventRepo: mockEventRepo,
	}

	tests := []struct {
		name        string
		event       *model.Event
		procErr     error
		setupMocks  func()
		expectError bool
		errorMsg    string
	}{
		{
			name: "成功更新事件状态为成功",
			event: &model.Event{
				ID:              1,
				WorkspacePath:   "/workspace",
				EventType:       model.EventTypeAddFile,
				SourceFilePath:  "/workspace/file.go",
				CodegraphStatus: model.CodegraphStatusInit,
			},
			procErr: nil,
			setupMocks: func() {
				// 期望更新事件状态为成功
				mockEventRepo.EXPECT().UpdateEvent(gomock.Any()).DoAndReturn(func(event *model.Event) error {
					assert.Equal(t, model.CodegraphStatusSuccess, event.CodegraphStatus)
					return nil
				})
			},
			expectError: false,
		},
		{
			name: "成功更新事件状态为失败",
			event: &model.Event{
				ID:              2,
				WorkspacePath:   "/workspace",
				EventType:       model.EventTypeAddFile,
				SourceFilePath:  "/workspace/file.go",
				CodegraphStatus: model.CodegraphStatusInit,
			},
			procErr: errors.New("processing error"),
			setupMocks: func() {
				// 期望更新事件状态为失败
				mockEventRepo.EXPECT().UpdateEvent(gomock.Any()).DoAndReturn(func(event *model.Event) error {
					assert.Equal(t, model.CodegraphStatusFailed, event.CodegraphStatus)
					return nil
				})
			},
			expectError: true,
			errorMsg:    "processing error",
		},
		{
			name: "更新事件状态失败",
			event: &model.Event{
				ID:              3,
				WorkspacePath:   "/workspace",
				EventType:       model.EventTypeAddFile,
				SourceFilePath:  "/workspace/file.go",
				CodegraphStatus: model.CodegraphStatusInit,
			},
			procErr: errors.New("processing error"),
			setupMocks: func() {
				// 更新事件状态失败
				updateErr := errors.New("update failed")
				mockEventRepo.EXPECT().UpdateEvent(gomock.Any()).Return(updateErr)
			},
			expectError: true,
			errorMsg:    "failed to update failed processed event. update err: update failed, index err: processing error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			err := processor.updateEventStatusFinally(tt.event, tt.procErr)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Equal(t, tt.errorMsg, err.Error())
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCodegraphProcessor_convertFilePathToAbs(t *testing.T) {
	// 创建测试实例（不需要 mock）
	processor := &CodegraphProcessor{}

	tests := []struct {
		name           string
		event          *model.Event
		expectedSource string
		expectedTarget string
	}{
		{
			name: "源路径已经是绝对路径",
			event: &model.Event{
				WorkspacePath:  "/workspace",
				SourceFilePath: "/workspace/file.go",
				TargetFilePath: "/workspace/new.go",
			},
			expectedSource: "/workspace/file.go",
			expectedTarget: "/workspace/new.go",
		},
		{
			name: "源路径不是绝对路径",
			event: &model.Event{
				WorkspacePath:  "/workspace",
				SourceFilePath: "file.go",
				TargetFilePath: "new.go",
			},
			expectedSource: filepath.Join("/workspace", "file.go"), // 修正：应该是 "/workspace/file.go"
			expectedTarget: filepath.Join("/workspace", "new.go"),  // 修正：应该是 "/workspace/new.go"
		},
		{
			name: "源路径是绝对路径，目标路径不是",
			event: &model.Event{
				WorkspacePath:  "/workspace",
				SourceFilePath: "/workspace/file.go",
				TargetFilePath: "new.go",
			},
			expectedSource: "/workspace/file.go",
			expectedTarget: filepath.Join("/workspace", "new.go"), // 修正：应该是 "/workspace/new.go"
		},
		{
			name: "源路径不是绝对路径，目标路径是",
			event: &model.Event{
				WorkspacePath:  "/workspace",
				SourceFilePath: "file.go",
				TargetFilePath: "/workspace/new.go",
			},
			expectedSource: filepath.Join("/workspace", "file.go"), // 修正：应该是 "/workspace/file.go"
			expectedTarget: "/workspace/new.go",
		},
		{
			name: "空路径",
			event: &model.Event{
				WorkspacePath:  "/workspace",
				SourceFilePath: "",
				TargetFilePath: "",
			},
			expectedSource: "/workspace",
			expectedTarget: "/workspace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 复制事件以避免修改原始数据
			eventCopy := *tt.event
			processor.convertWorkspaceFilePathToAbs(&eventCopy)

			assert.Equal(t, tt.expectedSource, eventCopy.SourceFilePath)
			assert.Equal(t, tt.expectedTarget, eventCopy.TargetFilePath)
		})
	}
}
