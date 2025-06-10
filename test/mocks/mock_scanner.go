package mocks

import (
	"codebase-syncer/internal/scanner"

	"github.com/stretchr/testify/mock"
)

type MockScanner struct {
	mock.Mock
}

func (m *MockScanner) CalculateFileHash(filePath string) (string, error) {
	args := m.Called(filePath)
	return args.String(0), args.Error(1)
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
