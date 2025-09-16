package mocks

import (
	"codebase-indexer/internal/wiki"
	"context"
	"time"
)

// MockLLMClient 用于测试的Mock LLM客户端
type MockLLMClient struct {
	Response string
	Error    error
}

// NewMockLLMClient 创建Mock LLM客户端
func NewMockLLMClient(response string, err error) wiki.LLMClient {
	return &MockLLMClient{
		Response: response,
		Error:    err,
	}
}

// GenerateContent 生成内容（Mock实现）
func (m *MockLLMClient) GenerateContent(ctx context.Context, prompt string, maxTokens int, temperature float64) (string, error) {
	return m.Response, m.Error
}

// GenerateContentWithRetry 带重试机制的内容生成（Mock实现）
func (m *MockLLMClient) GenerateContentWithRetry(ctx context.Context, prompt string, maxTokens int, temperature float64, maxRetries int, retryDelay time.Duration) (string, error) {
	return m.Response, m.Error
}

// Close 关闭客户端（Mock实现）
func (m *MockLLMClient) Close() error {
	return nil
}
