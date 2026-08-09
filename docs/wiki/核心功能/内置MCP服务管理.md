---
title: Built-in MCP Service Management
tags: [Core Features, MCP, System Administration, Built-in Services]
aliases: [Built-in MCP, BuiltinMCP, BUILTIN_MCP_SERVICES]
source: BUILTIN_MCP_SERVICES.md
---

# Built-in MCP Service Management Guide

## Overview

Built-in MCP services are system-level MCP (Model Context Protocol) service configurations that are visible to all spaces, but with sensitive information hidden, and cannot be edited or deleted. Built-in MCP services are typically used to provide default system access to external tools and resources, ensuring that all spaces can use a unified set of MCP services.

> For MCP service operations from the user's perspective, see [MCP Feature Usage Guide](MCP功能使用说明.md)

## Built-in MCP Service Characteristics

- **Visible to all spaces**: Built-in MCP services are visible to all spaces without requiring separate configuration
- **Security protection**: Sensitive information for built-in MCP services (URL, authentication configuration, headers, environment variables) is hidden and cannot be viewed in detail
- **Read-only protection**: Built-in MCP services cannot be edited or deleted; only connection testing is supported
- **Unified management**: Maintained centrally by system administrators to ensure configuration consistency and security

## Comparison with Built-in Models

| Feature | Built-in Model | Built-in MCP Service |
|------|---------|--------------|
| Identifier field | `is_builtin` | `is_builtin` |
| Visibility scope | All spaces | All spaces |
| Hidden information | API Key, Base URL | URL, authentication configuration, headers, environment variables |
| Edit protection | Cannot be edited/deleted | Cannot be edited/deleted |
| Frontend label | Displays "Built-in" label | Displays "Built-in" label |
| Enable/disable control | — | Disable toggle (always enabled) |

> For detailed management of built-in models, see [Built-in Model Management](内置模型管理.md)

## How to Add a Built-in MCP Service

Built-in MCP services must be inserted directly through the database.

### 1. Prepare Service Data

- Service name (name)
- Service description (description)
- Transport type (transport_type): `sse` or `http-streamable`
- Service address (url)
- Authentication configuration (auth_config)
- Advanced configuration (advanced_config)
- Space ID (tenant_id): it is recommended to use a space ID smaller than 10000

**Supported transport types**:
- `sse`: Server-Sent Events, recommended for a streaming experience
- `http-streamable`: HTTP Streamable, standard HTTP compatible

> Note: For security reasons, the `stdio` transport type has been disabled on the server side.

### 2. Execute the SQL Insert Statement

```sql
INSERT INTO mcp_services (
    id, tenant_id, name, description, enabled,
    transport_type, url, auth_config, advanced_config, is_builtin
) VALUES (
    'builtin-mcp-001', 10000, 'Web Search', 'Built-in Web Search MCP service',
    true, 'sse', 'https://mcp.example.com/sse',
    '{"api_key": "your-api-key"}'::jsonb,
    '{"timeout": 30, "retry_count": 3, "retry_delay": 1}'::jsonb,
    true
) ON CONFLICT (id) DO NOTHING;
```

### 3. Verify the Insertion Result

```sql
SELECT id, name, transport_type, enabled, is_builtin
FROM mcp_services WHERE is_builtin = true ORDER BY created_at;
```

## Notes

1. **ID naming convention**: it is recommended to use the `builtin-mcp-{sequence number}` format
2. **Space ID**: it is recommended to use the first space ID (usually 10000)
3. **JSON format**: `auth_config`, `advanced_config`, and `headers` must be valid JSON
4. **Idempotency**: use `ON CONFLICT (id) DO NOTHING` to ensure repeated execution does not raise errors
5. **Security**: the URL and authentication information of built-in MCP services are automatically hidden on the frontend
6. **Transport type restriction**: only `sse` and `http-streamable` are supported

## Setting an Existing MCP Service as a Built-in Service

```sql
UPDATE mcp_services SET is_builtin = true WHERE id = 'service ID' AND name = 'service name';
```

## Removing a Built-in MCP Service

```sql
UPDATE mcp_services SET is_builtin = false WHERE id = 'service ID';
```

---

## Backlinks

- [Home](../Home.md) — Wiki home navigation
- [MCP Feature Usage Guide](MCP功能使用说明.md) — MCP service operations from the user's perspective
- [Built-in Model Management](内置模型管理.md) — Same built-in system configuration pattern, similar approach
