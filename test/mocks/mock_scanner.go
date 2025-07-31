package mocks

import (
	"codebase-indexer/internal/config"
	"codebase-indexer/internal/utils"

	gitignore "github.com/sabhiram/go-gitignore"
	"github.com/stretchr/testify/mock"
)

type MockScanner struct {
	mock.Mock
}

func (m *MockScanner) SetScannerConfig(config *config.ScannerConfig) {
	m.Called(config)
}

func (m *MockScanner) GetScannerConfig() *config.ScannerConfig {
	args := m.Called()
	return args.Get(0).(*config.ScannerConfig)
}

func (m *MockScanner) CalculateFileHash(filePath string) (string, error) {
	args := m.Called(filePath)
	return args.String(0), args.Error(1)
}

func (m *MockScanner) LoadIgnoreRules(codebasePath string) *gitignore.GitIgnore {
	args := m.Called(codebasePath)
	return args.Get(0).(*gitignore.GitIgnore)
}

func (m *MockScanner) LoadFileIgnoreRules(codebasePath string) *gitignore.GitIgnore {
	args := m.Called(codebasePath)
	return args.Get(0).(*gitignore.GitIgnore)
}

func (m *MockScanner) LoadFolderIgnoreRules(codebasePath string) *gitignore.GitIgnore {
	args := m.Called(codebasePath)
	return args.Get(0).(*gitignore.GitIgnore)
}

func (m *MockScanner) ScanCodebase(codebasePath string) (map[string]string, error) {
	args := m.Called(codebasePath)
	if args.Get(0) != nil {
		return args.Get(0).(map[string]string), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockScanner) ScanFilePaths(codebasePath string, filePaths []string) (map[string]string, error) {
	args := m.Called(codebasePath, filePaths)
	if args.Get(0) != nil {
		return args.Get(0).(map[string]string), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockScanner) ScanDirectory(codebasePath, dirPath string) (map[string]string, error) {
	args := m.Called(codebasePath, dirPath)
	if args.Get(0) != nil {
		return args.Get(0).(map[string]string), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockScanner) ScanFile(codebasePath, filePath string) (string, error) {
	args := m.Called(codebasePath, filePath)
	return args.String(0), args.Error(1)
}

func (m *MockScanner) CalculateFileChanges(local, remote map[string]string) []*utils.FileStatus {
	args := m.Called(local, remote)
	if args.Get(0) != nil {
		return args.Get(0).([]*utils.FileStatus)
	}
	return nil
}

func (m *MockScanner) CalculateFileChangesWithoutDelete(local, remote map[string]string) []*utils.FileStatus {
	args := m.Called(local, remote)
	if args.Get(0) != nil {
		return args.Get(0).([]*utils.FileStatus)
	}
	return nil
}
