package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	pluginv1 "github.com/mozilla-ai/mcpd-plugins-sdk-go/pkg/plugins/v1/plugins"
	"google.golang.org/protobuf/types/known/emptypb"
)

// ToolAuditPlugin implements auditing for MCP tool calls.
type ToolAuditPlugin struct {
	pluginv1.BasePlugin

	initialized bool
}

func newToolAuditPlugin() *ToolAuditPlugin {
	return &ToolAuditPlugin{}
}

func (p *ToolAuditPlugin) GetMetadata(ctx context.Context, _ *emptypb.Empty) (*pluginv1.Metadata, error) {
	return &pluginv1.Metadata{
		Name:        "tool-audit",
		Version:     "1.0.0",
		Description: "Logs MCP server and tool usage for auditing purposes",
	}, nil
}

func (p *ToolAuditPlugin) GetCapabilities(ctx context.Context, _ *emptypb.Empty) (*pluginv1.Capabilities, error) {
	return &pluginv1.Capabilities{
		Flows: []pluginv1.Flow{pluginv1.FlowRequest},
	}, nil
}

func (p *ToolAuditPlugin) Configure(ctx context.Context, cfg *pluginv1.PluginConfig) (*emptypb.Empty, error) {
	p.initialized = true
	log.Println("Tool audit plugin initialized successfully")
	return &emptypb.Empty{}, nil
}

func (p *ToolAuditPlugin) Stop(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	log.Println("Tool audit plugin cleaning up...")
	p.initialized = false
	return &emptypb.Empty{}, nil
}

func (p *ToolAuditPlugin) CheckHealth(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	if !p.initialized {
		return nil, fmt.Errorf("tool audit plugin not initialized")
	}
	return &emptypb.Empty{}, nil
}

func (p *ToolAuditPlugin) CheckReady(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	if !p.initialized {
		return nil, fmt.Errorf("tool audit plugin not ready")
	}
	return &emptypb.Empty{}, nil
}

func (p *ToolAuditPlugin) HandleRequest(ctx context.Context, req *pluginv1.HTTPRequest) (*pluginv1.HTTPResponse, error) {
	log.Printf("Tool audit handling request: %s %s", req.Method, req.Path)

	auditInfo := p.extractAuditInfo(req)
	p.logToolUsage(auditInfo)

	headers := make(map[string]string)
	for k, v := range req.Headers {
		headers[k] = v
	}

	headers["X-Tool-Audit-ID"] = fmt.Sprintf("audit-%d", time.Now().Unix())
	headers["X-Tool-Audit-Timestamp"] = time.Now().UTC().Format(time.RFC3339)

	return &pluginv1.HTTPResponse{
		Continue: true,
		Headers:  headers,
	}, nil
}

// auditInfo represents extracted audit information.
type auditInfo struct {
	Timestamp   time.Time         `json:"timestamp"`
	Method      string            `json:"method"`
	Path        string            `json:"path"`
	MCPServer   string            `json:"mcp_server,omitempty"`
	ToolName    string            `json:"tool_name,omitempty"`
	UserAgent   string            `json:"user_agent,omitempty"`
	ContentType string            `json:"content_type,omitempty"`
	BodyPreview string            `json:"body_preview,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
}

// extractAuditInfo extracts relevant audit information from the request.
func (p *ToolAuditPlugin) extractAuditInfo(req *pluginv1.HTTPRequest) auditInfo {
	info := auditInfo{
		Timestamp: time.Now().UTC(),
		Method:    req.Method,
		Path:      req.Path,
		Headers:   req.Headers,
	}

	if server := req.Headers["x-mcp-server"]; server != "" {
		info.MCPServer = server
	}

	if tool := req.Headers["x-tool-name"]; tool != "" {
		info.ToolName = tool
	}

	if ua := req.Headers["user-agent"]; ua != "" {
		info.UserAgent = ua
	}

	if ct := req.Headers["content-type"]; ct != "" {
		info.ContentType = ct
	}

	if len(req.Body) > 0 && strings.Contains(info.ContentType, "application/json") {
		info.BodyPreview = p.extractToolFromBody(req.Body)
	}

	return info
}

// extractToolFromBody attempts to extract tool information from JSON body.
func (p *ToolAuditPlugin) extractToolFromBody(body []byte) string {
	var jsonBody map[string]interface{}
	if err := json.Unmarshal(body, &jsonBody); err != nil {
		if len(body) > 100 {
			return string(body[:100]) + "..."
		}
		return string(body)
	}

	if method, ok := jsonBody["method"].(string); ok {
		if params, ok := jsonBody["params"].(map[string]interface{}); ok {
			if toolName, ok := params["name"].(string); ok {
				return fmt.Sprintf("method=%s, tool=%s", method, toolName)
			}
			return fmt.Sprintf("method=%s", method)
		}
		return fmt.Sprintf("method=%s", method)
	}

	if name, ok := jsonBody["name"].(string); ok {
		return fmt.Sprintf("name=%s", name)
	}

	if tool, ok := jsonBody["tool"].(string); ok {
		return fmt.Sprintf("tool=%s", tool)
	}

	bodyStr := string(body)
	if len(bodyStr) > 200 {
		return bodyStr[:200] + "..."
	}
	return bodyStr
}

// logToolUsage logs the tool usage audit information.
func (p *ToolAuditPlugin) logToolUsage(info auditInfo) {
	logEntry := map[string]interface{}{
		"audit_type": "tool_usage",
		"timestamp":  info.Timestamp.Format(time.RFC3339),
		"request": map[string]interface{}{
			"method": info.Method,
			"path":   info.Path,
		},
	}

	if info.MCPServer != "" {
		logEntry["mcp_server"] = info.MCPServer
	}

	if info.ToolName != "" {
		logEntry["tool_name"] = info.ToolName
	}

	if info.UserAgent != "" {
		logEntry["user_agent"] = info.UserAgent
	}

	if info.ContentType != "" {
		logEntry["content_type"] = info.ContentType
	}

	if info.BodyPreview != "" {
		logEntry["body_preview"] = info.BodyPreview
	}

	if jsonLog, err := json.Marshal(logEntry); err == nil {
		log.Printf("AUDIT: %s", string(jsonLog))
	} else {
		log.Printf("AUDIT: %s %s - MCP Server: %s, Tool: %s",
			info.Method, info.Path, info.MCPServer, info.ToolName)
	}
}

func main() {
	if err := pluginv1.Serve(newToolAuditPlugin()); err != nil {
		log.Fatal(err)
	}
}
