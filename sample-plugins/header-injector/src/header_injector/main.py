#!/usr/bin/env python3
"""Header Injector Plugin - A Python-based middleware plugin example.

This plugin demonstrates how to inject custom headers into HTTP requests
using the mcpd-plugins SDK. It adds metadata headers to all requests that
are not explicitly excluded.
"""

import asyncio
import logging
import sys
import time

from google.protobuf.empty_pb2 import Empty
from grpc.aio import ServicerContext
from mcpd_plugins import BasePlugin, serve
from mcpd_plugins.v1.plugins.plugin_pb2 import (
    FLOW_REQUEST,
    Capabilities,
    HTTPRequest,
    HTTPResponse,
    Metadata,
    PluginConfig,
)

logging.basicConfig(level=logging.INFO, format="%(asctime)s - %(name)s - %(levelname)s - %(message)s")
logger = logging.getLogger("header-injector")


class HeaderInjectorPlugin(BasePlugin):
    """A plugin that adds custom headers to HTTP requests."""

    def __init__(self):
        """Initialize the plugin with default configuration."""
        super().__init__()
        self.custom_headers = {
            "X-Plugin-Name": "header-injector",
            "X-Plugin-Language": "python",
            "X-Request-Processed": "true",
        }
        self.excluded_paths = ["/health", "/docs"]

    async def GetMetadata(self, request: Empty, context: ServicerContext) -> Metadata:
        """Return plugin metadata."""
        _ = (request, context)
        return Metadata(
            name="header-injector",
            version="1.0.0",
            description="A Python plugin that injects custom headers into requests",
        )

    async def GetCapabilities(self, request: Empty, context: ServicerContext) -> Capabilities:
        """Declare support for request flow only."""
        _ = (request, context)
        return Capabilities(flows=[FLOW_REQUEST])

    async def Configure(self, request: PluginConfig, context: ServicerContext) -> Empty:
        """Apply configuration to the plugin."""
        _ = context
        if request.config:
            logger.info("Configured with: %s", dict(request.config))
        return Empty()

    async def HandleRequest(self, request: HTTPRequest, context: ServicerContext) -> HTTPResponse:
        """Process the request by injecting custom headers.

        Adds custom headers to requests unless the path is in the excluded list.
        """
        _ = context
        logger.info("Processing request: %s %s", request.method, request.url)

        # Check if this path should be excluded.
        should_skip = any(request.url.startswith(path) for path in self.excluded_paths)

        if should_skip:
            logger.info("Skipping excluded path: %s", request.url)
            return HTTPResponse(**{"continue": True})

        # Create response with Continue=True to pass the request through.
        response = HTTPResponse(**{"continue": True})

        # Copy the original request and add custom headers.
        response.modified_request.CopyFrom(request)

        # Add custom headers.
        for header_name, header_value in self.custom_headers.items():
            response.modified_request.headers[header_name] = header_value

        # Add timestamp header.
        response.modified_request.headers["X-Request-Timestamp"] = str(int(time.time()))

        logger.info("Added %d custom headers", len(self.custom_headers) + 1)
        return response


if __name__ == "__main__":
    asyncio.run(serve(HeaderInjectorPlugin(), sys.argv))
