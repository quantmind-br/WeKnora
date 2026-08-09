---
title: Workspace RBAC Guide
tags: [security authentication, RBAC, permissions, multi-workspace, roles]
aliases: [RBAC, role permissions, workspace roles, TenantRBAC]
source: RBAC-Guide.md
---

# Workspace RBAC Guide

This document describes WeKnora's **Workspace RBAC (in-workspace access control)**, including the role matrix, the resource-ownership model, and how it relates to [Shared Workspaces](./Shared-Spaces-Guide.md).

> Status: released; controlled by the `tenant.enable_rbac` config option, `true` by default (enforced authorization).
> For the full description, rollout plan, schema, route guards, and more, see [`docs/RBAC-Guide.md`](../../RBAC-Guide.md).

## The problem it solves

Before RBAC was introduced, any caller that successfully authenticated via `X-API-Key` or JWT was effectively an admin within the workspace. Once a workspace has more than one real human member, you need to distinguish:

- who can delete knowledge bases or revoke API keys;
- who can edit "their own" KBs / Agents;
- who is read-only.

## Role matrix

| Role | Identifier | Key capabilities |
|------|------|----------|
| Viewer | `viewer` | Read only |
| Contributor | `contributor` | Can modify resources where `creator_id == self`; other people's resources are treated as Viewer |
| Admin | `admin` | Can modify any resource in the workspace; manages members and shared infrastructure |
| Owner | `owner` | Admin + can delete the workspace; every workspace has at least one, and can have more than one |

Hierarchy `viewer < contributor < admin < owner`; higher roles inherit the capabilities of lower ones.

The Owner count constraint is "at least one," not "only one." The system allows multiple active Owners in the same workspace; when demoting or removing an Owner, only the operation that would leave the workspace with zero Owners is rejected.

### Exceptions at the authorization layer

- **Cross-workspace super admin**: when `enable_cross_tenant_access` is on and the account has `CanAccessAllTenants=true`, switching via `X-Tenant-ID` is equivalent to Admin.
- **API Key**: the synthesized virtual user is fixed at Admin within its own workspace (except for deleting the workspace).
- **Orphan workspace self-healing**: the first authenticated real human is automatically promoted to Owner, preventing API-Key-only workspaces from becoming permanently locked.

## Resource ownership

Migration `000043` adds `creator_id` to the key tables:

- `knowledge_bases.creator_id` — for legacy data, this is backfilled with the workspace's Owner;
- `custom_agents.creator_id` + `runnable_by_viewer` (defaults to `true`, allowing Viewers to invoke it in conversations).

Child resources trace ownership back along the chain `chunk → knowledge → kb → creator_id`.

This produces two kinds of guards:

- **Role guards**: check only the role; used for workspace-level infrastructure (models, vector stores, IM channels, etc.).
- **Ownership guards** (`OwnedXxxOrAdmin`): allow the action if either the creator or an Admin+ role matches; used for write operations on specific resources.

## Relationship to Shared Workspaces

| Dimension | What it solves | Primary key |
|------|---------|------|
| **Workspace RBAC** | Within a single workspace, "what can you do to your own resources / others' resources / shared infrastructure" | `tenant_members(user_id, tenant_id, role)` |
| **Shared Workspaces** | Across workspaces, "letting people in another workspace use my KB / Agent" | `organization_members` + sharing relationships |

The two are **orthogonal**:

- Shared workspaces don't hold the KB / Agent itself — they only record "what permission was used to share what into which workspace"; resource ownership and `creator_id` remain unchanged;
- A write operation on a KB **shared by someone else** requires all of the following: sharing was configured as "writable" + you are not a Viewer in that workspace + the source workspace's RBAC still allows it;
- API Key cross-workspace access does **not** carry an Admin halo — the sharing path is governed by `organization_members.role`, independent of the API Key.

The decision order is in `internal/middleware/kb_access.go`:

```text
Does the KB belong to my current workspace? ─Yes─► Apply workspace RBAC (role + creator_id)
            └─No─► Is the KB shared with my workspace? ─Yes─► Take min(share permission, workspace role)
                                       └─No─► 403 / 404
```

In short: **Workspace RBAC is vertical defense-in-depth, while Shared Workspaces are a horizontal collaboration channel; cross-workspace write actions must pass through both gates.**

## Configuration

```yaml
tenant:
  enable_rbac: true          # if false, enters a "log only, don't block" rollout window
  enable_cross_tenant_access: false
auth:
  registration_mode: self_serve   # or invite_only
audit:
  retention_days: 90              # 0 means no cleanup
```

The environment variables `WEKNORA_TENANT_ENABLE_RBAC` / `WEKNORA_AUDIT_RETENTION_DAYS` override the YAML. `DISABLE_REGISTRATION=true` is equivalent to forcing `registration_mode` to `invite_only`.

## Auditing

The `audit_logs` table records:

- `rbac.member_added` / `removed` / `role_changed` / `left`
- `rbac.access_denied` (only when authorization is enforced; deduplicated with a 1-minute sliding window)

A daily background goroutine cleans up rows older than `audit.retention_days`.

## Related topics

- [Shared Workspaces Guide](./Shared-Spaces-Guide.md) — cross-workspace collaboration and sharing, orthogonal to RBAC
- [OIDC Authentication Flow](./OIDC-Authentication-Flow.md) — the authentication entry point for the multi-workspace user system
- [Lite vs. Standard Edition Differences](../Project-Overview/Lite-vs-Standard-Edition.md) — in Lite's single-user scenario, RBAC has no practical effect

---

## Backlinks

- [Home](../Home.md) — Wiki home navigation
- [Shared Workspaces Guide](./Shared-Spaces-Guide.md) — shared-workspace access ultimately resolves to workspace RBAC checks
- [OIDC Authentication Flow](./OIDC-Authentication-Flow.md) — after JWT parsing, control flow proceeds to RBAC role matching
