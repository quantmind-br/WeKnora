# DocReader Service

DocReader is the gRPC service in the WeKnora project responsible for document parsing and processing. It supports reading multiple document formats, OCR recognition, multimodal processing, and more.

## Docker Compose Environment Variable Configuration

In the `docker-compose.yml` file, the docreader service is configured with the following environment variables:

```yaml
docreader:
  image: wechatopenai/weknora-docreader:${WEKNORA_VERSION:-latest}
  environment:
    - MINIO_ENDPOINT=minio:9000
    - MINIO_PUBLIC_ENDPOINT=http://localhost:${MINIO_PORT:-9000}
    - MINERU_ENDPOINT=${MINERU_ENDPOINT:-}
    - MAX_FILE_SIZE_MB=${MAX_FILE_SIZE_MB:-}
```

### Environment Variable Descriptions

#### 1. MINIO_ENDPOINT

- **Description**: The internal access address for the MinIO service (inter-container communication)
- **Default**: `minio:9000`
- **Purpose**: The DocReader service uses this address to connect to the MinIO object storage service, for reading and storing files during document processing
- **Configuration example**:
  ```yaml
  - MINIO_ENDPOINT=minio:9000  # Internal Docker network address
  ```

#### 2. MINIO_PUBLIC_ENDPOINT

- **Description**: The public access address for the MinIO service (external access)
- **Default**: `http://localhost:9000`
- **Purpose**: Used to generate externally accessible file URLs, for example when returning image links after document parsing
- **Important notes**:
  - If access is needed from other devices or containers, replace `localhost` with the actual IP address
  - You can configure `MINIO_PORT` in the `.env` file to customize the port
- **Configuration example**:
  ```bash
  # .env file
  MINIO_PORT=9000
  ```
  Or modify it directly in docker-compose.yml:
  ```yaml
  - MINIO_PUBLIC_ENDPOINT=http://192.168.1.100:9000  # Use the actual IP
  ```

#### 3. MINERU_ENDPOINT

- **Description**: The access address for the MinerU service (optional)
- **Default**: Empty (MinerU not used)
- **Purpose**: MinerU is an advanced document parsing service that supports more complex document structure recognition and processing. Once this variable is configured, DocReader can call MinerU for document parsing
- **Configuration example**:
  ```bash
  # .env file
  MINERU_ENDPOINT=http://mineru-service:8080
  ```

#### 4. MAX_FILE_SIZE_MB

- **Description**: The maximum allowed upload file size (in MB)
- **Default**: `50` MB
- **Purpose**: Limits the file size accepted by the gRPC service, preventing overly large files from causing service crashes or performance issues
- **Configuration example**:
  ```bash
  # .env file
  MAX_FILE_SIZE_MB=100  # Allow files up to 100MB
  ```

## Other Configurable Environment Variables

In addition to the variables already configured in docker-compose.yml, DocReader also supports the following environment variables (which can be added as needed):

### gRPC Configuration

- `DOCREADER_GRPC_MAX_WORKERS`: Maximum number of worker threads for the gRPC service (default: 4)
- `DOCREADER_GRPC_PORT`: The port the gRPC service listens on (default: 50051)

### Parser Resource Control

- `DOCREADER_MARKITDOWN_MAX_WORKERS`: Maximum concurrency for MarkItDown parsing (default: 1; set to 0 to disable throttling)
- `DOCREADER_PDF_RENDER_MAX_WORKERS`: Maximum concurrency for rendering scanned PDFs into images (default: 1; set to 0 to disable throttling)
- `DOCREADER_PDF_RENDER_DPI`: DPI for rendering scanned PDFs (default: 200)
- `DOCREADER_PDF_JPEG_QUALITY`: JPEG output quality for scanned PDFs (default: 85; automatically clamped to the 1–95 range)

### OCR / VLM

DocReader no longer bundles its own OCR and VLM backends. Scanned PDFs are rendered into JPEG images and then handed off to the Go App side to call the OCR/VLM services — see the main project documentation for related configuration.

### Storage Configuration

DocReader supports multiple storage backends:

#### MinIO/S3 Storage (recommended)

- `STORAGE_TYPE`: Set to `minio`
- `MINIO_ACCESS_KEY_ID`: MinIO access key ID (default: minioadmin)
- `MINIO_SECRET_ACCESS_KEY`: MinIO secret access key (default: minioadmin)
- `MINIO_BUCKET_NAME`: MinIO bucket name (default: WeKnora)
- `MINIO_PATH_PREFIX`: File path prefix
- `MINIO_USE_SSL`: Whether to use SSL (default: false)

#### Tencent Cloud COS Storage

- `STORAGE_TYPE`: Set to `cos`
- `COS_SECRET_ID`: COS access key ID
- `COS_SECRET_KEY`: COS access key
- `COS_REGION`: COS region
- `COS_BUCKET_NAME`: COS bucket name
- `COS_APP_ID`: COS app ID
- `COS_PATH_PREFIX`: File path prefix
- `COS_ENABLE_OLD_DOMAIN`: Whether to use the legacy domain (default: true)

#### Alibaba Cloud OSS Storage

- `STORAGE_TYPE`: Set to `oss`
- `OSS_ACCESS_KEY_ID`: OSS access key ID
- `OSS_ACCESS_KEY_SECRET`: OSS access key secret
- `OSS_ENDPOINT`: OSS endpoint (e.g. `oss-cn-hangzhou.aliyuncs.com`)
- `OSS_BUCKET_NAME`: OSS bucket name
- `OSS_REGION`: OSS region (e.g. `cn-hangzhou`)
- `OSS_PATH_PREFIX`: File path prefix

### Proxy Configuration

If external services need to be accessed through a proxy:

- `EXTERNAL_HTTP_PROXY`: HTTP proxy address
- `EXTERNAL_HTTPS_PROXY`: HTTPS proxy address

### Image Processing Configuration

Scanned PDFs are rendered into JPEG images and then handed off to the Go App side for OCR processing. If resource usage is too high when importing multiple large PDFs,
try lowering `DOCREADER_PDF_RENDER_MAX_WORKERS` or `DOCREADER_MARKITDOWN_MAX_WORKERS` first.

## Configuration Examples

### Basic Configuration (using MinIO)

```yaml
docreader:
  environment:
    - MINIO_ENDPOINT=minio:9000
    - MINIO_PUBLIC_ENDPOINT=http://localhost:9000
    - MAX_FILE_SIZE_MB=50
```

### Advanced Configuration (enabling MinerU)

```yaml
docreader:
  environment:
    - MINIO_ENDPOINT=minio:9000
    - MINIO_PUBLIC_ENDPOINT=http://192.168.1.100:9000
    - MINERU_ENDPOINT=http://mineru:8080
    - MAX_FILE_SIZE_MB=100
```

### Using Tencent Cloud COS

```yaml
docreader:
  environment:
    - STORAGE_TYPE=cos
    - COS_SECRET_ID=your_secret_id
    - COS_SECRET_KEY=your_secret_key
    - COS_REGION=ap-guangzhou
    - COS_BUCKET_NAME=your-bucket
    - COS_APP_ID=your_app_id
    - MAX_FILE_SIZE_MB=50
```

### Using Alibaba Cloud OSS

```yaml
docreader:
  environment:
    - STORAGE_TYPE=oss
    - OSS_ACCESS_KEY_ID=your_access_key_id
    - OSS_ACCESS_KEY_SECRET=your_access_key_secret
    - OSS_ENDPOINT=oss-cn-hangzhou.aliyuncs.com
    - OSS_BUCKET_NAME=your-bucket
    - OSS_REGION=cn-hangzhou
    - MAX_FILE_SIZE_MB=50
```

## FAQ

### 1. DocReader service fails to start?

Check the container logs for missing dependencies or permission-related errors, and if necessary confirm that `MINIO_ENDPOINT` and related storage environment variables are configured correctly.

### 2. Images not displaying?

Check the `MINIO_PUBLIC_ENDPOINT` configuration:
- Make sure the address used is accessible from the browser
- If accessing from another device, don't use `localhost` — use the actual IP address instead

### 3. File upload failing?

Check the `MAX_FILE_SIZE_MB` configuration and make sure the limit is large enough. Also make sure the file size limits on the frontend and backend services are consistent.

## Service Health Check

The DocReader service is configured with a health check:

```yaml
healthcheck:
  test: ["CMD", "grpc_health_probe", "-addr=localhost:50051"]
  interval: 30s
  timeout: 10s
  retries: 3
  start_period: 60s
```

You can check the service status with the following commands:

```bash
docker ps | grep docreader
docker logs WeKnora-docreader
```

## More Information

- Service port: 50051 (gRPC)
- Container name: WeKnora-docreader
- Network: WeKnora-network
- Restart policy: unless-stopped
