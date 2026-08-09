---
title: FAQ
tags: [Operations & Troubleshooting, FAQ, Troubleshooting, Deployment]
aliases: [FAQ, Troubleshooting, QA]
source: QA.md
---

# FAQ

## 1. How do I view the logs?

```bash
docker compose logs -f app docreader postgres
```

## 2. How do I start and stop the services?

```bash
./scripts/start_all.sh           # Start services
./scripts/start_all.sh --stop    # Stop services
./scripts/start_all.sh --stop && make clean-db  # Clear the database
```

> For starting/stopping in development mode, see the [Development Guide](../Development-Deployment/Development-Guide.md)

## 3. Documents can't be uploaded after the service starts?

This is usually caused by the Embedding model and chat model not being configured correctly.

### Troubleshooting steps

1. Check whether the model information in your `.env` configuration is complete:

```bash
INIT_LLM_MODEL_NAME=your_llm_model
INIT_EMBEDDING_MODEL_NAME=your_embedding_model
INIT_EMBEDDING_MODEL_DIMENSION=your_embedding_model_dimension
INIT_EMBEDDING_MODEL_ID=your_embedding_model_id
```

2. If you're using a remote API, you also need to provide `BASE_URL` and `API_KEY`
3. If you need the reranking feature, configure a Rerank model as well
4. Check the main service logs for any `ERROR` output

> For model management, see [Built-in Model Management](../Core-Features/Builtin-Model-Management.md)

## 4. No images, or broken image links are shown?

1. Confirm the multimodal feature is configured correctly (Knowledge Base Settings → Advanced Settings → Multimodal Feature)
2. Confirm the MinIO service is running
3. Check the MinIO bucket permissions
4. Configure `MINIO_PUBLIC_ENDPOINT` (if you need to access it from another device)

**Important**: The bucket name must not contain special characters; if you're unable to change its permissions, you can enter a bucket name that doesn't exist yet — the system will create it automatically.

## 5. Platform compatibility notes

`OCR_BACKEND=paddle` may not work correctly on some platforms.

- **Option 1**: Disable OCR recognition (remove the `OCR_BACKEND` setting)
- **Option 2**: Use an external OCR model (recommended) — set `OCR_BACKEND=vlm`

## 6. How do I use the data analysis feature?

- **Intelligent reasoning**: Requires checking "View Data Metadata" and "Data Analysis" in the tool configuration
- **Quick Q&A agent**: No manual tool selection needed

### Notes

- Only CSV and Excel formats are supported
- Only read-only queries are supported (SELECT, SHOW, etc.)

## 7. Configuration I just saved on the page disappears again a few seconds later?

This is usually caused by browser proxies, caching, or extension interference. Suggestions:
1. Disable your browser proxy and packet-capturing tools
2. Force-refresh the page or use an incognito window
3. Check the Network panel to confirm requests aren't being rewritten by a proxy

## 8. SSRF validation whitelist

The `SSRF_WHITELIST` setting is used to bypass the standard SSRF restrictions. It supports: exact domains, wildcard domains, IPv4/IPv6, and CIDR ranges.

```bash
# SSRF_WHITELIST=internal.service,*.corp.example,172.16.0.0/12
```

> Please configure this carefully in production. For the hardcoded API endpoint policy for search engines, see [Adding a Web Search Engine](../Integration-Extension/Adding-a-New-Search-Engine.md)

## Related Topics

- [Development Guide](../Development-Deployment/Development-Guide.md) — Troubleshooting development environment issues
- [Enabling the Knowledge Graph Feature](../Core-Features/Enabling-Knowledge-Graph.md) — Troubleshooting the knowledge graph feature
- [Built-in Model Management](../Core-Features/Builtin-Model-Management.md) — Model configuration issues
- [IM Integration Development](../Integration-Extension/IM-Integration-Development.md) — IM integration issues

---

## Backlinks

- [Home](../Home.md) — Wiki home navigation
- [Development Guide](../Development-Deployment/Development-Guide.md) — Development environment setup and troubleshooting
- [Enabling the Knowledge Graph Feature](../Core-Features/Enabling-Knowledge-Graph.md) — Troubleshooting related to the knowledge graph feature
- [Built-in Model Management](../Core-Features/Builtin-Model-Management.md) — Model configuration-related issues
- [Quick Development Mode](../Development-Deployment/Quick-Development-Mode.md) — Development mode-related issues
- [Adding a Web Search Engine](../Integration-Extension/Adding-a-New-Search-Engine.md) — SSRF whitelist and search API security policy
