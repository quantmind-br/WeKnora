Aqui está a tradução completa do documento para inglês, com toda a estrutura markdown preservada:

---

# Feishu Drive Data Source Guide

The Feishu Drive data source (`feishu_drive` / `lark_drive`) can automatically sync documents and files from a folder in Feishu/Lark Drive into the WeKnora knowledge base, supporting incremental sync, scheduled sync, and recursive subfolder traversal.

---

## 1. Prerequisite: Create a Feishu App

1. Log in to the [Feishu Open Platform](https://open.feishu.cn/app) (Lark users should use the [Lark Open Platform](https://open.larksuite.com/app)) and create a **self-built enterprise app**.
2. Record the app's **App ID** (starting with `cli_`) and **App Secret** — you'll need these when configuring the data source.
3. Enable the permissions listed in the table in the next section, then **publish an app version** (permission changes only take effect after republishing).

> Note: Feishu (open.feishu.cn) and Lark (open.larksuite.com) are two independent systems, and apps are not shared between them. Use a Feishu app to sync Feishu Drive, and a Lark app to sync Lark Drive — credentials cannot be mixed.

---

## 2. Required Permissions (Detailed)

In the app's backend under "Permission Management," enable the following **3 permissions**:

| Permission ID | Name | Purpose | Symptom When Missing |
|---|---|---|---|
| `drive:drive:readonly` | View files in Drive | List folder contents (list API), download regular Drive files | 403 error when loading folders/syncing, with a message saying "the folder must first be shared with the group the app belongs to" |
| `drive:export:readonly` | Export cloud documents | Export docx/doc/sheet/bitable as docx/xlsx for parsing | Cloud document files fail to sync |
| `docx:document:readonly` | Read new-version document content | Parse docx document body and attachments via the blocks API (the primary path when export fails) | docx document parsing fails, and the export fallback also fails |

Notes:

- Compared to the "Feishu Knowledge Base" connector, Drive does **not** require `wiki:wiki:readonly`; the other permissions are the same.
- The list API also accepts `drive:drive` (read-write) or `space:document:retrieve` as alternatives, but it's recommended to enable only the read-only `drive:drive:readonly` for minimal privilege.
- After enabling permissions, you must **create and publish a new version**, otherwise the API will still report insufficient permissions.

---

## 3. Key Step: Share the Folder with the App

Feishu's permission model requires that even if the app has the above API permissions enabled, it can only access files that have been **explicitly shared with it**.

1. Create a Feishu (Lark) group, or reuse an existing Feishu (Lark) group.
2. Add the **administrators of the Feishu Drive** you want to import to the group.
> PS: Not adding them to the group will also result in no access permission.
3. Open the target folder in Feishu Drive.
4. Click "Share" / "··· → Add Collaborator," and share the folder with the **group the app belongs to** (add the app to a group, then share the folder with that group), granting "Can view" access.
5. Subfolders and the files within them are authorized along with the parent folder, so there's no need to share them individually.
6. If newly added files later fail to sync with a 403 error, check whether the file is within this already-shared directory tree.

This is a one-time operation, but skipping it will cause folder loading to fail immediately with "The app does not have access to this folder."

---

## 4. Configuring the Data Source (Four Steps)

Location: Knowledge Base → Settings → Data Sources → New Data Source, then select "Feishu Drive."

### Step 1: Select Type

Select "Feishu Drive" (for the international version, select "Lark Drive").

### Step 2: Configure Credentials

Enter the App ID and App Secret. When you click Next, the system automatically tests the connection (verifying whether a tenant_access_token can be obtained). This step only verifies the app's identity, not its folder permissions.

### Step 3: Select Scope

1. In the "Drive Folder Token" input field, enter the target folder's `folder_token`, **or simply paste the folder's full link** (Feishu `https://xxx.feishu.cn/drive/folder/<token>` or Lark `https://xxx.larksuite.com/drive/folder/<token>` — the system automatically extracts the token from the path; both link formats are supported).
2. Click "Load" to list the full directory tree under that folder.
3. Check the files/folders to sync — supports expanding levels one by one, and select all/collapse branch.

Notes:

- **The Drive root directory is not supported** (the root directory is not paginated and does not return shortcuts) — you must select a specific folder.
- If loading fails, follow the on-screen guidance: 403 → go back to Section 3 to share the folder; invalid token → re-copy it from the folder URL.

### Step 4: Sync Strategy

| Setting | Description | Default |
|---|---|---|
| Sync schedule | Cron expression, defaults to every 6 hours; leave blank to trigger manually only | `0 0 */6 * * *` |
| Sync mode | Incremental (based on modification-time cursor) / Full | Incremental |
| Conflict strategy | Overwrite on content change / Skip | Overwrite |
| Sync deletions | Documents deleted at the source are **only counted, not automatically deleted** from the knowledge base — must be deleted manually in the knowledge base | Enabled |

After saving, the data source starts running according to the configured strategy; you can also manually "Trigger Sync" from the data source card.

---

## 5. Supported File Types

| Drive Type | Handling |
|---|---|
| `docx` / `doc` (new and legacy documents) | Parsed via the blocks API for body content and attachments; falls back to exporting as docx for parsing if this fails |
| `sheet` / `bitable` (spreadsheets/multi-dimensional tables) | Exported as xlsx and parsed |
| `file` (regular uploaded files, e.g. PDF/PPT/images) | Downloaded directly and parsed according to file type |
| `shortcut` | Automatically resolved to the target file for syncing (shortcuts cannot point to folders) |
| `folder` | Traversed recursively |
| `mindnote` / `slides` / `board` | Not supported, skipped |

Additional behavior:

- **Attachments** within a docx are synced as independent knowledge entries (associated with the parent document; when the parent document is updated, removed attachments are automatically cleaned up).
- Images embedded in documents are processed via OCR/multimodal parsing when possible; if object storage or a VLM is not configured, this step is automatically skipped without affecting the sync of the document body.

---

## 6. docx Parsing Modes and Environment Variables

Feishu's new-version cloud documents (docx) support two parsing paths, controlled by the `FEISHU_DOCX_PARSE_MODE` environment variable. This variable applies to the WeKnora **app service** (not the data source configuration) and affects both the Feishu Drive and Feishu Knowledge Base connectors simultaneously.

### Mode Comparison

| | export (default) | blocks |
|---|---|---|
| Environment variable value | Blank / `export` | `blocks` |
| Parsing path | Async export API -> .docx binary -> docreader parsing | blocks API -> Markdown |
| Image-document association | ✅ Images are inlined into the parent document, linked via `parent_chunk_id` | ❌ Images become independent knowledge entries, disconnected from the document |
| Can retrieval/Wiki/agent associate images? | Yes | No |
| Sync speed | Slow (async export + docx parsing) | Fast |
| Attachments within docx (file block) | Lost (.docx export doesn't include them) | Preserved, as independent entries |
| Required permissions | `drive:drive:readonly` + `drive:export:readonly` | `drive:drive:readonly` + `drive:export:readonly` + `docx:document:readonly` |

### Why Image Association Differs

- **export mode**: After exporting the .docx, docreader parses it and inlines images into the parent document (consistent with regular docx uploads), establishing a parent-child association within the same knowledge entry via `parent_chunk_id`. All three scenarios (retrieval, Wiki, agent Q&A) can return the image content together with the document in a single retrieval.
- **blocks mode**: Goes through the blocks API, where image blocks are rendered as empty `![image]()` placeholders; images are downloaded separately as independent knowledge entries, with only a weak metadata-level association to the parent document. WeKnora's retrieval, Wiki construction, and agent Q&A pipelines do not associate the image content back with the document — the image and the document body remain disconnected.

### Configuration

Set the following in the WeKnora service's `.env` or in the app service's environment variables in `docker-compose.yml`:

```env
FEISHU_DOCX_PARSE_MODE=blocks
```

If unset or set to `export`, the default mode is used. The app service must be restarted after changing this for it to take effect.

### Trade-offs of export Mode

- **Slower sync**: Every docx must go through async export (creating a task + polling + downloading) plus docreader parsing, which is slower than the blocks API.
- **Attachment loss**: File-block attachments within a docx are not included when the .docx is exported and downloaded; if attachments are needed, use blocks mode or sync them separately.
- **Image content depends on multimodal processing**: After images are inlined, OCR/captioning is generated asynchronously by the multimodal service; if object storage or a VLM is not configured, images are only stored without generated content (they display normally in the frontend, but remain weak at the retrieval level).

### Recommendation

- If you need image content to be associated with the document in retrieval/Wiki/agent scenarios: use the default **export** mode.
- If you only need the document body, want to preserve attachments, and prioritize sync speed: use **blocks** mode.

---

## 7. Sync Behavior

- **Incremental sync**: Uses the file modification time as a cursor, pulling only content that changed since the last sync; resumes from the breakpoint after an interruption.
- **Partial failures don't halt the sync**: If a subfolder lacks permission or a file fails to download, that entry is marked as failed while the rest of the content continues syncing; failure details can be viewed in the "Sync Log."
- **Update semantics**: Files whose content has changed have their old knowledge entry deleted and rebuilt; the document is briefly unavailable during parsing, which is normal.
- **Safety constraint**: To prevent accidental deletion, files removed at the source are not automatically removed from the knowledge base (see "Sync deletions" in Step 4).

---

## 8. FAQ

| Symptom | Cause and Resolution |
|---|---|
| "Please enter a specific folder's folder_token; the Drive root directory is not supported" | The input is empty or a root directory link was pasted — use a specific folder link instead |
| "The app does not have access to this folder. Please share it with the group the app belongs to…" | Section 3's sharing step was not completed, or the folder was shared with the wrong target (not the app's group) |
| "Invalid app credentials or missing Drive permissions" | Incorrect App ID/Secret, or the permissions in Section 2 were not enabled/published |
| "folder_token does not exist or has been deleted" | The token was copied incorrectly — get it again from the folder's "Share → Copy Link" option |
| Some entries in the sync log show as failed | Open the log to see the failure stage: `list_children` usually means an unauthorized subfolder; `fetch` usually means a permission issue or unsupported type for a single file |
| Source display in the knowledge list | Documents synced via Drive are labeled with the source "Feishu Drive," distinct from "Feishu" for Knowledge Base sync |
