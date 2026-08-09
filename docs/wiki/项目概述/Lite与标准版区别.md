# Lite vs. Standard Edition Differences

Lite targets scenarios where quick local use and minimal deployment complexity are the priority; the Standard Edition targets multi-workspace collaboration and full enterprise capabilities. The main differences are as follows.

| Dimension | Lite | Standard Edition |
|------|------|--------|
| **Shared Workspace** | Does not provide a shared workspace (member invitations, cross-member sharing of knowledge bases and agents, etc.) | Provides a shared workspace and collaboration capabilities such as workspace-isolated retrieval |
| **Workspace and Account** | Single workspace; works out of the box, **no registration required** | Multiple workspaces; typically requires registration, login, and organization management |
| **Document Parsing** | Only the **Simple** parsing engine is built in; other parsing capabilities can be accessed via **Cloud** and similar methods | Multiple parsing engines can be configured (including high-precision options), integrated with the full document processing pipeline |
| **Deployment Model** | **Single application, zero dependencies** (does not rely on external service stacks such as a standalone database or message queue) | Typically a multi-service deployment via Docker Compose or similar, with more dependencies and components |
| **Data Ownership** | Data is stored and processed **entirely on the local machine** | For private deployments, data can also be kept local; this depends on your specific deployment method |
| **Network Exposure** | **Accessible from localhost only by default**; can be configured as needed, with the **option to expose it to the public internet** | Address binding and gateway configuration are handled according to your deployment and security policy |

> If you don't need multi-team collaboration, a complex parsing pipeline, or a multi-service architecture, Lite is better suited for individuals or small teams trying it out locally with zero dependencies. If you need a shared workspace, multiple workspaces, or the full parsing engine matrix, use the Standard Edition instead.

## Related Topics

- For details on the shared workspace, see [Shared Workspace Guide](../安全认证/共享空间说明.md) — a Standard Edition–only feature
- For the authentication system, see [OIDC Authentication Flow](../安全认证/OIDC认证调用流程.md) — multi-workspace OIDC login for the Standard Edition
- For setting up a development environment, see [Development Guide](../开发部署/开发指南.md)
- For deployment-related FAQ, see [FAQ](../运维排障/常见问题.md)

---

## Backlinks

- [Home](../Home.md) — Wiki home navigation
- [Roadmap](版本路线图.md) — the lightweight deployment direction in the roadmap
- [Shared Workspace Guide](../安全认证/共享空间说明.md) — details on the shared workspace feature not supported by Lite
- [OIDC Authentication Flow](../安全认证/OIDC认证调用流程.md) — the authentication method for multi-workspace scenarios in the Standard Edition
