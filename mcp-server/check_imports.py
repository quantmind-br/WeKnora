#!/usr/bin/env python3
"""
Manual MCP import check script (not a unittest).

Usage: python check_imports.py
"""

try:
    import mcp.server.stdio  # noqa: F401

    print("✓ mcp.server.stdio imported successfully")
except ImportError as e:
    print(f"✗ mcp.server.stdio import failed: {e}")

try:
    # mcp 2.x high-level server API (MCPServer, formerly FastMCP).
    from mcp.server import MCPServer

    print("✓ MCPServer imported from mcp.server successfully")
except ImportError as e:
    print(f"✗ MCPServer import failed: {e}")

try:
    from mcp.server.mcpserver.exceptions import ToolError  # noqa: F401

    print("✓ ToolError imported from mcp.server.mcpserver.exceptions successfully")
except ImportError as e:
    print(f"✗ ToolError import failed: {e}")

try:
    import mcp_types  # noqa: F401  # standalone protocol types in mcp 2.x

    print("✓ mcp_types imported successfully")
except ImportError as e:
    print(f"✗ mcp_types import failed: {e}")

# Check MCP package structure
import mcp

print(f"\nMCP package version: {getattr(mcp, '__version__', 'unknown')}")
print(f"MCP package path: {mcp.__file__}")
