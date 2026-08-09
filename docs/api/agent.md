# Agent Management API

[Back to index](./README.md)

## Overview

The Agent API is used to manage Custom Agents. The system provides built-in agents while also supporting user-created custom agents to meet different business scenario needs.

> Agent sharing and cross-organization distribution (`/agents/:id/shares`, etc.) are organizational collaboration capabilities documented in [Organization Management API](./organization.md). This document only covers the CRUD, copy, placeholders, type presets, and suggested questions endpoints for agents themselves.

### Built-in Agents

The system provides the following built-in agents by default:

| ID | Name | Description | Mode |
|----|------|------|------|
| `builtin-quick-answer` | Quick Answer | RAG-based Q&A over the knowledge base, delivering fast and accurate answers | quick-answer |
| `builtin-smart-reasoning` | Smart Reasoning | ReAct reasoning framework, supporting multi-step thinking and tool calls | smart-reasoning |
| `builtin-data-analyst` | Data Analyst | Professional data analysis agent, supporting SQL queries and statistical analysis over CSV/Excel files | smart-reasoning |

### Agent Modes

| Mode | Description |
|------|------|
| `quick-answer` | RAG mode, quick Q&A, generates answers directly from knowledge base retrieval results |
| `smart-reasoning` | ReAct mode, supporting multi-step reasoning and tool calls |

## API List

| Method | Path                       | Description                 |
| ------ | -------------------------- | ---------------------------- |
| POST   | `/agents`                  | Create an agent              |
| GET    | `/agents`                  | Get the agent list           |
| GET    | `/agents/:id`              | Get agent details            |
| PUT    | `/agents/:id`              | Update an agent              |
| DELETE | `/agents/:id`              | Delete an agent              |
| POST   | `/agents/:id/copy`         | Copy an agent                |
| GET    | `/agents/placeholders`     | Get placeholder definitions  |

---

## POST `/agents` - Create an Agent

Creates a new custom agent. Returns HTTP 201 on success.

**Request body parameters**:

| Parameter     | Type   | Required | Description                                       |
| ------------- | ------ | -------- | -------------------------------------------------- |
| `name`        | string | Yes      | Agent name                                         |
| `description` | string | No       | Agent description                                  |
| `avatar`      | string | No       | Avatar (emoji or icon name)                        |
| `config`      | object | No       | Agent configuration, see [Configuration Parameters](#configuration-parameters) |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/agents' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "name": "我的智能体",
    "description": "自定义智能体描述",
    "avatar": "🤖",
    "config": {
        "agent_mode": "smart-reasoning",
        "system_prompt": "你是一个专业的助手...",
        "temperature": 0.7,
        "max_iterations": 10,
        "kb_selection_mode": "all",
        "web_search_enabled": true,
        "multi_turn_enabled": true,
        "history_turns": 5
    }
}'
```

**Response**:

```json
{
    "success": true,
    "data": {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "name": "我的智能体",
        "description": "自定义智能体描述",
        "avatar": "🤖",
        "is_builtin": false,
        "tenant_id": 1,
        "created_by": "user-123",
        "config": {
            "agent_mode": "smart-reasoning",
            "system_prompt": "你是一个专业的助手...",
            "temperature": 0.7,
            "max_iterations": 10
        },
        "created_at": "2025-01-19T10:00:00Z",
        "updated_at": "2025-01-19T10:00:00Z"
    }
}
```

**Error responses**:

| Status Code | Error Code | Error                  | Description                              |
| ----------- | ---------- | ----------------------- | ------------------------------------------ |
| 400         | 1000       | Bad Request             | Invalid request parameters or empty agent name |
| 500         | 1007       | Internal Server Error   | Internal server error                      |

---

## GET `/agents` - Get the Agent List

Gets all agents in the current space, including built-in agents and custom agents. The response additionally returns `disabled_own_agent_ids`, indicating the list of the current space's own agent IDs that have been actively hidden from the frontend's conversation dropdown (this does not affect other spaces).

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/agents' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**:

```json
{
    "success": true,
    "data": [
        {
            "id": "builtin-quick-answer",
            "name": "快速问答",
            "description": "基于知识库的 RAG 问答，快速准确地回答问题",
            "avatar": "💬",
            "is_builtin": true,
            "tenant_id": 10000,
            "created_by": "",
            "config": {
                "agent_mode": "quick-answer",
                "temperature": 0.3,
                "max_completion_tokens": 2048,
                "kb_selection_mode": "all",
                "web_search_enabled": false,
                "multi_turn_enabled": true,
                "history_turns": 5
            },
            "created_at": "2025-12-29T20:06:01.696308+08:00",
            "updated_at": "2025-12-29T20:06:01.696308+08:00",
            "deleted_at": null
        },
        {
            "id": "550e8400-e29b-41d4-a716-446655440000",
            "name": "我的智能体",
            "is_builtin": false,
            "config": {
                "agent_mode": "smart-reasoning"
            }
        }
    ],
    "disabled_own_agent_ids": []
}
```

**Error responses**:

| Status Code | Error Code | Error                  | Description         |
| ----------- | ---------- | ----------------------- | --------------------- |
| 401         | 1001       | Unauthorized             | Missing space context |
| 500         | 1007       | Internal Server Error   | Internal server error |

---

## GET `/agents/:id` - Get Agent Details

Gets the detailed information of an agent by ID.

**Path parameters**:

| Parameter | Type   | Description |
| --------- | ------ | ------------ |
| `id`      | string | Agent ID     |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/agents/builtin-quick-answer' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**:

```json
{
    "success": true,
    "data": {
        "id": "builtin-quick-answer",
        "name": "快速问答",
        "description": "基于知识库的 RAG 问答，快速准确地回答问题",
        "is_builtin": true,
        "tenant_id": 1,
        "config": {
            "agent_mode": "quick-answer",
            "system_prompt": "",
            "context_template": "请根据以下参考资料回答用户问题...",
            "temperature": 0.7,
            "max_completion_tokens": 2048,
            "kb_selection_mode": "all",
            "web_search_enabled": true,
            "multi_turn_enabled": true,
            "history_turns": 5
        },
        "created_at": "2025-01-01T00:00:00Z",
        "updated_at": "2025-01-01T00:00:00Z"
    }
}
```

**Error responses**:

| Status Code | Error Code | Error                  | Description         |
| ----------- | ---------- | ----------------------- | --------------------- |
| 400         | 1000       | Bad Request             | Empty agent ID        |
| 404         | 1003       | Not Found               | Agent does not exist  |
| 500         | 1007       | Internal Server Error   | Internal server error |

---

## PUT `/agents/:id` - Update an Agent

Updates an agent's name, description, avatar, and configuration. Built-in agents cannot be modified.

**Path parameters**:

| Parameter | Type   | Description |
| --------- | ------ | ------------ |
| `id`      | string | Agent ID     |

**Request body parameters**:

| Parameter     | Type   | Required | Description        |
| ------------- | ------ | -------- | -------------------- |
| `name`        | string | No       | Agent name           |
| `description` | string | No       | Agent description    |
| `avatar`      | string | No       | Agent avatar          |
| `config`      | object | No       | Agent configuration   |

**Request**:

```curl
curl --location --request PUT 'http://localhost:8080/api/v1/agents/550e8400-e29b-41d4-a716-446655440000' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "name": "更新后的智能体",
    "description": "更新后的描述",
    "config": {
        "agent_mode": "smart-reasoning",
        "temperature": 0.8,
        "max_iterations": 20
    }
}'
```

**Response**:

```json
{
    "success": true,
    "data": {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "name": "更新后的智能体",
        "description": "更新后的描述",
        "config": {
            "agent_mode": "smart-reasoning",
            "temperature": 0.8,
            "max_iterations": 20
        },
        "updated_at": "2025-01-19T11:00:00Z"
    }
}
```

**Error responses**:

| Status Code | Error Code | Error                  | Description                              |
| ----------- | ---------- | ----------------------- | ------------------------------------------ |
| 400         | 1000       | Bad Request             | Invalid request parameters or empty agent name |
| 403         | 1002       | Forbidden               | Cannot modify a built-in agent             |
| 404         | 1003       | Not Found               | Agent does not exist                       |
| 500         | 1007       | Internal Server Error   | Internal server error                      |

---

## DELETE `/agents/:id` - Delete an Agent

Deletes the specified custom agent. Built-in agents cannot be deleted.

**Path parameters**:

| Parameter | Type   | Description |
| --------- | ------ | ------------ |
| `id`      | string | Agent ID     |

**Request**:

```curl
curl --location --request DELETE 'http://localhost:8080/api/v1/agents/550e8400-e29b-41d4-a716-446655440000' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**:

```json
{
    "success": true,
    "message": "Agent deleted successfully"
}
```

**Error responses**:

| Status Code | Error Code | Error                  | Description             |
| ----------- | ---------- | ----------------------- | -------------------------- |
| 400         | 1000       | Bad Request             | Empty agent ID              |
| 403         | 1002       | Forbidden               | Cannot delete a built-in agent |
| 404         | 1003       | Not Found               | Agent does not exist        |
| 500         | 1007       | Internal Server Error   | Internal server error       |

---

## POST `/agents/:id/copy` - Copy an Agent

Copies the specified agent, creating a new copy that is always a custom agent. Supports copying built-in agents. Returns HTTP 201 on success.

**Path parameters**:

| Parameter | Type   | Description       |
| --------- | ------ | ------------------- |
| `id`      | string | Source agent ID     |

**Request**:

```curl
curl --location --request POST 'http://localhost:8080/api/v1/agents/builtin-smart-reasoning/copy' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**:

```json
{
    "success": true,
    "data": {
        "id": "660e8400-e29b-41d4-a716-446655440001",
        "name": "智能推理 (副本)",
        "description": "ReAct 推理框架，支持多步思考和工具调用",
        "is_builtin": false,
        "config": {
            "agent_mode": "smart-reasoning",
            "max_iterations": 50
        },
        "created_at": "2025-01-19T12:00:00Z",
        "updated_at": "2025-01-19T12:00:00Z"
    }
}
```

**Error responses**:

| Status Code | Error Code | Error                  | Description         |
| ----------- | ---------- | ----------------------- | --------------------- |
| 400         | 1000       | Bad Request             | Empty agent ID        |
| 404         | 1003       | Not Found               | Agent does not exist  |
| 500         | 1007       | Internal Server Error   | Internal server error |

---

## GET `/agents/placeholders` - Get Placeholder Definitions

Gets all available prompt placeholder definitions, grouped by field type. These placeholders can be used in system prompts and context templates.

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/agents/placeholders' \
--header 'X-API-Key: your_api_key'
```

**Response**:

```json
{
    "success": true,
    "data": {
        "all": [...],
        "system_prompt": [...],
        "agent_system_prompt": [...],
        "context_template": [...],
        "rewrite_system_prompt": [...],
        "rewrite_prompt": [...],
        "fallback_prompt": [...]
    }
}
```

---

## Configuration Parameters

The agent's `config` object supports the following configuration items:

### Basic Settings

| Parameter | Type | Default | Description |
|------|------|--------|------|
| `agent_mode` | string | - | Agent mode: `quick-answer` (RAG) or `smart-reasoning` (ReAct) |
| `system_prompt` | string | - | System prompt, supports the use of placeholders |
| `system_prompt_id` | string | - | System prompt template ID (references a template in the `prompt_templates/` YAML files) |
| `context_template` | string | - | Context template (used only in quick-answer mode) |
| `context_template_id` | string | - | Context template ID (references a template in the `prompt_templates/` YAML files) |

### Model Settings

| Parameter | Type | Default | Description |
|------|------|--------|------|
| `model_id` | string | - | Conversation model ID |
| `rerank_model_id` | string | - | Rerank model ID |
| `temperature` | float | 0.7 | Temperature parameter (0-1) |
| `max_completion_tokens` | int | 2048 | Maximum number of tokens to generate |
| `thinking` | *bool | nil | Whether to enable thinking mode (applies to models that support extended thinking) |

### Agent Mode Settings

| Parameter | Type | Default | Description |
|------|------|--------|------|
| `max_iterations` | int | 10 | Maximum number of ReAct iterations |
| `allowed_tools` | []string | - | List of tools allowed for use |
| `mcp_selection_mode` | string | - | MCP service selection mode: `all`/`selected`/`none` |
| `mcp_services` | []string | - | List of selected MCP service IDs |
| `skills_selection_mode` | string | - | Skills selection mode: `all`/`selected`/`none` |
| `selected_skills` | []string | - | List of selected Skill names (when mode is `selected`) |

### Knowledge Base Settings

| Parameter | Type | Default | Description |
|------|------|--------|------|
| `kb_selection_mode` | string | - | Knowledge base selection mode: `all`/`selected`/`none` |
| `knowledge_bases` | []string | - | List of associated knowledge base IDs |
| `retrieve_kb_only_when_mentioned` | bool | false | Only retrieve from the knowledge base when explicitly mentioned by the user via @ |
| `supported_file_types` | []string | - | Supported file types (e.g. `["csv", "xlsx"]`) |

### Image Upload / Multimodal Settings

| Parameter | Type | Default | Description |
|------|------|--------|------|
| `image_upload_enabled` | bool | false | Whether to allow image uploads |
| `vlm_model_id` | string | - | VLM model ID used for image analysis |
| `image_storage_provider` | string | - | Image storage provider: `local`/`minio`/`cos`/`tos`/`oss`; uses the global default if left empty |

### FAQ Strategy Settings

| Parameter | Type | Default | Description |
|------|------|--------|------|
| `faq_priority_enabled` | bool | true | FAQ priority strategy switch |
| `faq_direct_answer_threshold` | float | 0.9 | FAQ direct-answer threshold |
| `faq_score_boost` | float | 1.2 | FAQ score boost coefficient |

### Web Search Settings

| Parameter | Type | Default | Description |
|------|------|--------|------|
| `web_search_enabled` | bool | true | Whether to enable web search |
| `web_search_max_results` | int | 5 | Maximum number of web search results |
| `web_search_provider_id` | string | - | Web search provider ID; uses the space's default provider if left empty |
| `web_fetch_enabled` | bool | false | Whether to automatically fetch the full text of reranked search result pages |
| `web_fetch_top_n` | int | 3 | Maximum number of pages to fetch full text for after reranking |

### Multi-turn Conversation Settings

| Parameter | Type | Default | Description |
|------|------|--------|------|
| `multi_turn_enabled` | bool | true | Whether to enable multi-turn conversation |
| `history_turns` | int | 5 | Number of historical turns to retain |

### Retrieval Strategy Settings

| Parameter | Type | Default | Description |
|------|------|--------|------|
| `embedding_top_k` | int | 10 | Vector retrieval TopK |
| `keyword_threshold` | float | 0.3 | Keyword retrieval threshold |
| `vector_threshold` | float | 0.5 | Vector retrieval threshold |
| `rerank_top_k` | int | 5 | Rerank TopK |
| `rerank_threshold` | float | 0.5 | Rerank threshold |

### Suggested Questions Settings

`question_suggestions` is a unified strategy owned by the agent. Channels such as web embeds can only toggle its display off — they cannot override the content or generation rules.

| Parameter | Type | Default | Description |
|------|------|--------|------|
| `question_suggestions.starters.enabled` | bool | true | Whether to show starter questions before the first query |
| `question_suggestions.starters.mode` | string | `hybrid` | `curated`, `knowledge`, or `hybrid` |
| `question_suggestions.starters.items` | []string | `[]` | Operator-configured starter questions |
| `question_suggestions.starters.count` | int | 6 | Number to display, range 1-8 |
| `question_suggestions.follow_ups.enabled` | bool | false | Whether to asynchronously generate follow-up questions after each complete answer |
| `question_suggestions.follow_ups.mode` | string | `hybrid` | `generated`, `knowledge`, or `hybrid` |
| `question_suggestions.follow_ups.count` | int | 3 | Number to generate, range 1-5 |
| `question_suggestions.follow_ups.model_id` | string | - | Dedicated generation model; uses the current turn's conversation model if left empty |
| `question_suggestions.follow_ups.categories` | []string | `clarify,deepen,action` | Allowed question types |
| `question_suggestions.follow_ups.max_context_turns` | int | 2 | Number of recent conversation turns used for generation, range 1-5 |
| `question_suggestions.follow_ups.additional_instruction` | string | - | Additional generation requirements from the agent's author |
| `question_suggestions.follow_ups.suppress_on_fallback` | bool | true | Do not display after a fallback answer |
| `question_suggestions.follow_ups.suppress_when_answer_asks_question` | bool | true | Do not display when the answer itself ends with a question |
| `question_suggestions.follow_ups.knowledge_fallback` | bool | true | Use knowledge base candidates as a fallback when the model fails |
| `question_suggestions.follow_ups.allow_regenerate` | bool | false | Whether to allow the user to request a new batch |

The legacy `suggested_prompts` field is written into `starters.items` in a one-time database migration; the API no longer accepts this field.

### Advanced Settings

| Parameter | Type | Default | Description |
|------|------|--------|------|
| `enable_query_expansion` | bool | true | Whether to enable query expansion |
| `enable_rewrite` | bool | true | Whether to enable multi-turn conversation query rewriting |
| `rewrite_prompt_system` | string | - | Rewrite system prompt |
| `rewrite_prompt_user` | string | - | Rewrite user prompt template |
| `fallback_strategy` | string | `model` | Fallback strategy: `fixed` (fixed reply) or `model` (model-generated); defaults to `model` on the server side if not set |
| `fallback_response` | string | - | Fixed fallback reply (used when `fallback_strategy` is `fixed`) |
| `fallback_prompt` | string | - | Fallback prompt (used when `fallback_strategy` is `model`) |

---

## Using an Agent for Q&A

After creating or obtaining an agent, you can use the `/agent-chat/:session_id` endpoint to perform Q&A with the agent. See [Chat API](./chat.md) for details.

Use the `agent_id` parameter in the Q&A request to specify which agent to use:

```curl
curl --location 'http://localhost:8080/api/v1/agent-chat/session-123' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "query": "帮我分析一下这份数据",
    "agent_enabled": true,
    "agent_id": "builtin-data-analyst"
}'
```

## Related Documentation

- Agent organizational sharing, cross-space distribution, and disabling (`/agents/:id/shares`, `/shared-agents`, etc.): see [Organization Management API](./organization.md)
- Binding an agent to IM channels (`/agents/:id/im-channels`): see the organization/IM channel documentation
- Web search provider configuration (referenced by `web_search_provider_id`): see [Web Search API](./web-search.md)
