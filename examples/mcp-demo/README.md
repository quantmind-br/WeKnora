# WeKnora Local MCP Demo

Minimal external MCP server for testing WeKnora **as an MCP client** connecting to third-party tools.

Provides 6 demo tools:

| Tool | Purpose |
| --- | --- |
| `echo` | Connectivity self-check |
| `add` | Add two numbers |
| `server_time` | Return server UTC time |
| `lookup_policy` | Query demo policies (warranty, reimbursement, POC, etc., consistent with `website-docs/sample-data/`) |
| `list_team_contacts` | List demo project team members |
| `send_demo_alert` | Simulate an outbound notification (suitable for testing manual approval) |

## 1. Startup

```bash
cd examples/mcp-demo
chmod +x start.sh
./start.sh
```

`start.sh` automatically creates a `.venv` and installs dependencies. By default it listens on `http://127.0.0.1:8010/mcp`, with auth token `weknora-demo-token`.

Customization:

```bash
export MCP_SERVER_AUTH_TOKEN=my-secret
export MCP_PORT=9000
./start.sh
```

## 2. Self-check

In a separate terminal:

```bash
cd examples/mcp-demo
source .venv/bin/activate
python test_tools.py
```

This should list the 6 tools.

## 3. Connecting to WeKnora

1. Open **Settings → MCP Services → New**
2. Fill in:

| Field | Value |
| --- | --- |
| Name | `Local MCP Demo` |
| Transport | **HTTP Streamable** |
| URL | `http://127.0.0.1:8010/mcp` |
| Auth | **Bearer** |
| Token | `weknora-demo-token` (must match `MCP_SERVER_AUTH_TOKEN`) |

3. After saving, click **Test Connection** — it should discover 6 tools.
4. In the **Agent** configuration, check this MCP service (or select all tools).
5. (Optional) Enable **manual approval** for `send_demo_alert` — during a conversation, the Agent will prompt for confirmation before invoking it.

## 4. Suggested questions to try

Ask in the Agent conversation:

- "Use the MCP tool to check how long the smart home hub's warranty lasts" → should trigger `lookup_policy`
- "Who is the POC owner in the R&D department" → `lookup_policy` or `list_team_contacts`
- "What time is it on the MCP Demo server right now" → `server_time`

If you've also imported the documents from `website-docs/sample-data/`, you can compare whether the **knowledge base retrieval answer** and the **MCP tool response** are consistent.

## 5. Notes

- The WeKnora UI **does not support stdio** transport; you must use **HTTP Streamable** or **SSE**.
- The demo only binds to `127.0.0.1` — do not expose it to the public internet.
- `send_demo_alert` does not actually send a message; it only returns a simulated result.

## 6. SSE mode (optional)

```bash
MCP_TRANSPORT=sse MCP_PORT=8011 ./start.sh
```

In WeKnora, select **SSE** as the transport and enter `http://127.0.0.1:8011/sse` as the URL.
