# WeKnora API Documentation

## Table of Contents

- [Overview](#overview)
- [Most Authoritative Reference: Swagger UI](#most-authoritative-reference-swagger-ui)
- [Basic Information](#basic-information)
- [Authentication](#authentication)
- [Error Handling](#error-handling)
- [File and Image References (`resource://` and Direct Links)](#file-and-image-references-resource-and-direct-links)
- [API Overview](#api-overview)

## Overview

WeKnora provides a set of RESTful APIs for creating and managing knowledge bases, retrieving knowledge, and performing knowledge-based Q&A. This document describes in detail how to use these APIs.

## Most Authoritative Reference: Swagger UI

WeKnora also provides OpenAPI-based Swagger documentation. **After starting the service, visit `http://localhost:8080/swagger/index.html`** to see the complete parameters and request/response schemas for all endpoints, and try them directly in the browser — it updates automatically with the code, making it the most accurate API reference.

The markdown documentation in this directory provides more readable examples and scenario explanations, and is maintained in sync with Swagger; when the two differ, Swagger takes precedence.

> The Swagger UI is only mounted in non-release mode (`GIN_MODE != release`); it is disabled by default in production deployments.

## Basic Information

- **Base URL**: `/api/v1`
- **Response format**: JSON
- **Authentication method**: API Key

## Authentication

All API requests must include `X-API-Key` in the HTTP request header for authentication:

```
X-API-Key: your_api_key
```

To facilitate issue tracking and debugging, it is recommended to add `X-Request-ID` to the HTTP request header of each request:

```
X-Request-ID: unique_request_id
```

### Getting an API Key

After completing account registration on the web page, please go to the account information page to get your API Key.

Please keep your API Key safe and avoid leaking it. The API Key represents your account identity and has full API access privileges.

## Error Handling

All APIs use standard HTTP status codes to indicate request status, and return a unified error response format:

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

## File and Image References (`resource://` and Direct Links)

Images, charts, and attachments in responses are returned by default as internal references in the form `resource://<handle>`, for example
`![diagram](resource://xifDo7NTSL300Lp1goVutw)` in a Q&A answer. This type of reference cannot be loaded directly by a browser — the client
needs to call the authenticated `GET /files?file_path=<reference>` proxy to fetch the byte stream.

If you are integrating WeKnora into your own app, you can have the server return **loadable http(s) direct links**
directly, saving you this extra request:

| Method | Usage | Scope |
|------|------|----------|
| Single request | Add `?resource_urls=public` to the URL | This request only |
| Entire deployment | Environment variable `RESOURCE_URL_MODE=public` | All requests that don't explicitly pass the parameter |

`resource_urls` accepts the value `handle` (default, keeps the internal reference) or `public` (returns a direct link); any other value returns
`400`. The per-request parameter takes precedence over the environment variable, so even after setting the deployment default to `public`, you can
still fall back per-request with `?resource_urls=handle`.

Endpoints that support this parameter:

- `POST /api/v1/knowledge-chat/{session_id}` (SSE)
- `POST /api/v1/agent-chat/{session_id}` (SSE)
- `GET /api/v1/sessions/continue-stream/{session_id}` (SSE)
- `GET /api/v1/messages/{session_id}/load`
- `POST /api/v1/knowledge-search`

The rewrite covers the answer body, `knowledge_references` (including `image_info`), Agent execution steps and tool results, as well as image
attachments on messages. References that get split across two chunks in streaming responses are buffered before being rewritten, so the client
always receives the complete link.

### Notes

- **Requires outbound link capability.** Direct links are pre-signed by the storage backend, or provided via `APP_EXTERNAL_URL` + `/r/<token>`. When neither is
  available (e.g., local storage without `APP_EXTERNAL_URL` set), the reference **remains as `resource://`**, and the client
  can still fall back to the `/files` proxy. See the `APP_EXTERNAL_URL` description in `.env.example` for details.
- **Direct links are time-limited and anonymously readable** (WeKnora-issued grants last 2 hours, MinIO pre-signed URLs last 24 hours). Anyone who
  obtains the link can read the file before it expires — do not write it to logs or forward it to a party who shouldn't see that file.
- **Embed channels do not support this parameter.** Their visitors are anonymous, so endpoints under `/api/v1/embed/...` always force
  `handle` (even if `?resource_urls=public` is passed, or the deployment default is `public`), and images still go through the channel-scoped authenticated proxy.
- **Knowledge-base-scoped API Keys cannot use `public`** — this returns `403`. Such keys are also denied access to the `/files`
  proxy, so obtaining an anonymous direct link would bypass that same restriction. Use `handle` instead to call normally.
- **Direct links for the same file are reused within their validity period**: repeated requests do not re-issue credentials repeatedly, nor do they
  return a different URL each time, so client and CDN caches can hit. Once a credential is revoked or expires, the link becomes invalid immediately.

## API Overview

The WeKnora API is organized into the following categories by function:

| Category | Description | Documentation Link |
|------|------|----------|
| Authentication management | User registration, login, token management; OIDC flow | [auth.md](./auth.md) · [OIDC-Authentication-Flow.md](../OIDC-Authentication-Flow.md) |
| Space management | Create and manage space accounts | [tenant.md](./tenant.md) |
| Knowledge base management | Create, query, and manage knowledge bases | [knowledge-base.md](./knowledge-base.md) |
| Knowledge management | Upload, retrieve, and manage knowledge content | [knowledge.md](./knowledge.md) |
| Model management | Configure and manage various AI models | [model.md](./model.md) |
| Chunk management | Manage chunked knowledge content | [chunk.md](./chunk.md) |
| Tag management | Manage knowledge base tag categories | [tag.md](./tag.md) |
| FAQ management | Manage FAQ Q&A pairs | [faq.md](./faq.md) |
| Agent management | Create and manage custom agents | [agent.md](./agent.md) |
| Session management | Create and manage conversation sessions | [session.md](./session.md) |
| Knowledge search | Search content within knowledge bases | [knowledge-search.md](./knowledge-search.md) |
| Chat functionality | Knowledge-base and Agent-based Q&A | [chat.md](./chat.md) |
| Message management | Get and manage conversation messages | [message.md](./message.md) |
| Evaluation | Evaluate model performance | [evaluation.md](./evaluation.md) |
| Initialization management | Knowledge base model configuration and Ollama management | [initialization.md](./initialization.md) |
| System management | System info, parsing engines, storage engines | [system.md](./system.md) |
| MCP services | MCP tool service management | [mcp-service.md](./mcp-service.md) |
| Organization management | Organization, members, knowledge base/agent sharing | [organization.md](./organization.md) |
| Skills | Pre-installed agent skills | [skill.md](./skill.md) |
| Web search | Web search providers | [web-search.md](./web-search.md) |
| Vector store | Vector database connection management | [vector-store.md](./vector-store.md) |
| Storage backend | Object/file storage instance (multi-instance) management | [storage-backend.md](./storage-backend.md) |
| IM channels | Integration with WeCom / Feishu / Slack and other IM platforms, including channel CRUD and callbacks | [../IM-Integration-Development.md](../IM-Integration-Development.md) |
| Data source import | Integration and sync of external data sources such as Feishu / WeCom / Notion / Confluence | [../Data-Source-Import-Development.md](../Data-Source-Import-Development.md) |
