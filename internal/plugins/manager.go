package plugins

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	pluginv1 "github.com/mozilla-ai/mcpd-plugins-sdk-go/pkg/plugins/v1/plugins"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"

	pkg "github.com/peteski22/plugins-demo/pkg/contract/plugin"
)

// Manager manages plugin processes. It starts plugins, maintains process control,
// and can force-kill them at any time. Plugins are untrusted third-party code.
type Manager struct {
	logger       hclog.Logger
	mu           sync.Mutex
	plugins      map[string]*runningPlugin
	startTimeout time.Duration
	callTimeout  time.Duration
}

// runningPlugin tracks a plugin process and its gRPC connection.
type runningPlugin struct {
	cmd      *exec.Cmd
	conn     *grpc.ClientConn
	client   pluginv1.PluginClient
	instance *PluginInstance
	address  string
	network  string
}

// NewManager creates a new plugin manager.
func NewManager(logger hclog.Logger) *Manager {
	return &Manager{
		logger:       logger.Named("plugin-manager"),
		plugins:      make(map[string]*runningPlugin),
		startTimeout: 10 * time.Second,
		callTimeout:  5 * time.Second,
	}
}

// Start launches a plugin binary, connects to it, and returns a PluginInstance.
// The manager maintains control of the process and can kill it at any time.
func (m *Manager) Start(ctx context.Context, binaryPath string) (*PluginInstance, error) {
	m.logger.Info("starting plugin", "path", binaryPath)

	address, network := m.generateAddress(filepath.Base(binaryPath))
	m.logger.Debug("transport selected", "network", network, "address", address)

	cmd := exec.CommandContext(ctx, binaryPath, "--address", address, "--network", network)
	cmd.Stdout = m.logger.StandardWriter(&hclog.StandardLoggerOptions{InferLevels: true})

	// Temporary debugging for C# plugin.
	if strings.Contains(binaryPath, "prompt-guard") {
		debugFile, err := os.Create("/tmp/prompt-guard-debug.log")
		if err == nil {
			cmd.Stderr = debugFile
			defer func() {
				if closeErr := debugFile.Close(); closeErr != nil {
					m.logger.Warn("failed to close debug file", "error", closeErr)
				}
			}()
		} else {
			cmd.Stderr = m.logger.StandardWriter(&hclog.StandardLoggerOptions{InferLevels: true})
		}
	} else {
		cmd.Stderr = m.logger.StandardWriter(&hclog.StandardLoggerOptions{InferLevels: true})
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start process: %w", err)
	}

	m.logger.Debug("plugin process started", "pid", cmd.Process.Pid, "address", address)

	dialCtx, cancel := context.WithTimeout(ctx, m.startTimeout)
	defer cancel()

	var dialAddr string
	if network == "unix" {
		dialAddr = "unix://" + address
	} else {
		dialAddr = address
	}

	if err := m.waitForSocket(dialCtx, network, address); err != nil {
		if killErr := cmd.Process.Kill(); killErr != nil {
			m.logger.Warn("failed to kill plugin process", "error", killErr)
		}
		return nil, fmt.Errorf("plugin didn't start in time: %w", err)
	}

	conn, err := grpc.NewClient(dialAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		if killErr := cmd.Process.Kill(); killErr != nil {
			m.logger.Warn("failed to kill plugin process", "error", killErr)
		}
		return nil, fmt.Errorf("failed to connect to plugin: %w", err)
	}

	client := pluginv1.NewPluginClient(conn)

	metaCtx, metaCancel := context.WithTimeout(ctx, m.callTimeout)
	defer metaCancel()

	metadata, err := client.GetMetadata(metaCtx, &emptypb.Empty{})
	if err != nil {
		if closeErr := conn.Close(); closeErr != nil {
			m.logger.Warn("failed to close connection", "error", closeErr)
		}
		if killErr := cmd.Process.Kill(); killErr != nil {
			m.logger.Warn("failed to kill plugin process", "error", killErr)
		}
		return nil, fmt.Errorf("failed to get metadata: %w", err)
	}

	m.logger.Info("plugin started",
		"name", metadata.Name,
		"version", metadata.Version,
		"pid", cmd.Process.Pid)

	adapter, err := NewGRPCPluginAdapter(client)
	if err != nil {
		// Clean up plugin process.
		_ = cmd.Process.Kill()
		_ = conn.Close()
		return nil, fmt.Errorf("creating adapter: %w", err)
	}
	instance := &PluginInstance{
		Plugin:   adapter,
		id:       metadata.Name,
		config:   pkg.PluginConfig{},
		required: false,
	}

	rp := &runningPlugin{
		cmd:      cmd,
		conn:     conn,
		client:   client,
		instance: instance,
		address:  address,
		network:  network,
	}

	m.mu.Lock()
	m.plugins[metadata.Name] = rp
	m.mu.Unlock()

	return instance, nil
}

// Plugins returns all started plugin instances.
func (m *Manager) Plugins() []*PluginInstance {
	m.mu.Lock()
	defer m.mu.Unlock()

	instances := make([]*PluginInstance, 0, len(m.plugins))
	for _, rp := range m.plugins {
		instances = append(instances, rp.instance)
	}
	return instances
}

// StopAll stops all running plugins. Force-kills any that don't stop gracefully.
func (m *Manager) StopAll(ctx context.Context) error {
	m.mu.Lock()
	plugins := make([]*runningPlugin, 0, len(m.plugins))
	for _, rp := range m.plugins {
		plugins = append(plugins, rp)
	}
	m.plugins = make(map[string]*runningPlugin)
	m.mu.Unlock()

	for _, rp := range plugins {
		if err := m.stopPlugin(ctx, rp); err != nil {
			m.logger.Error("error stopping plugin", "error", err)
		}
	}

	return nil
}

func (m *Manager) stopPlugin(ctx context.Context, rp *runningPlugin) error {
	m.logger.Info("stopping plugin", "instance", rp.instance.ID())

	stopCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if _, err := rp.client.Stop(stopCtx, &emptypb.Empty{}); err != nil {
		m.logger.Warn("graceful stop failed, force killing", "error", err)
	}

	if err := rp.conn.Close(); err != nil {
		m.logger.Warn("error closing connection", "error", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- rp.cmd.Wait()
	}()

	select {
	case <-time.After(2 * time.Second):
		m.logger.Warn("plugin didn't exit, force killing", "instance", rp.instance.ID())
		if err := rp.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to kill process: %w", err)
		}
		<-done
	case err := <-done:
		if err != nil {
			m.logger.Debug("plugin process exited with error", "error", err)
		}
	}

	if rp.network == "unix" {
		if err := os.Remove(rp.address); err != nil && !os.IsNotExist(err) {
			m.logger.Warn("failed to remove socket file", "path", rp.address, "error", err)
		}
	}

	m.logger.Info("plugin stopped", "instance", rp.instance.ID())
	return nil
}

func (m *Manager) generateAddress(pluginName string) (address string, network string) {
	switch runtime.GOOS {
	case "windows":
		port := 50000 + (time.Now().UnixNano() % 10000)
		return fmt.Sprintf("localhost:%d", port), "tcp"
	default:
		sockPath := filepath.Join(os.TempDir(), fmt.Sprintf("plugin-%s-%d.sock",
			strings.ReplaceAll(pluginName, " ", "-"),
			time.Now().UnixNano()%1000000))
		return sockPath, "unix"
	}
}

func (m *Manager) waitForSocket(ctx context.Context, network, address string) error {
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			conn, err := net.DialTimeout(network, address, 100*time.Millisecond)
			if err == nil {
				_ = conn.Close()
				return nil
			}
		}
	}
}
