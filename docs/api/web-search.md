# Web Search API

[Back to Table of Contents](./README.md)

Contains two groups of endpoints:
- `/web-search/providers`: returns **available provider types** (read-only metadata)
- `/web-search-providers/*`: CRUD and connectivity testing for the providers **custom-saved** in the current space

| Method | Path                                  | Description                                |
| ------ | ------------------------------------- | ------------------------------------------- |
| GET    | `/web-search/providers`               | Get the list of web search provider types     |
| GET    | `/web-search-providers/types`         | Get provider type metadata (including parameter definitions) |
| POST   | `/web-search-providers/test`          | Test connectivity using raw credentials (not persisted) |
| POST   | `/web-search-providers`               | Create a space-level provider configuration |
| GET    | `/web-search-providers`               | Get the list of providers saved in the current space |
| GET    | `/web-search-providers/:id`           | Get details of a specific provider          |
| PUT    | `/web-search-providers/:id`           | Update a provider                           |
| DELETE | `/web-search-providers/:id`           | Delete a provider                           |
| POST   | `/web-search-providers/:id/test`      | Test connectivity using saved credentials   |

## GET `/web-search/providers` - Get the list of web search provider types

Retrieves the list of web search providers available in the system (system-level metadata, independent of space).

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/web-search/providers' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json'
```

**Response**:

```json
{
    "data": [
        {
            "name": "google",
            "label": "Google Search",
            "description": "Web search via the Google Custom Search API",
            "enabled": true
        },
        {
            "name": "bing",
            "label": "Bing Search",
            "description": "Web search via the Bing Search API",
            "enabled": true
        },
        {
            "name": "serpapi",
            "label": "SerpAPI",
            "description": "Search engine result scraping via SerpAPI",
            "enabled": false
        }
    ],
    "success": true
}
```

## GET `/web-search-providers/types` - Get provider type metadata

Returns all provider types and their parameter definitions needed by the UI form (which fields each provider requires, their type, and whether they're required).

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/web-search-providers/types' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**:

```json
{
    "data": [
        {
            "provider": "google",
            "label": "Google Search",
            "description": "...",
            "parameter_schema": [
                { "name": "api_key", "label": "API Key", "type": "string", "required": true },
                { "name": "cx", "label": "Search Engine ID", "type": "string", "required": true }
            ]
        }
    ],
    "success": true
}
```

## POST `/web-search-providers/test` - Test connectivity using raw credentials

Used by the "Test Connection" button on the frontend form: runs a sample search using credentials that have not yet been saved.

**Parameters (request body)**:

| Field      | Type   | Required | Description                              |
| ---------- | ------ | -------- | ----------------------------------------- |
| provider   | string | Yes      | Provider type (e.g. `google`, `bing`)     |
| parameters | object | Yes      | Credentials and parameters required by this provider (corresponds to `parameter_schema` in `/types`) |

**Request**:

```curl
curl --location --request POST 'http://localhost:8080/api/v1/web-search-providers/test' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "provider": "google",
    "parameters": {
        "api_key": "AIza...",
        "cx": "0123456789:abcdefg"
    }
}'
```

**Response**:

```json
{ "success": true }
```

On failure:

```json
{ "success": false, "error": "google api: 403 forbidden" }
```

### Zhipu AI configuration

Zhipu uses its own independent Web Search API. `search_engine` and `content_size` are stored in
`parameters.extra_config`; when not specified, they default to `search_std` and `medium` respectively.

```json
{
    "provider": "zhipu",
    "parameters": {
        "api_key": "your-zhipu-api-key",
        "extra_config": {
            "search_engine": "search_std",
            "content_size": "medium"
        }
    }
}
```

`search_engine` supports `search_std`, `search_pro`, `search_pro_sogou`, and
`search_pro_quark`; `content_size` supports `medium` and `high`.

## POST `/web-search-providers` - Create a provider

**Parameters (request body)**:

| Field       | Type    | Required | Description                                       |
| ----------- | ------- | -------- | -------------------------------------------------- |
| name        | string  | Yes      | Provider display name (must be unique and friendly within the space) |
| provider    | string  | Yes      | Provider type (from `/web-search-providers/types`) |
| description | string  | No       | Remarks                                            |
| parameters  | object  | No       | Credentials and parameters                         |
| is_default  | boolean | No       | Whether to set as the default provider for the current space |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/web-search-providers' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "name": "Company Google CSE",
    "provider": "google",
    "description": "For internal network search",
    "parameters": {
        "api_key": "AIza...",
        "cx": "0123456789:abcdefg"
    },
    "is_default": true
}'
```

**Response**:

```json
{
    "data": {
        "id": "wsp-...",
        "tenant_id": 1,
        "name": "Company Google CSE",
        "provider": "google",
        "is_default": true,
        "parameters": { "api_key": "***", "cx": "0123456789:abcdefg" }
    },
    "success": true
}
```

## GET `/web-search-providers` - Get the list of providers

Returns all providers saved in the current space.

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/web-search-providers' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**:

```json
{
    "data": [
        { "id": "wsp-001", "name": "Company Google CSE", "provider": "google", "is_default": true }
    ],
    "success": true
}
```

## GET `/web-search-providers/:id` - Get provider details

**Path parameters**:

| Field | Type   | Description  |
| ----- | ------ | ------------ |
| id    | string | Provider ID  |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/web-search-providers/wsp-001' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**: Same as the create endpoint. A 404 indicates the provider does not exist.

## PUT `/web-search-providers/:id` - Update a provider

**Note**: The `provider` field (type) cannot be changed after creation; only `name` / `description` / `parameters` / `is_default` can be updated.

**Parameters (request body)**: Same as the create endpoint, but without `provider`.

**Request**:

```curl
curl --location --request PUT 'http://localhost:8080/api/v1/web-search-providers/wsp-001' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "name": "Company Google CSE (v2)",
    "parameters": { "api_key": "NEW...", "cx": "0123456789:abcdefg" },
    "is_default": false
}'
```

**Response**: `{ "data": {...updated entity...}, "success": true }`

## DELETE `/web-search-providers/:id` - Delete a provider

**Request**:

```curl
curl --location --request DELETE 'http://localhost:8080/api/v1/web-search-providers/wsp-001' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**: `{ "success": true }`

## POST `/web-search-providers/:id/test` - Test a saved provider

Runs a sample search using the credentials already saved in the database.

**Request**:

```curl
curl --location --request POST 'http://localhost:8080/api/v1/web-search-providers/wsp-001/test' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**: Same as `POST /web-search-providers/test`.
