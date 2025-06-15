package utils

import (
	"testing"

	adk "github.com/inference-gateway/a2a/adk"
	sdk "github.com/inference-gateway/sdk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	zap "go.uber.org/zap"
)

func TestOptimizedMessageConverter_ConvertToSDK(t *testing.T) {
	logger := zap.NewNop()
	converter := NewOptimizedMessageConverter(logger)

	tests := []struct {
		name           string
		input          []adk.Message
		expectedOutput []sdk.Message
		expectError    bool
	}{
		{
			name: "convert simple text message",
			input: []adk.Message{
				{
					Kind:      "message",
					MessageID: "test-msg-1",
					Role:      "user",
					Parts: []adk.Part{
						map[string]interface{}{
							"kind": "text",
							"text": "Hello, world!",
						},
					},
				},
			},
			expectedOutput: []sdk.Message{
				{
					Role:    sdk.User,
					Content: "Hello, world!",
				},
			},
			expectError: false,
		},
		{
			name: "convert assistant message",
			input: []adk.Message{
				{
					Kind:      "message",
					MessageID: "test-msg-2",
					Role:      "assistant",
					Parts: []adk.Part{
						map[string]interface{}{
							"kind": "text",
							"text": "Hi there!",
						},
					},
				},
			},
			expectedOutput: []sdk.Message{
				{
					Role:    sdk.Assistant,
					Content: "Hi there!",
				},
			},
			expectError: false,
		},
		{
			name: "convert system message",
			input: []adk.Message{
				{
					Kind:      "message",
					MessageID: "test-msg-3",
					Role:      "system",
					Parts: []adk.Part{
						map[string]interface{}{
							"kind": "text",
							"text": "You are a helpful assistant.",
						},
					},
				},
			},
			expectedOutput: []sdk.Message{
				{
					Role:    sdk.System,
					Content: "You are a helpful assistant.",
				},
			},
			expectError: false,
		},
		{
			name: "convert message with empty role defaults to user",
			input: []adk.Message{
				{
					Kind:      "message",
					MessageID: "test-msg-4",
					Role:      "",
					Parts: []adk.Part{
						map[string]interface{}{
							"kind": "text",
							"text": "Default role test",
						},
					},
				},
			},
			expectedOutput: []sdk.Message{
				{
					Role:    sdk.User,
					Content: "Default role test",
				},
			},
			expectError: false,
		},
		{
			name: "convert message with multiple text parts",
			input: []adk.Message{
				{
					Kind:      "message",
					MessageID: "test-msg-5",
					Role:      "user",
					Parts: []adk.Part{
						map[string]interface{}{
							"kind": "text",
							"text": "Part 1. ",
						},
						map[string]interface{}{
							"kind": "text",
							"text": "Part 2.",
						},
					},
				},
			},
			expectedOutput: []sdk.Message{
				{
					Role:    sdk.User,
					Content: "Part 1. Part 2.",
				},
			},
			expectError: false,
		},
		{
			name: "convert message with data part",
			input: []adk.Message{
				{
					Kind:      "message",
					MessageID: "test-msg-6",
					Role:      "tool",
					Parts: []adk.Part{
						map[string]interface{}{
							"kind": "data",
							"data": map[string]interface{}{
								"result": "Tool execution result",
							},
						},
					},
				},
			},
			expectedOutput: []sdk.Message{
				{
					Role:    sdk.Tool,
					Content: "Tool execution result",
				},
			},
			expectError: false,
		},
		{
			name: "convert strongly-typed message part",
			input: []adk.Message{
				{
					Kind:      "message",
					MessageID: "test-msg-7",
					Role:      "user",
					Parts: []adk.Part{
						adk.OptimizedMessagePart{
							Kind: adk.MessagePartKindText,
							Text: stringPtr("Strongly typed message"),
						},
					},
				},
			},
			expectedOutput: []sdk.Message{
				{
					Role:    sdk.User,
					Content: "Strongly typed message",
				},
			},
			expectError: false,
		},
		{
			name: "convert message with file part (no content extraction)",
			input: []adk.Message{
				{
					Kind:      "message",
					MessageID: "test-msg-8",
					Role:      "user",
					Parts: []adk.Part{
						map[string]interface{}{
							"kind": "text",
							"text": "Please analyze this file: ",
						},
						adk.OptimizedMessagePart{
							Kind: adk.MessagePartKindFile,
							File: &adk.FileData{
								Name:     stringPtr("test.txt"),
								MIMEType: stringPtr("text/plain"),
								Bytes:    stringPtr("base64encodedcontent"),
							},
						},
					},
				},
			},
			expectedOutput: []sdk.Message{
				{
					Role:    sdk.User,
					Content: "Please analyze this file: ",
				},
			},
			expectError: false,
		},
		{
			name: "convert multiple messages",
			input: []adk.Message{
				{
					Kind:      "message",
					MessageID: "test-msg-9",
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
					MessageID: "test-msg-10",
					Role:      "assistant",
					Parts: []adk.Part{
						map[string]interface{}{
							"kind": "text",
							"text": "Second message",
						},
					},
				},
			},
			expectedOutput: []sdk.Message{
				{
					Role:    sdk.User,
					Content: "First message",
				},
				{
					Role:    sdk.Assistant,
					Content: "Second message",
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := converter.ConvertToSDK(tt.input)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, len(tt.expectedOutput), len(result))

			for i, expected := range tt.expectedOutput {
				assert.Equal(t, expected.Role, result[i].Role)
				assert.Equal(t, expected.Content, result[i].Content)
			}
		})
	}
}

func TestOptimizedMessageConverter_ConvertFromSDK(t *testing.T) {
	logger := zap.NewNop()
	converter := NewOptimizedMessageConverter(logger)

	tests := []struct {
		name           string
		input          sdk.Message
		expectedOutput *adk.Message
		expectError    bool
	}{
		{
			name: "convert SDK user message",
			input: sdk.Message{
				Role:    sdk.User,
				Content: "Hello from SDK",
			},
			expectedOutput: &adk.Message{
				Kind: "message",
				Role: "user",
				Parts: []adk.Part{
					map[string]interface{}{
						"kind": "text",
						"text": "Hello from SDK",
					},
				},
			},
			expectError: false,
		},
		{
			name: "convert SDK assistant message",
			input: sdk.Message{
				Role:    sdk.Assistant,
				Content: "Response from assistant",
			},
			expectedOutput: &adk.Message{
				Kind: "message",
				Role: "assistant",
				Parts: []adk.Part{
					map[string]interface{}{
						"kind": "text",
						"text": "Response from assistant",
					},
				},
			},
			expectError: false,
		},
		{
			name: "convert SDK system message",
			input: sdk.Message{
				Role:    sdk.System,
				Content: "System instructions",
			},
			expectedOutput: &adk.Message{
				Kind: "message",
				Role: "system",
				Parts: []adk.Part{
					map[string]interface{}{
						"kind": "text",
						"text": "System instructions",
					},
				},
			},
			expectError: false,
		},
		{
			name: "convert SDK tool message",
			input: sdk.Message{
				Role:    sdk.Tool,
				Content: "Tool response",
			},
			expectedOutput: &adk.Message{
				Kind: "message",
				Role: "tool",
				Parts: []adk.Part{
					map[string]interface{}{
						"kind": "text",
						"text": "Tool response",
					},
				},
			},
			expectError: false,
		},
		{
			name: "convert SDK message with tool calls",
			input: sdk.Message{
				Role:    sdk.Assistant,
				Content: "I'll help you with that",
				ToolCalls: &[]sdk.ChatCompletionMessageToolCall{
					{
						Id:   "call_123",
						Type: "function",
						Function: sdk.ChatCompletionMessageToolCallFunction{
							Name:      "get_weather",
							Arguments: `{"location": "New York"}`,
						},
					},
				},
			},
			expectedOutput: &adk.Message{
				Kind: "message",
				Role: "assistant",
				Parts: []adk.Part{
					map[string]interface{}{
						"kind": "text",
						"text": "I'll help you with that",
					},
					map[string]interface{}{
						"kind": "data",
						"data": map[string]interface{}{
							"tool_calls": []sdk.ChatCompletionMessageToolCall{
								{
									Id:   "call_123",
									Type: "function",
									Function: sdk.ChatCompletionMessageToolCallFunction{
										Name:      "get_weather",
										Arguments: `{"location": "New York"}`,
									},
								},
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "convert SDK message with empty content",
			input: sdk.Message{
				Role:    sdk.Assistant,
				Content: "",
			},
			expectedOutput: &adk.Message{
				Kind: "message",
				Role: "assistant",
				Parts: []adk.Part{
					map[string]interface{}{
						"kind": "text",
						"text": "",
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := converter.ConvertFromSDK(tt.input)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedOutput.Kind, result.Kind)
			assert.Equal(t, tt.expectedOutput.Role, result.Role)
			assert.Equal(t, len(tt.expectedOutput.Parts), len(result.Parts))

			for i, expectedPart := range tt.expectedOutput.Parts {
				if partMap, ok := expectedPart.(map[string]interface{}); ok {
					resultPartMap, ok := result.Parts[i].(map[string]interface{})
					require.True(t, ok, "Expected result part to be map[string]interface{}")
					assert.Equal(t, partMap["kind"], resultPartMap["kind"])

					switch partMap["kind"] {
					case "text":
						assert.Equal(t, partMap["text"], resultPartMap["text"])
					case "data":
						expectedData := partMap["data"].(map[string]interface{})
						resultData := resultPartMap["data"].(map[string]interface{})
						assert.Equal(t, expectedData, resultData)
					}
				}
			}
		})
	}
}

func TestOptimizedMessageConverter_ValidateMessagePart(t *testing.T) {
	logger := zap.NewNop()
	converter := NewOptimizedMessageConverter(logger)

	tests := []struct {
		name        string
		input       adk.Part
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid strongly-typed text part",
			input: adk.OptimizedMessagePart{
				Kind: adk.MessagePartKindText,
				Text: stringPtr("Valid text"),
			},
			expectError: false,
		},
		{
			name: "valid strongly-typed file part",
			input: adk.OptimizedMessagePart{
				Kind: adk.MessagePartKindFile,
				File: &adk.FileData{
					Name:     stringPtr("test.txt"),
					MIMEType: stringPtr("text/plain"),
				},
			},
			expectError: false,
		},
		{
			name: "valid strongly-typed data part",
			input: adk.OptimizedMessagePart{
				Kind: adk.MessagePartKindData,
				Data: map[string]interface{}{
					"key": "value",
				},
			},
			expectError: false,
		},
		{
			name: "invalid strongly-typed text part (missing text)",
			input: adk.OptimizedMessagePart{
				Kind: adk.MessagePartKindText,
				Text: nil,
			},
			expectError: true,
			errorMsg:    "text part missing text field",
		},
		{
			name: "invalid strongly-typed file part (missing file)",
			input: adk.OptimizedMessagePart{
				Kind: adk.MessagePartKindFile,
				File: nil,
			},
			expectError: true,
			errorMsg:    "file part missing file field",
		},
		{
			name: "invalid strongly-typed data part (missing data)",
			input: adk.OptimizedMessagePart{
				Kind: adk.MessagePartKindData,
				Data: nil,
			},
			expectError: true,
			errorMsg:    "data part missing data field",
		},
		{
			name: "valid map-based text part",
			input: map[string]interface{}{
				"kind": "text",
				"text": "Valid text content",
			},
			expectError: false,
		},
		{
			name: "valid map-based data part",
			input: map[string]interface{}{
				"kind": "data",
				"data": map[string]interface{}{
					"result": "some result",
				},
			},
			expectError: false,
		},
		{
			name: "valid map-based file part",
			input: map[string]interface{}{
				"kind": "file",
				"file": map[string]interface{}{
					"name":     "test.txt",
					"mimeType": "text/plain",
				},
			},
			expectError: false,
		},
		{
			name: "invalid map-based part (missing kind)",
			input: map[string]interface{}{
				"text": "Missing kind field",
			},
			expectError: true,
			errorMsg:    "message part missing kind field",
		},
		{
			name: "invalid map-based part (non-string kind)",
			input: map[string]interface{}{
				"kind": 123,
				"text": "Invalid kind type",
			},
			expectError: true,
			errorMsg:    "message part kind must be string",
		},
		{
			name: "invalid map-based part (invalid kind value)",
			input: map[string]interface{}{
				"kind": "invalid_kind",
				"text": "Invalid kind value",
			},
			expectError: true,
			errorMsg:    "invalid message part kind: invalid_kind",
		},
		{
			name:        "unsupported part type",
			input:       "unsupported string part",
			expectError: true,
			errorMsg:    "unsupported message part type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := converter.ValidateMessagePart(tt.input)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestOptimizedMessageConverter_RoundTrip(t *testing.T) {
	logger := zap.NewNop()
	converter := NewOptimizedMessageConverter(logger)

	originalMessage := adk.Message{
		Kind:      "message",
		MessageID: "round-trip-test",
		Role:      "user",
		Parts: []adk.Part{
			map[string]interface{}{
				"kind": "text",
				"text": "Round trip test message",
			},
		},
	}

	sdkMessages, err := converter.ConvertToSDK([]adk.Message{originalMessage})
	require.NoError(t, err)
	require.Len(t, sdkMessages, 1)

	convertedMessage, err := converter.ConvertFromSDK(sdkMessages[0])
	require.NoError(t, err)

	assert.Equal(t, originalMessage.Kind, convertedMessage.Kind)
	assert.Equal(t, originalMessage.Role, convertedMessage.Role)
	assert.Len(t, convertedMessage.Parts, 1)

	originalPart := originalMessage.Parts[0].(map[string]interface{})
	convertedPart := convertedMessage.Parts[0].(map[string]interface{})
	assert.Equal(t, originalPart["kind"], convertedPart["kind"])
	assert.Equal(t, originalPart["text"], convertedPart["text"])
}

func TestOptimizedMessageConverter_PerformanceWithManyMessages(t *testing.T) {
	logger := zap.NewNop()
	converter := NewOptimizedMessageConverter(logger)

	messages := make([]adk.Message, 1000)
	for i := 0; i < 1000; i++ {
		messages[i] = adk.Message{
			Kind:      "message",
			MessageID: "perf-test-" + string(rune(i)),
			Role:      "user",
			Parts: []adk.Part{
				map[string]interface{}{
					"kind": "text",
					"text": "Performance test message number " + string(rune(i)),
				},
			},
		}
	}

	result, err := converter.ConvertToSDK(messages)
	require.NoError(t, err)
	assert.Len(t, result, 1000)

	for _, sdkMsg := range result {
		assert.Equal(t, sdk.User, sdkMsg.Role)
		assert.Contains(t, sdkMsg.Content, "Performance test message")
	}
}

func stringPtr(s string) *string {
	return &s
}
