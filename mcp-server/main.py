#!/usr/bin/env python3
"""
WeKnora MCP Server main entry point

This file provides a unified entry point for starting the WeKnora MCP server.
It can be run in several ways:
1. python main.py
2. python -m weknora_mcp_server
3. weknora-mcp-server (after installation)

Note: under stdio transport, stdout is the JSON-RPC channel; all diagnostic/informational messages must be written to
stderr, otherwise the MCP protocol stream breaks and the client will report a "startup failure". All print statements in this file
go through stderr.
"""

import argparse
import asyncio
import os
import sys
from pathlib import Path


def setup_environment():
    """Set up environment and paths"""
    # Ensure the current directory is on the Python path
    current_dir = Path(__file__).parent.absolute()
    if str(current_dir) not in sys.path:
        sys.path.insert(0, str(current_dir))


def check_dependencies():
    """Check whether dependencies are installed"""
    try:
        import mcp
        import requests

        return True
    except ImportError as e:
        print(f"Missing dependency: {e}", file=sys.stderr)
        print("Please run: pip install -r requirements.txt", file=sys.stderr)
        return False


def check_environment_variables():
    """Check environment variable configuration"""
    base_url = os.getenv("WEKNORA_BASE_URL")
    api_key = os.getenv("WEKNORA_API_KEY")

    print("=== WeKnora MCP Server environment check ===", file=sys.stderr)
    print(f"Base URL: {base_url or 'http://localhost:8080/api/v1 (default)'}", file=sys.stderr)
    print(f"API Key: {'set' if api_key else 'not set (warning)'}", file=sys.stderr)

    if not base_url:
        print("Tip: you can set the WEKNORA_BASE_URL environment variable", file=sys.stderr)

    if not api_key:
        print("Warning: it is recommended to set the WEKNORA_API_KEY environment variable", file=sys.stderr)

    print("=" * 40, file=sys.stderr)
    return True


def parse_arguments():
    """Parse command-line arguments"""
    parser = argparse.ArgumentParser(
        description="WeKnora MCP Server - Model Context Protocol server for WeKnora API",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  python main.py                    # start with default configuration
  python main.py --check-only       # only check environment, don't start the server
  python main.py --verbose          # enable verbose logging
  
Environment variables:
  WEKNORA_BASE_URL       WeKnora API base URL (default: http://localhost:8080/api/v1)
  WEKNORA_API_KEY        WeKnora API key
  MCP_SERVER_AUTH_TOKEN  Required for SSE/HTTP transport; clients pass it via Authorization: Bearer
        """,
    )

    parser.add_argument(
        "--check-only", action="store_true", help="only check environment configuration without starting the server"
    )

    parser.add_argument("--verbose", "-v", action="store_true", help="enable verbose logging output")

    parser.add_argument(
        "--version", action="version", version="WeKnora MCP Server 1.1.1"
    )

    parser.add_argument(
        "--transport",
        choices=["stdio", "sse", "http"],
        default=os.getenv("MCP_TRANSPORT", "stdio"),
        help="Transport type: stdio (default), sse, or http",
    )
    parser.add_argument(
        "--host",
        default=os.getenv("MCP_HOST", "127.0.0.1"),
        help="Bind host for network transports (default: 127.0.0.1)",
    )
    parser.add_argument(
        "--port",
        type=int,
        default=int(os.getenv("MCP_PORT", "8000")),
        help="Bind port for network transports (default: 8000)",
    )

    return parser.parse_args()


async def main():
    """Main function"""
    args = parse_arguments()

    # Set up environment
    setup_environment()

    # Check dependencies
    if not check_dependencies():
        sys.exit(1)

    # Check environment variables
    check_environment_variables()

    # If only checking the environment, exit
    if args.check_only:
        print("Environment check complete.", file=sys.stderr)
        return

    # Set log level
    if args.verbose:
        import logging

        logging.basicConfig(level=logging.DEBUG)
        print("Verbose logging mode enabled", file=sys.stderr)

    try:
        print(f"Starting WeKnora MCP Server (transport={args.transport})...", file=sys.stderr)

        from weknora_mcp_server import run_stdio, run_sse, run_http

        # Select transport mode based on CLI argument or MCP_TRANSPORT env var
        # - stdio: Default, used by VS Code Copilot for local integration
        # - sse: Server-Sent Events over HTTP, suitable for cloud/remote deployments
        # - http: Streamable HTTP sessions (MCP 2025-03-26 spec), compatible with REST clients
        if args.transport == "stdio":
            # Stdio mode: communication via stdin/stdout pipes (typical for CLI integrations)
            await run_stdio()
        elif args.transport == "sse":
            # SSE mode: HTTP server with Server-Sent Events for bidirectional streaming
            await run_sse(args.host, args.port)
        elif args.transport == "http":
            # HTTP mode: HTTP REST server with request/response model
            await run_http(args.host, args.port)

    except ImportError as e:
        print(f"Import error: {e}", file=sys.stderr)
        print("Make sure all files are in the correct location", file=sys.stderr)
        sys.exit(1)
    except KeyboardInterrupt:
        print("\nServer stopped", file=sys.stderr)
    except Exception as e:
        print(f"Server runtime error: {e}", file=sys.stderr)
        if args.verbose:
            import traceback

            traceback.print_exc()
        sys.exit(1)


def sync_main():
    """Synchronous version of the main function, used for entry_points"""
    asyncio.run(main())


if __name__ == "__main__":
    asyncio.run(main())
