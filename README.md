<h1 align="center">Agent Development Kit (ADK)</h1>

<p align="center">
  <strong>Build powerful, interoperable AI agents with the Agent-to-Agent (A2A) protocol</strong>
</p>

<p align="center">
  <!-- CI Status Badge -->
  <a href="https://github.com/inference-gateway/a2a/actions/workflows/ci.yml?query=branch%3Amain">
    <img src="https://github.com/inference-gateway/a2a/actions/workflows/ci.yml/badge.svg?branch=main" alt="CI Status"/>
  </a>
  <!-- Version Badge -->
  <a href="https://github.com/inference-gateway/a2a/releases">
    <img src="https://img.shields.io/github/v/release/inference-gateway/a2a?color=blue&style=flat-square" alt="Version"/>
  </a>
  <!-- License Badge -->
  <a href="https://github.com/inference-gateway/a2a/blob/main/LICENSE">
    <img src="https://img.shields.io/github/license/inference-gateway/a2a?color=blue&style=flat-square" alt="License"/>
  </a>
  <!-- Go Version -->
  <img src="https://img.shields.io/github/go-mod/go-version/inference-gateway/a2a?style=flat-square" alt="Go Version"/>
</p>

---

- [Overview](#overview)
  - [What is A2A?](#what-is-a2a)
- [🚀 Quick Start](#-quick-start)
  - [Installation](#installation)
  - [Basic Usage (Minimal Server)](#basic-usage-minimal-server)
  - [AI-Powered Server](#ai-powered-server)
  - [Examples](#examples)
- [✨ Key Features](#-key-features)
  - [Core Capabilities](#core-capabilities)
  - [Developer Experience](#developer-experience)
  - [Production Ready](#production-ready)
- [🛠️ Development](#️-development)
  - [Prerequisites](#prerequisites)
  - [Development Workflow](#development-workflow)
  - [Available Tasks](#available-tasks)
- [📖 API Reference](#-api-reference)
  - [Core Components](#core-components)
    - [A2AServer](#a2aserver)
    - [A2AServerBuilder](#a2aserverbuilder)
    - [A2AClient](#a2aclient)
  - [Configuration](#configuration)
- [🔧 Advanced Usage](#-advanced-usage)
  - [Custom Tools](#custom-tools)
  - [Agent Metadata](#agent-metadata)
  - [Environment Configuration](#environment-configuration)
- [🌐 A2A Ecosystem](#-a2a-ecosystem)
  - [Related Projects](#related-projects)
  - [A2A Agents](#a2a-agents)
- [📋 Requirements](#-requirements)
- [🐳 Docker Support](#-docker-support)
- [🧪 Testing](#-testing)
- [📄 License](#-license)
- [🤝 Contributing](#-contributing)
  - [Getting Started](#getting-started)
  - [Development Guidelines](#development-guidelines)
  - [Before Submitting](#before-submitting)
  - [Pull Request Process](#pull-request-process)
- [📞 Support](#-support)
  - [Issues \& Questions](#issues--questions)
- [🗺️ Roadmap](#️-roadmap)
- [🔗 Resources](#-resources)
  - [Documentation](#documentation)

---

## Overview

The **A2A ADK (Agent Development Kit)** is a Go library that simplifies building [Agent-to-Agent (A2A) protocol](https://github.com/inference-gateway/schemas/tree/main/a2a) compatible agents. A2A enables seamless communication between AI agents, allowing them to collaborate, delegate tasks, and share capabilities across different systems and providers.

### What is A2A?

Agent-to-Agent (A2A) is a standardized protocol that enables AI agents to:

- **Communicate** with each other using a unified JSON-RPC interface
- **Delegate tasks** to specialized agents with specific capabilities
- **Stream responses** in real-time for better user experience
- **Authenticate** securely using OIDC/OAuth2
- **Discover capabilities** through standardized agent cards

## 🚀 Quick Start

### Installation

```bash
go get github.com/inference-gateway/a2a
```

### Basic Usage (Minimal Server)

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"

    "github.com/inference-gateway/a2a/adk/server"
    "go.uber.org/zap"
)

func main() {
    // Initialize logger
    logger, _ := zap.NewDevelopment()
    defer logger.Sync()

    // Create the simplest A2A server
    a2aServer := server.NewDefaultA2AServer(nil)

    // Start server
    ctx := context.Background()
    go func() {
        if err := a2aServer.Start(ctx); err != nil {
            log.Fatal("Server failed to start:", err)
        }
    }()

    logger.Info("Server running on port 8080")

    // Wait for shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    logger.Info("Shutting down server...")
    a2aServer.Stop(ctx)
}
```

### AI-Powered Server

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/inference-gateway/a2a/adk/server"
    "github.com/inference-gateway/a2a/adk/server/config"
    "github.com/sethvargo/go-envconfig"
    "go.uber.org/zap"
)

func main() {
    // Initialize logger
    logger, _ := zap.NewDevelopment()
    defer logger.Sync()

    // Load configuration from environment
    cfg := config.Config{
        AgentName:        "AI Assistant",
        AgentDescription: "An AI-powered A2A agent",
        Port:             "8080",
    }

    ctx := context.Background()
    if err := envconfig.Process(ctx, &cfg); err != nil {
        log.Fatal("Failed to load config:", err)
    }

    // Create toolbox with custom tools
    toolBox := server.NewDefaultToolBox()

    // Add a weather tool
    weatherTool := server.NewBasicTool(
        "get_weather",
        "Get weather information",
        map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "location": map[string]interface{}{
                    "type": "string",
                    "description": "City name",
                },
            },
            "required": []string{"location"},
        },
        func(ctx context.Context, args map[string]interface{}) (string, error) {
            location := args["location"].(string)
            return fmt.Sprintf(`{"location": "%s", "temperature": "22°C"}`, location), nil
        },
    )
    toolBox.AddTool(weatherTool)

    // Create LLM client (requires AGENT_CLIENT_API_KEY environment variable)
    var a2aServer server.A2AServer
    if cfg.AgentConfig != nil && cfg.AgentConfig.APIKey != "" {
        llmClient, err := server.NewOpenAICompatibleLLMClient(cfg.AgentConfig, logger)
        if err != nil {
            log.Fatal("Failed to create LLM client:", err)
        }
        agent := server.NewOpenAICompatibleAgentWithLLM(logger, llmClient)
        agent.SetToolBox(toolBox)
        a2aServer = server.SimpleA2AServerWithAgent(cfg, logger, agent)
    } else {
        // Mock mode without actual LLM
        agent := server.NewDefaultOpenAICompatibleAgent(logger)
        agent.SetToolBox(toolBox)
        a2aServer = server.NewA2AServerBuilder(cfg, logger).
            WithAgent(agent).
            Build()
    }

    // Start server
    go func() {
        if err := a2aServer.Start(ctx); err != nil {
            log.Fatal("Server failed to start:", err)
        }
    }()

    logger.Info("AI-powered A2A server running", zap.String("port", cfg.Port))

    // Wait for shutdown signal
    select {}
}
```

### Examples

For complete working examples, see the [examples](./examples/) directory:

- **[Minimal Server](./examples/server/cmd/minimal/)** - Basic A2A server without AI capabilities
- **[AI-Powered Server](./examples/server/cmd/aipowered/)** - Full A2A server with LLM integration
- **[Client Example](./examples/client/)** - A2A client implementation

## ✨ Key Features

### Core Capabilities

- 🤖 **A2A Protocol Compliance**: Full implementation of the Agent-to-Agent communication standard
- 🔌 **Multi-Provider Support**: Works with OpenAI, Ollama, Groq, Cohere, and other LLM providers
- 🌊 **Real-time Streaming**: Stream responses as they're generated from language models
- 🔧 **Custom Tools**: Easy integration of custom tools and capabilities
- 🔐 **Secure Authentication**: Built-in OIDC/OAuth2 authentication support

### Developer Experience

- ⚙️ **Environment Configuration**: Simple setup through environment variables
- 📊 **Task Management**: Built-in task queuing, polling, and lifecycle management
- 🏗️ **Extensible Architecture**: Pluggable components for custom business logic
- 📚 **Type-Safe**: Generated types from A2A schema for compile-time safety
- 🧪 **Well Tested**: Comprehensive test coverage with table-driven tests

### Production Ready

- 🌿 **Lightweight**: Optimized binary size
- 🛡️ **Production Hardened**: Configurable timeouts, TLS support, and error handling
- 🐳 **Containerized**: OCI compliant and works with Docker and Docker Compose
- ☸️ **Kubernetes Native**: Ready for cloud-native deployments
- 📊 **Observability**: OpenTelemetry integration for monitoring and tracing

## 🛠️ Development

### Prerequisites

- Go 1.24 or later
- [Task](https://taskfile.dev/) for build automation
- [golangci-lint](https://golangci-lint.run/) for linting

### Development Workflow

1. **Download latest A2A schema**:

   ```bash
   task a2a:download-schema
   ```

2. **Generate types from schema**:

   ```bash
   task a2a:generate-types
   ```

3. **Run linting**:

   ```bash
   task lint
   ```

4. **Build the application**:

   ```bash
   task build
   ```

5. **Run tests**:
   ```bash
   task test
   ```

### Available Tasks

| Task                       | Description                       |
| -------------------------- | --------------------------------- |
| `task a2a:download-schema` | Download the latest A2A schema    |
| `task a2a:generate-types`  | Generate Go types from A2A schema |
| `task generate:mocks`      | Generate all testing mocks        |
| `task lint`                | Run static analysis and linting   |
| `task build`               | Build the application binary      |
| `task test`                | Run all tests                     |
| `task tidy`                | Tidy Go modules                   |
| `task clean`               | Clean up build artifacts          |

## 📖 API Reference

### Core Components

#### A2AServer

The main server interface that handles A2A protocol communication.

```go
type A2AServer interface {
    // Start starts the A2A server on the configured port
    Start(ctx context.Context) error

    // Stop gracefully stops the A2A server
    Stop(ctx context.Context) error

    // GetAgentCard returns the agent's capabilities and metadata
    GetAgentCard() adk.AgentCard

    // ProcessTask processes a task with the given message
    ProcessTask(ctx context.Context, task *adk.Task, message *adk.Message) (*adk.Task, error)

    // SetTaskHandler sets the task handler for processing tasks
    SetTaskHandler(handler TaskHandler)

    // SetAgent sets the OpenAI-compatible agent for processing tasks
    SetAgent(agent OpenAICompatibleAgent)
}

// Create a default A2A server
func NewDefaultA2AServer(cfg *config.Config) A2AServer

// Create a server with agent integration
func SimpleA2AServerWithAgent(cfg config.Config, logger *zap.Logger, agent OpenAICompatibleAgent) A2AServer
```

#### A2AServerBuilder

Provides a fluent interface for building A2A servers with custom configurations.

```go
type A2AServerBuilder interface {
    // WithTaskHandler sets a custom task handler for processing A2A tasks
    WithTaskHandler(handler TaskHandler) A2AServerBuilder

    // WithTaskResultProcessor sets a custom task result processor
    WithTaskResultProcessor(processor TaskResultProcessor) A2AServerBuilder

    // WithAgent sets a pre-configured OpenAI-compatible agent
    WithAgent(agent OpenAICompatibleAgent) A2AServerBuilder

    // WithLogger sets a custom logger for the builder and resulting server
    WithLogger(logger *zap.Logger) A2AServerBuilder

    // Build creates and returns the configured A2A server
    Build() A2AServer
}

// Create a new server builder
func NewA2AServerBuilder(cfg config.Config, logger *zap.Logger) A2AServerBuilder
```

#### A2AClient

The client interface for communicating with A2A servers.

```go
type A2AClient interface {
    // Agent discovery
    GetAgentCard(ctx context.Context) (*adk.AgentCard, error)

    // Task operations
    SendTask(ctx context.Context, params adk.MessageSendParams) (*adk.JSONRPCSuccessResponse, error)
    SendTaskStreaming(ctx context.Context, params adk.MessageSendParams, eventChan chan<- interface{}) error
    GetTask(ctx context.Context, params adk.TaskQueryParams) (*adk.JSONRPCSuccessResponse, error)
    CancelTask(ctx context.Context, params adk.TaskIdParams) (*adk.JSONRPCSuccessResponse, error)

    // Configuration
    SetTimeout(timeout time.Duration)
    SetHTTPClient(client *http.Client)
    GetBaseURL() string
}
```

### Configuration

The configuration is managed through environment variables and the config package:

```go
type Config struct {
    AgentName                     string              `env:"AGENT_NAME,default=helloworld-agent"`
    AgentDescription              string              `env:"AGENT_DESCRIPTION,default=A simple greeting agent"`
    AgentURL                      string              `env:"AGENT_URL,default=http://helloworld-agent:8080"`
    AgentVersion                  string              `env:"AGENT_VERSION,default=1.0.0"`
    Debug                         bool                `env:"DEBUG,default=false"`
    Port                          string              `env:"PORT,default=8080"`
    StreamingStatusUpdateInterval time.Duration       `env:"STREAMING_STATUS_UPDATE_INTERVAL,default=1s"`
    AgentConfig                   *AgentConfig        `env:",prefix=AGENT_CLIENT_"`
    CapabilitiesConfig            *CapabilitiesConfig `env:",prefix=CAPABILITIES_"`
    TLSConfig                     *TLSConfig          `env:",prefix=TLS_"`
    AuthConfig                    *AuthConfig         `env:",prefix=AUTH_"`
    QueueConfig                   *QueueConfig        `env:",prefix=QUEUE_"`
    ServerConfig                  *ServerConfig       `env:",prefix=SERVER_"`
    TelemetryConfig               *TelemetryConfig    `env:",prefix=TELEMETRY_"`
}

type AgentConfig struct {
    Provider                    string            `env:"PROVIDER"`
    Model                       string            `env:"MODEL"`
    BaseURL                     string            `env:"BASE_URL"`
    APIKey                      string            `env:"API_KEY"`
    Timeout                     time.Duration     `env:"TIMEOUT,default=30s"`
    MaxRetries                  int               `env:"MAX_RETRIES,default=3"`
    MaxChatCompletionIterations int               `env:"MAX_CHAT_COMPLETION_ITERATIONS,default=10"`
    MaxTokens                   int               `env:"MAX_TOKENS,default=4096"`
    Temperature                 float64           `env:"TEMPERATURE,default=0.7"`
    SystemPrompt                string            `env:"SYSTEM_PROMPT"`
    QueueConfig                 QueueConfig       `env:",prefix=QUEUE_"`
}

```

## 🔧 Advanced Usage

### Custom Tools

Create custom tools to extend your agent's capabilities:

```go
// Create a toolbox
toolBox := server.NewDefaultToolBox()

// Create a custom tool using NewBasicTool
weatherTool := server.NewBasicTool(
    "get_weather",
    "Get current weather for a location",
    map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "location": map[string]interface{}{
                "type":        "string",
                "description": "The city and state, e.g. San Francisco, CA",
            },
        },
        "required": []string{"location"},
    },
    func(ctx context.Context, args map[string]interface{}) (string, error) {
        location := args["location"].(string)

        // Your weather API logic here
        result := getWeather(location)

        response, _ := json.Marshal(result)
        return string(response), nil
    },
)

// Add the tool to the toolbox
toolBox.AddTool(weatherTool)

// Set the toolbox on your agent
agent.SetToolBox(toolBox)
```

### Custom Task Processing

Implement custom business logic for task completion:

```go
type CustomTaskProcessor struct{}

func (ctp *CustomTaskProcessor) ProcessToolResult(toolCallResult string) *adk.Message {
    // Parse the tool result
    var result map[string]interface{}
    json.Unmarshal([]byte(toolCallResult), &result)

    // Apply your business logic
    if shouldCompleteTask(result) {
        return &adk.Message{
            Role:    "assistant",
            Parts: []adk.Part{
                {
                    Kind:    "text",
                    Content: "Task completed successfully!",
                },
            },
        }
    }

    // Return nil to continue processing
    return nil
}

// Set the processor when building your server
server := server.NewA2AServerBuilder(cfg, logger).
    WithTaskResultProcessor(&CustomTaskProcessor{}).
    Build()
```

### Agent Metadata

Customize your agent's capabilities and metadata through configuration:

```go
cfg := config.Config{
    AgentName:        "Weather Assistant",
    AgentDescription: "Specialized weather analysis agent",
    AgentVersion:     "2.0.0",
    CapabilitiesConfig: &config.CapabilitiesConfig{
        Streaming:              true,
        PushNotifications:      true,
        StateTransitionHistory: false,
    },
}
```

### Environment Configuration

Key environment variables for configuring your agent:

```bash
# Basic agent information
AGENT_NAME="My Agent"
AGENT_DESCRIPTION="A helpful AI assistant"
AGENT_VERSION="1.0.0"
PORT="8080"

# LLM client configuration
AGENT_CLIENT_PROVIDER="openai"              # openai, anthropic, deepseek, ollama
AGENT_CLIENT_MODEL="gpt-4"                  # Model name
AGENT_CLIENT_API_KEY="your-api-key"         # Required for AI features
AGENT_CLIENT_BASE_URL="https://api.openai.com/v1"  # Custom endpoint
AGENT_CLIENT_MAX_TOKENS="4096"              # Max tokens for completion
AGENT_CLIENT_TEMPERATURE="0.7"              # Temperature for completion
AGENT_CLIENT_SYSTEM_PROMPT="You are a helpful assistant"

# Capabilities
CAPABILITIES_STREAMING="true"
CAPABILITIES_PUSH_NOTIFICATIONS="true"
CAPABILITIES_STATE_TRANSITION_HISTORY="false"

# Authentication (optional)
AUTH_ENABLE="false"
AUTH_ISSUER_URL="http://keycloak:8080/realms/inference-gateway-realm"
AUTH_CLIENT_ID="inference-gateway-client"
AUTH_CLIENT_SECRET="your-secret"

# TLS (optional)
TLS_ENABLE="false"
TLS_CERT_PATH="/path/to/cert.pem"
TLS_KEY_PATH="/path/to/key.pem"
```

## 🌐 A2A Ecosystem

This ADK is part of the broader Inference Gateway ecosystem:

### Related Projects

- **[Inference Gateway](https://github.com/inference-gateway/inference-gateway)** - Unified API gateway for AI providers
- **[Go SDK](https://github.com/inference-gateway/go-sdk)** - Go client library for Inference Gateway
- **[TypeScript SDK](https://github.com/inference-gateway/typescript-sdk)** - TypeScript/JavaScript client library
- **[Python SDK](https://github.com/inference-gateway/python-sdk)** - Python client library
- **[Rust SDK](https://github.com/inference-gateway/rust-sdk)** - Rust client library

### A2A Agents

- **[Awesome A2A](https://github.com/inference-gateway/awesome-a2a)** - Curated list of A2A-compatible agents
- **[Google Calendar Agent](https://github.com/inference-gateway/google-calendar-agent)** - Google Calendar integration agent

## 📋 Requirements

- **Go**: 1.24 or later
- **Dependencies**: See [go.mod](./go.mod) for full dependency list

## 🐳 Docker Support

Build and run your agent in a container:

```dockerfile
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o bin/agent .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/bin/agent .
CMD ["./agent"]
```

## 🧪 Testing

The ADK follows table-driven testing patterns and provides comprehensive test coverage:

```go
func TestA2AServerEndpoints(t *testing.T) {
    tests := []struct {
        name           string
        endpoint       string
        method         string
        expectedStatus int
    }{
        {
            name:           "health check",
            endpoint:       "/health",
            method:         "GET",
            expectedStatus: http.StatusOK,
        },
        {
            name:           "agent info",
            endpoint:       "/.well-known/agent.json",
            method:         "GET",
            expectedStatus: http.StatusOK,
        },
        {
            name:           "a2a endpoint",
            endpoint:       "/a2a",
            method:         "POST",
            expectedStatus: http.StatusOK,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Each test case has isolated mocks
            server := setupTestServer(t)
            defer server.Close()

            // Test implementation with table-driven approach
            resp := makeRequest(t, server, tt.method, tt.endpoint)
            assert.Equal(t, tt.expectedStatus, resp.StatusCode)
        })
    }
}
```

Run tests with:

```bash
task test
```

## 📄 License

This project is licensed under the MIT License. See the [LICENSE](./LICENSE) file for details.

## 🤝 Contributing

We welcome contributions! Here's how you can help:

### Getting Started

1. **Fork the repository**
2. **Clone your fork**:

   ```bash
   git clone https://github.com/your-username/a2a.git
   cd a2a
   ```

3. **Create a feature branch**:
   ```bash
   git checkout -b feature/amazing-feature
   ```

### Development Guidelines

- Follow the established code style and conventions
- Write table-driven tests for new functionality
- Use early returns to simplify logic and avoid deep nesting
- Prefer switch statements over if-else chains
- Ensure type safety with proper interfaces
- Use lowercase log messages for consistency

### Before Submitting

1. **Download latest schema**: `task a2a:download-schema`
2. **Generate types**: `task a2a:generate-types`
3. **Generate mocks** (if interfaces changed): `task generate:mocks`
4. **Run linting**: `task lint`
5. **Build successfully**: `task build`
6. **All tests pass**: `task test`

### Pull Request Process

1. Update documentation for any new features
2. Add tests for new functionality
3. Ensure all CI checks pass
4. Request review from maintainers

For more details, see [CONTRIBUTING.md](./CONTRIBUTING.md).

## 📞 Support

### Issues & Questions

- **Bug Reports**: [GitHub Issues](https://github.com/inference-gateway/a2a/issues)
- **Documentation**: [Official Docs](https://docs.inference-gateway.com)

## 🗺️ Roadmap

- [x] **Enhanced Tool System**: More built-in tools and better tool chaining
- [ ] **Agent Discovery**: Automatic discovery and registration of agents
- [ ] **Monitoring Dashboard**: Built-in monitoring and analytics
- [ ] **Multi-language SDKs**: Additional language support beyond Go
- [ ] **Performance Optimization**: Further reduce resource consumption

## 🔗 Resources

### Documentation

- [A2A Protocol Specification](https://github.com/inference-gateway/schemas/tree/main/a2a)
- [API Documentation](https://docs.inference-gateway.com/a2a)

---

<p align="center">
  <strong>Built with ❤️ by the Inference Gateway team</strong>
</p>

<p align="center">
  <a href="https://github.com/inference-gateway">GitHub</a> •
  <a href="https://docs.inference-gateway.com">Documentation</a>
</p>
