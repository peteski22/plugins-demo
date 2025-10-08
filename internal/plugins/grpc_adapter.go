package plugins

import (
	"context"
	"fmt"

	pb "github.com/mozilla-ai/mcpd-plugins-sdk-go/pkg/plugins/v1/plugins"
	"go.opentelemetry.io/otel/metric"
	mnop "go.opentelemetry.io/otel/metric/noop"
	"go.opentelemetry.io/otel/trace"
	tnop "go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/protobuf/types/known/emptypb"

	pkg "github.com/peteski22/plugins-demo/pkg/contract/plugin"
)

// Ensure grpcPluginAdapter implements plugins.Plugin
var _ pkg.Plugin = (*grpcPluginAdapter)(nil)

// grpcPluginAdapter adapts a gRPC PluginClient to the internal Plugin interface.
type grpcPluginAdapter struct {
	client       pb.PluginClient
	metadata     *pkg.Metadata
	capabilities pkg.Capabilities
}

func NewGRPCPluginAdapter(client pb.PluginClient) (pkg.Plugin, error) {
	adapter := &grpcPluginAdapter{client: client}
	// Eagerly fetch and cache static metadata and capabilities.
	var err error
	adapter.metadata, err = adapter.fetchMetadata()
	if err != nil {
		return nil, fmt.Errorf("fetching metadata: %w", err)
	}
	adapter.capabilities, err = adapter.fetchCapabilities()
	if err != nil {
		return nil, fmt.Errorf("fetching capabilities: %w", err)
	}
	return adapter, nil
}

func (g *grpcPluginAdapter) fetchMetadata() (*pkg.Metadata, error) {
	resp, err := g.client.GetMetadata(context.Background(), &emptypb.Empty{})
	if err != nil {
		return nil, err
	}
	return &pkg.Metadata{
		Name:        resp.Name,
		Version:     resp.Version,
		Description: resp.Description,
		CommitHash:  resp.CommitHash,
		BuildDate:   resp.BuildDate,
	}, nil
}

func (g *grpcPluginAdapter) fetchCapabilities() (pkg.Capabilities, error) {
	resp, err := g.client.GetCapabilities(context.Background(), &emptypb.Empty{})
	if err != nil {
		return nil, err
	}

	flows := make(map[pkg.Flow]struct{}, len(resp.Flows))
	for _, f := range resp.Flows {
		switch f {
		case pb.Flow_FLOW_REQUEST:
			flows[pkg.FlowRequest] = struct{}{}
		case pb.Flow_FLOW_RESPONSE:
			flows[pkg.FlowResponse] = struct{}{}
		}
	}

	if len(flows) == 0 {
		return nil, fmt.Errorf("plugin returned empty capabilities")
	}

	return flows, nil
}

// Metadata returns plugin static information.
func (g *grpcPluginAdapter) Metadata() pkg.Metadata {
	return *g.metadata
}

// Capabilities returns the plugin static capabilities.
func (g *grpcPluginAdapter) Capabilities() pkg.Capabilities {
	return g.capabilities
}

// Configure forwards PluginConfig to gRPC plugin.
func (g *grpcPluginAdapter) Configure(ctx context.Context, cfg pkg.PluginConfig) error {
	req := &pb.PluginConfig{
		Telemetry: &pb.TelemetryConfig{
			OtlpEndpoint: cfg.Telemetry.OTLPEndpoint,
			ServiceName:  cfg.Telemetry.ServiceName,
			Environment:  cfg.Telemetry.Environment,
			SampleRatio:  cfg.Telemetry.SampleRatio,
		},
		CustomConfig: cfg.CustomConfig,
	}
	_, err := g.client.Configure(ctx, req)
	return err
}

func (g *grpcPluginAdapter) Stop(ctx context.Context) error {
	_, err := g.client.Stop(ctx, &emptypb.Empty{})
	return err
}

func (g *grpcPluginAdapter) Health(ctx context.Context) error {
	_, err := g.client.CheckHealth(ctx, &emptypb.Empty{})
	return err
}

func (g *grpcPluginAdapter) Ready(ctx context.Context) (bool, error) {
	_, err := g.client.CheckReady(ctx, &emptypb.Empty{})
	if err != nil {
		return false, err
	}
	return true, nil
}

func (g *grpcPluginAdapter) HandleRequest(ctx context.Context, req any) (any, error) {
	httpReq, ok := req.(*pb.HTTPRequest)
	if !ok {
		return nil, ErrInvalidRequestType
	}
	resp, err := g.client.HandleRequest(ctx, httpReq)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (g *grpcPluginAdapter) HandleResponse(ctx context.Context, resp any) (any, error) {
	httpResp, ok := resp.(*pb.HTTPResponse)
	if !ok {
		return nil, ErrInvalidResponseType
	}
	out, err := g.client.HandleResponse(ctx, httpResp)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (g *grpcPluginAdapter) Tracer() trace.Tracer {
	return tnop.NewTracerProvider().Tracer("")
}

func (g *grpcPluginAdapter) Meter() metric.Meter {
	return mnop.NewMeterProvider().Meter("")
}
