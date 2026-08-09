// Curated sample texts for the chunking debug drawer. Each preset is sized
// to ≈2000–4000 characters and shaped to exercise a distinct tier of the
// chunker, so users can quickly see how their config behaves on realistic
// content without preparing their own sample.

export interface ChunkingSample {
  id: string
  // i18n key under knowledgeEditor.chunking.debug.samples.<id>
  labelKey: string
  text: string
}

const MARKDOWN_SAMPLE = `# WeKnora Knowledge Framework

WeKnora is an open-source, LLM-based enterprise knowledge framework that combines RAG Q&A, ReAct agents, and a Wiki knowledge graph. This document covers its design rationale, architecture, and typical usage.

## Design Rationale

Enterprise knowledge is scattered across systems such as Confluence, Feishu, Notion, and Git repositories. Traditional full-text search struggles with semantics, while a bare LLM lacks a trustworthy source of context. WeKnora aims to provide:

- **Multi-source ingestion**: extract, clean, and vectorize scattered content uniformly
- **Multi-strategy retrieval**: dense, sparse, and knowledge-graph recall + RRF fusion
- **Explainable reasoning**: ReAct agents keep the reasoning chain traceable and steerable
- **Observability**: native Langfuse integration — every reasoning step and tool call is traced

## Core Features

### Multi-Model Support

Compatible with 20+ mainstream LLMs: OpenAI, Anthropic, DeepSeek, Qwen, Zhipu, Hunyuan, Gemini, and local Ollama. The model layer abstracts the \`Provider\` interface; adding a vendor only requires implementing the \`Chat()\` method and registering it.

### Multi-Vector-DB Support

Switch via the \`RETRIEVE_DRIVER\` environment variable:

| Driver | Best for |
|------|---------|
| pgvector | Existing PostgreSQL, scale < 10M chunks |
| Elasticsearch | Hybrid retrieval + keyword highlighting |
| Milvus | Large scale, low latency |
| Weaviate | Hybrid BM25 + GraphQL queries |
| Qdrant | Default recommendation, simple deployment, balanced performance |

### Data Source Connectors

- **Feishu**: auto-sync knowledge base and document tree with incremental updates
- **Notion**: extract Pages / Databases via API
- **Yuque**: pull by Group, preserving multi-level directories

Every connector implements the unified \`DataSource\` interface; a new source takes roughly 200 lines of code.

## Quick Start

### Requirements

- Docker 20.10+ and Docker Compose v2
- At least 8GB RAM and 20GB disk
- For locally hosted LLMs, an NVIDIA GPU with 24GB+ VRAM is recommended

### Startup Commands

\`\`\`bash
git clone https://github.com/Tencent/WeKnora && cd WeKnora
cp .env.example .env       # adjust your model and database config
make dev-start             # start postgres / redis / qdrant
make dev-app               # start backend with hot reload
make dev-frontend          # start frontend with hot reload
\`\`\`

Open http://localhost:5173 in your browser.

## Architecture Overview

The WeKnora backend follows a clean layered architecture:

\`\`\`
┌────────────────────────┐
│   HTTP layer (Gin)     │  routing / auth / rate limiting
├────────────────────────┤
│   Service layer        │  core algorithms / orchestration
├────────────────────────┤
│   Repository layer     │  database access / caching
├────────────────────────┤
│   Domain types         │  KnowledgeBase / Chunk / Agent ...
└────────────────────────┘
\`\`\`

The dependency injection container (go/dig) wires everything in \`cmd/server/main.go\`. Handlers get services through interfaces, services get repositories through interfaces, and every replaceable implementation is bound at startup.

### Retrieval Pipeline

Upload → docreader parses → chunking (parent/child) → embedding → write to vector and graph DBs → multi-path recall on query → RRF fusion → rerank → response.

### Agent Orchestration

The agent engine runs the classic ReAct loop:

1. **Reason**: the LLM produces reasoning from system prompt and current observations
2. **Act**: parse the tool-call JSON and dispatch to built-in or MCP tools
3. **Observe**: feed tool results back into the context
4. Repeat 1–3 until a final answer is produced or the iteration limit is reached

Every step is streamed to the frontend over SSE and written to Langfuse traces for debugging.

## Further Reading

- API docs: \`docs/api/README.md\`
- Config reference: \`config/config.yaml\` and \`.env.example\`
- Troubleshooting: \`docs/QA.md\`
- Roadmap: \`docs/ROADMAP.md\``

const FAQ_SAMPLE = `# WeKnora Deployment & Usage FAQ

This document collects the most common questions from the community and internal users, grouped by "Installation / Configuration / Retrieval / Models / Performance".

## Installation & Startup

### Q1: Where can I download the Docker images?
Official images are distributed through the daocloud accelerator:

\`\`\`bash
docker pull docker.m.daocloud.io/wechatopenai/weknora-app:v0.5.0
docker pull docker.m.daocloud.io/wechatopenai/weknora-docreader:v0.5.0
docker pull docker.m.daocloud.io/wechatopenai/weknora-ui:v0.5.0
\`\`\`

### Q2: Port 5173 shows an old UI after startup?
The browser cached old frontend assets. Force-refresh with Ctrl+Shift+R, or clear the site storage if needed.

### Q3: New user registration returns 500 Internal Server Error
Usually the backend container did not start properly. First check \`make dev-logs | grep app\`; common causes are database connection failures or missing migrations.

### Q4: Uploading documents fails with column "xxx" does not exist
Database migrations are incomplete. Run \`make migrate-up\`, verify all migrations succeed, then restart the app container.

### Q5: Documents stay in processing state after upload
Check whether the docreader container is running (\`docker compose ps\`) and inspect its logs for parse errors. Common causes: missing Python packages, encrypted PDFs, or OCR timeouts on scanned documents.

## Configuration

### Q6: How do I switch vector databases?
Change \`RETRIEVE_DRIVER\` in \`.env\`: \`qdrant\` (default) / \`pgvector\` / \`elasticsearch\` / \`milvus\` / \`weaviate\`. Existing data must be re-embedded after switching.

### Q7: How do I change the default chunk size?
The global default lives in the \`knowledge.chunking\` section of \`config/config.yaml\`; individual knowledge bases can override it in the "Chunking Settings" UI. Changes apply only to newly uploaded documents.

### Q8: How do I enable multimodal support?
Turn it on in the knowledge base "Multimodal Configuration" and select a VLM model (e.g. GPT-4o, Qwen2.5-VL). Once enabled, images in documents are described by the VLM and included in retrieval.

### Q9: How do I switch logs to JSON format?
Set \`LOG_FORMAT=json LOG_LEVEL=info\` to feed ELK / Loki log stacks.

## Retrieval & Recall

### Q10: Search results are empty but the document is indexed
Troubleshoot in this order:

1. Confirm the knowledge base status is \`completed\`, not \`indexing\` or \`failed\`
2. Check the vector DB connection (default Qdrant on port 6333): \`docker compose ps\` should show it healthy
3. Verify the embedding model matches the one used at indexing time
4. Try a shorter query and check the recall thresholds (top_k / rerank threshold)
5. Inspect Langfuse traces for the query to see whether retrieval returned zero chunks

### Q11: FAQ entries are never matched
Make sure FAQ is enabled and the score threshold (\`faq_direct_answer_threshold\`) is not too high. Keep entries concise; similar questions boost matching.

## Models

### Q12: Which providers are supported?
OpenAI-compatible endpoints, Anthropic, DeepSeek, Qwen, Zhipu, Hunyuan, Gemini, Ollama (local), and custom HTTP providers. Register providers in the admin "Model Providers" page.

### Q13: Why is the model returning empty responses?
Check the provider key/quota, the \`max_completion_tokens\` setting, and the Langfuse trace for the request. Some providers return empty content when the safety filter triggers.

### Q14: How do I use a local Ollama model?
Install Ollama, pull a model (e.g. \`qwen2.5\`), then add an Ollama provider in the admin panel and point the base URL at \`http://host.docker.internal:11434\` from containers.

## Performance

### Q15: Retrieval is slow. What should I tune first?
Profile with Langfuse first. Common wins: enable reranking with a small cross-encoder, reduce \`embedding_top_k\` before rerank, and ensure the vector DB index is built (HNSW defaults are fine for most cases).

### Q16: Indexing takes too long for large PDFs
Scale docreader workers, increase parsing concurrency, and enable incremental chunking so only changed documents are re-embedded.

### Q17: How can I reduce memory usage?
Use a smaller embedding model (e.g. bge-small instead of bge-large), lower docreader concurrency, and set JVM/process heap limits for Elasticsearch/Milvus if you run them locally.`

const CHAPTER_SAMPLE = `# Chapter 1 Introduction

## 1.1 Purpose of This Document

This document describes the overall architecture, component breakdown, and deployment plan of a distributed knowledge retrieval platform. It is intended for operations engineers, SREs, and system architects. Readers should be familiar with base components such as Docker, Kubernetes, and PostgreSQL.

## 1.2 Terminology

- **Gateway layer**: handles ingress traffic routing and protocol translation
- **Service layer**: carries domain logic and orchestration
- **Storage layer**: encapsulates persistence and caching details
- **Inference layer**: isolated layer that interacts with external LLM / embedding models

## 1.3 Document Version

This version is v1.4, corresponding to platform code v0.5.x. The document is kept in sync with the code; major changes are listed in Appendix A.

# Chapter 2 System Architecture

## 2.1 High-Level Design

The platform uses a classic microservice architecture but stays conservative about boundaries — services are split only where independent scaling is genuinely required. There are currently 4 long-running services: gateway, business backend, document parser, and vector index.

## 2.2 Module Breakdown

### 2.2.1 Users & Permissions

Users, organizations, shared spaces, roles, and permission rules are centralized in the user-service. The RBAC model supports inheritance and override; external OIDC integration is done through an adapter layer.

### 2.2.2 Content & Indexing

The content side owns the document lifecycle: upload, parse, chunk, embed, store, retrieve, and citation tracking. A chunk is the finest retrieval unit; every chunk carries source-document metadata and position information.

### 2.2.3 Inference & Orchestration

The agent orchestrator drives the ReAct loop; tool calls are dispatched via the MCP protocol or the built-in registry. All LLM calls go through a unified model proxy layer for easy switching, rate limiting, and metering.

## 2.3 Data Flow

After upload, documents pass through: parse → clean → chunk → embed → store. Query path: rewrite → multi-path recall → fusion → rerank → context assembly → LLM generation.

# Chapter 3 Deployment Guide

## 3.1 Requirements

Physical resources: start at 8 vCPU / 16 GB RAM / 100 GB SSD; production is recommended at 16 vCPU / 32 GB / 500 GB. GPU is only required for local inference.

## 3.2 Container Orchestration

### 3.2.1 Single-Host Docker Compose

Suitable for PoCs and small teams (< 100 people, < 1M chunks). One command brings up every service:

\`\`\`bash
docker compose up -d
\`\`\`

### 3.2.2 Kubernetes Helm

Suitable for scaled deployments. The Helm chart lives in helm/, containing statefulsets, config secrets, and ingress templates. It can integrate with existing PG / Redis clusters.

## 3.3 Configuration Best Practices

- Keep external dependencies (database, object storage, vector DB) in a config center instead of hardcoding them
- Manage model API keys as Secrets, isolated per environment
- Emit logs to stdout/stderr and let the orchestration platform collect them, avoiding local files

# Chapter 4 Common Operations

## 4.1 Upgrade Process

Blue-green or rolling deployments both work. Always check CHANGELOG for schema changes before upgrading; run migrations before replacing business images.

## 4.2 Backup & Recovery

Daily full PostgreSQL backups + WAL increments; MinIO/object storage relies on cloud versioning. Qdrant uses its snapshot API to periodically persist to object storage.

## 4.3 Troubleshooting

Investigate layer by layer across "gateway → service → storage → model". Each layer exposes a health-check endpoint and Langfuse trace entry points; combined, they locate 80% of issues within 5 minutes.`

const PLAIN_SAMPLE = `Knowledge base retrieval quality depends on several factors, and the most direct one is how well the chunking strategy matches the embedding model. Chunking too coarsely mixes multiple topics in a single segment and dilutes relevance; chunking too finely loses context, so retrieving one segment alone cannot answer cross-segment questions. As a rule of thumb, set the chunk size to 50%–80% of the embedding model's recommended window — enough to keep semantics intact while leaving headroom. Common embedding models such as BGE and Cohere's embed-v3 recommend a 512-token window, which maps to roughly 300–800 characters, because a Chinese character takes about 0.5–1.2 tokens while an English word takes about 1.3 tokens.`

export const CHUNKING_SAMPLES: ChunkingSample[] = [
  { id: "markdown", labelKey: "samples.markdown", text: MARKDOWN_SAMPLE },
  { id: "faq", labelKey: "samples.faq", text: FAQ_SAMPLE },
  { id: "chapter", labelKey: "samples.chapter", text: CHAPTER_SAMPLE },
  { id: "plain", labelKey: "samples.plain", text: PLAIN_SAMPLE },
]

export const DEFAULT_SAMPLE_ID = "markdown";
