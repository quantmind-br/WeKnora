# Requirement Building Guide

Converts WeKnora RAG retrieval results into the `requirement` format needed by OpenMAIC course generation.

## Core Principles

The `requirement` field of OpenMAIC needs to be a **structured instructional requirement description**, not a raw document snippet. When building it, consider:

1. **Teaching topic**: what exactly to teach
2. **Target audience**: who you are teaching (e.g., beginners, professionals, students)
3. **Teaching depth**: introductory, intermediate, advanced
4. **Content scope**: which knowledge sources it is based on

## Conversion Templates

### Template 1: Pure Requirement (no retrieval results)

When the user describes the need directly, use their description as the requirement:

```
User: "Help me create an introductory course about quantum mechanics"
→ requirement: "Create an introductory classroom on quantum mechanics for beginners"
```

### Template 2: Based on RAG Retrieval Results

```
Steps:
1. Use knowledge_search to retrieve relevant knowledge
2. Extract from the retrieval results:
   - Core topics / concepts
   - Key knowledge points
   - Document source information
3. Build a structured requirement
```

Build format:
```
Based on the following knowledge content, create a [depth-level] course for [target audience]:

Core topic: [main concepts extracted from the retrieval results]
Key knowledge points:
- [knowledge point 1]
- [knowledge point 2]
- ...
Content sources: [list of document names]
```

### Template 3: Based on a Single Document

```
Based on the content of the document "[Document Name]", create a course for [target audience],
focusing on the following aspects:
- [focus area 1 specified by the user]
- [focus area 2 specified by the user]
```

### Template 4: Based on Multiple Documents / Knowledge Chunks

```
Combining the content of the following documents, create a systematic course:

Document 1 "[Name 1]": [brief content summary]
Document 2 "[Name 2]": [brief content summary]
Document 3 "[Name 3]": [brief content summary]

Requirements:
- Teaching depth: [level]
- Target audience: [description]
- Key coverage: [list of key topics]
```

### Template 5: Concept Graph Traversal (Concept Graph)

Use this template when generating micro-classrooms from knowledge graph concept pages and their linked entities. It is generated automatically by `scripts/concept-to-requirement.py`.

**Input structure**:
```json
{
  "concept": { "slug": "concept/rag", "title": "RAG retrieval-augmented generation", "summary": "...", "content": "..." },
  "entities": [
    { "slug": "entity/vector-db", "title": "Vector database", "summary": "...", "link_type": "outlink" },
    { "slug": "entity/embedding", "title": "Embedding model", "summary": "...", "link_type": "bidirectional" }
  ],
  "language": "en-US",
  "depth": "intermediate",
  "audience": "learners in the relevant field"
}
```

**Output requirement structure**:
```
Based on the knowledge graph concept "[concept.title]", create a [depth] micro-classroom for [audience].

Teaching anchor: [concept.summary]

Learning objectives:
  - understand [key sentence from concept.summary]

Core knowledge points:
  - [definitions/mechanisms parsed from concept.content]

Linked entities (practice segment):
  - Case study: [entity.title]: [entity.summary]
  - Tool: [entity.title]: [entity.summary]
  - Application scenarios: [entity.title]: [entity.summary]
  - Prerequisites: [entity.title]

Practice tasks:
  - practice the application of [concept.title] through [entity.title]

Common misconception check:
  - [misconceptions parsed from concept.content]

Assessment prompts:
  - please explain [concept.title]
  - apply [concept.title] to a real scenario
```

### Template 6: Technical Document → Course

```
Retrieval results:
- Document: "Kubernetes Deployment Guide.pdf"
- Key content: Pod management, Service configuration, Ingress routing, storage volumes

Built requirement:
"Based on the Kubernetes deployment guide, create an intermediate course for DevOps engineers.
Focus on: Pod lifecycle management, Service and Ingress network configuration, and persistent storage volume management.
The course should include hands-on practice."
```

### Example 2: Product Manual → Introductory Course

```
Retrieval results:
- Document: "Product User Manual v2.0.pdf"
- Key content: product overview, quick start, core features, FAQs

Built requirement:
"Based on the product user manual v2.0, create an introductory course for new users.
Help users quickly understand the product's core features, master the basic operations,
and complete common tasks independently. The course language should be English."
```

### Example 3: Research Paper → Advanced Course

```
Retrieval results:
- Document: "Transformer Architecture Survey.pdf"
- Key content: Attention mechanism, positional encoding, multi-head attention, training tips

Built requirement:
"Based on the Transformer architecture survey, create an advanced course for researchers
with a deep learning background. Cover in depth the mathematical principles of the
Attention mechanism, the variants of positional encoding, the design rationale behind
multi-head attention, and practical tips for training large models."
```

## Optional Feature Configuration Suggestions

When building a request, recommend optional features based on the user's needs:

| Scenario | Recommended features |
|------|----------|
| Technical training | `enableWebSearch: true` (supplement with the latest technical developments) |
| Product introduction | `enableImageGeneration: true` (generate product screenshots / interface images) |
| Marketing | `enableImageGeneration: true, enableVideoGeneration: true` |
| Language teaching | `enableTTS: true` (voice narration) |
| Academic research | default configuration is fine (no multimedia needed) |

## Multi-Document Handling

When several independent documents each need their own course:

1. Build a separate requirement for each document
2. Call the generation API sequentially (no parallelism)
3. Return the URL as soon as each is complete, then continue with the next
4. Finally, summarize all Classroom URLs

> Note: OpenMAIC hosted mode allows at most 10 generations per day; local mode depends on the LLM Provider's quota.
