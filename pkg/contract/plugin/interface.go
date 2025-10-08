package plugin

import (
	"context"

	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// Plugin defines the contract for in-process plugins.
//
// This is a Go-specific interface for plugins running in the same process as the host.
// For external multi-language plugins, this contract is exposed via gRPC protobuf definitions.
// See proto/plugin.proto for the gRPC equivalent.
type Plugin interface {
	// Metadata returns static information about the plugin's identity and build.
	Metadata() Metadata

	// Capabilities returns which flows (request/response) this plugin supports.
	Capabilities() Capabilities

	// Configure initializes the plugin with the provided application configuration (e.g., telemetry settings).
	// The host calls this once immediately after the gRPC connection is established.
	Configure(ctx context.Context, config PluginConfig) error

	// Stop performs graceful shutdown, cleaning up resources and closing connections.
	Stop(ctx context.Context) error

	// Health checks internal correctness and returns error if the plugin is unhealthy.
	// This verifies the plugin's core functionality is operational.
	Health(ctx context.Context) error

	// Ready returns true when the plugin is initialized and safe to receive traffic.
	// A plugin may be healthy but not yet ready during startup.
	Ready(ctx context.Context) (bool, error)

	// HandleRequest processes an inbound request before it reaches the application.
	// Plugins can reject, modify, or pass through the request.
	HandleRequest(ctx context.Context, req any) (any, error)

	// HandleResponse processes an outbound response after the application responds.
	// Plugins can modify or pass through the response.
	HandleResponse(ctx context.Context, resp any) (any, error)

	// Tracer provides OpenTelemetry tracing for distributed request tracking.
	// The host can collect spans and propagate trace context.
	Tracer() trace.Tracer

	// Meter provides OpenTelemetry metrics for counters, histograms, and gauges.
	// The host can scrape or export these metrics.
	Meter() metric.Meter
}

// Capabilities declares which flows a plugin supports (request, response, or both).
type Capabilities map[Flow]struct{}

// PluginConfig contains host-provided configuration for the plugin.
type PluginConfig struct {
	// Telemetry configures OpenTelemetry exports for traces and metrics.
	Telemetry TelemetryConfig

	// CustomConfig holds plugin-specific configuration from YAML.
	CustomConfig map[string]string
}

// Metadata provides static identity information for a plugin.
type Metadata struct {
	// Name is the unique identifier for this plugin.
	Name string

	// Description provides human-readable information about the plugin's purpose.
	Description string

	// Version is the semantic version of the plugin.
	Version string

	// CommitHash is the git commit hash the plugin was built from.
	CommitHash string

	// BuildDate is the ISO 8601 timestamp when the plugin was compiled.
	BuildDate string
}

// TelemetryConfig provides OpenTelemetry configuration for the plugin.
type TelemetryConfig struct {
	// OTLPEndpoint is the OTLP collector endpoint (e.g., "localhost:4317").
	OTLPEndpoint string

	// ServiceName identifies this plugin in traces and metrics.
	ServiceName string

	// Environment indicates the deployment environment (e.g., "production", "staging").
	Environment string

	// SampleRatio determines the fraction of traces to sample (0.0 to 1.0).
	SampleRatio float64
}
