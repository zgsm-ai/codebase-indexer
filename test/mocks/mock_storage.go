package mocks

import (
	"codebase-indexer/internal/config"

	"github.com/stretchr/testify/mock"
)

type MockStorageManager struct {
	mock.Mock
}

func (m *MockStorageManager) GetCodebaseConfigs() map[string]*config.CodebaseConfig {
	args := m.Called()
	if args.Get(0) != nil {
		return args.Get(0).(map[string]*config.CodebaseConfig)
	}
	return nil
}

func (m *MockStorageManager) GetCodebaseConfig(codebaseId string) (*config.CodebaseConfig, error) {
	args := m.Called(codebaseId)
	if args.Get(0) != nil {
		return args.Get(0).(*config.CodebaseConfig), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockStorageManager) SaveCodebaseConfig(config *config.CodebaseConfig) error {
	args := m.Called(config)
	return args.Error(0)
}

func (m *MockStorageManager) DeleteCodebaseConfig(codebaseId string) error {
	args := m.Called(codebaseId)
	return args.Error(0)
}

func (m *MockStorageManager) GetCodebaseEnv() *config.CodebaseEnv {
	args := m.Called()
	return args.Get(0).(*config.CodebaseEnv)
}

func (m *MockStorageManager) SaveCodebaseEnv(codebaseEnv *config.CodebaseEnv) error {
	args := m.Called(codebaseEnv)
	return args.Error(0)
}
