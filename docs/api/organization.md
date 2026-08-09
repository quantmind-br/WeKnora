# Organization Management API

[Back to index](./README.md)

An Organization (also called a "Space") is WeKnora's multi-space collaboration unit. A user can create/join multiple organizations, participating with the role of owner / admin / editor / viewer; knowledge bases and agents can be shared within an organization, and organization members are granted corresponding access based on their role.

This page covers the following six categories of endpoints:

- Organization management: organization CRUD, invite codes, search, joining and leaving
- Member management: member list, role changes, removal, invitations
- Join requests: request list, review
- Knowledge base sharing: share a knowledge base to an organization / unshare / change permission
- Agent sharing: share an agent to an organization / unshare
- My shared view: all shared knowledge bases / agents accessible to the current user

General notes:
- All paths are prefixed with `/api/v1`
- Auth header: `X-API-Key: sk-xxxxx` (or `Authorization: Bearer ...`)
- Error responses uniformly take the form `{ "success": false, "error": "..." }`; HTTP status codes follow RESTful semantics
- Role (`OrgMemberRole`) values: `owner` / `admin` / `editor` / `viewer`
- Share permission (`permission`) values: `viewer` / `editor` (typically only these two are allowed on creation)

## Route overview

### Organization management

| Method | Path                                          | Description                                |
| ------ | --------------------------------------------- | -------------------------------------------- |
| POST   | `/organizations`                              | Create an organization                      |
| GET    | `/organizations`                              | Get my organization list (with resource counts) |
| GET    | `/organizations/preview/:code`                | Preview an organization via invite code (without joining) |
| POST   | `/organizations/join`                         | Join an organization via invite code        |
| POST   | `/organizations/join-request`                 | Submit a join request (for organizations requiring review) |
| GET    | `/organizations/search`                       | Search for joinable, searchable organizations |
| POST   | `/organizations/join-by-id`                   | Join a searchable organization via organization ID |
| GET    | `/organizations/:id`                          | Get organization details                    |
| PUT    | `/organizations/:id`                          | Update an organization                      |
| DELETE | `/organizations/:id`                          | Delete an organization                      |
| POST   | `/organizations/:id/leave`                    | Leave an organization                       |
| POST   | `/organizations/:id/request-upgrade`          | Existing member requests a role upgrade     |
| POST   | `/organizations/:id/invite-code`              | Regenerate the invite code                  |

### Member management

| Method | Path                                          | Description                            |
| ------ | --------------------------------------------- | ---------------------------------------- |
| GET    | `/organizations/:id/search-users`             | Search for invitable users (admin only) |
| POST   | `/organizations/:id/invite`                   | Directly add a user as a member (admin only) |
| GET    | `/organizations/:id/members`                  | Get the member list                     |
| PUT    | `/organizations/:id/members/:user_id`         | Update a member's role                  |
| DELETE | `/organizations/:id/members/:user_id`         | Remove a member                         |

### Join requests

| Method | Path                                                    | Description                     |
| ------ | -------------------------------------------------------- | --------------------------------- |
| GET    | `/organizations/:id/join-requests`                      | Get the list of pending join requests |
| PUT    | `/organizations/:id/join-requests/:request_id/review`   | Review a join request (admin only) |

### Knowledge base sharing

| Method | Path                                          | Description                                    |
| ------ | --------------------------------------------- | -------------------------------------------------- |
| POST   | `/knowledge-bases/:id/shares`                 | Share a knowledge base to an organization         |
| GET    | `/knowledge-bases/:id/shares`                 | Get the share list for this knowledge base        |
| PUT    | `/knowledge-bases/:id/shares/:share_id`       | Update the share permission                       |
| DELETE | `/knowledge-bases/:id/shares/:share_id`       | Unshare                                           |
| GET    | `/organizations/:id/shares`                   | Get the list of knowledge bases shared into an organization |
| GET    | `/organizations/:id/shared-knowledge-bases`   | All knowledge bases in an organization (including ones I shared, space view) |

### Agent sharing

| Method | Path                                          | Description                                    |
| ------ | --------------------------------------------- | -------------------------------------------------- |
| POST   | `/agents/:id/shares`                          | Share an agent to an organization                 |
| GET    | `/agents/:id/shares`                          | Get the share list for this agent                 |
| DELETE | `/agents/:id/shares/:share_id`                | Unshare                                           |
| GET    | `/organizations/:id/agent-shares`             | Get the list of agents shared into an organization |
| GET    | `/organizations/:id/shared-agents`            | All agents in an organization (including ones I shared, space view) |

### My shared view

| Method | Path                          | Description                                    |
| ------ | ------------------------------ | -------------------------------------------------- |
| GET    | `/shared-knowledge-bases`     | Get all knowledge bases shared with me (cross-organization) |
| GET    | `/shared-agents`              | Get all agents shared with me (cross-organization) |

---

## Organization management

### POST `/organizations` - Create an organization

**Request body**:

| Field                        | Type    | Required | Description                                        |
| ---------------------------- | ------- | -------- | --------------------------------------------------- |
| name                        | string  | Yes  | Organization name (1-255 characters)                    |
| description                 | string  | No   | Organization description (up to 1000 characters)        |
| avatar                      | string  | No   | Avatar URL (up to 512 characters)                        |
| invite_code_validity_days   | int     | No   | Invite code validity in days: `0`=permanent, `1` / `7` / `30`, default 7 |
| member_limit                | int     | No   | Member limit, `0`=unlimited, default 50               |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/organizations' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "name": "AI Tech Team",
    "description": "Focused on AI technology research and knowledge management",
    "invite_code_validity_days": 7,
    "member_limit": 50
}'
```

**Response** (`201 Created`):

```json
{
    "data": {
        "id": "org-00000001",
        "name": "AI Tech Team",
        "description": "Focused on AI technology research and knowledge management",
        "avatar": "",
        "owner_id": "user-00000001",
        "invite_code": "",
        "invite_code_validity_days": 7,
        "require_approval": false,
        "searchable": false,
        "member_limit": 50,
        "member_count": 1,
        "share_count": 0,
        "agent_share_count": 0,
        "pending_join_request_count": 0,
        "is_owner": true,
        "my_role": "owner",
        "has_pending_upgrade": false,
        "created_at": "2025-08-12T10:00:00+08:00",
        "updated_at": "2025-08-12T10:00:00+08:00"
    },
    "success": true
}
```

### GET `/organizations` - Get my organization list

Returns all organizations the current user belongs to; the `resource_counts` field includes the knowledge base count and agent count for each space, for the list page sidebar to render directly.

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/organizations' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**:

```json
{
    "data": {
        "organizations": [
            {
                "id": "org-00000001",
                "name": "AI Tech Team",
                "description": "Focused on AI technology research and knowledge management",
                "owner_id": "user-00000001",
                "invite_code": "ABC123XY",
                "invite_code_expires_at": "2025-08-19T10:00:00+08:00",
                "invite_code_validity_days": 7,
                "require_approval": false,
                "searchable": false,
                "member_limit": 50,
                "member_count": 3,
                "share_count": 2,
                "agent_share_count": 1,
                "pending_join_request_count": 0,
                "is_owner": true,
                "my_role": "owner",
                "has_pending_upgrade": false,
                "created_at": "2025-08-12T10:00:00+08:00",
                "updated_at": "2025-08-12T10:00:00+08:00"
            }
        ],
        "total": 1,
        "resource_counts": {
            "knowledge_bases": { "by_organization": { "org-00000001": 5 } },
            "agents":         { "by_organization": { "org-00000001": 2 } }
        }
    },
    "success": true
}
```

### GET `/organizations/preview/:code` - Preview an organization via invite code

No need to become a member beforehand; can be used for a "preview before joining" page. `:code` is the invite code string.

**Path parameters**:

| Field | Type   | Description   |
| ---- | ------ | ------ |
| code | string | Invite code |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/organizations/preview/ABC123XY' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**:

```json
{
    "data": {
        "id": "org-00000001",
        "name": "AI Tech Team",
        "description": "Focused on AI technology research and knowledge management",
        "avatar": "",
        "member_count": 3,
        "share_count": 2,
        "agent_share_count": 1,
        "is_already_member": false,
        "require_approval": true,
        "created_at": "2025-08-12T10:00:00+08:00"
    },
    "success": true
}
```

### POST `/organizations/join` - Join an organization via invite code

Only applies to organizations with `require_approval: false`. For organizations requiring review, use `/organizations/join-request` instead.

**Request body**:

| Field        | Type   | Required | Description                  |
| ----------- | ------ | ---- | --------------------- |
| invite_code | string | Yes   | Invite code (8-32 characters)   |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/organizations/join' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "invite_code": "ABC123XY"
}'
```

**Response**: Returns the organization information after joining (same structure as `GET /organizations/:id`).

### POST `/organizations/join-request` - Submit a join request

Used when the organization has review enabled (`require_approval: true`).

**Request body**:

| Field        | Type   | Required | Description                                        |
| ----------- | ------ | ---- | ------------------------------------------- |
| invite_code | string | Yes   | Invite code (8-32 characters)                         |
| message     | string | No   | Request message (up to 500 characters)                   |
| role        | string | No   | Desired role: `viewer` / `editor` / `admin`     |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/organizations/join-request' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "invite_code": "ABC123XY",
    "message": "Would like to join the team to help build out the knowledge base",
    "role": "editor"
}'
```

**Response**:

```json
{
    "data": {
        "id": "jr-00000001",
        "user_id": "user-00000002",
        "request_type": "join",
        "requested_role": "editor",
        "status": "pending",
        "created_at": "2025-08-14T10:00:00+08:00"
    },
    "success": true
}
```

### GET `/organizations/search` - Search for joinable organizations

Returns organizations with `searchable: true`. Only returns metadata, never the invite code.

**Query parameters**:

| Field  | Type   | Required | Description                              |
| ----- | ------ | ---- | ---------------------------------- |
| q     | string | No   | Search keyword (fuzzy match on name or description)   |
| limit | int    | No   | Number of results to return (1-100, default 20)         |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/organizations/search?q=AI&limit=10' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**:

```json
{
    "data": [
        {
            "id": "org-00000001",
            "name": "AI Tech Team",
            "description": "Focused on AI technology research and knowledge management",
            "avatar": "",
            "member_count": 3,
            "member_limit": 50,
            "share_count": 2,
            "agent_share_count": 1,
            "is_already_member": false,
            "require_approval": true
        }
    ],
    "total": 1,
    "success": true
}
```

### POST `/organizations/join-by-id` - Join via organization ID

Used for the "search for a joinable space" flow, without needing an invite code; the target organization must have `searchable: true`. If the organization has review enabled, a join request is created; otherwise the user joins directly.

**Request body**:

| Field             | Type   | Required | Description                                       |
| ---------------- | ------ | ---- | ------------------------------------------- |
| organization_id  | string | Yes   | Target organization ID                                |
| message          | string | No   | Request message (up to 500 characters)                  |
| role             | string | No   | Desired role: `viewer` / `editor` / `admin`    |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/organizations/join-by-id' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "organization_id": "org-00000001",
    "message": "Would like to join your team",
    "role": "viewer"
}'
```

**Response**: Returns the organization information after joining (same structure as `GET /organizations/:id`).

### GET `/organizations/:id` - Get organization details

**Path parameters**:

| Field | Type   | Description    |
| ---- | ------ | ------- |
| id   | string | Organization ID |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/organizations/org-00000001' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**:

```json
{
    "data": {
        "id": "org-00000001",
        "name": "AI Tech Team",
        "description": "Focused on AI technology research and knowledge management",
        "avatar": "",
        "owner_id": "user-00000001",
        "invite_code": "ABC123XY",
        "invite_code_expires_at": "2025-08-19T10:00:00+08:00",
        "invite_code_validity_days": 7,
        "require_approval": false,
        "searchable": true,
        "member_limit": 50,
        "member_count": 3,
        "share_count": 2,
        "agent_share_count": 1,
        "pending_join_request_count": 1,
        "is_owner": true,
        "my_role": "owner",
        "has_pending_upgrade": false,
        "created_at": "2025-08-12T10:00:00+08:00",
        "updated_at": "2025-08-12T10:00:00+08:00"
    },
    "success": true
}
```

`invite_code` and `invite_code_expires_at` are only returned when the current user is the owner or an admin.

### PUT `/organizations/:id` - Update an organization

**Request body** (all fields are optional; passing `null` is equivalent to not updating):

| Field                        | Type    | Description                                          |
| ---------------------------- | ------- | --------------------------------------------- |
| name                        | string  | Organization name (1-255 characters)                        |
| description                 | string  | Organization description (up to 1000 characters)                    |
| avatar                      | string  | Avatar URL (up to 512 characters)                     |
| require_approval            | bool    | Whether joining requires review                              |
| searchable                  | bool    | Whether it can be discovered in `/organizations/search`     |
| invite_code_validity_days   | int     | Invite code validity in days (0=permanent, 1/7/30)              |
| member_limit                | int     | Member limit (0=unlimited)                            |

**Request**:

```curl
curl --location --request PUT 'http://localhost:8080/api/v1/organizations/org-00000001' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "description": "Focused on AI technology research and knowledge management (updated)",
    "require_approval": true,
    "searchable": true
}'
```

**Response**: Returns the updated organization information (same structure as `GET /organizations/:id`).

### DELETE `/organizations/:id` - Delete an organization

Only the owner can delete; deletion also cleans up membership relationships and share records.

**Request**:

```curl
curl --location --request DELETE 'http://localhost:8080/api/v1/organizations/org-00000001' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**:

```json
{ "success": true }
```

### POST `/organizations/:id/leave` - Leave an organization

The owner cannot leave their own organization; they must transfer ownership or delete the organization first.

**Request**:

```curl
curl --location --request POST 'http://localhost:8080/api/v1/organizations/org-00000001/leave' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**:

```json
{ "success": true, "message": "Left organization successfully" }
```

### POST `/organizations/:id/request-upgrade` - Request a role upgrade

Initiated by an existing member, awaiting admin review (appears in `/organizations/:id/join-requests` as `request_type: "upgrade"`).

**Request body**:

| Field           | Type   | Required | Description                                                 |
| -------------- | ------ | ---- | ---------------------------------------------------- |
| requested_role | string | Yes   | Desired role: `viewer` / `editor` / `admin`              |
| message        | string | No   | Reason for the request (up to 500 characters)                            |

**Request**:

```curl
curl --location --request POST 'http://localhost:8080/api/v1/organizations/org-00000001/request-upgrade' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "requested_role": "admin",
    "message": "Need admin permissions to manage knowledge base sharing"
}'
```

**Response**:

```json
{
    "data": {
        "id": "jr-00000002",
        "request_type": "upgrade",
        "prev_role": "editor",
        "requested_role": "admin",
        "status": "pending",
        "created_at": "2025-08-14T11:00:00+08:00"
    },
    "success": true
}
```

### POST `/organizations/:id/invite-code` - Regenerate the invite code

Only owner / admin can call this; it invalidates the old invite code.

**Request**:

```curl
curl --location --request POST 'http://localhost:8080/api/v1/organizations/org-00000001/invite-code' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**:

```json
{
    "data": { "invite_code": "NEW1CODE" },
    "success": true
}
```

---

## Member management

### GET `/organizations/:id/search-users` - Search for invitable users

Only owner / admin can call this. Matches against username or email, automatically excluding users already in the organization.

**Query parameters**:

| Field  | Type   | Required | Description                          |
| ----- | ------ | ---- | ----------------------------- |
| q     | string | Yes   | Keyword (username or email)         |
| limit | int    | No   | Maximum number of results, default 10          |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/organizations/org-00000001/search-users?q=zhang&limit=10' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**:

```json
{
    "data": [
        {
            "id": "user-00000002",
            "username": "zhangsan",
            "email": "zhangsan@example.com",
            "avatar": ""
        }
    ],
    "success": true
}
```

### POST `/organizations/:id/invite` - Directly invite a user

Only owner / admin can call this; the invited user becomes a member directly (no review needed).

**Request body**:

| Field    | Type   | Required | Description                                        |
| ------- | ------ | ---- | ------------------------------------------- |
| user_id | string | Yes   | ID of the invited user                             |
| role    | string | Yes   | Role: `viewer` / `editor` / `admin`         |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/organizations/org-00000001/invite' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "user_id": "user-00000002",
    "role": "editor"
}'
```

**Response**:

```json
{ "success": true, "message": "Member added successfully" }
```

### GET `/organizations/:id/members` - Get the member list

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/organizations/org-00000001/members' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**:

```json
{
    "data": {
        "members": [
            {
                "id": "mem-00000001",
                "user_id": "user-00000001",
                "username": "admin",
                "email": "admin@example.com",
                "avatar": "",
                "role": "owner",
                "tenant_id": 1,
                "joined_at": "2025-08-12T10:00:00+08:00"
            },
            {
                "id": "mem-00000002",
                "user_id": "user-00000002",
                "username": "zhangsan",
                "email": "zhangsan@example.com",
                "avatar": "",
                "role": "editor",
                "tenant_id": 2,
                "joined_at": "2025-08-13T09:00:00+08:00"
            }
        ],
        "total": 2
    },
    "success": true
}
```

### PUT `/organizations/:id/members/:user_id` - Update a member's role

**Path parameters**:

| Field     | Type   | Description           |
| -------- | ------ | -------------- |
| id       | string | Organization ID        |
| user_id  | string | Target member's user ID |

**Request body**:

| Field | Type   | Required | Description                                |
| ---- | ------ | ---- | ----------------------------------- |
| role | string | Yes   | `viewer` / `editor` / `admin`       |

**Request**:

```curl
curl --location --request PUT 'http://localhost:8080/api/v1/organizations/org-00000001/members/user-00000002' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{ "role": "admin" }'
```

**Response**:

```json
{ "success": true }
```

### DELETE `/organizations/:id/members/:user_id` - Remove a member

Only owner / admin can call this; the owner cannot be removed.

**Request**:

```curl
curl --location --request DELETE 'http://localhost:8080/api/v1/organizations/org-00000001/members/user-00000002' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**:

```json
{ "success": true }
```

---

## Join requests

### GET `/organizations/:id/join-requests` - Get the list of pending requests

Only owner / admin can call this. Only returns records with `status: pending`; `request_type` distinguishes new join requests (`join`) from upgrade requests (`upgrade`).

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/organizations/org-00000001/join-requests' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**:

```json
{
    "data": {
        "requests": [
            {
                "id": "jr-00000001",
                "user_id": "user-00000003",
                "username": "zhangwei",
                "email": "zhangwei@example.com",
                "message": "Would like to join the team to help build out the knowledge base",
                "request_type": "join",
                "prev_role": "",
                "requested_role": "editor",
                "status": "pending",
                "created_at": "2025-08-14T10:00:00+08:00"
            },
            {
                "id": "jr-00000002",
                "user_id": "user-00000002",
                "username": "zhangsan",
                "email": "zhangsan@example.com",
                "message": "Need admin permissions to manage knowledge base sharing",
                "request_type": "upgrade",
                "prev_role": "editor",
                "requested_role": "admin",
                "status": "pending",
                "created_at": "2025-08-14T11:00:00+08:00"
            }
        ],
        "total": 2
    },
    "success": true
}
```

### PUT `/organizations/:id/join-requests/:request_id/review` - Review a join request

Only owner / admin can call this.

**Path parameters**:

| Field       | Type   | Description     |
| ---------- | ------ | -------- |
| id         | string | Organization ID  |
| request_id | string | Request ID  |

**Request body**:

| Field     | Type   | Required | Description                                                                  |
| -------- | ------ | ---- | --------------------------------------------------------------------- |
| approved | bool   | Yes   | Whether to approve                                                              |
| message  | string | No   | Review message (up to 500 characters)                                             |
| role     | string | No   | Role to force-assign upon approval (defaults to the role requested by the applicant), only `viewer/editor/admin` |

**Request**:

```curl
curl --location --request PUT 'http://localhost:8080/api/v1/organizations/org-00000001/join-requests/jr-00000001/review' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "approved": true,
    "message": "Welcome aboard",
    "role": "editor"
}'
```

**Response**:

```json
{ "success": true, "message": "Review completed" }
```

---

## Knowledge base sharing

### POST `/knowledge-bases/:id/shares` - Share a knowledge base to an organization

**Path parameters**: `id` = knowledge base ID

**Request body**:

| Field             | Type   | Required | Description                            |
| ---------------- | ------ | ---- | -------------------------------- |
| organization_id  | string | Yes   | Target organization ID                     |
| permission       | string | Yes   | Share permission: `viewer` / `editor`   |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/shares' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "organization_id": "org-00000001",
    "permission": "viewer"
}'
```

**Response** (`201 Created`):

```json
{
    "data": {
        "id": "kbs-00000001",
        "knowledge_base_id": "kb-00000001",
        "organization_id": "org-00000001",
        "shared_by_user_id": "user-00000001",
        "source_tenant_id": 1,
        "permission": "viewer",
        "created_at": "2025-08-15T10:00:00+08:00"
    },
    "success": true
}
```

### GET `/knowledge-bases/:id/shares` - Get the knowledge base share list

Returns all organizations this knowledge base has been shared to.

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/shares' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**:

```json
{
    "data": {
        "shares": [
            {
                "id": "kbs-00000001",
                "knowledge_base_id": "kb-00000001",
                "knowledge_base_name": "Technical Documentation Library",
                "knowledge_base_type": "document",
                "knowledge_count": 12,
                "chunk_count": 0,
                "organization_id": "org-00000001",
                "organization_name": "AI Tech Team",
                "shared_by_user_id": "user-00000001",
                "shared_by_username": "admin",
                "source_tenant_id": 1,
                "permission": "viewer",
                "my_role_in_org": "owner",
                "my_permission": "viewer",
                "created_at": "2025-08-15T10:00:00+08:00"
            }
        ],
        "total": 1
    },
    "success": true
}
```

### PUT `/knowledge-bases/:id/shares/:share_id` - Update the share permission

**Path parameters**:

| Field     | Type   | Description           |
| -------- | ------ | -------------- |
| id       | string | Knowledge base ID      |
| share_id | string | Share record ID    |

**Request body**:

| Field       | Type   | Required | Description                          |
| ---------- | ------ | ---- | ----------------------------- |
| permission | string | Yes   | New permission: `viewer` / `editor`   |

**Request**:

```curl
curl --location --request PUT 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/shares/kbs-00000001' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{ "permission": "editor" }'
```

**Response**:

```json
{ "success": true }
```

### DELETE `/knowledge-bases/:id/shares/:share_id` - Unshare a knowledge base

**Request**:

```curl
curl --location --request DELETE 'http://localhost:8080/api/v1/knowledge-bases/kb-00000001/shares/kbs-00000001' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**:

```json
{ "success": true }
```

### GET `/organizations/:id/shares` - Get knowledge bases shared into an organization

Only visible to members of this organization. `my_permission = min(permission, my_role_in_org)`, i.e. the current user's effective permission.

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/organizations/org-00000001/shares' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**: Same structure as `GET /knowledge-bases/:id/shares`.

### GET `/organizations/:id/shared-knowledge-bases` - Organization knowledge base space view

Used by the knowledge base list page after "switching to a space." Returns all knowledge bases visible to the current user within this organization, including:

1. Knowledge bases shared directly via `POST /knowledge-bases/:id/shares`
2. Knowledge bases brought in via a shared agent (`agents/:id/shares`) (read-only; the `source_from_agent` field identifies the origin)
3. Knowledge bases the current user has shared themselves (`is_mine: true`)

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/organizations/org-00000001/shared-knowledge-bases' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**:

```json
{
    "data": [
        {
            "knowledge_base": {
                "id": "kb-00000001",
                "name": "Technical Documentation Library",
                "type": "document"
            },
            "share_id": "kbs-00000001",
            "organization_id": "org-00000001",
            "org_name": "AI Tech Team",
            "permission": "viewer",
            "source_tenant_id": 1,
            "shared_at": "2025-08-15T10:00:00+08:00",
            "is_mine": false,
            "source_from_agent": {
                "agent_id": "agent-00000005",
                "agent_name": "Smart Customer Service Assistant",
                "kb_selection_mode": "selected"
            }
        }
    ],
    "total": 1,
    "success": true
}
```

`source_from_agent` only appears when the KB was introduced via a shared agent; directly shared KBs do not include this field.

---

## Agent sharing

### POST `/agents/:id/shares` - Share an agent to an organization

Only allowed when the current user has the editor or admin role in the target organization; the agent must already be fully configured (a required chat model, and if the knowledge_search tool is enabled, a configured rerank model, etc.).

**Path parameters**: `id` = agent ID

**Request body**:

| Field             | Type   | Required | Description                            |
| ---------------- | ------ | ---- | -------------------------------- |
| organization_id  | string | Yes   | Target organization ID                     |
| permission       | string | Yes   | Share permission: `viewer` / `editor`   |

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/agents/agent-00000001/shares' \
--header 'X-API-Key: sk-xxxxx' \
--header 'Content-Type: application/json' \
--data '{
    "organization_id": "org-00000001",
    "permission": "viewer"
}'
```

**Response** (`201 Created`):

```json
{
    "data": {
        "id": "as-00000001",
        "agent_id": "agent-00000001",
        "organization_id": "org-00000001",
        "shared_by_user_id": "user-00000001",
        "source_tenant_id": 1,
        "permission": "viewer",
        "created_at": "2025-08-15T11:00:00+08:00"
    },
    "success": true
}
```

### GET `/agents/:id/shares` - Get the agent share list

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/agents/agent-00000001/shares' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**:

```json
{
    "data": {
        "shares": [
            {
                "id": "as-00000001",
                "agent_id": "agent-00000001",
                "organization_id": "org-00000001",
                "organization_name": "AI Tech Team",
                "shared_by_user_id": "user-00000001",
                "source_tenant_id": 1,
                "permission": "viewer",
                "created_at": "2025-08-15T11:00:00+08:00"
            }
        ],
        "total": 1
    },
    "success": true
}
```

### DELETE `/agents/:id/shares/:share_id` - Unshare an agent

Only the sharer themselves or an admin with the relevant permissions can unshare.

**Request**:

```curl
curl --location --request DELETE 'http://localhost:8080/api/v1/agents/agent-00000001/shares/as-00000001' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**:

```json
{ "success": true, "message": "Share removed successfully" }
```

### GET `/organizations/:id/agent-shares` - Get agents shared into an organization

Only visible to members of this organization. The response structure adds `my_role_in_org` / `my_permission` on top of `AgentShareResponse`, along with a summary of the agent's capability scope (`scope_kb` / `scope_web_search` / `scope_mcp`, etc.).

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/organizations/org-00000001/agent-shares' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**:

```json
{
    "data": {
        "shares": [
            {
                "id": "as-00000001",
                "agent_id": "agent-00000001",
                "agent_name": "Smart Customer Service Assistant",
                "agent_avatar": "🤖",
                "organization_id": "org-00000001",
                "organization_name": "AI Tech Team",
                "shared_by_user_id": "user-00000001",
                "shared_by_username": "admin",
                "source_tenant_id": 1,
                "permission": "viewer",
                "my_role_in_org": "editor",
                "my_permission": "viewer",
                "created_at": "2025-08-15T11:00:00+08:00",
                "scope_kb": "selected",
                "scope_kb_count": 2,
                "scope_web_search": true,
                "scope_mcp": "none"
            }
        ],
        "total": 1
    },
    "success": true
}
```

### GET `/organizations/:id/shared-agents` - Organization agent space view

Used by the agent list page after "switching to a space." Returns all agents visible to the current user within this organization, including both those shared by others and shared by the current user (`is_mine: true`).

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/organizations/org-00000001/shared-agents' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**:

```json
{
    "data": [
        {
            "agent": {
                "id": "agent-00000001",
                "name": "Smart Customer Service Assistant"
            },
            "share_id": "as-00000001",
            "organization_id": "org-00000001",
            "org_name": "AI Tech Team",
            "permission": "viewer",
            "source_tenant_id": 1,
            "shared_at": "2025-08-15T11:00:00+08:00",
            "shared_by_user_id": "user-00000001",
            "shared_by_username": "admin",
            "disabled_by_me": false,
            "is_mine": false
        }
    ],
    "total": 1,
    "success": true
}
```

---

## My shared view

### GET `/shared-knowledge-bases` - Get knowledge bases shared with me (cross-organization)

Returns the list of knowledge bases the current user has been granted via sharing across all organizations (not restricted to a specific `:id`), for use by the "All knowledge bases" view.

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/shared-knowledge-bases' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**:

```json
{
    "data": [
        {
            "knowledge_base": {
                "id": "kb-00000001",
                "name": "Technical Documentation Library"
            },
            "share_id": "kbs-00000001",
            "organization_id": "org-00000001",
            "org_name": "AI Tech Team",
            "permission": "viewer",
            "source_tenant_id": 1,
            "shared_at": "2025-08-15T10:00:00+08:00"
        }
    ],
    "total": 1,
    "success": true
}
```

### GET `/shared-agents` - Get agents shared with me (cross-organization)

Returns the list of agents the current user has been granted via sharing across all organizations, for use by the "All agents" view. `disabled_by_me: true` indicates that in the current space, this agent has already been hidden from the chat dropdown.

**Request**:

```curl
curl --location 'http://localhost:8080/api/v1/shared-agents' \
--header 'X-API-Key: sk-xxxxx'
```

**Response**:

```json
{
    "data": [
        {
            "agent": {
                "id": "agent-00000001",
                "name": "Smart Customer Service Assistant"
            },
            "share_id": "as-00000001",
            "organization_id": "org-00000001",
            "org_name": "AI Tech Team",
            "permission": "viewer",
            "source_tenant_id": 1,
            "shared_at": "2025-08-15T11:00:00+08:00",
            "shared_by_user_id": "user-00000001",
            "shared_by_username": "admin",
            "disabled_by_me": false
        }
    ],
    "total": 1,
    "success": true
}
```
