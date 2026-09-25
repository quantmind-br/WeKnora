# DeepSeek Harness Plugin

The DeepSeek Harness (dsh) plugin connects to an existing WeKnora deployment through the REST API and supports retrieving source passages, reading complete documents and generating answers with citations. The plugin source lives in `packages/dsh-weknora`.

## Installation and connection

```sh
dsh plugin --profile web add @wxg-prc-cpg/dsh-weknora

export WEKNORA_BASE_URL=https://weknora.example.com
export WEKNORA_API_KEY='<api-key>'
export WEKNORA_KNOWLEDGE_BASE_IDS=kb-123,kb-456
export WEKNORA_RESOURCE_URLS=public
dsh web
```

You can also run `dsh plugin --profile web add ./packages/dsh-weknora` from the repository root. The plugin itself requires Node ≥20.11; the dsh runtime verified in the repository needs a Node build with zstd (≥22.15 or ≥24) and pnpm ≥10. The verified versions are dsh 0.1.0-rc.8 / WeKnora 0.8.0; for other versions, confirm compatibility against their plugin interface.

The service address may include `/api/v1`; it is appended automatically when missing. The API Key is sent via `X-API-Key` and needs the `retrieve` capability; calling `ask` also needs `chat`. When using a platform Key, configure `tenantId`, which requests send as `X-Tenant-ID`.

## Four tools

| Tool | Purpose |
| --- | --- |
| `weknora_list_knowledge_bases` | List visible knowledge bases and their IDs |
| `weknora_search` | Hybrid retrieval of passages that also matches document titles; returns the knowledge ID, chunk index and score |
| `weknora_read_document` | Read a document's ordered body; the first page includes the title/summary, and long documents can be paged |
| `weknora_ask` | Create or continue a session in WeKnora, returning the answer, citations and the server-side tool steps |

`search` returns retrieval evidence for dsh to generate an answer; `ask` calls the model on the WeKnora server and creates a session and messages, which suits synthesizing multiple sources. Neither modifies knowledge base content.

When no knowledge base scope is specified, the plugin resolves all knowledge bases visible to the credential, caches them in-process and shares them between search/ask. With agentId configured, ask goes through the agent flow, and the knowledge scope is resolved on the server by that agent.

## Profile configuration

Set the following in `$DSH_HOME/profiles/<name>/cordis.patch.yml`:

```yaml
- id: weknora
  config:
    baseUrl: https://weknora.example.com
    apiKey: !!js process.env.WEKNORA_API_KEY
    knowledgeBaseIds: [kb-product-docs]
    agentId: ''
    maxResults: 8
    maxChunkChars: 1200
    requestTimeoutMs: 30000
    chatTimeoutMs: 300000
    resourceUrls: public
    toolPrefix: weknora
    tools:
      listKnowledgeBases: true
      search: true
      readDocument: true
      ask: true
```

A configuration update replaces that entry's `config` as a whole, so include every field you want to keep. All `tools.*` are enabled by default. When connecting multiple deployments, set a different `toolPrefix` for each; the name must start with a lowercase letter and contain only lowercase letters, digits and underscores.

## Cited images and troubleshooting

`resourceUrls` defaults to `public`, which asks the server to convert resource handles into loadable links. When an API Key scoped to specific knowledge bases receives 403, the plugin automatically switches to `handle` and keeps that mode for subsequent calls. `resource://` handles cannot be loaded directly in a browser; for external link deployment requirements, see [File Access](../03-features/21-file-access.md).

| Symptom | Check |
| --- | --- |
| Configuration error at load time | Fix baseUrl, parameter ranges and toolPrefix according to the per-field errors |
| Retrieval 401/403 | Whether the Key is valid, has the retrieve capability and is authorized for the target space/knowledge base |
| ask is rejected | Whether the Key also has chat, and whether a platform Key provides tenantId |
| No direct image links | Whether it is a knowledge-base-scoped Key, or the deployment has not configured accessible external links |
| Long questions time out | Adjust chatTimeoutMs and check WeKnora's model calls |

## Implementation reference

`packages/dsh-weknora/src/`, `test/fixtures/api-contract.json`, `contract/contract_test.go`.
