# Official Middleware Plugins

This directory contains officially supported middleware plugins that can be:
1. **Imported directly** as Go packages
2. **Compiled as plugins** (`.so` files) for dynamic loading

## Available Plugins

### Rate Limiting (`ratelimit/`)

Rate limits requests per IP address with configurable limits and time windows.

**Direct Import:**
```go
import "github.com/peteski22/middleware-test/plugins/ratelimit"

// Use with custom config
plugin := ratelimit.NewWithConfig(ratelimit.Config{
    Limit:  50,
    Window: 30 * time.Second,
})

// Apply to Chi router
router.Use(plugin.Middleware())
```

**As Compiled Plugin:**
```bash
cd plugins/ratelimit
make build    # Creates ratelimit.so
make install  # Installs to /etc/middleware-test/plugins/
```

## Plugin Interface

All plugins implement the `types.Plugin` interface:

```go
type Plugin interface {
    Name() string
    Version() string
    Description() string
    Middleware() func(http.Handler) http.Handler
    Configure(config map[string]interface{}) error
    AppliesTo(clientName string, request *http.Request) bool
    Initialize() error
    Cleanup() error
}
```

## Creating New Plugins

1. **Create package directory**: `plugins/myplugin/`
2. **Implement the interface** in `myplugin.go`
3. **Create plugin wrapper** in `plugin.go` (for .so compilation)
4. **Add tests** in `myplugin_test.go`
5. **Add Makefile** for build automation

See `ratelimit/` as a complete example.

## Testing Plugins

```bash
# Test all plugins
go test ./plugins/...

# Test specific plugin
cd plugins/ratelimit
make test

# Test as package (not compiled plugin)
make test-package
```