# Cross-session Long-term Memory

Long-term memory stores the caller's profile, preferences, stable facts and long-term interests for use in later sessions. Data is isolated by workspace and caller identity: each space, IM user or embed visitor keeps its own personal memory.

## Enabling long-term memory {#enable-and-use}

1. A space Admin/Owner turns on the space switch under "Settings → Long-term memory"; it is off by default.
2. Choose "Explicit only" or "Distill automatically". In automatic mode you can specify an extraction model; leave it blank to use the model the conversation uses.
3. Users can turn off their own memory in personal memory management; it is not forced on them even when the space has it enabled.
4. State the preference to remember explicitly in a conversation, or add it by hand in memory management; then check in another session whether it is applied.
5. Review pending items regularly and confirm, edit or reject inferences; items that no longer apply can be deleted.

| Mode | Behavior |
| --- | --- |
| `explicit_only` | Only records what is explicitly asked to be remembered; no background LLM distillation runs |
| `auto` | Additionally extracts from conversations in the background, batching work by delay and minimum interval |

An agent's `memory_enabled=false` disables memory reads and writes for that agent alone; omitting it inherits the space setting. IM/Embed use the memory preference of the bound agent; an API Key must be full-access to use the personal memory management endpoints. The feature is governed jointly by the space, personal and per-request switches.

## Managing memories

| Action | Purpose |
| --- | --- |
| Add / Edit | Maintain profile, preferences, facts, tasks or interests directly; manual edits are no longer overwritten by background extraction |
| Confirm pending items | An inferred item goes from pending to usable; unconfirmed items are not injected into the prompt. If the inference modifies an existing memory, the old item stays in effect until confirmation and is replaced on confirm; confirmation fails if the inference is stale, in which case refresh the list |
| Reject / Delete | Revoke memories that are wrong or no longer needed; rejecting leaves a suppression record that reduces repeated extraction |
| Promote topic | Turn a repeatedly discussed topic into a long-term interest immediately, or stop tracking a topic |
| Document preferences | View repeatedly cited documents and remove personalized retrieval preferences you no longer need |
| Tidy up now | Merge near-duplicate items and archive expired tasks without waiting for background consolidation |
| Export / Clear | Download your memories as JSON, or clear your memory data |

Memory items are classified as profile, preference, fact, task and interest. A new fact on the same topic can supersede an old one. Turning off the personal switch pauses usage; deleting or clearing removes the data. For the category and status values used by the API, see the [Memory API](../04-api/02-api-memory.md).

## Effect on answers and retrieval

The system treats profile, preferences and interests as length-limited resident context, and recalls relevant facts and tasks based on the current question. Once an embedding model is configured, semantic recall ranks all of that identity's memories by relevance to the question, regardless of item importance; with PostgreSQL and pgvector enabled the ranking happens in the database, otherwise it is computed in the service. Smart reasoning can also actively search memories and past conversations. Memory only affects question understanding and source selection; it does not extend knowledge base access permissions.

With retrieval personalization enabled, the system uses topics to help understand the question and uses repeatedly cited documents to assist ranking. The conversation timeline shows memory-related steps. Background tasks keep an extraction cursor and pending state per session, so rapid consecutive questions, batch truncation or a service restart do not cause messages to be missed. If the model output for a segment of messages keeps failing to parse, that segment is skipped after at most three retry rounds and processing continues with later messages; skipped messages are not re-extracted automatically.

## Troubleshooting memory that does not take effect {#troubleshooting}

| Symptom | Check |
| --- | --- |
| Memory is not used | Space switch, personal switch, agent switch; check whether the space or identity changed |
| Automatic extraction does not appear | write_mode, extraction delay/minimum interval, model connection and background tasks |
| Inference does not affect answers | Whether it is still pending; it is used only after confirmation |
| A document should no longer be cited | Remove it in document preferences; knowledge base permissions are still controlled independently |
| Tidy up now merged nothing | Check the returned skipped reason; there may be too few items or no candidates |

## Space configuration

Space admins can maintain the memory switch, extraction mode and models on the settings page. The configuration is stored in `memory_config` and is read and updated through `GET/PUT /tenants/kv/memory-config`; for all fields, see the [Memory API](../04-api/02-api-memory.md).

| Name | Type | Default | Description |
| --- | --- | --- | --- |
| `enabled` | bool | `false` | Space switch |
| `write_mode` | string | `explicit_only` | `explicit_only` / `auto`; other values are treated as `explicit_only` |
| `extract_model_id` | string | empty | Extraction model; an empty value uses the conversation model |
| `extract_delay_seconds` | int | `90` | Delay extraction after an answer completes to batch multiple turns within a short time; range 5–3600 |
| `extract_min_interval_seconds` | int | `300` | Minimum interval between two extractions for the same person, bounding call frequency; messages within the interval are deferred; max 86400 |
| `extract_instructions` | string | empty | The space's own extraction rules, up to 1000 characters |
| `max_items` | int | `200` | Active memory cap per identity, max 2000; 0 uses the default |
| `interest_threshold` | int | `3` | Number of sessions a topic must appear in before it becomes an interest, max 20 |
| `embedding_model_id` | string | empty | Embedding model used for memory recall; when empty only lexical matching is done |
| `vector_recall` | bool | on when omitted | Whether to enable semantic recall when an embedding model is configured |
| `retrieval_conditioning` | bool | on when omitted | Allows memory to influence question understanding and document ranking |

The extraction model is used for background distillation, and the embedding model for matching the current question. The memory feature adds the corresponding model calls; when a model is referenced by the memory configuration, deleting it shows the dependency details.

## Implementation reference

- `internal/application/service/memory/`: scope, extraction, recall, topics, document preferences, consolidation
- `internal/handler/memory.go`, `internal/router/routes_memory.go`: personal API
- `internal/types/memory.go`: models, budgets, statuses and configuration
- `frontend/src/views/settings/MemorySettings.vue`, `MemoryWorkspaceSettings.vue`
