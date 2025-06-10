package scheduler

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"codebase-syncer/internal/scanner"
	"codebase-syncer/internal/storage"
	"codebase-syncer/internal/syncer"
	"codebase-syncer/internal/utils"
	"codebase-syncer/test/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var (
	mockLogger      = &mocks.MockLogger{}
	mockStorage     = &mocks.MockStorageManager{}
	mockHttpSync    = &mocks.MockHTTPSync{}
	mockFileScanner = &mocks.MockScanner{}
)

func TestPerformSync(t *testing.T) {
	mockLogger.On("Info", mock.Anything, mock.Anything).Return()

	s := &Scheduler{
		httpSync:    mockHttpSync,
		fileScanner: mockFileScanner,
		storage:     mockStorage,
		logger:      mockLogger,
	}

	t.Run("AlreadyRunning", func(t *testing.T) {
		s.isRunning = true
		defer func() { s.isRunning = false }()

		s.performSync()
	})

	t.Run("NormalSync", func(t *testing.T) {
		utils.UploadTmpDir = t.TempDir()
		codebaseConfigs := map[string]*storage.CodebaseConfig{
			"test-id": {
				CodebaseId:   "test-id",
				CodebasePath: "/test/path",
				RegisterTime: time.Now().Add(-time.Minute),
			},
		}

		mockStorage.On("GetCodebaseConfigs").Return(codebaseConfigs)
		mockStorage.On("SaveCodebaseConfig", mock.Anything).Return(nil)
		mockStorage.On("DeleteCodebaseConfig", mock.Anything).Return(nil)
		mockFileScanner.On("ScanDirectory", mock.Anything).Return(make(map[string]string), nil)
		mockFileScanner.On("CalculateFileChanges", mock.Anything, mock.Anything).Return([]*scanner.FileStatus{})
		mockHttpSync.On("FetchServerHashTree", mock.Anything).Return(make(map[string]string), nil)

		s.performSync()

		mockLogger.AssertCalled(t, "Info", "开始执行同步任务", mock.Anything)
		mockLogger.AssertCalled(t, "Info", "同步任务完成，总耗时: %v", mock.Anything)
	})
}

func TestPerformSyncForCodebase(t *testing.T) {
	mockLogger.On("Info", mock.Anything, mock.Anything).Return()
	mockLogger.On("Error", mock.Anything, mock.Anything).Return()

	s := &Scheduler{
		httpSync:    mockHttpSync,
		fileScanner: mockFileScanner,
		storage:     mockStorage,
		logger:      mockLogger,
	}

	t.Run("ScanDirectoryError", func(t *testing.T) {
		config := &storage.CodebaseConfig{
			CodebaseId:   "test-id",
			CodebasePath: "/test/path",
		}

		mockFileScanner.On("ScanDirectory", config.CodebasePath).
			Return(nil, errors.New("scan error")).
			Once()
		mockFileScanner.On("CalculateFileChanges", mock.Anything, mock.Anything).Return([]*scanner.FileStatus{})

		s.performSyncForCodebase(config)

		mockLogger.AssertCalled(t, "Info", "开始执行同步任务，codebase: %s", mock.Anything)
		mockLogger.AssertCalled(t, "Error", "扫描本地目录(%s)失败: %v", mock.Anything, mock.Anything)
	})
}

func TestProcessFileChanges(t *testing.T) {
	mockLogger.On("Info", mock.Anything, mock.Anything).Return()
	mockLogger.On("Warn", mock.Anything, mock.Anything).Return()
	mockLogger.On("Error", mock.Anything, mock.Anything).Return()

	s := &Scheduler{
		httpSync:    mockHttpSync,
		fileScanner: mockFileScanner,
		storage:     mockStorage,
		logger:      mockLogger,
	}

	t.Run("NormalProcessFileChanges", func(t *testing.T) {
		tmp := t.TempDir()
		utils.UploadTmpDir = tmp
		codebasePath := filepath.Join(tmp, "test", "zipSuccess")
		config := &storage.CodebaseConfig{
			CodebaseId:   "test-id",
			CodebasePath: codebasePath,
		}
		changes := []*scanner.FileStatus{
			{
				Path:   filepath.Join(codebasePath, "file1.go"),
				Status: scanner.FILE_STATUS_MODIFIED,
			},
		}

		mockHttpSync.On("UploadFile", mock.Anything, mock.Anything).Return(nil)

		err := s.processFileChanges(config, changes)

		assert.NoError(t, err)
		mockLogger.AssertCalled(t, "Info", "开始上报zip文件: %s", mock.Anything)
		mockLogger.AssertCalled(t, "Info", "zip文件上报成功", mock.Anything)
	})

	t.Run("CreateChangesZipError", func(t *testing.T) {
		tmpFile := filepath.Join(os.TempDir(), "file")
		// 构造tmpFile为文件，保证创建zip文件失败
		file, err := os.Create(tmpFile)
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		utils.UploadTmpDir = tmpFile
		codebasePath := filepath.Join(tmpFile, "test", "zipFail")
		config := &storage.CodebaseConfig{
			CodebaseId:   "test-id",
			CodebasePath: codebasePath,
		}
		changes := []*scanner.FileStatus{
			{
				Path:   filepath.Join(codebasePath, "file1.go"),
				Status: scanner.FILE_STATUS_MODIFIED,
			},
		}

		err = s.processFileChanges(config, changes)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "创建zip文件失败")
	})

	t.Run("UploadChangesZipError", func(t *testing.T) {
		tmp := t.TempDir()
		utils.UploadTmpDir = tmp
		codebasePath := filepath.Join(tmp, "test", "zipSuccess")
		config := &storage.CodebaseConfig{
			CodebaseId:   "test-id",
			CodebasePath: codebasePath,
		}
		changes := []*scanner.FileStatus{
			{
				Path:   filepath.Join(codebasePath, "file1.go"),
				Status: scanner.FILE_STATUS_MODIFIED,
			},
		}
		newMockHttpSync := &mocks.MockHTTPSync{}
		s.httpSync = newMockHttpSync
		newMockHttpSync.On("UploadFile", mock.Anything, mock.Anything).Return(errors.New("upload error"))

		err := s.processFileChanges(config, changes)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "上传zip文件失败")
		mockLogger.AssertCalled(t, "Error", "上报zip文件最终失败: %v", mock.Anything)
	})
}

func TestCreateChangesZip(t *testing.T) {
	mockLogger.On("Warn", mock.Anything, mock.Anything).Return()

	s := &Scheduler{
		httpSync:    mockHttpSync,
		fileScanner: mockFileScanner,
		storage:     mockStorage,
		logger:      mockLogger,
	}

	t.Run("NormalChanges", func(t *testing.T) {
		tmp := t.TempDir()
		utils.UploadTmpDir = tmp
		codebasePath := filepath.Join(tmp, "test", "normalChanges")
		config := &storage.CodebaseConfig{
			CodebaseId:   "test-id",
			CodebasePath: codebasePath,
		}
		changes := []*scanner.FileStatus{
			{
				Path:   filepath.Join(codebasePath, "file1.go"),
				Status: scanner.FILE_STATUS_MODIFIED,
			},
		}

		path, err := s.createChangesZip(config, changes)

		assert.NoError(t, err)
		assert.NotEmpty(t, path)
		mockLogger.AssertCalled(t, "Warn", "添加文件到zip失败: %s, 错误: %v", mock.Anything, mock.Anything)
	})
}

func TestUploadChangesZip(t *testing.T) {
	mockLogger.On("Info", mock.Anything, mock.Anything).Return()
	mockLogger.On("Warn", mock.Anything, mock.Anything).Return()

	s := &Scheduler{
		httpSync:    mockHttpSync,
		fileScanner: mockFileScanner,
		storage:     mockStorage,
		logger:      mockLogger,
	}

	t.Run("SuccessAfterRetry", func(t *testing.T) {
		tempFile := filepath.Join(t.TempDir(), "test.zip")
		zipfile, err := os.Create(tempFile)
		if err != nil {
			t.Fatal(err)
		}
		defer zipfile.Close()

		uploadReq := &syncer.UploadReq{
			ClientId:     "test-client",
			CodebasePath: "/test/path",
		}

		mockHttpSync.On("UploadFile", mock.Anything, mock.Anything).
			Return(errors.New("failed")).
			Times(2)
		mockHttpSync.On("UploadFile", mock.Anything, mock.Anything).
			Return(nil)

		err = s.uploadChangesZip(tempFile, uploadReq)

		assert.NoError(t, err)
		mockLogger.AssertCalled(t, "Info", "zip文件上报成功", mock.Anything)
		mockHttpSync.AssertExpectations(t)
	})
}
