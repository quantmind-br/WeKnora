# Sessions and Conversation Experience

The previous chapters covered "how knowledge gets in, how it gets retrieved" — this chapter covers **the chat box itself**: what the user sees and can do during a round of questions and answers. These capabilities are spread across the session, message, attachment, and suggested-question interfaces; this chapter brings them together.

## 1. What the user sees during a round of Q&A

| UI Element | Description |
| --- | --- |
| Pipeline progress bar | Shows the current stage before the answer is generated: attachment parsing, image understanding, document retrieval, web search, tool calls, thinking, generating answer |
| Thinking process | The model's reasoning content is displayed inline in the Agent timeline, collapsible |
| Citation markers | Source markers in the answer body; clicking jumps to the original text chunk |
| References drawer | A sidebar listing all retrieval sources for this round, including results returned by the Wiki tool |
| Follow-up suggestions | Next-step questions offered after the answer finishes, see the "Suggested Questions" section of [Agent Engine](07-agent.md) |

<Screenshot
  src="/screenshots/chat-references-drawer.png"
  caption="Chat page: answer, citation markers, and the references drawer on the right"
  hint="Shows a round of answer with citations, the expanded references drawer (including source titles and snippets), and the session action bar at the top." />

### Two waiting states of the progress bar

When all visible stages have finished but the model still hasn't started emitting text, there's a period of silence. The progress bar distinguishes two prompts based on this: rounds that actually ran retrieval show "Generating answer," while attachment-only Q&A with no retrieval step shows the neutral "Preparing." If no answer arrives after 60 seconds, it switches to a stalled state — when the SSE connection drops, the backend no longer sends a completion event, so without this cap the progress bar would keep claiming "almost done" forever. See the implementation in [Web Frontend](../05-clients/01-frontend.md).

### Whether citations are on or off has nothing to do with the references drawer

The Agent config's `citation_enabled` only controls **the markers in the answer body**. Turning it off keeps the body clean, but retrieval sources still get sent to the references drawer as usual — in other words, "not showing citations" does not mean "not providing sources." When this field is `nil`, it's treated as enabled, to preserve the behavior of Agents saved before this option was introduced.

### Exporting a conversation

The session action bar can export the entire conversation as Markdown (`buildSessionMarkdown()` in `frontend/src/utils/sessionMarkdown.ts`), including the session title, ID, export time, and each round of Q&A — handy for pasting into a ticket or a weekly report. Export happens entirely on the frontend and triggers no additional API calls.

## 2. Temporary attachments within a session

You can drop files directly into a conversation and ask about them, without first creating a knowledge base — these are called **temporary documents** (`temporary_documents` table, migration `000070`), and they belong only to the current session.

Endpoints (all under `/api/v1/sessions`):

| Method | Path | Description |
| --- | --- | --- |
| POST | `/:session_id/attachments` | Upload an attachment |
| GET | `/:id/attachments` | List attachments for this session |
| GET | `/:id/attachments/:attachment_id` | Attachment details |
| GET | `/:id/attachments/:attachment_id/preview` | Preview |
| DELETE | `/:id/attachments/:attachment_id` | Delete |

Key behaviors:

- State machine: `uploaded` → `processing` → `ready`; parsing is asynchronous. If an attachment hasn't finished parsing when the question is asked, the system waits up to `WEKNORA_CHAT_ATTACHMENT_WAIT_TIMEOUT_SEC` (default 60 seconds; recommended to increase for scanned documents);
- Parsed output is cleaned up after `WEKNORA_CHAT_ATTACHMENT_TTL_HOURS` (default 24 hours), so attachments don't occupy storage long-term;
- Scanned documents/image-based documents go through VLM OCR; concurrency and page limits are controlled by `WEKNORA_CHAT_ATTACHMENT_OCR_CONCURRENCY` (default 8) and `WEKNORA_CHAT_ATTACHMENT_OCR_MAX_PAGES` (default 8);
- There are three related Agent-side settings: `supported_file_types` (restricts allowed file types), `attachment_image_understanding` (whether to understand images), and `chat_parser_engine_rules` (which parsing engine attachments use) — see [Agent Engine](07-agent.md);
- Temporary attachments and knowledge base documents are two separate things: attachments don't enter the vector index, don't appear in the knowledge base list, and expire once the session ends. Material that needs long-term retrieval should be formally ingested into a knowledge base.

## 3. Visibility of channel sessions

Besides web chat, IM bots, web widget visitors, and API Key calls all generate sessions too. These "channel sessions" are **not visible by default** in the console, because they're isolated by Key, visitor, or IM identity respectively.

The rules live in `internal/application/service/session.go`:

- When the session list's `source` filter is empty or `web`, only the caller's own sessions are returned;
- Filtering by `api` / `im` / `embed` is a **space-level view**, requiring Admin+; otherwise it returns 403 (`listing channel sessions requires tenant admin or owner role`). Once the check passes, the per-user narrowing is dropped, so admins can then observe these otherwise mutually isolated sessions;
- The IM / Embed / API groupings in the sidebar are also admin-only, and they first probe the count — only showing up if there are sessions, to avoid leaving an always-empty entry point for regular users;
- Even for an admin, opening a channel session is **read-only observation**; sessions generated by an API Key are always scoped by ownership on write endpoints.

The intent of this design: admins need to be able to investigate "how did the bot answer yesterday," without letting regular members browse other people's support conversations.

## 4. Cross-session history search

Chat history can be indexed and searched across sessions:

| Method | Path | Permission |
| --- | --- | --- |
| POST | `/api/v1/messages/search` | Viewer+; API Key requires `message_history` capability or full access |
| GET | `/api/v1/messages/chat-history-stats` | Same as above |
| GET | `/api/v1/messages/:session_id/load` | Viewer+; API Key requires `chat` capability (can only read its own sessions) |

`message_history` is a standalone capability, intended to let data-analysis integrations search historical metadata without needing to hand them a full-access Key. The toggle and retention policy live under "Settings → Chat History" (`chathistory` section, requires Admin).

## 5. Related chapters

- Suggested questions (opening questions and follow-ups): [Agent Engine](07-agent.md)
- How images and files in answers get sent to the client: the "File Reference Formats" section of [API Overview](../04-api/01-api-overview.md)
- Session models for IM and web widgets: [IM Integration](12-im-integration.md), [Web Embed](13-embed-channel.md)
- Full session and message interfaces: [API Reference: Sessions and Chat](../04-api/02-api-chat.md)
