package mocks

import (
	"codebase-syncer/internal/storage"
	"codebase-syncer/internal/syncer"

	"github.com/stretchr/testify/mock"
)

type MockHTTPSync struct {
	mock.Mock
}

func (m *MockHTTPSync) SetSyncConfig(config *syncer.SyncConfig) {
	m.Called(config)
}

func (m *MockHTTPSync) GetSyncConfig() *syncer.SyncConfig {
	args := m.Called()
	return args.Get(0).(*syncer.SyncConfig)
}

func (m *MockHTTPSync) FetchServerHashTree(codebasePath string) (map[string]string, error) {
	args := m.Called(codebasePath)
	return args.Get(0).(map[string]string), args.Error(1)
}

func (m *MockHTTPSync) UploadFile(filePath string, uploadReq *syncer.UploadReq) error {
	args := m.Called(filePath, uploadReq)
	return args.Error(0)
}

func (m *MockHTTPSync) GetClientConfig() (storage.ClientConfig, error) {
	args := m.Called()
	return args.Get(0).(storage.ClientConfig), args.Error(1)
}
