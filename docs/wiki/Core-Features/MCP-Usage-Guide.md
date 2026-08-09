---
title: MCP Feature Usage Guide
tags: [Core Features, MCP, Tool Integration]
aliases: [MCP Usage, MCP Features]
source: MCP-Usage-Guide.md
---

# MCP Feature Usage Guide

## Feature Overview

- MCP (Model Context Protocol) lets WeKnora securely connect to external tools or data sources, extending the capabilities the Agent can invoke during reasoning.
- All services are centrally managed from `Settings > MCP Services` (`frontend/src/views/settings/McpSettings.vue`) in the frontend, with no need to manually edit configuration files.
- Each service includes a name, transport method (SSE / HTTP Streamable / Stdio), connection address or command, authentication information, and advanced timeout and retry policies.

> For system-level management of built-in MCP services, see [Built-in MCP Service Management](Builtin-MCP-Service-Management.md)

## Entry Point and Interface

- Open the `Settings -> MCP Services` menu on the left side of the console to see the list of all MCP services in the current space.
- From the list you can quickly enable/disable services, view descriptions, and use the menu on the right to perform "Test / Edit / Delete".
- The "Add Service" button opens the `McpServiceDialog` for creating or modifying a service.

## Common Workflows

### 1. Creating a New Service

- Click "Add Service", fill in the name and description, and select a transport method.
- SSE / HTTP Streamable requires an accessible service URL; Stdio requires configuring a `uvx`/`npx` command and arguments, with optional environment variables attached.
- Fill in the API Key, Bearer Token, timeout, and retry policy as needed. Once saved, the service will appear in the list.

### 2. Enabling/Disabling a Service

- Toggle the enable state in the list, and the system will immediately call the backend `updateMCPService`. If it fails, the state is automatically rolled back and an alert is shown.

### 3. Connection Test

- Select "Test" from the overflow menu; the frontend calls `/api/v1/mcp-services/{id}/test` and displays `McpTestResult`.
- On success, it shows the list of tools available on the service (including input schema) and resources; on failure, it displays an error message to help troubleshoot network or authentication issues.

### 4. Edit / Delete

- "Edit" brings up the existing configuration; save after making changes.
- "Delete" requires confirmation in the popup, after which the list refreshes automatically.

## Usage Recommendations

- **Choosing a transport method**: Prefer SSE for a streaming experience; switch to standard HTTP Streamable compatibility when needed; Stdio is suitable for local debugging or offline environments, running the MCP Server on the same machine.
- **Authentication management**: Store the API Key / Token in "Authentication Configuration". In production, it's recommended to create a separate minimal-privilege Key and rotate it regularly.
- **Retry policy**: For public internet or third-party services, increase `retry_count` and `retry_delay` appropriately to avoid Agent interruptions caused by intermittent timeouts.

## Related Topics

- [Built-in MCP Service Management](../Core-Features/Builtin-MCP-Service-Management.md) — Built-in MCP service configuration from a system administrator's perspective
- [Agent Skills System](Agent-Skills-System.md) — Another Agent extension mechanism
- [IM Integration Development](../Integration-Extension/IM-Integration-Development.md) — Using MCP tools with the Agent in IM channels
- [Add Web Search Engine](../Integration-Extension/Adding-a-New-Search-Engine.md) — Another way to extend search capabilities

---

## Backlinks

- [Home](../Home.md) — Wiki home navigation
- [Built-in MCP Service Management](Builtin-MCP-Service-Management.md) — System-level management of MCP (administrator's perspective)
- [Agent Skills System](Agent-Skills-System.md) — An Agent extension mechanism parallel to MCP
- [IM Integration Development](../Integration-Extension/IM-Integration-Development.md) — The Agent can invoke MCP tools in IM channels
