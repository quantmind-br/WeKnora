---
title: API Documentation Overview
tags: [API Reference, REST, Authentication, Endpoints]
aliases: [API Overview, API Documentation, API Reference]
source: api/README.md
---

# API Documentation Overview

WeKnora provides a set of RESTful APIs for creating and managing knowledge bases, retrieving knowledge, and performing knowledge-based question answering.

## Basic Information

- **Base URL**: `/api/v1`
- **Response Format**: JSON
- **Authentication Method**: API Key

> API authentication uses WeKnora's local JWT; for the OIDC authentication flow, see [OIDC Authentication Flow](../Security-Authentication/OIDC-Authentication-Flow.md)

## Authentication Mechanism

All API requests must include `X-API-Key` in the HTTP request header:

```
X-API-Key: your_api_key
X-Request-ID: unique_request_id  # Recommended, for tracing purposes
```

The API Key can be obtained from the account information page after completing account registration on the web page.

## Error Handling

```json
{
  "success": false,
  "error": {
    "code": "error code",
    "message": "error message",
    "details": "error details"
  }
}
```

## API Categories

| Category | Description | Detailed Documentation |
|------|------|----------|
| Authentication Management | User registration, login, token management; OIDC flow | [auth.md](../../api/auth.md) · [OIDC-Authentication-Flow.md](../Security-Authentication/OIDC-Authentication-Flow.md) |
| Space Management | Create and manage space accounts | [tenant.md](../../api/tenant.md) |
| Knowledge Base Management | Create, query, and manage knowledge bases | [knowledge-base.md](../../api/knowledge-base.md) |
| Knowledge Management | Upload, retrieve, and manage knowledge content | [knowledge.md](../../api/knowledge.md) |
| Model Management | Configure and manage various AI models | [model.md](../../api/model.md) |
| Chunk Management | Manage chunked content of knowledge | [chunk.md](../../api/chunk.md) |
| Tag Management | Manage knowledge base tag categories | [tag.md](../../api/tag.md) |
| FAQ Management | Manage FAQ question-answer pairs | [faq.md](../../api/faq.md) |
| Agent Management | Create and manage custom agents | [agent.md](../../api/agent.md) |
| Session Management | Create and manage conversation sessions | [session.md](../../api/session.md) |
| Knowledge Search | Search content within knowledge bases | [knowledge-search.md](../../api/knowledge-search.md) |
| Chat Feature | Question answering based on knowledge bases and Agents | [chat.md](../../api/chat.md) |
| Message Management | Retrieve and manage conversation messages | [message.md](../../api/message.md) |
| Evaluation Feature | Evaluate model performance | [evaluation.md](../../api/evaluation.md) |
| Initialization Management | Knowledge base model configuration and Ollama management | [initialization.md](../../api/initialization.md) |
| System Management | System information, parsing engine, storage engine | [system.md](../../api/system.md) |
| MCP Service | MCP tool service management | [mcp-service.md](../../api/mcp-service.md) |
| Organization Management | Organization, members, knowledge base/agent sharing | [organization.md](../../api/organization.md) |
| Skills | Pre-installed agent skills | [skill.md](../../api/skill.md) |
| Web Search | Web search providers | [web-search.md](../../api/web-search.md) |
| Vector Store | Vector database connection management | [vector-store.md](../../api/vector-store.md) |

> For detailed information on each API, see the corresponding documents in the `docs/api/` directory

## Related Topics

- [OIDC Authentication Flow](../Security-Authentication/OIDC-Authentication-Flow.md) — The OIDC flow for API authentication
- [Built-in Model Management](../Core-Features/Builtin-Model-Management.md) — Configuration reference for the model management API
- [MCP Feature Usage Guide](../Core-Features/MCP-Usage-Guide.md) — Usage of the MCP service management API
- [Shared Space Guide](../Security-Authentication/Shared-Spaces-Guide.md) — Business logic of the organization management API
- [IM Integration Development](../Integration-Extension/IM-Integration-Development.md) — IM channel management API
- [Data Source Import Development](../Integration-Extension/Data-Source-Import-Development.md) — Data source management API

---

## Backlinks

- [Home](../Home.md) — Wiki homepage navigation
- [OIDC Authentication Flow](../Security-Authentication/OIDC-Authentication-Flow.md) — API authentication mechanism and OIDC-related
- [Built-in Model Management](../Core-Features/Builtin-Model-Management.md) — Underlying configuration for the model management API
- [MCP Feature Usage Guide](../Core-Features/MCP-Usage-Guide.md) — Usage scenarios for the MCP service API
- [Shared Space Guide](../Security-Authentication/Shared-Spaces-Guide.md) — Business logic of the organization management API
- [IM Integration Development](../Integration-Extension/IM-Integration-Development.md) — Usage scenarios for the IM channel API
- [Data Source Import Development](../Integration-Extension/Data-Source-Import-Development.md) — Usage scenarios for the data source API
