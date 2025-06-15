package server

import (
	"context"

	adk "github.com/inference-gateway/a2a/adk"
	zap "go.uber.org/zap"
)

// AgentTaskHandler is a TaskHandler that delegates to an OpenAICompatibleAgent
type AgentTaskHandler struct {
	logger *zap.Logger
	agent  OpenAICompatibleAgent
}

// NewAgentTaskHandler creates a new task handler that uses an OpenAI-compatible agent
func NewAgentTaskHandler(logger *zap.Logger, agent OpenAICompatibleAgent) *AgentTaskHandler {
	return &AgentTaskHandler{
		logger: logger,
		agent:  agent,
	}
}

// HandleTask processes a task by delegating to the OpenAI-compatible agent
func (h *AgentTaskHandler) HandleTask(ctx context.Context, task *adk.Task, message *adk.Message) (*adk.Task, error) {
	h.logger.Info("processing task with openai-compatible agent",
		zap.String("task_id", task.ID),
		zap.String("context_id", task.ContextID))

	if h.agent == nil {
		h.logger.Error("agent not configured")
		return h.handleError(task, "Agent not configured"), nil
	}

	return h.agent.ProcessTask(ctx, task, message)
}

// handleError creates an error response task
func (h *AgentTaskHandler) handleError(task *adk.Task, errorMsg string) *adk.Task {
	task.Status.State = adk.TaskStateFailed

	errorMessage := &adk.Message{
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

	if task.History == nil {
		task.History = []adk.Message{}
	}
	task.History = append(task.History, *errorMessage)
	task.Status.Message = errorMessage

	return task
}
