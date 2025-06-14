package server_test

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/inference-gateway/a2a/adk"
	"github.com/inference-gateway/a2a/adk/server"
	"github.com/inference-gateway/a2a/adk/server/config"
	"github.com/inference-gateway/a2a/adk/server/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestA2AServer_TaskManager_CreateTask(t *testing.T) {
	tests := []struct {
		name      string
		contextID string
		state     adk.TaskState
		message   *adk.Message
	}{
		{
			name:      "create task with submitted state",
			contextID: "test-context-1",
			state:     adk.TaskStateSubmitted,
			message: &adk.Message{
				Kind:      "message",
				MessageID: "test-message-1",
				Role:      "user",
				Parts: []adk.Part{
					map[string]interface{}{
						"kind": "text",
						"text": "Hello world",
					},
				},
			},
		},
		{
			name:      "create task with working state",
			contextID: "test-context-2",
			state:     adk.TaskStateWorking,
			message: &adk.Message{
				Kind:      "message",
				MessageID: "test-message-2",
				Role:      "assistant",
				Parts: []adk.Part{
					map[string]interface{}{
						"kind": "text",
						"text": "Processing your request",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zap.NewNop()
			taskManager := server.NewDefaultTaskManager(logger)

			task := taskManager.CreateTask(tt.contextID, tt.state, tt.message)

			assert.NotNil(t, task)
			assert.NotEmpty(t, task.ID)
			assert.Equal(t, tt.contextID, task.ContextID)
			assert.Equal(t, tt.state, task.Status.State)
			assert.Equal(t, tt.message, task.Status.Message)
			assert.NotNil(t, task.Status.Timestamp)
		})
	}
}

func TestA2AServer_TaskManager_UpdateTask(t *testing.T) {
	tests := []struct {
		name        string
		newState    adk.TaskState
		newMessage  *adk.Message
		expectError bool
	}{
		{
			name:     "update to completed state",
			newState: adk.TaskStateCompleted,
			newMessage: &adk.Message{
				Kind:      "message",
				MessageID: "test-message-updated",
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
				MessageID: "test-message-error",
				Role:      "assistant",
				Parts: []adk.Part{
					map[string]interface{}{
						"kind": "text",
						"text": "Task failed",
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zap.NewNop()
			taskManager := server.NewDefaultTaskManager(logger)

			task := taskManager.CreateTask("test-context", adk.TaskStateSubmitted, &adk.Message{
				Kind:      "message",
				MessageID: "initial-message",
				Role:      "user",
			})

			err := taskManager.UpdateTask(task.ID, tt.newState, tt.newMessage)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				updatedTask, exists := taskManager.GetTask(task.ID)
				assert.True(t, exists)
				assert.Equal(t, tt.newState, updatedTask.Status.State)
				assert.Equal(t, tt.newMessage, updatedTask.Status.Message)
			}
		})
	}
}

func TestA2AServer_TaskManager_GetTask(t *testing.T) {
	logger := zap.NewNop()
	taskManager := server.NewDefaultTaskManager(logger)

	message := &adk.Message{
		Kind:      "message",
		MessageID: "test-message",
		Role:      "user",
	}
	task := taskManager.CreateTask("test-context", adk.TaskStateSubmitted, message)

	retrievedTask, exists := taskManager.GetTask(task.ID)
	assert.True(t, exists)
	assert.Equal(t, task.ID, retrievedTask.ID)
	assert.Equal(t, task.ContextID, retrievedTask.ContextID)

	nonExistentTask, exists := taskManager.GetTask("non-existent-id")
	assert.False(t, exists)
	assert.Nil(t, nonExistentTask)
}

func TestA2AServer_ResponseSender_SendSuccess(t *testing.T) {
	logger := zap.NewNop()
	responseSender := server.NewDefaultResponseSender(logger)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	result := map[string]interface{}{
		"status": "success",
		"data":   "test data",
	}

	assert.NotPanics(t, func() {
		responseSender.SendSuccess(ctx, "test-id", result)
	})
}

func TestA2AServer_ResponseSender_SendError(t *testing.T) {
	logger := zap.NewNop()
	responseSender := server.NewDefaultResponseSender(logger)

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)

	assert.NotPanics(t, func() {
		responseSender.SendError(ctx, "test-id", 500, "test error message")
	})
}

func TestA2AServer_MessageHandler_Integration(t *testing.T) {
	logger := zap.NewNop()
	taskManager := server.NewDefaultTaskManager(logger)

	messageHandler := server.NewDefaultMessageHandler(logger, taskManager)

	contextID := "test-context"
	params := adk.MessageSendParams{
		Message: adk.Message{
			ContextID: &contextID,
			Kind:      "message",
			MessageID: "test-message",
			Role:      "user",
			Parts: []adk.Part{
				map[string]interface{}{
					"kind": "text",
					"text": "Hello, world!",
				},
			},
		},
	}

	ctx := context.Background()
	task, err := messageHandler.HandleMessageSend(ctx, params)

	assert.NoError(t, err)
	assert.NotNil(t, task)
	assert.Equal(t, contextID, task.ContextID)
	assert.Equal(t, adk.TaskStateSubmitted, task.Status.State)
}

func TestA2AServer_TaskProcessing_Background(t *testing.T) {
	cfg := config.Config{
		QueueConfig: &config.QueueConfig{
			MaxSize:         10,
			CleanupInterval: 50 * time.Millisecond,
		},
		CapabilitiesConfig: &config.CapabilitiesConfig{
			Streaming:              true,
			PushNotifications:      false,
			StateTransitionHistory: true,
		},
		AuthConfig: &config.AuthConfig{
			Enable: false,
		},
	}
	logger := zap.NewNop()

	a2aServer := server.NewA2AServer(&cfg, logger, nil)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	go a2aServer.StartTaskProcessor(ctx)

	time.Sleep(100 * time.Millisecond)

	assert.True(t, true)
}

func TestDefaultA2AServer_SetDependencies(t *testing.T) {
	customConfig := &config.Config{
		AgentName:        "custom-test-agent",
		AgentDescription: "A custom test agent for dependency injection",
		AgentURL:         "http://custom-agent:9090",
		AgentVersion:     "2.5.0",
		Port:             "9090",
		Debug:            true,
	}

	a2aServer := server.NewDefaultA2AServer(customConfig)

	mockTaskHandler := &mocks.FakeTaskHandler{}
	a2aServer.SetTaskHandler(mockTaskHandler)

	mockProcessor := &mocks.FakeTaskResultProcessor{}
	a2aServer.SetTaskResultProcessor(mockProcessor)

	agentCard := a2aServer.GetAgentCard()
	assert.Equal(t, "custom-test-agent", agentCard.Name)
	assert.Equal(t, "A custom test agent for dependency injection", agentCard.Description)
	assert.Equal(t, "http://custom-agent:9090", agentCard.URL)
	assert.Equal(t, "2.5.0", agentCard.Version)
}

func TestA2AServerBuilder_UsesProvidedConfiguration(t *testing.T) {
	cfg := config.Config{
		AgentName:        "test-custom-agent",
		AgentDescription: "A test agent with custom configuration",
		AgentURL:         "http://test-agent:9999",
		AgentVersion:     "2.0.0",
		Port:             "9999",
		Debug:            true,
	}

	logger := zap.NewNop()

	serverInstance := server.NewA2AServerBuilder(cfg, logger).Build()

	assert.NotNil(t, serverInstance)

	agentCard := serverInstance.GetAgentCard()
	assert.Equal(t, "test-custom-agent", agentCard.Name)
	assert.Equal(t, "A test agent with custom configuration", agentCard.Description)
	assert.Equal(t, "http://test-agent:9999", agentCard.URL)
	assert.Equal(t, "2.0.0", agentCard.Version)

	assert.NotNil(t, agentCard.Capabilities.Streaming)
	assert.NotNil(t, agentCard.Capabilities.PushNotifications)
	assert.NotNil(t, agentCard.Capabilities.StateTransitionHistory)
	assert.True(t, *agentCard.Capabilities.Streaming)
	assert.True(t, *agentCard.Capabilities.PushNotifications)
	assert.False(t, *agentCard.Capabilities.StateTransitionHistory)
}

func TestA2AServerBuilder_UsesProvidedCapabilitiesConfiguration(t *testing.T) {
	cfg := config.Config{
		AgentName:        "test-agent",
		AgentDescription: "A test agent",
		AgentURL:         "http://test-agent:8080",
		AgentVersion:     "1.0.0",
		Port:             "8080",
		CapabilitiesConfig: &config.CapabilitiesConfig{
			Streaming:              false,
			PushNotifications:      false,
			StateTransitionHistory: true,
		},
	}

	logger := zap.NewNop()

	serverInstance := server.NewA2AServerBuilder(cfg, logger).Build()

	assert.NotNil(t, serverInstance)

	agentCard := serverInstance.GetAgentCard()
	assert.Equal(t, "test-agent", agentCard.Name)

	assert.NotNil(t, agentCard.Capabilities.Streaming)
	assert.NotNil(t, agentCard.Capabilities.PushNotifications)
	assert.NotNil(t, agentCard.Capabilities.StateTransitionHistory)
	assert.False(t, *agentCard.Capabilities.Streaming)
	assert.False(t, *agentCard.Capabilities.PushNotifications)
	assert.True(t, *agentCard.Capabilities.StateTransitionHistory)
}

func TestA2AServerBuilder_HandlesNilConfigurationSafely(t *testing.T) {
	cfg := config.Config{
		AgentName:          "test-agent",
		AgentDescription:   "A test agent",
		AgentURL:           "http://test-agent:8080",
		AgentVersion:       "1.0.0",
		Port:               "8080",
		CapabilitiesConfig: nil,
		QueueConfig:        nil,
		ServerConfig:       nil,
	}

	logger := zap.NewNop()

	serverInstance := server.NewA2AServerBuilder(cfg, logger).Build()

	assert.NotNil(t, serverInstance)

	agentCard := serverInstance.GetAgentCard()
	assert.Equal(t, "test-agent", agentCard.Name)
	assert.Equal(t, "A test agent", agentCard.Description)
	assert.Equal(t, "http://test-agent:8080", agentCard.URL)
	assert.Equal(t, "1.0.0", agentCard.Version)

	assert.NotNil(t, agentCard.Capabilities.Streaming)
	assert.NotNil(t, agentCard.Capabilities.PushNotifications)
	assert.NotNil(t, agentCard.Capabilities.StateTransitionHistory)
	assert.True(t, *agentCard.Capabilities.Streaming)
	assert.True(t, *agentCard.Capabilities.PushNotifications)
	assert.False(t, *agentCard.Capabilities.StateTransitionHistory)
}

func TestA2AServer_TaskProcessing_MessageContent(t *testing.T) {
	logger := zap.NewNop()

	mockTaskHandler := &mocks.FakeTaskHandler{}
	mockTaskHandler.HandleTaskReturns(&adk.Task{
		ID:        "test-task",
		ContextID: "test-context",
		Status: adk.TaskStatus{
			State: adk.TaskStateCompleted,
			Message: &adk.Message{
				Kind:      "message",
				MessageID: "response-msg",
				Role:      "assistant",
				Parts: []adk.Part{
					map[string]interface{}{
						"kind": "text",
						"text": "Hello! I received your message.",
					},
				},
			},
		},
	}, nil)

	cfg := &config.Config{
		AgentName:        "test-agent",
		AgentDescription: "A test agent",
		AgentURL:         "http://test-agent:8080",
		AgentVersion:     "1.0.0",
		Port:             "8080",
		Debug:            false,
		QueueConfig: &config.QueueConfig{
			MaxSize:         10,
			CleanupInterval: 1 * time.Second,
		},
	}

	serverInstance := server.NewA2AServer(cfg, logger, nil)
	serverInstance.SetTaskHandler(mockTaskHandler)

	originalMessage := &adk.Message{
		Kind:      "message",
		MessageID: "original-msg",
		Role:      "user",
		Parts: []adk.Part{
			map[string]interface{}{
				"kind": "text",
				"text": "What is the weather like today?",
			},
		},
	}

	task := &adk.Task{
		ID:        "test-task",
		ContextID: "test-context",
		Status: adk.TaskStatus{
			State:   adk.TaskStateSubmitted,
			Message: originalMessage,
		},
	}

	ctx := context.Background()
	result, err := serverInstance.ProcessTask(ctx, task, originalMessage)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, adk.TaskStateCompleted, result.Status.State)
	assert.Equal(t, 1, mockTaskHandler.HandleTaskCallCount())

	_, actualTask, actualMessage := mockTaskHandler.HandleTaskArgsForCall(0)
	assert.NotNil(t, actualTask)
	assert.NotNil(t, actualMessage)

	assert.NotEmpty(t, actualMessage.Parts)
	assert.Len(t, actualMessage.Parts, 1)

	part := actualMessage.Parts[0]
	partMap, ok := part.(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "text", partMap["kind"])
	assert.Equal(t, "What is the weather like today?", partMap["text"])
}

func TestA2AServer_ProcessQueuedTask_MessageContent(t *testing.T) {
	logger := zap.NewNop()

	mockTaskHandler := &mocks.FakeTaskHandler{}
	mockTaskHandler.HandleTaskReturns(&adk.Task{
		ID:        "test-task",
		ContextID: "test-context",
		Status: adk.TaskStatus{
			State: adk.TaskStateCompleted,
			Message: &adk.Message{
				Kind:      "message",
				MessageID: "response-msg",
				Role:      "assistant",
				Parts: []adk.Part{
					map[string]interface{}{
						"kind": "text",
						"text": "I received your weather question and here's the answer...",
					},
				},
			},
		},
	}, nil)

	cfg := &config.Config{
		AgentName:        "weather-agent",
		AgentDescription: "A weather agent",
		AgentURL:         "http://weather-agent:8080",
		AgentVersion:     "1.0.0",
		Port:             "8080",
		Debug:            false,
		QueueConfig: &config.QueueConfig{
			MaxSize:         10,
			CleanupInterval: 1 * time.Second,
		},
	}

	serverInstance := server.NewA2AServer(cfg, logger, nil)
	serverInstance.SetTaskHandler(mockTaskHandler)

	originalUserMessage := &adk.Message{
		Kind:      "message",
		MessageID: "user-msg-123",
		Role:      "user",
		Parts: []adk.Part{
			map[string]interface{}{
				"kind": "text",
				"text": "What is the weather like today in San Francisco?",
			},
		},
	}

	task := &adk.Task{
		ID:        "task-456",
		ContextID: "context-789",
		Status: adk.TaskStatus{
			State:   adk.TaskStateSubmitted,
			Message: originalUserMessage,
		},
		History: []adk.Message{},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go serverInstance.StartTaskProcessor(ctx)

	time.Sleep(10 * time.Millisecond)

	result, err := serverInstance.ProcessTask(ctx, task, originalUserMessage)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, adk.TaskStateCompleted, result.Status.State)

	assert.Equal(t, 1, mockTaskHandler.HandleTaskCallCount())

	_, actualTask, actualMessage := mockTaskHandler.HandleTaskArgsForCall(0)

	assert.NotNil(t, actualTask)
	assert.NotNil(t, actualMessage)

	assert.NotEmpty(t, actualMessage.Parts, "Message parts should not be empty - this was the reported bug")
	assert.Len(t, actualMessage.Parts, 1, "Should have exactly one message part")

	part := actualMessage.Parts[0]
	partMap, ok := part.(map[string]interface{})
	assert.True(t, ok, "Message part should be a map")
	assert.Equal(t, "text", partMap["kind"], "Message part should be of kind 'text'")
	assert.Equal(t, "What is the weather like today in San Francisco?", partMap["text"],
		"Message content should be preserved exactly as sent by the client")

	assert.Equal(t, "user", actualMessage.Role, "Message role should be 'user'")
}
