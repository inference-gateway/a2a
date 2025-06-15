package server

import (
	"context"
	"encoding/json"
	"fmt"

	adk "github.com/inference-gateway/a2a/adk"
	config "github.com/inference-gateway/a2a/adk/server/config"
	utils "github.com/inference-gateway/a2a/adk/server/utils"
	sdk "github.com/inference-gateway/sdk"
	zap "go.uber.org/zap"
)

// OpenAICompatibleAgent represents an agent that can interact with OpenAI-compatible LLM APIs and execute tools
type OpenAICompatibleAgent interface {
	// ProcessTask processes a task with optional tool calling capabilities
	ProcessTask(ctx context.Context, task *adk.Task, message *adk.Message) (*adk.Task, error)

	// SetSystemPrompt sets the system prompt for the agent
	SetSystemPrompt(prompt string)

	// GetSystemPrompt returns the current system prompt
	GetSystemPrompt() string

	// SetToolBox sets the toolbox for the agent
	SetToolBox(toolBox ToolBox)

	// GetToolBox returns the current toolbox
	GetToolBox() ToolBox

	// SetLLMClient sets the LLM client for the agent
	SetLLMClient(client LLMClient)

	// GetLLMClient returns the current LLM client
	GetLLMClient() LLMClient
}

// DefaultOpenAICompatibleAgent is the default implementation of OpenAICompatibleAgent
type DefaultOpenAICompatibleAgent struct {
	logger                      *zap.Logger
	llmClient                   LLMClient
	toolBox                     ToolBox
	systemPrompt                string
	converter                   utils.MessageConverter
	maxChatCompletionIterations int
}

// NewDefaultOpenAICompatibleAgent creates a new DefaultOpenAICompatibleAgent
func NewDefaultOpenAICompatibleAgent(logger *zap.Logger) *DefaultOpenAICompatibleAgent {
	return &DefaultOpenAICompatibleAgent{
		logger:                      logger,
		systemPrompt:                "You are a helpful AI assistant.",
		converter:                   utils.NewOptimizedMessageConverter(logger),
		maxChatCompletionIterations: 10, // Default value
	}
}

// NewDefaultOpenAICompatibleAgentWithConfig creates a new DefaultOpenAICompatibleAgent with configuration
func NewDefaultOpenAICompatibleAgentWithConfig(logger *zap.Logger, maxIterations int) *DefaultOpenAICompatibleAgent {
	return &DefaultOpenAICompatibleAgent{
		logger:                      logger,
		systemPrompt:                "You are a helpful AI assistant.",
		converter:                   utils.NewOptimizedMessageConverter(logger),
		maxChatCompletionIterations: maxIterations,
	}
}

// NewOpenAICompatibleAgentWithLLM creates a new agent with an LLM client
func NewOpenAICompatibleAgentWithLLM(logger *zap.Logger, llmClient LLMClient) *DefaultOpenAICompatibleAgent {
	agent := NewDefaultOpenAICompatibleAgent(logger)
	agent.llmClient = llmClient
	return agent
}

// NewOpenAICompatibleAgentWithConfig creates a new agent with LLM configuration
func NewOpenAICompatibleAgentWithConfig(logger *zap.Logger, config *config.AgentConfig) (*DefaultOpenAICompatibleAgent, error) {
	client, err := NewOpenAICompatibleLLMClient(config, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create llm client: %w", err)
	}

	agent := NewDefaultOpenAICompatibleAgent(logger)
	agent.llmClient = client
	agent.maxChatCompletionIterations = config.MaxChatCompletionIterations
	return agent, nil
}

// ProcessTask processes a task with optional tool calling capabilities
func (a *DefaultOpenAICompatibleAgent) ProcessTask(ctx context.Context, task *adk.Task, message *adk.Message) (*adk.Task, error) {
	if a.llmClient == nil {
		return a.processWithoutLLM(task, message), nil
	}

	messages := make([]adk.Message, 0)

	if a.systemPrompt != "" {
		systemMessage := adk.Message{
			Kind:      "message",
			MessageID: "system-prompt",
			Role:      "system",
			Parts: []adk.Part{
				map[string]interface{}{
					"kind": "text",
					"text": a.systemPrompt,
				},
			},
		}
		messages = append(messages, systemMessage)
	}

	messages = append(messages, task.History...)

	messages = append(messages, *message)

	if a.toolBox != nil && len(a.toolBox.GetTools()) > 0 {
		return a.processWithToolCalling(ctx, task, messages)
	}

	return a.processWithoutToolCalling(ctx, task, messages)
}

// processWithoutToolCalling processes the task without tool calling
func (a *DefaultOpenAICompatibleAgent) processWithoutToolCalling(ctx context.Context, task *adk.Task, messages []adk.Message) (*adk.Task, error) {
	sdkMessages, err := a.converter.ConvertToSDK(messages)
	if err != nil {
		a.logger.Error("failed to convert A2A messages to SDK format", zap.Error(err))
		task.Status.State = adk.TaskStateFailed
		errorMessage := &adk.Message{
			Kind:      "message",
			MessageID: "error-" + task.ID,
			Role:      "assistant",
			Parts: []adk.Part{
				map[string]interface{}{
					"kind": "text",
					"text": fmt.Sprintf("Message conversion failed: %v", err),
				},
			},
		}
		task.Status.Message = errorMessage
		return task, nil
	}

	result, err := a.llmClient.CreateChatCompletion(ctx, sdkMessages)
	if err != nil {
		a.logger.Error("llm completion failed",
			zap.Error(err),
			zap.String("task_id", task.ID),
			zap.String("context_id", task.ContextID))
		task.Status.State = adk.TaskStateFailed
		errorMessage := &adk.Message{
			Kind:      "message",
			MessageID: "error-" + task.ID,
			Role:      "assistant",
			Parts: []adk.Part{
				map[string]interface{}{
					"kind": "text",
					"text": fmt.Sprintf("LLM request failed: %v", err),
				},
			},
		}
		task.Status.Message = errorMessage
		return task, nil
	}

	if len(result.Choices) == 0 {
		a.logger.Error("no choices returned from llm",
			zap.String("task_id", task.ID),
			zap.String("context_id", task.ContextID))
		task.Status.State = adk.TaskStateFailed
		errorMessage := &adk.Message{
			Kind:      "message",
			MessageID: "error-" + task.ID,
			Role:      "assistant",
			Parts: []adk.Part{
				map[string]interface{}{
					"kind": "text",
					"text": "No response received from LLM",
				},
			},
		}
		task.Status.Message = errorMessage
		return task, nil
	}

	sdkMessage := sdk.Message{
		Role:    result.Choices[0].Message.Role,
		Content: result.Choices[0].Message.Content,
	}

	response, err := a.converter.ConvertFromSDK(sdkMessage)
	if err != nil {
		a.logger.Error("failed to convert SDK response to A2A format", zap.Error(err))
		task.Status.State = adk.TaskStateFailed
		errorMessage := &adk.Message{
			Kind:      "message",
			MessageID: "error-" + task.ID,
			Role:      "assistant",
			Parts: []adk.Part{
				map[string]interface{}{
					"kind": "text",
					"text": fmt.Sprintf("Response conversion failed: %v", err),
				},
			},
		}
		task.Status.Message = errorMessage
		return task, nil
	}

	response.MessageID = "response-" + task.ID

	task.History = append(task.History, *response)
	task.Status.State = adk.TaskStateCompleted
	task.Status.Message = response

	a.logger.Info("task completed with llm response",
		zap.String("task_id", task.ID),
		zap.String("context_id", task.ContextID))
	return task, nil
}

// processWithToolCalling processes the task with tool calling capability using iterative approach
func (a *DefaultOpenAICompatibleAgent) processWithToolCalling(ctx context.Context, task *adk.Task, messages []adk.Message) (*adk.Task, error) {
	tools := a.toolBox.GetTools()
	currentMessages := messages
	iteration := 0

	for iteration < a.maxChatCompletionIterations {
		iteration++
		a.logger.Debug("starting chat completion iteration",
			zap.Int("iteration", iteration),
			zap.Int("max_iterations", a.maxChatCompletionIterations))

		sdkMessages, err := a.converter.ConvertToSDK(currentMessages)
		if err != nil {
			a.logger.Error("failed to convert A2A messages to SDK format", zap.Error(err))
			return a.createErrorTask(task, fmt.Sprintf("Message conversion failed: %v", err)), nil
		}

		result, err := a.llmClient.CreateChatCompletion(ctx, sdkMessages, tools...)
		if err != nil {
			a.logger.Error("llm completion failed",
				zap.Error(err),
				zap.String("task_id", task.ID),
				zap.String("context_id", task.ContextID),
				zap.Int("iteration", iteration))
			return a.createErrorTask(task, fmt.Sprintf("LLM request failed: %v", err)), nil
		}

		if len(result.Choices) == 0 {
			a.logger.Error("no choices in llm response",
				zap.String("task_id", task.ID),
				zap.String("context_id", task.ContextID),
				zap.Int("iteration", iteration))
			return a.createErrorTask(task, "No response received from LLM"), nil
		}

		choice := result.Choices[0]

		if choice.Message.ToolCalls != nil && len(*choice.Message.ToolCalls) > 0 {
			a.logger.Info("processing tool calls",
				zap.Int("count", len(*choice.Message.ToolCalls)),
				zap.Int("iteration", iteration))

			assistantMessage := &adk.Message{
				Kind:      "message",
				MessageID: fmt.Sprintf("assistant-%s-%d", task.ID, iteration),
				Role:      "assistant",
				Parts: []adk.Part{
					map[string]interface{}{
						"kind": "data",
						"data": map[string]interface{}{
							"tool_calls": *choice.Message.ToolCalls,
							"content":    choice.Message.Content,
						},
					},
				},
			}
			task.History = append(task.History, *assistantMessage)
			currentMessages = append(currentMessages, *assistantMessage)

			toolResults, err := a.executeTools(ctx, task, *choice.Message.ToolCalls)
			if err != nil {
				a.logger.Error("tool execution failed", zap.Error(err))
				return a.createErrorTask(task, fmt.Sprintf("Tool execution failed: %v", err)), nil
			}

			currentMessages = append(currentMessages, toolResults...)
			continue
		}

		response := &adk.Message{
			Kind:      "message",
			MessageID: fmt.Sprintf("response-%s-%d", task.ID, iteration),
			Role:      "assistant",
			Parts: []adk.Part{
				map[string]interface{}{
					"kind": "text",
					"text": choice.Message.Content,
				},
			},
		}

		task.History = append(task.History, *response)
		task.Status.State = adk.TaskStateCompleted
		task.Status.Message = response

		a.logger.Info("task completed successfully",
			zap.String("task_id", task.ID),
			zap.String("context_id", task.ContextID),
			zap.Int("iterations", iteration))
		return task, nil
	}

	a.logger.Warn("max chat completion iterations reached",
		zap.String("task_id", task.ID),
		zap.Int("max_iterations", a.maxChatCompletionIterations))
	return a.createErrorTask(task, fmt.Sprintf("Maximum iterations (%d) reached without completion", a.maxChatCompletionIterations)), nil
}

// processWithoutLLM processes the task without LLM when no client is available
func (a *DefaultOpenAICompatibleAgent) processWithoutLLM(task *adk.Task, message *adk.Message) *adk.Task {
	response := &adk.Message{
		Kind:      "message",
		MessageID: fmt.Sprintf("msg-%d", len(task.History)+1),
		Role:      "assistant",
		Parts: []adk.Part{
			map[string]interface{}{
				"kind": "text",
				"text": "I'm an AI assistant, but I don't have access to an LLM client right now. Please configure an LLM client to enable my full capabilities.",
			},
		},
	}

	task.History = append(task.History, *response)
	task.Status.State = adk.TaskStateCompleted
	task.Status.Message = response

	a.logger.Info("task completed without llm",
		zap.String("task_id", task.ID),
		zap.String("context_id", task.ContextID))
	return task
}

// SetSystemPrompt sets the system prompt for the agent
func (a *DefaultOpenAICompatibleAgent) SetSystemPrompt(prompt string) {
	a.systemPrompt = prompt
}

// GetSystemPrompt returns the current system prompt
func (a *DefaultOpenAICompatibleAgent) GetSystemPrompt() string {
	return a.systemPrompt
}

// SetToolBox sets the toolbox for the agent
func (a *DefaultOpenAICompatibleAgent) SetToolBox(toolBox ToolBox) {
	a.toolBox = toolBox
}

// GetToolBox returns the current toolbox
func (a *DefaultOpenAICompatibleAgent) GetToolBox() ToolBox {
	return a.toolBox
}

// SetLLMClient sets the LLM client for the agent
func (a *DefaultOpenAICompatibleAgent) SetLLMClient(client LLMClient) {
	a.llmClient = client
}

// GetLLMClient returns the current LLM client
func (a *DefaultOpenAICompatibleAgent) GetLLMClient() LLMClient {
	return a.llmClient
}

// createErrorTask creates a task with error state and message
func (a *DefaultOpenAICompatibleAgent) createErrorTask(task *adk.Task, errorMsg string) *adk.Task {
	task.Status.State = adk.TaskStateFailed
	task.Status.Message = &adk.Message{
		Kind:      "message",
		MessageID: "error-" + task.ID,
		Role:      "assistant",
		Parts: []adk.Part{
			map[string]interface{}{
				"kind": "text",
				"text": errorMsg,
			},
		},
	}
	return task
}

// executeTools executes all tool calls and returns the tool result messages
func (a *DefaultOpenAICompatibleAgent) executeTools(ctx context.Context, task *adk.Task, toolCalls []sdk.ChatCompletionMessageToolCall) ([]adk.Message, error) {
	toolResults := make([]adk.Message, 0, len(toolCalls))

	for _, toolCall := range toolCalls {
		if toolCall.Type != "function" {
			continue
		}

		function := toolCall.Function
		if function.Name == "" {
			continue
		}

		var args map[string]interface{}
		if function.Arguments != "" {
			if err := json.Unmarshal([]byte(function.Arguments), &args); err != nil {
				a.logger.Error("failed to parse tool arguments",
					zap.String("tool", function.Name),
					zap.Error(err))
				continue
			}
		}

		result, err := a.toolBox.ExecuteTool(ctx, function.Name, args)
		if err != nil {
			result = fmt.Sprintf("Error executing tool: %v", err)
			a.logger.Error("tool execution failed",
				zap.String("tool", function.Name),
				zap.Error(err))
		} else {
			a.logger.Info("tool executed successfully",
				zap.String("tool", function.Name))
		}

		toolResultMessage := adk.Message{
			Kind:      "message",
			MessageID: fmt.Sprintf("tool-result-%s", toolCall.Id),
			Role:      "tool",
			Parts: []adk.Part{
				map[string]interface{}{
					"kind": "data",
					"data": map[string]interface{}{
						"tool_call_id": toolCall.Id,
						"tool_name":    function.Name,
						"result":       result,
					},
				},
			},
		}

		toolResults = append(toolResults, toolResultMessage)
		task.History = append(task.History, toolResultMessage)
	}

	return toolResults, nil
}
