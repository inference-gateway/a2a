package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	config "github.com/inference-gateway/a2a/adk/server/config"
	sdk "github.com/inference-gateway/sdk"
	zap "go.uber.org/zap"
)

// LLMClient defines the interface for Language Model clients
type LLMClient interface {
	// CreateChatCompletion sends a chat completion request using SDK messages
	CreateChatCompletion(ctx context.Context, messages []sdk.Message, tools ...sdk.ChatCompletionTool) (*sdk.CreateChatCompletionResponse, error)

	// CreateStreamingChatCompletion sends a streaming chat completion request using SDK messages
	CreateStreamingChatCompletion(ctx context.Context, messages []sdk.Message) (<-chan *sdk.CreateChatCompletionStreamResponse, <-chan error)
}

var _ LLMClient = (*OpenAICompatibleLLMClient)(nil)

// OpenAICompatibleLLMClient implements LLMClient using an OpenAI-compatible API via the Inference Gateway SDK
type OpenAICompatibleLLMClient struct {
	client   sdk.Client
	config   *config.AgentConfig
	logger   *zap.Logger
	provider sdk.Provider
	model    string
}

// NewOpenAICompatibleLLMClient creates a new OpenAI-compatible LLM client
func NewOpenAICompatibleLLMClient(cfg *config.AgentConfig, logger *zap.Logger) (*OpenAICompatibleLLMClient, error) {
	if cfg == nil {
		return nil, fmt.Errorf("llm provider client config is required")
	}

	if cfg.Provider == "" {
		return nil, fmt.Errorf("provider is required")
	}

	if cfg.Model == "" {
		return nil, fmt.Errorf("model is required")
	}

	clientOptions := &sdk.ClientOptions{}

	if cfg.BaseURL != "" {
		clientOptions.BaseURL = cfg.BaseURL
	}

	if cfg.APIKey != "" {
		clientOptions.APIKey = cfg.APIKey
	}

	if cfg.Timeout > 0 {
		clientOptions.Timeout = cfg.Timeout
	}

	if len(cfg.CustomHeaders) > 0 {
		clientOptions.Headers = cfg.CustomHeaders
	}

	client := sdk.NewClient(clientOptions)

	provider, err := parseProvider(cfg.Provider)
	if err != nil {
		return nil, fmt.Errorf("invalid provider %s: %w", cfg.Provider, err)
	}

	model := parseModelName(cfg.Model, cfg.Provider)

	return &OpenAICompatibleLLMClient{
		client:   client,
		config:   cfg,
		logger:   logger,
		provider: provider,
		model:    model,
	}, nil
}

// CreateChatCompletion implements LLMClient.CreateChatCompletion using SDK messages
func (c *OpenAICompatibleLLMClient) CreateChatCompletion(ctx context.Context, messages []sdk.Message, tools ...sdk.ChatCompletionTool) (*sdk.CreateChatCompletionResponse, error) {
	options := &sdk.CreateChatCompletionRequest{}

	if c.config.MaxTokens > 0 {
		options.MaxTokens = &c.config.MaxTokens
	}

	var response *sdk.CreateChatCompletionResponse
	var lastErr error

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			c.logger.Debug("retrying llm request",
				zap.Int("attempt", attempt),
				zap.Int("max_retries", c.config.MaxRetries),
				zap.Error(lastErr))

			backoff := time.Duration(attempt) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		if len(tools) > 0 {
			response, lastErr = c.client.WithOptions(options).WithTools(&tools).GenerateContent(
				ctx,
				c.provider,
				c.model,
				messages,
			)
		} else {
			response, lastErr = c.client.WithOptions(options).GenerateContent(
				ctx,
				c.provider,
				c.model,
				messages,
			)
		}

		if lastErr == nil {
			break
		}

		c.logger.Debug("llm request failed",
			zap.Error(lastErr),
			zap.Int("attempt", attempt+1))
	}

	if lastErr != nil {
		return nil, fmt.Errorf("llm request failed after %d retries: %w", c.config.MaxRetries, lastErr)
	}

	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned from llm")
	}

	c.logger.Info("llm chat completion successful",
		zap.Int("choices", len(response.Choices)),
		zap.Bool("has_tools", len(tools) > 0))

	return response, nil
}

// CreateStreamingChatCompletion implements LLMClient.CreateStreamingChatCompletion using SDK messages
func (c *OpenAICompatibleLLMClient) CreateStreamingChatCompletion(ctx context.Context, messages []sdk.Message) (<-chan *sdk.CreateChatCompletionStreamResponse, <-chan error) {
	responseChan := make(chan *sdk.CreateChatCompletionStreamResponse)
	errorChan := make(chan error, 1)

	go func() {
		defer close(responseChan)
		defer close(errorChan)

		options := &sdk.CreateChatCompletionRequest{}

		if c.config.MaxTokens > 0 {
			options.MaxTokens = &c.config.MaxTokens
		}

		events, err := c.client.WithOptions(options).GenerateContentStream(
			ctx,
			c.provider,
			c.model,
			messages,
		)
		if err != nil {
			errorChan <- fmt.Errorf("failed to create stream: %w", err)
			return
		}

		for event := range events {
			if event.Event == nil {
				continue
			}

			switch *event.Event {
			case sdk.ContentDelta:
				if event.Data != nil {
					var streamResponse sdk.CreateChatCompletionStreamResponse
					if err := json.Unmarshal(*event.Data, &streamResponse); err != nil {
						c.logger.Debug("error parsing stream response", zap.Error(err))
						continue
					}

					select {
					case responseChan <- &streamResponse:
					case <-ctx.Done():
						return
					}
				}

			case sdk.StreamEnd:
				return

			default:
				if event.Data != nil {
					var errResp struct {
						Error string `json:"error"`
					}
					if err := json.Unmarshal(*event.Data, &errResp); err == nil && errResp.Error != "" {
						errorChan <- fmt.Errorf("llm error: %s", errResp.Error)
						return
					}
				}
			}
		}
	}()

	return responseChan, errorChan
}

// parseProvider converts a provider string to SDK Provider type
func parseProvider(provider string) (sdk.Provider, error) {
	p := sdk.Provider(strings.ToLower(provider))
	if p == "" {
		return "", fmt.Errorf("invalid provider: %s", provider)
	}
	return p, nil
}

// parseModelName removes provider prefix from model name if present
func parseModelName(model, provider string) string {
	if strings.Contains(model, "/") {
		parts := strings.SplitN(model, "/", 2)
		if len(parts) == 2 {
			return parts[1]
		}
	}
	return model
}
