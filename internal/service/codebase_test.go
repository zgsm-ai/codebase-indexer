package service

import (
	"codebase-indexer/internal/config"
	"codebase-indexer/internal/dto"
	"codebase-indexer/internal/model"
	"codebase-indexer/internal/repository"
	"codebase-indexer/pkg/codegraph/proto/codegraphpb"
	"codebase-indexer/pkg/codegraph/store"
	"codebase-indexer/pkg/codegraph/workspace"
	"codebase-indexer/test/mocks"
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestCodebaseService_GetFileSkeleton(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// 创建 mock 对象
	mockManager := &mockStorageManager{}
	mockLogger := &mocks.MockLogger{}
	mockWorkspaceReader := mocks.NewMockWorkspaceReader(ctrl)
	mockWorkspaceRepo := mocks.NewMockWorkspaceRepository(ctrl)
	mockIndexer := mocks.NewMockIndexer(ctrl)

	// 创建测试实例
	service := &codebaseService{
		manager:             mockManager,
		logger:              mockLogger,
		workspaceReader:     mockWorkspaceReader,
		workspaceRepository: mockWorkspaceRepo,
		indexer:             mockIndexer,
	}

	// 设置 mock manager 的 workspace repository 返回
	mockManager.workspaceRepo = mockWorkspaceRepo

	tests := []struct {
		name          string
		req           *dto.GetFileSkeletonRequest
		setupMocks    func()
		expectedError bool
		errorMsg      string
		validateData  func(*dto.FileSkeletonData)
	}{
		{
			name: "成功获取文件骨架（绝对路径）",
			req: &dto.GetFileSkeletonRequest{
				ClientId:      "test-client",
				WorkspacePath: "/workspace",
				FilePath:      "/workspace/file.go",
				FilteredBy:    "",
			},
			setupMocks: func() {
				// 检查工作区存在
				workspace := &model.Workspace{
					ID:            1,
					WorkspacePath: "/workspace",
				}
				mockWorkspaceRepo.EXPECT().GetWorkspaceByPath("/workspace").Return(workspace, nil)

				// 返回 FileElementTable（使用 gomock.Any() 匹配路径，因为不同操作系统分隔符不同）
				fileTable := &codegraphpb.FileElementTable{
					Path:     "/workspace/file.go",
					Language: "go",
					Elements: []*codegraphpb.Element{
						{Name: "TestFunc", IsDefinition: true, ElementType: codegraphpb.ElementType_FUNCTION, Range: []int32{0, 0, 2, 0}},
						{Name: "TestVar", IsDefinition: false, ElementType: codegraphpb.ElementType_REFERENCE, Range: []int32{5, 0, 5, 10}},
					},
				}
				mockIndexer.EXPECT().GetFileElementTable(gomock.Any(), gomock.Any(), gomock.Any()).Return(fileTable, nil)
				
				// Mock 文件读取
				mockWorkspaceReader.EXPECT().ReadFile(gomock.Any(), gomock.Any(), gomock.Any()).Return([]byte("func TestFunc() {\n}\n\nvar TestVar int"), nil)
			},
			expectedError: false,
			validateData: func(data *dto.FileSkeletonData) {
				assert.NotNil(t, data)
				assert.Equal(t, "/workspace/file.go", data.Path)
				assert.Equal(t, "go", data.Language)
				assert.Len(t, data.Elements, 2)
				// 验证 range 已转换为 1-based
				assert.Equal(t, []int{1, 1, 3, 1}, data.Elements[0].Range)
				assert.Equal(t, "FUNCTION", data.Elements[0].ElementType)
			},
		},
		{
			name: "成功获取文件骨架（相对路径）",
			req: &dto.GetFileSkeletonRequest{
				ClientId:      "test-client",
				WorkspacePath: "/workspace",
				FilePath:      "src/file.go",
				FilteredBy:    "",
			},
			setupMocks: func() {
				// 检查工作区存在
				workspace := &model.Workspace{
					ID:            1,
					WorkspacePath: "/workspace",
				}
				mockWorkspaceRepo.EXPECT().GetWorkspaceByPath("/workspace").Return(workspace, nil)

				// 返回 FileElementTable（相对路径会被转换为绝对路径）
				fileTable := &codegraphpb.FileElementTable{
					Path:     "/workspace/src/file.go",
					Language: "go",
					Elements: []*codegraphpb.Element{
						{Name: "TestFunc", IsDefinition: true, ElementType: codegraphpb.ElementType_FUNCTION, Range: []int32{0, 0, 2, 0}},
					},
				}
				mockIndexer.EXPECT().GetFileElementTable(gomock.Any(), "/workspace", gomock.Any()).Return(fileTable, nil)
				mockWorkspaceReader.EXPECT().ReadFile(gomock.Any(), gomock.Any(), gomock.Any()).Return([]byte("func TestFunc() {\n}\n"), nil)
			},
			expectedError: false,
			validateData: func(data *dto.FileSkeletonData) {
				assert.NotNil(t, data)
				assert.Len(t, data.Elements, 1)
			},
		},
		{
			name: "成功过滤 definition",
			req: &dto.GetFileSkeletonRequest{
				ClientId:      "test-client",
				WorkspacePath: "/workspace",
				FilePath:      "/workspace/file.go",
				FilteredBy:    "definition",
			},
			setupMocks: func() {
				// 检查工作区存在
				workspace := &model.Workspace{
					ID:            1,
					WorkspacePath: "/workspace",
				}
				mockWorkspaceRepo.EXPECT().GetWorkspaceByPath("/workspace").Return(workspace, nil)

				// 返回 FileElementTable
				fileTable := &codegraphpb.FileElementTable{
					Path:     "/workspace/file.go",
					Language: "go",
					Elements: []*codegraphpb.Element{
						{Name: "TestFunc", IsDefinition: true, ElementType: codegraphpb.ElementType_FUNCTION, Range: []int32{0, 0, 2, 0}},
						{Name: "TestVar", IsDefinition: false, ElementType: codegraphpb.ElementType_REFERENCE, Range: []int32{5, 0, 5, 10}},
						{Name: "TestClass", IsDefinition: true, ElementType: codegraphpb.ElementType_CLASS, Range: []int32{7, 0, 10, 0}},
					},
				}
				mockIndexer.EXPECT().GetFileElementTable(gomock.Any(), gomock.Any(), gomock.Any()).Return(fileTable, nil)
				mockWorkspaceReader.EXPECT().ReadFile(gomock.Any(), gomock.Any(), gomock.Any()).Return([]byte("func TestFunc() {}\n\nvar TestVar int\n\ntype TestClass struct {}"), nil)
			},
			expectedError: false,
			validateData: func(data *dto.FileSkeletonData) {
				assert.NotNil(t, data)
				assert.Len(t, data.Elements, 2) // 只有 2 个 definition
				for _, elem := range data.Elements {
					assert.True(t, elem.IsDefinition)
				}
			},
		},
		{
			name: "成功过滤 reference",
			req: &dto.GetFileSkeletonRequest{
				ClientId:      "test-client",
				WorkspacePath: "/workspace",
				FilePath:      "/workspace/file.go",
				FilteredBy:    "reference",
			},
			setupMocks: func() {
				// 检查工作区存在
				workspace := &model.Workspace{
					ID:            1,
					WorkspacePath: "/workspace",
				}
				mockWorkspaceRepo.EXPECT().GetWorkspaceByPath("/workspace").Return(workspace, nil)

				// 返回 FileElementTable
				fileTable := &codegraphpb.FileElementTable{
					Path:     "/workspace/file.go",
					Language: "go",
					Elements: []*codegraphpb.Element{
						{Name: "TestFunc", IsDefinition: true, ElementType: codegraphpb.ElementType_FUNCTION, Range: []int32{0, 0, 2, 0}},
						{Name: "TestVar", IsDefinition: false, ElementType: codegraphpb.ElementType_REFERENCE, Range: []int32{5, 0, 5, 10}},
						{Name: "TestCall", IsDefinition: false, ElementType: codegraphpb.ElementType_CALL, Range: []int32{7, 0, 7, 10}},
					},
				}
				mockIndexer.EXPECT().GetFileElementTable(gomock.Any(), gomock.Any(), gomock.Any()).Return(fileTable, nil)
				mockWorkspaceReader.EXPECT().ReadFile(gomock.Any(), gomock.Any(), gomock.Any()).Return([]byte("func TestFunc() {}\n\nvar TestVar int\n\nTestCall()"), nil)
			},
			expectedError: false,
			validateData: func(data *dto.FileSkeletonData) {
				assert.NotNil(t, data)
				assert.Len(t, data.Elements, 2) // 只有 2 个 reference
				for _, elem := range data.Elements {
					assert.False(t, elem.IsDefinition)
				}
			},
		},
		{
			name: "workspacePath 为空",
			req: &dto.GetFileSkeletonRequest{
				ClientId:      "test-client",
				WorkspacePath: "",
				FilePath:      "/workspace/file.go",
			},
			setupMocks:    func() {},
			expectedError: true,
			errorMsg:      "workspacePath",
		},
		{
			name: "filePath 为空",
			req: &dto.GetFileSkeletonRequest{
				ClientId:      "test-client",
				WorkspacePath: "/workspace",
				FilePath:      "",
			},
			setupMocks:    func() {},
			expectedError: true,
			errorMsg:      "filePath",
		},
		{
			name: "工作区不存在",
			req: &dto.GetFileSkeletonRequest{
				ClientId:      "test-client",
				WorkspacePath: "/nonexistent",
				FilePath:      "/nonexistent/file.go",
			},
			setupMocks: func() {
				mockWorkspaceRepo.EXPECT().GetWorkspaceByPath("/nonexistent").Return(nil, errors.New("workspace not found"))
			},
			expectedError: true,
			errorMsg:      "workspace not found",
		},
		{
			name: "索引不存在",
			req: &dto.GetFileSkeletonRequest{
				ClientId:      "test-client",
				WorkspacePath: "/workspace",
				FilePath:      "/workspace/notfound.go",
			},
			setupMocks: func() {
				workspace := &model.Workspace{
					ID:            1,
					WorkspacePath: "/workspace",
				}
				mockWorkspaceRepo.EXPECT().GetWorkspaceByPath("/workspace").Return(workspace, nil)

				// 索引不存在
				mockIndexer.EXPECT().GetFileElementTable(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(nil, errors.New("index not found for file /workspace/notfound.go"))
			},
			expectedError: true,
			errorMsg:      "index not found",
		},
		{
			name: "不识别的过滤类型（不进行过滤）",
			req: &dto.GetFileSkeletonRequest{
				ClientId:      "test-client",
				WorkspacePath: "/workspace",
				FilePath:      "/workspace/file.go",
				FilteredBy:    "unknown",
			},
			setupMocks: func() {
				workspace := &model.Workspace{
					ID:            1,
					WorkspacePath: "/workspace",
				}
				mockWorkspaceRepo.EXPECT().GetWorkspaceByPath("/workspace").Return(workspace, nil)

				fileTable := &codegraphpb.FileElementTable{
					Path:     "/workspace/file.go",
					Language: "go",
					Elements: []*codegraphpb.Element{
						{Name: "TestFunc", IsDefinition: true, ElementType: codegraphpb.ElementType_FUNCTION, Range: []int32{0, 0, 2, 0}},
						{Name: "TestVar", IsDefinition: false, ElementType: codegraphpb.ElementType_REFERENCE, Range: []int32{5, 0, 5, 10}},
					},
				}
				mockIndexer.EXPECT().GetFileElementTable(gomock.Any(), gomock.Any(), gomock.Any()).Return(fileTable, nil)
				mockWorkspaceReader.EXPECT().ReadFile(gomock.Any(), gomock.Any(), gomock.Any()).Return([]byte("func TestFunc() {}\n\nvar TestVar int"), nil)
			},
			expectedError: false,
			validateData: func(data *dto.FileSkeletonData) {
				assert.NotNil(t, data)
				assert.Len(t, data.Elements, 2) // 不过滤，全部返回
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			result, err := service.GetFileSkeleton(context.Background(), tt.req)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tt.validateData != nil {
					tt.validateData(result)
				}
			}
		})
	}
}

func TestIndexer_GetFileElementTable(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// 创建 mock 对象
	mockStorage := mocks.NewMockGraphStorage(ctrl)
	mockWorkspaceReader := mocks.NewMockWorkspaceReader(ctrl)
	mockLogger := &mocks.MockLogger{}

	// 创建测试实例
	indexer := &indexer{
		storage:         mockStorage,
		workspaceReader: mockWorkspaceReader,
		logger:          mockLogger,
	}

	tests := []struct {
		name          string
		workspacePath string
		filePath      string
		setupMocks    func()
		expectedError bool
		errorMsg      string
		validateData  func(*codegraphpb.FileElementTable)
	}{
		{
			name:          "成功获取文件元素表（绝对路径）",
			workspacePath: "/workspace",
			filePath:      "/workspace/test.go",
			setupMocks: func() {
				// GetProjectByFilePath
				project := &workspace.Project{
					Name: "test-project",
					Path: "/workspace",
					Uuid: "test-uuid",
				}
				mockWorkspaceReader.EXPECT().GetProjectByFilePath(gomock.Any(), gomock.Any(), gomock.Any(), true).
					Return(project, nil)

				// ProjectIndexExists
				mockStorage.EXPECT().ProjectIndexExists("test-uuid").Return(true, nil)

				// Get FileElementTable
				fileTableBytes := mustMarshal(&codegraphpb.FileElementTable{
					Path:     "/workspace/test.go",
					Language: "go",
					Elements: []*codegraphpb.Element{
						{Name: "TestFunc", IsDefinition: true},
					},
				})
				mockStorage.EXPECT().Get(gomock.Any(), "test-uuid", gomock.Any()).Return(fileTableBytes, nil)
			},
			expectedError: false,
			validateData: func(table *codegraphpb.FileElementTable) {
				assert.Equal(t, "/workspace/test.go", table.Path)
				assert.Equal(t, "go", table.Language)
				assert.Len(t, table.Elements, 1)
			},
		},
		{
			name:          "成功获取文件元素表（相对路径）",
			workspacePath: "/workspace",
			filePath:      "src/test.go",
			setupMocks: func() {
				// GetProjectByFilePath
				project := &workspace.Project{
					Name: "test-project",
					Path: "/workspace",
					Uuid: "test-uuid",
				}
				mockWorkspaceReader.EXPECT().GetProjectByFilePath(gomock.Any(), gomock.Any(), gomock.Any(), true).
					Return(project, nil)

				// ProjectIndexExists
				mockStorage.EXPECT().ProjectIndexExists("test-uuid").Return(true, nil)

				// Get FileElementTable
				fileTableBytes := mustMarshal(&codegraphpb.FileElementTable{
					Path:     "/workspace/src/test.go",
					Language: "go",
				})
				mockStorage.EXPECT().Get(gomock.Any(), "test-uuid", gomock.Any()).Return(fileTableBytes, nil)
			},
			expectedError: false,
			validateData: func(table *codegraphpb.FileElementTable) {
				assert.NotNil(t, table)
			},
		},
		{
			name:          "项目不存在",
			workspacePath: "/workspace",
			filePath:      "/workspace/test.go",
			setupMocks: func() {
				mockWorkspaceReader.EXPECT().GetProjectByFilePath(gomock.Any(), gomock.Any(), gomock.Any(), true).
					Return(nil, errors.New("project not found"))
			},
			expectedError: true,
			errorMsg:      "failed to get project",
		},
		{
			name:          "不支持的文件类型",
			workspacePath: "/workspace",
			filePath:      "/workspace/test.txt",
			setupMocks: func() {
				project := &workspace.Project{
					Name: "test-project",
					Path: "/workspace",
					Uuid: "test-uuid",
				}
				mockWorkspaceReader.EXPECT().GetProjectByFilePath(gomock.Any(), gomock.Any(), gomock.Any(), true).
					Return(project, nil)

				mockStorage.EXPECT().ProjectIndexExists("test-uuid").Return(true, nil)
			},
			expectedError: true,
			errorMsg:      "unsupported file type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMocks()

			result, err := indexer.GetFileElementTable(context.Background(), tt.workspacePath, tt.filePath)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tt.validateData != nil {
					tt.validateData(result)
				}
			}
		})
	}
}

// mustMarshal 辅助函数用于测试
func mustMarshal(msg *codegraphpb.FileElementTable) []byte {
	data, err := proto.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return data
}

// mockStorageManager 简单的 mock 实现
type mockStorageManager struct {
	workspaceRepo repository.WorkspaceRepository
}

func (m *mockStorageManager) GetCodebaseConfigs() map[string]*config.CodebaseConfig {
	return nil
}

func (m *mockStorageManager) GetCodebaseConfig(codebaseId string) (*config.CodebaseConfig, error) {
	return nil, nil
}

func (m *mockStorageManager) SaveCodebaseConfig(cfg *config.CodebaseConfig) error {
	return nil
}

func (m *mockStorageManager) DeleteCodebaseConfig(codebaseId string) error {
	return nil
}

func (m *mockStorageManager) GetCodebaseEnv() *config.CodebaseEnv {
	return &config.CodebaseEnv{Switch: dto.SwitchOn}
}

func (m *mockStorageManager) SaveCodebaseEnv(codebaseEnv *config.CodebaseEnv) error {
	return nil
}

func (m *mockStorageManager) GetCodegraphStorage() store.GraphStorage {
	return nil
}
