# Built-in Model Management Guide

## Overview

Built-in models are system-level model configurations, visible to all spaces. Regular space users (including space admins) see sensitive information hidden, and cannot modify it; system administrators can edit configuration and credentials in the "Model Management" interface. Built-in models are typically used to provide the system's default model configuration, ensuring all spaces can use a unified model service.

## Built-in Model Features

- **Visible to all spaces**: Built-in models are visible to all spaces, no individual configuration required
- **Security protection**: API Keys and other credentials are never returned in plaintext; only system administrators can see management information such as Base URL and whether credentials are configured
- **Permission protection**: Regular space users can only view; system administrators can edit configuration and credentials
- **Unified management**: Built-in models take effect for all spaces; deletion is still managed via YAML or SQL to avoid conflicts between deployment configuration and runtime state

## Editing in the Admin Interface

System administrators can directly click a built-in model in "Settings → Model Management" to modify model parameters or update credentials. Regular space admins can still only view.

The first time a YAML-managed built-in model is saved through the admin interface, that row switches to runtime management (`managed_by` is cleared), and subsequent application startups will no longer overwrite it with the YAML entry of the same ID. This ensures that results saved via the interface remain in effect after a restart. To hand it back to YAML management, the deployment administrator needs to restore that row's `managed_by` to `yaml` in the database, or remove the runtime override and restart.

## How to Add Built-in Models

WeKnora supports two ways to add built-in models: the **recommended** approach uses declarative YAML configuration (automatically applied idempotently), while direct SQL insertion is retained as a compatibility path.

### Method One (Recommended): YAML Configuration File

#### File Location

The default path is `config/builtin_models.yaml` (in the same directory as `config.yaml` and `builtin_agents.yaml`). To mount it elsewhere, set the environment variable `BUILTIN_MODELS_CONFIG=/absolute/path/builtin_models.yaml` to override.

If the file doesn't exist, it will be skipped at startup without error; parsing failures are only logged as a Warning and don't affect the main process. Every application startup re-reads the file and UPSERTs into the `models` table by the `id` field (preserving `created_at`, refreshing other fields). If a model with the same ID has already been saved through the interface by a system administrator and switched to runtime management, the runtime configuration is preserved and no longer overwritten by YAML.

#### Schema

```yaml
builtin_models:
  - id: <required, stable, UPSERT key>
    tenant_id: <int, default 10000>          # aligned with the tenants_id_seq starting point
    name: <string>
    type: KnowledgeQA | Embedding | Rerank | VLLM | ASR
    source: <string, default "remote">       # local | remote | aliyun | ...
    description: <string, optional>
    is_default: <bool, default false>
    status: <string, default "active">
    parameters:
      base_url: <string>
      api_key: <string, supports ${ENV_VAR}>
      provider: <string>                     # openai | generic | moonshot | ...
      embedding_parameters:                  # Embedding type only
        dimension: <int>
        truncate_prompt_tokens: <int>
```

#### Full Example

```yaml
builtin_models:
  - id: builtin-openai-chat
    name: gpt-4o-mini
    type: KnowledgeQA
    source: remote
    description: OpenAI default chat model
    is_default: false
    parameters:
      base_url: https://api.openai.com/v1
      api_key: ${OPENAI_API_KEY}
      provider: openai

  - id: builtin-openai-embeddings
    name: text-embedding-3-small
    type: Embedding
    source: remote
    parameters:
      base_url: https://api.openai.com/v1
      api_key: ${OPENAI_API_KEY}
      provider: openai
      embedding_parameters:
        dimension: 1536
        truncate_prompt_tokens: 0

  - id: builtin-rerank
    name: bge-reranker-v2-m3
    type: Rerank
    source: remote
    parameters:
      base_url: ${RERANK_BASE_URL}
      api_key: ${RERANK_API_KEY}
      provider: generic
```

#### `${ENV}` Interpolation

Any **string** field such as `api_key` / `base_url` / `name` can reference an environment variable: `${OPENAI_API_KEY}` will be replaced at startup by the corresponding `os.Getenv("OPENAI_API_KEY")`.

- If the environment variable exists → it's replaced with the actual value
- If the environment variable doesn't exist → the literal `${OPENAI_API_KEY}` string is **preserved** (so a 401 error makes it immediately obvious the env wasn't set, making it easier to troubleshoot)
- Shell-style expansions like `${VAR:-default}` are not supported, matching the interpolation behavior already implemented for `config.yaml`
- **Non-string fields cannot be env-ified** (e.g. `type`, `dimension`, `is_default`), since they must be parsed as their target YAML type

#### How Env Variables Get Into the Container

The `app` service in `docker-compose.yml` already has this preconfigured:

```yaml
env_file:
  - path: .env
    required: false
```

This means you write the variable values into the `.env` file at the project root, and they're automatically passed through to the container at startup. There's **no need** to pass them through individually in the `environment:` block. `required: false` ensures the container can still start even if `.env` doesn't exist (accommodating a fresh clone from upstream).

The repository's `.env.example` reserves a **Built-in Models** comment section at the top, listing reference variable names for LLM / Embedding / Rerank as a starting point; after copying `.env.example` to `.env`, just uncomment and fill in the values. The variable names are decided by the YAML itself — the reference section is just a common template, not a reserved keyword.

Complete end-to-end example:

`.env`
```bash
LLM_MODEL_NAME=gpt-4o-mini
LLM_BASE_URL=https://api.openai.com/v1
LLM_API_KEY=sk-...
LLM_PROVIDER=openai
```

`config/builtin_models.yaml`
```yaml
builtin_models:
  - id: builtin-llm-default
    type: KnowledgeQA
    is_default: true
    name: ${LLM_MODEL_NAME}
    parameters:
      base_url: ${LLM_BASE_URL}
      api_key: ${LLM_API_KEY}
      provider: ${LLM_PROVIDER}
```

Startup:
```bash
docker compose up -d
```

#### Verifying After Startup

```bash
docker compose logs app | grep -E 'Built-in models? config'
```

You'll see something like:

```
Built-in model upserted: id=builtin-openai-chat name=gpt-4o-mini type=KnowledgeQA
Built-in model upserted: id=builtin-openai-embeddings name=text-embedding-3-small type=Embedding
Built-in models config applied: 2 entries from /app/config/builtin_models.yaml.
```

#### Docker Deployment

Mount the file in the `volumes` block of the `app` service in `docker-compose.yml`:

```yaml
services:
  app:
    volumes:
      - ./config/builtin_models.yaml:/app/config/builtin_models.yaml:ro
```

The repository provides `config/builtin_models.yaml.example` as a starting point; copy it to `config/builtin_models.yaml` and modify as needed.

### Method Two: Direct SQL Insertion

Supported providers: `generic` (custom), `openai`, `aliyun`, `zhipu`, `volcengine`, `hunyuan`, `deepseek`, `minimax`, `mimo`, `siliconflow`, `jina`, `openrouter`, `requesty`, `gemini`, `modelscope`, `moonshot`, `qianfan`, `qiniu`, `longcat`, `gpustack`

```sql
-- Example: LLM built-in model
INSERT INTO models (
    id, tenant_id, name, type, source, description,
    parameters, is_default, status, is_builtin
) VALUES (
    'builtin-llm-001',
    10000,
    'gpt-4o-mini',
    'KnowledgeQA',
    'remote',
    'System built-in LLM model',
    '{"base_url": "https://api.openai.com/v1", "api_key": "sk-xxx", "provider": "openai"}'::jsonb,
    false,
    'active',
    true
) ON CONFLICT (id) DO NOTHING;

-- Embedding
INSERT INTO models (
    id, tenant_id, name, type, source, description,
    parameters, is_default, status, is_builtin
) VALUES (
    'builtin-embedding-001',
    10000,
    'text-embedding-3-small',
    'Embedding',
    'remote',
    'System built-in Embedding model',
    '{"base_url": "https://api.openai.com/v1", "api_key": "sk-xxx", "provider": "openai", "embedding_parameters": {"dimension": 1536, "truncate_prompt_tokens": 0}}'::jsonb,
    false,
    'active',
    true
) ON CONFLICT (id) DO NOTHING;

-- Rerank
INSERT INTO models (
    id, tenant_id, name, type, source, description,
    parameters, is_default, status, is_builtin
) VALUES (
    'builtin-rerank-001',
    10000,
    'bge-reranker-v2-m3',
    'Rerank',
    'remote',
    'System built-in Rerank model',
    '{"base_url": "https://api.jina.ai/v1", "api_key": "jina-xxx", "provider": "jina"}'::jsonb,
    false,
    'active',
    true
) ON CONFLICT (id) DO NOTHING;
```

### Verifying the Insertion Result

```sql
SELECT id, name, type, is_builtin, status
FROM models
WHERE is_builtin = true
ORDER BY type, created_at;
```

## Setting an Existing Model as a Built-in Model

If you've already manually created a regular model and want to upgrade it to a built-in model:

```sql
UPDATE models
SET is_builtin = true
WHERE id = 'model ID';
```

## Removing Built-in Models

**Just delete the entry from YAML.** At application startup, rows in the `models` table that are YAML-managed and no longer declared in YAML are automatically soft-deleted — you no longer need to manually run SQL.

How it works: every row written by YAML is tagged with `managed_by = 'yaml'`. At restart, the loader proceeds in two steps:

1. UPSERT all entries in the current YAML (idempotent by `id`, including resetting a previously soft-deleted `deleted_at` back to NULL — meaning removing an entry from YAML and adding it back is equivalent to "reviving" it)
2. Soft-delete rows where `is_builtin = true AND managed_by = 'yaml' AND id NOT IN (the set of ids currently in YAML)`

**Rows manually inserted via SQL as builtin (`managed_by = ''`) are never touched by the loader**, and are completely isolated from YAML.

### Additional Notes for the Manual Path

If you're managing via the SQL path (`managed_by = ''`), removal still follows the old method:

```sql
-- Unmark builtin, revert to a regular model
UPDATE models SET is_builtin = false WHERE id = 'model ID';

-- Or delete directly
DELETE FROM models WHERE id = 'model ID';
```

### Emergency Shutdown of YAML Takeover

If you accidentally modified YAML and want to immediately disable the takeover without clearing the file, the fastest method is: point the environment variable `BUILTIN_MODELS_CONFIG` to a nonexistent path and restart — the loader will see the file is missing and simply no-op, **including skipping the drift sweep**, and the already-written YAML-managed rows remain unchanged.

## Notes

1. **ID naming convention**: it's recommended to use the format `builtin-{type}-{slug}`, e.g. `builtin-openai-chat`, `builtin-rerank`
2. **Space ID**: built-in models can belong to any space, defaulting to `10000` (matching the `tenants_id_seq` starting point)
3. **YAML and SQL coexistence**: both methods can be used simultaneously; the loader only touches rows with `managed_by='yaml'`; builtin rows inserted via SQL are completely invisible to the loader
4. **`is_default` single guarantee**: when an entry in YAML is marked `is_default: true`, the loader first sets other default models under the same `(tenant_id, type)` to `false`, preventing the "one default model per type" semantics maintained by the API path from being broken
5. **Restart to take effect**: after modifying YAML, `docker compose restart app` is enough to apply the new configuration
6. **Encryption**: the API Key is stored encrypted within the `parameters` JSONB (if `SYSTEM_AES_KEY` is configured); if not configured, it falls back to a plaintext-compatible path
7. **Security**: the frontend automatically hides the API Key and Base URL of built-in models, but the raw data still exists in the database — please safeguard database access properly
8. **Self-protection against parse errors**: if YAML parsing fails, the loader only logs a warning and skips the reconcile step, and **will not** perform a drift sweep, ensuring that an accidental YAML edit doesn't cause mass soft-deletion of existing built-in models
