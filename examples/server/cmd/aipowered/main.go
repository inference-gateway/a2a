package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	server "github.com/inference-gateway/a2a/adk/server"
	config "github.com/inference-gateway/a2a/adk/server/config"
	envconfig "github.com/sethvargo/go-envconfig"
	zap "go.uber.org/zap"
)

// AI-Powered A2A Server Example
//
// This example demonstrates how to create an A2A server with AI capabilities.
// It requires proper AI provider configuration to run.
//
// REQUIRED Configuration:
//
//	AGENT_CLIENT_API_KEY - Your LLM provider API key (REQUIRED)
//
// Optional Configuration:
//
//	AGENT_CLIENT_PROVIDER - LLM provider: "openai", "anthropic", "deepseek", "ollama" (default: "openai")
//	AGENT_CLIENT_MODEL    - Model name (uses provider defaults if not set)
//	AGENT_CLIENT_BASE_URL - Custom API endpoint URL
//	PORT                  - Server port (default: "8080")
//
// Examples:
//
//	# OpenAI (default)
//	export AGENT_CLIENT_API_KEY="sk-..." && go run main.go
//
//	# Anthropic
//	export AGENT_CLIENT_API_KEY="sk-ant-..." AGENT_CLIENT_PROVIDER="anthropic" && go run main.go
//
//	# Via Inference Gateway
//	AGENT_CLIENT_API_KEY="test" AGENT_CLIENT_PROVIDER="deepseek" AGENT_CLIENT_MODEL="deepseek-chat" AGENT_CLIENT_BASE_URL="http://localhost:8080/v1" go run main.go
//
// To run: go run main.go
func main() {
	fmt.Println("🤖 Starting AI-Powered A2A Server...")

	// Step 1: Initialize logger
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Step 2: Check for required API key
	apiKey := os.Getenv("AGENT_CLIENT_API_KEY")
	if apiKey == "" {
		fmt.Println("\n❌ ERROR: AI provider configuration required!")
		fmt.Println("\n🔑 Please set AGENT_CLIENT_API_KEY environment variable:")
		fmt.Println("\n📋 Examples:")
		fmt.Println("  # OpenAI")
		fmt.Println("  export AGENT_CLIENT_API_KEY=\"sk-...\"")
		fmt.Println("\n  # Anthropic")
		fmt.Println("  export AGENT_CLIENT_API_KEY=\"sk-ant-...\"")
		fmt.Println("  export AGENT_CLIENT_PROVIDER=\"anthropic\"")
		fmt.Println("\n  # Via Inference Gateway")
		fmt.Println("  export AGENT_CLIENT_API_KEY=\"your-key\"")
		fmt.Println("  export AGENT_CLIENT_BASE_URL=\"http://localhost:3000/v1\"")
		fmt.Println("\n💡 For a server without AI, use the minimal example instead:")
		fmt.Println("  go run ../minimal/main.go")
		os.Exit(1)
	}

	// Step 3: Load configuration from environment
	cfg := config.Config{
		AgentName:        "AI-Powered Assistant",
		AgentDescription: "An AI assistant with weather and time tools",
		Port:             "8080",
		QueueConfig: &config.QueueConfig{
			CleanupInterval: 5 * time.Minute,
		},
	}

	ctx := context.Background()
	if err := envconfig.Process(ctx, &cfg); err != nil {
		logger.Fatal("failed to process environment config", zap.Error(err))
	}

	// Step 4: Create AI agent with LLM client
	llmClient, err := server.NewOpenAICompatibleLLMClient(cfg.AgentConfig, logger)
	if err != nil {
		logger.Fatal("failed to create LLM client", zap.Error(err))
	}

	agent := server.NewOpenAICompatibleAgentWithLLM(logger, llmClient)
	agent.SetSystemPrompt("You are a helpful AI assistant. Be concise and friendly in your responses.")

	// Step 5: Create toolbox with sample tools
	toolBox := server.NewDefaultToolBox()

	// Add weather tool
	weatherTool := server.NewBasicTool(
		"get_weather",
		"Get current weather information for a location",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"location": map[string]interface{}{
					"type":        "string",
					"description": "The city name",
				},
			},
			"required": []string{"location"},
		},
		func(ctx context.Context, args map[string]interface{}) (string, error) {
			location := args["location"].(string)
			return fmt.Sprintf(`{"location": "%s", "temperature": "22°C", "condition": "sunny", "humidity": "65%%"}`, location), nil
		},
	)
	toolBox.AddTool(weatherTool)

	// Add time tool
	timeTool := server.NewBasicTool(
		"get_current_time",
		"Get the current date and time",
		map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		func(ctx context.Context, args map[string]interface{}) (string, error) {
			now := time.Now()
			return fmt.Sprintf(`{"current_time": "%s", "timezone": "%s"}`,
				now.Format("2006-01-02 15:04:05"), now.Location()), nil
		},
	)
	toolBox.AddTool(timeTool)

	agent.SetToolBox(toolBox)

	// Step 6: Create and start server
	a2aServer := server.SimpleA2AServerWithAgent(cfg, logger, agent)

	logger.Info("✅ AI-powered A2A server created",
		zap.String("provider", cfg.AgentConfig.Provider),
		zap.String("model", cfg.AgentConfig.Model),
		zap.String("tools", "weather, time"))

	// Start server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := a2aServer.Start(ctx); err != nil {
			logger.Fatal("server failed to start", zap.Error(err))
		}
	}()

	logger.Info("🌐 server running", zap.String("port", cfg.Port))
	fmt.Printf("\n🎯 Test with curl:\n")
	fmt.Printf(`curl -X POST http://localhost:%s/a2a \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "message/send",
    "params": {
      "message": {
        "kind": "message",
        "messageId": "msg-123",
        "role": "user",
        "parts": [
          {
            "kind": "text",
            "text": "What'\''s the weather in Tokyo?"
          }
        ]
      }
    },
    "id": 1
  }'`, cfg.Port)
	fmt.Println()

	// Wait for shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("🛑 shutting down server...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := a2aServer.Stop(shutdownCtx); err != nil {
		logger.Error("shutdown error", zap.Error(err))
	} else {
		logger.Info("✅ goodbye!")
	}
}
