# Built-in MCP Service Management Guide

## Overview

Built-in MCP services are system-level MCP (Model Context Protocol) service configurations, visible to all spaces, but sensitive information is hidden and they cannot be edited or deleted. Built-in MCP services are typically used to provide default system access to external tools and resources, ensuring that all spaces can use a unified set of MCP services.

## Built-in MCP Service Features

- **Visible to all spaces**: Built-in MCP services are visible to all spaces without needing separate configuration
- **Security protection**: Sensitive information for built-in MCP services (URL, authentication configuration, Headers, environment variables) is hidden and details cannot be viewed
- **Read-only protection**: Built-in MCP services cannot be edited or deleted, and only support testing the connection
- **Unified management**: Maintained centrally by system administrators, ensuring configuration consistency and security

## Comparison with Built-in Models

| Feature | Built-in Model | Built-in MCP Service |
|------|---------|--------------|
| Identifier field | `is_builtin` | `is_builtin` |
| Visibility scope | All spaces | All spaces |
| Hidden information | API Key, Base URL | URL, authentication configuration, Headers, environment variables |
| Edit protection | Cannot be edited/deleted | Cannot be edited/deleted |
| Frontend label | Shows "Built-in" label | Shows "Built-in" label |
| Enable/disable control | — | Disable toggle (always enabled) |

## How to Add a Built-in MCP Service

Built-in MCP services need to be inserted directly via the database. Below are the steps for adding a built-in MCP service:

### 1. Prepare the Service Data

First, make sure you have the configuration information you want to set up as a built-in MCP service, including:
- Service name (name)
- Service description (description)
- Transport type (transport_type): `sse` or `http-streamable`
- Service URL (url): required for SSE / HTTP Streamable
- Authentication configuration (auth_config): optional, including api_key, token, etc.
- Advanced configuration (advanced_config): optional, including timeout, retry policy, etc.
- Space ID (tenant_id): it is recommended to use a space ID smaller than 10000 to avoid conflicts

**Supported transport types**:
- `sse`: Server-Sent Events, recommended for a streaming experience
- `http-streamable`: HTTP Streamable, standard HTTP compatible

> Note: For security reasons, the `stdio` transport type has been disabled on the server side.

### 2. Execute the SQL Insert Statement

Use the following SQL statements to insert a built-in MCP service:

```sql
-- Example: Insert a built-in MCP service using the SSE transport type
INSERT INTO mcp_services (
    id,
    tenant_id,
    name,
    description,
    enabled,
    transport_type,
    url,
    auth_config,
    advanced_config,
    is_builtin
) VALUES (
    'builtin-mcp-001',                                -- Use a fixed ID; the builtin-mcp- prefix is recommended
    10000,                                             -- Space ID (use the first space)
    'Web Search',                                      -- Service name
    'Built-in Web Search MCP Service',                            -- Description
    true,                                              -- Enabled status
    'sse',                                             -- Transport type
    'https://mcp.example.com/sse',                     -- Service URL
    '{"api_key": "your-api-key"}'::jsonb,              -- Authentication configuration
    '{"timeout": 30, "retry_count": 3, "retry_delay": 1}'::jsonb,  -- Advanced configuration
    true                                               -- Mark as a built-in service
) ON CONFLICT (id) DO NOTHING;

-- Example: Insert a built-in MCP service using the HTTP Streamable transport type
INSERT INTO mcp_services (
    id,
    tenant_id,
    name,
    description,
    enabled,
    transport_type,
    url,
    headers,
    auth_config,
    advanced_config,
    is_builtin
) VALUES (
    'builtin-mcp-002',
    10000,
    'Code Interpreter',
    'Built-in Code Interpreter MCP Service',
    true,
    'http-streamable',
    'https://mcp.example.com/stream',
    '{"X-Custom-Header": "value"}'::jsonb,
    '{"token": "your-bearer-token"}'::jsonb,
    '{"timeout": 60, "retry_count": 2, "retry_delay": 2}'::jsonb,
    true
) ON CONFLICT (id) DO NOTHING;
```

### 3. Verify the Insertion Result

Execute the following SQL query to verify that the built-in MCP service was inserted successfully:

```sql
SELECT id, name, transport_type, enabled, is_builtin
FROM mcp_services
WHERE is_builtin = true
ORDER BY created_at;
```

## Notes

1. **ID naming convention**: It is recommended to use the format `builtin-mcp-{sequence number}`, for example `builtin-mcp-001`, `builtin-mcp-002`
2. **Space ID**: Built-in MCP services can belong to any space, but it is recommended to use the first space's ID (usually 10000)
3. **JSON format**: Fields such as `auth_config`, `advanced_config`, and `headers` must be in valid JSON format
4. **Idempotency**: Using `ON CONFLICT (id) DO NOTHING` ensures that repeated execution will not raise an error
5. **Security**: The URL and authentication information for built-in MCP services are automatically hidden on the frontend, but the raw data still exists in the database — please safeguard your database access permissions carefully
6. **Transport type restrictions**: Only `sse` and `http-streamable` are supported; `stdio` has been disabled

## Setting an Existing MCP Service as a Built-in Service

If you already have an MCP service and want to set it as a built-in service, you can use an UPDATE statement:

```sql
UPDATE mcp_services
SET is_builtin = true
WHERE id = 'service-id' AND name = 'service-name';
```

## Removing a Built-in MCP Service

If you need to remove the built-in flag (restoring it to a regular MCP service), execute:

```sql
UPDATE mcp_services
SET is_builtin = false
WHERE id = 'service-id';
```

Note: After removing the built-in flag, the MCP service will revert to a regular service and can be edited and deleted.
