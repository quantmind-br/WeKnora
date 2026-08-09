# Wiki Capability

After uploading a pile of scattered documents, you often still don't know "what's actually in this batch of material." Wiki exists for exactly this: once documents are ingested, WeKnora uses an LLM to extract entries — people, products, concepts — from the source text, generates a sourced Markdown page for each entry, and links the pages together into a knowledge site you can browse like Wikipedia.

The difference from ordinary Q&A is this: Q&A is "you ask, I answer," while Wiki is "the knowledge gets organized for you up front." The more material there is, and the more scattered it is, the more value Wiki delivers. And these pages aren't just for humans to read — the Agent can read and write them too, using Wiki as long-term memory; anywhere the model got something wrong, you can edit it directly, and every edit keeps a version and can be rolled back (see "Manual Editing and Version History").

<Screenshot
  src="/screenshots/wiki-browser.png"
  caption="Wiki browser: folder tree on the left, generated entry pages and sources on the right"
  hint="Shows the folder tree grouped by type on the left, an entity page's body, in-page wiki links, and source document citations." />

## How to Enable It

1. Edit the knowledge base → open **Wiki** under "Indexing Strategy";
2. Upload documents (existing documents are included too — no need to re-upload);
3. Wait for generation. Wiki generation is asynchronous and can take a while when there are many documents; the knowledge base breadcrumb shows an "Indexing" indicator;
4. Once done, go to the knowledge base's **Wiki** tab to browse; the **Graph** tab shows the link relationships between entries.

Generation calls the LLM, so cost scales with document volume. Extraction density can be adjusted in the knowledge base's Wiki configuration (`focused` / `standard` / `exhaustive`, see "Extraction Granularity" below).

<Screenshot
  src="/screenshots/wiki-graph.png"
  caption="Wiki graph view: link relationships between entries"
  hint="Shows the graph overview mode, with nodes colored by page type, clickable to jump to a specific page." />

## Page Model and Hierarchy

### Page Types (PageType)

`internal/types/wiki_page.go` defines 6 page types:

| Type | Description |
| --- | --- |
| `summary` | Summary page for a single source document (slug shaped like `summary/<knowledge-uuid>`) |
| `entity` | Entity page (person, organization, product, technology, etc.) |
| `concept` | Concept/topic page |
| `index` | Wiki-level index page (metadata) |
| `synthesis` | Synthesis/analysis page, **created only by the Agent via the `wiki_write_page` tool** |
| `comparison` | Comparison page, **created only by the Agent via the `wiki_write_page` tool** |

Page status (`WikiPageStatus`): `draft` / `published` (default) / `archived`.

### Folder Hierarchy

Migration `000061_wiki_page_hierarchy.up.sql` introduces a standalone `wiki_folders` table (adjacency-list model):

- `WikiFolder` organizes the tree via `ParentID` (empty string = root) plus a materialized `Path` (a `/`-joined chain of names); empty folders can exist independently, letting users lay out the skeleton first;
- `WikiPage.FolderID` is the **single source of truth** for which page belongs where (FK → `wiki_folders.id`, empty string means the wiki root);
- `CategoryPath` / `WikiPath` / `Depth` / `SortOrder` on the page are **cached projections** derived from the folder chain;
- Folders go at most 3 levels deep (constant `WikiCategoryMaxDepth = 3`); `CleanWikiCategoryPath()` normalizes full-width separators (`／`, `｜` → `/`) and strips type labels.

### Key Fields

- `Slug`: the page's unique identifier within a KB (see next section);
- `SourceRefs`: source citations, formatted as `"<knowledge_id>|<doc_title>"`; `ChunkRefs`: chunk-level evidence citations;
- `InLinks` / `OutLinks`: wiki-link backward/forward links maintaining the graph structure; `GET /graph` supports both a global and an ego view;
- `Aliases`: alternate names (used for search and for pointing old names to their post-merge target); `Version`: version number.

## Generation Pipeline

Wiki generation is **triggered by document ingestion (knowledge ingest)** and runs asynchronously through a Redis task queue. Task types are defined in `internal/types/task.go`:

```go
TypeWikiIngest   = "wiki:ingest"
TypeWikiFinalize = "wiki:finalize"
```

The pipeline has four stages overall (Map-Reduce structure):

| Stage | Task | What it does | LLM prompt (`internal/agent/prompts_wiki.go`) |
| --- | --- | --- | --- |
| Pass 0: Candidate extraction | `wiki:ingest` | Extracts a candidate slug skeleton from the document (JSON of entities + concepts) | `WikiCandidateSlugPrompt` |
| Pass 1..N: Chunk citation | `wiki:ingest` | Tags citations to candidate slugs chunk by chunk, outputting `{ citations: {"slug": ["c001", ...]}, new_slugs: [...] }`; reuses prefix caching for the long shared prefix | `WikiChunkCitationPrompt` |
| Reduce: Page merging | `wiki:ingest` | Incrementally updates or merges pages by slug, outputting `SUMMARY: ...` plus Markdown body; strictly grounded, no hallucination, deduplicated, no self-links | `WikiPageModifySystemPrompt` + `WikiPageModifyUserPrompt` |
| Finalize: Wrap-up | `wiki:finalize` | Rebuilds the index page, cleans up dead links, fills in cross-links, prunes folders — pure SQL/graph algorithms, **no LLM calls** | — |

Supporting prompts:

- `WikiTaxonomyPlanPrompt`: plans folder paths uniformly for all entities/concepts in the same batch (at most 2 levels, prefers reusing existing folders), keeping the folder tree coherent;
- `WikiDeduplicationPrompt`: determines whether a newly extracted item refers to the same thing as an existing page, following the core principle **"related ≠ same"**, returning `{ merges: { "entity/new": "entity/existing" } }`.

### Extraction Granularity

`WikiExtractionGranularity` in `WikiConfig` (stored in the `knowledge_bases.wiki_config` JSONB column) controls extraction density:

| Granularity | Behavior |
| --- | --- |
| `focused` | Only 3–7 major topics |
| `standard` (default) | Topics plus entities/concepts that are substantively discussed (a paragraph, multiple mentions, or 2–3+ sentences) |
| `exhaustive` | Enumerates every named thing and every recognized concept |

### Concurrency and Batching

`WikiConfig` parameters (`internal/types/wiki_page.go`):

| Parameter | Default | Description |
| --- | --- | --- |
| `IngestBatchSize` | 5 | Number of pending documents claimed per batch |
| `IngestMapParallel` | 10 | errgroup concurrency for the Map stage (extraction + citation per document) |
| `IngestReduceParallel` | 10 | Concurrency for the Reduce stage (writing a page per slug) |
| `IngestMaxInflight` | 4 | Max concurrent batches per KB (ensures fairness across KBs) |

### Generation Flow Diagram

```mermaid
flowchart TD
    A["Document ingest (knowledge ingest)"] --> B["Task enqueued: wiki:ingest (Redis queue)"]
    B --> C["Pass 0: Candidate slug extraction<br/>WikiCandidateSlugPrompt"]
    C --> D["Taxonomy planning<br/>WikiTaxonomyPlanPrompt unifies folder paths"]
    C --> E["Pass 1..N: Chunk citation tagging<br/>WikiChunkCitationPrompt + prefix caching"]
    E --> F["Deduplication check<br/>WikiDeduplicationPrompt (related ≠ same)"]
    F --> G["Reduce: concurrent page writes by slug<br/>WikiPageModifySystemPrompt<br/>incremental merge / create, citation grounding enforced"]
    D --> G
    G --> H["Write to wiki_pages<br/>changes projected to the KB activity feed (audit)"]
    H --> I["Task enqueued: wiki:finalize<br/>(TaskID = wiki-finalize-KBID, deduped per KB)"]
    I --> J["Finalize: rebuild index / clean dead links / cross-link<br/>pure SQL and graph algorithms, no LLM"]
    J --> K["Published pages browsable in WikiBrowser<br/>readable/writable by Agent tools"]
```

## Slug Mechanism

- **Format**: `<type>/<name>`, e.g. `entity/acme-corp`, `concept/rag`, `summary/<knowledge-uuid>`; lowercase, hyphen-separated, non-Latin names get romanized/pinyinized;
- **Uniqueness**: enforced by a database unique index (`000037_wiki_and_indexing.up.sql`):

  ```sql
  CREATE UNIQUE INDEX idx_kb_slug ON wiki_pages (knowledge_base_id, slug) WHERE deleted_at IS NULL
  ```

  meaning the slug is unique **within a single knowledge base**, but can repeat across KBs;
- **Stability**: when a document is updated and re-extracted, the prompt forces the model to reuse the old slug —

  > If an entity or concept from the previous extraction still exists in the current document, **reuse its exact slug** from the previous list. Do NOT generate a new slug for the same thing.

  New slugs are only generated for genuinely new things; items that disappeared are simply no longer emitted;
- **Slug Handles (handle proxying)**: during ingest LLM calls, high-entropy real slugs (especially UUID-bearing `summary/...` ones) are replaced with short handles (`ref-1`, `ref-2`); the model outputs `[[ref-1|title]]`, which the backend then resolves back to the real slug, avoiding transcription errors on UUIDs by the model (`internal/application/service/wiki_slug_handles.go`);
- **Reference usage**: wiki citations in Agent responses appear as `[[slug|title]]`; `InLinks`/`OutLinks` maintain the page graph by slug; renaming a slug (via the `wiki_rename_page` tool) automatically updates all backlinks.

## Publishing and Access

All Wiki routes are mounted under `/api/v1/knowledgebase/:kb_id/wiki` (`internal/router/router.go`); **there is no public, unauthenticated access mode** — both reads and writes are gated by RBAC and KB access control:

### Read Endpoints (Viewer + KBAccessRead)

| Method | Path | Description |
| --- | --- | --- |
| GET | `/pages` | List pages |
| GET | `/pages/*slug` | Fetch a single page by slug |
| GET | `/folders` | Folder tree |
| GET | `/index` | Index page |
| GET | `/graph` | Link graph (global overview / ego mode) |
| GET | `/stats` | Statistics |
| GET | `/search?q=...` | Search |
| GET | `/lint` / `/issues` | Quality check results / issue list |
| GET | `/revisions/*slug` | List version history; `?version=N` fetches that version's full text |

`KBAccessRead` covers: KB owners, organization sharing, and access granted through a shared Agent.

### Write Endpoints (OwnedWikiKBOrAdmin + KBAccessWrite)

| Method | Path | Description |
| --- | --- | --- |
| POST / PUT / DELETE | `/pages`, `/pages/*slug` | Create / update / delete a page |
| POST / PUT / DELETE | `/folders`, `/folders/:folder_id` | Folder management |
| PUT | `/move-page` | Move a page to a folder |
| POST | `/rebuild-links` | Rebuild the link graph |
| POST | `/auto-fix` | Trigger auto-fix |
| PUT | `/issues/:issue_id/status` | Update issue status |
| POST | `/revert` | Revert to a specific version (body: `{slug, version}`) |

Write access is determined by KB ownership: contributors can manage a wiki as long as they own that KB; otherwise, 403. API key scenarios map to `ingest` / `retrieve` capabilities.

The frontend provides the browsing UI via `WikiBrowser.vue`; while a document is being parsed, `wikiStatusRefresh.ts` polls `parse_status` (`pending` / `processing` / `finalizing`), continuing to poll if parsing has finished but the summary is still being generated. The folder tree's expansion state is maintained separately by `wikiDirectoryState.ts`: after a new folder is created or data is refreshed, `expandWikiDirectoryPath()` marks each level along the current path as "user-expanded," preventing the whole tree from collapsing back to its default folded state on refresh.

## Manual Editing and Version History

Wiki pages are LLM-generated, so there will inevitably be spots that need manual correction. Pages therefore support direct editing, with full version history retained (migration `000075`).

### Who Made Each Version

`wiki_pages` carries two provenance fields; `last_edit_source` marks the author type of the **current version**:

| `edit_source` | Meaning |
| --- | --- |
| `pipeline` | Written by the wiki generation pipeline (legacy rows with an empty string are treated as `pipeline`) |
| `agent` | Written by the Agent via tools like `wiki_write_page` / `wiki_replace_text` |
| `user` | Edited manually in the editor |
| `revert` | A version produced by a revert |

The paired `last_editor_id` records who performed the action (empty when written by the background pipeline). The UI uses this to distinguish "this was written by the model" from "this was edited by a human."

### Snapshots and Rollback

- Before a page is overwritten, the **old version** is first snapshotted in full into `wiki_page_revisions` (title, body, summary, page type, status, aliases, plus that version's author and timestamp); only the current version lives in `wiki_pages`. A unique index on `(page_id, version)` combined with `ON CONFLICT DO NOTHING` makes the "snapshot then update" write path idempotent under retries;
- `GET /revisions/*slug` lists historical versions (newest first, body excluded, with the current version number attached); `?version=N` fetches that version's full text, for diffing;
- `POST /revert` with `{slug, version}` reverts. Reverting does **not** roll the version number backward — instead, it produces a new version containing the target version's content, with `edit_source` set to `revert`, so a revert can itself be reverted. Reverting to the current version returns 400 (usually meaning the frontend is working from a stale history list).

### History Retention Policy

Unbounded snapshot retention would get flooded by the pipeline, so a two-tier cap applies (`internal/types/wiki_page.go`):

- **Soft cap of 50 versions**: only prunes "prunable" snapshots — those written by `pipeline` and legacy empty-source rows;
- **Hard cap of 200 versions**: prunes regardless of author, ensuring even a page maintained purely by hand has bounded storage.

The intent behind this design: when a hot page gets rewritten repeatedly by the pipeline, that shouldn't push the manual edits the user actually cares about out of the history.

<Screenshot
  src="/screenshots/wiki-revision-history.png"
  caption="Wiki page version history: version list broken down by source, with a revert entry point"
  hint="Shows the history drawer for a wiki page, including version number, edit source (pipeline/manual/Agent/revert), editor, and timestamp, plus compare/revert buttons." />

## Relationship with the Agent

Wiki isn't just for humans to read — it's a first-class workspace for the Agent. `internal/agent/tools/definitions.go` registers 10 wiki tools:

| Tool | Purpose | Key parameters |
| --- | --- | --- |
| `wiki_read_page` | Batch-read full page content by slug | `slugs: string[]` |
| `wiki_search` | Regex search over pages | `queries`, `limit?`, `knowledge_base_id?` |
| `wiki_write_page` | Create/overwrite a whole page (`synthesis` and `comparison` pages can only be created this way) | `slug`, `title`, `summary`, `content`, `page_type`, `aliases?`, `source_refs?` |
| `wiki_replace_text` | Exact in-page text replacement | `slug`, `old_text`, `new_text` |
| `wiki_rename_page` | Rename a slug, auto-updating backlinks | `slug`, `new_slug` |
| `wiki_delete_page` | Delete a page and clean up dead links | `slug` |
| `wiki_read_source_doc` | Read back the original source document (with context) | document ID |
| `wiki_flag_issue` | Flag a page issue | `slug`, `issue_type ∈ {mixed_entities, contradictory_facts, out_of_date, other}`, `description` |
| `wiki_read_issue` | View issue details | issue ID |
| `wiki_update_issue` | Update issue status | issue ID, `status ∈ {pending, ignored, resolved}` |

Tool output is XML-like (`<wiki_page><metadata>...<summary>...<content>...`); the frontend's `parseWikiToolReferences()` in `frontend/src/utils/wikiToolReferences.ts` parses it into reference cards rendered inline in the conversation.

Supporting mechanisms:

- **Wiki Scope**: within an Agent session, a whitelist of wiki KBs is maintained; an `@mention` can narrow scope to specific documents/tags, and tool execution automatically filters `source_refs` accordingly (`internal/agent/tools/wiki_tools.go`);
- **Wiki Fixer**: a built-in Agent (`types.BuiltinWikiFixerID`) responsible for auto-fixing wiki issues (dead links, entity confusion, etc.). Cross-tenant access to a shared KB requires a tenant role of Editor or above, and is automatically elevated into the source tenant's context (`internal/handler/session/wiki_fixer_scope.go`);
- **Issue loop**: the `wiki_page_issues` table, the lint endpoint, and `auto-fix` together let both humans and the Agent report and resolve issues.

## Operation History (Knowledge Base Activity Feed)

Wiki used to maintain its own operation log (the `wiki_log_entries` table + `GET /wiki/log` endpoint + a log tab in WikiBrowser). This feed overlapped with the knowledge base activity feed and has been removed entirely in migration `000077_remove_wiki_log`: the table was dropped, legacy `page_type = 'log'` pages were deleted along with it, and `log` is no longer a valid page type.

Now the **knowledge base activity feed is the single entry point for operation history**:

- at the end of an ingest batch, `wiki_ingest_batch.go` tallies the counts of each action type in that batch and calls `service.RecordWikiContentActivity()` to record a `wiki_content_changed` activity;
- when a page is manually created/updated/deleted in WikiBrowser, `internal/handler/wiki_page.go` likewise projects `manual_create` / `manual_update` / `manual_delete` into the activity feed;
- activity records land in the audit log system (`kb_activity.go` → `AuditLogService`), viewable under "Knowledge Base → Settings → Activity," with the same retention policy as other audit logs (see [Observability and Auditing](16-observability.md));
- writes are best-effort: a failed activity record does not fail the wiki edit itself.

Upgrade note: if any external integration still calls `GET /api/v1/knowledgebase/:kb_id/wiki/log`, switch it to the knowledge base activity feed endpoint `GET /api/v1/knowledge-bases/:id/activity`.

## Failure Recovery

`internal/container/recover_pending_wiki_tasks.go` closes the gap left at service startup by Lite mode (the in-process `SyncTaskExecutor`) or by a Redis enqueue interruption:

1. scans the persisted `task_pending_ops` table for pending combinations where `scope = knowledge_base` and `task_type ∈ {wiki:ingest, wiki:finalize}`;
2. cleans up leftover rows for deleted KBs (fail-closed);
3. re-enqueues trigger tasks for each active KB: `wiki:ingest` without a TaskID (allowing multiple concurrent batches), and `wiki:finalize` using `"wiki-finalize-" + KB_ID` for deduplication (only one finalize retained per KB). Redundant enqueuing is harmless — ingest claims disjoint rows, and finalize merges within its lane.

## Implementation Reference

For locating source when reading the code (paths relative to the repo root):

| Layer | File |
| --- | --- |
| Data structures | `internal/types/wiki_page.go` |
| HTTP Handler | `internal/handler/wiki_page.go` |
| Generation pipeline | `internal/application/service/wiki_ingest.go`, `wiki_ingest_batch.go`, `wiki_ingest_cite.go`, `wiki_ingest_dedup.go`, `wiki_ingest_taxonomy.go` |
| Page service | `internal/application/service/wiki_page.go`, `wiki_linkify.go`, `wiki_lint.go`, `wiki_slug_alias.go`, `wiki_slug_handles.go` |
| LLM prompts | `internal/agent/prompts_wiki.go` |
| Agent tools | `internal/agent/tools/wiki_*.go` (registered in `internal/agent/tools/definitions.go`) |
| Failure recovery | `internal/container/recover_pending_wiki_tasks.go` |
| Routing | `internal/router/router.go` (behavioral tests in `internal/router/router_wiki_test.go`) |
| Database migrations | `migrations/versioned/000037_wiki_and_indexing.up.sql`, `000061_wiki_page_hierarchy.up.sql`, `000077_remove_wiki_log.up.sql` |
| Frontend | `frontend/src/views/knowledge/wiki/WikiBrowser.vue`, `frontend/src/api/wiki/`, `frontend/src/utils/wikiToolReferences.ts` |
