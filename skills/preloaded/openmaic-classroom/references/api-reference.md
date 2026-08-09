# OpenMAIC API Reference

Specification for the OpenMAIC service API. All API endpoints share the base path `{base_url}/api`, defaulting to `http://localhost:3000/api`.

## Authentication

- If OpenMAIC does not set the `ACCESS_CODE` environment variable: all endpoints are open
- If `ACCESS_CODE` is set: obtain a cookie first via `/api/access-code/verify`
- `/api/health` never requires authentication

### Obtaining the Authentication Cookie

```
POST {base_url}/api/access-code/verify
Content-Type: application/json

{ "code": "<ACCESS_CODE>" }
```

On success it returns `{ "success": true, "valid": true }` and sets the `openmaic_access` cookie (valid for 7 days).

## Standard Response Format

**Success**:
```json
{ "success": true, ...additionalFields }
```

**Error**:
```json
{ "success": false, "errorCode": "<Code>", "error": "<message>", "details?": "<detail>" }
```

---

## Core Endpoints

### 1. Health Check

```
GET {base_url}/api/health
```

Response:
```json
{
  "success": true,
  "status": "ok",
  "version": "0.1.0",
  "capabilities": {
    "webSearch": true,
    "imageGeneration": false,
    "videoGeneration": false,
    "tts": true
  }
}
```

> `capabilities` is used for feature detection; only enable the corresponding optional feature when this field is present.

---

### 2. Generate Course (async job)

#### 2a. Create a Generation Job

```
POST {base_url}/api/generate-classroom
Content-Type: application/json
```

Request body:
```json
{
  "requirement": "teaching topic description",
  "pdfContent": { "text": "PDF text content", "images": [] },
  "enableWebSearch": false,
  "enableImageGeneration": false,
  "enableVideoGeneration": false,
  "enableTTS": false,
  "agentMode": "default"
}
```

Success response (202):
```json
{
  "success": true,
  "jobId": "abc123xyz",
  "status": "queued",
  "step": "queued",
  "message": "Classroom generation job queued",
  "pollUrl": "{base_url}/api/generate-classroom/abc123xyz",
  "pollIntervalMs": 5000
}
```

Errors:
- `400`: the `requirement` field is missing
- `500`: internal error

#### 2b. Poll the Job Status

```
GET {pollUrl}
```

Response:
```json
{
  "success": true,
  "jobId": "abc123xyz",
  "status": "running",
  "step": "generating_outlines",
  "progress": 35,
  "message": "Generating scene outlines...",
  "pollUrl": "...",
  "pollIntervalMs": 5000,
  "scenesGenerated": 2,
  "totalScenes": 6,
  "result": null,
  "error": null,
  "done": false
}
```

Final success response:
```json
{
  "success": true,
  "status": "succeeded",
  "result": {
    "classroomId": "Uyh82Y32ZK",
    "url": "{base_url}/classroom/Uyh82Y32ZK",
    "scenesCount": 6
  },
  "done": true
}
```

Final failure response:
```json
{
  "success": true,
  "status": "failed",
  "error": "specific error message",
  "done": true
}
```

---

### 3. Parse PDF

```
POST {base_url}/api/parse-pdf
Content-Type: multipart/form-data
```

Form fields:
- `pdf` (file, required): the PDF file
- `providerId` (optional): PDF provider; defaults to `"unpdf"`
- `apiKey` (optional): the provider API key
- `baseUrl` (optional): the provider base URL

Response:
```json
{
  "success": true,
  "data": {
    "text": "extracted text content",
    "images": [],
    "metadata": {
      "pageCount": 10,
      "fileName": "document.pdf",
      "fileSize": 1024000
    }
  }
}
```

---

### 4. Course Storage

#### 4a. Fetch a Course

```
GET {base_url}/api/classroom?id=<classroomId>
```

#### 4b. Persist a Course

```
POST {base_url}/api/classroom
Content-Type: application/json

{ "stage": {...}, "scenes": [...] }
```

---

### 5. Web Search

```
POST {base_url}/api/web-search
Content-Type: application/json

{ "query": "search keywords", "pdfText": "PDF context", "apiKey": "Tavily Key" }
```

---

### 6. Verify the LLM Connection

```
POST {base_url}/api/verify-model
Content-Type: application/json

{ "model": "openai:gpt-4o", "apiKey": "...", "baseUrl": "..." }
```

---

## Generation Pipeline (step-by-step calls)

For finer-grained control, use the following step-by-step generation endpoints instead of `/api/generate-classroom`:

### A. Generate the Scenario Outline (SSE stream)

```
POST {base_url}/api/generate/scene-outlines-stream
```

Request body:
```json
{
  "requirements": { "requirement": "topic", "userNickname": "user nickname" },
  "pdfText": "PDF text",
  "pdfImages": [],
  "researchContext": "web search results",
  "agents": []
}
```

Headers:
- `x-image-generation-enabled`: `"true"` or `"false"`
- `x-video-generation-enabled`: `"true"` or `"false"`

SSE event types: `languageDirective`, `outline`, `done`, `error`, `retry`

### B. Generate the Scenario Content

```
POST {base_url}/api/generate/scene-content
Content-Type: application/json

{
  "outline": {...},
  "allOutlines": [...],
  "stageId": "stage-1",
  "pdfImages": [],
  "agents": [],
  "languageDirective": "Use Simplified Chinese"
}
```

### C. Generate the Scenario Actions

```
POST {base_url}/api/generate/scene-actions
Content-Type: application/json

{
  "outline": {...},
  "allOutlines": [...],
  "content": {...},
  "stageId": "stage-1",
  "agents": [],
  "languageDirective": "Use Simplified Chinese"
}
```

### D. Generate the Agent Persona

```
POST {base_url}/api/generate/agent-profiles
Content-Type: application/json

{
  "stageInfo": { "name": "beginner stage", "description": "..." },
  "sceneOutlines": [{ "title": "...", "description": "..." }],
  "languageDirective": "Use Simplified Chinese",
  "availableAvatars": [],
  "avatarDescriptions": []
}
```
