package server

import (
	"context"
	"fmt"

	adk "github.com/inference-gateway/a2a/adk"
	zap "go.uber.org/zap"
)

// TaskHandler defines how to handle task processing
// This interface should be implemented by domain-specific task handlers
type TaskHandler interface {
	// HandleTask processes a task and returns the updated task
	// This is where the main business logic should be implemented
	HandleTask(ctx context.Context, task *adk.Task, message *adk.Message) (*adk.Task, error)
}

// DefaultTaskHandler implements the TaskHandler interface
// This handler throws an error to enforce using proper handlers like AgentTaskHandler
type DefaultTaskHandler struct {
	logger *zap.Logger
}

// NewDefaultTaskHandler creates a new default task handler
func NewDefaultTaskHandler(logger *zap.Logger) *DefaultTaskHandler {
	return &DefaultTaskHandler{
		logger: logger,
	}
}

// HandleTask processes a task and returns the updated task
// This is a simple fallback implementation that marks tasks as completed
func (th *DefaultTaskHandler) HandleTask(ctx context.Context, task *adk.Task, message *adk.Message) (*adk.Task, error) {
	th.logger.Error("default task handler should not be used directly",
		zap.String("task_id", task.ID),
		zap.String("context_id", task.ContextID),
		zap.String("message", "configure an agent or custom task handler"))

	return nil, fmt.Errorf("no task handler configured: use AgentTaskHandler with an OpenAI-compatible agent or implement a custom TaskHandler")
}
