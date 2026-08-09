# Storage Backends

Original files, extracted images, and export artifacts all need to land on some object storage. Early on, this could only be configured via environment variables for a single global storage setup; starting with migration `000068`, this changed to support **registering multiple storage instances** — a tenant picks one as the default, and individual knowledge bases can also be bound to a specific instance.

Typical use cases:

- Data from different teams/projects lands in different buckets, for easier cost allocation and permission isolation;
- Compliance requirements mandate that certain document types must reside in buckets in a specific region;
- When migrating from self-hosted MinIO to cloud object storage, new knowledge bases use the new backend while old ones remain untouched.

<Screenshot
  src="/screenshots/settings-storage-backends.png"
  caption="Storage backend settings: multi-instance list, default instance, and connectivity test"
  hint="Shows cards for registered storage backends (provider, status, default flag) along with the create/edit form, including the 'Test Connection' result." />

## How to configure it

The entry point is in "Settings → Storage" (the `storage` section, requires Admin):

1. Create a new backend, choose a provider (`local` / `minio` / `cos` / `oss` / `s3` / `tos` / `obs`, etc., consistent with the storage providers in the [document ingestion pipeline](../02-architecture/03-document-pipeline.md)), and fill in the connection parameters;
2. **Click "Test" before saving**: the connectivity test performs an actual read/write, so a misconfigured bucket or an expired key is caught immediately, rather than only surfacing when a document is uploaded later;
3. If needed, set it as the tenant default (`PUT /storage-backends/:id/default`, which also writes back to `tenants.default_storage_backend_id`). When a new knowledge base doesn't specify an instance, this default is used;
4. If a specific knowledge base needs a different instance, select it under the "Storage" tab in the knowledge base edit dialog — this corresponds to `knowledge_bases.storage_backend_id`.

## API

| Method | Path | Permission |
| --- | --- | --- |
| GET | `/storage-backends/types` | Viewer+, returns the supported providers and their field definitions |
| GET | `/storage-backends`, `/storage-backends/:id` | Viewer+ |
| POST | `/storage-backends` | Admin+ |
| PUT / DELETE | `/storage-backends/:id` | Admin+ |
| POST | `/storage-backends/test` | Admin+, tests a connection using unsaved parameters |
| POST | `/storage-backends/:id/test` | Admin+, tests an already-saved instance |
| PUT | `/storage-backends/:id/default` | Admin+, sets it as the tenant default |

API Keys require the `manage_storage_backends` capability or full access.

## Data model and a few constraints

Key fields of the `storage_backends` table (isolated by `tenant_id`, soft-deleted):

| Field | Description |
| --- | --- |
| `name` | Unique within the tenant (partial unique index accounting for soft deletes) |
| `provider` | Storage type |
| `config` | JSONB, contains encrypted credentials |
| `source` | `user` (registered via the UI) / other (system-generated) |
| `status` | `active` / disabled |
| `legacy_alias` | See below |

**`legacy_alias` exists to enable a smooth upgrade path**: the single global storage configuration set up via environment variables before the upgrade gets converted into an alias record, so that file paths for existing knowledge bases remain resolvable without requiring a data migration. Only one alias record is allowed per provider within a given tenant (enforced by a partial unique index), so it won't get mixed up with instances you register manually.

**Once a knowledge base has files, its storage choice can no longer be changed**: a knowledge base's storage selection can be **changed while it's empty**, but once it contains files, the selector in the UI is disabled and prompts that a migration is required (`KBStorageSettings.vue` checks this via `hasFiles`). The reason is that the paths of already-ingested files were generated based on the backend in use at the time — changing the binding directly would orphan the old files. If a change is truly needed, create a new knowledge base and migrate the content over.

## Difference from vector stores

These two are easy to confuse:

| | Storage Backend | Vector Store |
| --- | --- | --- |
| What it stores | Original files, images, export artifacts | Vectors and retrieval indexes |
| Where it's configured | "Settings → Storage" | "Settings → Vector Store" |
| Knowledge base field | `storage_backend_id` | `vector_store_id` |
| Related section | This page | [Retrieval Engines & Vector Stores](05-retrieval-engines.md) |
