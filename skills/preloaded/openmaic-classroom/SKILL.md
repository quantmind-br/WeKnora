---
name: OpenMAIC Classroom
description: Converts RAG retrieval results, document chunks, or knowledge graph concepts into OpenMAIC interactive courses. Use this skill when the user asks to convert knowledge base content, retrieved document snippets, uploaded documents, or knowledge graph concepts into teaching materials / interactive lessons in bulk. Supports pure requirement generation, course generation from PDF content, and bulk classroom generation from concept graph traversal.
---

# OpenMAIC Classroom Generator

Converts RAG retrieval results or document content in the WeKnora knowledge base into OpenMAIC interactive courses.

## Core Capabilities

1. **RAG → Course**: distill knowledge retrieval results into teaching requirements and generate interactive courses through the OpenMAIC API
2. **PDF → Course**: parse user-uploaded PDFs and generate courses from the content
3. **Document chunks → Course set**: organize multiple document chunks / knowledge snippets into a multi-stage course set
4. **Concept graph traversal → bulk micro-classrooms**: traverse all concept pages in the knowledge graph and generate one micro-classroom per concept

## Scope Boundaries

> **Through the mcp_api_requester MCP tool registered in WeKnora, you can call the OpenMAIC API directly** (HTTP POST/GET requests).
> The tool is registered in the WeKnora agent as `mcp_{service_name}_{tool_name}` (e.g. `mcp_mcp_api_requester_make_request`).
> You **must** use this MCP tool to call the API. If the MCP tool is not configured or unavailable, guide the user to deploy mcp-api-requester first instead of providing curl commands.

> **⚠️ MCP availability check (must run every time this file is read):**
> Before starting work, check whether the available MCP tools include one whose name contains `mcp_api_requester`.
> - **If not found**: you must remind the user: "The `mcp_api_requester` MCP service was not detected. Please download the code from https://github.com/yryuu/mcp-api-requester, deploy it, and then register this MCP service in WeKnora."
> - **If found**: continue with the flow below and call the API through that MCP tool.

## Mode Selection

OpenMAIC has two usage modes; **choose based on the user's scenario**:

| Mode | Base URL | Authentication | Fits |
|------|----------|----------|------|
| Hosted mode (recommended for quick use) | `https://open.maic.chat` | `Authorization: Bearer <access-code>` | The user has an open.maic.chat access code and does not need a local deployment |
| Local mode | Provided by the user (see Local Mode Base URL Handling) | No authentication (self-deployed locally) | The user deployed their own OpenMAIC instance |

**Decision rules**:
- The user mentions "online service", "open.maic.chat", or "access code" → use hosted mode
- The user mentions "local deployment", "self-hosted" → use local mode
- If the user does not specify, ask which mode they want to use first

**Local mode Base URL handling**:
1. If the user chooses local mode, you must ask: "Please enter your OpenMAIC local deployment address (e.g. `http://localhost:3000` or `http://192.168.1.100:3000`)"
2. After receiving the address, process it as follows:
   - Replace `127.0.0.1` in the address with `host.docker.internal`
   - Replace `localhost` in the address with `host.docker.internal`
   - Leave other addresses unchanged

> ⚠️ WeKnora runs inside a Docker container; `localhost` and `127.0.0.1` point to the container itself and cannot reach host services. You must use `host.docker.internal` as the bridge address from the container to the host.

## Prerequisites

| Item | Description |
|--------|------|
| Mode | Hosted mode or local mode (see decision rules above) |
| `accessCode` | Required in hosted mode — the access code (starting with `sk-`), obtained by the user on open.maic.chat |
| MCP tool | `mcp_api_requester` must be registered in WeKnora (see MCP availability check) |

## Scenario Identification

Identify which scenario the user is in and follow the corresponding flow:

- **Scenario 1 (pure requirement)**: the user only provides a teaching topic/description, no source content
- **Scenario 2 (RAG results)**: the agent has retrieved document chunks; convert them into a course
- **Scenario 3 (PDF upload)**: the user uploaded a PDF and wants a course generated from its content
- **Scenario 4 (concept graph traversal)**: the user wants to generate micro-classrooms in bulk from knowledge graph concepts

### Phase 1.1: RAG → Requirement Conversion (only for Scenario 2)

When Scenario 2 has RAG retrieval results, call `scripts/rag-to-requirement.py` to convert the chunks into a requirement:

```
execute_skill_script(
  skill_name: "openmaic-classroom",
  script_path: "scripts/rag-to-requirement.py",
  input: '{"chunks": [...retrieved chunks...], "query": "user query", "audience": "target audience"}'
)
```

**input parameter format (JSON string; must be passed via `input`, NOT `--file`):**
- `chunks` (required): array of RAG retrieval results; each item contains `document_name`, `content`, `metadata`
- `query` (optional): the user's original query
- `audience` (optional): target audience description; defaults to "learners in the relevant field"
- `depth` (optional): teaching depth `beginner|intermediate|advanced`; defaults to `intermediate`
- `language` (optional): `zh-CN|en-US`; defaults to `en-US`
- `focus_areas` (optional): array of focus areas

**Notes:**
- The chunks data must be passed via the `input` parameter (equivalent to `echo '{"chunks":...}' | python script.py`)
- **Do not** call this script without any arguments, or it will exit with an error
- If the script fails, you can build the requirement manually from the retrieval results

### Phase 1.2: Concept Graph → Requirement Conversion (only for Scenario 4)

When Scenario 4 requires generating classrooms in bulk from knowledge graph concepts, follow these steps:

**Step 1: List all concept pages**

Use the `wiki_search` tool to search for all pages of type concept:

```
wiki_search("^concept/", limit=50)
```

If there are more than 50 concepts, paginate with multiple calls until all are fetched.

**Step 2: Get details and linked entities for each concept**

For each concept page:

1. Call `wiki_read_page([concept_slug])` to get the page details (including OutLinks and InLinks)
2. From OutLinks and InLinks, filter the slugs that start with `entity/*`
3. Determine each entity's link_type:
   - Present in both OutLinks and InLinks → `bidirectional`
   - Present only in OutLinks → `outlink`
   - Present only in InLinks → `inlink`
4. Call `wiki_read_page([entity_slugs])` to read the linked entities in bulk (only title + summary, not full content)

**Step 3: Convert to requirement**

For each concept, call `scripts/concept-to-requirement.py` to convert the concept + linked entities into a requirement:

```
execute_skill_script(
  skill_name: "openmaic-classroom",
  script_path: "scripts/concept-to-requirement.py",
  input: '{"concept": {"slug": "...", "title": "...", "summary": "...", "content": "..."}, "entities": [{"slug": "...", "title": "...", "summary": "...", "link_type": "..."}], "language": "en-US", "depth": "intermediate"}'
)
```

**input parameter format (JSON string; must be passed via `input`):**
- `concept` (required): the concept page object, containing `slug`, `title`, `summary`, `content`
- `entities` (optional): array of linked entities; each item contains `slug`, `title`, `summary`, `link_type`
- `language` (optional): `zh-CN|en-US`; defaults to `en-US`
- `depth` (optional): teaching depth `beginner|intermediate|advanced`; defaults to `intermediate`

**Batching**:
- You may batch multiple concepts into a single `concept-to-requirement.py` call (pass an array) to reduce MCP round trips

**Batch manifest**:
- Write a manifest JSON recording each concept's generation status; on failure, resume from the checkpoint: skip the concepts in `generated`, and start from the first one in `pending`.

**Key constraints**:
- wiki reads and script conversion may be batched
- OpenMAIC API generation concurrency=1
- concept.Summary anchors the core of the requirement
- entities only provide title + summary, not full content
- A concept with no linked entities can still generate a classroom (it just lacks a practice segment)

### Phase 2: Build the Generation Request

Build the request body based on the input source; **field reference**:

| Field | Type | Required | Description |
|------|------|------|------|
| `requirement` | string | yes | teaching topic description, 1-2 sentences |
| `pdfContent` | object | no | PDF parsed text and images |
| `language` | string | no | `"zh-CN"` or `"en-US"`; defaults to `"en-US"` |
| `enableWebSearch` | bool | no | whether to enable web search; defaults false |
| `enableImageGeneration` | bool | no | whether to generate illustrations; defaults false |
| `enableVideoGeneration` | bool | no | whether to generate videos; defaults false |
| `enableTTS` | bool | no | whether to generate voice narration; defaults false |
| `agentMode` | string | no | `"default"` or `"generate"`; defaults `"default"` |

Scenario adaptation:
- **Scenario 1 (pure requirement)**: `requirement` uses the user's description directly
- **Scenario 2 (RAG results)**: `requirement` uses the `requirement` field from the Phase 1.1 script output
- **Scenario 3 (PDF)**: `requirement` is built from the text extracted from the PDF; `pdfContent` receives the parsed result
- **Scenario 4 (concept graph traversal)**: `requirement` uses the `requirement` field from the Phase 1.2 script output; call the API once per concept

### Phase 3: Call the OpenMAIC API

**Preferred approach**: call the API directly through the MCP tool registered in WeKnora.

**Step 1: Identify the HTTP request tool**
- Among your available MCP tools, find the one used for HTTP requests
- Tool names follow the format `mcp_{service_name}_{tool_name}` (e.g. `mcp_mcp_api_requester_make_request`)
- Identify it by the tool description: look for keywords such as "HTTP request", "API", "GET/POST"
- If no HTTP-request MCP tool is found, guide the user to deploy mcp_api_requester (see MCP availability check)

**Step 2: Determine the Base URL and auth header**

| Mode | Base URL | Auth Header |
|------|----------|-------------|
| Hosted mode | `https://open.maic.chat` | `Authorization: Bearer <access-code>` |
| Local mode | the address provided by the user (already replaced `localhost`/`127.0.0.1` with `host.docker.internal`) | none |

**Step 3: Feature detection (before sending optional features)**

Before sending the generation request, first query `GET /features` (or the equivalent endpoint) to check whether the mode supports optional features:
- If a requested optional feature is not supported, skip it: do not pass that field, do not mention it to the user, and do not question the user
- In local mode the backend may support fewer features; adjust based on the feature detection result

**Handling when the MCP tool is unavailable**:

Tell the user:

> The `mcp_api_requester` MCP service was not detected. Please download the code from https://github.com/yryuu/mcp-api-requester, deploy it, and then register this MCP service in WeKnora.

### Phase 4: Poll the Task Progress

Once the API returns `jobId` and `pollUrl`, follow this flow:

**1st poll (immediately after submission)**:
1. Call the HTTP request tool `GET {pollUrl}` to get the current status
2. Check `status`:
   - If `succeeded` → proceed to Phase 5
   - If `failed` → report the error and stop
   - If `queued` or `running` → **stop polling and tell the user**:

     > The course is being generated and will take about 2-10 minutes. Ask me again later for the progress.
     > Job ID: {jobId}

**When the user asks about progress (2nd poll)**:
1. Call `GET {pollUrl}` again
2. Check `status`:
   - If `succeeded` → proceed to Phase 5
   - If `failed` → report the error and stop
   - If still `queued` or `running` → **stop polling and tell the user to keep waiting**:

     > The course is still being generated. Please try again later.
     > Job ID: {jobId}

**Important rules**:
- Poll only **once** after submission; do not keep polling
- Poll only **once** when the user asks for progress; do not keep polling
- Only proceed when `status` is `succeeded` or `failed` — otherwise you must stop and tell the user to wait
- Do not try to resubmit the job — keep polling the same `pollUrl`

### Phase 5: Return the Result

After generation succeeds, return:

```
Classroom ID: <classroomId>
Classroom URL:
<BASE_URL>/classroom/<classroomId>
```

Hosted mode URL format: `https://open.maic.chat/classroom/<classroomId>`

> The URL must be printed as plain text on its own line — no bold, no code formatting, no Markdown link.

## Error Handling

| Error | Meaning | Handling |
|------|------|----------|
| Connection failure | network unreachable or the service is not running | check whether the Base URL is correct and the service is up |
| 401 | invalid access code (hosted mode) | tell the user to check or regenerate the access code on open.maic.chat |
| 403 | daily quota exhausted (hosted mode) | tell the user about the daily limit of 10 generations, resetting at midnight |
| 500 | server error | suggest retrying later or switching to local mode |
| Provider config error | model / provider / auth issue | guide the user to check the configuration or contact the administrator |

## Multi-Document → Course Set

When the user wants to generate a course set from multiple documents / knowledge snippets:

1. Collect all document content
2. Generate a separate requirement for each document / topic
3. Call the generation API sequentially through the MCP tool (no parallelism, to avoid quota conflicts)
4. If the MCP tool is unavailable, tell the user to deploy mcp_api_requester first (see MCP availability check)
5. Summarize and return all Classroom URLs

## Concept Graph Traversal → Bulk Micro-Classrooms

When the user wants to generate classrooms in bulk from knowledge graph concepts (Scenario 4), follow the full Phase 1.2 flow.

**MVP course orchestration strategy**: one concept → one micro-classroom

**Course type annotation**: mark `micro-classroom` in the requirement

**Batch resumability**: generate a manifest JSON recording each concept's generation status, so a failure can resume from the checkpoint.

**Key constraints**:
- wiki reads and script conversion may be batched
- OpenMAIC API generation concurrency=1
- concept.Summary anchors the core of the requirement
- entities only provide title + summary, not full content
- A concept with no linked entities can still generate a classroom (it just lacks a practice segment)

## Notes

- Scripts run in a Docker sandbox; **network access is disabled by default in the sandbox**
- **You must call the OpenMAIC API through the WeKnora MCP tool** — do not provide curl commands as a fallback
- MCP tool names follow the format `mcp_{service_name}_{tool_name}`; identify the HTTP request tool by its description
- If the MCP tool is disabled or unavailable, tell the user to download the code from https://github.com/yryuu/mcp-api-requester, deploy it, and register this MCP service in WeKnora
- A single generation task takes about 2-10 minutes, depending on content complexity and optional features
- Hosted mode (open.maic.chat) allows at most 10 generations per day, independent of the Web UI quota
- If the user asks to generate a new course while the same job is still running, do not resubmit — first check the existing job status
