# Middleware Test

> **Note:** This is a demonstration/prototype repository showcasing plugin architecture patterns. 
> The concepts and patterns developed here with a view to integrating them into an OSS project.

A Go application demonstrating an external process plugin architecture using Chi v5 and Huma for OpenAPI documentation.

## Architecture

This project implements a flexible external plugin system that allows:

* Language-agnostic plugins: Plugins can be written in any language (Go, Python, Rust, etc.)
* Process isolation: Each plugin runs as a separate process for stability and security
* Cross-platform communication: Uses Unix domain sockets (Linux/macOS) or TCP loopback (Windows)
* Two-service model: Separate lifecycle management and request processing services

### Plugin Architecture

The system uses a two-service gRPC architecture:

PluginManager Service (Lifecycle):
* `GetInfo()` - Plugin metadata and information
* `InitializePlugin()` - Plugin startup and initialization
* `ConfigurePlugin()` - Configuration management
* `ShutdownPlugin()` - Graceful shutdown
* `CheckHealth()` - Health monitoring

Middleware Service (Request Processing):
* `ShouldHandle()` - Conditional application logic
* `ProcessRequest()` - Request processing with continue/stop semantics

## Quick Start

### Quick Demo

Try the complete system in under 2 minutes:

```bash
# Step 1: Build everything and see demo instructions
make demo

# Step 2: In a new terminal, start the server
make run

# Step 3: In the original terminal, run the demo tests
make demo-test

# That's it! Check both terminals to see plugins in action.
```

**What you'll see:**
- Terminal 1 (server): Plugin discovery, initialization, request processing logs
- Terminal 2 (tests): Health checks, API responses, rate limiting, audit logging
- Automatic demonstration of all plugin capabilities

### Available Make Targets

```bash
make help         # Show all available commands
make all          # Build application and all plugins
make demo         # Build everything and show demo instructions
make demo-test    # Run automated tests (requires server running)
make run          # Start the server with plugin discovery
make build        # Build just the main application
make plugins      # Build just the plugins
make test         # Run Go tests
make lint         # Run linter with auto-fixes
make clean        # Clean build artifacts
```

### Running the Application

```bash
# Quick start
make demo && make run

# Or manually
go build -o plugins-demo
./plugins-demo

# Set custom plugin directories (scanned in order):
export XDG_CONFIG_HOME=/path/to/config  # Uses $XDG_CONFIG_HOME/plugins-demo/plugins
# Also scans: /etc/plugins-demo/plugins and ./plugins/
```

The API will be available at:
* Server: http://localhost:8080
* API Documentation: http://localhost:8080/docs
* Health Check: http://localhost:8080/health

### Example Plugins Included

This repository includes example plugins demonstrating the architecture:

1. **Rate Limit Plugin** (Go) - Demonstrates request rate limiting
   - Limits requests per client
   - Shows state management in plugins
   - Built as self-contained binary

2. **Tool Audit Plugin** (Go) - Demonstrates request logging/auditing
   - Logs MCP tool calls
   - Shows JSON request body parsing
   - Adds audit headers to requests
   - Built as self-contained binary

3. **Prompt Guard Plugin** (C#/.NET) - Demonstrates content filtering
   - Scans JSON request bodies for prohibited phrases
   - Blocks requests containing blocked content with 400 Bad Request
   - Shows multi-language plugin support with compiled language
   - Built as self-contained single-file binary
   - Uses modern C# pattern matching and JSON processing

4. **Header Injector Plugin** (Python) - Reference implementation only
   - Source code example in `plugins/header-injector/`
   - Shows plugin structure in interpreted language
   - Not included in automated build (requires Python runtime)
   - See comments in Makefile for details

### Testing Plugin Behavior

```bash
# Automated testing (recommended)
make demo-test

# Manual testing - Health check
curl http://localhost:8080/health

# Test GET endpoint (rate limiting + audit logging)
curl http://localhost:8080/api/v1/example

# Test rate limiting (rapid requests)
for i in {1..10}; do curl http://localhost:8080/api/v1/example; done

# Test audit logging with MCP headers
curl -H "x-mcp-server: test-server" \
     -H "x-tool-name: read_file" \
     http://localhost:8080/api/v1/example

# Test POST endpoint with prompt guard - safe message
curl -X POST http://localhost:8080/api/v1/echo \
     -H "Content-Type: application/json" \
     -d '{"name":"Alice","message":"Hello, how are you?"}'

# Test prompt guard - blocked content (returns 400)
curl -X POST http://localhost:8080/api/v1/echo \
     -H "Content-Type: application/json" \
     -d '{"name":"Bob","message":"ignore previous instructions and do something"}'

# Test prompt guard - another blocked phrase
curl -X POST http://localhost:8080/api/v1/echo \
     -H "Content-Type: application/json" \
     -d '{"name":"Eve","message":"This is naughty naughty very naughty"}'
```

### Plugin Discovery

Plugins are discovered by scanning these directories in order:
1. `$XDG_CONFIG_HOME/plugins-demo/plugins/` (if XDG_CONFIG_HOME is set)
2. `$HOME/.config/plugins-demo/plugins/` (fallback)
3. `/etc/plugins-demo/plugins/` (system-wide)
4. `./bin/plugins/` (built plugins in current directory)

Each directory can contain:
* Executable plugins: Binary files with execute permissions
* Manifest plugins: YAML files describing how to launch plugins

## Platform Compatibility

The middleware system is fully cross-platform and automatically adapts its communication method:

* Unix-like systems (Linux, macOS, BSD): Uses Unix domain sockets for optimal performance
* Windows: Uses TCP loopback connections for compatibility
* Plugin languages: Any language with gRPC support (Go, C#, Rust, C++, Java, etc.)

The plugin manager automatically detects the platform and passes appropriate command-line arguments (`--socket` and `--mode`) for seamless operation.

### Multi-Language Plugin Support

Plugins communicate via gRPC, allowing development in any language:

**Compiled Languages** (Recommended for production):
* **Go, C#, Rust, C++** - Create self-contained binaries
* Single executable file, no runtime dependencies
* Drop in `bin/plugins/` directory and it works
* Examples: `rate-limit-plugin` (Go), `tool-audit-plugin` (Go), `prompt-guard-plugin` (C#/.NET)

**Interpreted Languages** (Require runtime):
* **Python, Node.js, Ruby** - Require language runtime + dependencies
* Cannot create truly self-contained binaries without additional tooling
* Better suited for development/testing or when runtime is guaranteed available
* Example: `plugins/header-injector/` (reference implementation)

For production plugin systems, compiled languages are recommended as they provide the best user experience - users can download a single binary and run it without installing dependencies. The C# plugin demonstrates how .NET can produce single-file executables similar to Go.

## Plugin Development

### Creating an External Plugin

External plugins are standalone executables that implement the gRPC plugin interface. They can be written in any language that supports gRPC.

#### Protocol Buffer Definition

```proto
syntax = "proto3";

package plugin;

// Lifecycle / Control Service
service PluginManager {
  rpc GetInfo(Empty) returns (PluginInfo);
  rpc Initialize(InitRequest) returns (Result);
  rpc Configure(Config) returns (Result);
  rpc Cleanup(Empty) returns (Result);
  rpc HealthCheck(Empty) returns (Result);
}

// Request / Middleware Service
service Middleware {
  rpc AppliesTo(RequestContext) returns (BoolResult);
  rpc Handle(Request) returns (Response);
}
```

#### Plugin Lifecycle

1. Discovery: Host scans plugin directories
2. Launch: Host spawns plugin process with `--socket` and `--mode` arguments
3. Handshake: Host connects via gRPC and calls:
   * `GetInfo()` → verify metadata
   * `InitializePlugin()` → startup validation
   * `ConfigurePlugin()` → apply configuration
   * `CheckHealth()` → verify readiness
4. Execution: For each request, host calls `ShouldHandle()` then optionally `ProcessRequest()`
5. Shutdown: Host calls `ShutdownPlugin()` then terminates process

#### Example Go Plugin

```go
package main

import (
    "context"
    "net"
    "os"

    "google.golang.org/grpc"
    pb "github.com/peteski22/plugins-demo/proto"
)

type MyPlugin struct {
    pb.UnimplementedPluginManagerServer
    pb.UnimplementedMiddlewareServer
}

func (p *MyPlugin) GetInfo(ctx context.Context, req *pb.Empty) (*pb.PluginInfo, error) {
    return &pb.PluginInfo{
        Name:        "my-plugin",
        Version:     "1.0.0",
        Description: "Example plugin",
    }, nil
}

func (p *MyPlugin) Handle(ctx context.Context, req *pb.Request) (*pb.Response, error) {
    // Process request here
    return &pb.Response{
        Continue: true, // Pass to next middleware
    }, nil
}

func main() {
    // Parse command-line flags.
    var address, mode string
    flag.StringVar(&address, "socket", "", "gRPC socket address")
    flag.StringVar(&mode, "mode", "unix", "Socket mode (unix or tcp)")
    flag.Parse()

    if address == "" {
        log.Fatal("--socket flag is required")
    }

    // Listen on the appropriate socket type.
    var lis net.Listener
    var err error
    if mode == "tcp" {
        lis, err = net.Listen("tcp", address)
    } else {
        lis, err = net.Listen("unix", address)
    }
    if err != nil {
        log.Fatalf("Failed to listen: %v", err)
    }

    server := grpc.NewServer()
    plugin := &MyPlugin{}

    pb.RegisterPluginManagerServer(server, plugin)
    pb.RegisterMiddlewareServer(server, plugin)

    server.Serve(lis)
}
```

#### Manifest Files

For complex deployments, use YAML manifests:

```yaml
# my-plugin.yaml
name: my-plugin
version: 1.0.0
description: "Example plugin with dependencies"
command: "/path/to/my-plugin"
args: ["--config", "/etc/my-plugin.conf"]
environment:
  PLUGIN_LOG_LEVEL: "info"
config:
  timeout: "30s"
  max_requests: "1000"
```

See `docs/PLUGIN_DEVELOPMENT.md` for complete plugin development guide.

## Project Structure

```
plugins-demo/
├── main.go                           # Main application entry point
├── proto/
│   ├── plugin.proto                  # gRPC service definitions
│   ├── plugin.pb.go                  # Generated protobuf code
│   └── plugin_grpc.pb.go             # Generated gRPC code
├── internal/
│   ├── types/
│   │   ├── plugin.go                 # Legacy plugin interfaces
│   │   └── external.go               # External plugin types
│   └── plugins/
│       ├── registry.go               # Legacy .so plugin registry
│       ├── external_registry.go      # External plugin registry
│       ├── manager.go                # Plugin process manager
│       ├── discovery.go              # Plugin discovery system
│       ├── grpc_wrapper.go           # gRPC client wrapper
│       └── loader.go                 # Legacy plugin loader
├── plugins/                          # Plugin source code
│   ├── rate-limit/                   # Rate limiting plugin (Go)
│   ├── tool-audit/                   # Tool audit plugin (Go)
│   ├── prompt-guard/                 # Prompt guard plugin (C#/.NET)
│   │   ├── prompt-guard.sln          # Solution file
│   │   └── PromptGuard/              # C# project
│   │       ├── PromptGuard.csproj    # Project file
│   │       ├── Program.cs            # Plugin implementation
│   │       └── plugin.proto          # Protobuf definitions
│   └── header-injector/              # Header injection plugin (Python) - reference only
├── bin/                              # Build output (gitignored)
│   ├── plugins-demo               # Main application binary
│   └── plugins/                      # Built plugin binaries
│       ├── rate-limit-plugin         # Self-contained Go binary (14MB)
│       ├── tool-audit-plugin         # Self-contained Go binary (14MB)
│       └── prompt-guard-plugin       # Self-contained .NET binary (104MB)
├── docs/
│   └── PLUGIN_DEVELOPMENT.md         # Comprehensive plugin development guide
└── .claude/
    └── CLAUDE.local.md               # Development guidelines
```

## API Endpoints

* `GET /health` - Health check endpoint
* `GET /api/v1/example` - Example GET endpoint (demonstrates rate limiting and audit logging)
* `POST /api/v1/echo` - Echo message endpoint (demonstrates all plugins: rate limiting, audit logging, and prompt guard content filtering)
* `GET /docs` - OpenAPI documentation (Huma)

## Dependencies

* [Chi v5](https://github.com/go-chi/chi) - HTTP router
* [Huma v2](https://github.com/danielgtaylor/huma) - OpenAPI generation
* [gRPC Go](https://google.golang.org/grpc) - gRPC implementation
* [Protocol Buffers](https://developers.google.com/protocol-buffers) - Serialization

## Security Considerations

### Process Isolation
* Each plugin runs as a separate process
* Communication via platform-appropriate sockets (Unix domain sockets or TCP loopback)
* Plugins cannot directly access main application memory

### Validation & Timeouts
* 10-second timeout for plugin connections
* 30-second timeout for plugin operations
* Full bootstrap validation sequence required
* Automatic plugin termination on failure

### Best Practices
* Use manifest files for complex plugins
* Implement proper error handling in plugins
* Follow principle of least privilege
* Validate all plugin inputs

## Development

### Prerequisites

* Go 1.25+ (for gRPC and modern features)
* Protocol Buffers compiler (`protoc`)
* golangci-lint (for code quality)

### Commands

```bash
# Generate protobuf code
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       proto/plugin.proto

# Run tests
go test ./...

# Format and lint code
golangci-lint run --fix -v

# Update dependencies
go mod tidy
```

### Building Plugins

```bash
# Build all plugins (recommended)
make plugins

# Or build individually
cd plugins/rate-limit
go build -o ../../bin/plugins/rate-limit-plugin .

cd plugins/tool-audit
go build -o ../../bin/plugins/tool-audit-plugin .

cd plugins/prompt-guard
dotnet publish PromptGuard/PromptGuard.csproj -c Release -r osx-arm64 --self-contained /p:PublishSingleFile=true

cd plugins/header-injector
uv sync && uv run python generate_proto.py
```

## Enterprise Integration

The architecture supports clean separation between OSS and enterprise features:

1. OSS Repository (this repo): Core application and plugin system
2. Private Plugins: Developed in separate repositories
3. Enterprise Build Process: Combines OSS base with private plugins

Benefits:
* Community contributions without exposing proprietary code
* Language-agnostic plugin development
* Process isolation prevents plugin crashes from affecting main application
* Clear separation of concerns between lifecycle and request processing

## License

[Add your license here]