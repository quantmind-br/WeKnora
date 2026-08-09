# WeKnora Lite vs. Standard Edition Differences

Lite is aimed at scenarios where you want to get up and running locally quickly with as simple a deployment as possible; the Standard Edition targets multi-space collaboration and full enterprise capabilities. The main differences are as follows.

| Dimension | Lite | Standard Edition |
| --------- | ---- | ----------------- |
| **Shared Spaces** | No shared spaces (member invitations, cross-member sharing of knowledge bases and agents, etc.) | Provides shared spaces and collaboration capabilities such as space-isolated retrieval |
| **Spaces & Accounts** | Single space; works out of the box, **no registration required** | Multiple spaces; typically requires registration, login, and organization management |
| **Document Parsing** | Ships with only the **Simple** parsing engine; other parsing capabilities can be accessed via **Cloud** and similar methods | Multiple parsing engines can be configured (including high-precision options), integrated with the full document processing pipeline |
| **Deployment Form** | **Single application, zero dependencies** (does not rely on an external service stack such as a standalone database or message queue) | Typically a multi-service deployment via Docker Compose or similar, with more dependencies and components |
| **Data Ownership** | Data is stored and processed **entirely locally** | For private deployments, data can also be kept local; the specifics depend on your deployment method |
| **Network Exposure** | **Local access only by default**; can be configured as needed to **choose whether to expose it to the public internet** | Address and gateway binding is up to you, based on your deployment and security policies |

If you don't need multi-team collaboration, complex parsing pipelines, or a multi-service architecture, Lite is better suited for individuals or small teams trying it out locally with zero dependencies. If you need shared spaces, multiple spaces, and the full matrix of parsing engines, use the Standard Edition instead.
