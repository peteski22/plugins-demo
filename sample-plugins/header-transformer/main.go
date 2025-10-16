package main

import (
	"context"
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/mozilla-ai/mcpd-plugins-sdk-go/pkg/plugins/v1/plugins"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type headerTransformerPlugin struct {
	pb.UnimplementedPluginServer
}

func (p *headerTransformerPlugin) GetMetadata(_ context.Context, _ *emptypb.Empty) (*pb.Metadata, error) {
	return &pb.Metadata{
		Name:        "header-transformer",
		Version:     "1.0.0",
		Description: "Transforms request headers by adding custom headers",
		CommitHash:  "abc123",
	}, nil
}

func (p *headerTransformerPlugin) GetCapabilities(_ context.Context, _ *emptypb.Empty) (*pb.Capabilities, error) {
	return &pb.Capabilities{
		Flows: []pb.Flow{pb.Flow_FLOW_REQUEST},
	}, nil
}

func (p *headerTransformerPlugin) CheckHealth(_ context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (p *headerTransformerPlugin) CheckReady(_ context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (p *headerTransformerPlugin) Configure(_ context.Context, _ *pb.PluginConfig) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (p *headerTransformerPlugin) Stop(_ context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (p *headerTransformerPlugin) HandleRequest(_ context.Context, req *pb.HTTPRequest) (*pb.HTTPResponse, error) {
	// Create a modified request with additional headers.
	modifiedReq := &pb.HTTPRequest{
		Method:     req.Method,
		Url:        req.Url,
		Path:       req.Path,
		Body:       req.Body,
		RemoteAddr: req.RemoteAddr,
		RequestUri: req.RequestUri,
		Headers:    make(map[string]string),
	}

	// Copy existing headers.
	for k, v := range req.Headers {
		modifiedReq.Headers[k] = v
	}

	// Add custom transformation headers.
	modifiedReq.Headers["X-Transformed-By"] = "header-transformer-plugin"
	modifiedReq.Headers["X-Original-Path"] = req.Path

	log.Printf("Transformed request: added headers X-Transformed-By and X-Original-Path")

	// Return response with modified request.
	return &pb.HTTPResponse{
		Continue:        true,
		StatusCode:      0,
		ModifiedRequest: modifiedReq,
	}, nil
}

func (p *headerTransformerPlugin) HandleResponse(_ context.Context, resp *pb.HTTPResponse) (*pb.HTTPResponse, error) {
	// Pass through unchanged.
	return resp, nil
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("")

	var (
		address = flag.String("address", "", "Address to listen on (e.g., /tmp/plugin.sock or localhost:50051)")
		network = flag.String("network", "unix", "Network type: 'unix' or 'tcp'")
	)
	flag.Parse()

	if *address == "" {
		log.Fatal("--address is required")
	}

	// Setup listener.
	var listener net.Listener
	var err error

	if *network == "unix" {
		// Clean up existing socket if it exists.
		_ = os.Remove(*address)
		listener, err = net.Listen("unix", *address)
	} else {
		listener, err = net.Listen("tcp", *address)
	}

	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterPluginServer(grpcServer, &headerTransformerPlugin{})

	// Handle graceful shutdown.
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Shutting down...")
		grpcServer.GracefulStop()
		if *network == "unix" {
			_ = os.Remove(*address)
		}
	}()

	log.Printf("header-transformer plugin listening on %s (%s)", *address, *network)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
