# Extension Points Guide

WeKnora provides clear extension points at nine levels: document parsing, chunking, retrieval, model integration, web search, data sources, IM channels, Agent tools, and object storage. This chapter presents each one in turn: **core interface definition (actual source code) → list of existing implementations → steps for adding a new implementation (including the registration point file)**. All interface code is excerpted from the current repository source.

## 0. Extension Points Overview

```mermaid
graph LR
    subgraph DR["docreader (Python)"]
        P1["Document Parser<br/>(parser/registry.py)"]
    end
    subgraph APP["app (Go, internal/)"]
        P2["Chunking Strategy<br/>(infrastructure/chunker)"]
        P3["Retrieval Engine<br/>(application/repository/retriever)"]
        P4["Model Provider<br/>(models/provider)"]
        P5["Web Search Engine<br/>(infrastructure/web_search)"]
        P6["Data Source Connector<br/>(datasource/connector)"]
        P7["IM Platform Adapter<br/>(im/adapter.go)"]
        P8["Agent Tools<br/>(agent/tools)"]
        P9["Storage Backend<br/>(application/service/file)"]
    end
    DOC["Raw Document"] --> P1
    P1 -->|"markdown + images"| P2
    P2 -->|"chunks"| P3
    P6 -->|"external content sync"| P1
    P7 -->|"IM messages"| AG["Agent Engine"]
    AG --> P8
    P8 --> P3
    P8 --> P5
    AG --> P4
    P1 -.->|"file read/write"| P9
    P2 -.-> P9
    CT["container.go<br/>(dependency injection / registration hub)"] -.->|"register"| P3
    CT -.->|"register"| P5
    CT -.->|"register"| P6
    CT -.->|"register"| P7
```

On the Go side, the **registration hub** for most extension points is `internal/container/container.go` (the dependency injection container): retrieval engines via `initRetrieveEngineRegistry()`, web search via `registerWebSearchProviders()`, IM adapters via `registerIMAdapterFactories()`, and data source connectors via `initConnectorRegistry()`.

---

## 1. Adding a New Document Parser (docreader, Python)

### Interface Definition

The base class is in `docreader/parser/base_parser.py`. After the lightweight refactor, `BaseParser` is only responsible for converting a document into markdown text + raw image references (chunking, image storage, OCR, and VLM captioning are all handled on the Go side):

```python
# docreader/parser/base_parser.py
class BaseParser(ABC):
    """Base parser interface."""

    def __init__(self, file_name: str = "", file_type: Optional[str] = None, **kwargs):
        self.file_name = file_name
        self.file_type = file_type or os.path.splitext(file_name)[1].lstrip(".")

    @abstractmethod
    def parse_into_text(self, content: bytes) -> Document:
        """Parse document content into markdown text.

        Returns:
            Document with ``content`` (markdown string) and optional
            ``images`` dict mapping storage-relative paths to base64 data.
        """
```

The return value `Document` (`docreader/models/document.py`, a pydantic model) has two core fields: `content: str` (markdown) and `images: Dict[str, str]` (path → base64).

### Registration Mechanism

The `ParserEngineRegistry` in `docreader/parser/registry.py` manages parsers via a two-level mapping of "engine name → {file extension → Parser class}"; when the requested engine doesn't support a given file type, it automatically falls back to the `builtin` engine. The default registry is built by `_build_default_registry()`, and the module-level singleton is `registry = _build_default_registry()`.

```python
# docreader/parser/registry.py (excerpt)
class ParserEngineRegistry:
    def register(self, name: str, file_types: Dict[str, Type[BaseParser]],
                 description: str = "", check_available: Callable = None,
                 unavailable_hint: str = ""): ...
    def get_parser_class(self, engine: str, file_type: str) -> Type[BaseParser]: ...
```

### Existing Implementations

| Engine | Parser | File |
| --- | --- | --- |
| `builtin` | `Docx2Parser` / `DocParser` / `PDFParser` / `MarkdownParser` / `ExcelParser` / `EPUBParser` / `HTMLParser` / `MHTMLParser` / `ImageParser` (jpg/png/gif/bmp/tiff/webp, etc.) | `docreader/parser/docx2_parser.py`, `doc_parser.py`, `pdf_parser.py`, `markdown_parser.py`, `excel_parser.py`, `epub_parser.py`, `html_parser.py`, `mhtml_parser.py`, `image_parser.py` |
| `markitdown` | `MarkitdownParser` (Microsoft MarkItDown, multi-format) | `docreader/parser/markitdown_parser.py` |
| `opendataloader` | `OpenDataLoaderParser` (PDF layout analysis, requires Java 11+, includes `check_available` detection) | `docreader/parser/opendataloader_parser.py` |

### Steps to Add One

1. Create `docreader/parser/my_parser.py`, inheriting from `BaseParser`, implementing `parse_into_text(content: bytes) -> Document`;
2. **Registration point: `docreader/parser/registry.py`** — add to `_build_default_registry()`:

```python
reg.register(
    "my_engine",
    {"myext": MyParser},
    description="My parsing engine",
    check_available=lambda overrides: (True, ""),   # optional: dependency availability check
    unavailable_hint="Hint shown to the user when a dependency is missing",
)
```

3. If you're just swapping in a new implementation for an existing file extension, you can instead simply add a line `"ext": MyParser` to the `builtin` mapping;
4. Add a unittest under `docreader/tests/` (see `test_parser_routing.py` for reference), and verify with `uv run python -m unittest`.

---

## 2. Adding a New Chunking Strategy (internal/infrastructure/chunker)

### Interface Definition

Chunking doesn't use an interface — instead it follows a pattern of **strategy tiers + package-level function-variable overrides**. The public entry point is in `internal/infrastructure/chunker/strategy.go`:

```go
// internal/infrastructure/chunker/strategy.go
// Strategy values for SplitterConfig.Strategy.
const (
    StrategyAuto      = "auto"
    StrategyHeading   = "heading"
    StrategyHeuristic = "heuristic"
    StrategyRecursive = "recursive"
    StrategyLegacy    = "legacy"
)

func Split(text string, cfg SplitterConfig) []Chunk
func SplitWithDiagnostics(text string, cfg SplitterConfig) ([]Chunk, *Diagnostics)
func SplitParentChild(text string, parentCfg, childCfg SplitterConfig) ParentChildResult
```

The config and result types are in `internal/infrastructure/chunker/splitter.go`:

```go
// internal/infrastructure/chunker/splitter.go
type Chunk struct {
    Content       string
    ContextHeader string
    Seq           int
    Start         int
    End           int
}

type SplitterConfig struct {
    ChunkSize    int
    ChunkOverlap int
    Separators   []string
    Strategy     string   // empty = legacy (for backward compatibility)
    TokenLimit   int      // limit chunk size by approximate token count, 0 = use ChunkSize character count
    Languages    []string // multi-language heuristic hint, empty = auto-detect
}
```

Strategy dispatch happens in `runTier()`; the heading and heuristic implementations are overridden via package-level function variables in each file's `init()`:

```go
// internal/infrastructure/chunker/strategy.go
func runTier(tier StrategyTier, text string, cfg SplitterConfig, profile *DocProfile) []Chunk {
    switch tier {
    case TierHeading:
        return splitByHeadings(text, cfg, profile)
    case TierHeuristic:
        return splitByHeuristics(text, cfg, profile)
    case TierLegacy:
        return SplitText(text, cfg)
    }
    return SplitText(text, cfg)
}

var splitByHeadings = func(text string, cfg SplitterConfig, _ *DocProfile) []Chunk {
    return SplitText(text, cfg) // overridden by heading_splitter.go's init()
}
var splitByHeuristics = func(text string, cfg SplitterConfig, _ *DocProfile) []Chunk {
    return SplitText(text, cfg) // overridden by heuristic_splitter.go's init()
}
```

### Existing Implementations

| Strategy Tier | Description | File |
| --- | --- | --- |
| `TierHeading` | Chunks by Markdown heading hierarchy | `internal/infrastructure/chunker/heading_hierarchy.go`, etc. |
| `TierHeuristic` | Multi-language heuristic chunking | `internal/infrastructure/chunker/heuristic_splitter.go` |
| `TierLegacy` (= `recursive`) | Recursive separator-based chunking (original implementation) | `SplitText()` in `internal/infrastructure/chunker/splitter.go` |
| Validator | Every tier's output is checked by `ValidateChunks`; failures fall back down the chain | `internal/infrastructure/chunker/validator.go` |

### Steps to Add One

1. Create `internal/infrastructure/chunker/my_splitter.go`, implementing `func(text string, cfg SplitterConfig, profile *DocProfile) []Chunk`;
2. **Registration point: `internal/infrastructure/chunker/strategy.go`** —
   - Add a strategy constant (e.g. `StrategyMine = "mine"`) and a new `StrategyTier`;
   - In the switch statements of `resolveChain`/`resolveChainWithProfile`, return the tier chain for the new strategy (recommended to fall back to `TierLegacy` as a bottom rung);
   - Add a new case in `runTier()`;
3. No changes are needed on the caller side: the knowledge base's `chunking_config.strategy` (JSONB) is passed in via `buildSplitterConfig` in `internal/application/service/knowledge.go`;
4. Write unit tests with `SplitWithDiagnostics` to verify tier selection and `ValidateChunks` acceptance behavior.

---

## 3. Adding a New Retrieval Engine

### Interface Definition

The interface is in `internal/types/interfaces/retriever.go` (three layers: engine → repository → service + registry):

```go
// internal/types/interfaces/retriever.go
type RetrieveEngine interface {
    EngineType() types.RetrieverEngineType
    Retrieve(ctx context.Context, params types.RetrieveParams) ([]*types.RetrieveResult, error)
    Support() []types.RetrieverType // supported retrieval types (vector/keyword)
}

type RetrieveEngineRepository interface {
    Save(ctx context.Context, indexInfo *types.IndexInfo, params map[string]any) error
    BatchSave(ctx context.Context, indexInfoList []*types.IndexInfo, params map[string]any) error
    EstimateStorageSize(ctx context.Context, indexInfoList []*types.IndexInfo, params map[string]any) int64
    DeleteByChunkIDList(ctx context.Context, indexIDList []string, dimension int, knowledgeType string) error
    DeleteBySourceIDList(ctx context.Context, sourceIDList []string, dimension int, knowledgeType string) error
    CopyIndices(ctx context.Context, sourceKnowledgeBaseID string,
        sourceToTargetKBIDMap map[string]string,
        sourceToTargetChunkIDMap map[string]string,
        targetKnowledgeBaseID string, dimension int, knowledgeType string) error
    DeleteByKnowledgeIDList(ctx context.Context, knowledgeIDList []string, dimension int, knowledgeType string) error
    BatchUpdateChunkEnabledStatus(ctx context.Context, chunkStatusMap map[string]bool) error
    BatchUpdateChunkTagID(ctx context.Context, chunkTagMap map[string]string) error
    RetrieveEngine
}

type RetrieveEngineRegistry interface {
    Register(indexService RetrieveEngineService) error
    GetRetrieveEngineService(engineType types.RetrieverEngineType) (RetrieveEngineService, error)
    GetAllRetrieveEngineServices() []RetrieveEngineService
    GetByStoreID(storeID string) (RetrieveEngineService, error)
}
```

The engine type enumeration is in `internal/types/retriever.go`:

```go
// internal/types/retriever.go
const (
    PostgresRetrieverEngineType        RetrieverEngineType = "postgres"
    ElasticsearchRetrieverEngineType   RetrieverEngineType = "elasticsearch"
    InfinityRetrieverEngineType        RetrieverEngineType = "infinity"
    ElasticFaissRetrieverEngineType    RetrieverEngineType = "elasticfaiss"
    QdrantRetrieverEngineType          RetrieverEngineType = "qdrant"
    MilvusRetrieverEngineType          RetrieverEngineType = "milvus"
    WeaviateRetrieverEngineType        RetrieverEngineType = "weaviate"
    DorisRetrieverEngineType           RetrieverEngineType = "doris"
    SQLiteRetrieverEngineType          RetrieverEngineType = "sqlite"
    TencentVectorDBRetrieverEngineType RetrieverEngineType = "tencent_vectordb"
    OpenSearchRetrieverEngineType      RetrieverEngineType = "opensearch"
)
```

### Existing Implementations

All under `internal/application/repository/retriever/`: `postgres/` (pgvector + BM25/ParadeDB), `elasticsearch/v7/`, `elasticsearch/v8/`, `qdrant/`, `milvus/`, `weaviate/`, `doris/`, `sqlite/` (sqlite-vec + FTS5), `tencentvectordb/`, `opensearch/`.

### Steps to Add One

1. Add a `RetrieverEngineType` constant in `internal/types/retriever.go`;
2. Create a new package under `internal/application/repository/retriever/myengine/`, implementing the `RetrieveEngineRepository` interface (you can reference `qdrant/` or `sqlite/`);
3. **Registration point: `initRetrieveEngineRegistry()` in `internal/container/container.go`** — conditionally register based on the `RETRIEVE_DRIVER` environment variable (comma-separated):

```go
// internal/container/container.go (excerpt)
retrieveDriver := strings.Split(os.Getenv("RETRIEVE_DRIVER"), ",")
if slices.Contains(retrieveDriver, "postgres") {
    postgresRepo := postgresRepo.NewPostgresRetrieveEngineRepository(db)
    if err := registry.Register(
        retriever.NewKVHybridRetrieveEngine(postgresRepo, types.PostgresRetrieverEngineType),
    ); err != nil { ... }
}
```

   Follow this example to add a branch for your new engine, wrapping it with `retriever.NewKVHybridRetrieveEngine(repo, engineType)` before registering;
4. If the engine requires an independent deployment, add a profile-gated service in `docker-compose.dev.yml` (reference `qdrant`/`opensearch`), and add the corresponding connection variables in `.env.example`.

---

## 4. Adding a New Model Provider (internal/models/provider)

### Interface Definition

The provider metadata interface and global registry are in `internal/models/provider/provider.go`:

```go
// internal/models/provider/provider.go
type ProviderName string // "openai" / "anthropic" / "aliyun" / "zhipu" / "deepseek" / ...

type Provider interface {
    // Info returns the provider's metadata
    Info() ProviderInfo
    // ValidateConfig validates the provider's configuration
    ValidateConfig(config *Config) error
}

// Register adds a provider to the global registry
func Register(p Provider)
```

`ProviderInfo` describes the display name, the default BaseURL for each model type (chat/embedding/rerank), whether authentication is required, extra config fields, etc. Differentiated adaptation of chat requests (endpoint construction, thinking parameters, auth headers, tool-call metadata) is handled by the internal adapter interface in `internal/models/chat/provider.go`:

```go
// internal/models/chat/provider.go
type providerAdapter interface {
    Name() provider.ProviderName
    Matches(model string) bool
    Thinking() ThinkingStrategy
    ShapeRequest(req *openai.ChatCompletionRequest, opts *ChatOptions, isStream bool)
    TransformMessages(msgs []openai.ChatCompletionMessage) []openai.ChatCompletionMessage
    Endpoint(baseURL, modelID string, isStream bool) string
    Auth(req *http.Request, creds authCreds, body []byte)
    ForceRawHTTP() bool
    ExtractToolCallMetadata(raw json.RawMessage) types.ToolCallMetadata
    InjectToolCallMetadata(toolCall map[string]any, metadata types.ToolCallMetadata)
}
```

Embedding and Rerank each have their own separate interface:

```go
// internal/models/embedding/embedder.go
type Embedder interface {
    Embed(ctx context.Context, text string) ([]float32, error)
    BatchEmbed(ctx context.Context, texts []string) ([][]float32, error)
    GetModelName() string
    GetDimensions() int
    GetModelID() string
    EmbedderPooler
}

// internal/models/rerank/reranker.go
type Reranker interface {
    Rerank(ctx context.Context, query string, documents []string) ([]RankResult, error)
    GetModelName() string
    GetModelID() string
}
```

### Existing Implementations

`internal/models/provider/provider.go` currently defines 26 `ProviderName` constants: openai, anthropic, aliyun, zhipu, openrouter, requesty, siliconflow, jina, generic, deepseek, gemini, volcengine, hunyuan, minimax, mimo, gpustack, moonshot, modelscope, qianfan, qiniu, longcat, lkeap, nvidia, and others. Concrete Provider implementations are spread across the files under `internal/models/provider/` (e.g. `zhipu.go`, `gemini.go`, `hunyuan.go`, `generic.go`); special embedding implementations such as `internal/models/embedding/jina.go`, `volcengine.go`, `nvidia.go`.

### Steps to Add One

1. **Registration point one: `internal/models/provider/provider.go`** — add a `ProviderName` constant;
2. Create `internal/models/provider/myprovider.go`, implementing the `Provider` interface (`Info()` gives the default URL / supported model types), and register it into the global registry via `provider.Register(...)` (typically in `init()` or a centralized initialization spot) — for OpenAI-compatible protocol providers this step alone is enough to make them usable, since the chat side defaults to the generic OpenAI adapter;
3. **Registration point two (optional): `internal/models/chat/provider.go`** — if the API protocol differs (non-standard endpoint, special auth, thinking field), implement and register a `providerAdapter`;
4. **Registration point three (optional)**: if a proprietary Embedding/Rerank protocol is needed, add an implementation under `internal/models/embedding/` or `internal/models/rerank/` respectively and hook it into their construction factory;
5. If you want an out-of-the-box built-in model, add a declaration to `config/builtin_models.yaml` (it will be synced into the `models` table at startup).

---

## 5. Adding a New Web Search Engine (internal/infrastructure/web_search)

### Interface Definition

```go
// internal/types/interfaces/web_search.go
type WebSearchProvider interface {
    // Name returns the name of the provider
    Name() string
    // Search performs a web search
    Search(ctx context.Context, query string, maxResults int, includeDate bool) ([]*types.WebSearchResult, error)
}
```

The registry is a factory map (instantiated on demand with tenant parameters):

```go
// internal/infrastructure/web_search/registry.go
type ProviderFactory func(params types.WebSearchProviderParameters) (interfaces.WebSearchProvider, error)

type Registry struct {
    factories map[string]ProviderFactory
    mu        sync.RWMutex
}

func (r *Registry) Register(id string, factory ProviderFactory)
func (r *Registry) CreateProvider(providerType string, params types.WebSearchProviderParameters) (interfaces.WebSearchProvider, error)
```

### Existing Implementations

Under `internal/infrastructure/web_search/`: `duckduckgo.go`, `google.go`, `bing.go`, `tavily.go`, `ollama.go`, `baidu.go`, `searxng.go`, `keenable.go`, `zhipu.go` (plus `proxy.go` for outbound proxy support). Type constants are in `internal/types/web_search_provider.go` (`WebSearchProviderTypeBing/Google/DuckDuckGo/Tavily/Ollama/Baidu/Searxng/Keenable/Zhipu`).

### Steps to Add One

1. Add a `WebSearchProviderType` constant in `internal/types/web_search_provider.go`;
2. Create `internal/infrastructure/web_search/mysearch.go`, implementing `WebSearchProvider` and exposing a factory `func NewMySearchProvider(params types.WebSearchProviderParameters) (interfaces.WebSearchProvider, error)`;
3. **Registration point: `registerWebSearchProviders()` in `internal/container/container.go`**:

```go
func registerWebSearchProviders(registry *infra_web_search.Registry) {
    registry.Register("duckduckgo", infra_web_search.NewDuckDuckGoProvider)
    registry.Register("google", infra_web_search.NewGoogleProvider)
    // ... add here:
    registry.Register("mysearch", infra_web_search.NewMySearchProvider)
}
```

4. If the new engine needs to appear in the frontend's provider dropdown and parameter form, update the corresponding config page components under `frontend/`; tenant configuration is persisted in the `web_search_providers` table.

---

## 6. Adding a New Data Source Connector (internal/datasource/connector)

> The directory includes an implementation guide, `internal/datasource/CONNECTOR_IMPLEMENTATION_GUIDE.md`, which you can read alongside this section.

### Interface Definition

```go
// internal/datasource/connector.go
type Connector interface {
    // Type returns the connector type identifier (e.g., "feishu", "notion")
    Type() string

    // Validate verifies that the provided configuration is valid by testing
    // connectivity and checking credentials.
    Validate(ctx context.Context, config *types.DataSourceConfig) error

    // ListResources lists available resources that can be synced.
    // parentID supports lazy-loading of hierarchical resources: "" returns the top level, non-empty returns the direct children of that resource.
    ListResources(ctx context.Context, config *types.DataSourceConfig, parentID string) ([]types.Resource, error)

    // ResolveResourceAncestors resolves the ancestor chain for existing selected items in a lazy-loaded tree (O(depth)).
    ResolveResourceAncestors(
        ctx context.Context, config *types.DataSourceConfig, resourceIDs []string,
    ) ([]string, error)

    // FetchAll performs a full sync of the specified resources.
    FetchAll(ctx context.Context, config *types.DataSourceConfig, resourceIDs []string) ([]types.FetchedItem, error)

    // FetchIncremental performs an incremental sync based on the provided cursor.
    FetchIncremental(ctx context.Context, config *types.DataSourceConfig, cursor *types.SyncCursor) ([]types.FetchedItem, *types.SyncCursor, error)
}
```

An optional streaming interface (for paginated checkpointing with large data volumes, keeping only a single item in memory at a time):

```go
// internal/datasource/connector.go
type StreamHandler interface {
    Emit(ctx context.Context, item types.FetchedItem) error
    Checkpoint(ctx context.Context, cursor *types.SyncCursor) error
}

type StreamingConnector interface {
    Connector
    FetchStream(ctx context.Context, config *types.DataSourceConfig,
        cursor *types.SyncCursor, h StreamHandler) (*types.SyncCursor, error)
}
```

The registry is in the same file: `ConnectorRegistry` (`NewConnectorRegistry()` / `Register(connector)` / `Get(type)` / `List()`); the connector's UI metadata (name, AuthType, capabilities) lives in the `ConnectorMetadataRegistry` map in the same file.

### Existing Implementations

| Type | Directory | Description |
| --- | --- | --- |
| `feishu` / `lark` | `internal/datasource/connector/feishu/` | Same implementation, `NewConnector(RegionFeishu / RegionLark)` distinguishes the region |
| `notion` | `internal/datasource/connector/notion/` | Pages and databases |
| `yuque` | `internal/datasource/connector/yuque/` | Yuque |
| `rss` | `internal/datasource/connector/rss/` | RSS feeds |

### Steps to Add One

1. Create a new package under `internal/datasource/connector/mysource/`, implementing `Connector` (for large data volumes, it's recommended to also implement `StreamingConnector`), and provide `NewConnector()`;
2. **Registration point one: `initConnectorRegistry()` in `internal/container/container.go`**:

```go
if err := registry.Register(mysourceConnector.NewConnector()); err != nil {
    errs = errors.Join(errs, fmt.Errorf("register mysource connector: %w", err))
}
```

3. **Registration point two: `ConnectorMetadataRegistry` in `internal/datasource/connector.go`** — add a type constant (`ConnectorTypeXxx` in `internal/types`) and a metadata entry (Name/Description/AuthType/Capabilities);
4. Sync the config structure: if `types.DataSourceConfig` needs new credential fields, mind the encrypted-storage conventions; the frontend's data source onboarding page renders based on the metadata.

---

## 7. Adding a New IM Platform Adapter (internal/im)

### Interface Definition

```go
// internal/im/adapter.go
type Platform string // "wecom" / "feishu" / "lark" / "slack" / "telegram" / "dingtalk" /
                     // "mattermost" / "wechat" / "qqbot" / "yunzhijia"

// Adapter is the interface every IM platform must implement.
type Adapter interface {
    // Platform returns the platform identifier.
    Platform() Platform

    // VerifyCallback verifies the signature/token of an incoming callback request.
    VerifyCallback(c *gin.Context) error

    // ParseCallback parses the raw IM callback request into a unified IncomingMessage.
    // Returns nil message for non-message events (e.g., URL verification).
    ParseCallback(c *gin.Context) (*IncomingMessage, error)

    // SendReply sends a reply back to the IM platform.
    SendReply(ctx context.Context, incoming *IncomingMessage, reply *ReplyMessage) error

    // HandleURLVerification handles the initial URL verification challenge.
    HandleURLVerification(c *gin.Context) bool
}
```

Two optional capability interfaces:

```go
// internal/im/adapter.go
// StreamSender: once implemented, the IM service pushes real-time streaming replies (e.g. Feishu streaming cards, Telegram message editing)
type StreamSender interface {
    StartStream(ctx context.Context, incoming *IncomingMessage) (string, error)
    UpdateStreamContent(ctx context.Context, incoming *IncomingMessage, streamID string, fullContent string) error
    FinalizeStream(ctx context.Context, incoming *IncomingMessage, streamID string, finalContent string) error
    EndStream(ctx context.Context, incoming *IncomingMessage, streamID string) error
}

// FileDownloader: once implemented, channels configured with a knowledge_base_id will ingest file messages into the knowledge base
type FileDownloader interface {
    DownloadFile(ctx context.Context, msg *IncomingMessage) (io.ReadCloser, string, error)
}
```

Adapters are instantiated per channel by a factory (`internal/im/service.go`):

```go
// internal/im/service.go
type AdapterFactory func(ctx context.Context, channel *IMChannel,
    msgHandler func(ctx context.Context, msg *IncomingMessage) error,
) (Adapter, context.CancelFunc, error)

func (s *Service) RegisterAdapterFactory(platform string, factory AdapterFactory)
```

### Existing Implementations

Under `internal/im/`, one subpackage per platform: `wecom/`, `feishu/` (also reused for lark, via `feishu.NewFactory(RegionLark)`), `slack/`, `telegram/`, `dingtalk/`, `mattermost/`, `wechat/`, `qqbot/`, `yunzhijia/`.

### Steps to Add One

1. Add a `Platform` constant in `internal/im/adapter.go`;
2. Create `internal/im/myplatform/`, implementing `Adapter` (adding `StreamSender`/`FileDownloader` as needed) and `NewFactory() im.AdapterFactory`;
3. **Registration point: `registerIMAdapterFactories()` in `internal/container/container.go`**:

```go
func registerIMAdapterFactories(imService *imPkg.Service) {
    imService.RegisterAdapterFactory("wecom", wecom.NewFactory())
    // ... add here:
    imService.RegisterAdapterFactory("myplatform", myplatform.NewFactory())
    if err := imService.LoadAndStartChannels(); err != nil { ... }
}
```

4. Channel configuration is persisted in the `im_channels` table, with session mappings in `im_channel_sessions`; the frontend's channel management page needs a configuration form added for the new platform.

---

## 8. Adding a New Agent Tool (internal/agent/tools)

### Interface Definition

The tool interface is defined in `internal/types/agent.go`:

```go
// internal/types/agent.go
type Tool interface {
    // Name returns the unique identifier for this tool
    Name() string

    // Description returns a human-readable description of what the tool does
    Description() string

    // Parameters returns the JSON Schema for the tool's parameters
    Parameters() json.RawMessage

    // Execute runs the tool with the given arguments
    Execute(ctx context.Context, args json.RawMessage) (*ToolResult, error)
}
```

The runtime registry is in `internal/agent/tools/registry.go`:

```go
// internal/agent/tools/registry.go
type ToolRegistry struct {
    tools             map[string]types.Tool
    maxToolOutputSize int
}

// RegisterTool adds a tool to the registry.
// Same-named tools are first-wins, to prevent name-collision hijacking (GHSA-67q9-58vj-32qx).
func (r *ToolRegistry) RegisterTool(tool types.Tool)
func (r *ToolRegistry) GetTool(name string) (types.Tool, error)
func (r *ToolRegistry) ListTools() []string
```

### Existing Implementations

Tool name constants are centralized in `internal/agent/tools/definitions.go`: `thinking`, `todo_write`, `grep_chunks`, `knowledge_search`, `list_knowledge_chunks`, `query_knowledge_graph`, `get_document_info`, `database_query`, `data_analysis`, `data_schema`, `web_search`, `web_fetch`, the skills tools (`execute_skill_script`, `read_skill`), and the wiki tools (`wiki_read_page`, `wiki_write_page`, `wiki_replace_text`, `wiki_rename_page`, `wiki_delete_page`, `wiki_search`, `wiki_read_source_doc`, `wiki_flag_issue`, `wiki_read_issue`, `wiki_update_issue`). Implementation files share their tool's name (e.g. `grep_chunks.go`, `knowledge_search.go`, `data_analysis.go`, `mcp_tool.go` — the latter wraps an MCP service's remote tools as a `types.Tool`).

### Steps to Add One

1. Create `internal/agent/tools/my_tool.go`, implementing the four `types.Tool` methods (`Parameters()` returns a JSON Schema; note the OpenAI limit of ≤ 64 characters for tool names, see `maxFunctionNameLength` in `definitions.go`);
2. **Registration point one: `internal/agent/tools/definitions.go`** — add a `ToolMyTool = "my_tool"` constant, and add the tool to `AvailableToolDefinitions()` (the UI's list of selectable tools; the comment there explicitly requires it to stay in sync with registered tools);
3. **Registration point two: the Agent engine's tool assembly logic** — add the construction of the new tool into the service logic that builds the `ToolRegistry` (Agent session initialization, which instantiates and `RegisterTool`s tools based on the Agent's configured allow-list); if resource cleanup is needed, implement `Cleanup` (`types.Cleanable`);
4. For tools with large output volumes, be aware of `ToolRegistry`'s `maxToolOutputSize` truncation behavior; write a `_test.go` for the tool (there are plenty of references in the same directory, e.g. `grep_chunks_scope_test.go`).

---

## 9. Adding a New Storage Backend (Object Storage)

### Interface Definition

The file service interface is in `internal/types/interfaces/file.go`:

```go
// internal/types/interfaces/file.go
type FileService interface {
    CheckConnectivity(ctx context.Context) error
    SaveFile(ctx context.Context, file *multipart.FileHeader, tenantID uint64, knowledgeID string) (string, error)
    SaveBytes(ctx context.Context, data []byte, tenantID uint64, fileName string, temp bool) (string, error)
    GetFile(ctx context.Context, filePath string) (io.ReadCloser, error)
    GetFileURL(ctx context.Context, filePath string) (string, error)
    DeleteFile(ctx context.Context, filePath string) error
    CopyFile(ctx context.Context, srcPath string, tenantID uint64, knowledgeID string) (string, error)
}
```

Multi-backend resolution (tenant-level `storage_backends` table config → `FileService` instance) goes through `internal/types/interfaces/storagebackend.go`:

```go
// internal/types/interfaces/storagebackend.go
type StorageBackendService interface {
    Create(ctx context.Context, backend *types.StorageBackend) error
    Update(ctx context.Context, backend *types.StorageBackend) error
    Delete(ctx context.Context, tenantID uint64, id string) error
    SetDefault(ctx context.Context, tenantID uint64, id string) error
    Test(ctx context.Context, backend *types.StorageBackend) error
}

type StorageBackendResolver interface {
    ResolveFileService(ctx context.Context, tenant *types.Tenant, backendID, provider, localBaseDir string) (FileService, string, error)
    ResolveBackend(ctx context.Context, tenant *types.Tenant, backendID, provider string) (*types.StorageBackend, error)
}
```

### Existing Implementations

All under `internal/application/service/file/`:

| provider | File | Description |
| --- | --- | --- |
| `local` | `local.go` | Local filesystem |
| `minio` | `minio.go` | MinIO / S3-compatible |
| `cos` | `cos.go` | Tencent Cloud COS |
| `tos` | `tos.go` | Volcengine TOS |
| `s3` | `s3.go` | AWS S3 and compatible services |
| `obs` | `obs.go` | Huawei Cloud OBS |
| `oss` | `oss.go` | Alibaba Cloud OSS |
| `ks3` | `ks3.go` | Kingsoft Cloud KS3 |

### Steps to Add One

1. Create `internal/application/service/file/mystore.go`, implementing all `FileService` methods (`CheckConnectivity` is used for the frontend's "Test Connection" button, i.e. `StorageBackendService.Test`);
2. **Registration point: the `NewFileServiceFromStorageConfig()` switch statement in `internal/application/service/file/factory.go`** — add a case to the provider switch:

```go
switch p {
case "local":  // NewLocalFileService(...)
case "minio":  // NewMinioFileService(...)
// ... add here:
case "mystore":
    return NewMyStoreFileService(cfg), p, nil
default:
    return nil, p, fmt.Errorf("unsupported storage provider: %s", p)
}
```

3. If the new provider needs new config fields (endpoint/bucket/region, etc.), extend `StorageEngineConfig` / `StorageBackend.config` (JSONB) in `internal/types`;
4. Add a form for the corresponding provider on the frontend's storage backend management page; tenant configuration lives in the `storage_backends` table (the `provider` column is the switch key).

---

## Appendix: Extension Points Quick Reference

| Extension Point | Core Interface | Interface File | Registration Point |
| --- | --- | --- | --- |
| Document Parser | `BaseParser.parse_into_text` | `docreader/parser/base_parser.py` | `docreader/parser/registry.py` `_build_default_registry()` |
| Chunking Strategy | tier function `func(text, cfg, profile) []Chunk` | `internal/infrastructure/chunker/strategy.go` | Same file's `runTier()` + strategy constants |
| Retrieval Engine | `RetrieveEngineRepository` | `internal/types/interfaces/retriever.go` | `container.go` `initRetrieveEngineRegistry()` (gated by `RETRIEVE_DRIVER`) |
| Model Provider | `Provider` / `providerAdapter` / `Embedder` / `Reranker` | `internal/models/provider/provider.go`, etc. | `provider.Register()` + `internal/models/chat/provider.go` |
| Web Search | `WebSearchProvider` | `internal/types/interfaces/web_search.go` | `container.go` `registerWebSearchProviders()` |
| Data Source Connector | `Connector` / `StreamingConnector` | `internal/datasource/connector.go` | `container.go` `initConnectorRegistry()` + `ConnectorMetadataRegistry` |
| IM Adapter | `Adapter` (+`StreamSender`/`FileDownloader`) | `internal/im/adapter.go` | `container.go` `registerIMAdapterFactories()` |
| Agent Tool | `types.Tool` | `internal/types/agent.go` | `internal/agent/tools/definitions.go` + `ToolRegistry.RegisterTool` |
| Storage Backend | `FileService` | `internal/types/interfaces/file.go` | `internal/application/service/file/factory.go` switch |
