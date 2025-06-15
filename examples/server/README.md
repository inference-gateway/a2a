# A2A Server Examples

This directory contains examples demonstrating how to create A2A (Agent-to-Agent) compatible servers using the A2A ADK (Agent Development Kit).

## Overview

The A2A protocol enables agents to communicate with each other using JSON-RPC over HTTP. These examples show different approaches to creating A2A servers:

1. **Minimal Server** - A working server with custom task handler that provides simple responses without AI
2. **AI-Powered Server** - A full-featured server with LLM integration and tool calling capabilities

## Quick Start

### 1. Minimal Server (No AI Required)

A working A2A server with simple conversational responses using a custom task handler:

```bash
cd cmd/minimal
go run main.go
```

This minimal example:

- ✅ Handles A2A protocol messages correctly (`message/send`, `tasks/get`, `tasks/cancel`)
- ✅ Provides conversational responses (greetings, status, help, time)
- ✅ Agent metadata via `/.well-known/agent.json`
- ✅ Health check endpoint at `/health`
- ✅ Echo functionality for any text input
- ✅ **Works immediately** - no configuration required
- ❌ No AI/LLM integration (by design)
- ❌ No advanced tools or function calling

Perfect for learning the A2A protocol, creating deterministic business logic agents, or simple automation tasks.

### 2. AI-Powered Server (API Key Required)

For AI capabilities with LLM integration and tool calling:

```bash
cd cmd/aipowered

# Required: Set your API key
export AGENT_CLIENT_API_KEY="sk-..."  # OpenAI
# OR
export AGENT_CLIENT_API_KEY="sk-ant-..." AGENT_CLIENT_PROVIDER="anthropic"  # Anthropic

go run main.go
```

This AI-powered example:

- ✅ **Requires valid API key** - will not start without proper configuration
- ✅ Supports multiple LLM providers (OpenAI, Anthropic, DeepSeek, Ollama)
- ✅ Tool calling capabilities (weather, time tools included)
- ✅ Full conversation context and history
- ✅ Works with Inference Gateway for unified LLM access
- ✅ Production-ready AI agent architecture

## Example Usage

### Test the A2A Protocol

Both servers support the standard A2A protocol. Here's how to test them:

```bash
# Get agent information
curl http://localhost:8080/.well-known/agent.json | jq .

# Send a message (proper A2A format)
curl -X POST http://localhost:8080/a2a \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "message/send",
    "params": {
      "message": {
        "kind": "message",
        "messageId": "msg-001",
        "role": "user",
        "parts": [
          {
            "kind": "text",
            "text": "Hello! Can you help me?"
          }
        ]
      }
    },
    "id": 1
  }' | jq .

# Get task results
curl -X POST http://localhost:8080/a2a \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tasks/get",
    "params": {
      "id": "TASK_ID_FROM_PREVIOUS_RESPONSE"
    },
    "id": 2
  }' | jq .
```

## Configuration

### Minimal Server
- `PORT` - Server port (default: "8080")
- No other configuration required!

### AI-Powered Server
**Required:**
- `AGENT_CLIENT_API_KEY` - Your LLM provider API key

**Required for specific providers:**
- `AGENT_CLIENT_PROVIDER` - LLM provider: "openai", "anthropic", "deepseek", "ollama" (required for non-OpenAI providers)
- `AGENT_CLIENT_BASE_URL` - Custom API endpoint (required for Ollama, Inference Gateway, or custom deployments)

**Optional:**
- `AGENT_CLIENT_MODEL` - Model name (uses provider defaults if not specified)
- `PORT` - Server port (default: "8080")

### Provider Examples

```bash
# OpenAI (default)
export AGENT_CLIENT_API_KEY="sk-..."

# Anthropic
export AGENT_CLIENT_API_KEY="sk-ant-..."
export AGENT_CLIENT_PROVIDER="anthropic"

# Via Inference Gateway
export AGENT_CLIENT_API_KEY="your-key"
export AGENT_CLIENT_BASE_URL="http://localhost:3000/v1"

# Local Ollama
export AGENT_CLIENT_PROVIDER="ollama"
export AGENT_CLIENT_MODEL="llama3.2"
export AGENT_CLIENT_BASE_URL="http://localhost:11434/v1"
```

## Architecture

### Minimal Server
- **CustomTaskHandler**: Processes messages with simple business logic
- **No LLM dependency**: Fast, deterministic responses
- **A2A compliant**: Full protocol support without AI complexity

### AI-Powered Server  
- **OpenAICompatibleAgent**: Handles LLM communication
- **ToolBox**: Function calling capabilities  
- **Multiple LLM Providers**: Flexible provider support
- **Conversation Management**: Context-aware interactions

## Files

- `cmd/minimal/main.go` - Simple working server with custom task handler
- `cmd/aipowered/main.go` - AI-powered server with LLM integration and tools
- `README.md` - This documentation

## Next Steps

1. **Start Simple**: Run the minimal example to understand A2A protocol basics
2. **Add Business Logic**: Customize the task handler for your specific use case  
3. **Add AI**: Use the AI-powered example with your API key for intelligent responses
4. **Extend Tools**: Add custom tools and functions for your domain
5. **Production**: See the main ADK documentation for advanced patterns and deployment

For more information, see the [A2A ADK documentation](../../README.md) and [A2A Protocol Specification](https://github.com/inference-gateway/schemas/tree/main/a2a).
