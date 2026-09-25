# FAQ and Upgrade Troubleshooting

First check the application version, database migration status and dependency connection status in **Settings → System Information**, then look at the logs of the relevant service. Run the Compose commands below from the repository root. For local development, dependency services use `docker compose -f docker-compose.dev.yml`; read app logs from the host terminal or its log file, since the development Compose has no `app` service.

```bash
docker compose ps
docker compose logs --tail=200 app
docker compose logs --tail=200 docreader
docker compose logs --tail=200 postgres redis
```

After editing `.env`, recreate the affected container with `docker compose up -d <service name>`; `docker compose restart` does not reload the container's environment variables.

## Diagnose by symptom

| Symptom | Checks, in order |
| --- | --- |
| Upload fails, or parsing stays in progress | Failure reason and parsing trace in the document details → parsing engine connectivity → background task queue; see [Document Parsing](../03-features/03-document-parsing.md) and [Async Tasks](../02-architecture/05-async-tasks.md) |
| Images display on this machine but not on other devices | Check whether the returned address contains `localhost`, a container service name, or an unreachable object storage address; third-party clients that receive `resource://` must resolve it as described in [File Access](../03-features/21-file-access.md), not use it directly as an image URL |
| A saved setting reverts to its old value | Check whether a proxy or browser cache rewrites the response, whether you are connected to another environment, or whether you switched spaces; for built-in models also check the YAML startup sync; see [Model Management](../03-features/06-models.md) |
| "Insufficient permissions" after an upgrade | Verify the current space role, resource ownership and the API key's capability scope; see [Authentication and Authorization](../03-features/01-tenant-auth.md). Platform admin and space Owner are not the same concept |
| An agent reports that its model is not ready | Confirm the models the agent references still exist and are fully configured, and run the model connectivity test; see [Model Management](../03-features/06-models.md) |
| Connections to a private-network data source, model or vector database are refused | Check the SSRF validation and port policy; allow only the targets you need, following the [Configuration Reference](04-configuration.md) |
| The error says outbound traffic is whitelist-only | The deployment has `SSRF_DNS_WHITELIST_ONLY` enabled, so hosts not on the whitelist are rejected before DNS resolution. Add the target host to `SSRF_WHITELIST` or `SSRF_WHITELIST_EXTRA`, then recreate the app / docreader containers; see [Configuration Explained](04-configuration.md) |
| The login API returns an HTML 404 | Check whether `SEARXNG_PORT` equals `APP_PORT` (default 8080). On Linux, once SearXNG binds `127.0.0.1:8080`, requests to `localhost:8080` reach SearXNG first. Keep SearXNG on 8888 or another free port |
| Pulling the MinIO image fails with `pull access denied` | MinIO no longer publishes images to Docker Hub. The current Compose and Helm files use `quay.io/minio/minio`; custom orchestration files must update the image address too |
| Embedded pages or signed links stop working after an upgrade (the embed page reports `embed session signing key is not configured` and the startup log shows `[startup-env] no usable signing key`) | The signing key comes from `SYSTEM_SIGNING_KEY` and falls back to `SYSTEM_AES_KEY` when unset; the example keys and keys shorter than 16 characters cannot sign. Generate one with `openssl rand -hex 32` and set it as `SYSTEM_SIGNING_KEY`. **Do not change `SYSTEM_AES_KEY` for this**, or saved credentials can no longer be decrypted. v0.8.2 changed the signing algorithm for embed sessions, so every embed session issued before the upgrade is invalidated and visitors must re-enter; multi-replica deployments must use the same key |
| The DingTalk bot stops replying after an upgrade | Since v0.8.2 DingTalk only supports Stream mode, and the upgrade migration switches `webhook` channels to `websocket`. Enable Stream mode for the app in the DingTalk developer console; see [IM Integration](../03-features/12-im-integration.md) |
| The background queue keeps backing up | Diagnose using the oldest task's wait time, the active workers and downstream quotas; adding workers does not increase the model provider's quota; see [Capacity Planning](../02-architecture/05-async-tasks.md#capacity-planning) |
| A skill is in the catalog but cannot run | Adding to the catalog and installing into a sandbox are two separate steps; check the installation records, the agent's sandbox selection and its skill scope; see [Skill Catalog and Sandbox](../03-features/22-skills-sandbox.md) |
| The Local sandbox is missing after an upgrade | The `local` backend was removed; reconfigure Docker, CubeSandbox or E2B; see [Sandbox Deployment and Troubleshooting](../06-development/04-sandbox-deployment.md) |
| The local browser is paired but no web tasks run | The local browser must be enabled in the Smart Reasoning input box; paused tasks must be resumed explicitly; see [Local Browser](../05-clients/09-local-browser.md) |
| The embed page fails to load or returns 403 | Put the actual host Origin on the allowlist; check the CSP, the reverse proxy and the Origin of the secure-mode exchange; see [Embed Channel](../03-features/13-embed-channel.md) |
| The Feishu app test succeeds but no folders load | The connection test only verifies the app identity; folders also need authorization; see [Feishu Drive Integration](../03-features/24-feishu-drive.md) |

## Database migration failures {#database-migrations}

By default the application runs migrations at startup. It may keep starting after a failure, so "the page opens" does not mean the schema was upgraded successfully. The migration version, dirty flag and error in System Information, together with the app startup log, are where to start.

**Do not assume a failed migration was fully rolled back.** The actual state depends on the SQL transaction boundaries and where the failure happened. Before recovering, keep the logs, back up the database, and compare the migration file for the failed version against the actual schema; do not skip the error by deleting data volumes or editing the version number directly.

PostgreSQL uses `migrations/versioned/` and SQLite uses `migrations/sqlite/`. The Make command below calls the PostgreSQL migration script; Lite's SQLite database must be handled with its own driver and migration directory, not with a PostgreSQL DSN.

```bash
make migrate-version
```

You can also run a read-only check in the target database:

```sql
SELECT version, dirty FROM schema_migrations;
-- The next two statements are PostgreSQL only
SELECT version();
SELECT extname, extversion FROM pg_extension;
```

### Missing extensions or insufficient privileges

A missing `gin_trgm_ops` usually points to `pg_trgm`, a missing `vector` type to pgvector, and BM25 features depend on ParadeDB's `pg_search`. First confirm which database the application actually connects to, then have the database administrator check the extension packages, the extensions in the database, and object ownership.

```sql
SELECT name, default_version, installed_version
FROM pg_available_extensions
WHERE name IN ('pg_trgm', 'vector', 'pg_search');
```

Extension files not being installed and insufficient SQL privileges are different problems; `CREATE EXTENSION IF NOT EXISTS` neither installs operating-system packages nor upgrades an extension that already exists. Create only the extensions the current deployment actually needs, so that a deployment using an external retrieval engine is not accidentally switched to PostgreSQL retrieval.

For upgrading the image and extension version of an existing ParadeDB database, see [ParadeDB Upgrade](06-paradedb-upgrade.md).

### Dirty state

`AUTO_RECOVER_DIRTY` is on by default and tries to reset the migration version and rerun at startup. It is a retry mechanism and cannot fix schema differences caused by missing extensions, a full disk or manual changes.

For manual recovery, first stop application writes and check whether the failed migration left partial changes. Use `force` only after confirming that the database matches the last successful version and that the failed migration can be safely rerun. It **only changes the migration version marker and does not run any rollback SQL**. For example, after confirming that the last successful version is 98:

```bash
make migrate-force version=98
make migrate-up
```

98 here is an example; replace it with the version you actually confirmed, and do not mechanically subtract one from the number in the error. A failed initial migration also requires a separate check of the initialization state. After recovering, restart the app, confirm that System Information no longer shows an error, and verify the affected features.

### Interrupted concurrent index build

Migration `000106` builds an index on `messages` with `CREATE INDEX CONCURRENTLY`, which does not block writes during the build. If the build is interrupted (process restart, timeout, full disk), it leaves an INVALID `idx_messages_session_created_id`, and `IF NOT EXISTS` will not rebuild it. Confirm and drop that index first, then restore the migration state as described above and rerun:

```sql
SELECT indexrelid::regclass, indisvalid FROM pg_index
WHERE indexrelid = 'idx_messages_session_created_id'::regclass;
DROP INDEX CONCURRENTLY IF EXISTS idx_messages_session_created_id;
```

### Full disk and schema differences

Index builds need extra temporary space. On `No space left on device`, check the capacity of the database data volume and temporary directory, free up space, and then recover according to the actual migration state.

When column types, indexes or constraints do not match what a migration expects, compare against the failed file and the last successful version; do not repeatedly run the failing SQL directly on the production database. When you need to reproduce a write, verify it first in an isolated database restored from a backup.

## What to include when reporting an issue

Provide the application version/commit, the time of the failure, the full error, the database type and version, the migration version/dirty state, and any relevant non-default configuration. Redact tokens, passwords, connection strings and document content from the logs first. For how database migrations are implemented and developed, see [Database and Migrations](../06-development/02-database-schema.md).
