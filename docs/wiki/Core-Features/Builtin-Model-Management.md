---
title: Built-in Model Management
tags: [Core Features, Models, System Administration, Built-in Services]
aliases: [Built-in Models, BuiltinModel, BUILTIN_MODELS]
source: BUILTIN_MODELS.md
---

# Built-in Model Management Guide

## Overview

Built-in models are system-level model configurations that are visible to all spaces, but sensitive information is hidden, and they cannot be edited or deleted. Built-in models are typically used to provide the system's default model configuration, ensuring that all spaces can use a unified model service.

> Also part of the built-in system configuration: [Built-in MCP Service Management](Builtin-MCP-Service-Management.md)

## Built-in Model Features

- **Visible to all spaces**: no separate configuration needed
- **Security protection**: API Key and Base URL are hidden
- **Read-only protection**: cannot be edited or deleted, can only be set as the default model
- **Unified management**: maintained centrally by the system administrator

## How to Add a Built-in Model

### 1. Prepare the Model Data

- Model name (name)
- Model type (type): `KnowledgeQA`, `Embedding`, `Rerank`, or `VLLM`
- Model source (source): `local` or `remote`
- Model parameters (parameters): including base_url, api_key, provider, etc.
- Space ID (tenant_id): it is recommended to use a space ID smaller than 10000

**Supported providers (provider)**: `generic`, `openai`, `aliyun`, `zhipu`, `volcengine`, `hunyuan`, `deepseek`, `minimax`, `mimo`, `siliconflow`, `jina`, `openrouter`, `requesty`, `gemini`, `modelscope`, `moonshot`, `qianfan`, `qiniu`, `longcat`, `gpustack`

### 2. Run the SQL Insert Statement

```sql
-- LLM built-in model
INSERT INTO models (
    id, tenant_id, name, type, source, description, parameters,
    is_default, status, is_builtin
) VALUES (
    'builtin-llm-001', 10000, 'GPT-4', 'KnowledgeQA', 'remote',
    '内置 LLM 模型',
    '{"base_url": "https://api.openai.com/v1", "api_key": "sk-xxx", "provider": "openai"}'::jsonb,
    false, 'active', true
) ON CONFLICT (id) DO NOTHING;

-- Embedding built-in model
INSERT INTO models (...) VALUES (...) ON CONFLICT (id) DO NOTHING;

-- Rerank built-in model
INSERT INTO models (...) VALUES (...) ON CONFLICT (id) DO NOTHING;
```

### 3. Verification

```sql
SELECT id, name, type, is_builtin, status
FROM models WHERE is_builtin = true ORDER BY type, created_at;
```

## Notes

1. **ID naming convention**: it is recommended to use the `builtin-{type}-{sequence number}` format
2. **Space ID**: it is recommended to use the first space ID (usually 10000)
3. **Parameter format**: the `parameters` field must be valid JSON
4. **Idempotency**: use `ON CONFLICT (id) DO NOTHING`
5. **Security**: API Key and Base URL are automatically hidden on the frontend

## Model Configuration and FAQ

- If the Embedding model is not configured correctly, document uploads will fail. See [FAQ](../Operations-Troubleshooting/FAQ.md) for troubleshooting steps
- The `provider` in the model parameters determines the API call format and authentication method

---

## Backlinks

- [Home](../Home.md) — Wiki home navigation
- [Built-in MCP Service Management](../Core-Features/Builtin-MCP-Service-Management.md) — Also part of the built-in system configuration, similar pattern
- [FAQ](../Operations-Troubleshooting/FAQ.md) — Troubleshooting related to model configuration
- [Roadmap](../Project-Overview/Version-Roadmap.md) — Model training direction in the roadmap
