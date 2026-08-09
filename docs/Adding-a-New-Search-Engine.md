# Adding a New Web Search Engine

This document explains how to add a new web search engine type (such as Brave Search, Searx, etc.) to WeKnora.

## Architecture Overview

```
internal/
├── types/
│   └── web_search_provider.go       # Entity definitions + Provider type metadata
├── infrastructure/
│   └── web_search/
│       ├── registry.go              # Provider factory registry
│       ├── bing.go                  # Bing implementation
│       ├── google.go                # Google implementation
│       ├── duckduckgo.go            # DuckDuckGo implementation
│       └── tavily.go                # Tavily implementation
├── container/
│   └── container.go                 # DI registration (registerWebSearchProviders)
└── types/interfaces/
    └── web_search.go                # WebSearchProvider interface
```

Search engine API endpoints are **hardcoded** in the code, and the BaseURL is not exposed to users — eliminating SSRF risk at the source.

## Steps

Using **Brave Search** as an example.

### 1. Register the type constant in `types/web_search_provider.go`

```go
const (
    WebSearchProviderTypeBing       WebSearchProviderType = "bing"
    WebSearchProviderTypeGoogle     WebSearchProviderType = "google"
    WebSearchProviderTypeDuckDuckGo WebSearchProviderType = "duckduckgo"
    WebSearchProviderTypeTavily     WebSearchProviderType = "tavily"
    WebSearchProviderTypeBrave      WebSearchProviderType = "brave"      // ← New
)
```

### 2. Add type metadata in `GetWebSearchProviderTypes()`

```go
{
    ID:             "brave",
    Name:           "Brave Search",
    Free:           false,
    RequiresAPIKey: true,
    Description:    "Brave Search API",
    DocsURL:        "https://brave.com/search/api/",
},
```

Field descriptions:

| Field             | Description                                           |
| ---------------- | ---------------------------------------------- |
| `ID`             | Unique identifier, stored in the database, cannot be changed                 |
| `Name`           | Display name shown on the frontend                                   |
| `Free`           | Whether it's free                                          |
| `RequiresAPIKey` | Whether an API Key is required                               |
| `RequiresEngineID` | Whether an additional ID is required (e.g., Google CSE)             |
| `Description`    | Short description                                       |
| `DocsURL`        | Link to official documentation, shown by the frontend in the "add" dialog           |

### 3. Create the Provider implementation

Create a new file at `internal/infrastructure/web_search/brave.go`:

```go
package web_search

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"

    "github.com/Tencent/WeKnora/internal/types"
    "github.com/Tencent/WeKnora/internal/types/interfaces"
)

const defaultBraveSearchURL = "https://api.search.brave.com/res/v1/web/search"

type BraveProvider struct {
    client *http.Client
    apiKey string
}

// NewBraveProvider creates an instance from parameters (does not read environment variables)
func NewBraveProvider(params types.WebSearchProviderParameters) (interfaces.WebSearchProvider, error) {
    if params.APIKey == "" {
        return nil, fmt.Errorf("API key is required for Brave provider")
    }
    return &BraveProvider{
        client: &http.Client{Timeout: 10 * time.Second},
        apiKey: params.APIKey,
    }, nil
}

func BraveProviderTypeInfo() types.WebSearchProviderTypeInfo {
    return types.WebSearchProviderTypeInfo{
        ID:             "brave",
        Name:           "Brave Search",
        Free:           false,
        RequiresAPIKey: true,
        Description:    "Brave Search API",
        DocsURL:        "https://brave.com/search/api/",
    }
}

func (p *BraveProvider) Name() string { return "brave" }

func (p *BraveProvider) Search(
    ctx context.Context, query string, maxResults int, includeDate bool,
) ([]*types.WebSearchResult, error) {
    // Build the request — BaseURL is hardcoded
    req, err := http.NewRequestWithContext(ctx, "GET",
        fmt.Sprintf("%s?q=%s&count=%d", defaultBraveSearchURL, query, maxResults), nil)
    if err != nil {
        return nil, err
    }
    req.Header.Set("X-Subscription-Token", p.apiKey)
    req.Header.Set("Accept", "application/json")

    resp, err := p.client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    // Parse the response
    body, _ := io.ReadAll(resp.Body)
    var data braveResponse
    if err := json.Unmarshal(body, &data); err != nil {
        return nil, err
    }

    results := make([]*types.WebSearchResult, 0, len(data.Web.Results))
    for _, r := range data.Web.Results {
        results = append(results, &types.WebSearchResult{
            Title:   r.Title,
            URL:     r.URL,
            Snippet: r.Description,
            Source:  "brave",
        })
    }
    return results, nil
}

type braveResponse struct {
    Web struct {
        Results []struct {
            Title       string `json:"title"`
            URL         string `json:"url"`
            Description string `json:"description"`
        } `json:"results"`
    } `json:"web"`
}
```

**Key requirements**:

1. The **constructor function signature** must be `func(types.WebSearchProviderParameters) (interfaces.WebSearchProvider, error)`
2. The **API endpoint must be hardcoded** as a constant, not read from parameters
3. It must **implement the `interfaces.WebSearchProvider` interface**: `Name()` and `Search()`

### 4. Add the new type to the Service's parameter validation

Edit `internal/application/service/web_search_provider.go`:

```go
func isValidProviderType(provider types.WebSearchProviderType) bool {
    switch provider {
    case types.WebSearchProviderTypeBing,
        types.WebSearchProviderTypeGoogle,
        types.WebSearchProviderTypeDuckDuckGo,
        types.WebSearchProviderTypeTavily,
        types.WebSearchProviderTypeBrave:     // ← New
        return true
    default:
        return false
    }
}
```

### 5. Register in the DI container

Edit the `registerWebSearchProviders` function in `internal/container/container.go`:

```go
func registerWebSearchProviders(registry *infra_web_search.Registry) {
    // ... existing registrations ...

    // Register Brave provider type
    registry.Register(infra_web_search.BraveProviderTypeInfo(), infra_web_search.NewBraveProvider)
}
```

### 6. Verify

```bash
# Build
go build ./...

# After starting the server, call the API to verify the type list
curl http://localhost:8080/api/v1/web-search-providers/types \
  -H 'X-API-Key: your_key'

# Create a Brave search engine instance
curl -X POST http://localhost:8080/api/v1/web-search-providers \
  -H 'X-API-Key: your_key' \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "Brave Search",
    "provider": "brave",
    "parameters": { "api_key": "BSA..." },
    "is_default": true
  }'
```

## When Additional Parameters Are Needed

If the new engine requires parameters beyond the API Key (similar to Google's `engine_id`), there are two approaches:

### Approach 1: Use `ExtraConfig`

Take advantage of the `WebSearchProviderParameters.ExtraConfig` field, without needing to change the type definition:

```go
func NewFooProvider(params types.WebSearchProviderParameters) (interfaces.WebSearchProvider, error) {
    region := params.ExtraConfig["region"]
    if region == "" {
        region = "us"
    }
    // ...
}
```

The frontend can annotate which extra fields are needed in `GetWebSearchProviderTypes()` (dynamic form rendering support planned for the future).

### Approach 2: Add a dedicated field

If the parameter is fairly generic (e.g., needed by multiple engines), you can add a new field to `WebSearchProviderParameters`:

```go
type WebSearchProviderParameters struct {
    APIKey      string            `json:"api_key,omitempty"`
    EngineID    string            `json:"engine_id,omitempty"`
    Region      string            `json:"region,omitempty"`      // ← New
    ExtraConfig map[string]string `json:"extra_config,omitempty"`
}
```

Also add a field such as `RequiresRegion bool` to `WebSearchProviderTypeInfo`, so the frontend can dynamically show the corresponding input field based on it.

## File Change Checklist

| File | Action |
| ---- | ---- |
| `internal/types/web_search_provider.go` | Add constant + type metadata |
| `internal/infrastructure/web_search/brave.go` | **New** Provider implementation |
| `internal/application/service/web_search_provider.go` | Add new type to `isValidProviderType` |
| `internal/container/container.go` | Register in `registerWebSearchProviders` |
