package llm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/sirupsen/logrus"
)

// Provider represents different LLM providers
type Provider string

const (
	ProviderOpenAI    Provider = "openai"
	ProviderOpenAICompatible Provider = "openai-compatible"
	ProviderGemini    Provider = "gemini"
	ProviderClaude    Provider = "claude"
	ProviderOllama    Provider = "ollama"
)

// ChatMessage represents a chat message
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest represents a chat completion request
type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
}

// ChatResponse represents a chat completion response
type ChatResponse struct {
	Choices []struct {
		Message ChatMessage `json:"message"`
	} `json:"choices"`
}

// GeminiRequest represents Gemini API request format
type GeminiRequest struct {
	Contents []struct {
		Role  string `json:"role"`
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	} `json:"contents"`
}

// GeminiResponse represents Gemini API response format
type GeminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

// ClaudeRequest represents Claude API request format
type ClaudeRequest struct {
	Model     string        `json:"model"`
	MaxTokens int           `json:"max_tokens"`
	Messages  []ChatMessage `json:"messages"`
}

// ClaudeResponse represents Claude API response format
type ClaudeResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

// Client represents an LLM client
type Client struct {
	provider Provider
	model    string
	apiKey   string
	baseURL  string
	timeout  time.Duration
	client   *resty.Client
	logger   *logrus.Logger
}

// NewClient creates a new LLM client
func NewClient(provider Provider, model, apiKey, baseURL string, timeout time.Duration) *Client {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	
	client := resty.New().
		SetTimeout(timeout).
		SetHeader("User-Agent", "RepoSense/1.0")
	
	return &Client{
		provider: provider,
		model:    model,
		apiKey:   apiKey,
		baseURL:  baseURL,
		timeout:  timeout,
		client:   client,
		logger:   logger,
	}
}

// SetLogLevel sets the logging level
func (c *Client) SetLogLevel(level logrus.Level) {
	c.logger.SetLevel(level)
}

// Chat sends a chat completion request
func (c *Client) Chat(ctx context.Context, messages []ChatMessage) (string, error) {
	switch c.provider {
	case ProviderOpenAI, ProviderOpenAICompatible:
		return c.chatOpenAI(ctx, messages)
	case ProviderGemini:
		return c.chatGemini(ctx, messages)
	case ProviderClaude:
		return c.chatClaude(ctx, messages)
	case ProviderOllama:
		return c.chatOllama(ctx, messages)
	default:
		return "", fmt.Errorf("unsupported provider: %s", c.provider)
	}
}

// chatOpenAI handles OpenAI and OpenAI-compatible API requests
func (c *Client) chatOpenAI(ctx context.Context, messages []ChatMessage) (string, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	
	request := ChatRequest{
		Model:       c.model,
		Messages:    messages,
		Temperature: 0.3,
		MaxTokens:   500,
	}
	
	var response ChatResponse
	resp, err := c.client.R().
		SetContext(ctx).
		SetAuthToken(c.apiKey).
		SetHeader("Content-Type", "application/json").
		SetBody(request).
		SetResult(&response).
		Post(baseURL + "/chat/completions")
	
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	
	if resp.StatusCode() != 200 {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode(), resp.String())
	}
	
	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned")
	}
	
	return strings.TrimSpace(response.Choices[0].Message.Content), nil
}

// chatGemini handles Google Gemini API requests
func (c *Client) chatGemini(ctx context.Context, messages []ChatMessage) (string, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://generativelanguage.googleapis.com/v1beta"
	}
	
	// Convert messages to Gemini format
	var contents []struct {
		Role  string `json:"role"`
		Parts []struct {
			Text string `json:"text"`
		} `json:"parts"`
	}
	
	for _, msg := range messages {
		// Map OpenAI roles to Gemini roles
		var geminiRole string
		switch msg.Role {
		case "system", "user":
			geminiRole = "user"
		case "assistant":
			geminiRole = "model"
		default:
			geminiRole = "user"
		}
		
		contents = append(contents, struct {
			Role  string `json:"role"`
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		}{
			Role: geminiRole,
			Parts: []struct {
				Text string `json:"text"`
			}{
				{Text: msg.Content},
			},
		})
	}
	
	request := GeminiRequest{
		Contents: contents,
	}
	
	var response GeminiResponse
	resp, err := c.client.R().
		SetContext(ctx).
		SetQueryParam("key", c.apiKey).
		SetHeader("Content-Type", "application/json").
		SetBody(request).
		SetResult(&response).
		Post(fmt.Sprintf("%s/models/%s:generateContent", baseURL, c.model))
	
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	
	if resp.StatusCode() != 200 {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode(), resp.String())
	}
	
	if len(response.Candidates) == 0 || len(response.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no response content returned")
	}
	
	return strings.TrimSpace(response.Candidates[0].Content.Parts[0].Text), nil
}

// chatClaude handles Anthropic Claude API requests
func (c *Client) chatClaude(ctx context.Context, messages []ChatMessage) (string, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "https://api.anthropic.com/v1"
	}
	
	request := ClaudeRequest{
		Model:     c.model,
		MaxTokens: 500,
		Messages:  messages,
	}
	
	var response ClaudeResponse
	resp, err := c.client.R().
		SetContext(ctx).
		SetHeader("x-api-key", c.apiKey).
		SetHeader("anthropic-version", "2023-06-01").
		SetHeader("Content-Type", "application/json").
		SetBody(request).
		SetResult(&response).
		Post(baseURL + "/messages")
	
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	
	if resp.StatusCode() != 200 {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode(), resp.String())
	}
	
	if len(response.Content) == 0 {
		return "", fmt.Errorf("no response content returned")
	}
	
	return strings.TrimSpace(response.Content[0].Text), nil
}

// chatOllama handles Ollama API requests
func (c *Client) chatOllama(ctx context.Context, messages []ChatMessage) (string, error) {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	
	request := ChatRequest{
		Model:    c.model,
		Messages: messages,
	}
	
	var response ChatResponse
	resp, err := c.client.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetBody(request).
		SetResult(&response).
		Post(baseURL + "/api/chat")
	
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	
	if resp.StatusCode() != 200 {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode(), resp.String())
	}
	
	if len(response.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned")
	}
	
	return strings.TrimSpace(response.Choices[0].Message.Content), nil
}

// GenerateDescription generates a project description from README content
func (c *Client) GenerateDescription(ctx context.Context, readmeContent, language string) (string, error) {
	c.logger.Debugf("Generating description for content length: %d, language: %s", len(readmeContent), language)
	
	// Prepare the prompt based on language
	var systemPrompt, userPrompt string
	
	switch language {
	case "en":
		systemPrompt = "You are a helpful assistant that summarizes GitHub project README files. Generate a concise, single-line description (max 80 characters) that captures the essence of the project. Focus on what the project does, not how to use it. Avoid technical jargon when possible."
		userPrompt = fmt.Sprintf("Summarize this project in English (max 80 chars):\n\n%s", readmeContent)
	case "ja":
		systemPrompt = "あなたはGitHubプロジェクトのREADMEファイルを要約するアシスタントです。プロジェクトの本質を捉えた簡潔な一行の説明（最大80文字）を生成してください。使い方ではなく、プロジェクトが何をするかに焦点を当ててください。"
		userPrompt = fmt.Sprintf("このプロジェクトを日本語で要約してください（最大80文字）：\n\n%s", readmeContent)
	default: // "zh" or fallback
		systemPrompt = "你是一个专门总结GitHub项目README文件的助手。请生成一个简洁的单行描述（最多80个字符），捕捉项目的核心功能。专注于项目的作用，而不是如何使用。尽量避免技术术语。"
		userPrompt = fmt.Sprintf("用中文总结这个项目（最多80字符）：\n\n%s", readmeContent)
	}
	
	messages := []ChatMessage{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}
	
	description, err := c.Chat(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("failed to generate description: %w", err)
	}
	
	// Clean up the description
	description = strings.TrimSpace(description)
	description = strings.Trim(description, "\"'")
	
	// Ensure it's not too long
	if len(description) > 100 {
		description = description[:97] + "..."
	}
	
	c.logger.Debugf("Generated description: %s", description)
	return description, nil
}