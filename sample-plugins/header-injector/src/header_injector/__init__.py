"""Header Injector Plugin package."""

import asyncio
import sys

from mcpd_plugins import serve as sdk_serve

from .main import HeaderInjectorPlugin

__all__ = ["serve", "HeaderInjectorPlugin"]


def serve():
    """Entry point for the header-injector plugin.

    This function is called when the plugin is executed as a standalone application.
    It creates a plugin instance and starts the gRPC server.
    """
    plugin = HeaderInjectorPlugin()
    asyncio.run(sdk_serve(plugin, sys.argv))
