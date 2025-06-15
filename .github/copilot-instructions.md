# Custom Instructions for Copilot

Today is June 15, 2025.

- Always use context7 to check for the latest updates, features, or best practices of a library relevant to the task at hand.
- Always prefer Table-Driven Testing: When writing tests.
- Always use Early Returns: Favor early returns to simplify logic and avoid deep nesting with if-else structures.
- Always prefer switch statements over if-else chains: Use switch statements for cleaner and more readable code when checking multiple conditions.
- Always run `task lint` before committing code to ensure it adheres to the project's linting rules.
- Always download the latest schema: `a2a:download-schema`.
- Always generate the types after downloading the schema: `a2a:generate-types`.
- Always run `task build` to verify compilation after making changes.
- Always run `task test` before committing code to ensure all tests pass.
- Always search for the simplest solution first before considering more complex alternatives.
- Always prefer type safety over dynamic typing: Use strong typing and interfaces to ensure type safety and reduce runtime errors.
- Always use lowercase log messages for consistency and readability.
- When possible code to an interface so it's easier to mock in tests.
- When writing tests, each test case should have it's own isolated mock server mock dependencies so it's easier to understand and maintain.

## Development Workflow

1. Run `task a2a:download-schema` to download the latest A2A schema - when working on A2A.
2. Run `task a2a:generate-types` If added new Schemas to openapi.yaml, update internal/openapi/schemas.go to include the new schemas.
3. Run `task generate:mocks` to generate testing mocks, when interfaces are adjusted.
4. Run `task lint` to ensure code quality.
5. Run `task build` to verify successful compilation.
6. Run `task test` to ensure all tests pass.

## Available Tools and MCPs

- context7 - Helps by finding the latest updates, features, or best practices of a library relevant to the task at hand.

## Related Repositories

### Core Inference Gateway

- **[Main Repository](https://github.com/inference-gateway)** - The main inference gateway org.
- **[Documentation](https://docs.inference-gateway.com)** - Official documentation and guides
- **[UI](https://github.com/inference-gateway/ui)** - Web interface for the inference gateway
- **[Schemas](https://github.com/inference-gateway/schemas)** - Shared schemas and type definitions

### SDKs & Client Libraries

- **[Go SDK](https://github.com/inference-gateway/go-sdk)** - Go client library
- **[Rust SDK](https://github.com/inference-gateway/rust-sdk)** - Rust client library
- **[TypeScript SDK](https://github.com/inference-gateway/typescript-sdk)** - TypeScript/JavaScript client library
- **[Python SDK](https://github.com/inference-gateway/python-sdk)** - Python client library

### A2A (Agent-to-Agent) Ecosystem

- **[Awesome A2A](https://github.com/inference-gateway/awesome-a2a)** - Curated list of A2A-compatible agents
- **[Google Calendar Agent](https://github.com/inference-gateway/google-calendar-agent)** - Agent for Google Calendar integration

### Internal Tools

- **[Internal Tools](https://github.com/inference-gateway/tools)** - Collection of internal tools and utilities

## A2A Official Documentation

- **[A2A Official Docs](https://google-a2a.github.io/A2A/latest/)** - Official documentation for A2A agents and integrations
