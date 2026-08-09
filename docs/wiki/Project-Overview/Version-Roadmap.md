---
title: Version Roadmap
tags: [Project Overview, Roadmap, Planning]
aliases: [ROADMAP, Roadmap]
source: ROADMAP.md
---

# Version Roadmap

This document describes WeKnora's product planning and intended direction, and will be continuously updated as the project progresses.

## Lightweight Deployment

- [ ] WeKnora officially provides atomic invocation interfaces (Embedding, ReRank, LLM, document parsing, etc.), along with a certain free usage quota
- [ ] WeKnora officially provides a complete cloud service, allowing users to experience WeKnora's capabilities directly on the platform
- [ ] Launch a WeKnora Lite version for users without strong private deployment needs to quickly experience the product's capabilities

> For detailed differences regarding the Lite version, see [Differences Between Lite and Standard Editions](Lite-vs-Standard-Edition.md)

## Knowledge Understanding

- [x] Abstract the overall document parsing module, supporting switching between built-in parsing, MinerU, or other parsing methods
- [ ] Optimize document chunking strategy, supporting semantic chunking, section chunking, etc., in addition to rule-based chunking
- [ ] Document structure visualization: display the parsed document's section structure, graph relationships, etc.
- [ ] Support more document formats such as audio and video, enhancing multimodal understanding capabilities

> For knowledge graph-related features, see [Knowledge Graph](../Core-Features/Knowledge-Graph.md) and [Enabling the Knowledge Graph Feature](../Core-Features/Enabling-Knowledge-Graph.md)

## Retrieval and Summarization

- [ ] Support specifying retrieval scope via "@tags" in the input box
- [ ] Support uploading images and attachments for retrieval in the input box
    - [x] Support uploading images in the dialog box
    - [ ] Support uploading attachments in the dialog box

> For retrieval engine extensions, see [Integrating a Vector Database](../Integration-Extension/Integrating-a-Vector-Database.md)

## Knowledge Base-Related Model Training

- [ ] Train models related to retrieval recall (Embedding, ReRank, LLM, etc.)
- [ ] Continue exploring document parsing and document understanding, advancing self-developed related models

> For model management, see [Built-in Model Management](../Core-Features/Builtin-Model-Management.md)

## Knowledge Base Forms

- [ ] Expand knowledge base forms to support storage and indexing of time-series data
- [ ] Explore application scenarios combining knowledge bases with Memory

## IM Integration

- [x] Support integration with IM systems such as WeCom and Feishu, enabling the use of WeKnora's capabilities within IM

> For details on IM integration development, see [IM Integration Development](../Integration-Extension/IM-Integration-Development.md)

## Components and Extensions

- [ ] Encourage the community to maintain components such as various vendors' model services and web search services
- [ ] Encourage the community to provide more knowledge base-related Skills

> For extension development, see [Adding a Web Search Engine](../Integration-Extension/Adding-a-New-Search-Engine.md), [Integrating a Vector Database](../Integration-Extension/Integrating-a-Vector-Database.md), [Agent Skills System](../Core-Features/Agent-Skills-System.md)

## Surrounding Ecosystem Building

- [ ] Provide a Chrome extension supporting a "clipping"-like feature
- [ ] Provide a mini-program plugin (specific form to be determined)
- [ ] Provide a JS SDK to facilitate integrating WeKnora's capabilities into web pages
- [ ] Encourage the community to provide editor/IDE plugins for VSCode, Cursor, Claude Code, etc.

## Documentation Building

- [ ] Improve official documentation (usage instructions, API, deployment, etc.)
- [ ] Encourage users to contribute documentation, blogs, videos, etc., forming a community-driven documentation system
- [ ] Build a WeKnora content collection on the Zhihu platform

---

## Backlinks

- [Home](../Home.md) — Wiki homepage navigation
- [Differences Between Lite and Standard Editions](Lite-vs-Standard-Edition.md) — The Lite version's positioning corresponds to the lightweight deployment direction in the roadmap
- [Built-in Model Management](../Core-Features/Builtin-Model-Management.md) — The model training direction in the roadmap relates to built-in model management
- [IM Integration Development](../Integration-Extension/IM-Integration-Development.md) — The IM integration milestone completed in the roadmap
