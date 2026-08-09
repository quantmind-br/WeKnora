# MCP Tool Manual Review (Dangerous Calls)

Corresponding requirement: the agent can be interrupted before calling an MCP tool, pending manual confirmation before execution (GitHub #1173).

## Behavior

1. After a successful connection test in **Settings → MCP**, toggle **"Requires manual review"** on the tool list to flag that tool.
2. If the Agent is about to call a flagged tool during runtime, it pushes a `tool_approval_required` event, and the chat interface displays an approval card (with editable JSON parameters).
3. Once the user **approves** or **rejects**, the backend resumes execution; on rejection, the tool returns an error message to the model, and the remote MCP is not called.
4. If left unhandled past the timeout (default 10 minutes, adjustable via the `agent.tool_approval_timeout_seconds` configuration), it is treated as a rejection.

## Configuration Example

Choose one:

**1. config.yaml**

```yaml
agent:
  tool_approval_timeout_seconds: 600  # optional, default 600 (seconds)
```

**2. Environment variable** (takes precedence over yaml)

```bash
# Supports plain seconds or Go duration (30s / 5m / 1h)
WEKNORA_AGENT_TOOL_APPROVAL_TIMEOUT=600
```

## API

- `GET /api/v1/mcp-services/:id/tool-approvals` — List saved review configurations
- `PUT /api/v1/mcp-services/:id/tool-approvals/:tool_name` — Set whether a given tool requires review (`{"require_approval": true}`)
- `POST /api/v1/agent/tool-approvals/:pending_id` — Submit a result from the approval card
  - body: `{"decision":"approve"|"reject","modified_args":{...} optional,"reason":"..." optional}`

## Deployment and Limitations

- **Approval wait state is stored in process memory**: `pending_id` is only valid for the current instance; if the process restarts, in-progress waits will fail (manifesting as rejection/cancellation).
- **Multi-replica deployment**: when `REDIS_ADDR` is configured, `Resolve` forwards across instances via Redis Pub/Sub (channel `weknora:mcp_approval:resolve`), so the SSE connection and the HTTP request submitting the approval can land on different instances and still correctly wake the waiter; without Redis configured, it falls back to single-instance mode and requires sticky sessions.
- **Approval waits are not canceled by the tool's default 60s timeout**: the approval phase uses a round-level ctx (without `defaultToolExecTimeout`), and is governed only by `agent.tool_approval_timeout_seconds` and request-level cancellation.
- Security boundary: parameters submitted after approval are still submitted under the currently logged-in tenant space; only grant "approve" permission in a trusted environment.
