# Storage Backends

Storage backends hold original files, extracted images, and export artifacts. A space can register multiple instances, designate a default instance, and choose a separate storage location for individual knowledge bases.

Multi-instance storage fits the following use cases:

- Keep material from different teams or projects in separate buckets, managing usage and permissions for each;
- Choose the bucket's region according to data residency requirements;
- When migrating to cloud object storage, new knowledge bases use the new backend while existing knowledge bases keep accessing their original files.

<Screenshot
  src="/screenshots/settings-storage-backends.png"
  caption="Storage backend settings: multi-instance list, default instance, and connectivity test"
  hint="Shows cards for registered storage backends (provider, status, default flag) along with the create/edit form, including the 'Test Connection' result." />

## Registering and selecting storage backends {#how-to-configure-it}

Space Admins can register and manage instances under "Settings → Storage":

1. Create a new backend, choose a provider (`local` / `minio` / `cos` / `tos` / `s3` / `oss` / `ks3` / `obs`, consistent with the storage providers in the [document ingestion pipeline](../02-architecture/03-document-pipeline.md)), and fill in the connection parameters (see the table below). The selectable list is restricted by the `STORAGE_ALLOW_LIST` environment variable; leaving it empty makes all providers selectable;
2. **Test the connection**: the test performs an actual read/write against the storage, verifying the endpoint, bucket, and credentials;
3. Set the instance as the space default. When a new knowledge base doesn't specify an instance, this default is used;
4. When a knowledge base needs its own instance, select it under the "Storage" tab in the knowledge base edit dialog.

A knowledge base's storage backend can be changed while the knowledge base is empty; once it contains files, the UI disables the selector and prompts that a migration is required. File paths depend on the backend in use at ingestion time, so changing the binding directly would break access to the original files. If a change is needed, create a new knowledge base and migrate the content over.

## Connection parameters {#connection-parameters}

All providers share one set of configuration fields; `access_key_id` / `secret_access_key` are stored encrypted and masked in API responses.

| Name | Type | Default | Description |
| --- | --- | --- | --- |
| `endpoint` | string | empty | Service address. Required for `minio` (`mode=remote`), `tos`, `s3`, `oss`, `ks3`, and `obs`; not needed for `cos`. SSRF validation runs on save, and internal addresses must be added to `SSRF_WHITELIST` |
| `region` | string | empty | Region. Required for `cos`, `tos`, `s3`, `oss`, `ks3`, and `obs` |
| `access_key_id` / `secret_access_key` | string | empty | Access keys; for COS these are SecretId / SecretKey. Not needed for `local` or MinIO with `mode=docker` |
| `bucket_name` | string | empty | Bucket; required for everything except `local` |
| `path_prefix` | string | empty | Object key prefix; must be a relative path and cannot start with `/` or contain `..` |
| `mode` | string | `remote` | MinIO only: `docker` uses the MinIO bundled with the deployment (address and keys are read from environment variables such as `MINIO_ENDPOINT`), `remote` connects to an external MinIO |
| `use_ssl` | bool | false | Whether MinIO, S3, and OBS use HTTPS |
| `force_path_style` | bool | false | S3 only: use path-style addressing; most S3-compatible services need this enabled |
| `app_id` | string | empty | COS only: Tencent Cloud AppID |
| `temp_bucket_name` / `temp_region` | string | empty | COS, TOS, and OSS only: bucket and region used for temporary files; OSS also needs `use_temp_bucket=true` to enable them |

- **OBS**: when the endpoint is a domain name, virtual-hosted addressing is used (`<bucket>.<endpoint>`), because Huawei Cloud has rejected path-style requests to domain-name endpoints since 2023-12-30; when the endpoint is an IP, path-style is still used. Without a configured proxy domain, file URLs look like `<scheme>://<bucket>.<endpoint-host>/<key>`.
- **Transfer timeouts**: single uploads/downloads for S3, COS, KS3, OBS, and OSS are no longer bound by the 30-second overall timeout; instead a single transfer may take up to 30 minutes. Connection setup, the TLS handshake, and waiting for response headers still have their own timeouts, so an unresponsive peer fails fairly quickly.
- **KS3 redirects**: when following redirects, SSRF validation runs on every hop, and request signatures are no longer forwarded to the redirect target host.

## Difference from vector stores

File storage and vector storage manage original files and retrieval indexes respectively:

| | Storage Backend | Vector Store |
| --- | --- | --- |
| What it stores | Original files, images, export artifacts | Vectors and retrieval indexes |
| Where it's configured | "Settings → Storage" | "Settings → Vector Store" |
| Knowledge base field | `storage_backend_id` | `vector_store_id` |
| Related section | This page | [Retrieval Engines & Vector Stores](05-retrieval-engines.md) |

## API reference {#api}

| Method | Path | Permission |
| --- | --- | --- |
| GET | `/storage-backends/types` | Viewer+, returns the list of provider names allowed by `STORAGE_ALLOW_LIST` |
| GET | `/storage-backends`, `/storage-backends/:id` | Viewer+ |
| POST | `/storage-backends` | Admin+ |
| PUT / DELETE | `/storage-backends/:id` | Admin+ |
| POST | `/storage-backends/test` | Admin+, tests a connection using unsaved parameters |
| POST | `/storage-backends/:id/test` | Admin+, tests an already-saved instance |
| PUT | `/storage-backends/:id/default` | Admin+, sets it as the tenant default |

API Keys require the `manage_storage_backends` capability or full access.

## Data model and compatibility rules {#data-model-and-a-few-constraints}

Key fields of the `storage_backends` table (isolated by `tenant_id`, soft-deleted):

| Field | Description |
| --- | --- |
| `name` | Unique within the space (partial unique index accounting for soft deletes) |
| `provider` | Storage type |
| `config` | JSONB, contains encrypted credentials |
| `source` | `user` (registered via the UI or API) / `env` (generated from environment variable configuration) |
| `status` | `active` / `disabled` |
| `legacy_alias` | See below |

`legacy_alias` provides compatibility with the legacy storage configured via environment variables. On upgrade an alias record is created so that existing file paths remain resolvable without moving any data. Only one alias record is allowed per provider within a given space, and manually registered instances are stored separately.
