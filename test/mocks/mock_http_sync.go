package mocks

import (
	"codebase-indexer/internal/config"
	"codebase-indexer/internal/dto"

	"github.com/stretchr/testify/mock"
)

type MockHTTPSync struct {
	mock.Mock
}

func (m *MockHTTPSync) SetSyncConfig(config *config.SyncConfig) {
	m.Called(config)
}

func (m *MockHTTPSync) GetSyncConfig() *config.SyncConfig {
	args := m.Called()
	return args.Get(0).(*config.SyncConfig)
}

func (m *MockHTTPSync) FetchServerHashTree(codebasePath string) (map[string]string, error) {
	args := m.Called(codebasePath)
	return args.Get(0).(map[string]string), args.Error(1)
}

func (m *MockHTTPSync) FetchUploadToken(req dto.UploadTokenReq) (*dto.UploadTokenResp, error) {
	args := m.Called(req)
	return args.Get(0).(*dto.UploadTokenResp), args.Error(1)
}

func (m *MockHTTPSync) UploadFile(filePath string, uploadReq dto.UploadReq) error {
	args := m.Called(filePath, uploadReq)
	return args.Error(0)
}

func (m *MockHTTPSync) FetchFileStatus(req dto.FileStatusReq) (*dto.FileStatusResp, error) {
	args := m.Called(req)
	return args.Get(0).(*dto.FileStatusResp), args.Error(1)
}

func (m *MockHTTPSync) GetClientConfig() (config.ClientConfig, error) {
	args := m.Called()
	return args.Get(0).(config.ClientConfig), args.Error(1)
}

func (m *MockHTTPSync) DeleteEmbedding(req dto.DeleteEmbeddingReq) (*dto.DeleteEmbeddingResp, error) {
	args := m.Called(req)
	return args.Get(0).(*dto.DeleteEmbeddingResp), args.Error(1)
}
