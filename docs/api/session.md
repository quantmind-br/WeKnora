# Session Management API

[Back to Table of Contents](./README.md)

A Session is a pure conversation container that stores only basic information (title, description, pinned status, etc.). All configuration related to knowledge bases, models, and retrieval strategies is now provided by the Custom Agent at query time, and is no longer stored in the session.

| Method | Path                                       | Description                    |
| ------ | ------------------------------------------ | ------------------------------- |
| POST   | `/sessions`                                | Create a session                |
| DELETE | `/sessions/batch`                          | Batch delete sessions           |
| GET    | `/sessions/:id`                            | Get session details             |
| GET    | `/sessions`                                | Get the session list for the current space |
| PUT    | `/sessions/:id`                            | Update a session                |
| DELETE | `/sessions/:id`                            | Delete a session                |
| DELETE | `/sessions/:id/messages`                   | Clear session messages          |
| POST   | `/sessions/:session_id/generate_title`     | Generate a session title        |
| POST   | `/sessions/:session_id/stop`               | Stop generation                 |
| POST   | `/sessions/:session_id/pin`                | Pin a session                   |
| DELETE | `/sessions/:id/pin`                        | Unpin a session                 |
| GET    | `/sessions/continue-stream/:session_id`    | Resume an unfinished streaming response |

> **Route naming note**: The pin endpoints' POST and DELETE use different path parameter names (POST uses `:session_id`, DELETE uses `:id`). This is because the gin router maintains a separate radix tree for each HTTP method, and the wildcard names in the existing trees differ; they must be kept as-is to avoid a `wildcard conflicts` panic at registration time. Semantically, both refer to the session ID.

## POST `/sessions` - Create a session

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/sessions' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "title": "My new conversation",
    "description": "A discussion about AI"
}'
```

**Request parameters**:

| Field         | Type   | Required | Description       |
| ------------- | ------ | -------- | ------------------ |
| `title`       | string | No       | Session title       |
| `description` | string | No       | Session description |

**Response**:

```json
{
    "success": true,
    "data": {
        "id": "411d6b70-9a85-4d03-bb74-aab0fd8bd12f",
        "title": "My new conversation",
        "description": "A discussion about AI",
        "tenant_id": 1,
        "user_id": "u-001",
        "is_pinned": false,
        "created_at": "2026-03-27T12:26:19.611616+08:00",
        "updated_at": "2026-03-27T12:26:19.611616+08:00",
        "deleted_at": null
    }
}
```

> When calling via an API key, `user_id` may be empty, in which case the session is visible at the space level.

## DELETE `/sessions/batch` - Batch delete sessions

Two modes are supported: batch delete by a list of IDs, or delete all sessions in the current space.

**Request - Delete by ID list**:

```curl
curl --location --request DELETE 'http://localhost:8080/api/v1/sessions/batch' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "ids": [
        "411d6b70-9a85-4d03-bb74-aab0fd8bd12f",
        "ceb9babb-1e30-41d7-817d-fd584954304b"
    ]
}'
```

**Request - Delete all sessions**:

```curl
curl --location --request DELETE 'http://localhost:8080/api/v1/sessions/batch' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "delete_all": true
}'
```

**Request parameters**:

| Field        | Type     | Required | Description                                                       |
| ------------ | -------- | -------- | ------------------------------------------------------------------ |
| `ids`        | string[] | No       | List of session IDs to delete (required when `delete_all` is `false`) |
| `delete_all` | bool     | No       | When set to `true`, deletes all sessions in the current space, ignoring the `ids` field |

**Response**:

```json
{
    "success": true,
    "message": "Sessions deleted successfully"
}
```

When `delete_all=true`, the `message` is `"All sessions deleted successfully"`.

## GET `/sessions/:id` - Get session details

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/sessions/ceb9babb-1e30-41d7-817d-fd584954304b' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json'
```

**Path parameters**:

| Field | Type   | Required | Description |
| ----- | ------ | -------- | ------------ |
| `id`  | string | Yes      | Session ID   |

**Response**:

```json
{
    "success": true,
    "data": {
        "id": "ceb9babb-1e30-41d7-817d-fd584954304b",
        "title": "Model optimization strategy",
        "description": "",
        "tenant_id": 1,
        "user_id": "u-001",
        "is_pinned": true,
        "pinned_at": "2026-04-01T09:12:33.123456+08:00",
        "created_at": "2026-03-27T10:24:38.308596+08:00",
        "updated_at": "2026-03-27T10:25:41.317761+08:00",
        "deleted_at": null
    }
}
```

Returns `404` if the session does not exist.

## GET `/sessions` - Get the session list for the current space

Retrieves the session list for the current space, with support for pagination, keyword search, and filtering by source / Agent.

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/sessions?page=1&page_size=10&keyword=AI&source=web' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json'
```

**Query parameters**:

| Field       | Type   | Required | Description                                                              |
| ----------- | ------ | -------- | -------------------------------------------------------------------------- |
| `page`      | int    | No       | Page number (default 1)                                                    |
| `page_size` | int    | No       | Items per page (default 10)                                                |
| `keyword`   | string | No       | Fuzzy match on title (ILIKE `%keyword%`)                                   |
| `source`    | string | No       | Source filter: `web` (no IM mapping) or an IM platform name, e.g. `feishu`, `wechat`, `slack` |
| `agent_id`  | string | No       | Filter by Agent (only applies to IM sessions)                              |

**Response**:

```json
{
    "success": true,
    "data": [
        {
            "id": "411d6b70-9a85-4d03-bb74-aab0fd8bd12f",
            "title": "My new conversation",
            "description": "",
            "tenant_id": 1,
            "user_id": "u-001",
            "is_pinned": true,
            "pinned_at": "2026-04-01T09:12:33.123456+08:00",
            "created_at": "2026-03-27T12:26:19.611616+08:00",
            "updated_at": "2026-03-27T12:26:19.611616+08:00",
            "deleted_at": null,
            "im_platform": "feishu",
            "im_chat_id": "oc_xxx",
            "im_agent_id": "agent-001"
        }
    ],
    "total": 1,
    "page": 1,
    "page_size": 10
}
```

> List items always include the pinned status field. IM-source-related fields (`im_platform`, `im_chat_id`, `im_thread_id`, `im_user_id`, `im_agent_id`, `im_channel_id`) are only populated for sessions created via IM, and are omitted for Web sessions.

## PUT `/sessions/:id` - Update a session

**Request**:

```curl
curl --location --request PUT 'http://localhost:8080/api/v1/sessions/411d6b70-9a85-4d03-bb74-aab0fd8bd12f' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "title": "WeKnora technical discussion",
    "description": "A discussion about WeKnora's architecture"
}'
```

**Path parameters**:

| Field | Type   | Required | Description |
| ----- | ------ | -------- | ------------ |
| `id`  | string | Yes      | Session ID   |

**Request parameters**:

| Field         | Type   | Required | Description       |
| ------------- | ------ | -------- | ------------------ |
| `title`       | string | No       | Session title       |
| `description` | string | No       | Session description |

**Response**:

```json
{
    "success": true,
    "data": {
        "id": "411d6b70-9a85-4d03-bb74-aab0fd8bd12f",
        "title": "WeKnora technical discussion",
        "description": "A discussion about WeKnora's architecture",
        "tenant_id": 1,
        "user_id": "u-001",
        "is_pinned": false,
        "created_at": "2026-03-27T12:26:19.611616+08:00",
        "updated_at": "2026-03-27T14:20:56.738424+08:00",
        "deleted_at": null
    }
}
```

Returns `404` if the session does not exist.

## DELETE `/sessions/:id` - Delete a session

**Request**:

```curl
curl --location --request DELETE 'http://localhost:8080/api/v1/sessions/411d6b70-9a85-4d03-bb74-aab0fd8bd12f' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json'
```

**Path parameters**:

| Field | Type   | Required | Description |
| ----- | ------ | -------- | ------------ |
| `id`  | string | Yes      | Session ID   |

**Response**:

```json
{
    "success": true,
    "message": "Session deleted successfully"
}
```

## DELETE `/sessions/:id/messages` - Clear session messages

Deletes all messages in the session, and also clears the LLM context and chat-history knowledge base entries. The session itself is retained.

**Request**:

```curl
curl --location --request DELETE 'http://localhost:8080/api/v1/sessions/ceb9babb-1e30-41d7-817d-fd584954304b/messages' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json'
```

**Path parameters**:

| Field | Type   | Required | Description |
| ----- | ------ | -------- | ------------ |
| `id`  | string | Yes      | Session ID   |

**Response**:

```json
{
    "success": true,
    "message": "Session messages cleared successfully"
}
```

## POST `/sessions/:session_id/generate_title` - Generate a session title

Automatically generates a session title based on message content.

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/sessions/ceb9babb-1e30-41d7-817d-fd584954304b/generate_title' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
  "messages": [
    {
      "role": "user",
      "content": "Hi, I'd like to learn about artificial intelligence"
    },
    {
      "role": "assistant",
      "content": "Artificial intelligence is a branch of computer science..."
    }
  ]
}'
```

**Path parameters**:

| Field        | Type   | Required | Description |
| ------------ | ------ | -------- | ------------ |
| `session_id` | string | Yes      | Session ID   |

**Request parameters**:

| Field      | Type      | Required | Description                                  |
| ---------- | --------- | -------- | ---------------------------------------------- |
| `messages` | Message[] | Yes      | List of messages used as context for title generation |

**Response**:

```json
{
    "success": true,
    "data": "Artificial Intelligence Basics"
}
```

## POST `/sessions/:session_id/stop` - Stop generation

Stops the assistant's currently in-progress reply generation task. The backend appends a `stop` event to the stream, which the active SSE handling goroutine detects and uses to trigger cancellation.

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/sessions/7c966c74-610e-4516-8d5b-05e14b2e4ee0/stop' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "message_id": "ebbf7e53-dfe6-44d5-882f-36a4104910b5"
}'
```

**Path parameters**:

| Field        | Type   | Required | Description |
| ------------ | ------ | -------- | ------------ |
| `session_id` | string | Yes      | Session ID   |

**Request parameters**:

| Field        | Type   | Required | Description                          |
| ------------ | ------ | -------- | -------------------------------------- |
| `message_id` | string | Yes      | ID of the assistant message to stop generating |

**Response**:

```json
{
    "success": true,
    "message": "Generation stopped"
}
```

If the message has already completed (no need to stop):

```json
{
    "success": true,
    "message": "Message already completed"
}
```

> Returns `403` if the message does not belong to the current session; returns `404` if the message or session does not exist.

## POST `/sessions/:session_id/pin` - Pin a session

Pins the specified session (per user).

**Request**:

```curl
curl --location --request POST 'http://localhost:8080/api/v1/sessions/ceb9babb-1e30-41d7-817d-fd584954304b/pin' \
--header 'X-API-Key: sk-xxxxx'
```

**Path parameters**:

| Field        | Type   | Required | Description |
| ------------ | ------ | -------- | ------------ |
| `session_id` | string | Yes      | Session ID   |

**Response**:

```json
{
    "success": true,
    "is_pinned": true
}
```

Returns `404` if the session does not exist or is not visible to the current user.

## DELETE `/sessions/:id/pin` - Unpin a session

Unpins the specified session.

**Request**:

```curl
curl --location --request DELETE 'http://localhost:8080/api/v1/sessions/ceb9babb-1e30-41d7-817d-fd584954304b/pin' \
--header 'X-API-Key: sk-xxxxx'
```

**Path parameters**:

| Field | Type   | Required | Description |
| ----- | ------ | -------- | ------------ |
| `id`  | string | Yes      | Session ID   |

**Response**:

```json
{
    "success": true,
    "is_pinned": false
}
```

> As above, `POST /pin` and `DELETE /pin` use different path parameter names, but both refer to the session ID semantically.

## GET `/sessions/continue-stream/:session_id` - Resume an unfinished streaming response

Used to reconnect to an in-progress streaming response after an SSE connection has been dropped: it first replays all events already produced for that message, then continues pushing subsequent events until `complete`.

**Path parameters**:

| Field        | Type   | Required | Description |
| ------------ | ------ | -------- | ------------ |
| `session_id` | string | Yes      | Session ID   |

**Query parameters**:

| Field        | Type   | Required | Description                                                                       |
| ------------ | ------ | -------- | ------------------------------------------------------------------------------------- |
| `message_id` | string | Yes      | The ID of a message whose `is_completed` is `false`, obtained from the `/messages/:session_id/load` endpoint |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/sessions/continue-stream/ceb9babb-1e30-41d7-817d-fd584954304b?message_id=b8b90eeb-7dd5-4cf9-81c6-5ebcbd759451' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json'
```

**Response format**:

A Server-Sent Events stream, with event structure identical to that returned by `/knowledge-chat/:session_id` and `/agent-chat/:session_id`. Returns `404 No stream events found` if there are currently no stream events for the message; returns `404 Incomplete message not found` if the message record does not exist.
