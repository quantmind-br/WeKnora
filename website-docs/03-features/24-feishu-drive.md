# Feishu Drive Integration

`feishu_drive` / `lark_drive` sync the documents and files in a specified folder into a knowledge base. Wiki spaces use the `feishu` / `lark` connectors; for the general sync flow, see [Data Source Sync](10-datasource.md).

## App identity and folder permissions

Use the App ID and App Secret of a custom enterprise app in the matching region; Feishu and Lark credentials cannot be mixed. Grant read scopes according to the APIs actually used: folder listing and file download, cloud document export, and docx content reading in blocks mode. For scope names, alternative scopes and application requirements, refer to the Feishu Open Platform documentation for the [folder list API](https://open.feishu.cn/document/server-docs/docs/drive-v1/folder/list), the [export API](https://open.feishu.cn/document/server-docs/docs/drive-v1/export_task/create) and related APIs.

After configuring scopes, publish an app version and grant the target folder to a scope the app can access. If sharing through a group the app belongs to, verify the authorization relationship between the app, the folder owner and the group. API scopes and file access are two separate checks: being able to obtain a tenant access token does not mean the app can read any folder.

The connection test in WeKnora validates the app credentials; folder access is only validated when the resource tree is loaded afterwards. If the connection succeeds but the folder returns 403, first check the sharing scope and whether the permission changes were published.

## Configuring in a knowledge base

1. Knowledge base settings → Data sources → New, then choose "Feishu Drive" or "Lark Drive".
2. Enter the App ID and App Secret and run the connection test.
3. Enter a specific folder's `folder_token` or the full `/drive/folder/<token>` link, load the resource tree and choose the sync scope. It cannot be left empty to use the cloud space root.
4. Choose full/incremental, sync schedule, conflict strategy and sync deletion; after saving, first sync a few files manually and check the logs and parsing results.

Incremental sync compares the old cursor with the current file tree: files that disappeared are marked as deleted only when the selected resources were listed completely; if listing a subfolder fails, detection is deferred. Only with sync deletion enabled does this delete the **knowledge entries belonging to the current data source**; with it disabled they are kept. A first sync, or a full sync without an old cursor, cannot detect historical deletions this way, and a failed deletion is not guaranteed to be caught up by a single full sync; check the sync logs and the remaining entries.

An update to the document itself also cleans up attachment child items that disappeared in blocks mode. This is part of the content update and is not controlled by the source-document sync deletion switch above.

## Files and parsing modes

Regular files are downloaded and enter the parsing pipeline by file type; sheets/bitables are exported as xlsx, folders are traversed recursively, and shortcuts are resolved to their targets. Unsupported cloud document types are skipped; check the skip reason in the sync logs.

New-style docx has two paths, controlled by `FEISHU_DOCX_PARSE_MODE` on the **app service**, which affects both the Wiki and Drive connectors:

| Mode | Path | Use and limitations |
| --- | --- | --- |
| Empty or `export` (default) | Export docx asynchronously, then run document parsing | Images are parsed with the parent document; export and parsing take longer, and embedded file block attachments are not kept by the export |
| `blocks` | blocks API converted to Markdown; falls back to export when the API fails or the body is empty | Keeps parsable attachments as separate entries; with multimodal sync enabled, images become separate entries, and you cannot assume retrieval will link them back to the body |

To use blocks mode, set it in `.env` and rebuild the app:

```dotenv
FEISHU_DOCX_PARSE_MODE=blocks
```

```bash
docker compose up -d app
```

Changing the mode affects subsequent fetches; it does not automatically convert already ingested content to the other format. Whether image content is retrievable also depends on object storage, OCR/multimodal processing and indexing status; see [Document Parsing](03-document-parsing.md).

## FAQ

| Symptom | Resolution |
| --- | --- |
| Invalid token or credential test fails | Verify the Feishu/Lark region, App ID/Secret and private proxy address |
| Loading folders fails | Use a specific folder link; check API scopes and folder sharing authorization |
| Cloud documents fail while regular files work | Check export permission; blocks mode additionally needs docx read permission |
| Fewer synced items than files in the folder | Check unsupported types, subfolder authorization, and failed/skipped counts |
| Images cannot be retrieved with the body | Confirm the parsing mode and multimodal status; standalone images in blocks mode and inline images in export mode behave differently |
| Briefly not retrievable after a document change | An update may delete the old knowledge and re-ingest it; wait for the new version to finish parsing and indexing |

Implementation reference: `internal/datasource/connector/feishu/drive/`, `core/shared.go` and `internal/application/service/datasource_service.go`.
