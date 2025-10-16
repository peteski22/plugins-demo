# Plugin Development Guide

> **Note:** This guide provides examples to demonstrate plugin development for [mcpd](https://github.com/mozilla-ai/mcpd/pull/209).
> These examples show how to build plugins using language-specific SDKs for the `mcpd` plugin system.

## Overview

This guide covers how to develop external middleware plugins for `mcpd` using the gRPC-based external process architecture.

Plugins are standalone executables that communicate with the main application via gRPC using cross-platform transport (Unix domain sockets on Unix-like systems, TCP loopback on Windows). They can be written in any language that supports gRPC and can be executed from their own binary.

### Official Resources

**Protocol Definitions:**
- [mcpd-proto](https://github.com/mozilla-ai/mcpd-proto) - Official protobuf definitions for the `mcpd` plugin protocol

**Language SDKs:**
- [mcpd-plugins-sdk-go](https://github.com/mozilla-ai/mcpd-plugins-sdk-go) - Go SDK for building `mcpd` plugins
- [mcpd-plugins-sdk-dotnet](https://github.com/mozilla-ai/mcpd-plugins-sdk-dotnet) - .NET SDK for building `mcpd` plugins ([NuGet package](https://www.nuget.org/packages/MozillaAI.Mcpd.Plugins.Sdk))

For languages without an official SDK, use the protobuf definitions from [mcpd-proto](https://github.com/mozilla-ai/mcpd-proto) to generate gRPC code for your language.

## Architecture: External Process Plugins

### Plugin Service Interface

The plugin system uses a single gRPC service with the following methods:

**Lifecycle:**
* `Configure(context.Context, *PluginConfig)` - Initialize plugin with host-provided settings
* `Stop(context.Context, *emptypb.Empty)` - Gracefully shut down the plugin

**Identity and Capabilities:**
* `GetMetadata(context.Context, *emptypb.Empty)` - Returns plugin name, version, description, commit hash, and build date
* `GetCapabilities(context.Context, *emptypb.Empty)` - Declares supported request/response flows (FLOW_REQUEST, FLOW_RESPONSE)

**Health / Readiness:**
* `CheckHealth(context.Context, *emptypb.Empty)` - Verifies plugin operational status (returns error via gRPC status if unhealthy)
* `CheckReady(context.Context, *emptypb.Empty)` - Confirms readiness to handle requests (returns error via gRPC status if not ready)

**Request / Response Handling:**
* `HandleRequest(context.Context, *HTTPRequest)` - Processes incoming HTTP requests
* `HandleResponse(context.Context, *HTTPResponse)` - Processes outgoing HTTP responses

### Benefits of External Processes

1. Language Independence - Write plugins in Go, Python, Rust, Node.js, etc.
2. Process Isolation - Plugin crashes don't affect the main application
3. Memory Isolation - Plugins cannot access main application memory
4. Security - Each plugin runs as a separate process
5. Stability - Plugin failures are contained and recoverable

### Cross-Platform Communication

The plugin system automatically selects the optimal communication method based on the host platform:

Unix-like Systems (Linux, macOS, BSD):
* Uses Unix domain sockets for communication
* Arguments: `--socket=/path/to/socket.sock --mode=unix`
* Benefits: Lower overhead, better security isolation

Windows Systems:
* Uses TCP loopback connections (localhost)
* Arguments: `--socket=localhost:PORT --mode=tcp`
* Benefits: Cross-platform compatibility, standard networking

The plugin manager automatically:
* Detects the host platform (`runtime.GOOS`)
* Generates appropriate socket addresses
* Passes `--socket` and `--mode` as command-line arguments
* Handles connection establishment and management

Plugin implementations must support both modes by parsing the `--mode` flag.

## Plugin Protocol Definition

For the complete and authoritative plugin protocol definition, see the [mcpd-proto repository](https://github.com/mozilla-ai/mcpd-proto/blob/main/plugins/v1/plugin.proto).

The protocol defines a single `Plugin` service with methods for:
- Lifecycle management (Configure, Stop)
- Plugin identity and capabilities (GetMetadata, GetCapabilities)
- Health checks (CheckHealth, CheckReady)
- HTTP request/response processing (HandleRequest, HandleResponse)

**Note:** The examples below may be outdated. For production use, we recommend using the official SDKs:
- **Go:** [mcpd-plugins-sdk-go](https://github.com/mozilla-ai/mcpd-plugins-sdk-go)
- **.NET:** [mcpd-plugins-sdk-dotnet](https://github.com/mozilla-ai/mcpd-plugins-sdk-dotnet) ([NuGet](https://www.nuget.org/packages/MozillaAI.Mcpd.Plugins.Sdk))

The SDKs provide convenience wrappers, proper error handling, and are kept in sync with the protocol definitions.

## Plugin Lifecycle

### 1. Discovery and Launch
* Host application scans plugin directories
* For each plugin, host spawns process with `PLUGIN_SOCKET` environment variable
* Plugin must bind gRPC server to specified Unix socket

### 2. Handshake Sequence
1. Connection: Host connects to plugin via gRPC (10s timeout)
2. `GetInfo()`: Host retrieves plugin metadata
3. `Initialize()`: Host initializes plugin with environment
4. `Configure()`: Host applies configuration (if any)
5. `HealthCheck()`: Host verifies plugin is ready
6. Test Validation: Host sends dummy request to verify `AppliesTo()` and `Handle()`

### 3. Request Processing
* For each HTTP request, host calls `AppliesTo()`
* If plugin applies, host calls `Handle()`
* Plugin returns response with continue/stop decision

### 4. Shutdown
* Host calls `Cleanup()` on each plugin
* Host terminates plugin processes (SIGTERM then SIGKILL)

## Creating a Go Plugin

### Step 1: Project Setup

```bash
mkdir my-plugin
cd my-plugin
go mod init my-plugin

# Copy protobuf definitions from main project
cp /path/to/middleware-test/proto/plugin.proto .

# Generate Go code
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       plugin.proto
```

### Step 2: Implement Plugin Services

```go
// main.go
package main

import (
    "context"
    "fmt"
    "log"
    "net"
    "os"
    "strings"

    "google.golang.org/grpc"
    pb "your-module/plugin" // Adjust import path
)

type MyPlugin struct {
    pb.UnimplementedPluginManagerServer
    pb.UnimplementedMiddlewareServer

    // Plugin state
    config map[string]string
    initialized bool
}

// PluginManager Service Implementation

func (p *MyPlugin) GetInfo(ctx context.Context, req *pb.Empty) (*pb.PluginInfo, error) {
    return &pb.PluginInfo{
        Name:        "my-awesome-plugin",
        Version:     "1.0.0",
        Description: "An example middleware plugin",
    }, nil
}

func (p *MyPlugin) InitializePlugin(ctx context.Context, req *pb.InitRequest) (*pb.Result, error) {
    p.initialized = true
    log.Println("Plugin initialized with env:", req.Env)

    return &pb.Result{
        Success: true,
        Message: "Plugin initialized successfully",
    }, nil
}

func (p *MyPlugin) ConfigurePlugin(ctx context.Context, req *pb.Config) (*pb.Result, error) {
    p.config = req.Values
    log.Println("Plugin configured with:", p.config)

    return &pb.Result{
        Success: true,
        Message: "Configuration applied",
    }, nil
}

func (p *MyPlugin) Cleanup(ctx context.Context, req *pb.Empty) (*pb.Result, error) {
    log.Println("Plugin cleaning up...")
    p.initialized = false

    return &pb.Result{
        Success: true,
        Message: "Cleanup completed",
    }, nil
}

func (p *MyPlugin) CheckHealth(ctx context.Context, req *pb.Empty) (*pb.Result, error) {
    if !p.initialized {
        return &pb.Result{
            Success: false,
            Message: "Plugin not initialized",
        }, nil
    }

    return &pb.Result{
        Success: true,
        Message: "Plugin is healthy",
    }, nil
}

// Middleware Service Implementation

func (p *MyPlugin) ShouldHandle(ctx context.Context, req *pb.RequestContext) (*pb.BoolResult, error) {
    // Example: Apply to all paths except /health
    applies := req.Path != "/health"

    log.Printf("AppliesTo called for %s %s: %v", req.Method, req.Path, applies)

    return &pb.BoolResult{
        Value: applies,
    }, nil
}

func (p *MyPlugin) Handle(ctx context.Context, req *pb.Request) (*pb.Response, error) {
    log.Printf("Handling request: %s %s", req.Method, req.Path)

    // Example middleware logic: Add custom header
    headers := make(map[string]string)
    headers["X-Plugin-Processed"] = "my-awesome-plugin"

    // Copy existing headers
    for k, v := range req.Headers {
        headers[k] = v
    }

    // Example: Block requests with certain patterns
    if strings.Contains(req.Path, "/admin") {
        return &pb.Response{
            Continue:   false, // Stop processing
            StatusCode: 403,
            Headers:    map[string]string{"Content-Type": "text/plain"},
            Body:       []byte("Access denied by plugin"),
        }, nil
    }

    // Continue to next middleware
    return &pb.Response{
        Continue: true,
        Headers:  headers,
    }, nil
}

func main() {
    // Get socket address from environment
    address := os.Getenv("PLUGIN_SOCKET")
    if address == "" {
        log.Fatal("PLUGIN_SOCKET environment variable not set")
    }

    // Get connection mode (unix or tcp)
    mode := os.Getenv("PLUGIN_MODE")
    if mode == "" {
        mode = "unix" // Default to unix socket
    }

    // Listen on appropriate socket type
    var lis net.Listener
    var err error
    switch mode {
    case "unix":
        lis, err = net.Listen("unix", address)
        if err != nil {
            log.Fatalf("Failed to listen on Unix socket %s: %v", address, err)
        }
        defer os.Remove(address) // Cleanup socket file
    case "tcp":
        lis, err = net.Listen("tcp", address)
        if err != nil {
            log.Fatalf("Failed to listen on TCP address %s: %v", address, err)
        }
    default:
        log.Fatalf("Unsupported PLUGIN_MODE: %s (use 'unix' or 'tcp')", mode)
    }

    // Create gRPC server
    server := grpc.NewServer()
    plugin := &MyPlugin{}

    // Register services
    pb.RegisterPluginManagerServer(server, plugin)
    pb.RegisterMiddlewareServer(server, plugin)

    log.Printf("Plugin server listening on %s (%s)", address, mode)

    // Start serving
    if err := server.Serve(lis); err != nil {
        log.Fatalf("Failed to serve: %v", err)
    }
}
```

### Step 3: Build and Deploy

```bash
# Build the plugin
go build -o my-plugin

# Make executable
chmod +x my-plugin

# Deploy to plugin directory
cp my-plugin /etc/middleware-test/plugins/
# Or local directory for testing
mkdir -p ./plugins
cp my-plugin ./plugins/
```

## Creating a Python Plugin

### Step 1: Setup

```bash
mkdir my-python-plugin
cd my-python-plugin

# Create virtual environment
python -m venv venv
source venv/bin/activate

# Install dependencies
pip install grpcio grpcio-tools

# Generate Python code from protobuf
python -m grpc_tools.protoc --proto_path=. --python_out=. --grpc_python_out=. plugin.proto
```

### Step 2: Implement Plugin

```python
# plugin.py
import grpc
import os
import logging
from concurrent import futures
import plugin_pb2
import plugin_pb2_grpc

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class MyPythonPlugin(plugin_pb2_grpc.PluginManagerServicer, plugin_pb2_grpc.MiddlewareServicer):
    def __init__(self):
        self.config = {}
        self.initialized = False

    # PluginManager Service

    def GetInfo(self, request, context):
        return plugin_pb2.PluginInfo(
            name="my-python-plugin",
            version="1.0.0",
            description="An example Python middleware plugin"
        )

    def Initialize(self, request, context):
        self.initialized = True
        logger.info(f"Plugin initialized with env: {dict(request.env)}")

        return plugin_pb2.Result(
            success=True,
            message="Plugin initialized successfully"
        )

    def Configure(self, request, context):
        self.config = dict(request.values)
        logger.info(f"Plugin configured with: {self.config}")

        return plugin_pb2.Result(
            success=True,
            message="Configuration applied"
        )

    def Cleanup(self, request, context):
        logger.info("Plugin cleaning up...")
        self.initialized = False

        return plugin_pb2.Result(
            success=True,
            message="Cleanup completed"
        )

    def HealthCheck(self, request, context):
        if not self.initialized:
            return plugin_pb2.Result(
                success=False,
                message="Plugin not initialized"
            )

        return plugin_pb2.Result(
            success=True,
            message="Plugin is healthy"
        )

    # Middleware Service

    def AppliesTo(self, request, context):
        # Example: Apply to all paths except /health and /docs
        applies = request.path not in ["/health", "/docs"]

        logger.info(f"AppliesTo called for {request.method} {request.path}: {applies}")

        return plugin_pb2.BoolResult(value=applies)

    def Handle(self, request, context):
        logger.info(f"Handling request: {request.method} {request.path}")

        # Example: Rate limiting logic
        if "rate-limit" in self.config:
            # Implement rate limiting logic here
            pass

        # Add custom header
        headers = dict(request.headers)
        headers["X-Python-Plugin"] = "processed"

        # Continue to next middleware
        return plugin_pb2.Response(
            continue_=True,  # Note: 'continue' is a Python keyword, so use 'continue_'
            headers=headers
        )

def serve():
    address = os.getenv("PLUGIN_SOCKET")
    if not address:
        raise ValueError("PLUGIN_SOCKET environment variable not set")

    mode = os.getenv("PLUGIN_MODE", "unix")  # Default to unix socket

    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    plugin = MyPythonPlugin()

    plugin_pb2_grpc.add_PluginManagerServicer_to_server(plugin, server)
    plugin_pb2_grpc.add_MiddlewareServicer_to_server(plugin, server)

    # Listen on the appropriate socket type
    if mode == "unix":
        server.add_insecure_port(f'unix:{address}')
    elif mode == "tcp":
        server.add_insecure_port(address)
    else:
        raise ValueError(f"Unsupported PLUGIN_MODE: {mode} (use 'unix' or 'tcp')")

    logger.info(f"Plugin server listening on {address} ({mode})")

    server.start()
    server.wait_for_termination()

if __name__ == "__main__":
    serve()
```

### Step 3: Package and Deploy

```bash
# Create executable with PyInstaller
pip install pyinstaller
pyinstaller --onefile plugin.py

# Deploy
cp dist/plugin /etc/middleware-test/plugins/my-python-plugin
chmod +x /etc/middleware-test/plugins/my-python-plugin
```

## Manifest-Based Deployment

For complex plugins or those with dependencies, use manifest files:

### Plugin Manifest Format

```yaml
# my-plugin.yaml
name: my-complex-plugin
version: 2.1.0
description: "A complex plugin with dependencies and configuration"
command: "/path/to/my-plugin"
args: ["--verbose", "--config-file", "/etc/my-plugin.conf"]
environment:
  PLUGIN_LOG_LEVEL: "debug"
  PLUGIN_CACHE_DIR: "/tmp/plugin-cache"
  CUSTOM_VAR: "value"
config:
  timeout: "30s"
  max_requests: "1000"
  feature_flags: "feature1,feature2"
```

### Deploy Manifest

```bash
# Copy manifest to plugin directory
cp my-plugin.yaml /etc/middleware-test/plugins/

# Ensure plugin binary is executable
chmod +x /path/to/my-plugin
```

## Advanced Plugin Patterns

### Rate Limiting Plugin

```go
func (p *RateLimitPlugin) Handle(ctx context.Context, req *pb.Request) (*pb.Response, error) {
    clientIP := req.Headers["X-Forwarded-For"]
    if clientIP == "" {
        clientIP = req.Headers["X-Real-IP"]
    }

    if p.isRateLimited(clientIP) {
        return &pb.Response{
            Continue:   false,
            StatusCode: 429,
            Headers: map[string]string{
                "Content-Type": "application/json",
                "Retry-After": "60",
            },
            Body: []byte(`{"error": "Rate limit exceeded"}`),
        }, nil
    }

    return &pb.Response{Continue: true}, nil
}
```

### Authentication Plugin

```go
func (p *AuthPlugin) Handle(ctx context.Context, req *pb.Request) (*pb.Response, error) {
    authHeader := req.Headers["Authorization"]
    if authHeader == "" {
        return &pb.Response{
            Continue:   false,
            StatusCode: 401,
            Headers: map[string]string{
                "Content-Type": "application/json",
                "WWW-Authenticate": "Bearer",
            },
            Body: []byte(`{"error": "Authentication required"}`),
        }, nil
    }

    if !p.validateToken(authHeader) {
        return &pb.Response{
            Continue:   false,
            StatusCode: 403,
            Headers: map[string]string{
                "Content-Type": "application/json",
            },
            Body: []byte(`{"error": "Invalid token"}`),
        }, nil
    }

    // Add user info to headers for downstream
    headers := make(map[string]string)
    for k, v := range req.Headers {
        headers[k] = v
    }
    headers["X-User-ID"] = p.extractUserID(authHeader)

    return &pb.Response{
        Continue: true,
        Headers:  headers,
    }, nil
}
```

## Testing Your Plugin

### Unit Testing

```go
func TestPlugin(t *testing.T) {
    plugin := &MyPlugin{}

    // Test GetInfo
    info, err := plugin.GetInfo(context.Background(), &pb.Empty{})
    require.NoError(t, err)
    assert.Equal(t, "my-plugin", info.Name)

    // Test Initialize
    result, err := plugin.Initialize(context.Background(), &pb.InitRequest{})
    require.NoError(t, err)
    assert.True(t, result.Success)

    // Test AppliesTo
    applies, err := plugin.AppliesTo(context.Background(), &pb.RequestContext{
        Path: "/api/test",
    })
    require.NoError(t, err)
    assert.True(t, applies.Value)
}
```

### Integration Testing

```bash
# Start plugin manually for testing
PLUGIN_SOCKET=/tmp/test-plugin.sock ./my-plugin &

# Test with grpcurl
grpcurl -plaintext -unix /tmp/test-plugin.sock plugin.PluginManager/GetInfo
```

## Best Practices

### Error Handling
* Always return meaningful error messages
* Use proper gRPC status codes
* Log errors for debugging

### Performance
* Keep `AppliesTo()` fast - it's called for every request
* Cache expensive computations
* Use connection pooling for external services

### Security
* Validate all inputs
* Don't log sensitive information
* Use principle of least privilege

### Configuration
* Support both environment variables and config files
* Provide sensible defaults
* Validate configuration on startup

### Logging
* Use structured logging
* Include correlation IDs when available
* Log at appropriate levels

## Debugging Plugins

### Common Issues

1. Socket Permission Errors: Ensure plugin has write access to socket directory
2. Timeout Errors: Plugin taking too long to start or respond
3. gRPC Connection Failures: Check socket path and network configuration
4. Protocol Mismatches: Ensure plugin uses same protobuf definitions

### Debug Tools

```bash
# Check if plugin socket exists
ls -la /tmp/middleware-plugins-*/

# Test plugin manually
grpcurl -plaintext -unix /path/to/socket plugin.PluginManager/GetInfo

# Monitor plugin logs
tail -f /var/log/my-plugin.log

# Check process status
ps aux | grep my-plugin
```

## Migration from .so Plugins

If you have existing Go `.so` plugins, you can wrap them:

```go
type LegacyPluginWrapper struct {
    pb.UnimplementedPluginManagerServer
    pb.UnimplementedMiddlewareServer

    plugin types.Plugin // Your existing plugin
}

func (w *LegacyPluginWrapper) Handle(ctx context.Context, req *pb.Request) (*pb.Response, error) {
    // Convert gRPC request to http.Request
    httpReq := convertToHTTPRequest(req)

    // Create response recorder
    rec := httptest.NewRecorder()

    // Call legacy middleware
    w.plugin.Middleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Continue handler
    })).ServeHTTP(rec, httpReq)

    // Convert response back to gRPC
    return convertToGRPCResponse(rec), nil
}
```

This allows gradual migration from .so to external process plugins.