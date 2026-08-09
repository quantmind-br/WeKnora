---
title: Shared Space Guide
tags: [Security & Authentication, Shared Space, Collaboration, Multi-Space, Permissions]
aliases: [Shared Space, SharedSpace, Organization, Org]
source: Shared-Spaces-Guide.md
---

# Shared Space Guide

This document explains the **Shared Space** feature in WeKnora, including space creation and joining, member roles and permissions, and knowledge base and agent sharing rules.

> Shared Space is a Standard Edition feature — the [Lite Edition](../Project-Overview/Lite-vs-Standard-Edition.md) does not provide it

## Shared Space Overview

A shared space is a vehicle for cross-space collaboration. Users can belong to different spaces (accounts) within the same system, and by **joining the same shared space**, they can achieve:

- Shared knowledge bases: share a knowledge base from your own space to the shared space, for use by other members within it
- Shared agents: share an agent from your own space to the shared space, for use by other members in conversations and other scenarios
- Access to knowledge bases and agents shared by others

### Core Concept Reference

| Concept | Description |
|------|------|
| Shared Space | An "Organization" within the system, used for cross-space sharing |
| Space Creator | Automatically becomes an admin; cannot be removed or demoted |
| Space Member | Three roles: Admin / Editor / Read-only |
| Knowledge base/agent ownership | Always belongs to a single space; sharing does not change ownership |

## Roles and Permissions

| Capability | Admin | Editor | Read-only |
|------|:-:|:-:|:-:|
| View and search shared knowledge bases | ✓ | ✓ | ✓ |
| Edit shared knowledge base content | ✓ | ✓ | ✗ |
| Share a knowledge base to this space | ✓ | ✓ | ✗ |
| Manage space settings and members | ✓ | ✗ | ✗ |
| Generate/refresh invite codes | ✓ | ✗ | ✗ |

## Knowledge Base Sharing Rules

- Only users within the space that owns the knowledge base can initiate sharing
- The initiator must be an **Admin** or **Editor** of the target space
- Permissions are specified at share time: read-only or writable
- The same knowledge base can be shared to multiple spaces, each with its own independently configured permissions

## Agent Sharing Rules

- When an agent is shared to a space, **only read-only access is supported**
- The same agent can be shared to multiple spaces

> An agent must be fully configured (e.g., a model selected, and if it uses a knowledge base, a rerank model selected) before it can be shared. For model configuration, see [Built-in Model Management](../Core-Features/Builtin-Model-Management.md)

## Agent Deactivation Mechanism

- Deactivation is a personal preference setting within the current space for **agents obtained through a shared space**
- It only affects how the agent is displayed when selecting an agent for conversations within this space, and does not change the sharing relationship
- It does not affect other members

## Related Topics

- [Space RBAC Guide](./RBAC-Guide.md) — roles and resource ownership within a single space; cross-space write operations must satisfy both sides
- [Lite vs. Standard Edition Differences](../Project-Overview/Lite-vs-Standard-Edition.md) — Lite does not support shared spaces
- [OIDC Authentication Flow](../Security-Authentication/OIDC-Authentication-Flow.md) — user authentication in multi-space scenarios
- [Data Source Import Development](../Integration-Extension/Data-Source-Import-Development.md) — knowledge bases from data source imports can be shared
- [Built-in Model Management](../Core-Features/Builtin-Model-Management.md) — models must be configured before an agent can be shared

---

## Backlinks

- [Home](../Home.md) — Wiki home navigation
- [Space RBAC Guide](./RBAC-Guide.md) — shared space access ultimately resolves to space RBAC validation
- [Lite vs. Standard Edition Differences](../Project-Overview/Lite-vs-Standard-Edition.md) — Lite does not support shared spaces
- [OIDC Authentication Flow](../Security-Authentication/OIDC-Authentication-Flow.md) — the multi-space user system underpins shared spaces
- [Data Source Import Development](../Integration-Extension/Data-Source-Import-Development.md) — imported knowledge bases can be shared via shared spaces
- [Built-in Model Management](../Core-Features/Builtin-Model-Management.md) — model configuration requirements before agent sharing
