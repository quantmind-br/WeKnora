# migrations/mysql — unwired MySQL schema script

⚠️ **The SQL here is not executed by any code or script. MySQL is currently not a supported primary database for WeKnora.**

## Current state

In `initDatabase()` in `internal/container/container.go`, the `DB_DRIVER` switch has only two branches:

| `DB_DRIVER` | Migration source |
| --- | --- |
| `postgres` | `migrations/versioned/` (golang-migrate incremental migrations) |
| `sqlite` | `migrations/sqlite/` (golang-migrate incremental migrations, Lite mode) |

Any other value returns `unsupported database driver`, so setting `DB_DRIVER=mysql` makes the service fail at startup.

`00-init-db.sql` in this directory is a one-off schema creation script covering 10 core tables (`tenants`, `models`, `knowledge_bases`, `knowledges`, `sessions`, `messages`, `message_suggestion_sets`, `message_suggestion_events`, `chunks`, `chunk_revisions`). It gets updated along the way when someone changes these base tables, but **no Go code, Makefile target or compose configuration references it**.

## What is still missing for real MySQL support

This schema script alone is not enough — it only matches the initial schema and lacks an incremental migration set equivalent to `migrations/versioned/` (the Postgres side already has 100+ incremental migrations). Full wiring needs at least:

1. Add `case "mysql"` to `initDatabase()`, wiring up `gorm.io/driver/mysql` and a MySQL DSN for golang-migrate;
2. Build an incremental migration sequence under `migrations/mysql/` and keep it in sync with the schema evolution in `versioned/`;
3. Provide MySQL equivalents or fallbacks for Postgres-specific features (`JSONB`, array types, `ON CONFLICT`, ParadeDB BM25 indexes, etc.);
4. Decide how vector retrieval is handled — MySQL itself provides no vector index, so it has to rely on an external vector database (`RETRIEVE_DRIVER`).

See [#1418](https://github.com/Tencent/WeKnora/issues/1418) for the related discussion. Until the work above is done, do not claim `DB_DRIVER=mysql` support in docs or configuration comments.
