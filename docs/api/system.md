# System Management API

[Back to Table of Contents](./README.md)

| Method | Path                              | Description                     |
| ------ | --------------------------------- | -------------------------------- |
| GET    | `/system/info`                    | Get system information           |
| GET    | `/system/parser-engines`          | Get parser engine list           |
| POST   | `/system/parser-engines/check`    | Check parser engine availability |
| POST   | `/system/docreader/reconnect`     | Reconnect document parsing service |
| GET    | `/system/storage-engine-status`   | Get storage engine status        |
| POST   | `/system/storage-engine-check`    | Check storage engine connectivity |

## GET `/system/info` - Get System Information

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/system/info' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json'
```

**Response**:

```json
{
    "data": {
        "version": "1.2.0",
        "edition": "community",
        "commit_id": "a1b2c3d",
        "build_time": "2025-08-12T08:00:00Z",
        "go_version": "go1.21.5",
        "keyword_index_engine": "bleve",
        "vector_store_engine": "milvus",
        "graph_database_engine": "neo4j",
        "minio_enabled": true,
        "db_version": "20250810_001"
    },
    "success": true
}
```

## GET `/system/parser-engines` - Get Parser Engine List

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/system/parser-engines' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json'
```

**Response**:

```json
{
    "data": [
        {
            "name": "docreader",
            "label": "DocReader",
            "description": "High-precision document parsing engine",
            "available": true
        },
        {
            "name": "tika",
            "label": "Apache Tika",
            "description": "General-purpose document parsing engine",
            "available": false
        }
    ],
    "connected": true,
    "success": true
}
```

## POST `/system/parser-engines/check` - Check Parser Engine Availability

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/system/parser-engines/check' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "addr": "http://docreader:8000"
}'
```

**Response**:

```json
{
    "data": [
        {
            "name": "docreader",
            "label": "DocReader",
            "description": "High-precision document parsing engine",
            "available": true
        }
    ],
    "success": true
}
```

## POST `/system/docreader/reconnect` - Reconnect Document Parsing Service

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/system/docreader/reconnect' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "addr": "http://docreader:8000"
}'
```

**Response**:

```json
{
    "success": true
}
```

## GET `/system/storage-engine-status` - Get Storage Engine Status

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/system/storage-engine-status' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json'
```

**Response**:

```json
{
    "data": {
        "engines": [
            {
                "name": "minio",
                "available": true,
                "description": "MinIO object storage"
            },
            {
                "name": "cos",
                "available": false,
                "description": "Tencent Cloud COS object storage"
            },
            {
                "name": "s3",
                "available": false,
                "description": "AWS S3 object storage"
            },
            {
                "name": "oss",
                "available": false,
                "description": "Alibaba Cloud OSS object storage"
            }
        ],
        "minio_env_available": true
    },
    "success": true
}
```

## POST `/system/storage-engine-check` - Check Storage Engine Connectivity

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/system/storage-engine-check' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "provider": "minio",
    "minio": {
        "endpoint": "localhost:9000",
        "access_key": "minioadmin",
        "secret_key": "minioadmin",
        "bucket": "weknora",
        "use_ssl": false
    }
}'
```

**Response**:

```json
{
    "data": {
        "ok": true,
        "message": "Connection successful",
        "bucket_created": false
    },
    "success": true
}
```
