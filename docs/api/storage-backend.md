--- DOCUMENT START ---
# Storage Backend API

[Back to Table of Contents](./README.md)

The Storage Backend API is used to manage a space's object/file storage instances. A space can register multiple storage instances (`local`, `minio`, `cos`, `tos`, `s3`, `oss`, `ks3`, `obs`) and bind different knowledge bases to different instances; each space also has a default instance at the space level (`default_storage_backend_id`), which new knowledge bases use when they are not explicitly bound to a specific instance.

The API manages both user-created instances (`source: "user"`) and read-only instances generated from environment variable snapshots (`source: "env"`). Storage backend CRUD requires the **Admin+** role; for API Keys, the `manage_storage_backends` capability (or full-access) is required.

| Method | Path                              | Description                                   | Minimum Permission |
| ------ | --------------------------------- | -------------------------------------- | -------- |
| GET    | `/storage-backends/types`         | Get the storage types allowed by `STORAGE_ALLOW_LIST` | Viewer+  |
| POST   | `/storage-backends/test`          | Test connectivity using a raw configuration (not persisted) | Admin+   |
| POST   | `/storage-backends`               | Create a storage instance                           | Admin+   |
| GET    | `/storage-backends`               | Get the list of storage instances                       | Viewer+  |
| GET    | `/storage-backends/:id`           | Get storage instance details                       | Viewer+  |
| PUT    | `/storage-backends/:id`           | Update a storage instance (name/credentials/status)          | Admin+   |
| DELETE | `/storage-backends/:id`           | Delete a storage instance (soft delete)                  | Admin+   |
| POST   | `/storage-backends/:id/test`      | Test connectivity of a saved instance                  | Admin+   |
| PUT    | `/storage-backends/:id/default`   | Set as the space's default storage instance                    | Admin+   |

> Sensitive fields (`access_key_id`, `secret_access_key`) are masked in all responses. If a masked placeholder is submitted during an update, the original real credentials in the database are preserved and not overwritten by the placeholder.

## Storage Configuration Fields (`config`)

Different providers share the same normalized configuration object, using the relevant subset per provider:

| Field                | Type    | Description                                                          |
| ------------------- | ------- | ------------------------------------------------------------------------- |
| mode                | string  | MinIO mode: `docker` (reuses environment variable credentials) or `remote`            |
| endpoint            | string  | Object storage endpoint (COS uses region instead, no endpoint needed)          |
| region              | string  | Region                                                          |
| access_key_id       | string  | Access key ID (corresponds to SecretID for COS); masked in responses                   |
| secret_access_key   | string  | Access key secret (corresponds to SecretKey for COS); masked in responses             |
| bucket_name         | string  | Bucket name                                                   |
| path_prefix         | string  | Object prefix; must be a relative path, cannot start with `/` or contain `..` traversal            |
| app_id              | string  | Tencent Cloud COS AppID                                                              |
| use_ssl             | boolean | Whether to use SSL                                                  |
| force_path_style    | boolean | Whether S3 uses path-style addressing                                   |
| use_temp_bucket     | boolean | Whether OSS uses a temporary bucket                                       |
| temp_bucket_name    | string  | Temporary bucket name                                              |
| temp_region         | string  | Temporary bucket region                                              |

> `endpoint`, `region`, `bucket_name`, and `path_prefix` determine the physical location of objects and **cannot be changed after creation** (update attempts will be rejected); use the storage migration process if migration is needed. Credentials can be rotated individually via update.

## GET `/storage-backends/types` - Get Allowed Storage Types

Returns the list of providers allowed by `STORAGE_ALLOW_LIST`, which can be used to dynamically generate frontend forms.

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/storage-backends/types' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**:

```json
{
    "success": true,
    "data": ["local", "minio", "cos", "s3"]
}
```

## POST `/storage-backends/test` - Test Connectivity with a Raw Configuration

Runs a connectivity test using configuration from the frontend form that has not yet been saved, without writing to the database.

**Parameters (Request Body)**:

| Field     | Type   | Required | Description                                        |
| -------- | ------ | ---- | ------------------------------------------- |
| name     | string | Yes   | Instance display name                                  |
| provider | string | Yes   | Storage type, taken from `/storage-backends/types`    |
| config   | object | No   | Storage configuration fields corresponding to this provider              |

**Request**:

```curl
curl --location --request POST 'http://localhost:8080/api/v1/storage-backends/test' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "name": "s3-hot",
    "provider": "s3",
    "config": {
        "endpoint": "https://s3.example.com",
        "region": "ap-test-1",
        "access_key_id": "AKID",
        "secret_access_key": "SECRET",
        "bucket_name": "weknora"
    }
}'
```

**Response (Success)**:

```json
{
    "success": true
}
```

**Response (Failure)**:

```json
{
    "success": false,
    "error": "Connection refused, please confirm the service is running and the port is correct"
}
```

> When the test fails, the HTTP status code is still `200`; the error message is returned via `success: false` + `error`. The `error` message is sanitized and will not leak internal hostnames, IPs, ports, or TLS details.

## POST `/storage-backends` - Create a Storage Instance

Creates a new storage instance for the current space. Before creation, the configuration is validated, an SSRF check is performed (except for local storage and MinIO in docker mode), and a connectivity test is run; failure at any step returns `400`. Instance names must be unique within the same space.

**Parameters (Request Body)**:

| Field     | Type   | Required | Description                                        |
| -------- | ------ | ---- | ------------------------------------------- |
| name     | string | Yes   | Instance display name (unique within the space)                     |
| provider | string | Yes   | Storage type, taken from `/storage-backends/types`    |
| config   | object | No   | Storage configuration fields corresponding to this provider              |
| status   | string | No   | `active` (default) or `disabled`               |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/storage-backends' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "name": "s3-hot",
    "provider": "s3",
    "config": {
        "endpoint": "https://s3.example.com",
        "region": "ap-test-1",
        "access_key_id": "AKID",
        "secret_access_key": "SECRET",
        "bucket_name": "weknora",
        "path_prefix": "prod"
    }
}'
```

**Response** (201):

```json
{
    "success": true,
    "data": {
        "id": "550e8400-e29b-41d4-a716-446655440000",
        "name": "s3-hot",
        "provider": "s3",
        "config": {
            "endpoint": "https://s3.example.com",
            "region": "ap-test-1",
            "access_key_id": "***",
            "secret_access_key": "***",
            "bucket_name": "weknora",
            "path_prefix": "prod"
        },
        "source": "user",
        "status": "active",
        "legacy_alias": false,
        "created_at": "2026-07-15T10:00:00Z",
        "updated_at": "2026-07-15T10:00:00Z"
    }
}
```

## GET `/storage-backends` - Get the List of Storage Instances

Returns all storage instances for the current space (credentials masked), along with the space's default instance id at the top level.

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/storage-backends' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**:

```json
{
    "success": true,
    "data": [
        {
            "id": "550e8400-e29b-41d4-a716-446655440000",
            "name": "s3-hot",
            "provider": "s3",
            "config": { "endpoint": "https://s3.example.com", "access_key_id": "***", "secret_access_key": "***", "bucket_name": "weknora" },
            "source": "user",
            "status": "active",
            "legacy_alias": false
        }
    ],
    "default_storage_backend_id": "550e8400-e29b-41d4-a716-446655440000"
}
```

## GET `/storage-backends/:id` - Get Storage Instance Details

Retrieves a single storage instance under the current space by ID, with credentials masked.

**Path Parameters**:

| Field | Type   | Required | Description          |
| ---- | ------ | ---- | ------------- |
| id   | string | Yes   | Storage instance ID   |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/storage-backends/550e8400-e29b-41d4-a716-446655440000' \
--header 'X-API-Key: sk-xxxxx'
```

## PUT `/storage-backends/:id` - Update a Storage Instance

Updates the mutable fields of an instance (`name`, credentials, `status`). `provider` and the physical location fields (`endpoint`, `region`, `bucket_name`, `path_prefix`) cannot be changed; attempting to change them returns `400`. Instances sourced from environment variables (`source: "env"`) are read-only and cannot be updated. Updates also trigger validation and a connectivity test.

> If `access_key_id` / `secret_access_key` are submitted as masked placeholders (`***`), the original real credentials in the database are preserved.

**Disable Protection**: Changing an instance to `disabled` is rejected (`400`) if it is currently the default instance or still has knowledge bases bound to it.

**Path Parameters**:

| Field | Type   | Required | Description          |
| ---- | ------ | ---- | ------------- |
| id   | string | Yes   | Storage instance ID   |

**Request**:

```curl
curl --location --request PUT 'http://localhost:8080/api/v1/storage-backends/550e8400-e29b-41d4-a716-446655440000' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "name": "s3-hot-renamed",
    "provider": "s3",
    "config": {
        "access_key_id": "***",
        "secret_access_key": "NEW_SECRET"
    }
}'
```

## DELETE `/storage-backends/:id` - Delete a Storage Instance

Performs a soft delete on a storage instance. Deletion is rejected (`400`) in the following cases: the instance is the space's default instance, it still has knowledge bases bound to it, it is sourced from environment variables (read-only), or it is a legacy alias (old file paths may still reference it). Deletion runs within a transaction; on PostgreSQL, a row lock is applied to the target row to avoid concurrent binding race conditions.

**Path Parameters**:

| Field | Type   | Required | Description          |
| ---- | ------ | ---- | ------------- |
| id   | string | Yes   | Storage instance ID   |

**Request**:

```curl
curl --location --request DELETE 'http://localhost:8080/api/v1/storage-backends/550e8400-e29b-41d4-a716-446655440000' \
--header 'X-API-Key: sk-xxxxx'
```

**Response (Success)**:

```json
{
    "success": true
}
```

## POST `/storage-backends/:id/test` - Test Connectivity of a Saved Instance

Runs a connectivity test on a saved storage instance using its stored credentials.

**Path Parameters**:

| Field | Type   | Required | Description          |
| ---- | ------ | ---- | ------------- |
| id   | string | Yes   | Storage instance ID   |

**Request**:

```curl
curl --location --request POST 'http://localhost:8080/api/v1/storage-backends/550e8400-e29b-41d4-a716-446655440000/test' \
--header 'X-API-Key: sk-xxxxx'
```

**Response (Success)**:

```json
{
    "success": true
}
```

> As with `/storage-backends/test`, the HTTP status code remains `200` when the test fails, with a sanitized error returned via `success: false` + `error`.

## PUT `/storage-backends/:id/default` - Set as the Space's Default Instance

Marks a storage instance as the space's default instance. Only instances with `active` status can be set as default. New knowledge bases that are not explicitly bound to a storage instance will use the default instance.

**Path Parameters**:

| Field | Type   | Required | Description          |
| ---- | ------ | ---- | ------------- |
| id   | string | Yes   | Storage instance ID   |

**Request**:

```curl
curl --location --request PUT 'http://localhost:8080/api/v1/storage-backends/550e8400-e29b-41d4-a716-446655440000/default' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**:

```json
{
    "success": true
}
```

## Environment Variable Storage Instances

Storage configured via environment variables such as `STORAGE_TYPE` participates in instance resolution as a read-only instance (`source: "env"`, `legacy_alias: true`), so that env-only deployments and user-managed instances follow the same resolution path. These instances are refreshed from environment variables on every startup and cannot be updated or deleted via the API.

## Error Codes

| HTTP Status Code | Meaning                                                             |
| ----------- | ---------------------------------------------------------------- |
| 400         | Invalid request parameters, validation failure, SSRF validation failure, connectivity test failure, attempt to change an immutable field, attempt to modify a read-only instance, attempt to delete a protected instance, attempt to disable a referenced instance, attempt to set a non-active instance as default |
| 401         | Not authenticated (missing space context or API Key)                               |
| 403         | Insufficient permissions (requires Admin+ or the API Key `manage_storage_backends` capability) |
| 404         | Storage instance not found                                                   |
| 409         | A storage instance with the same name already exists                                             |

--- DOCUMENT END ---
