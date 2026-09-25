# Legacy docs migration record

From now on, product, deployment, API and development documentation is maintained in `website-docs/`. This record lets maintainers review the scope of the migration and is not published to the docs site. This change deletes 87 legacy hand-written documents that were migrated, duplicated or outdated, without keeping a second copy of the content. The old content can be traced through Git history; `docs/` only keeps the engineering resources listed below.

## Content added to the new site

| Legacy document (relative to docs) | New entry | Handling |
| --- | --- | --- |
| `QA.md`, `migration-troubleshooting.md` | [FAQ and Upgrade Troubleshooting](01-getting-started/05-troubleshooting.md) | Keeps symptom diagnosis and recovery flows, with feature details linked to existing chapters; corrects the claims that "failure always rolls back completely" and about mechanical force |
| `paradedb-upgrade.md` | [ParadeDB Upgrade](01-getting-started/06-paradedb-upgrade.md) | Keeps backup, image and extension SQL upgrade, rollback and verification entry points; historical test results are not treated as verification for this change |
| `sandbox-cluster.md`, `sandbox-docker-backend.md`, `sandbox-desktop.md`, `sandbox-protocol.md` | [Sandbox Deployment and Troubleshooting](06-development/04-sandbox-deployment.md) | Keeps the template, daemon, gateway, Redis, desktop and snapshot boundaries; explains that the old local backend has been removed and how the desktop host differs from server-side named configurations; drops the old Docker implementation comparison and unverified third-party deployment specs |
| `browser-skill-integration.md`, `browser-skill-production.md` | [Local Browser](05-clients/09-local-browser.md) | Adds pairing, human involvement, task recovery, matching builds, proxy and multi-replica requirements; distinguishes it from the knowledge assistant extension |
| `agent-prompt-assembly.md` | [Conversation Prompt Assembly](06-development/05-agent-prompts.md) | Keeps the current assembly entry point, template saving and message boundaries; omits round-by-round review notes |
| `mcp-tool-directory.md` | [MCP Integration](03-features/08-mcp.md#mcp-tool-directory), [MCP API](04-api/02-api-agent-mcp.md) | Replaces the old full-registration description, adding the persistent directory, principal isolation, on-demand loading and sync endpoints; fixes the old document's contradictory statements about refresh behavior |
| `worker-pool-governance.md` | [Async Task Capacity](02-architecture/05-async-tasks.md#capacity-planning) | Follows the new site's more recent queue topology, adding aggregate configuration retirement, capacity estimation and downstream quota boundaries |
| `embed-subdomain.md`, `embed-secure-mode.md` | [Embed Channel](03-features/13-embed-channel.md#embed-subdomain) | The secure mode core was already covered; adds the standalone subdomain, runtime address and proxy policy; does not migrate the pseudo-login check example that only tested whether a Cookie/Header exists |
| `wiki/集成扩展/飞书云盘数据源接入说明.md` | [Feishu Drive Integration](03-features/24-feishu-drive.md) | Adds folder authorization and parsing modes; corrects, per the current implementation, the default export, blocks fallback and image conditions, deletion detection based on the old cursor, and failure recovery boundaries |
| `dev/opensearch-integration-test.md` | [Retrieval Engines: Local Testing](03-features/05-retrieval-engines.md#opensearch-local-testing) | Merged into the existing page; adds the development cluster, SSRF and verification flow, and clarifies that a replica count of 0 currently falls back to 1; the cleanup command now stops targeted services to avoid deleting other development data volumes |
| `cloud-image/README.md`, `cloud-image/tencent-lighthouse.md` | [Development Guide: Cloud Image Scripts](06-development/01-dev-guide.md#cloud-image-scripts) | Only keeps script boundaries and first-boot recovery, without a separate deployment tutorial; does not carry over old cloud platform quotas, prices, review times or fixed console paths |

## Already covered by the new site, not copied in full

The comparison below is by topic; "covered" means the working features and development entry points already have a home, not that every old example, screenshot or function-by-function walkthrough is kept.

| Legacy document | Maintained in |
| --- | --- |
| `开发指南.md`, `快速开发模式说明.md` | [Development Guide](06-development/01-dev-guide.md) |
| `LITE.md` | [Installation & Deployment](01-getting-started/02-installation.md), [Desktop Client](05-clients/05-desktop.md); the release package README still has a separate dependency, see below |
| `CHUNKING.md` | [Chunking](03-features/04-chunking.md) |
| `KnowledgeGraph.md`, `开启知识图谱功能.md` | [Knowledge Graph](03-features/09-knowledge-graph.md) |
| `使用其他向量数据库.md` | [Retrieval Engines](03-features/05-retrieval-engines.md), [Extension Points](06-development/03-extension-points.md) |
| `添加新的网络搜索引擎.md` | [Web Search](03-features/11-web-search.md), [Extension Points](06-development/03-extension-points.md) |
| `数据源导入开发文档.md` | [Data Source Sync](03-features/10-datasource.md), [Extension Points](06-development/03-extension-points.md); the cloud drive walkthrough gets a new page |
| `IM集成开发文档.md` | [IM Integration](03-features/12-im-integration.md), [Extension Points](06-development/03-extension-points.md); old platform console screenshots/bulk permission lists are not reused directly |
| `OIDC认证调用流程.md`, `RBAC说明.md`, `共享空间说明.md` | [Authentication & Authorization](03-features/01-tenant-auth.md), [Auth API](04-api/02-api-auth.md), [Organization API](04-api/02-api-org.md) |
| `BUILTIN_MODELS.md` | [Model Management](03-features/06-models.md), [Platform Management](03-features/20-platform-admin.md) |
| `BUILTIN_MCP_SERVICES.md`, `MCP功能使用说明.md`, `zh/mcp-approval.md` | [MCP Integration](03-features/08-mcp.md) |
| `Langfuse集成.md`, `日志配置.md` | [Observability](03-features/16-observability.md), [Configuration Reference](01-getting-started/04-configuration.md) |
| `agent-skills.md`, `agent-tools-design.md` | [Skills & Sandbox](03-features/22-skills-sandbox.md), [Agent Engine](03-features/07-agent.md), and the new sandbox deployment page; old tool names, local configuration and the outdated regular-user/read-only image contract are not migrated |
| `chat-steering.md` | [Chat Experience](03-features/18-chat-experience.md), [Session API](04-api/02-api-chat.md) |
| `client-integration-upgrade-notes.md` | [CLI](05-clients/02-cli.md), [Go SDK](05-clients/03-go-sdk.md), [Mini Program](05-clients/04-miniprogram.md), [File Access](03-features/21-file-access.md) |
| `api/*.md` | [API Overview](04-api/01-api-overview.md) and the topic pages in the same directory; adds the missing MCP metadata and sandbox desktop endpoints, and no longer maintains a second hand-written API reference |
| Other pages in `wiki/` | Summary copies of the topics above, not migrated wholesale; their old navigation, backlinks and graphs do not enter the new site |

## Not migrated as current product documentation

- `ROADMAP.md`: the old plan includes capabilities that are already implemented and cannot be treated as a current commitment; the entry point now goes to the existing product introduction, and plans should be reconfirmed before being written.
- `code-slimming-audit.md`: a one-off code slimming audit; the content is deleted and kept only through Git history.
- `plans/2026-09-10-faq-enabled-filter-design.md`: a historical design process; the content is deleted, the final interface is covered by the FAQ/API chapters, and the design process can be found in Git history.
- `poc/docker-sandbox/`: a standalone Go module and old Docker feasibility check, not user documentation; it can later move to a development experiments directory and is kept out of the site.

## Engineering resources kept in the docs directory

1. **Swagger generated package**: `internal/router/router.go` imports `github.com/Tencent/WeKnora/docs`, and the `docs` target in the `Makefile` writes its output to that directory. `docs.go` currently takes part in the backend build, and `swagger_contract_test.go` also reads the JSON/YAML. Regular builds and CI do not generate them automatically, so the three generated artifacts and the test are kept for now. In the future they can move to a standalone generated package, updating imports, generation paths, tests and lint exclusions; if the generated artifacts are no longer committed, a pinned-version generation step must first be wired into every build, test and release entry point.
2. **Lite release README**: `scripts/package-lite.sh` and `.github/workflows/release-lite.yml` copy `docs/LITE.md` into the offline release package; when moving it out, keep a README suitable for offline reading, rather than replacing it directly with a long page full of site-relative links.
3. **Images**: the multilingual READMEs, Helm and others still reference `docs/images/` and `docs/assets/`, so they are kept this time. The new site's own images live in `public/` and do not depend on the old directories.
4. **Historical experiments**: the `poc/docker-sandbox/` standalone Go module and its run instructions are kept; it is not part of the maintained product documentation. The old hand-written content (including the API, Wiki copies, roadmap, audits and design records) has been deleted.

Clickable documentation links in environment variable examples, frontend help and chunking examples, Helm install notes, sample projects, code comments and the CHANGELOG now point to the new site. Historical text in the CHANGELOG describing which files older versions added is kept as is and is not used as a current documentation entry point. `scripts/cloud-image/README.md` is reduced to an entry point into the new site to avoid maintaining a duplicate tutorial.

## Scope of this review

The migration is based on the current repository's routes, handlers, sandbox/browser implementation, data source services, migration SQL and deployment scripts. The site ran link checks, Mermaid checks and the documentation build; this documentation cleanup did not run production database upgrades, cloud image cleanup, real sandbox clusters or IM platform integrations.

## Accuracy and necessity review

Six kinds of new standalone pages are kept in this change: general troubleshooting, in-place ParadeDB upgrade, Feishu Drive integration, local browser, sandbox deployment and prompt maintenance. They respectively add cross-chapter troubleshooting, stateful upgrades, authorization and parsing choices, standalone client deployment, sandbox operations and the development contract; existing pages keep only summaries and links.

- OpenSearch local testing is merged into Retrieval Engines and cloud image maintenance into the Development Guide, avoiding standalone tutorials with little new content or no real-world testing yet.
- The prompt page was checked against `prompts.go`, `observe.go`, `finalize.go` and `config/agent_prompts.go`; the old Agent chapter's date format, MCP mention scope and wrap-up message role were corrected at the same time, and the duplicated description of the browser's internal protocol was removed.
- Feishu Drive was checked against the shared `core/engine.go`, `core/shared.go` and the data source service; it no longer promises that a full sync can catch up on deletions. Third-party permission requests are covered by the official API documentation, without fixing a permission list that was not verified on the platform in this change.
- OpenSearch was checked against the driver configuration and index implementation; the invalid zero-replica example was removed, and the old page's single-index dimension description was corrected.
- Database upgrade steps distinguish production and development Compose; the sandbox docs distinguish the desktop host from the three server-side named configurations; the cloud image notes state the fixed systemd paths and when the first-boot marker is actually written.

The above is a review of source code and configuration; it is not equivalent to real platform integration or production upgrade acceptance.
