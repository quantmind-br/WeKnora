#!/usr/bin/env python3
"""
WeKnora MCP Server convenience startup script

This is a simplified startup script that provides only the most basic functionality.
For more options, use main.py

Note: under stdio transport, stdout is the JSON-RPC channel, so all diagnostic/prompt messages must be written to
stderr, otherwise it will break the MCP protocol stream and cause the client to determine "startup failed". All print statements in this script
are output via stderr.
"""

import os
import sys
from pathlib import Path


def main():
    """Simple startup function"""
    # Add the current directory to the Python path
    current_dir = Path(__file__).parent.absolute()
    if str(current_dir) not in sys.path:
        sys.path.insert(0, str(current_dir))

    # Check environment variables
    base_url = os.getenv("WEKNORA_BASE_URL", "http://localhost:8080/api/v1")
    api_key = os.getenv("WEKNORA_API_KEY", "")

    print("WeKnora MCP Server", file=sys.stderr)
    print(f"Base URL: {base_url}", file=sys.stderr)
    print(f"API Key: {'set' if api_key else 'not set'}", file=sys.stderr)
    print("-" * 40, file=sys.stderr)

    try:
        # Import and run
        from main import sync_main

        sync_main()
    except ImportError:
        print("Error: unable to import required modules", file=sys.stderr)
        print("Make sure you run: pip install -r requirements.txt", file=sys.stderr)
        sys.exit(1)
    except KeyboardInterrupt:
        print("\nServer stopped", file=sys.stderr)
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
