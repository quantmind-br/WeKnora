#!/usr/bin/env python3
"""
WeKnora Local MCP Demo Server

Minimal runnable external MCP service, used to test client integration in WeKnora's "Settings → MCP Services"
Listens on Streamable HTTP at http://127.0.0.1:8010/mcp by default

Start:
  export MCP_SERVER_AUTH_TOKEN=weknora-demo-token
  python server.py

WeKnora configuration:
  Transport: HTTP Streamable
  URL：http://127.0.0.1:8010/mcp
  Auth: Bearer, token must match MCP_SERVER_AUTH_TOKEN
"""

from __future__ import annotations

import argparse
import asyncio
import logging
import os
import secrets
import sys
from datetime import datetime, timezone
from typing import Any

from mcp.server import MCPServer

logging.basicConfig(level=logging.INFO, format="%(levelname)s %(message)s")
logger = logging.getLogger("mcp-demo")

mcp = MCPServer("weknora-mcp-demo", version="0.1.0")

# Demo corpus paired with website-docs/sample-data/, for cross-checking knowledge base answers after Agent calls.
DEMO_POLICIES: dict[str, str] = {
    "warranty": "智能家居中控 Pro 整机保修 24 个月，电池类配件 12 个月；人为拆解、进水不在保修范围。",
    "offline_voice": "若语音走云端识别，断外网后仅支持 App 与本地触摸屏；配置本地语音包后可继续使用基础指令。",
    "device_limit": "个人版账号最多绑定 3 台中控；企业版按合同授权，默认 50 台。",
    "travel_hotel_tier1": "一线城市（北上广深）出差住宿报销上限 600 元/晚（含税）。",
    "travel_meal": "出差期间餐饮费不单独报销；一线城市差旅补贴 150 元/天。",
    "poc_owner": "售后知识库 POC 技术负责人是研发部张明，产品对接人是李薇，测试负责人是赵磊。",
    "poc_deadline": "售后知识库 POC 目标 2024-03-01 前完成内网演示。",
    "matter_cert": "固件 3.5 计划在 2024 年 3 月底前发布灰度，完成 Matter 1.2 认证。",
}

DEMO_CONTACTS: list[dict[str, str]] = [
    {"name": "陈浩", "role": "产品总监", "department": "产品部"},
    {"name": "张明", "role": "知识库与 AI 模块负责人", "department": "研发部"},
    {"name": "李薇", "role": "产品运营", "department": "产品部"},
    {"name": "王雪", "role": "交互设计负责人", "department": "设计部"},
    {"name": "赵磊", "role": "测试经理", "department": "测试部"},
]


def network_transport_auth_token() -> str:
    return os.getenv("MCP_SERVER_AUTH_TOKEN", "").strip()


def require_network_transport_auth(transport: str) -> str:
    token = network_transport_auth_token()
    if transport in ("sse", "http") and not token:
        logger.error(
            "MCP_SERVER_AUTH_TOKEN is required for %s transport. "
            "Example: export MCP_SERVER_AUTH_TOKEN=weknora-demo-token",
            transport,
        )
        sys.exit(1)
    return token


class MCPAuthMiddleware:
    """Bearer auth middleware for SSE / HTTP transport."""

    def __init__(self, app, token: str):
        self.app = app
        self.token = token

    async def __call__(self, scope, receive, send):
        if scope.get("type") != "http":
            await self.app(scope, receive, send)
            return

        headers = {
            k.decode("latin-1").lower(): v.decode("latin-1")
            for k, v in scope.get("headers", [])
        }
        provided = ""
        auth = headers.get("authorization", "")
        if auth.lower().startswith("bearer "):
            provided = auth[7:].strip()
        elif "x-mcp-auth-token" in headers:
            provided = headers["x-mcp-auth-token"]

        if not provided or not secrets.compare_digest(provided, self.token):
            body = b'{"error":"unauthorized"}'
            await send(
                {
                    "type": "http.response.start",
                    "status": 401,
                    "headers": [[b"content-type", b"application/json"]],
                }
            )
            await send({"type": "http.response.body", "body": body})
            return

        await self.app(scope, receive, send)


@mcp.tool()
def echo(message: str) -> dict[str, Any]:
    """Echo a message, used to verify MCP connectivity."""
    return {"echo": message}


@mcp.tool()
def add(a: float, b: float) -> dict[str, Any]:
    """Compute the sum of two numbers."""
    return {"a": a, "b": b, "sum": a + b}


@mcp.tool()
def server_time() -> dict[str, str]:
    """Return the current UTC time of the MCP Demo server."""
    now = datetime.now(timezone.utc)
    return {
        "iso": now.isoformat(),
        "unix": str(int(now.timestamp())),
    }


@mcp.tool()
def lookup_policy(topic: str) -> dict[str, Any]:
    """Query demo policy/project info. topic can be warranty/offline_voice/device_limit/travel_hotel_tier1/travel_meal/poc_owner/poc_deadline/matter_cert, or Chinese keywords like "保修" "报销" "POC"."""
    key = topic.strip().lower().replace(" ", "_")
    aliases = {
        "保修": "warranty",
        "质保": "warranty",
        "离线": "offline_voice",
        "语音": "offline_voice",
        "设备数": "device_limit",
        "住宿": "travel_hotel_tier1",
        "报销": "travel_hotel_tier1",
        "餐饮": "travel_meal",
        "补贴": "travel_meal",
        "负责人": "poc_owner",
        "张明": "poc_owner",
        "poc": "poc_owner",
        "验收": "poc_deadline",
        "matter": "matter_cert",
        "认证": "matter_cert",
    }
    for alias, mapped in aliases.items():
        if alias in topic:
            key = mapped
            break

    if key in DEMO_POLICIES:
        return {"topic": key, "answer": DEMO_POLICIES[key], "source": "mcp-demo/static"}

    matches = {
        k: v
        for k, v in DEMO_POLICIES.items()
        if key in k or any(ch in k for ch in key if len(key) >= 2)
    }
    if len(matches) == 1:
        only_key = next(iter(matches))
        return {"topic": only_key, "answer": matches[only_key], "source": "mcp-demo/static"}

    return {
        "topic": topic,
        "available_topics": sorted(DEMO_POLICIES.keys()),
        "hint": "传入 topic 为上述键名，或中文关键词如「保修」「报销」「POC」。",
    }


@mcp.tool()
def list_team_contacts(department: str = "") -> dict[str, Any]:
    """List demo project team members; can filter by department name (Product / R&D / Design / QA)."""
    rows = DEMO_CONTACTS
    if department.strip():
        needle = department.strip()
        rows = [c for c in rows if needle in c["department"]]
    return {"count": len(rows), "contacts": rows}


@mcp.tool()
def send_demo_alert(channel: str, message: str) -> dict[str, Any]:
    """Simulate sending a notification to an external channel (for demo purposes only, no actual outbound message).

    Suitable for testing MCP tool manual approval in WeKnora; recommend marking this tool as requiring approval.
    """
    return {
        "ok": True,
        "simulated": True,
        "channel": channel,
        "message": message,
        "sent_at": datetime.now(timezone.utc).isoformat(),
    }


async def run_http(host: str, port: int) -> None:
    auth_token = require_network_transport_auth("http")
    try:
        import uvicorn
    except ImportError as e:
        raise ImportError("HTTP transport requires: pip install starlette uvicorn") from e

    starlette_app = MCPAuthMiddleware(
        mcp.streamable_http_app(host=host, stateless_http=True),
        auth_token,
    )
    logger.info("Streamable HTTP MCP demo listening on http://%s:%d/mcp", host, port)
    config = uvicorn.Config(starlette_app, host=host, port=port, log_level="info")
    server = uvicorn.Server(config)
    await server.serve()


async def run_sse(host: str, port: int) -> None:
    auth_token = require_network_transport_auth("sse")
    try:
        import uvicorn
    except ImportError as e:
        raise ImportError("SSE transport requires: pip install starlette uvicorn") from e

    starlette_app = MCPAuthMiddleware(
        mcp.sse_app(host=host, message_path="/sse/messages/"),
        auth_token,
    )
    logger.info("SSE MCP demo listening on http://%s:%d/sse", host, port)
    config = uvicorn.Config(starlette_app, host=host, port=port, log_level="info")
    server = uvicorn.Server(config)
    await server.serve()


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="WeKnora local MCP demo server")
    parser.add_argument(
        "--transport",
        choices=["http", "sse"],
        default=os.getenv("MCP_TRANSPORT", "http"),
        help="Network transport (default: http / Streamable HTTP)",
    )
    parser.add_argument("--host", default=os.getenv("MCP_HOST", "127.0.0.1"))
    parser.add_argument("--port", type=int, default=int(os.getenv("MCP_PORT", "8010")))
    return parser.parse_args()


async def main() -> None:
    args = parse_args()
    if args.transport == "http":
        await run_http(args.host, args.port)
    else:
        await run_sse(args.host, args.port)


if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        logger.info("stopped")
