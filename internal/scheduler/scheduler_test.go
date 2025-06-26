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
	sechedulerConfig = &SchedulerConfig{
		IntervalMinutes:       5,
		RegisterExpireMinutes: 30,
		HashTreeExpireHours:   24,
		MaxRetries:            3,
		RetryIntervalSeconds:  5,
	}
)

func TestPerformSync(t *testing.T) {
	var (
		mockLogger      = &mocks.MockLogger{}
		mockStorage     = &mocks.MockStorageManager{}
		mockHttpSync    = &mocks.MockHTTPSync{}
		mockFileScanner = &mocks.MockScanner{}
	)
	mockLogger.On("Info", mock.Anything, mock.Anything).Return()

	s := &Scheduler{
		httpSync:         mockHttpSync,
		fileScanner:      mockFileScanner,
		storage:          mockStorage,
		sechedulerConfig: sechedulerConfig,
		logger:           mockLogger,
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

		mockLogger.AssertCalled(t, "Info", "starting sync task", mock.Anything)
		mockLogger.AssertCalled(t, "Info", "sync task completed, total time: %v", mock.Anything)
	})
}

func TestPerformSyncForCodebase(t *testing.T) {
	var (
		mockLogger      = &mocks.MockLogger{}
		mockStorage     = &mocks.MockStorageManager{}
		mockHttpSync    = &mocks.MockHTTPSync{}
		mockFileScanner = &mocks.MockScanner{}
	)
	mockLogger.On("Info", mock.Anything, mock.Anything).Return()
	mockLogger.On("Error", mock.Anything, mock.Anything).Return()

	s := &Scheduler{
		httpSync:         mockHttpSync,
		fileScanner:      mockFileScanner,
		storage:          mockStorage,
		sechedulerConfig: sechedulerConfig,
		logger:           mockLogger,
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

		mockLogger.AssertCalled(t, "Info", "starting sync for codebase: %s", mock.Anything)
		mockLogger.AssertCalled(t, "Error", "failed to scan local directory (%s): %v", mock.Anything, mock.Anything)
	})
}

func TestProcessFileChanges(t *testing.T) {
	var (
		mockLogger      = &mocks.MockLogger{}
		mockStorage     = &mocks.MockStorageManager{}
		mockHttpSync    = &mocks.MockHTTPSync{}
		mockFileScanner = &mocks.MockScanner{}
	)
	mockLogger.On("Info", mock.Anything, mock.Anything).Return()
	mockLogger.On("Warn", mock.Anything, mock.Anything).Return()
	mockLogger.On("Error", mock.Anything, mock.Anything).Return()

	s := &Scheduler{
		httpSync:         mockHttpSync,
		fileScanner:      mockFileScanner,
		storage:          mockStorage,
		sechedulerConfig: sechedulerConfig,
		logger:           mockLogger,
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
		mockLogger.AssertCalled(t, "Info", "starting to upload zip file: %s", mock.Anything)
		mockLogger.AssertCalled(t, "Info", "zip file uploaded successfully", mock.Anything)
	})

	t.Run("CreateChangesZipError", func(t *testing.T) {
		tmpFile := filepath.Join(os.TempDir(), "file")
		// Make tmpFile a file to ensure zip creation fails
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
		assert.Contains(t, err.Error(), "failed to create changes zip")
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
		assert.Contains(t, err.Error(), "failed to upload changes zip")
	})
}

func TestCreateChangesZip(t *testing.T) {
	var (
		mockLogger      = &mocks.MockLogger{}
		mockStorage     = &mocks.MockStorageManager{}
		mockHttpSync    = &mocks.MockHTTPSync{}
		mockFileScanner = &mocks.MockScanner{}
	)
	mockLogger.On("Warn", mock.Anything, mock.Anything).Return()

	s := &Scheduler{
		httpSync:         mockHttpSync,
		fileScanner:      mockFileScanner,
		storage:          mockStorage,
		sechedulerConfig: sechedulerConfig,
		logger:           mockLogger,
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
		mockLogger.AssertCalled(t, "Warn", "failed to add file to zip: %s, error: %v", mock.Anything, mock.Anything)
	})
}

func TestUploadChangesZip(t *testing.T) {
	var (
		mockLogger      = &mocks.MockLogger{}
		mockStorage     = &mocks.MockStorageManager{}
		mockHttpSync    = &mocks.MockHTTPSync{}
		mockFileScanner = &mocks.MockScanner{}
	)
	mockLogger.On("Info", mock.Anything, mock.Anything).Return()
	mockLogger.On("Warn", mock.Anything, mock.Anything).Return()

	s := &Scheduler{
		httpSync:         mockHttpSync,
		fileScanner:      mockFileScanner,
		storage:          mockStorage,
		sechedulerConfig: sechedulerConfig,
		logger:           mockLogger,
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
		mockLogger.AssertCalled(t, "Info", "zip file uploaded successfully", mock.Anything)
		mockHttpSync.AssertExpectations(t)
	})
}

func TestIsAbortRetryError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "NilError",
			err:      nil,
			expected: false,
		},
		{
			name:     "UnauthorizedError",
			err:      errors.New("401 Unauthorized"),
			expected: true,
		},
		{
			name:     "TooManyRequestsError",
			err:      errors.New("429 Too Many Requests"),
			expected: true,
		},
		{
			name:     "ServiceUnavailableError",
			err:      errors.New("503 Service Unavailable"),
			expected: true,
		},
		{
			name:     "OtherError",
			err:      errors.New("some other error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isAbortRetryError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
