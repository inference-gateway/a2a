package server_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/inference-gateway/a2a/adk"
	"github.com/inference-gateway/a2a/adk/server"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestDefaultTaskManager_CreateTask(t *testing.T) {
	tests := []struct {
		name      string
		contextID string
		state     adk.TaskState
		message   *adk.Message
	}{
		{
			name:      "create task with submitted state",
			contextID: "context-1",
			state:     adk.TaskStateSubmitted,
			message: &adk.Message{
				Kind:      "message",
				MessageID: "msg-1",
				Role:      "user",
				Parts: []adk.Part{
					map[string]interface{}{
						"kind": "text",
						"text": "Test message",
					},
				},
			},
		},
		{
			name:      "create task with working state",
			contextID: "context-2",
			state:     adk.TaskStateWorking,
			message: &adk.Message{
				Kind:      "message",
				MessageID: "msg-2",
				Role:      "assistant",
				Parts: []adk.Part{
					map[string]interface{}{
						"kind": "text",
						"text": "Processing...",
					},
				},
			},
		},
		{
			name:      "create task with completed state",
			contextID: "context-3",
			state:     adk.TaskStateCompleted,
			message: &adk.Message{
				Kind:      "message",
				MessageID: "msg-3",
				Role:      "assistant",
				Parts: []adk.Part{
					map[string]interface{}{
						"kind": "text",
						"text": "Task completed",
					},
				},
			},
		},
		{
			name:      "create task with failed state",
			contextID: "context-4",
			state:     adk.TaskStateFailed,
			message: &adk.Message{
				Kind:      "message",
				MessageID: "msg-4",
				Role:      "assistant",
				Parts: []adk.Part{
					map[string]interface{}{
						"kind": "text",
						"text": "Task failed",
					},
				},
			},
		},
		{
			name:      "create task with nil message",
			contextID: "context-5",
			state:     adk.TaskStateSubmitted,
			message:   nil,
		},
		{
			name:      "create task with empty context",
			contextID: "",
			state:     adk.TaskStateSubmitted,
			message: &adk.Message{
				Kind:      "message",
				MessageID: "msg-6",
				Role:      "user",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zap.NewNop()
			taskManager := server.NewDefaultTaskManager(logger, 20)

			task := taskManager.CreateTask(tt.contextID, tt.state, tt.message)

			assert.NotNil(t, task)
			assert.NotEmpty(t, task.ID)
			assert.Equal(t, tt.contextID, task.ContextID)
			assert.Equal(t, tt.state, task.Status.State)
			assert.Equal(t, tt.message, task.Status.Message)
			assert.NotNil(t, task.Status.Timestamp)

			if task.Status.Timestamp != nil {
				timestamp, err := time.Parse(time.RFC3339Nano, *task.Status.Timestamp)
				assert.NoError(t, err)
				assert.WithinDuration(t, time.Now(), timestamp, time.Second)
			}
		})
	}
}

func TestDefaultTaskManager_GetTask(t *testing.T) {
	logger := zap.NewNop()
	taskManager := server.NewDefaultTaskManager(logger, 20)

	message := &adk.Message{
		Kind:      "message",
		MessageID: "test-msg",
		Role:      "user",
		Parts: []adk.Part{
			map[string]interface{}{
				"kind": "text",
				"text": "Test message",
			},
		},
	}

	createdTask := taskManager.CreateTask("test-context", adk.TaskStateSubmitted, message)

	retrievedTask, exists := taskManager.GetTask(createdTask.ID)
	assert.True(t, exists)
	assert.NotNil(t, retrievedTask)
	assert.Equal(t, createdTask.ID, retrievedTask.ID)
	assert.Equal(t, createdTask.ContextID, retrievedTask.ContextID)
	assert.Equal(t, createdTask.Status.State, retrievedTask.Status.State)

	nonExistentTask, exists := taskManager.GetTask("non-existent-id")
	assert.False(t, exists)
	assert.Nil(t, nonExistentTask)

	emptyTask, exists := taskManager.GetTask("")
	assert.False(t, exists)
	assert.Nil(t, emptyTask)
}

func TestDefaultTaskManager_UpdateTask(t *testing.T) {
	tests := []struct {
		name        string
		newState    adk.TaskState
		newMessage  *adk.Message
		expectError bool
	}{
		{
			name:     "update to working state",
			newState: adk.TaskStateWorking,
			newMessage: &adk.Message{
				Kind:      "message",
				MessageID: "updated-msg-1",
				Role:      "assistant",
				Parts: []adk.Part{
					map[string]interface{}{
						"kind": "text",
						"text": "Now working on the task",
					},
				},
			},
			expectError: false,
		},
		{
			name:     "update to completed state",
			newState: adk.TaskStateCompleted,
			newMessage: &adk.Message{
				Kind:      "message",
				MessageID: "updated-msg-2",
				Role:      "assistant",
				Parts: []adk.Part{
					map[string]interface{}{
						"kind": "text",
						"text": "Task completed successfully",
					},
				},
			},
			expectError: false,
		},
		{
			name:     "update to failed state",
			newState: adk.TaskStateFailed,
			newMessage: &adk.Message{
				Kind:      "message",
				MessageID: "updated-msg-3",
				Role:      "assistant",
				Parts: []adk.Part{
					map[string]interface{}{
						"kind": "text",
						"text": "Task failed with error",
					},
				},
			},
			expectError: false,
		},
		{
			name:        "update with nil message",
			newState:    adk.TaskStateWorking,
			newMessage:  nil,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zap.NewNop()
			taskManager := server.NewDefaultTaskManager(logger, 20)

			originalMessage := &adk.Message{
				Kind:      "message",
				MessageID: "original-msg",
				Role:      "user",
				Parts: []adk.Part{
					map[string]interface{}{
						"kind": "text",
						"text": "Original message",
					},
				},
			}

			task := taskManager.CreateTask("test-context", adk.TaskStateSubmitted, originalMessage)
			originalTimestamp := task.Status.Timestamp

			err := taskManager.UpdateTask(task.ID, tt.newState, tt.newMessage)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				updatedTask, exists := taskManager.GetTask(task.ID)
				assert.True(t, exists)
				assert.Equal(t, tt.newState, updatedTask.Status.State)
				assert.Equal(t, tt.newMessage, updatedTask.Status.Message)

				if originalTimestamp != nil && updatedTask.Status.Timestamp != nil {
					originalTime, err := time.Parse(time.RFC3339Nano, *originalTimestamp)
					assert.NoError(t, err)
					updatedTime, err := time.Parse(time.RFC3339Nano, *updatedTask.Status.Timestamp)
					assert.NoError(t, err)
					assert.True(t, updatedTime.After(originalTime))
				}
			}
		})
	}
}

func TestDefaultTaskManager_UpdateNonExistentTask(t *testing.T) {
	logger := zap.NewNop()
	taskManager := server.NewDefaultTaskManager(logger, 20)

	message := &adk.Message{
		Kind:      "message",
		MessageID: "test-msg",
		Role:      "assistant",
		Parts: []adk.Part{
			map[string]interface{}{
				"kind": "text",
				"text": "Update message",
			},
		},
	}

	err := taskManager.UpdateTask("non-existent-id", adk.TaskStateCompleted, message)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task not found")
}

func TestDefaultTaskManager_CleanupCompletedTasks(t *testing.T) {
	logger := zap.NewNop()
	taskManager := server.NewDefaultTaskManager(logger, 20)

	submittedTask := taskManager.CreateTask("context-1", adk.TaskStateSubmitted, nil)
	workingTask := taskManager.CreateTask("context-2", adk.TaskStateWorking, nil)
	completedTask := taskManager.CreateTask("context-3", adk.TaskStateCompleted, nil)
	failedTask := taskManager.CreateTask("context-4", adk.TaskStateFailed, nil)

	_, exists := taskManager.GetTask(submittedTask.ID)
	assert.True(t, exists)
	_, exists = taskManager.GetTask(workingTask.ID)
	assert.True(t, exists)
	_, exists = taskManager.GetTask(completedTask.ID)
	assert.True(t, exists)
	_, exists = taskManager.GetTask(failedTask.ID)
	assert.True(t, exists)

	taskManager.CleanupCompletedTasks()

	_, exists = taskManager.GetTask(submittedTask.ID)
	assert.True(t, exists, "submitted task should remain")
	_, exists = taskManager.GetTask(workingTask.ID)
	assert.True(t, exists, "working task should remain")
	_, exists = taskManager.GetTask(completedTask.ID)
	assert.False(t, exists, "completed task should be cleaned up")
	_, exists = taskManager.GetTask(failedTask.ID)
	assert.False(t, exists, "failed task should be cleaned up")
}

func TestDefaultTaskManager_ConcurrentAccess(t *testing.T) {
	logger := zap.NewNop()
	taskManager := server.NewDefaultTaskManager(logger, 20)

	numGoroutines := 10
	tasksChan := make(chan *adk.Task, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			message := &adk.Message{
				Kind:      "message",
				MessageID: "concurrent-msg",
				Role:      "user",
			}
			task := taskManager.CreateTask("concurrent-context", adk.TaskStateSubmitted, message)
			tasksChan <- task
		}(i)
	}

	createdTasks := make([]*adk.Task, 0, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		task := <-tasksChan
		createdTasks = append(createdTasks, task)
	}

	taskIDs := make(map[string]bool)
	for _, task := range createdTasks {
		assert.NotEmpty(t, task.ID)
		assert.False(t, taskIDs[task.ID], "Task ID should be unique: %s", task.ID)
		taskIDs[task.ID] = true
	}

	assert.Len(t, taskIDs, numGoroutines, "All tasks should have unique IDs")
}

func TestNewDefaultTaskManager(t *testing.T) {
	logger := zap.NewNop()
	taskManager := server.NewDefaultTaskManager(logger, 20)

	assert.NotNil(t, taskManager)
}

func TestNewDefaultTaskManager_WithNilLogger(t *testing.T) {
	assert.NotPanics(t, func() {
		taskManager := server.NewDefaultTaskManager(nil, 20)
		assert.NotNil(t, taskManager)
	})
}

func TestDefaultTaskManager_ConversationContextPreservation(t *testing.T) {
	logger := zap.NewNop()
	taskManager := server.NewDefaultTaskManager(logger, 20)

	contextID := "test-conversation-context"

	firstMessage := &adk.Message{
		Kind:      "message",
		MessageID: "msg-1",
		Role:      "user",
		Parts: []adk.Part{
			map[string]interface{}{
				"kind": "text",
				"text": "Hello, what's the weather like?",
			},
		},
	}

	task1 := taskManager.CreateTask(contextID, adk.TaskStateSubmitted, firstMessage)
	assert.NotNil(t, task1)
	assert.Equal(t, contextID, task1.ContextID)
	assert.Len(t, task1.History, 1)
	assert.Equal(t, *firstMessage, task1.History[0])

	assistantResponse1 := &adk.Message{
		Kind:      "message",
		MessageID: "msg-response-1",
		Role:      "assistant",
		Parts: []adk.Part{
			map[string]interface{}{
				"kind": "text",
				"text": "It's sunny today with a temperature of 72°F.",
			},
		},
	}

	err := taskManager.UpdateTask(task1.ID, adk.TaskStateCompleted, assistantResponse1)
	assert.NoError(t, err)

	updatedTask1, exists := taskManager.GetTask(task1.ID)
	assert.True(t, exists)
	assert.Len(t, updatedTask1.History, 2)
	assert.Equal(t, *firstMessage, updatedTask1.History[0])
	assert.Equal(t, *assistantResponse1, updatedTask1.History[1])

	secondMessage := &adk.Message{
		Kind:      "message",
		MessageID: "msg-2",
		Role:      "user",
		Parts: []adk.Part{
			map[string]interface{}{
				"kind": "text",
				"text": "What about tomorrow?",
			},
		},
	}

	task2 := taskManager.CreateTask(contextID, adk.TaskStateSubmitted, secondMessage)
	assert.NotNil(t, task2)
	assert.Equal(t, contextID, task2.ContextID)
	assert.NotEqual(t, task1.ID, task2.ID)

	assert.Len(t, task2.History, 3)
	assert.Equal(t, *firstMessage, task2.History[0])
	assert.Equal(t, *assistantResponse1, task2.History[1])
	assert.Equal(t, *secondMessage, task2.History[2])

	assistantResponse2 := &adk.Message{
		Kind:      "message",
		MessageID: "msg-response-2",
		Role:      "assistant",
		Parts: []adk.Part{
			map[string]interface{}{
				"kind": "text",
				"text": "Tomorrow will be partly cloudy with a high of 68°F.",
			},
		},
	}

	err = taskManager.UpdateTask(task2.ID, adk.TaskStateCompleted, assistantResponse2)
	assert.NoError(t, err)

	updatedTask2, exists := taskManager.GetTask(task2.ID)
	assert.True(t, exists)
	assert.Len(t, updatedTask2.History, 4)
	assert.Equal(t, *firstMessage, updatedTask2.History[0])
	assert.Equal(t, *assistantResponse1, updatedTask2.History[1])
	assert.Equal(t, *secondMessage, updatedTask2.History[2])
	assert.Equal(t, *assistantResponse2, updatedTask2.History[3])

	thirdMessage := &adk.Message{
		Kind:      "message",
		MessageID: "msg-3",
		Role:      "user",
		Parts: []adk.Part{
			map[string]interface{}{
				"kind": "text",
				"text": "Should I bring an umbrella?",
			},
		},
	}

	task3 := taskManager.CreateTask(contextID, adk.TaskStateSubmitted, thirdMessage)
	assert.NotNil(t, task3)
	assert.Equal(t, contextID, task3.ContextID)

	assert.Len(t, task3.History, 5)
	assert.Equal(t, *firstMessage, task3.History[0])
	assert.Equal(t, *assistantResponse1, task3.History[1])
	assert.Equal(t, *secondMessage, task3.History[2])
	assert.Equal(t, *assistantResponse2, task3.History[3])
	assert.Equal(t, *thirdMessage, task3.History[4])
}

func TestDefaultTaskManager_ConversationHistoryIsolation(t *testing.T) {
	logger := zap.NewNop()
	taskManager := server.NewDefaultTaskManager(logger, 20)

	contextID1 := "context-1"
	contextID2 := "context-2"

	message1 := &adk.Message{
		Kind:      "message",
		MessageID: "msg-1",
		Role:      "user",
		Parts: []adk.Part{
			map[string]interface{}{
				"kind": "text",
				"text": "Message for context 1",
			},
		},
	}

	message2 := &adk.Message{
		Kind:      "message",
		MessageID: "msg-2",
		Role:      "user",
		Parts: []adk.Part{
			map[string]interface{}{
				"kind": "text",
				"text": "Message for context 2",
			},
		},
	}

	task1 := taskManager.CreateTask(contextID1, adk.TaskStateSubmitted, message1)
	task2 := taskManager.CreateTask(contextID2, adk.TaskStateSubmitted, message2)

	assert.Len(t, task1.History, 1)
	assert.Equal(t, *message1, task1.History[0])

	assert.Len(t, task2.History, 1)
	assert.Equal(t, *message2, task2.History[0])

	response1 := &adk.Message{
		Kind:      "message",
		MessageID: "response-1",
		Role:      "assistant",
		Parts: []adk.Part{
			map[string]interface{}{
				"kind": "text",
				"text": "Response to context 1",
			},
		},
	}

	err := taskManager.UpdateTask(task1.ID, adk.TaskStateCompleted, response1)
	assert.NoError(t, err)

	message3 := &adk.Message{
		Kind:      "message",
		MessageID: "msg-3",
		Role:      "user",
		Parts: []adk.Part{
			map[string]interface{}{
				"kind": "text",
				"text": "Follow-up for context 1",
			},
		},
	}

	task3 := taskManager.CreateTask(contextID1, adk.TaskStateSubmitted, message3)

	assert.Len(t, task3.History, 3)
	assert.Equal(t, *message1, task3.History[0])
	assert.Equal(t, *response1, task3.History[1])
	assert.Equal(t, *message3, task3.History[2])

	message4 := &adk.Message{
		Kind:      "message",
		MessageID: "msg-4",
		Role:      "user",
		Parts: []adk.Part{
			map[string]interface{}{
				"kind": "text",
				"text": "Follow-up for context 2",
			},
		},
	}

	task4 := taskManager.CreateTask(contextID2, adk.TaskStateSubmitted, message4)

	assert.Len(t, task4.History, 2)
	assert.Equal(t, *message2, task4.History[0])
	assert.Equal(t, *message4, task4.History[1])
}

func TestDefaultTaskManager_GetConversationHistory(t *testing.T) {
	logger := zap.NewNop()
	taskManager := server.NewDefaultTaskManager(logger, 20)

	contextID := "test-context"

	history := taskManager.GetConversationHistory(contextID)
	assert.Empty(t, history)

	message := &adk.Message{
		Kind:      "message",
		MessageID: "msg-1",
		Role:      "user",
		Parts: []adk.Part{
			map[string]interface{}{
				"kind": "text",
				"text": "Test message",
			},
		},
	}

	task := taskManager.CreateTask(contextID, adk.TaskStateSubmitted, message)

	response := &adk.Message{
		Kind:      "message",
		MessageID: "response-1",
		Role:      "assistant",
		Parts: []adk.Part{
			map[string]interface{}{
				"kind": "text",
				"text": "Test response",
			},
		},
	}

	err := taskManager.UpdateTask(task.ID, adk.TaskStateCompleted, response)
	assert.NoError(t, err)

	history = taskManager.GetConversationHistory(contextID)
	assert.Len(t, history, 2)
	assert.Equal(t, *message, history[0])
	assert.Equal(t, *response, history[1])

	history[0].Parts = []adk.Part{
		map[string]interface{}{
			"kind": "text",
			"text": "Modified message",
		},
	}

	freshHistory := taskManager.GetConversationHistory(contextID)
	assert.Len(t, freshHistory, 2)
	assert.Equal(t, "Test message", freshHistory[0].Parts[0].(map[string]interface{})["text"])
}

func TestDefaultTaskManager_UpdateConversationHistory(t *testing.T) {
	logger := zap.NewNop()
	taskManager := server.NewDefaultTaskManager(logger, 20)

	contextID := "test-context"

	messages := []adk.Message{
		{
			Kind:      "message",
			MessageID: "msg-1",
			Role:      "user",
			Parts: []adk.Part{
				map[string]interface{}{
					"kind": "text",
					"text": "First message",
				},
			},
		},
		{
			Kind:      "message",
			MessageID: "msg-2",
			Role:      "assistant",
			Parts: []adk.Part{
				map[string]interface{}{
					"kind": "text",
					"text": "First response",
				},
			},
		},
	}

	taskManager.UpdateConversationHistory(contextID, messages)

	history := taskManager.GetConversationHistory(contextID)
	assert.Len(t, history, 2)
	assert.Equal(t, messages[0], history[0])
	assert.Equal(t, messages[1], history[1])

	messages[0].Parts = []adk.Part{
		map[string]interface{}{
			"kind": "text",
			"text": "Modified message",
		},
	}

	freshHistory := taskManager.GetConversationHistory(contextID)
	assert.Equal(t, "First message", freshHistory[0].Parts[0].(map[string]interface{})["text"])
}

func TestDefaultTaskManager_ConversationHistoryLimit(t *testing.T) {
	logger := zap.NewNop()
	maxHistory := 3
	taskManager := server.NewDefaultTaskManager(logger, maxHistory)

	contextID := "test-context"

	messages := []adk.Message{
		{Kind: "message", MessageID: "msg-1", Role: "user", Parts: []adk.Part{map[string]interface{}{"kind": "text", "text": "Message 1"}}},
		{Kind: "message", MessageID: "msg-2", Role: "assistant", Parts: []adk.Part{map[string]interface{}{"kind": "text", "text": "Response 1"}}},
		{Kind: "message", MessageID: "msg-3", Role: "user", Parts: []adk.Part{map[string]interface{}{"kind": "text", "text": "Message 2"}}},
		{Kind: "message", MessageID: "msg-4", Role: "assistant", Parts: []adk.Part{map[string]interface{}{"kind": "text", "text": "Response 2"}}},
		{Kind: "message", MessageID: "msg-5", Role: "user", Parts: []adk.Part{map[string]interface{}{"kind": "text", "text": "Message 3"}}},
	}

	taskManager.UpdateConversationHistory(contextID, messages)

	history := taskManager.GetConversationHistory(contextID)
	assert.Len(t, history, maxHistory)

	assert.Equal(t, "msg-3", history[0].MessageID)
	assert.Equal(t, "msg-4", history[1].MessageID)
	assert.Equal(t, "msg-5", history[2].MessageID)
}

func TestDefaultTaskManager_ConversationHistoryLimitViaCreateTask(t *testing.T) {
	logger := zap.NewNop()
	maxHistory := 2
	taskManager := server.NewDefaultTaskManager(logger, maxHistory)

	contextID := "test-context"

	message1 := &adk.Message{
		Kind:      "message",
		MessageID: "msg-1",
		Role:      "user",
		Parts: []adk.Part{
			map[string]interface{}{
				"kind": "text",
				"text": "First message",
			},
		},
	}
	task1 := taskManager.CreateTask(contextID, adk.TaskStateSubmitted, message1)

	response1 := &adk.Message{
		Kind:      "message",
		MessageID: "response-1",
		Role:      "assistant",
		Parts: []adk.Part{
			map[string]interface{}{
				"kind": "text",
				"text": "First response",
			},
		},
	}
	err := taskManager.UpdateTask(task1.ID, adk.TaskStateCompleted, response1)
	assert.NoError(t, err)

	message2 := &adk.Message{
		Kind:      "message",
		MessageID: "msg-2",
		Role:      "user",
		Parts: []adk.Part{
			map[string]interface{}{
				"kind": "text",
				"text": "Second message",
			},
		},
	}
	_ = taskManager.CreateTask(contextID, adk.TaskStateSubmitted, message2)

	message3 := &adk.Message{
		Kind:      "message",
		MessageID: "msg-3",
		Role:      "user",
		Parts: []adk.Part{
			map[string]interface{}{
				"kind": "text",
				"text": "Third message",
			},
		},
	}
	task3 := taskManager.CreateTask(contextID, adk.TaskStateSubmitted, message3)

	assert.LessOrEqual(t, len(task3.History), maxHistory)

	assert.Equal(t, "msg-3", task3.History[len(task3.History)-1].MessageID)
}

func TestDefaultTaskManager_ConversationHistoryLimitZeroDefault(t *testing.T) {
	logger := zap.NewNop()
	taskManager := server.NewDefaultTaskManager(logger, 0)

	contextID := "test-context"
	messages := make([]adk.Message, 25)
	for i := 0; i < 25; i++ {
		messages[i] = adk.Message{
			Kind:      "message",
			MessageID: fmt.Sprintf("msg-%d", i),
			Role:      "user",
			Parts: []adk.Part{
				map[string]interface{}{
					"kind": "text",
					"text": fmt.Sprintf("Message %d", i),
				},
			},
		}
	}

	taskManager.UpdateConversationHistory(contextID, messages)
	history := taskManager.GetConversationHistory(contextID)

	assert.Len(t, history, 20)
	assert.Equal(t, "msg-5", history[0].MessageID)
	assert.Equal(t, "msg-24", history[19].MessageID)
}
