# API Reference: Long-term Memory

Manage the current caller's long-term memories, topics and document preferences, as well as the space-level memory configuration. Paths use the `/api/v1` prefix.

All personal endpoints require Viewer+, and an API Key must be full-access. The scope is determined from the credential; an arbitrary `subject_id` is not accepted. In the examples, `$BASE` is the service address and `$TOKEN` is the current user's Bearer token.

## Space configuration and request switches

Space configuration uses `GET/PUT /tenants/kv/memory-config`, not the tenant name/description update endpoint. Reads require Viewer+, writes require Admin+, and an API Key needs manage_tenant_settings or full-access. The response is `{success,data:MemoryConfig}`, and PUT takes the configuration object directly; the personal `PUT /memory/settings` cannot replace the space switch.

```bash
curl -X PUT "$BASE/api/v1/tenants/kv/memory-config" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"enabled":true,"write_mode":"explicit_only","max_items":200}'
```

`memory_config` fields: `enabled`, `write_mode` (explicit_only/auto), `extract_model_id`, `max_items`, `extract_delay_seconds`, `extract_min_interval_seconds`, `extract_instructions`, `interest_threshold`, `embedding_model_id`, `vector_recall`, `retrieval_conditioning`. For semantics, see [Long-term Memory](../03-features/23-memory.md). When updating, submit the complete configuration object you want to keep.

When `CustomAgentConfig.memory_enabled` is omitted it inherits the space setting; false forbids this agent from using memory. IM/Embed use the bound agent's configuration; the current channel structure has no separate memory_enabled field.

## Personal settings

| Method | Path | Request / Response |
| --- | --- | --- |
| GET | `/memory/settings` | `{success,data:{workspace_enabled,user_enabled,effective,write_mode,item_count,max_items}}` |
| PUT | `/memory/settings` | `{"enabled":true}`, enabled is required; returns the updated settings |

`effective` is the combined result of the space and personal switches; a given chat is also constrained by the agent switch.

```bash
curl -X PUT "$BASE/api/v1/memory/settings" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"enabled":true}'
```

## Items

| Method | Path | Request / Response |
| --- | --- | --- |
| GET | `/memory/items` | Optional status, limit, offset; `{success,data:[MemoryItem],total}` |
| POST | `/memory/items` | `{kind,content,importance}`; 200 `{success,data:MemoryItem}` |
| PUT | `/memory/items/:id` | `{content,importance}`; 200 `{success,data:MemoryItem}` |
| DELETE | `/memory/items/:id` | 200 `{"success":true}` |
| POST | `/memory/items/:id/confirm` | 200 `{success,data:MemoryItem}`; returns 409 when the inference is stale or its basis has been modified |
| POST | `/memory/items/:id/reject` | 200 `{"success":true}` |
| DELETE | `/memory/items` | Clears the current identity; 200 `{success,removed}` |

`status` can be active, pending, superseded or archived; omit it to skip filtering. `limit` defaults to 50 with a valid range of 1–200, and out-of-range values fall back to 50; `offset` defaults to 0, and negative values become zero. kind is profile/preference/fact/task/interest; content is a short memory of at most 300 characters, and importance is used for importance ranking.

MemoryItem includes `id`, `kind`, `content`, `topic`, `importance`, `origin`, `status`, `source_session_id`, `source_message_id`, `expires_at`, `superseded_by` and the created/updated timestamps. Pending items are not used in prompts; after an edit, an item is treated as manually maintained.

For a pending inference that modifies an existing memory, the old item stays in effect until confirmation; confirming activates the inference and replaces the old item in the same transaction. An inference that is stale, expired, or whose underlying content has been modified/deleted cannot be confirmed and returns 409; the client should refresh the list. When the content consists almost entirely of sensitive information such as credentials, adding or editing returns 400.

```bash
curl -X POST "$BASE/api/v1/memory/items" \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/json' \
  -d '{"kind":"preference","content":"Give the conclusion first, then explain the reasoning","importance":3}'

curl "$BASE/api/v1/memory/items?status=pending&limit=50&offset=0" \
  -H "Authorization: Bearer $TOKEN"

curl -X POST "$BASE/api/v1/memory/items/item-1/confirm" \
  -H "Authorization: Bearer $TOKEN"
```

## Topics and document preferences

| Method | Path | Description |
| --- | --- | --- |
| GET | `/memory/topics` | Topics being tracked but not yet promoted; limit/offset as for items, returns data and total |
| POST | `/memory/topics/:id/promote` | Manually promote to a long-term interest; returns `{success,data:MemoryItem}` |
| DELETE | `/memory/topics/:id` | Stop tracking the topic; returns success |
| GET | `/memory/documents` | Document preferences; limit/offset as for items, returns data and total |
| DELETE | `/memory/documents/:id` | Delete the preference record; returns success and does not delete the knowledge base document |

Topic fields include id/topic/aliases/hits/threshold/last_seen_at; Document fields include id/knowledge_id/knowledge_base_id/title/hits/last_used_at. Deletion uses the preference record id.

```bash
curl "$BASE/api/v1/memory/topics" -H "Authorization: Bearer $TOKEN"
curl -X POST "$BASE/api/v1/memory/topics/topic-1/promote" \
  -H "Authorization: Bearer $TOKEN"
curl -X DELETE "$BASE/api/v1/memory/documents/affinity-1" \
  -H "Authorization: Bearer $TOKEN"
```

## Export and tidy up now

`GET /memory/export` returns `{success,total,truncated,data}` with the download file name `weknora-memories.json`. At most 20,000 items are exported; check truncated when the limit is reached.

`POST /memory/consolidate` returns `{success,data:{merged,demoted,expired,reviewed,candidates,skipped?}}` and immediately merges near-duplicate items and archives expired tasks. When nothing changes, skipped explains why.

```bash
curl "$BASE/api/v1/memory/export" -H "Authorization: Bearer $TOKEN" \
  -o weknora-memories.json
curl -X POST "$BASE/api/v1/memory/consolidate" -H "Authorization: Bearer $TOKEN"
```

Invalid parameters, sensitive content or memory not enabled return 400; an item not found for the current identity returns 404; a confirmation conflict returns 409; unmet authentication/permissions return 401/403. The API has no subject parameter for admins to read other people's memories. Implementation: `internal/handler/memory.go`, `internal/router/routes_memory.go`.
