## MCP Feature Usage Guide

### Feature Overview
- MCP (Model Context Protocol) lets WeKnora securely connect to external tools or data sources, extending the capabilities the Agent can call on during reasoning.
- All services are managed centrally in the frontend under `Settings > MCP Services` (`frontend/src/views/settings/McpSettings.vue`), with no need to manually edit configuration files.
- Each service includes a name, transport method (SSE / HTTP Streamable / Stdio), connection address or command, authentication information, and advanced timeout and retry policies.

### Entry Point and Interface
- Open `Settings -> MCP Services` in the left-hand console menu to see the full list of MCP services in the current workspace.
- From the list you can quickly enable/disable services, view descriptions, and use the right-side menu to run "Test / Edit / Delete".
- The "Add Service" button opens the `McpServiceDialog` for creating or modifying a service.

### Common Workflows
1. **Create a new service**
   - Click "Add Service", fill in a name and description, and choose the transport method.
   - SSE / HTTP Streamable require an accessible service URL; Stdio requires configuring a `uvx`/`npx` command and arguments, with optional environment variables.
   - Fill in the API Key, Bearer Token, timeout, and retry policy as needed — once saved, the service will appear in the list.
2. **Enable/disable a service**
   - Toggle the switch in the list to change the enabled state; the system immediately calls the backend `updateMCPService`. On failure, the state automatically rolls back and a prompt is shown.
3. **Connection test**
   - Select "Test" from the more-options menu. The frontend calls `/api/v1/mcp-services/{id}/test` and displays the `McpTestResult` popup.
   - On success, it shows the list of tools available from the service (including input schema) and the resource list; on failure, it displays the error message, making it easy to troubleshoot network or authentication issues.
4. **Edit / Delete**
   - "Edit" brings up the existing configuration — modify it and save.
   - "Delete" requires confirmation in the popup; once done, the list refreshes automatically.

### Usage Recommendations
- **Choosing a transport method**: Prefer SSE for a streaming experience; switch to standard HTTP Streamable when that compatibility is needed; Stdio is suitable for local debugging or offline environments, running the MCP Server on the same machine.
- **Authentication management**: Store API Keys/Tokens in "Authentication Configuration". In production, it's recommended to create a separate minimal-privilege key and rotate it regularly.
- **Retry policy**: For public internet or third-party services, increase `retry_count` and `retry_delay` appropriately to avoid the Agent being interrupted by intermittent timeouts.
