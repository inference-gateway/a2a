# Changelog

All notable changes to this project will be documented in this file.

## [0.2.0-rc.2](https://github.com/inference-gateway/a2a/compare/v0.2.0-rc.1...v0.2.0-rc.2) (2025-06-15)

### ♻️ Improvements

* Enhance message conversion logging in convertToSDKMessages ([e6b0bd8](https://github.com/inference-gateway/a2a/commit/e6b0bd8c4456d5f6aa5af3e4761ec8ec9a8471e0))

## [0.2.0-rc.1](https://github.com/inference-gateway/a2a/compare/v0.1.9-rc.5...v0.2.0-rc.1) (2025-06-15)

### ✨ Features

* Implement conversation history management in TaskManager ([ce299e5](https://github.com/inference-gateway/a2a/commit/ce299e531a87bfcfdea1ef3cb459371a040e791c))

## [0.1.9-rc.5](https://github.com/inference-gateway/a2a/compare/v0.1.9-rc.4...v0.1.9-rc.5) (2025-06-15)

### ♻️ Improvements

* Add context_id to logging for better traceability in task processing ([fbe1e4e](https://github.com/inference-gateway/a2a/commit/fbe1e4e6bcb9f18504fd32aa473c90c19e241098))

## [0.1.9-rc.4](https://github.com/inference-gateway/a2a/compare/v0.1.9-rc.3...v0.1.9-rc.4) (2025-06-15)

### 🐛 Bug Fixes

* Update SendTask and SendTaskStreaming to use getA2AEndpointURL for constructing the endpoint URL ([fa58951](https://github.com/inference-gateway/a2a/commit/fa589512bf89f7b55c0fa6c2ce6322ecd17dd06c))

## [0.1.9-rc.3](https://github.com/inference-gateway/a2a/compare/v0.1.9-rc.2...v0.1.9-rc.3) (2025-06-15)

### 🐛 Bug Fixes

* Have to ensure the assistant message is first appended with the tool_calls before I can add a message of role tool with the results ([1b679f5](https://github.com/inference-gateway/a2a/commit/1b679f5309db415f530d71101fca27a49d50b005))

## [0.1.9-rc.2](https://github.com/inference-gateway/a2a/compare/v0.1.9-rc.1...v0.1.9-rc.2) (2025-06-15)

### 🐛 Bug Fixes

* Add toolCallId handling in convertToSDKMessages for messages with the role tool ([87b4d64](https://github.com/inference-gateway/a2a/commit/87b4d64840a21b5a973d6bfcba2cdea24d521307))

## [0.1.9-rc.1](https://github.com/inference-gateway/a2a/compare/v0.1.8...v0.1.9-rc.1) (2025-06-15)

### 🐛 Bug Fixes

* Update tool result message structure and enhance test coverage for tool processing ([2937e0f](https://github.com/inference-gateway/a2a/commit/2937e0fbdb114ba80eb6406b607525452befbb36))

## [0.1.8](https://github.com/inference-gateway/a2a/compare/v0.1.7...v0.1.8) (2025-06-15)

### 🐛 Bug Fixes

* Empty message for tool processing - when pulling a task from the Queue we need to check the task.Status.Message which is where the task handler is storing the original message ([703dfba](https://github.com/inference-gateway/a2a/commit/703dfba8cf033a33385da66692bc70e49b61d531))

## [0.1.7](https://github.com/inference-gateway/a2a/compare/v0.1.6...v0.1.7) (2025-06-15)

### 🐛 Bug Fixes

* Update API key validation to be optional in NewOpenAICompatibleLLMClient ([ea0b1bd](https://github.com/inference-gateway/a2a/commit/ea0b1bd73111266da8c241ad51d38c7b82a2d5bd))

### 📚 Documentation

* Update CONTRIBUTING.md for clarity on type generation and mocks ([438fb91](https://github.com/inference-gateway/a2a/commit/438fb91254e0db9e5e8d8198ec619c0e4fa4c2cd))

## [0.1.6](https://github.com/inference-gateway/a2a/compare/v0.1.5...v0.1.6) (2025-06-14)

### 📚 Documentation

* **fix:** Update README title and improve section organization ([03a086e](https://github.com/inference-gateway/a2a/commit/03a086ea0154c1c097c381c95eca34eb7156a7c5))
* Remove "Advanced Authentication" item from the roadmap ([322579e](https://github.com/inference-gateway/a2a/commit/322579e7247e377754e105efb17a2df948330a67))
* Remove redundant Table of Contents section from README.md ([4c9e33a](https://github.com/inference-gateway/a2a/commit/4c9e33a7ed29926f26b404b6b66635252ba619c7))
* Update README.md with enhanced table of contents and usage examples ([00d31eb](https://github.com/inference-gateway/a2a/commit/00d31eb78060d2b8cf4c0eddbc7a9a46bf4654a7))

### 🔧 Miscellaneous

* Update README.md ([e3a9480](https://github.com/inference-gateway/a2a/commit/e3a9480ad5cec677b1d494456c7cfb16a9efbe15))

## [0.1.5](https://github.com/inference-gateway/a2a/compare/v0.1.4...v0.1.5) (2025-06-14)

### ✅ Miscellaneous

* Test the configurations and ensure everything works as expected ([#3](https://github.com/inference-gateway/a2a/issues/3)) ([aa0629e](https://github.com/inference-gateway/a2a/commit/aa0629e74a890c516263b343c5e5c20acae13523))

## [0.1.4](https://github.com/inference-gateway/a2a/compare/v0.1.3...v0.1.4) (2025-06-14)

### ♻️ Improvements

* Cleanup - remove unintended usage of functions ([#2](https://github.com/inference-gateway/a2a/issues/2)) ([9275cb5](https://github.com/inference-gateway/a2a/commit/9275cb5208ccdf75503e2d8939e0bdd784cd7d25))

## [0.1.3](https://github.com/inference-gateway/a2a/compare/v0.1.2...v0.1.3) (2025-06-14)

### ♻️ Improvements

* Improve the structure of the library ([#1](https://github.com/inference-gateway/a2a/issues/1)) ([b41c040](https://github.com/inference-gateway/a2a/commit/b41c0400710eae83a55e07b41067d5b0966a0fe9))

## [0.1.2](https://github.com/inference-gateway/a2a/compare/v0.1.1...v0.1.2) (2025-06-12)

### ♻️ Improvements

* Replace a2a types with local types in agent.go and update go.mod/go.sum ([35274b6](https://github.com/inference-gateway/a2a/commit/35274b6bb31f1ae1b42fb122285a6d0ba45dcd09))

### 🐛 Bug Fixes

* Update documentation link for Core Inference Gateway ([0fd71ce](https://github.com/inference-gateway/a2a/commit/0fd71cef658ddafe45206072762f0b9d48e15ce5))

## [0.1.1](https://github.com/inference-gateway/a2a/compare/v0.1.0...v0.1.1) (2025-06-12)

### ♻️ Improvements

* Add clean task to remove build artifacts ([b2f0030](https://github.com/inference-gateway/a2a/commit/b2f00307dfc5c87f4cfc110b4d7b2df5318a84b2))
* Remove unused semantic release plugin from Dockerfile and release workflow ([66febcf](https://github.com/inference-gateway/a2a/commit/66febcfb6b77d7af412f45e241be1b96d9fb1283))

### 🐛 Bug Fixes

* Update repository URL in release configuration ([bc4606b](https://github.com/inference-gateway/a2a/commit/bc4606b74a3f93954070abb551b960d452c247b4))
