package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	pluginv1 "github.com/mozilla-ai/mcpd-plugins-sdk-go/pkg/plugins/v1/plugins"
	"google.golang.org/protobuf/types/known/emptypb"
)

// RateLimitPlugin implements rate limiting for incoming requests.
type RateLimitPlugin struct {
	pluginv1.BasePlugin

	mu          sync.RWMutex
	requests    map[string]int
	lastReset   time.Time
	maxRequests int
	window      time.Duration
	initialized bool
}

func newRateLimitPlugin() *RateLimitPlugin {
	return &RateLimitPlugin{
		requests:    make(map[string]int),
		lastReset:   time.Now(),
		maxRequests: 100,
		window:      time.Minute,
	}
}

func (p *RateLimitPlugin) GetMetadata(ctx context.Context, _ *emptypb.Empty) (*pluginv1.Metadata, error) {
	return &pluginv1.Metadata{
		Name:        "rate-limit",
		Version:     "2.0.0",
		Description: "Rate limiting middleware plugin",
	}, nil
}

func (p *RateLimitPlugin) GetCapabilities(ctx context.Context, _ *emptypb.Empty) (*pluginv1.Capabilities, error) {
	return &pluginv1.Capabilities{
		Flows: []pluginv1.Flow{pluginv1.FlowRequest},
	}, nil
}

func (p *RateLimitPlugin) Configure(ctx context.Context, cfg *pluginv1.PluginConfig) (*emptypb.Empty, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if maxReqStr, exists := cfg.CustomConfig["max_requests"]; exists {
		if maxReq, err := strconv.Atoi(maxReqStr); err == nil && maxReq > 0 {
			p.maxRequests = maxReq
			log.Printf("Rate limit max_requests configured to: %d", p.maxRequests)
		}
	}

	if windowStr, exists := cfg.CustomConfig["window"]; exists {
		if window, err := time.ParseDuration(windowStr); err == nil && window > 0 {
			p.window = window
			log.Printf("Rate limit window configured to: %v", p.window)
		}
	}

	p.initialized = true
	p.lastReset = time.Now()

	log.Printf("Rate limit plugin initialized with limits: %d requests per %v", p.maxRequests, p.window)

	return &emptypb.Empty{}, nil
}

func (p *RateLimitPlugin) Stop(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	log.Println("Rate limit plugin cleaning up...")

	p.mu.Lock()
	defer p.mu.Unlock()

	p.initialized = false
	p.requests = make(map[string]int)

	return &emptypb.Empty{}, nil
}

func (p *RateLimitPlugin) CheckHealth(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.initialized {
		return nil, fmt.Errorf("rate limit plugin not initialized")
	}

	return &emptypb.Empty{}, nil
}

func (p *RateLimitPlugin) CheckReady(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if !p.initialized {
		return nil, fmt.Errorf("rate limit plugin not ready")
	}

	return &emptypb.Empty{}, nil
}

func (p *RateLimitPlugin) HandleRequest(ctx context.Context, req *pluginv1.HTTPRequest) (*pluginv1.HTTPResponse, error) {
	log.Printf("Rate limit handling request: %s %s", req.Method, req.Path)

	clientID := p.extractClientID(req.Headers)

	if p.isRateLimited(clientID) {
		log.Printf("Rate limit exceeded for client: %s", clientID)

		return &pluginv1.HTTPResponse{
			Continue:   false,
			StatusCode: 429,
			Headers: map[string]string{
				"Content-Type":          "application/json",
				"Retry-After":           "60",
				"X-RateLimit-Limit":     strconv.Itoa(p.maxRequests),
				"X-RateLimit-Remaining": "0",
				"X-RateLimit-Reset":     strconv.FormatInt(p.getResetTime(), 10),
			},
			Body: []byte(`{"error": "Rate limit exceeded", "retry_after": 60}`),
		}, nil
	}

	remaining := p.incrementRequest(clientID)

	headers := make(map[string]string)
	for k, v := range req.Headers {
		headers[k] = v
	}

	headers["X-RateLimit-Limit"] = strconv.Itoa(p.maxRequests)
	headers["X-RateLimit-Remaining"] = strconv.Itoa(remaining)
	headers["X-RateLimit-Reset"] = strconv.FormatInt(p.getResetTime(), 10)

	log.Printf("Rate limit passed for client: %s, remaining: %d", clientID, remaining)

	return &pluginv1.HTTPResponse{
		Continue: true,
		Headers:  headers,
	}, nil
}

// extractClientID extracts client identifier from request headers.
func (p *RateLimitPlugin) extractClientID(headers map[string]string) string {
	if clientIP := headers["X-Forwarded-For"]; clientIP != "" {
		return clientIP
	}

	if clientIP := headers["X-Real-IP"]; clientIP != "" {
		return clientIP
	}

	return "unknown"
}

// isRateLimited checks if client has exceeded rate limit.
func (p *RateLimitPlugin) isRateLimited(clientID string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if time.Since(p.lastReset) >= p.window {
		p.requests = make(map[string]int)
		p.lastReset = time.Now()
	}

	count := p.requests[clientID]
	return count >= p.maxRequests
}

// incrementRequest increments request count for client and returns remaining requests.
func (p *RateLimitPlugin) incrementRequest(clientID string) int {
	p.mu.Lock()
	defer p.mu.Unlock()

	if time.Since(p.lastReset) >= p.window {
		p.requests = make(map[string]int)
		p.lastReset = time.Now()
	}

	p.requests[clientID]++
	return p.maxRequests - p.requests[clientID]
}

// getResetTime returns the timestamp when rate limits will reset.
func (p *RateLimitPlugin) getResetTime() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.lastReset.Add(p.window).Unix()
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("")

	if err := pluginv1.Serve(newRateLimitPlugin()); err != nil {
		log.Fatal(err)
	}
}
