# `mcpd` Plugin Examples

> **Note:** This repository contains example plugins demonstrating how to build plugins for `mcpd`'s plugin system using language-specific SDKs.

A collection of working plugin examples in multiple languages (Go, C#/.NET, Python) that demonstrate best practices for building `mcpd` plugins.

## Overview

This repository provides reference implementations for building `mcpd` plugins in different programming languages. 

Each example demonstrates:

* How to use the language-specific SDK
* Proper plugin lifecycle management
* Request/response processing patterns
* Language-specific best practices

## Available Plugin Examples

### 1. Rate Limit Plugin (Go)
**Location:** `sample-plugins/rate-limit/`

Demonstrates request rate limiting using in-memory state management.

**Features:**
- Token bucket rate limiting algorithm
- Per-client request tracking
- State management in plugins
- Using the Go SDK

**SDK:** [mcpd-plugins-sdk-go](https://github.com/mozilla-ai/mcpd-plugins-sdk-go)

### 2. Tool Audit Plugin (Go)
**Location:** `sample-plugins/tool-audit/`

Demonstrates request logging and auditing for MCP tool calls.

**Features:**
- JSON request body parsing
- Header inspection and manipulation
- Audit logging patterns
- Using the Go SDK

**SDK:** [mcpd-plugins-sdk-go](https://github.com/mozilla-ai/mcpd-plugins-sdk-go)

### 3. Header Transformer Plugin (Go)
**Location:** `sample-plugins/header-transformer/`

Demonstrates HTTP header manipulation and transformation.

**Features:**
- Request header modification
- Adding custom metadata
- Header-based routing logic
- Using the Go SDK

**SDK:** [mcpd-plugins-sdk-go](https://github.com/mozilla-ai/mcpd-plugins-sdk-go)

### 4. Prompt Guard Plugin (C#/.NET)
**Location:** `sample-plugins/prompt-guard/`

Demonstrates content filtering and validation using the .NET SDK.

**Features:**
- JSON body scanning for prohibited content
- Request rejection with appropriate HTTP status codes
- Modern C# pattern matching
- Full plugin lifecycle implementation (Configure, Stop, CheckHealth, CheckReady)
- Using the .NET SDK with BasePlugin inheritance

**SDK:** [mcpd-plugins-sdk-dotnet](https://github.com/mozilla-ai/mcpd-plugins-sdk-dotnet)

### 5. Header Injector Plugin (Python)
**Location:** `sample-plugins/header-injector/`

Reference implementation showing Python plugin structure.

**Features:**
- Python-based plugin example
- Demonstrates interpreted language approach
- Shows gRPC usage in Python

**Note:** This is a reference implementation. Python plugins require additional work via PyInstaller etc. to produce an executable binary.

## Building the Examples

### Prerequisites

* **Go 1.25+** (for Go plugins)
* **.NET 9.0+** (for C# plugin)
* **Python 3.12+ with uv** (optional, for Python plugin)

### Build All Plugins

```bash
make
```

This will build all compiled plugins and place them in `bin/sample-plugins/`:
- `rate-limit-plugin` (Go, ~14MB)
- `tool-audit-plugin` (Go, ~14MB)
- `header-transformer-plugin` (Go, ~14MB)
- `prompt-guard-plugin` (C#/.NET, ~104MB)

### Build Individual Plugins

**Go plugins:**
```bash
cd sample-plugins/rate-limit
go build -o ../../bin/sample-plugins/rate-limit-plugin .
```

**C# plugin:**
```bash
cd sample-plugins/prompt-guard
dotnet publish PromptGuard/PromptGuard.csproj -c Release -r osx-arm64 --self-contained /p:PublishSingleFile=true
```

**Python plugin:**
```bash
cd sample-plugins/header-injector
uv sync
uv run python generate_proto.py
```

## Plugin Architecture

All plugins in this repository follow the `mcpd` plugin architecture, which uses gRPC for communication and provides:

### Two-Service Model

**PluginManager Service (Lifecycle):**
- `GetInfo()` - Plugin metadata and information
- `Initialize()` - Plugin startup and initialization
- `Configure()` - Configuration management
- `Shutdown()` - Graceful shutdown
- `CheckHealth()` - Health monitoring

**Middleware Service (Request Processing):**
- `ShouldHandle()` - Conditional application logic
- `ProcessRequest()` - Request processing with continue/stop semantics

### Cross-Platform Communication

- **Unix-like systems** (Linux, macOS, BSD): Uses Unix domain sockets
- **Windows**: Uses TCP loopback connections
- **Communication**: gRPC with Protocol Buffers

## Choosing a Language

### Compiled Languages (Recommended for Production)

**Go** and **C#/.NET** plugins compile to self-contained binaries:
- ✅ Single executable file
- ✅ No runtime dependencies required
- ✅ Easy distribution and deployment
- ✅ Excellent performance

**Examples in this repo:** `rate-limit`, `tool-audit`, `header-transformer` (Go), `prompt-guard` (C#/.NET)

### Interpreted Languages (Development/Testing)

**Python**, **Node.js**, **Ruby** plugins require runtime:
- ⚠️ Requires language runtime + dependencies
- ⚠️ More complex deployment
- ✅ Rapid development and iteration
- ✅ Good for prototyping

**Example in this repo:** `header-injector` (Python, reference only)

## SDK Documentation

Each language has its own SDK with detailed documentation:

- **Go SDK:** [mcpd-plugins-sdk-go](https://github.com/mozilla-ai/mcpd-plugins-sdk-go)
- **.NET SDK:** [mcpd-plugins-sdk-dotnet](https://github.com/mozilla-ai/mcpd-plugins-sdk-dotnet)
- **Python SDK:** Coming soon

## Using These Examples

### As Learning Resources

Each plugin example is fully documented and can be read to understand:
1. How to structure a plugin in that language
2. How to use the language-specific SDK
3. Best practices for that language ecosystem

### As Starting Templates

Copy any example plugin as a starting point:

```bash
# Copy the Go rate-limit example
cp -r sample-plugins/rate-limit my-new-plugin
cd my-new-plugin
# Modify to implement your logic
```

### As Testing Examples

Use these plugins with your `mcpd` server to verify:
- Plugin discovery and loading
- Lifecycle management
- Request processing
- Error handling

## Plugin Development Guide

For detailed information on developing plugins, see [docs/PLUGIN_DEVELOPMENT.md](docs/PLUGIN_DEVELOPMENT.md).

Key topics covered:
- Plugin lifecycle in detail
- Request/response processing
- Configuration management
- Error handling patterns
- Testing strategies
- Deployment considerations

## Project Structure

```
plugins-demo/
├── sample-plugins/               # Plugin examples
│   ├── rate-limit/              # Go: Rate limiting
│   ├── tool-audit/              # Go: Audit logging
│   ├── header-transformer/      # Go: Header manipulation
│   ├── prompt-guard/            # C#/.NET: Content filtering
│   └── header-injector/         # Python: Reference implementation
├── bin/                         # Build output (gitignored)
│   └── sample-plugins/          # Built plugin binaries
├── docs/
│   └── PLUGIN_DEVELOPMENT.md    # Comprehensive plugin development guide
├── Makefile                     # Build automation
└── README.md                    # This file
```

## Contributing

When adding new example plugins:

1. Create a new directory in `sample-plugins/`
2. Include a README.md explaining what the plugin demonstrates
3. Add appropriate build commands to the Makefile
4. Ensure the plugin builds cleanly and demonstrates best practices
5. Document any language-specific requirements

## License

Apache 2.0 - See [LICENSE](LICENSE) file for details.

## Related Projects

- **`mcpd` Server:** [mozilla-ai/mcpd](https://github.com/mozilla-ai/mcpd) - The plugin host/runner
- **Go SDK:** [mozilla-ai/mcpd-plugins-sdk-go](https://github.com/mozilla-ai/mcpd-plugins-sdk-go)
- **.NET SDK:** [mozilla-ai/mcpd-plugins-sdk-dotnet](https://github.com/mozilla-ai/mcpd-plugins-sdk-dotnet)
- **Proto Definitions:** [mozilla-ai/mcpd-proto](https://github.com/mozilla-ai/mcpd-proto)
