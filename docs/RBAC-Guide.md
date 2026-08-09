# Workspace RBAC Guide

This document describes the design of WeKnora's **in-workspace permission control (Workspace RBAC)**, the role matrix, the resource-ownership model, configuration options, and its relationship with [Shared Workspaces](./Shared-Spaces-Guide.md).

> Status: shipped with #1303, controlled by the config item `tenant.enable_rbac`, default `true` (enforced authorization). Can be temporarily switched to `false` to enter a "log only, don't block" rollout window.

## 1. Why RBAC Is Needed

Before RBAC was introduced, any caller who successfully authenticated via `X-API-Key` or JWT was essentially an admin within the workspace. This was fine for single-user self-hosted scenarios, but as soon as a workspace has two or more real human members (a team sharing one knowledge base), you need to distinguish:

- Who can delete knowledge bases and revoke API Keys (Admin/Owner);
- Who can upload documents and edit "their own" knowledge bases (Contributor);
- Who can only read and ask questions (Viewer).

RBAC layers an **in-workspace role matrix** on top of the existing JWT / API Key authentication, making all three states first-class citizens.

## 2. Role Matrix

Each workspace member (a row in `tenant_members`) has exactly one role:

| Role | Identifier | Typical Scenario | Key Capabilities |
|------|------|----------|----------|
| Viewer | `viewer` | Members who only read and ask questions | Read-only, cannot initiate any changes |
| Contributor | `contributor` | Uploading documents, maintaining their own KB / Agent | Can modify resources where `creator_id == self`; other people's resources are treated like Viewer |
| Admin | `admin` | Workspace operations | Can modify any resource in the workspace; manage members; configure shared infrastructure (models, parsers, storage, vector stores, etc.) |
| Owner | `owner` | Workspace owner | All Admin permissions + can delete the workspace; every workspace has **at least one, and can have multiple** |

Roles increase in the order `viewer < contributor < admin < owner`, with higher roles inheriting the permissions of lower ones.

The constraint on Owner count is "at least one," not "only one." The system allows multiple active Owners in the same workspace; when demoting or removing an Owner, only operations that would leave the workspace with zero Owners are rejected.

### Two Exceptions at the Authorization Layer

- **Cross-workspace super admin**: when `User.CanAccessAllTenants=true` and `enable_cross_tenant_access=true`, switching to a target workspace via `X-Tenant-ID` is equivalent to Admin, without needing a `tenant_members` row in the target workspace. Used for multi-workspace operators.
- **API Key calls**: the virtual user synthesized from `X-API-Key` is fixed as Admin within its own workspace (deleting the workspace still requires Owner). No migration needed for script-based integrations.
- **Orphan workspace self-healing**: if a workspace has no active members in the `tenant_members` table (typical scenario: the workspace has only ever been used via API Key), the first authenticated real human is automatically promoted to Owner, avoiding a lockout.

## 3. Resource Ownership Model

The role matrix alone isn't enough, otherwise Contributors could interfere with each other. To address this, migration `000043` added `creator_id` to the key tables:

- `knowledge_bases.creator_id` — backfilled to the workspace's Owner for legacy data; an empty string/NULL means "shared by the workspace, only Admin+ can modify."
- `custom_agents.creator_id` — the Agent's creator.
- `custom_agents.runnable_by_viewer` — defaults to `true`, allowing Viewers to invoke this Agent in conversations; setting it to `false` raises the requirement to Contributor and above.

Sub-resources trace back through the ownership chain to the KB's `creator_id`:

```
chunk_id ─► knowledge_id ─► kb_id ─► knowledge_bases.creator_id
```

The same applies to FAQ entries, generated questions, KB tags, and wiki pages.

This gives rise to two kinds of guards:

- **Role guards**: `Viewer()` / `Contributor()` / `Admin()` / `Owner()` — check the role only. Used for workspace-level infrastructure (models, vector stores, IM channels, etc.).
- **Ownership guards**: `OwnedKBOrAdmin()` / `OwnedAgentOrAdmin()` / `OwnedChunkKBOrAdmin()` … — satisfied if either "I am this resource's `creator_id`" **or** "I am at least Admin" holds. Used for write operations on specific resources.

This naturally makes it so that "a Contributor behaves like an Owner in their own KB, and like a Viewer in someone else's KB."

## 4. Relationship with Shared Workspaces (Key Section)

[Shared Workspaces](./Shared-Spaces-Guide.md) (Organization) and Workspace RBAC solve **different dimensions** of the problem, and both must be satisfied to complete a cross-workspace operation:

| Dimension | What It Solves | Primary Key Model | Role Set |
|------|---------|---------|---------|
| **Workspace RBAC** | Within the same workspace, "what can you do to yourself / others / shared infrastructure" | `tenant_members(user_id, tenant_id, role)` | viewer / contributor / admin / owner |
| **Shared Workspaces** | Cross-workspace, "let people in another workspace use my KB / Agent" | `organization_members(user_id, org_id, role)` + sharing relation tables | Admin / Editor / Read-only |

The two are **orthogonal**:

- A shared workspace holds no KB or Agent itself — it's merely a relationship record of "some KB shared with some permission into some workspace."
- A resource always belongs to a single workspace; ownership and `creator_id` never change due to sharing.
- A write operation on a KB **shared with you by someone else** requires all of the following to hold simultaneously:
  1. **Sharing side**: the KB has been shared, with "write" permission, into a shared workspace that both you and the sharer belong to;
  2. **Workspace role side**: your role in that shared workspace is not "read-only" (i.e., you're at least an Editor);
  3. **Workspace RBAC side**: your access, resolved via the sharing path, is treated as access to the source workspace "acting as a member of the shared workspace," and must still pass the source workspace's RBAC check. Specifically, the access check falls back to shared-path validation only after confirming `kb.tenant_id == your current workspace` does not hold.

Simplified decision order (see `internal/middleware/kb_access.go`):

```text
┌────────────────────────┐
│ Does the KB belong to   │ ──Yes──► Enter Workspace RBAC:
│ my current workspace?   │         role + creator_id determine write access
└──────┬─────────────────┘
       No
       ▼
┌────────────────────────┐
│ Was the KB shared via a │ ──No──► 403 / 404
│ shared workspace I'm in?│
└──────┬─────────────────┘
       Yes
       ▼
┌────────────────────────┐
│ Am I a viewer in that   │ ──Yes──► Read-only
│ workspace?              │ ──No──► Execute per the "read-only/write" setting configured at share time
└────────────────────────┘
```

Key takeaways:

- **Shared workspaces never bypass Workspace RBAC**: if a KB is marked "write requires Admin+" in the source workspace (e.g., a workspace-shared KB with an empty `creator_id`), then even if "write" permission was granted at share time, members of the outside workspace can only read it — because no one can become an Admin of the source workspace across workspaces.
- **Cross-workspace access via API Key**: an API Key is Admin within its own workspace, but this **does not** automatically grant write access to KBs shared from other workspaces via a shared workspace — shared workspaces use `organization_members.role`, which is unrelated to API Keys.
- **Auditing is also separate**: in-workspace role changes are written to `audit_logs` (`rbac.member_*` actions), while member and sharing-relationship changes within a shared workspace are recorded by the shared workspace's own interfaces.

In one sentence: **Workspace RBAC is the "vertical" defense-in-depth, and shared workspaces are the "horizontal" collaboration channel; any cross-workspace operation with write side effects must pass through both gates simultaneously.**

## 5. Configuration

`config/config.yaml`:

```yaml
tenant:
  # Default true, enforced authorization. Set to false to enter a "log only, don't block" rollout window
  enable_rbac: true
  # Cross-workspace super admin switch, default false
  enable_cross_tenant_access: false

auth:
  # self_serve (default): anyone can register; a workspace + Owner member is created automatically
  # invite_only         : public registration disabled; new users must be invited via /tenants/:id/members
  registration_mode: self_serve

audit:
  # Audit log retention in days; cleaned up daily in the background; default 90; set to 0 to disable cleanup
  retention_days: 90
```

Environment variables (take precedence over YAML):

| Environment Variable | YAML Path | Value |
|----------|-----------|------|
| `WEKNORA_TENANT_ENABLE_RBAC` | `tenant.enable_rbac` | `true` / `false` |
| `WEKNORA_AUDIT_RETENTION_DAYS` | `audit.retention_days` | non-negative integer |

`auth.registration_mode` has no dedicated environment variable; it continues to use the legacy `DISABLE_REGISTRATION=true` — once set, `auth.registration_mode` is forced to `invite_only` at startup, keeping the backend API and the frontend registration entry point (driven by `/auth/config`) consistent.

The startup log prints a summary line confirming exactly which configuration set was used for this run and where each value's override came from.

## 6. Audit Logs

The `audit_logs` table uniformly records permission-related events:

| Action | Outcome | Triggered When |
|--------|---------|----------|
| `rbac.member_added` | success | `POST /tenants/:id/members` succeeds |
| `rbac.member_removed` | success | `DELETE /tenants/:id/members/:user_id` succeeds |
| `rbac.member_role_changed` | success | `PUT /tenants/:id/members/:user_id` succeeds |
| `rbac.member_left` | success | `POST /tenants/:id/members/leave` succeeds |
| `rbac.access_denied` | denied | `RequireRole` / `RequireOwnershipOrRole` denies a request (**only when enforcement is on**) |

`access_denied` is deduplicated using a 1-minute sliding window, to prevent malicious probing from flooding the table; the same denials remain visible line-by-line in the application log (`[rbac] role insufficient ...`).

Knowledge base activity logs also reuse this same immutable audit table, establishing a scoped index via `scope_type=knowledge_base`, `scope_id=<kb_id>`, rather than creating a separate log table per knowledge base. `GET /api/v1/knowledge-bases/:id/activity` provides cursor-based pagination, covering KB configuration, knowledge/FAQ, tags, data sources, sharing, clone/move tasks, and wiki content changes; the original workspace audit endpoint only returns RBAC events with no resource scope, avoiding KB activity from drowning out member and denial records. Parsing-stage detail continues to be shown via knowledge spans, and data-source sync detail continues to be shown via sync logs — the activity log only stores summaries and associated IDs.

This endpoint is readable only by the KB's creator or the workspace's Admin+, and explicitly denies reads from an organization shared workspace, avoiding leaking the source workspace's operators and configuration history to the recipient side. Activity details must not contain knowledge body content, external URLs, data source credentials, or raw error stack traces.

The background goroutine `AuditLogRetentionRunner` starts its first cleanup round ~10 minutes after startup, then sweeps rows older than `audit.retention_days` every 24 hours thereafter; when the retention period is `0`, the entire goroutine short-circuits and generates no DB traffic.

## 7. Rollout Recommendations

Whether for self-hosted operations or the upstream repository itself, switching from "log only" to "enforced authorization" is recommended to follow this process:

1. **Upgrade**: if you want to keep an observation window, set `tenant.enable_rbac=false` (or the corresponding environment variable) before upgrading. Otherwise, enforced authorization is the default — the schema is applied, `tenant_members` is auto-backfilled (one Owner per workspace, the rest Contributors), and all KBs automatically get `creator_id` written.
2. **Verify members**: call `GET /api/v1/tenants/:id/members` to confirm:
   - Each workspace has exactly one Owner;
   - Contributor / Viewer assignments match expectations;
   - Adjust via `PUT /api/v1/tenants/:id/members/:user_id` / `DELETE`. Every adjustment is written to `audit_logs`.
3. **Observe logs**: capture `[rbac] role insufficient (logged but not enforced) ...` lines from the application log — these are the requests that will become 403s once enforced authorization is switched on. Fix member roles or client identities one by one.
4. **Switch to enforced authorization**: remove the `tenant.enable_rbac=false` override (or explicitly set it to `true`), and restart the service. From then on:
   - Insufficient role → 403;
   - Also writes to `audit_logs.rbac.access_denied` (subject to deduplication).
5. **Optional: disable public registration**: change `auth.registration_mode` to `invite_only`. The registration entry on the login page disappears automatically, and `POST /auth/register` returns 403 directly.

### Rollback

```bash
export WEKNORA_TENANT_ENABLE_RBAC=false
# Restart the service to return to observation mode
```

The `tenant_members` rows and `creator_id` column are preserved, so re-enabling later doesn't require redoing the backfill. Unless you're abandoning this feature entirely, **do not** roll back the `000043` / `000044` migrations — their `down.sql` drops the entire `tenant_members` and `audit_logs` tables.

## 8. Frontend Behavior

The `authStore` in Pinia exposes:

- `authStore.currentTenantRole`: `''` before member info finishes loading (a loading signal — buttons wait to render until this resolves, avoiding a "bright then grayed-out" flicker); afterward, one of the four roles.
- `authStore.hasRole('admin')` etc.: convenience functions for hierarchical checks.
- Each resource page additionally layers on `isOwner` (e.g., `kb.creator_id === authStore.user?.id`) for per-resource checks.

This mirrors the backend guards: **any button that would 403 on the backend is simply hidden on the frontend, rather than letting the user click it and hit an error.**

### Actual Frontend UI

<table>
  <tr>
    <td colspan="2" align="center">
      <b>Member Management Page</b><br/>
      <img src="./images/rbac-member-management.png" alt="Member Management" width="100%"/>
      <br/><sub>Shows both "pending invitations" and "workspace members" lists side by side; only Owners can add / remove members; the "Audit Log" entry in the top-right corner links to the <code>audit_logs</code> view.</sub>
    </td>
  </tr>
  <tr>
    <td width="50%" align="center">
      <b>User Menu + Workspace Switcher</b><br/>
      <img src="./images/rbac-workspace-switcher.png" alt="User Menu + Switch Workspace" width="100%"/>
      <br/><sub>Left: current workspace role badge / settings entry / logout; right: switch to other workspaces, with a "current" badge marking the active workspace.</sub>
    </td>
    <td width="50%" align="center">
      <b>Self-Service Workspace Creation</b><br/>
      <img src="./images/rbac-create-workspace.png" alt="Create New Workspace" width="100%"/>
      <br/><sub>Any user can create a workspace on their own, automatically becoming the new workspace's Owner upon creation (protected by the <code>WEKNORA_TENANT_MAX_PER_USER</code> limit).</sub>
    </td>
  </tr>
  <tr>
    <td colspan="2" align="center">
      <b>Pending Invitations Dialog</b><br/>
      <img src="./images/rbac-pending-invitation.png" alt="My Invitations" width="80%"/>
      <br/><sub>The invitation bell in the user menu shows pending invitations from other workspaces, which can be directly "accepted / declined"; invitations expire automatically after 7 days without a response.</sub>
    </td>
  </tr>
</table>

## 9. FAQ

### After upgrading, everyone became a Contributor and I can't find an Admin?

The backfill logic picks "the earliest active user in each workspace" as the Owner, with everyone else uniformly becoming Contributor. If the account that created the workspace was actually a bot / shared account, you may need to first demote the bot and promote a real human to Admin via `PUT /api/v1/tenants/:id/members/:user_id`.

### After switching to enforced authorization, some script started getting 403s?

Most likely, the member corresponding to the script's JWT is a Viewer / Contributor rather than Admin. Two solutions:
- Upgrade the corresponding user to Admin via `tenant_members`;
- Or switch the script to use `X-API-Key` calls — an API Key is fixed as Admin within its own workspace.

### Why can't a member in a shared workspace read a KB I shared "with write access"?

Follow the decision order in Section 4 to troubleshoot:
- Does the KB actually belong to the source workspace, and is `tenant_id` configured correctly;
- Does the sharing relationship still currently exist (has it not been revoked);
- Is the caller a Viewer within the shared workspace;
- If the KB's `creator_id` is empty in the source workspace and the shared permission requires write, the source workspace's RBAC will still require Admin+, meaning cross-workspace write cannot succeed.

### Why are some 403s missing from the audit log?

Two possibilities:
- The 1-minute sliding window deduplication writes only one line per `(actor, path, action)` combination within a minute. The full sequence remains visible in the application log.
- When `tenant.enable_rbac=false`, only member-management events are logged; `rbac.access_denied` is not written.

### Can I define a more granular ACL than "role + ownership"?

Not in v1. This matrix deliberately keeps to a small fixed grid (Viewer < Contributor < Admin < Owner) + a per-resource "creator escape hatch." Finer-grained policies (e.g., "a Viewer can see their own audit log") are a future consideration.

## 10. Testing and Observability

- `make test` covers roughly 25 test cases including `internal/middleware/rbac_test.go`, `internal/handler/rbac_lookups_test.go`, `internal/application/service/audit_log_test.go`, and `internal/middleware/rbac_audit_test.go`.
- Langfuse / OpenTelemetry spans carry the resolved `TenantRole` and `TenantID`, so a denied request's corresponding role is visible right in the trace, without needing to manually cross-reference logs.

## Related Documents

- Cross-workspace collaboration: [`Shared-Spaces-Guide.md`](./Shared-Spaces-Guide.md)
- Multi-workspace authentication background: [`OIDC-Authentication-Flow.md`](./OIDC-Authentication-Flow.md)
- Configuration items and environment variables: [`.env.example`](../.env.example)
