# Claw Skill

Claw Skill gives agents in the OpenClaw ecosystem access to WeKnora, uploading documents, importing web pages, and searching across knowledge bases through the REST API.

The Skill is hosted on ClawHub, package name [`@lyingbug/weknora`](https://clawhub.ai/lyingbug/weknora), licensed under MIT-0. Its capabilities and permissions are determined by the WeKnora API it connects to.

## Key features {#what-it-can-do}

| Capability | Corresponding interface |
| --- | --- |
| Upload files | Feed PDF / Word / Excel and other documents into the knowledge base with automatic parsing and vectorization |
| Import web pages | Crawl body content by URL into the knowledge base, with support for polling parse status |
| Write Markdown | Create or edit knowledge entries in Markdown, suited to meeting notes and structured memos |
| Hybrid search | Single-base `hybrid-search` and cross-base `knowledge-search`, combining vector + keyword recall |
| Browse knowledge bases | List knowledge bases and entries, view details |

## Installation and connection {#how-to-configure-it}

Open "Settings → Publish & Integrations → Claw Skill" to see the setup guide and copy the current instance's API address, an environment variable example, and the install command.

1. **Get API credentials**: copy the API Key and API address from "Settings → Publish & Integrations → API Integration" (the "Open API Info" button on the setup guide jumps straight there);
2. **Set environment variables**: in your terminal or `~/.zshrc` / `~/.bashrc`, set

   ```bash
   export WEKNORA_BASE_URL=https://your-weknora.example.com/api/v1
   export WEKNORA_API_KEY=sk-xxxxx
   ```

   `WEKNORA_BASE_URL` must include `/api/v1`; the example on the setup guide is already filled in with the current instance address.

3. **Install the Skill**: in an environment with the OpenClaw CLI installed, run the command below, or follow the instructions on the ClawHub page to install it;

   ```bash
   openclaw skills install @lyingbug/weknora
   ```

4. **Verify**: have the Agent list knowledge bases once or run a search, to confirm credentials and network connectivity are working.

## Relationship with MCP

Both Claw Skill and the MCP Server can be called by external agents; which one to choose depends on the access methods the client supports and the capabilities you need:

| | Claw Skill | MCP Server（内置） |
| --- | --- | --- |
| Aimed at | Agents in the OpenClaw / ClawHub ecosystem | Clients supporting the MCP protocol (Claude Desktop, Cursor, Claude Code, VS Code Copilot, etc.) |
| Installation | Install the Skill via ClawHub | No extra deployment: create an endpoint under "Settings → Publish & Integrations → MCP Server", and clients connect to `/mcp/<endpoint_id>` |
| Authentication | Space API Key (`WEKNORA_API_KEY`) | A separate token per endpoint, which can be rotated or disabled |
| Transport | Direct REST calls | Streamable HTTP; clients that only support stdio bridge through `mcp-remote` |
| Scope of capability | Import, search, browse (5 categories) | Selected per endpoint: retrieval and reading, Q&A (the endpoint's default Agent), Wiki, writes; can be limited to specific knowledge bases |
| Documentation | This page | [MCP Integration](../03-features/08-mcp.md) |

Use the MCP Server when you need Q&A or Wiki tools, or want to limit tools and knowledge base scope per endpoint; for importing, searching, and browsing material in the OpenClaw ecosystem, use Claw Skill. The Python MCP service under `mcp-server/` in the repository is deprecated; new integrations should use the built-in MCP Server.

## Related documentation {#related}

- Narrowing credentials and capabilities: [Tenants, Users, and Authentication & Authorization](../03-features/01-tenant-auth.md)
- Underlying interface: [API Overview](../04-api/01-api-overview.md)
- Other integration methods: [Chrome Extension](06-chrome-extension.md), [MCP Integration](../03-features/08-mcp.md)
