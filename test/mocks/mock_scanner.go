package mocks

import (
	"codebase-indexer/internal/scanner"

	gitignore "github.com/sabhiram/go-gitignore"
	"github.com/stretchr/testify/mock"
)

type MockScanner struct {
	mock.Mock
}

func (m *MockScanner) SetScannerConfig(config *scanner.ScannerConfig) {
	m.Called(config)
}

func (m *MockScanner) GetScannerConfig() *scanner.ScannerConfig {
	args := m.Called()
	return args.Get(0).(*scanner.ScannerConfig)
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

func (m *MockScanner) ScanDirectory(codebasePath string) (map[string]string, error) {
	args := m.Called(codebasePath)
	if args.Get(0) != nil {
		return args.Get(0).(map[string]string), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockScanner) CalculateFileChanges(local, remote map[string]string) []*scanner.FileStatus {
	args := m.Called(local, remote)
	if args.Get(0) != nil {
		return args.Get(0).([]*scanner.FileStatus)
	}
	return nil
}
