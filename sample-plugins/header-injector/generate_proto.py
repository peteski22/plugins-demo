#!/usr/bin/env python3
"""Generate Python protobuf files from the plugin.proto definition."""

import subprocess
import sys
from pathlib import Path

def main():
    """Generate protobuf files using grpc_tools."""
    plugin_dir = Path(__file__).parent
    proto_dir = plugin_dir / "../../proto"
    output_dir = plugin_dir / "src/header_injector"

    # Generate Python protobuf files
    cmd = [
        sys.executable, "-m", "grpc_tools.protoc",
        f"--proto_path={proto_dir}",
        f"--python_out={output_dir}",
        f"--grpc_python_out={output_dir}",
        str(proto_dir / "plugin.proto")
    ]

    print(f"Running: {' '.join(cmd)}")
    result = subprocess.run(cmd, capture_output=True, text=True)

    if result.returncode != 0:
        print(f"Error generating protobuf files: {result.stderr}")
        sys.exit(1)

    print("Successfully generated protobuf files")

if __name__ == "__main__":
    main()