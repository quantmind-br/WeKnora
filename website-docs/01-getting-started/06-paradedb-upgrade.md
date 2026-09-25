# ParadeDB Upgrade for Existing Databases

The repository's production and development Compose files use `paradedb/paradedb:v0.22.6-pg17`. This page covers the **0.22.2 → 0.22.6 upgrade with PostgreSQL staying on major version 17**; it does not apply to upgrades across PostgreSQL major versions, nor does it mean other older versions in Helm can follow it as-is.

## Upgrade steps

Run all commands from the repository root. In the development environment, database commands use `docker compose -f docker-compose.dev.yml`, and the development backend is stopped manually on the host; the development Compose has no `app` service. Keep the original data volume; **do not run `down -v`, delete volumes, or switch to a PG18 image**.

1. Stop the app and every other database writer; if Langfuse is enabled, also stop its web/worker, and stop any development backend running locally.

   ```bash
   # Standard deployment; in development, stop the backend process on the host
   docker compose stop app
   # Only when Langfuse is enabled
   docker compose stop langfuse-web langfuse-worker
   ```

2. Back up all databases and roles, including the Langfuse database, and confirm the backup can be restored. Store the backup outside the database data volume.

   ```bash
   umask 077
   docker compose exec -T postgres sh -c 'pg_dumpall -U "$POSTGRES_USER"' > paradedb-before-upgrade.sql
   ```

   If the old image cannot start on the current CPU, first save a snapshot of the stopped data volume, then make a logical backup on a compatible machine or verify that the snapshot can be restored, before touching the only copy of the data.

3. With the updated Compose files, replace only the database container:

   ```bash
   docker compose pull postgres
   docker compose up -d --no-deps --wait postgres
   ```

4. Complete the extension's SQL upgrade. Replacing the image alone does not update the `pg_extension` version inside existing databases.

   ```bash
   docker compose exec -T postgres sh -c 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" -v ON_ERROR_STOP=1' <<'SQL'
   ALTER EXTENSION pg_search UPDATE TO '0.22.6';
   SELECT extname, extversion FROM pg_extension WHERE extname IN ('pg_search', 'vector');
   SELECT * FROM paradedb.version_info();
   SQL
   ```

   Run the same in **every other database where `pg_search` is installed**, including `postgres`, template databases or the Langfuse database if the extension was installed there. Do not install the extension into databases that did not need it just to upgrade. Both the catalog and `paradedb.version_info()` should report 0.22.6.

5. Restart the writer services that were running before, confirm the database migration status, and run representative keyword and vector retrievals. This patch upgrade does not require rebuilding all indexes or re-importing documents; keep the backup until verification is complete.

## Scope of the automatic migration

Migration `000099` upgrades the extension only inside the WeKnora database: the installed version must be 0.22.2–0.22.5, and the server must provide the 0.22.6 extension package. It honors `app.skip_embedding`, does not install a missing extension, and does not handle other version lines.

If this migration already ran **before** the database image was replaced, it will not run again automatically after the new image is installed; run the SQL above manually. Older applications without `000099` also need the manual upgrade. Extensions in other databases cannot be managed by WeKnora's migrations.

## Rollback and reproducible verification

Rolling back requires restoring the pre-upgrade backup/snapshot to a separate data volume and using it with the old image. Changing the image tag back to the old version does not undo the extension SQL changes, and the down file of `000099` does not try to downgrade the extension either.

The repository provides an isolated verification script:

```bash
docker pull paradedb/paradedb:v0.22.2-pg17
docker pull paradedb/paradedb:v0.22.6-pg17
python3 scripts/test_paradedb_upgrade.py
```

The script uses a temporary container and volume without mapping ports; it verifies the in-place upgrade of the same data volume, table contents, keyword/vector retrieval, migration idempotency and results after a restart, and outputs backups and logs. The test case verifies fixed test data; for a deployment you still need to check your own data and retrieval workload.
