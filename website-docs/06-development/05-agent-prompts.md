# Conversation Prompt Assembly and Editable Scope

This page is for developers who maintain prompts, the agent editor and the model message pipeline. For agent configuration and the execution flow, see [Agent Engine](../03-features/07-agent.md).

## Smart reasoning system sections

The single assembly entry point is `BuildSystemPromptSections` in `internal/agent/prompts.go`; both the engine and the string-compatible interface `BuildSystemPromptWithOptions` use it. The sections are listed below in actual assembly order; empty sections are omitted from the output.

| Section | Source and responsibility |
| --- | --- |
| `base` | Explicit body text takes precedence; otherwise the `pure` default template is used without bound knowledge bases, and the `rag` default template with knowledge bases |
| `steering` | How to handle messages the user adds while an answer is in progress (supplement, modify or cancel), keeping unfinished tasks; also included with custom body text |
| `runtime_contract` | Data and instruction boundaries, the current document scope, completion conditions; appends the default answer language when the user's language is known |
| `sources` | Selects source rules according to the set of actually registered tools; skill installation mode uses dedicated installation verification instructions |
| `tools` | General execution, file and sandbox conventions; the specific parameters remain in the tool definitions |
| `output` | Common output format, image conditions and completion checks |
| `skills` | Available skill metadata; appended only outside skill installation mode, when `read_file` is available and metadata exists |
| `memory` | Memories recalled for this turn |
| `protocol` | Source handle and output citation protocol |

Custom body text only replaces `base`; it does not remove other runtime sections or grant additional tool permissions. Sections are not different permission levels of the model API; callable tools are still controlled by backend registration and the execution path. The browser operation contract stays in the tool description of `local_browser`; for integration and operation, see [Local Browser](../05-clients/09-local-browser.md).

Placeholders in `base` are expanded by `renderPromptPlaceholdersWithStatus`:

| Placeholder | Current behavior |
| --- | --- |
| `{{knowledge_bases}}` | Points to the knowledge base directory in `runtime_context` of the user message; the full text is not expanded |
| `{{web_search_status}}` | Expands to `Enabled` / `Disabled` depending on whether the `web_search` tool is actually registered |
| `{{current_time}}` | `YYYY-MM-DD` date |
| `{{language}}` | The user's language name |
| `{{skills}}` | Cleared; skill metadata is provided by a separate section |

## Current-turn context and message roles

`internal/agent/observe.go` adds `runtime_context` to the current user message: per-turn information such as the date, session ID, bound knowledge base summaries, pinned documents and question source; it is not persisted as a historical instruction. Knowledge base names and descriptions are length-limited and escaped; FAQ answers and full document text must be read through retrieval tools.

`@MCP` / `@Skill` produce a `must_use` hint for the current turn. An MCP mention only sets a priority hint for authorized services and does not remove other configured services; services not yet exposed are discovered through `discover_mcp_tools`. A skill mention hints to read the corresponding `SKILL.md` first. These selections cannot authorize unrelated operations and remain subject to the user's explicit source restrictions and backend permissions.

Messages the user adds while an answer is in progress and chooses to supplement immediately are appended to the end of the message list as user messages before the next iteration: the content sent to the model is wrapped in `<steer_message>` with a `<continue_task>` note (`types.SteerMessageContent`), while the session history and the UI keep the user's original text; the `steering` section defines how the model treats such messages.

Normal tool results keep the `tool` role and call ID; during message repair, results that cannot be paired are kept as escaped `untrusted_tool_result` data blocks and must not be promoted to system instructions. The agent, regular Q&A and the model fallback share the source data boundary rules, but the three do not use exactly the same prompt assembly flow.

When the iteration limit is reached or on error wrap-up, `internal/agent/finalize.go` reuses the current message list, keeping history, images and tool call pairing, then appends a wrap-up request. The final call provides no tools, sets `tool_choice=none` and disables thinking.

## Editing and saving

The frontend editor shows the effective body text, and `frontend/src/utils/agentPromptTemplates.ts` distinguishes template references from custom content when saving:

- Body text identical to a known template: save `system_prompt_id` / `context_template_id` and leave the body empty.
- Body text actually modified: save the body and clear the old template ID.
- Rewrite and fallback fields identical to the current default template: save empty values to inherit the defaults; intent prompts only save overrides that deviate from the defaults.

For example, an unmodified Wiki template is saved as:

```json
{"agent_mode":"smart-reasoning","system_prompt_id":"wiki_researcher","system_prompt":""}
```

`ResolveCustomAgentPrompts` in `internal/config/agent_prompts.go` resolves references at request time; explicit body text always takes precedence, and references are only looked up in the template set for the owning field and mode. The resolved result is not written back to the saved object; an unknown reference returns an empty body, and the caller uses the default path.

Default template updates do not overwrite existing custom body text, nor do they batch-migrate historical copies of defaults. When an old configuration needs to follow the template again, restore the default in the editor and save.

## What to check during maintenance

Decide where a new rule belongs first: role and domain methods go in `base`, source selection in `sources`, calling conventions in the tool definition or `tools`, output shape in `output`, and citation encoding in `protocol`. Avoid maintaining the same rule in multiple places.

The `[Agent][Prompt] section=... bytes=...` log helps locate the size of each section; verifying behavior also requires checking the final model messages, the actual tool registry and execution permissions, and using tasks to verify source selection, failure recovery and output format.
