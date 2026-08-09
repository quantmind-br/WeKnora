# WeKnora Roadmap

This document describes WeKnora's product plans and roadmap direction, and will be continuously updated as the project progresses.

## Lightweight Deployment
- [ ] WeKnora officially provides atomic invocation interfaces (Embedding, ReRank, LLM, document parsing, etc.), with a certain amount of free usage quota
- [ ] WeKnora officially provides a complete cloud service, allowing users to directly experience WeKnora's capabilities on the platform
- [ ] Launch a WeKnora Lite version for users without strong private deployment needs to quickly experience the product's capabilities

## Knowledge Understanding
- [x] Abstract the overall document parsing module, supporting switching between built-in parsing, MinerU, or other parsing methods
- [ ] Optimize the document chunking strategy — in addition to rule-based chunking, support semantic chunking, section-based chunking, etc.
- [ ] Document structure visualization: display the parsed document's section structure, graph relationships, etc.
- [ ] Support more document formats such as audio and video, enhancing multimodal understanding capabilities

## Retrieval and Summarization
- [ ] Support specifying the retrieval scope via "@tags" in the input box
- [ ] Support uploading images and attachments for retrieval in the input box
    - [x] Support uploading images in the chat dialog
    - [ ] Support uploading attachments in the chat dialog

## Knowledge Base-Related Model Training
- [ ] Train models related to retrieval and recall (Embedding, ReRank, LLM, etc.)
- [ ] Continue exploring document parsing and document understanding, advancing self-developed related models

## Knowledge Base Forms
- [ ] Expand knowledge base forms, supporting storage and indexing of time-series data
- [ ] Explore application scenarios combining knowledge bases with Memory

## IM Integration
- [x] Support integration with IM systems such as WeCom (企微) and Feishu, enabling use of WeKnora's capabilities within IM

## Components and Extensions
- [ ] Encourage the community to maintain components such as model services and web search services for various vendors
- [ ] Encourage the community to provide more Skills related to knowledge bases

## Surrounding Ecosystem Development
- [ ] Provide a Chrome extension supporting a "clipping"-like feature, saving web page content to the knowledge base with support for retrieval, summarization, and Q&A
- [ ] Provide a mini-program plugin (specific form to be determined)
- [ ] Provide a JS SDK to facilitate integrating WeKnora's capabilities into web pages
- [ ] Encourage the community to provide editor/IDE plugins for VSCode, Cursor, Claude Code, etc.

## Documentation Development
- [ ] Improve official documentation (usage instructions, API, deployment, etc.)
- [ ] Encourage users to contribute documentation, blogs, videos, etc., forming a community-driven documentation ecosystem
- [ ] Build a WeKnora content collection on the Zhihu (知乎) platform
