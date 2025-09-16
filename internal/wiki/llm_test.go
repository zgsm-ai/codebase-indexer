package wiki

import (
	"codebase-indexer/test/mocks"
	"context"
	"fmt"
	"testing"
	"time"
)

func TestLLMClientInterface(t *testing.T) {
	// 测试接口实现
	var client LLMClient
	mockLogger := &mocks.MockLogger{}
	// 测试OpenAI客户端实现
	openaiClient, err := NewLLMClient("test-key", "https://api.test.com", "test-model", mockLogger)
	if err != nil {
		t.Errorf("Failed to create OpenAI client: %v", err)
	}
	client = openaiClient
	if client == nil {
		t.Error("OpenAI client should not be nil")
	}

	// 测试Mock客户端实现
	mockClient := mocks.NewMockLLMClient("mock response", nil)
	client = mockClient
	if client == nil {
		t.Error("Mock client should not be nil")
	}
}

func TestMockLLMClient(t *testing.T) {
	// 测试成功的Mock响应
	successClient := mocks.NewMockLLMClient("Success response", nil)

	response, err := successClient.GenerateContent(context.Background(), "Test prompt", 100, 0.7)
	if err != nil {
		t.Errorf("Mock client should not return error: %v", err)
	}
	if response != "Success response" {
		t.Errorf("Expected response 'Success response', got '%s'", response)
	}

	// 测试带重试的Mock响应
	response, err = successClient.GenerateContentWithRetry(context.Background(), "Test prompt", 100, 0.7, 3, 2*time.Second)
	if err != nil {
		t.Errorf("Mock client with retry should not return error: %v", err)
	}
	if response != "Success response" {
		t.Errorf("Expected response 'Success response', got '%s'", response)
	}

	// 测试失败的Mock响应
	errorClient := mocks.NewMockLLMClient("", fmt.Errorf("mock LLM error"))

	_, err = errorClient.GenerateContent(context.Background(), "Test prompt", 100, 0.7)
	if err == nil {
		t.Error("Mock client should return error")
	}

	// 测试关闭
	err = successClient.Close()
	if err != nil {
		t.Errorf("Mock client close should not return error: %v", err)
	}
}

func TestOpenAIClient(t *testing.T) {
	// 测试参数验证
	mockLogger := &mocks.MockLogger{}
	_, err := NewLLMClient("", "https://api.test.com", "test-model", mockLogger)
	if err == nil {
		t.Error("Empty API key should return error")
	}

	_, err = NewLLMClient("test-key", "", "test-model", mockLogger)
	if err == nil {
		t.Error("Empty base URL should return error")
	}

	_, err = NewLLMClient("test-key", "https://api.test.com", "", mockLogger)
	if err == nil {
		t.Error("Empty model should return error")
	}

	// 测试有效的客户端创建
	client, err := NewLLMClient("test-key", "https://api.test.com", "test-model", mockLogger)
	if err != nil {
		t.Errorf("Failed to create OpenAI client: %v", err)
	}

	// 验证接口实现
	if _, ok := client.(*OpenAIClient); !ok {
		t.Error("NewLLMClient should return *OpenAIClient type")
	}

	// 测试关闭
	err = client.Close()
	if err != nil {
		t.Errorf("OpenAI client close should not return error: %v", err)
	}

	// 测试带配置的客户端创建
	config := DefaultSimpleConfig()
	config.RequestTimeout = 30 * time.Second
	clientWithConfig, err := NewLLMClientWithConfig("test-key", "https://api.test.com", "test-model", config, mockLogger)
	if err != nil {
		t.Errorf("Failed to create OpenAI client with config: %v", err)
	}

	// 验证接口实现
	if _, ok := clientWithConfig.(*OpenAIClient); !ok {
		t.Error("NewLLMClientWithConfig should return *OpenAIClient type")
	}

	// 测试关闭
	err = clientWithConfig.Close()
	if err != nil {
		t.Errorf("OpenAI client with config close should not return error: %v", err)
	}

	// 测试空配置错误
	_, err = NewLLMClientWithConfig("test-key", "https://api.test.com", "test-model", nil, mockLogger)
	if err == nil {
		t.Error("Nil config should return error")
	}
}

// TestGeneratorWithMockLLM 测试使用Mock LLM的Generator
func TestGeneratorWithMockLLM(t *testing.T) {
	// 创建Mock LLM客户端
	mockLLM := mocks.NewMockLLMClient("This is a mock Wiki generation response", nil)

	// 创建配置
	config := DefaultSimpleConfig()
	config.MaxFiles = 10 // 限制文件数量以加快测试速度
	logger := &mocks.MockLogger{}
	// 创建Generator，使用Mock LLM
	generator := &Generator{
		llmClient: mockLLM,
		processor: NewProcessor(config, logger),
		config:    config,
	}

	// 注意：这里我们不测试实际的GenerateWiki，因为它需要真实的文件系统
	// 我们只验证Generator可以正确地使用Mock LLM客户端

	// 验证LLM客户端设置正确
	if generator.llmClient == nil {
		t.Error("Generator的llmClient不应该为nil")
	}

	// 测试关闭
	err := generator.Close()
	if err != nil {
		t.Errorf("Generator close should not return error: %v", err)
	}
}
