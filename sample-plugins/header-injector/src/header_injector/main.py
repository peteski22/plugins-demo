#!/usr/bin/env python3
"""
Header Injector Plugin - A simple Python-based middleware plugin example.
"""

import argparse
import grpc
import os
import logging
import sys
from concurrent import futures

# Import generated protobuf modules
try:
    from . import plugin_pb2
    from . import plugin_pb2_grpc
except ImportError:
    print("Error: protobuf files not found. Please run: make plugins")
    sys.exit(1)

logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(name)s - %(levelname)s - %(message)s')
logger = logging.getLogger('header-injector')


class HeaderInjectorPlugin(plugin_pb2_grpc.PluginManagerServicer, plugin_pb2_grpc.MiddlewareServicer):
    """A plugin that adds custom headers to HTTP requests."""

    def __init__(self):
        self.config = {}
        self.initialized = False
        self.custom_headers = {
            'X-Plugin-Name': 'header-injector',
            'X-Plugin-Language': 'python',
            'X-Request-Processed': 'true'
        }

    # PluginManager Service Implementation

    def GetInfo(self, request, context):
        """Return plugin metadata."""
        return plugin_pb2.PluginInfo(
            name="header-injector",
            version="1.0.0",
            description="A Python plugin that injects custom headers into requests"
        )

    def InitializePlugin(self, request, context):
        """Initialize the plugin."""
        logger.info("Header injector plugin initializing...")
        self.initialized = True
        return plugin_pb2.Result(success=True, message="Initialized successfully")

    def ConfigurePlugin(self, request, context):
        """Apply configuration to the plugin."""
        self.config = dict(request.values)
        logger.info(f"Configured with: {self.config}")
        return plugin_pb2.Result(success=True, message="Configuration applied")

    def ShutdownPlugin(self, request, context):
        """Clean up plugin resources."""
        logger.info("Cleaning up...")
        self.initialized = False
        return plugin_pb2.Result(success=True, message="Cleanup completed")

    def CheckHealth(self, request, context):
        """Check plugin health."""
        if not self.initialized:
            return plugin_pb2.Result(success=False, message="Not initialized")
        return plugin_pb2.Result(success=True, message="Healthy")

    # Middleware Service Implementation

    def ShouldHandle(self, request, context):
        """Determine if this plugin should process the request."""
        excluded_paths = ['/health', '/docs']
        applies = not any(request.path.startswith(path) for path in excluded_paths)
        logger.info(f"ShouldHandle {request.method} {request.path}: {applies}")
        return plugin_pb2.BoolResult(value=applies)

    def ProcessRequest(self, request, context):
        """Process the request by injecting headers."""
        logger.info(f"Processing request: {request.method} {request.path}")

        # Copy existing headers and add custom ones
        headers = dict(request.headers)
        headers.update(self.custom_headers)
        headers['X-Request-Timestamp'] = str(int(__import__('time').time()))

        logger.info(f"Added {len(self.custom_headers)} custom headers")

        return plugin_pb2.Response(continue_=True, headers=headers)


def serve():
    """Start the plugin gRPC server."""
    parser = argparse.ArgumentParser(description="Header Injector Plugin")
    parser.add_argument("--socket", required=True, help="gRPC socket address")
    parser.add_argument("--mode", default="unix", choices=["unix", "tcp"],
                        help="Socket mode (unix or tcp)")
    args = parser.parse_args()

    address = args.socket
    mode = args.mode

    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    plugin = HeaderInjectorPlugin()

    plugin_pb2_grpc.add_PluginManagerServicer_to_server(plugin, server)
    plugin_pb2_grpc.add_MiddlewareServicer_to_server(plugin, server)

    # Listen on the appropriate socket type
    if mode == "unix":
        server.add_insecure_port(f'unix:{address}')
    elif mode == "tcp":
        server.add_insecure_port(address)
    else:
        logger.error(f"Unsupported PLUGIN_MODE: {mode} (use 'unix' or 'tcp')")
        sys.exit(1)

    logger.info(f"Plugin server listening on {address} ({mode})")

    try:
        server.start()
        server.wait_for_termination()
    except KeyboardInterrupt:
        logger.info("Plugin shutting down...")
        server.stop(0)


if __name__ == "__main__":
    serve()