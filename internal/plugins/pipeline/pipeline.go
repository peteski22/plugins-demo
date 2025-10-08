package pipeline

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/hashicorp/go-hclog"
	pb "github.com/mozilla-ai/mcpd-plugins-sdk-go/pkg/plugins/v1/plugins"

	"github.com/peteski22/plugins-demo/internal/plugins"
	pkg "github.com/peteski22/plugins-demo/pkg/contract/plugin"
)

type Plugin interface {
	// Metadata() pkg.Metadata
	ID() string
	// Name() string
	HandleRequest(ctx context.Context, req any) (any, error)
	HandleResponse(ctx context.Context, resp any) (any, error)
	CanHandle(f pkg.Flow) bool
	Required() bool
}

// Pipeline hosts registered plugins grouped by category.
// NOTE: Use NewPipeline to create a new Pipeline.
type Pipeline struct {
	mu      sync.RWMutex
	logger  hclog.Logger
	plugins map[pkg.Category][]Plugin
}

// NewPipeline constructs a Pipeline.
func NewPipeline(logger hclog.Logger) *Pipeline {
	return &Pipeline{
		logger:  logger.Named("pipeline"),
		plugins: make(map[pkg.Category][]Plugin),
	}
}

// Register places a plugin into the pipeline according to its Metadata().Category.
// Plugins within a category are executed in registration order (this preserves YAML order).
func (p *Pipeline) Register(cat pkg.Category, pl Plugin) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.plugins[cat] = append(p.plugins[cat], pl)
}

// Run executes the pipeline for a given flow.
// The request is returned if the pipeline completes without errors,
// serial categories allow mutation of the request.
// Errors follow the category policy.
func (p *Pipeline) Run(ctx context.Context, flow pkg.Flow, req any) (any, error) {
	for _, cat := range OrderedCategories {
		props := PropsForCategory(cat)
		instances := p.plugins[cat]

		var active []Plugin
		for _, i := range instances {
			if i.CanHandle(flow) {
				active = append(active, i)
			}
		}

		// Sense check, next category if there are no plugins to run.
		if len(active) == 0 {
			p.logger.Debug("no active plugins", "category", cat)
			continue
		}

		switch props.Mode {
		case pkg.ExecSerial:
			for _, i := range active {
				var resp any
				var err error

				switch flow {
				case pkg.FlowRequest:
					resp, err = i.HandleRequest(ctx, req)
				case pkg.FlowResponse:
					resp, err = i.HandleResponse(ctx, req)
				default:
					err = fmt.Errorf("unknown flow: %v", flow)
				}

				if err == nil {
					// Check if plugin wants to short-circuit (e.g., block request).
					if httpResp, ok := resp.(*pb.HTTPResponse); ok && !httpResp.Continue {
						return httpResp, nil
					}

					// If modification is allowed and plugin provided a modified request, use it.
					if props.CanModify {
						if httpResp, ok := resp.(*pb.HTTPResponse); ok && httpResp.ModifiedRequest != nil {
							req = httpResp.ModifiedRequest
						}
					}
					continue
				}

				switch {
				case i.Required():
					// Required plugin failed, trigger pipeline failure.
					return nil, fmt.Errorf("%w: %w", plugins.ErrRequiredPluginFailed, err)
				case props.CanReject:
					// Allowed to trigger pipeline failure.
					return nil, err
				default:
					// Optional plugin failed, log and continue.
					p.logger.Error(
						"plugin failed to handle request",
						"flow", flow,
						"category", cat,
						"mode", props.Mode,
						"plugin", i.ID(),
						"err", err,
					)
					continue
				}
			}

		case pkg.ExecParallel:
			// Sanity check, don't allow modification in parallel.
			if props.CanModify {
				return nil, fmt.Errorf(
					"parallel execution and request mutation are mutually exclusive: '%s'",
					cat,
				)
			}

			var wg sync.WaitGroup
			errCh := make(chan error, len(active))

			for _, i := range active {
				wg.Add(1)
				go func(i Plugin, flow pkg.Flow) {
					defer wg.Done()

					var err error

					switch flow {
					case pkg.FlowRequest:
						_, err = i.HandleRequest(ctx, req)
					case pkg.FlowResponse:
						_, err = i.HandleResponse(ctx, req)
					default:
						err = fmt.Errorf("unknown flow: %v", flow)
					}

					if err == nil {
						return
					}

					switch {
					case i.Required():
						// Required plugin failed, trigger pipeline failure.
						errCh <- fmt.Errorf("%w: %w", plugins.ErrRequiredPluginFailed, err)
					case props.CanReject:
						// Allowed to trigger pipeline failure.
						errCh <- err
					default:
						// Optional plugin failed, log and continue.
						p.logger.Error(
							"plugin failed to handle request",
							"flow", flow,
							"category", cat,
							"mode", props.Mode,
							"plugin", i.ID(),
							"err", err,
						)
						return
					}
				}(i, flow)
			}

			wg.Wait()
			close(errCh)

			errs := make([]error, 0, len(errCh))
			for err := range errCh {
				errs = append(errs, err)
			}
			if len(errs) > 0 {
				return nil, errors.Join(errs...)
			}
		default:
			return nil, fmt.Errorf("unsupported execution mode for category '%s': %v", cat, props.Mode)
		}
	}

	return req, nil
}

// RunRequest executes the REQUEST flow through the pipeline.
// Takes an HTTP request and returns an HTTP response.
// If response.Continue == true, the caller should proceed to the next handler.
// If response.Continue == false, the caller should write the response and stop.
func (p *Pipeline) RunRequest(ctx context.Context, req *pb.HTTPRequest) (*pb.HTTPResponse, error) {
	result, err := p.Run(ctx, pkg.FlowRequest, req)
	if err != nil {
		return nil, err
	}

	// Plugins should return HTTPResponse for REQUEST flow
	if resp, ok := result.(*pb.HTTPResponse); ok {
		return resp, nil
	}

	// If no plugin modified/rejected, return Continue=true
	return &pb.HTTPResponse{Continue: true}, nil
}

// RunResponse executes the RESPONSE flow through the pipeline.
// Takes an HTTP response and returns a potentially modified HTTP response.
// Plugins can modify the response body, headers, or status code.
func (p *Pipeline) RunResponse(ctx context.Context, resp *pb.HTTPResponse) (*pb.HTTPResponse, error) {
	result, err := p.Run(ctx, pkg.FlowResponse, resp)
	if err != nil {
		return nil, err
	}

	// Should always return HTTPResponse for RESPONSE flow
	if modifiedResp, ok := result.(*pb.HTTPResponse); ok {
		return modifiedResp, nil
	}

	// Fallback: return original response
	return resp, nil
}
