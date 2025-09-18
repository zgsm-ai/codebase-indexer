package wiki

import (
	"codebase-indexer/pkg/logger"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultMaxRetries = 10
const defaultRetryInterval = 2 * time.Second

// ChatCompletionRequest 聊天完成请求
type ChatCompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature float64   `json:"temperature"`
}

// Message 消息
type Message struct {
	Role    string `json:"role"` // "system", "user", "assistant"
	Content string `json:"content"`
}

// ChatCompletionResponse 聊天完成响应
type ChatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

// Choice 选择
type Choice struct {
	Index        int     `json:"index"`
	Message      Message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

// Usage 使用统计
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// LLMClient LLM客户端接口
type LLMClient interface {
	GenerateContent(ctx context.Context, prompt string, maxTokens int, temperature float64) (string, error)
	GenerateContentWithRetry(ctx context.Context, prompt string, maxTokens int, temperature float64, maxRetries int, retryDelay time.Duration) (string, error)
	Close() error
}

// OpenAIClient OpenAI客户端的具体实现
type OpenAIClient struct {
	apiKey  string
	baseURL string
	model   string
	config  *SimpleConfig
	logger  logger.Logger
}

// NewLLMClient 创建OpenAI客户端
func NewLLMClient(apiKey, baseURL, model string, logger logger.Logger) (LLMClient, error) {
	return NewLLMClientWithConfig(apiKey, baseURL, model, DefaultSimpleConfig(), logger)
}

// NewLLMClientWithConfig 创建带配置的OpenAI客户端
func NewLLMClientWithConfig(apiKey, baseURL, model string, config *SimpleConfig, logger logger.Logger) (LLMClient, error) {
	// 参数验证
	if apiKey == "" {
		return nil, fmt.Errorf("api_key cannot be empty")
	}
	if baseURL == "" {
		return nil, fmt.Errorf("base_url cannot be empty")
	}
	if model == "" {
		return nil, fmt.Errorf("model cannot be empty")
	}
	if config == nil {
		return nil, fmt.Errorf("config cannot be empty")
	}

	return &OpenAIClient{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		config:  config,
		logger:  logger,
	}, nil
}

// GenerateContent 生成内容
func (c *OpenAIClient) GenerateContent(ctx context.Context, prompt string, maxTokens int, temperature float64) (string, error) {
	return c.GenerateContentWithRetry(ctx, prompt, maxTokens, temperature, defaultMaxRetries, defaultRetryInterval)
}

// GenerateContentWithRetry 带重试机制的内容生成
func (c *OpenAIClient) GenerateContentWithRetry(ctx context.Context, prompt string, maxTokens int, temperature float64, maxRetries int, retryDelay time.Duration) (string, error) {
	var lastErr error
	var totalDuration time.Duration

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// 检查上下文是否已取消
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			default:
			}

			// 等待重试延迟
			time.Sleep(retryDelay)
			// 指数退避
			retryDelay *= 2
		}

		startTime := time.Now()
		content, err := c.doGenerateContent(ctx, prompt, maxTokens, temperature)
		duration := time.Since(startTime)
		totalDuration += duration

		if err == nil {
			// 成功时记录总耗时（包含所有重试）
			c.logger.Debug("LLM request succeeded after %d attempts, total duration: %v", attempt+1, totalDuration)
			return content, nil
		}

		lastErr = err

		// 记录本次失败的调用
		c.logger.Debug("LLM request attempt %d failed after %v: %v", attempt+1, duration, err)

		//// 如果是网络错误或服务器错误，则重试
		//if isRetryableErrorInLLM(err) {
		//	continue
		//}
		//
		//// 非重试错误直接返回
		//break
	}

	c.logger.Debug("LLM request failed after %d attempts, total duration: %v", maxRetries+1, totalDuration)
	return "", fmt.Errorf("still failed after %d retries: %w", maxRetries, lastErr)
}

// doGenerateContent 实际的内容生成逻辑
func (c *OpenAIClient) doGenerateContent(ctx context.Context, prompt string, maxTokens int, temperature float64) (string, error) {
	startTime := time.Now()

	// 添加调试日志：开始请求（单行包含所有关键参数）
	c.logger.Debug("LLM request started - BaseURL: %s, Model: %s, MaxTokens: %d, Temperature: %.2f",
		c.baseURL, c.model, maxTokens, temperature)

	req := &ChatCompletionRequest{
		Model: c.model, // 使用指定模型
		Messages: []Message{
			{Role: "user", Content: prompt},
		},
		MaxTokens:   maxTokens,
		Temperature: temperature,
	}

	// 构建请求URL
	url := fmt.Sprintf("%s/chat/completions", c.baseURL)

	// 构建请求体
	requestBody, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// 创建HTTP请求
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(requestBody)))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	// 创建带有超时设置的HTTP客户端
	client := &http.Client{
		Timeout: c.config.RequestTimeout,
	}

	// HTTP请求阶段计时
	// 发送请求
	resp, err := client.Do(httpReq)
	if err != nil {
		c.logger.Debug("LLM request failed - error: %v", err)
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Debug("LLM request failed - read error: %v", err)
		return "", fmt.Errorf("failed to read response: %w", err)
	}
	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var response ChatCompletionResponse
	if err := json.Unmarshal(body, &response); err != nil {
		c.logger.Debug("LLM request failed - parse error: %v", err)
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(response.Choices) == 0 {
		c.logger.Debug("LLM request failed - no choices in response")
		return "", fmt.Errorf("no response from LLM")
	}

	// 计算总耗时并输出完成日志（单行包含所有关键信息）
	totalDuration := time.Since(startTime)
	c.logger.Debug("LLM request completed - ID: %s, Tokens: %d, Total: %v",
		response.ID, response.Usage.TotalTokens, totalDuration)

	return response.Choices[0].Message.Content, nil
}

// isRetryableErrorInLLM 判断错误是否可重试
func isRetryableErrorInLLM(err error) bool {
	errStr := err.Error()
	// 网络错误
	if strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "network is unreachable") ||
		strings.Contains(errStr, "no such host") {
		return true
	}

	// HTTP 5xx 服务器错误
	if strings.Contains(errStr, "status 5") {
		return true
	}

	// 速率限制错误
	if strings.Contains(errStr, "rate limit") ||
		strings.Contains(errStr, "too many requests") {
		return true
	}

	return false
}

// Close 关闭客户端
func (c *OpenAIClient) Close() error {
	// HTTP客户端不需要特殊关闭
	return nil
}
