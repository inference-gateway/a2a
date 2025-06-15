package server

import (
	"fmt"
	"sync"
	"time"

	uuid "github.com/google/uuid"
	adk "github.com/inference-gateway/a2a/adk"
	zap "go.uber.org/zap"
)

// TaskManager defines task lifecycle management
type TaskManager interface {
	// CreateTask creates a new task and stores it
	CreateTask(contextID string, state adk.TaskState, message *adk.Message) *adk.Task

	// UpdateTask updates an existing task
	UpdateTask(taskID string, state adk.TaskState, message *adk.Message) error

	// GetTask retrieves a task by ID
	GetTask(taskID string) (*adk.Task, bool)

	// CancelTask cancels a task
	CancelTask(taskID string) error

	// CleanupCompletedTasks removes old completed tasks from memory
	CleanupCompletedTasks()

	// PollTaskStatus periodically checks the status of a task until it is completed or failed
	PollTaskStatus(taskID string, interval time.Duration, timeout time.Duration) (*adk.Task, error)

	// GetConversationHistory retrieves conversation history for a context ID
	GetConversationHistory(contextID string) []adk.Message

	// UpdateConversationHistory updates conversation history for a context ID
	UpdateConversationHistory(contextID string, messages []adk.Message)
}

// DefaultTaskManager implements the TaskManager interface
type DefaultTaskManager struct {
	logger                 *zap.Logger
	tasks                  map[string]*adk.Task
	conversationHistory    map[string][]adk.Message // contextID -> conversation history
	maxConversationHistory int                      // maximum number of messages to keep in history
	tasksMu                sync.RWMutex
	conversationMu         sync.RWMutex
}

// NewDefaultTaskManager creates a new default task manager
func NewDefaultTaskManager(logger *zap.Logger, maxConversationHistory int) *DefaultTaskManager {
	if maxConversationHistory <= 0 {
		maxConversationHistory = 20 // default value
	}
	return &DefaultTaskManager{
		logger:                 logger,
		tasks:                  make(map[string]*adk.Task),
		conversationHistory:    make(map[string][]adk.Message),
		maxConversationHistory: maxConversationHistory,
	}
}

// CreateTask creates a new task and stores it
func (tm *DefaultTaskManager) CreateTask(contextID string, state adk.TaskState, message *adk.Message) *adk.Task {
	tm.tasksMu.Lock()
	defer tm.tasksMu.Unlock()

	timestamp := time.Now().UTC().Format(time.RFC3339Nano)

	history := tm.GetConversationHistory(contextID)

	if message != nil {
		history = append(history, *message)
		tm.UpdateConversationHistory(contextID, history)
		history = tm.GetConversationHistory(contextID)
	}

	task := &adk.Task{
		ID:   uuid.New().String(),
		Kind: "task",
		Status: adk.TaskStatus{
			State:     state,
			Message:   message,
			Timestamp: &timestamp,
		},
		ContextID: contextID,
		History:   history,
	}

	tm.tasks[task.ID] = task
	tm.logger.Debug("task created",
		zap.String("task_id", task.ID),
		zap.String("context_id", contextID),
		zap.String("state", string(state)),
		zap.Int("history_count", len(history)))

	return task
}

// UpdateTask updates an existing task
func (tm *DefaultTaskManager) UpdateTask(taskID string, state adk.TaskState, message *adk.Message) error {
	tm.tasksMu.Lock()
	defer tm.tasksMu.Unlock()

	task, exists := tm.tasks[taskID]
	if !exists {
		return NewTaskNotFoundError(taskID)
	}

	timestamp := time.Now().UTC().Format(time.RFC3339Nano)
	task.Status.State = state
	task.Status.Message = message
	task.Status.Timestamp = &timestamp

	if state == adk.TaskStateCompleted && message != nil && task.ContextID != "" {
		task.History = append(task.History, *message)
		tm.UpdateConversationHistory(task.ContextID, task.History)
	}

	tm.tasks[taskID] = task
	tm.logger.Debug("task updated",
		zap.String("task_id", taskID),
		zap.String("context_id", task.ContextID),
		zap.String("state", string(state)),
		zap.Int("history_count", len(task.History)))

	return nil
}

// GetTask retrieves a task by ID
func (tm *DefaultTaskManager) GetTask(taskID string) (*adk.Task, bool) {
	tm.tasksMu.RLock()
	defer tm.tasksMu.RUnlock()

	task, exists := tm.tasks[taskID]
	return task, exists
}

// CancelTask cancels a task
func (tm *DefaultTaskManager) CancelTask(taskID string) error {
	tm.tasksMu.Lock()
	defer tm.tasksMu.Unlock()

	task, exists := tm.tasks[taskID]
	if !exists {
		return NewTaskNotFoundError(taskID)
	}

	task.Status.State = adk.TaskStateCanceled
	tm.tasks[taskID] = task
	tm.logger.Info("task canceled", zap.String("task_id", taskID))

	return nil
}

// CleanupCompletedTasks removes old completed tasks from memory
func (tm *DefaultTaskManager) CleanupCompletedTasks() {
	tm.tasksMu.Lock()
	defer tm.tasksMu.Unlock()

	var toRemove []string

	for taskID, task := range tm.tasks {
		switch task.Status.State {
		case adk.TaskStateCompleted, adk.TaskStateFailed, adk.TaskStateCanceled:
			toRemove = append(toRemove, taskID)
		}
	}

	for _, taskID := range toRemove {
		delete(tm.tasks, taskID)
	}

	if len(toRemove) > 0 {
		tm.logger.Info("cleaned up completed tasks", zap.Int("count", len(toRemove)))
	}
}

// PollTaskStatus periodically checks the status of a task until it is completed or failed
func (tm *DefaultTaskManager) PollTaskStatus(taskID string, interval time.Duration, timeout time.Duration) (*adk.Task, error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	timeoutTimer := time.NewTimer(timeout)
	defer timeoutTimer.Stop()

	for {
		select {
		case <-ticker.C:
			task, exists := tm.GetTask(taskID)
			if !exists {
				return nil, NewTaskNotFoundError(taskID)
			}

			if task.Status.State == adk.TaskStateCompleted || task.Status.State == adk.TaskStateFailed {
				return task, nil
			}

		case <-timeoutTimer.C:
			return nil, fmt.Errorf("polling timed out for task %s", taskID)
		}
	}
}

// GetConversationHistory retrieves conversation history for a context ID
func (tm *DefaultTaskManager) GetConversationHistory(contextID string) []adk.Message {
	tm.conversationMu.RLock()
	defer tm.conversationMu.RUnlock()

	if history, exists := tm.conversationHistory[contextID]; exists {
		// Return a copy to avoid external modification
		result := make([]adk.Message, len(history))
		copy(result, history)
		return result
	}

	return []adk.Message{}
}

// UpdateConversationHistory updates conversation history for a context ID
func (tm *DefaultTaskManager) UpdateConversationHistory(contextID string, messages []adk.Message) {
	tm.conversationMu.Lock()
	defer tm.conversationMu.Unlock()

	// Store a copy to avoid external modification
	history := make([]adk.Message, len(messages))
	copy(history, messages)

	// Trim history to respect the maximum size limit
	trimmedHistory := tm.trimConversationHistory(history)
	tm.conversationHistory[contextID] = trimmedHistory

	tm.logger.Debug("conversation history updated",
		zap.String("context_id", contextID),
		zap.Int("message_count", len(trimmedHistory)),
		zap.Int("max_history", tm.maxConversationHistory))
}

// trimConversationHistory ensures conversation history doesn't exceed the maximum allowed size
// It keeps the most recent messages and removes the oldest ones
func (tm *DefaultTaskManager) trimConversationHistory(history []adk.Message) []adk.Message {
	if len(history) <= tm.maxConversationHistory {
		return history
	}

	// Keep the most recent messages, remove the oldest ones
	startIndex := len(history) - tm.maxConversationHistory
	trimmed := make([]adk.Message, tm.maxConversationHistory)
	copy(trimmed, history[startIndex:])

	tm.logger.Debug("conversation history trimmed",
		zap.Int("original_length", len(history)),
		zap.Int("trimmed_length", len(trimmed)),
		zap.Int("max_history", tm.maxConversationHistory))

	return trimmed
}

// TaskNotFoundError represents an error when a task is not found
type TaskNotFoundError struct {
	TaskID string
}

func (e *TaskNotFoundError) Error() string {
	return "task not found: " + e.TaskID
}

// NewTaskNotFoundError creates a new TaskNotFoundError
func NewTaskNotFoundError(taskID string) error {
	return &TaskNotFoundError{TaskID: taskID}
}
