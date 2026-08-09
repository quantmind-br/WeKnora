# Go SDK

The official WeKnora Go SDK lives in the repository's `client/` directory. It is a standalone Go module that wraps CRUD operations for all major resources under the WeKnora server's `/api/v1/*` API, along with SSE streaming chat capabilities. The server itself and the official CLI (`weknora`) are both built on top of this SDK.

## Installation

The SDK's module path is defined in `client/go.mod`:

```
module github.com/Tencent/WeKnora/client

go 1.24.2
```

Install with:

```bash
go get github.com/Tencent/WeKnora/client
```

Import:

```go
import "github.com/Tencent/WeKnora/client"
```

## Initialization and Authentication

The core types and constructor function are defined in `client/client.go`.

### The Client struct

```go
type Client struct {
    baseURL       string
    httpClient    *http.Client
    streamTimeout time.Duration
    apiKey        string
    bearerToken   string
    tenantID      *uint64
}
```

Create an instance via `NewClient(baseURL string, options ...ClientOption) *Client`. The default timeout for regular requests is 30 seconds; streaming (SSE) requests have **no timeout** by default — their lifecycle is controlled by the `context` (unless `WithTimeout` is explicitly called).

### ClientOption overview

| Option | Description |
|---|---|
| `WithAPIKey(key string)` | Sets a long-lived API key, sent via the `X-API-Key` header |
| `WithBearerToken(token string)` | Sets a short-lived JWT, sent via the `Authorization: Bearer <token>` header (typically used after a successful `Login`) |
| `WithToken(token string)` | **Deprecated**: a v0.x-compatible alias for `WithAPIKey`, to be removed in the next major version |
| `WithTimeout(timeout time.Duration)` | Sets the timeout ceiling for both regular requests and streaming requests |
| `WithTransport(rt http.RoundTripper)` | Replaces the underlying `http.RoundTripper` (for retry/telemetry/signing middleware, etc.); passing `nil` restores `http.DefaultTransport` |
| `WithTenantID(tenantID uint64)` | Attaches an `X-Tenant-ID` header to every request, intended only for explicit cross-tenant access by principals with `CanAccessAllTenants` permission |

### Authentication methods

The SDK supports two types of credentials, which can be configured simultaneously; at the HTTP layer, `X-API-Key` takes priority:

- **API Key** (long-lived): `WithAPIKey`, sent via the `X-API-Key` header;
- **Bearer JWT** (short-lived): `WithBearerToken`, sent via the `Authorization: Bearer <token>` header, used together with `Login` / `RefreshToken` / `GetCurrentUser` in `client/auth.go`.

A typical JWT login flow (corresponding to `POST /api/v1/auth/login`):

```go
c := client.NewClient("http://localhost:8080")
loginResp, err := c.Login(ctx, client.LoginRequest{ /* email + password */ })
// Then rebuild an authenticated client using the returned access token
authed := client.NewClient("http://localhost:8080",
    client.WithBearerToken(loginResp.AccessToken))
```

### Tenant and header injection

`applyAuthHeaders` (`client/client.go`) automatically injects the following into every request:

- `X-API-Key` / `Authorization` (depending on configuration);
- `X-Request-ID`: read from `ctx.Value("RequestID")` (a string), used for trace correlation;
- `X-Tenant-ID`: priority is given to the `"TenantID"` value in the context (supporting `uint64`, `*uint64`, or a numeric string) over the client-level default set via `WithTenantID`.

Example of overriding the tenant for a single request:

```go
tenantID := uint64(10000)
ctx := context.WithValue(context.Background(), "TenantID", &tenantID)
kb, err := apiClient.GetKnowledgeBase(ctx, kbID)
```

Note: JWTs and tenant-level API keys already carry tenant identity, so regular users **should not** set `X-Tenant-ID` (the server-side auth middleware performs cross-tenant validation on bearer requests carrying this header, and regular users will get a 403).

### Raw escape hatch

`Client.Raw(ctx, method, path, body)` (Experimental) issues an arbitrary HTTP request directly, using the authentication headers already configured on the client. It's intended for one-off integrations and for pass-through use by the `weknora api` CLI; prefer the typed methods whenever one is available.

## Resource and Method Overview

The following are all public methods of `Client`; internal methods (`buildRequest`, `doRequest`, `doRequestStream`, `processAgentSSEStream`, etc.) are not listed.

### Auth — `client/auth.go`

| Method | Description |
|---|---|
| `Login` | Email/password login, returns JWT access/refresh tokens |
| `GetCurrentUser` | Retrieves the current logged-in principal and tenant info (`GET /api/v1/auth/me`) |
| `RefreshToken` | Exchanges a refresh token for a new access token |

### KnowledgeBase — `client/knowledgebase.go`

| Method | Description |
|---|---|
| `CreateKnowledgeBase` | Creates a knowledge base |
| `GetKnowledgeBase` | Gets knowledge base details |
| `ListKnowledgeBases` | Lists knowledge bases |
| `UpdateKnowledgeBase` | Updates a knowledge base |
| `DeleteKnowledgeBase` | Deletes a knowledge base |
| `ClearKnowledgeBaseContents` | Clears a knowledge base's contents |
| `HybridSearch` | Performs hybrid search (vector + keyword) within a knowledge base |
| `TogglePinKnowledgeBase` | Pins/unpins a knowledge base |
| `ListMoveTargets` | Lists knowledge bases that knowledge can be moved to |
| `CopyKnowledgeBase` | Copies a knowledge base |
| `DuplicateKnowledgeBase` | Duplicates a knowledge base |
| `GetKBCloneProgress` | Queries the progress of a clone task |

### Knowledge — `client/knowledge.go`

| Method | Description |
|---|---|
| `CreateKnowledgeFromFile` | Creates knowledge by uploading a local file (multipart, supports metadata, multimodal toggles, custom filename, channel, and parsing-config overrides) |
| `CreateKnowledgeFromURL` | Creates knowledge from a URL |
| `GetKnowledge` | Gets knowledge details |
| `GetKnowledgeBatch` | Gets multiple pieces of knowledge in batch |
| `ListKnowledge` | Lists knowledge with pagination |
| `ListKnowledgeWithFilter` | Lists knowledge with filter conditions |
| `DeleteKnowledge` | Deletes knowledge |
| `DownloadKnowledgeFile` | Downloads a knowledge item's original file to a local path |
| `OpenKnowledgeFile` | Opens a knowledge item's original file as a stream (returns the filename plus an `io.ReadCloser`) |
| `UpdateKnowledge` | Updates knowledge |
| `ReparseKnowledge` | Re-parses knowledge |
| `CancelKnowledgeParse` | Cancels a parsing task |
| `GetKnowledgeProcessingSpans` | Gets the processing pipeline spans for a knowledge item |
| `UpdateImageInfo` | Updates image info |
| `CreateManualKnowledge` | Creates manually-authored knowledge |
| `UpdateManualKnowledge` | Updates manually-authored knowledge |
| `FilterKnowledge` | Filters knowledge by keyword/file type/agent |
| `MoveKnowledge` | Moves knowledge across knowledge bases |
| `GetKnowledgeMoveProgress` | Queries the progress of a move task |
| `PreviewKnowledgeFile` | Previews a knowledge file (returns the raw `*http.Response`) |
| `BatchUpdateKnowledgeTags` | Batch-updates knowledge tags |

### Chunk — `client/chunk.go`

| Method | Description |
|---|---|
| `ListKnowledgeChunks` | Lists a knowledge item's chunks with pagination |
| `UpdateChunk` | Updates a chunk's content/enabled state |
| `DeleteChunk` | Deletes a chunk |
| `GetChunkByIDOnly` | Gets a chunk by chunk ID alone |
| `DeleteGeneratedQuestion` | Deletes a question generated for a chunk |
| `DeleteChunksByKnowledgeID` | Deletes all chunks belonging to a knowledge item |

### Session — `client/session.go`

| Method | Description |
|---|---|
| `CreateSession` | Creates a session |
| `GetSession` | Gets a session |
| `GetSessionsByTenant` | Lists a tenant's sessions with pagination |
| `UpdateSession` | Updates a session |
| `DeleteSession` | Deletes a session |
| `BatchDeleteSessions` | Batch-deletes sessions |
| `GenerateTitle` | Generates a session title |
| `KnowledgeQAStream` | Knowledge Q&A (SSE streaming, see below) |
| `ContinueStream` | Resumes an in-progress stream (for reconnection scenarios) |
| `StopSession` | Stops generation of a given assistant message |
| `SearchKnowledge` | Knowledge retrieval |

### Message — `client/message.go`, `client/message_suggestion.go`

| Method | Description | Source file |
|---|---|---|
| `LoadMessages` | Loads messages by time | `client/message.go` |
| `GetRecentMessages` | Gets the most recent N messages | `client/message.go` |
| `GetMessagesBefore` | Gets messages before a given point in time | `client/message.go` |
| `SearchMessages` | Searches historical messages | `client/message.go` |
| `GetChatHistoryKBStats` | Chat history statistics broken down by knowledge base | `client/message.go` |
| `DeleteMessage` | Deletes a message | `client/message.go` |
| `EnsureMessageSuggestions` | Ensures (optionally force-regenerates) suggested questions | `client/message_suggestion.go` |
| `GetMessageSuggestions` | Gets a message's suggested questions | `client/message_suggestion.go` |
| `RecordMessageSuggestionEvent` | Reports click/impression events for suggested questions | `client/message_suggestion.go` |

### Agent chat (streaming) — `client/agent.go`

| Method | Description |
|---|---|
| `AgentQAStream` | Agent-mode streaming Q&A (Deprecated, simplified entry point) |
| `AgentQAStreamWithRequest` | Agent-mode streaming Q&A (full `AgentQARequest` payload) |
| `NewAgentSession` | Creates an `AgentSession` wrapper (exposes `Ask` / `AskWithRequest` / `GetSessionID`) |

### Agent management — `client/agent_manage.go`

| Method | Description |
|---|---|
| `CreateAgent` | Creates a custom Agent |
| `ListAgents` | Lists Agents |
| `GetAgent` | Gets an Agent |
| `UpdateAgent` | Updates an Agent |
| `DeleteAgent` | Deletes an Agent |
| `CopyAgent` | Copies an Agent |
| `GetAgentPlaceholders` | Gets an Agent's configuration placeholders |
| `GetSuggestedQuestions` | Gets an Agent's suggested questions |

### Model — `client/model.go`

| Method | Description |
|---|---|
| `CreateModel` | Creates a model |
| `GetModel` | Gets a model |
| `ListModels` | Lists models |
| `UpdateModel` | Updates a model |
| `DeleteModel` | Deletes a model |
| `ListModelProviders` | Lists model providers by model type |

### Tenant — `client/tenant.go`

| Method | Description |
|---|---|
| `CreateTenant` | Creates a tenant |
| `GetTenant` | Gets a tenant |
| `UpdateTenant` | Updates a tenant |
| `DeleteTenant` | Deletes a tenant |
| `ListTenants` | Lists tenants |
| `ListAllTenants` | Lists all tenants (admin) |
| `SearchTenants` | Searches tenants (paginated) |
| `ListTenantAPIKeys` | Lists a tenant's API keys |
| `CreateTenantAPIKey` | Creates a tenant API key |
| `DeleteTenantAPIKey` | Deletes a tenant API key |
| `GetTenantKV` | Reads tenant-level KV configuration |
| `UpdateTenantKV` | Updates tenant-level KV configuration |
| `GetAPIPrincipalConfig` | Gets API principal configuration |
| `UpdateAPIPrincipalConfig` | Updates API principal configuration |
| `CreateAPIPrincipalTestToken` | Creates a test token for an API principal |

### Organization and Sharing — `client/organization.go`

| Method | Description |
|---|---|
| `CreateOrganization` / `ListMyOrganizations` / `GetOrganization` / `UpdateOrganization` / `DeleteOrganization` | Organization CRUD |
| `SearchOrganizations` / `PreviewOrganizationByInviteCode` | Search organizations / preview an organization by invite code |
| `JoinOrganizationByInviteCode` / `SubmitJoinRequest` / `JoinByOrganizationID` / `LeaveOrganization` / `RequestRoleUpgrade` | Join/leave organizations, request role upgrades |
| `GenerateInviteCode` / `SearchUsersForInvite` / `InviteMember` | Member invitations |
| `ListOrgMembers` / `UpdateMemberRole` / `RemoveMember` | Member management |
| `ListJoinRequests` / `ReviewJoinRequest` | Join-request approval |
| `ShareKnowledgeBase` / `ListKBShares` / `UpdateSharePermission` / `RemoveKBShare` | Knowledge base sharing |
| `ShareAgent` / `ListAgentShares` / `RemoveAgentShare` | Agent sharing |
| `ListOrgShares` / `ListOrgAgentShares` / `ListSharedKnowledgeBases` / `ListSharedAgents` | Queries for shared resources |

### FAQ — `client/faq.go`

| Method | Description |
|---|---|
| `ListFAQEntries` | Lists FAQ entries with pagination |
| `UpsertFAQEntries` | Batch-creates/updates FAQ entries |
| `CreateFAQEntry` | Creates a single FAQ entry |
| `GetFAQEntry` | Gets a single FAQ entry |
| `UpdateFAQEntry` | Updates a single FAQ entry |
| `AddSimilarQuestions` | Adds similar questions |
| `UpdateFAQEntryFieldsBatch` | Batch-updates fields |
| `UpdateFAQEntryTagBatch` | Batch-updates tags |
| `DeleteFAQEntries` | Batch-deletes entries |
| `SearchFAQEntries` | Searches FAQ entries |
| `ExportFAQEntries` | Exports as CSV (returns `[]byte`) |
| `GetFAQImportProgress` | Queries progress of an async import task (including dry runs) |
| `UpdateLastFAQImportResultDisplayStatus` | Updates the display status of the most recent import result |

### Tag — `client/tag.go`

| Method | Description |
|---|---|
| `ListTags` | Lists tags |
| `CreateTag` | Creates a tag |
| `UpdateTag` / `UpdateTagBySeqID` | Updates a tag (by ID / by seq ID) |
| `DeleteTag` / `DeleteTagBySeqID` | Deletes a tag (by ID / by seq ID) |

### MCP Service — `client/mcp_service.go`

| Method | Description |
|---|---|
| `CreateMCPService` / `ListMCPServices` / `GetMCPService` / `UpdateMCPService` / `DeleteMCPService` | MCP service CRUD |
| `TestMCPService` | Connectivity test |
| `GetMCPServiceTools` / `GetMCPServiceResources` | Lists MCP tools/resources |
| `ResolveToolApproval` | Handles tool-call approval |

### Initialization and Model Detection — `client/initialization.go`

| Method | Description |
|---|---|
| `GetInitializationConfig` / `InitializeByKB` / `UpdateKBConfig` / `SetKBModelConfig` | Knowledge base initialization and model configuration |
| `CheckOllamaStatus` / `ListOllamaModels` / `CheckOllamaModels` | Ollama status and model detection |
| `DownloadOllamaModel` / `GetOllamaDownloadProgress` / `ListOllamaDownloadTasks` | Ollama model download tasks |
| `CheckRemoteModel` / `TestEmbeddingModel` / `CheckRerankModel` / `TestMultimodalFunction` | Connectivity checks for remote LLM / Embedding / Rerank / multimodal models |
| `ExtractTextRelations` | Text relation extraction test |

### System — `client/system.go`

| Method | Description |
|---|---|
| `GetSystemInfo` | Gets system info (version, etc.) |
| `ListParserEngines` / `CheckParserEngines` | Lists/checks document-parsing engines |
| `ReconnectDocReader` | Reconnects the DocReader service |
| `GetStorageEngineStatus` / `CheckStorageEngine` | Storage engine status/check |

### Others

| Method | Description | Source file |
|---|---|---|
| `StartEvaluation` / `GetEvaluationResult` | Starts an evaluation task / queries evaluation results | `client/evaluation.go` |
| `ListSkills` | Lists built-in Agent skills | `client/skill.go` |
| `GetWebSearchProviders` | Lists available web search providers | `client/web_search.go` |
| `Raw` | Raw HTTP escape hatch (Experimental) | `client/client.go` |

In total, roughly 170 public methods, covering about 20 resource categories.

## Streaming Chat (SSE)

The SDK's streaming interfaces use a **callback** mechanism rather than channels: internally, the SDK uses a `bufio.Scanner` to parse SSE line by line (`event:` / `data:` prefixes, with blank lines delimiting frames), invoking the callback once per parsed frame; if the callback returns a non-nil error, the stream is aborted. The SSE line-buffer limit has been raised to 4 MiB (`scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)`), to avoid large `references` frames triggering "token too long" errors.

Streaming requests go through `doRequestStream` (`client/client.go`) and are, by default, not subject to the 30-second timeout — the stream's lifecycle is controlled by the passed-in `ctx`.

### Knowledge Q&A stream: `KnowledgeQAStream` (`client/session.go`)

```go
func (c *Client) KnowledgeQAStream(
    ctx context.Context,
    sessionID string,
    request *KnowledgeQARequest,
    callback func(*StreamResponse) error,
) error
```

Each `StreamResponse` frame carries a `ResponseType` (`answer`, `references`, `thinking`, `tool_call`, `tool_result`, `error`, `reflection`, `session_title`, `agent_query`, `complete`), incremental `Content`, a `Done` end marker, and — on the `Done` frame — `KnowledgeReferences` (the cited sources).

### Agent Q&A stream: `AgentQAStreamWithRequest` (`client/agent.go`)

```go
type AgentEventCallback func(*AgentStreamResponse) error

func (c *Client) AgentQAStreamWithRequest(ctx context.Context,
    sessionID string, request *AgentQARequest, callback AgentEventCallback,
) error
```

`AgentQARequest` supports fields such as `KnowledgeBaseIDs`, `AgentID`, `WebSearchEnabled`, `MentionedItems` (@-mentions of knowledge bases/files/tags/MCP/skills), and `Images` (multimodal images). A convenience wrapper is also available:

```go
as := apiClient.NewAgentSession(session.ID)
err := as.Ask(ctx, "Tell me about WeKnora", func(ev *client.AgentStreamResponse) error {
    if ev.ResponseType == client.AgentResponseTypeAnswer {
        fmt.Print(ev.Content)
    }
    return nil
})
```

### Reconnection: `ContinueStream` (`client/session.go`)

`ContinueStream(ctx, sessionID, messageID, callback)` resumes a stream still being generated on the server side via `GET /api/v1/sessions/continue-stream/{sessionID}?message_id=...`, using the same callback mechanism as `KnowledgeQAStream`. Combined with `StopSession(ctx, sessionID, messageID)`, generation can be aborted.

## Error Handling

### HTTP layer: `APIError` (`client/client.go`)

All non-2xx responses are wrapped as `*APIError`. Use `errors.As` to branch on the HTTP status code or the server's structured error code:

```go
var apiErr *client.APIError
if errors.As(err, &apiErr) {
    switch {
    case apiErr.StatusCode == 404:
        // Resource not found
    case apiErr.Code == client.ServerErrUnauthorized: // 1001
        // Trigger re-login
    }
}
```

`Code` is the structured error code from the response body's `{"code":N}`; the package provides constants ranging from `ServerErrBadRequest` (1000) to `ServerErrValidation` (1010). `Error()` preserves the legacy `"HTTP error <status>: <body>"` format for compatibility with consumers doing string matching.

### Stream layer: `SSEStreamError` (`client/stream_errors.go`)

When the server emits a terminal error frame on an SSE stream (`response_type=error, done=true`), the SDK **first passes that frame to the callback**, then returns a `*SSEStreamError`:

```go
type SSEStreamError struct {
    Content string // Error frame content
}
```

Ways to check for it (both are equivalent; the former is recommended):

```go
// Option 1: sentinel error (SSEStreamError.Unwrap() returns it)
if errors.Is(err, client.ErrSSEStreamTerminal) { ... }

// Option 2: helper function (compatible with the older fmt.Errorf("SSE stream error: ...") chain)
if client.IsSSEStreamError(err) { ... }
```

## Logging and Tracing

`client/log.go` provides internal SDK debug logging based on `log/slog`, which writes to `io.Discard` by default (completely silent to consumers). Embedding applications can call, at startup:

```go
client.SetDebugLevel("debug") // "debug"/"info"/"warn"; any other value (including "error", "") is silent
```

Log output goes to stderr and includes trace information such as line-by-line SSE parsing and request failures. This function is **not concurrency-safe** and must be called once before any SDK calls are made.

For tracing, place `"RequestID"` (a string) into the context, and the SDK will automatically send it as the `X-Request-ID` header (see `applyAuthHeaders` in `client/client.go`):

```go
ctx := context.WithValue(context.Background(), "RequestID", "req-20260727-0001")
```

## Complete Examples

The following examples are adapted from real code in `client/example.go`.

### Example 1: Creating a knowledge base and uploading a file

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/Tencent/WeKnora/client"
)

func main() {
    apiClient := client.NewClient(
        "http://localhost:8080",
        client.WithAPIKey("your-api-key"),
        client.WithTimeout(30*time.Second),
    )

    // Create a knowledge base
    kb := &client.KnowledgeBase{
        Name:        "Test Knowledge Base",
        Description: "This is a test knowledge base",
        ChunkingConfig: client.ChunkingConfig{
            ChunkSize:    500,
            ChunkOverlap: 50,
            Separators:   []string{"\n\n", "\n", ". ", "? ", "! "},
        },
        EmbeddingModelID: "embedding_model_id",
        SummaryModelID:   "summary_model_id",
    }
    createdKB, err := apiClient.CreateKnowledgeBase(context.Background(), kb)
    if err != nil {
        fmt.Printf("Failed to create knowledge base: %v\n", err)
        return
    }
    fmt.Printf("Knowledge base created: ID=%s, Name=%s\n", createdKB.ID, createdKB.Name)

    // Upload a file to create knowledge
    metadata := map[string]string{"source": "local", "type": "document"}
    knowledge, err := apiClient.CreateKnowledgeFromFile(
        context.Background(), createdKB.ID, "path/to/sample.pdf",
        metadata, nil, "", "", nil)
    if err != nil {
        fmt.Printf("Failed to upload knowledge file: %v\n", err)
        return
    }
    fmt.Printf("File uploaded: Knowledge ID=%s, Title=%s\n", knowledge.ID, knowledge.Title)
}
```

### Example 2: Creating a session and running a streaming knowledge Q&A

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "strings"

    "github.com/Tencent/WeKnora/client"
)

func main() {
    apiClient := client.NewClient("http://localhost:8080",
        client.WithAPIKey("your-api-key"))

    // Create a session
    session, err := apiClient.CreateSession(context.Background(), &client.CreateSessionRequest{
        Title:       "Test Session",
        Description: "A test session for knowledge Q&A",
    })
    if err != nil {
        fmt.Printf("Failed to create session: %v\n", err)
        return
    }

    // Streaming Q&A: accumulate the answer and references
    question := "What is artificial intelligence?"
    var answer strings.Builder
    var references []*client.SearchResult

    err = apiClient.KnowledgeQAStream(context.Background(),
        session.ID,
        &client.KnowledgeQARequest{Query: question},
        func(response *client.StreamResponse) error {
            if response.ResponseType == client.ResponseTypeAnswer {
                answer.WriteString(response.Content)
            }
            if response.Done && len(response.KnowledgeReferences) > 0 {
                references = response.KnowledgeReferences
            }
            return nil
        })
    if err != nil {
        // Distinguish an SSE terminal error frame from other errors
        if errors.Is(err, client.ErrSSEStreamTerminal) {
            fmt.Printf("Stream terminated by server error: %v\n", err)
        } else {
            fmt.Printf("Q&A failed: %v\n", err)
        }
        return
    }
    fmt.Printf("Answer: %s\n", answer.String())
    for i, ref := range references {
        fmt.Printf("Reference %d: %s\n", i+1, ref.Content)
    }
}
```

### Example 3: Message history, chunk management, and resource cleanup

```go
package main

import (
    "context"
    "fmt"

    "github.com/Tencent/WeKnora/client"
)

func main() {
    apiClient := client.NewClient("http://localhost:8080",
        client.WithAPIKey("your-api-key"))
    ctx := context.Background()

    // Get the 10 most recent session messages
    sessionID := "your-session-id"
    messages, err := apiClient.GetRecentMessages(ctx, sessionID, 10)
    if err != nil {
        fmt.Printf("Failed to get session messages: %v\n", err)
    } else {
        for i, msg := range messages {
            fmt.Printf("%d. Role: %s, Content: %s\n", i+1, msg.Role, msg.Content)
        }
    }

    // Manage knowledge chunks: list with pagination and update the first one
    knowledgeID := "your-knowledge-id"
    chunks, total, err := apiClient.ListKnowledgeChunks(ctx, knowledgeID, 1, 10)
    if err != nil {
        fmt.Printf("Failed to get knowledge chunks: %v\n", err)
    } else {
        fmt.Printf("Knowledge has %d chunks, retrieved %d\n", total, len(chunks))
        if len(chunks) > 0 {
            updated, err := apiClient.UpdateChunk(ctx, knowledgeID, chunks[0].ID,
                &client.UpdateChunkRequest{
                    Content:   "Updated chunk content - " + chunks[0].Content,
                    IsEnabled: true,
                })
            if err != nil {
                fmt.Printf("Failed to update chunk: %v\n", err)
            } else {
                fmt.Printf("Chunk updated: ID=%s\n", updated.ID)
            }
        }
    }

    // Clean up resources
    if err := apiClient.DeleteSession(ctx, sessionID); err != nil {
        fmt.Printf("Failed to delete session: %v\n", err)
    }
    if err := apiClient.DeleteKnowledge(ctx, knowledgeID); err != nil {
        fmt.Printf("Failed to delete knowledge: %v\n", err)
    }
}
```

## Source Reference

- Client core and error types: `client/client.go`
- Authentication: `client/auth.go`
- Streaming Q&A: `client/session.go`, `client/agent.go`
- Streaming errors: `client/stream_errors.go`
- Logging: `client/log.go`
- Full usage examples: `client/example.go`
