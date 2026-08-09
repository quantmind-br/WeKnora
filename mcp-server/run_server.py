#!/usr/bin/env python3
"""
WeKnora MCP Server startup script

Note: under stdio transport, stdout is the JSON-RPC channel, so all diagnostic/prompt messages must be written to
stderr, otherwise it will break the MCP protocol stream and cause the client to determine "startup failed". All print statements in this script
are output via stderr.
"""

import asyncio
import os
import sys


def check_environment():
    """Check environment configuration"""
    base_url = os.getenv("WEKNORA_BASE_URL")
    api_key = os.getenv("WEKNORA_API_KEY")

    if not base_url:
        print(
            "Warning: WEKNORA_BASE_URL environment variable not set, using default: http://localhost:8080/api/v1",
            file=sys.stderr,
        )

    if not api_key:
        print("Warning: WEKNORA_API_KEY environment variable not set", file=sys.stderr)

    print(f"WeKnora Base URL: {base_url or 'http://localhost:8080/api/v1'}", file=sys.stderr)
    print(f"API Key: {'set' if api_key else 'not set'}", file=sys.stderr)


def main():
    """Main function"""
    print("Starting WeKnora MCP Server...", file=sys.stderr)
    check_environment()

    try:
        from weknora_mcp_server import run

        asyncio.run(run())
    except ImportError as e:
        print(f"Import error: {e}", file=sys.stderr)
        print("Make sure all dependencies are installed: pip install -r requirements.txt", file=sys.stderr)
        sys.exit(1)
    except KeyboardInterrupt:
        print("\nServer stopped", file=sys.stderr)
    except Exception as e:
        print(f"Server runtime error: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
