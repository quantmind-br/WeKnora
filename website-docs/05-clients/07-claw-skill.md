# Claw Skill

Claw Skill is a way to hook WeKnora up for AI Agents to use: once installed, Agents in the OpenClaw ecosystem can write content into knowledge bases and search across them through WeKnora's REST API.

The Skill is hosted on ClawHub, package name [`@lyingbug/weknora`](https://clawhub.ai/lyingbug/weknora) (MIT-0). It's a thin wrapper — the actual capability is WeKnora's REST interface.

## What it can do

| Capability | Corresponding interface |
| --- | --- |
| Upload files | Feed PDF / Word / Excel and other documents into the knowledge base with automatic parsing and vectorization |
| Import web pages | Crawl body content by URL into the knowledge base, with support for polling parse status |
| Write Markdown | Create or edit knowledge entries in Markdown, suited to meeting notes and structured memos |
| Hybrid search | Single-base `hybrid-search` and cross-base `knowledge-search`, combining vector + keyword recall |
| Browse knowledge bases | List knowledge bases and entries, view details |

## How to configure it

WeKnora's UI has a guided setup page: "Settings → Integrations → Claw Skill", which provides the current instance's API address along with a copyable environment variable example and install command. Steps:

1. **Get API credentials**: copy the API Key and API address from "Settings → API Info";
2. **Set environment variables**: in your terminal or `~/.zshrc` / `~/.bashrc`, set

   ```bash
   export WEKNORA_BASE_URL=https://your-weknora.example.com/api/v1
   export WEKNORA_API_KEY=sk-xxxxx
   ```

3. **Install the Skill**: in an environment with the OpenClaw CLI installed, run the install command shown on the guided setup page, or follow the instructions on the ClawHub page to install it;
4. **Verify**: have the Agent list knowledge bases once or run a search, to confirm credentials and network connectivity are working.

## Relationship with MCP

Both are about "making WeKnora available to external Agents" — which one to choose depends on the other side's ecosystem:

| | Claw Skill | MCP Server |
| --- | --- | --- |
| Aimed at | Agents in the OpenClaw / ClawHub ecosystem | Clients supporting the MCP protocol (Claude Desktop, VS Code Copilot, etc.) |
| Installation | Install the Skill via ClawHub | `pip install tencent-weknora-mcp` or run via `uvx` |
| Transport | Direct REST calls | stdio / SSE / Streamable HTTP |
| Scope of capability | Import, search, browse (5 categories) | 29 tools, also covering tenants, models, sessions, Agent Q&A, Wiki |
| Documentation | This page | [MCP Integration](../03-features/08-mcp.md) |

When you need fuller capability (running Agent conversations, managing models, reading the Wiki), use the MCP Server; if you just want Agents to store and access material, the Skill is lighter-weight.

## Related

- Narrowing credentials and capabilities: [Tenants, Users, and Authentication & Authorization](../03-features/01-tenant-auth.md)
- Underlying interface: [API Overview](../04-api/01-api-overview.md)
- Other integration methods: [Chrome Extension](06-chrome-extension.md), [MCP Integration](../03-features/08-mcp.md)
