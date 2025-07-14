package mocks

import (
	"codebase-indexer/internal/storage"

	"github.com/stretchr/testify/mock"
)

type MockStorageManager struct {
	mock.Mock
}

func (m *MockStorageManager) GetCodebaseConfigs() map[string]*storage.CodebaseConfig {
	args := m.Called()
	if args.Get(0) != nil {
		return args.Get(0).(map[string]*storage.CodebaseConfig)
	}
	return nil
}

func (m *MockStorageManager) GetCodebaseConfig(codebaseId string) (*storage.CodebaseConfig, error) {
	args := m.Called(codebaseId)
	if args.Get(0) != nil {
		return args.Get(0).(*storage.CodebaseConfig), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockStorageManager) SaveCodebaseConfig(config *storage.CodebaseConfig) error {
	args := m.Called(config)
	return args.Error(0)
}

func (m *MockStorageManager) DeleteCodebaseConfig(codebaseId string) error {
	args := m.Called(codebaseId)
	return args.Error(0)
}
