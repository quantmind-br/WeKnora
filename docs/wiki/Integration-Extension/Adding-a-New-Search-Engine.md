---
title: Adding a Web Search Engine
tags: [Integration Extensions, Search, Web Search, Provider]
aliases: [Web Search, WebSearch, Search Engine Extension]
source: Adding-a-New-Search-Engine.md
---

# Adding a New Web Search Engine

This document explains how to add a new web search engine type (such as Brave Search, Searx, etc.) to WeKnora.

## Architecture Overview

Search engine API endpoints are **hardcoded** in the code, and the BaseURL is not exposed to users — eliminating SSRF risk at the source.

```
internal/types/web_search_provider.go       # Entity definitions + Provider type metadata
internal/infrastructure/web_search/          # Provider implementations (bing/google/duckduckgo/tavily)
internal/container/container.go              # DI registration
internal/types/interfaces/web_search.go      # WebSearchProvider interface
```

> For a similar extension development pattern, see [Integrating a Vector Database](../Integration-Extension/Integrating-a-Vector-Database.md)

## Steps

Using **Brave Search** as an example:

### 1. Register the type constant

Add to `internal/types/web_search_provider.go`:

```go
WebSearchProviderTypeBrave WebSearchProviderType = "brave"
```

### 2. Add type metadata

Add to `GetWebSearchProviderTypes()`:

```go
{
    ID: "brave", Name: "Brave Search", Free: false,
    RequiresAPIKey: true, Description: "Brave Search API",
    DocsURL: "https://brave.com/search/api/",
}
```

### 3. Create the Provider implementation

Create a new file `internal/infrastructure/web_search/brave.go`:

- Constructor signature: `func(types.WebSearchProviderParameters) (interfaces.WebSearchProvider, error)`
- The API endpoint is **hardcoded** as a constant
- Implement the `interfaces.WebSearchProvider` interface: `Name()` and `Search()`

### 4. Add parameter validation

Add the new type to `isValidProviderType()`.

### 5. DI registration

Register it in `registerWebSearchProviders`.

### 6. Verify

```bash
go build ./...
# Call the API to verify
curl http://localhost:8080/api/v1/web-search-providers/types
```

## When Additional Parameters Are Needed

If the new engine requires parameters beyond an API key:

- **Option one**: Use the `WebSearchProviderParameters.ExtraConfig` field
- **Option two**: Add a dedicated field to `WebSearchProviderParameters`

## File Change Checklist

| File | Action |
|------|------|
| `internal/types/web_search_provider.go` | Add constant + type metadata |
| `internal/infrastructure/web_search/brave.go` | **Create** Provider implementation |
| `internal/application/service/web_search_provider.go` | Add new type to `isValidProviderType` |
| `internal/container/container.go` | Register in `registerWebSearchProviders` |

## Related Topics

- [Integrating a Vector Database](../Integration-Extension/Integrating-a-Vector-Database.md) — Similar extension development pattern (interface implementation + registration + DI)
- [MCP Feature Usage Guide](../Core-Features/MCP-Usage-Guide.md) — MCP can also integrate search tools
- [FAQ](../Operations-Troubleshooting/FAQ.md) — SSRF whitelist configuration

---

## Backlinks

- [Home](../Home.md) — Wiki home navigation
- [Integrating a Vector Database](../Integration-Extension/Integrating-a-Vector-Database.md) — Similar extension development pattern
- [MCP Feature Usage Guide](../Core-Features/MCP-Usage-Guide.md) — MCP search tools complement web search engines
- [Roadmap](../Project-Overview/Version-Roadmap.md) — Community component extension direction in the roadmap
