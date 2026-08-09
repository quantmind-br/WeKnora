# IM Integration Development

WeKnora's IM integration module connects enterprise instant messaging platforms (WeCom, Feishu, Lark, Slack, Telegram, DingTalk, Mattermost) to WeKnora's knowledge Q&A pipeline, allowing users to ask questions directly in IM and receive real-time streaming answers.

IM channels are bound to an Agent, and a single Agent can be connected to multiple IM channels.

> Agents in IM channels can use [MCP Tools](../Core-Features/MCP-Usage-Guide.md) and [Skills](../Core-Features/Agent-Skills-System.md)

## Supported Platforms

| Platform | WebSocket Mode | Webhook Mode | Streaming Output |
|------|:-:|:-:|:-:|
| WeCom | ✅ | ✅ | ✅ |
| Feishu | ✅ | ✅ | ✅ (CardKit) |
| Lark (International Feishu) | ✅ | ✅ | ✅ (CardKit) |
| Slack | ✅ (Socket Mode) | ✅ (Events API) | ✅ |
| Telegram | ✅ (Long Polling) | ✅ | ✅ |
| DingTalk | ✅ (Stream) | ✅ | ✅ (AI Card) |
| Mattermost | — | ✅ | ✅ |

## Quick Start Guide

### Prerequisites

- WeKnora is deployed and running
- At least one Agent (custom agent) has been created
- The Agent has been configured with a model and knowledge base

> For Agent model configuration, see [Built-in Model Management](../Core-Features/Builtin-Model-Management.md)

### WeCom Integration

Two modes are provided:
- **WebSocket Mode** (Smart Bot, recommended) — no public domain required
- **Webhook Mode** (self-built app) — requires a public callback address

### Feishu Integration

- **WebSocket Mode** (recommended) — no public domain required
- **Webhook Mode** — requires a public callback address

> Feishu is also a supported platform for data source imports; see [Data Source Import Development](Data-Source-Import-Development.md)

### Lark Integration

Lark is the international version of Feishu. The IM interfaces are identical and share the same adapter as Feishu. The integration steps are largely the same, but the open platform and the **permission manifest** differ:

- Feishu: <https://open.feishu.cn/>
- Lark: <https://open.larksuite.com/>

> **The permission manifest cannot be reused from Feishu**: the Feishu manifest mixes in permissions used by the data source connector (Wiki sync), some of which don't exist on Lark, causing the entire import to fail. Lark only needs the 6 IM-related permissions; see
> [IM Integration Development Documentation — Lark Permission Configuration](../../IM-Integration-Development.md#lark-权限配置).

> Apps on the two clouds are not interchangeable — credentials are only valid on the cloud where they were created.

### Slack Integration

- **Socket Mode** (recommended) — no public domain required
- **Events API** — requires a public callback address

### Telegram Integration

- **Long Polling Mode** (recommended) — no public domain required
- **Webhook Mode** — requires a public HTTPS callback

### DingTalk Integration

- **Stream Mode** (recommended) — no public domain required
- **Webhook Mode** — requires a public callback address

### Mattermost Integration

- Only **Webhook Mode** is supported (outgoing webhook + REST API v4)

## Architecture Design

The system uses the **Adapter Pattern**, with each platform implementing the `im.Adapter` interface, dynamically created via `AdapterFactory`. The core design patterns include:

| Pattern | Purpose |
|------|------|
| Adapter Pattern | Unifies differences across IM platforms |
| Factory Pattern | Dynamically creates Adapters from database channel configuration |
| Command Pattern | Pluggable slash command system |
| Producer-Consumer | QA queue + Worker Pool |

## Slash Command System

| Command | Description |
|------|------|
| `/help` | Show all available commands |
| `/info` | View information about the currently bound agent |
| `/search` | Perform a hybrid search against the knowledge base |
| `/stop` | Cancel the current QA request |
| `/clear` | Clear the current conversation memory |

## Extending with New Platforms

Adding a new IM platform only takes 3 steps:

1. Implement the `im.Adapter` interface (optionally `StreamSender`, `FileDownloader`)
2. Register the adapter factory
3. Add the platform option on the frontend

> The extension development pattern is similar to [Adding a Web Search Engine](Adding-a-New-Search-Engine.md) and [Integrating a Vector Database](Integrating-a-Vector-Database.md)

## Related Topics

- [Data Source Import Development](../Integration-Extension/Data-Source-Import-Development.md) — Feishu data source sync (shares Feishu app credentials)
- [MCP Feature Usage Guide](../Core-Features/MCP-Usage-Guide.md) — Agents can invoke MCP tools
- [Agent Skills System](../Core-Features/Agent-Skills-System.md) — Agents can use Skills
- [Development Guide](../Development-Deployment/Development-Guide.md) — Setting up the development environment

---

## Backlinks

- [Home](../Home.md) — Wiki home navigation
- [Data Source Import Development](../Integration-Extension/Data-Source-Import-Development.md) — Also involves Feishu integration, can share app credentials
- [MCP Feature Usage Guide](../Core-Features/MCP-Usage-Guide.md) — Agents in IM can invoke MCP tools
- [Agent Skills System](../Core-Features/Agent-Skills-System.md) — Agents in IM can use Skills
- [Version Roadmap](../Project-Overview/Version-Roadmap.md) — Completed milestones for IM integration
