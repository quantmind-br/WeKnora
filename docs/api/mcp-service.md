# MCP Service API

[Back to Table of Contents](./README.md)

MCP (Model Context Protocol) service management interface, providing CRUD operations for MCP services, connectivity testing, tool/resource discovery, and manual approval policy configuration for tools.

| Method | Path                                              | Description                                          |
| ------ | ------------------------------------------------- | --------------------------------------------- |
| POST   | `/mcp-services`                                   | Create an MCP service                                 |
| GET    | `/mcp-services`                                   | Get the list of MCP services in the current space                   |
| GET    | `/mcp-services/:id`                               | Get MCP service details                             |
| PUT    | `/mcp-services/:id`                               | Update an MCP service (partial field update)                 |
| DELETE | `/mcp-services/:id`                               | Delete an MCP service                                 |
| POST   | `/mcp-services/:id/test`                          | Test MCP service connectivity                           |
| GET    | `/mcp-services/:id/tools`                         | Get the MCP service's tool list                         |
| GET    | `/mcp-services/:id/resources`                     | Get the MCP service's resource list                         |
| GET    | `/mcp-services/:id/tool-approvals`                | List the manual approval policy for each tool under this service |
| PUT    | `/mcp-services/:id/tool-approvals/:tool_name`     | Set/update the manual approval policy for a tool  |
| POST   | `/agent/tool-approvals/:pending_id`               | Handle a pending Agent tool call approval request  |

## POST `/mcp-services` - Create an MCP Service

**Request parameters**:

| Field             | Type    | Required | Description                                                                                          |
| ---------------- | ------- | ---- | ----------------------------------------------------------------------------------------------- |
| name             | string  | Yes   | Service name                                                                                      |
| description      | string  | No   | Service description                                                                                      |
| transport_type   | string  | Yes   | Transport type, options: `sse`, `http-streamable`, `stdio`                                              |
| url              | string  | Conditional | Service address; required when `transport_type` is `sse` / `http-streamable` (subject to SSRF security validation)        |
| headers          | object  | No   | Custom request headers                                                                                  |
| auth_config      | object  | No   | Authentication configuration, supports `api_key`, `token`                                                              |
| advanced_config  | object  | No   | Advanced configuration, supports `timeout`, `retry_count`, `retry_delay`                                          |
| stdio_config     | object  | Conditional | stdio transport configuration, including `command`, `args`; required when `transport_type` is `stdio`                  |
| env_vars         | object  | No   | Environment variables (commonly used in stdio scenarios)                                                                    |
| enabled          | boolean | No   | Whether it's enabled                                                                                      |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/mcp-services' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "name": "天气查询服务",
    "description": "提供全球天气信息查询",
    "transport_type": "sse",
    "url": "https://mcp.example.com/weather/sse",
    "headers": {
        "X-Custom-Header": "value"
    },
    "auth_config": {
        "api_key": "weather-api-key-xxxxx"
    },
    "advanced_config": {
        "timeout": 30,
        "retry_count": 3,
        "retry_delay": 1
    }
}'
```

**Response**:

```json
{
    "data": {
        "id": "mcp-00000001",
        "tenant_id": 1,
        "name": "天气查询服务",
        "description": "提供全球天气信息查询",
        "enabled": true,
        "transport_type": "sse",
        "url": "https://mcp.example.com/weather/sse",
        "headers": {
            "X-Custom-Header": "value"
        },
        "auth_config": {
            "api_key": "weather-api-key-xxxxx"
        },
        "advanced_config": {
            "timeout": 30,
            "retry_count": 3,
            "retry_delay": 1
        },
        "is_builtin": false,
        "created_at": "2025-08-12T10:00:00+08:00",
        "updated_at": "2025-08-12T10:00:00+08:00"
    },
    "success": true
}
```

**Example: creating a stdio-type MCP service**:

```curl
curl --location 'http://localhost:8080/api/v1/mcp-services' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "name": "本地文件服务",
    "description": "通过 stdio 访问本地文件系统",
    "transport_type": "stdio",
    "stdio_config": {
        "command": "/usr/local/bin/mcp-file-server",
        "args": ["--root", "/data"]
    },
    "env_vars": {
        "MCP_LOG_LEVEL": "info"
    }
}'
```

## GET `/mcp-services` - Get the MCP Service List

Returns all MCP services configured in the current space.

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/mcp-services' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json'
```

**Response**:

```json
{
    "data": [
        {
            "id": "mcp-00000001",
            "tenant_id": 1,
            "name": "天气查询服务",
            "description": "提供全球天气信息查询",
            "enabled": true,
            "transport_type": "sse",
            "url": "https://mcp.example.com/weather/sse",
            "headers": {},
            "auth_config": {
                "api_key": "weather-api-key-xxxxx"
            },
            "advanced_config": {
                "timeout": 30,
                "retry_count": 3,
                "retry_delay": 1
            },
            "is_builtin": false,
            "created_at": "2025-08-12T10:00:00+08:00",
            "updated_at": "2025-08-12T10:00:00+08:00"
        },
        {
            "id": "mcp-00000002",
            "tenant_id": 1,
            "name": "本地文件服务",
            "description": "通过 stdio 访问本地文件系统",
            "enabled": true,
            "transport_type": "stdio",
            "headers": {},
            "auth_config": null,
            "advanced_config": null,
            "stdio_config": {
                "command": "/usr/local/bin/mcp-file-server",
                "args": ["--root", "/data"]
            },
            "env_vars": {
                "MCP_LOG_LEVEL": "info"
            },
            "is_builtin": false,
            "created_at": "2025-08-12T11:00:00+08:00",
            "updated_at": "2025-08-12T11:00:00+08:00"
        }
    ],
    "success": true
}
```

## GET `/mcp-services/:id` - Get MCP Service Details

**Path parameters**:

| Field | Type   | Description           |
| ---- | ------ | -------------- |
| id   | string | MCP service ID    |

> Note: for built-in (`is_builtin: true`) services, sensitive credential fields are hidden in the response.

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/mcp-services/mcp-00000001' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json'
```

**Response**:

```json
{
    "data": {
        "id": "mcp-00000001",
        "tenant_id": 1,
        "name": "天气查询服务",
        "description": "提供全球天气信息查询",
        "enabled": true,
        "transport_type": "sse",
        "url": "https://mcp.example.com/weather/sse",
        "headers": {},
        "auth_config": {
            "api_key": "weather-api-key-xxxxx"
        },
        "advanced_config": {
            "timeout": 30,
            "retry_count": 3,
            "retry_delay": 1
        },
        "is_builtin": false,
        "created_at": "2025-08-12T10:00:00+08:00",
        "updated_at": "2025-08-12T10:00:00+08:00"
    },
    "success": true
}
```

## PUT `/mcp-services/:id` - Update an MCP Service

Supports partial field updates; any subset of the following can be passed: `name`, `description`, `enabled`, `transport_type`, `url`, `stdio_config`, `env_vars`, `headers`, `auth_config`, `advanced_config`. If `url` is provided, SSRF security validation is performed again.

**Request**:

```curl
curl --location --request PUT 'http://localhost:8080/api/v1/mcp-services/mcp-00000001' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "name": "天气查询服务（更新）",
    "description": "提供全球天气信息查询，支持实时数据",
    "enabled": false
}'
```

**Response**:

```json
{
    "data": {
        "id": "mcp-00000001",
        "tenant_id": 1,
        "name": "天气查询服务（更新）",
        "description": "提供全球天气信息查询，支持实时数据",
        "enabled": false,
        "transport_type": "sse",
        "url": "https://mcp.example.com/weather/sse",
        "headers": {},
        "auth_config": {
            "api_key": "weather-api-key-xxxxx"
        },
        "advanced_config": {
            "timeout": 30,
            "retry_count": 3,
            "retry_delay": 1
        },
        "is_builtin": false,
        "created_at": "2025-08-12T10:00:00+08:00",
        "updated_at": "2025-08-12T12:00:00+08:00"
    },
    "success": true
}
```

## DELETE `/mcp-services/:id` - Delete an MCP Service

**Request**:

```curl
curl --location --request DELETE 'http://localhost:8080/api/v1/mcp-services/mcp-00000001' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json'
```

**Response**:

```json
{
    "success": true,
    "message": "MCP service deleted successfully"
}
```

## POST `/mcp-services/:id/test` - Test MCP Service Connectivity

The backend establishes an MCP connection using the saved configuration and returns the connection result along with the discovered tools/resources. If the connection fails, the HTTP status is still 200, but `data.success` is `false`, with the error reason in `data.message`.

**Request**:

```curl
curl --location --request POST 'http://localhost:8080/api/v1/mcp-services/mcp-00000001/test' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json'
```

**Response**:

```json
{
    "data": {
        "success": true,
        "message": "连接成功",
        "description": "提供全球天气信息查询",
        "tools": [
            {
                "name": "get_weather",
                "description": "获取指定城市的天气信息",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "city": {
                            "type": "string",
                            "description": "城市名称"
                        }
                    },
                    "required": ["city"]
                }
            }
        ],
        "resources": [
            {
                "uri": "weather://cities",
                "name": "城市列表",
                "description": "支持查询的城市列表",
                "mimeType": "application/json"
            }
        ]
    },
    "success": true
}
```

## GET `/mcp-services/:id/tools` - Get the MCP Service's Tool List

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/mcp-services/mcp-00000001/tools' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json'
```

**Response**:

```json
{
    "data": [
        {
            "name": "get_weather",
            "description": "获取指定城市的天气信息",
            "inputSchema": {
                "type": "object",
                "properties": {
                    "city": {
                        "type": "string",
                        "description": "城市名称"
                    }
                },
                "required": ["city"]
            }
        },
        {
            "name": "get_forecast",
            "description": "获取未来天气预报",
            "inputSchema": {
                "type": "object",
                "properties": {
                    "city": {
                        "type": "string",
                        "description": "城市名称"
                    },
                    "days": {
                        "type": "integer",
                        "description": "预报天数"
                    }
                },
                "required": ["city"]
            }
        }
    ],
    "success": true
}
```

## GET `/mcp-services/:id/resources` - Get the MCP Service's Resource List

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/mcp-services/mcp-00000001/resources' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json'
```

**Response**:

```json
{
    "data": [
        {
            "uri": "weather://cities",
            "name": "城市列表",
            "description": "支持查询的城市列表",
            "mimeType": "application/json"
        },
        {
            "uri": "weather://config",
            "name": "服务配置",
            "description": "当前服务配置信息",
            "mimeType": "application/json"
        }
    ],
    "success": true
}
```

## GET `/mcp-services/:id/tool-approvals` - List Tool Manual Approval Policies

Returns the persisted `require_approval` flag for each tool under this MCP service. Only tool records that have been explicitly configured in the database are returned; tools not appearing in the list do not require approval by default.

**Path parameters**:

| Field | Type   | Description        |
| ---- | ------ | ----------- |
| id   | string | MCP service ID |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/mcp-services/mcp-00000001/tool-approvals' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**:

```json
{
    "data": [
        {
            "tool_name": "delete_file",
            "require_approval": true,
            "updated_at": "2025-09-20T15:30:00+08:00"
        },
        {
            "tool_name": "get_weather",
            "require_approval": false,
            "updated_at": "2025-09-20T15:31:00+08:00"
        }
    ],
    "success": true
}
```

## PUT `/mcp-services/:id/tool-approvals/:tool_name` - Set a Tool's Manual Approval Policy

Sets/updates the manual approval requirement for a given tool under a specific MCP service. When `require_approval` is `true`, the Agent will block before calling this tool and generate a pending approval record, requiring the frontend to call `POST /agent/tool-approvals/:pending_id` to complete the approval.

**Path parameters**:

| Field       | Type   | Description                                                                |
| ---------- | ------ | ------------------------------------------------------------------- |
| id         | string | MCP service ID                                                         |
| tool_name  | string | Tool name (automatically URL-decoded by Gin; callers must URL-encode `%` and `/` in the name) |

**Request body**:

| Field              | Type    | Required | Description                                |
| ----------------- | ------- | ---- | ----------------------------------- |
| require_approval  | boolean | Yes   | Whether manual approval is required before this tool can execute    |

**Request**:

```curl
curl --location --request PUT 'http://localhost:8080/api/v1/mcp-services/mcp-00000001/tool-approvals/delete_file' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "require_approval": true
}'
```

**Response**:

```json
{
    "success": true
}
```

## POST `/agent/tool-approvals/:pending_id` - Handle a Pending Tool Call Approval

Used for scenarios where the Agent blocks and waits for manual approval during execution: when the Agent hits a tool with `require_approval = true`, a `pending_id` is generated; after the frontend obtains this ID, it calls this endpoint to relay the approval result back to the Agent, allowing it to continue execution (or terminate).

**Authentication requirement**: the request context must have an authenticated user (`user_id`), and that user must be the owner of the current pending session; both the space and user levels undergo fail-close validation.

**Path parameters**:

| Field        | Type   | Description                |
| ----------- | ------ | ------------------- |
| pending_id  | string | Pending approval record ID       |

**Request body**:

| Field           | Type   | Required | Description                                                                                                              |
| -------------- | ------ | ---- | ----------------------------------------------------------------------------------------------------------------- |
| decision       | string | Yes   | Approval decision, must be `approve` or `reject`                                                                            |
| modified_args  | object | No   | Only effective when `approve`, allows manual modification of the tool call's parameters for this invocation; must be a non-null JSON object, otherwise returns 400                    |
| reason         | string | No   | Approval reason (freeform, for audit purposes)                                                                                        |

**Request (approve)**:

```curl
curl --location --request POST 'http://localhost:8080/api/v1/agent/tool-approvals/pending-abcdef123456' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "decision": "approve",
    "modified_args": {
        "path": "/tmp/safe-target.txt"
    },
    "reason": "已确认目标路径安全"
}'
```

**Request (reject)**:

```curl
curl --location --request POST 'http://localhost:8080/api/v1/agent/tool-approvals/pending-abcdef123456' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "decision": "reject",
    "reason": "目标路径在受保护目录"
}'
```

**Response**:

```json
{
    "success": true
}
```

**Error code reference**:

| HTTP | Trigger condition                                                                                |
| ---- | --------------------------------------------------------------------------------------- |
| 400  | `decision` is neither `approve` nor `reject`; or `modified_args` is `null`/not an object; or space/user mismatch |
| 401  | No authenticated user in context (middleware did not inject `user_id`)                                            |
| 404  | `pending_id` does not exist or has already been completed (consumed earlier by timeout/cancellation)                                    |
