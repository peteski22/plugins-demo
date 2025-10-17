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

The plugin system uses a single gRPC `Plugin` service with the following methods:

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

For the complete protocol definition, see [mcpd-proto](https://github.com/mozilla-ai/mcpd-proto/blob/main/plugins/v1/plugin.proto).

**For production use, we recommend using the official SDKs:**
- **Go:** [mcpd-plugins-sdk-go](https://github.com/mozilla-ai/mcpd-plugins-sdk-go)
- **.NET:** [mcpd-plugins-sdk-dotnet](https://github.com/mozilla-ai/mcpd-plugins-sdk-dotnet) ([NuGet](https://www.nuget.org/packages/MozillaAI.Mcpd.Plugins.Sdk))

The SDKs provide convenience wrappers, proper error handling, and are kept in sync with the protocol definitions. The code examples below may be outdated.

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
* Arguments: `--address=/path/to/socket.sock --network=unix`
* Benefits: Lower overhead, better security isolation

Windows Systems:
* Uses TCP loopback connections (localhost)
* Arguments: `--address=localhost:PORT --network=tcp`
* Benefits: Cross-platform compatibility, standard networking

The plugin manager automatically:
* Detects the host platform (`runtime.GOOS`)
* Generates appropriate socket addresses
* Passes `--address` and `--network` as command-line arguments
* Handles connection establishment and management

Plugin implementations must support both modes by parsing the `--network` flag.

## Plugin Lifecycle

### 1. Discovery and Launch
* Host application scans plugin directories
* For each plugin, host spawns process with `--address` and `--network` command-line arguments
* Plugin must bind gRPC server to specified address

### 2. Handshake Sequence
1. Connection: Host connects to plugin via gRPC
2. `GetMetadata()`: Host retrieves plugin metadata (name, version, description, commit hash, build date)
3. `GetCapabilities()`: Host queries which flows the plugin supports (FLOW_REQUEST, FLOW_RESPONSE)
4. `Configure()`: Host applies configuration (if any)
5. `CheckHealth()`: Host verifies plugin operational status
6. `CheckReady()`: Host confirms plugin is ready to handle requests

### 3. Request Processing
* For each HTTP request, host may call `HandleRequest()` if plugin declared FLOW_REQUEST capability
* For each HTTP response, host may call `HandleResponse()` if plugin declared FLOW_RESPONSE capability
* Plugin returns HTTPResponse with `continue` field (true = continue to next plugin, false = stop processing)

### 4. Shutdown
* Host calls `Stop()` on each plugin
* Host terminates plugin processes (SIGTERM then SIGKILL)

## Creating a Go Plugin

**Recommended approach:** Use the official [mcpd-plugins-sdk-go](https://github.com/mozilla-ai/mcpd-plugins-sdk-go) SDK.

The SDK provides:
- Pre-built base types and interfaces
- Proper error handling
- Command-line argument parsing for `--address` and `--network`
- Cross-platform socket support
- Up-to-date protocol implementation

**Basic usage:**

```go
import (
    "log"

    pluginv1 "github.com/mozilla-ai/mcpd-plugins-sdk-go/pkg/plugins/v1"
)

func main() {
    if err := pluginv1.Serve(myPlugin); err != nil {
        log.Fatal(err)
    }
}
```

See the SDK repository for complete documentation and examples.

### Alternative: Manual Implementation

If you need to implement without the SDK:

**Step 1: Generate gRPC code from proto definitions**

```bash
mkdir my-plugin
cd my-plugin
go mod init my-plugin

# Clone the proto definitions
git clone https://github.com/mozilla-ai/mcpd-proto.git

# Generate Go code
protoc --go_out=. --go_opt=paths=source_relative \
       --go-grpc_out=. --go-grpc_opt=paths=source_relative \
       mcpd-proto/plugins/v1/plugin.proto
```

**Step 2: Implement the `Plugin` service interface**

You'll need to implement all 8 methods:
- `GetMetadata()`, `GetCapabilities()`
- `Configure()`, `Stop()`
- `CheckHealth()`, `CheckReady()`
- `HandleRequest()`, `HandleResponse()`

Parse `--address` and `--network` command-line arguments, create appropriate listener (unix domain socket or TCP), and register your gRPC server.

For complete implementation details, refer to the [mcpd-proto plugin.proto](https://github.com/mozilla-ai/mcpd-proto/blob/main/plugins/v1/plugin.proto) and see the working examples in this repository's `sample-plugins/` directory.

## Creating Plugins in Other Languages

For languages without an official SDK (Python, Rust, Node.js, etc.):

1. Generate gRPC code from [mcpd-proto](https://github.com/mozilla-ai/mcpd-proto/blob/main/plugins/v1/plugin.proto)
2. Implement the `Plugin` service interface (all 8 methods)
3. Parse `--address` and `--network` command-line arguments
4. Package as an executable binary

**Note:** Interpreted languages (Python, Node.js, Ruby) require additional packaging tools like PyInstaller, pkg, or similar to create standalone executable binaries. See `sample-plugins/header-injector/` for a Python reference implementation.

## Working Examples

See the `sample-plugins/` directory in this repository for complete, working examples:
- `rate-limit/` - Go plugin demonstrating rate limiting
- `tool-audit/` - Go plugin for audit logging
- `header-transformer/` - Go plugin for header manipulation
- `prompt-guard/` - C#/.NET plugin for content filtering
- `header-injector/` - Python reference implementation (Needs additional work to build executable binary)

## Plugin Deployment

Plugins must be executable binaries. When they are configured, `mcpd` discovers plugins by their binary name in the configured plugin directory.

**Example:** If configured as `prompt-guard`, the binary must be named `prompt-guard` and be executable.

```bash
# Make plugin executable
chmod +x prompt-guard

# Place in configured plugin directory
cp prompt-guard /path/to/plugins/
```

## Testing Your Plugin

### Integration Testing

```bash
# Start plugin manually for testing
./my-plugin --address=/tmp/test-plugin.sock --network=unix &

# Test with grpcurl
grpcurl -plaintext -unix /tmp/test-plugin.sock plugin.Plugin/GetMetadata
```

For unit testing examples, see the test files in the `sample-plugins/` directories.

## Best Practices

### Error Handling
* Always return meaningful error messages
* Use proper gRPC status codes
* Log errors for debugging

### Performance
* Keep `HandleRequest()` and `HandleResponse()` fast - they're called for every request/response
* Cache expensive computations
* Use connection pooling for external services

### Security
* Validate all inputs
* Don't log sensitive information
* Use principle of least privilege

### Configuration
* Support both environment variables and config files
* Provide sensible defaults
* Validate configuration on startup (and when receiving config via `Configure()`)

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
grpcurl -plaintext -unix /path/to/socket plugin.Plugin/GetMetadata

# Monitor plugin logs
tail -f /var/log/my-plugin.log

# Check process status
ps aux | grep my-plugin
```

