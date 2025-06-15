package utils

import (
	"testing"

	adk "github.com/inference-gateway/a2a/adk"
	sdk "github.com/inference-gateway/sdk"
	zap "go.uber.org/zap"
)

func BenchmarkMessageConverter_ConvertToSDK(b *testing.B) {
	logger := zap.NewNop()
	converter := NewOptimizedMessageConverter(logger)

	messages := []adk.Message{
		{
			Kind:      "message",
			MessageID: "bench-msg-1",
			Role:      "user",
			Parts: []adk.Part{
				map[string]interface{}{
					"kind": "text",
					"text": "This is a benchmark test message with some content to convert.",
				},
			},
		},
		{
			Kind:      "message",
			MessageID: "bench-msg-2",
			Role:      "assistant",
			Parts: []adk.Part{
				map[string]interface{}{
					"kind": "text",
					"text": "This is a response message from the assistant.",
				},
			},
		},
		{
			Kind:      "message",
			MessageID: "bench-msg-3",
			Role:      "system",
			Parts: []adk.Part{
				map[string]interface{}{
					"kind": "text",
					"text": "System message with instructions for the assistant.",
				},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := converter.ConvertToSDK(messages)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMessageConverter_ConvertFromSDK(b *testing.B) {
	logger := zap.NewNop()
	converter := NewOptimizedMessageConverter(logger)

	// Create test SDK message
	sdkMessage := sdk.Message{
		Role:    sdk.Assistant,
		Content: "This is a benchmark test response from the SDK with some content to convert back to A2A format.",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := converter.ConvertFromSDK(sdkMessage)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMessageConverter_ConvertToSDK_StronglyTyped(b *testing.B) {
	logger := zap.NewNop()
	converter := NewOptimizedMessageConverter(logger)

	messages := []adk.Message{
		{
			Kind:      "message",
			MessageID: "bench-typed-msg-1",
			Role:      "user",
			Parts: []adk.Part{
				adk.OptimizedMessagePart{
					Kind: adk.MessagePartKindText,
					Text: stringPtr("This is a strongly-typed benchmark test message."),
				},
			},
		},
		{
			Kind:      "message",
			MessageID: "bench-typed-msg-2",
			Role:      "assistant",
			Parts: []adk.Part{
				adk.OptimizedMessagePart{
					Kind: adk.MessagePartKindText,
					Text: stringPtr("This is a strongly-typed response message."),
				},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := converter.ConvertToSDK(messages)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMessageConverter_ConvertToSDK_LargeMessages(b *testing.B) {
	logger := zap.NewNop()
	converter := NewOptimizedMessageConverter(logger)

	largeContent := ""
	for i := 0; i < 1000; i++ {
		largeContent += "This is a large message content for benchmarking purposes. "
	}

	messages := []adk.Message{
		{
			Kind:      "message",
			MessageID: "bench-large-msg",
			Role:      "user",
			Parts: []adk.Part{
				map[string]interface{}{
					"kind": "text",
					"text": largeContent,
				},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := converter.ConvertToSDK(messages)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMessageConverter_ConvertToSDK_ManyMessages(b *testing.B) {
	logger := zap.NewNop()
	converter := NewOptimizedMessageConverter(logger)

	messages := make([]adk.Message, 100)
	for i := 0; i < 100; i++ {
		messages[i] = adk.Message{
			Kind:      "message",
			MessageID: "bench-many-msg-" + string(rune(i)),
			Role:      "user",
			Parts: []adk.Part{
				map[string]interface{}{
					"kind": "text",
					"text": "Message number " + string(rune(i)),
				},
			},
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := converter.ConvertToSDK(messages)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMessageConverter_ValidateMessagePart(b *testing.B) {
	logger := zap.NewNop()
	converter := NewOptimizedMessageConverter(logger)

	part := map[string]interface{}{
		"kind": "text",
		"text": "This is a test message part for validation benchmarking.",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := converter.ValidateMessagePart(part)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMessageConverter_ValidateMessagePart_StronglyTyped(b *testing.B) {
	logger := zap.NewNop()
	converter := NewOptimizedMessageConverter(logger)

	part := adk.OptimizedMessagePart{
		Kind: adk.MessagePartKindText,
		Text: stringPtr("This is a strongly-typed test message part for validation benchmarking."),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := converter.ValidateMessagePart(part)
		if err != nil {
			b.Fatal(err)
		}
	}
}
