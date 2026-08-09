---
title: OIDC Authentication Flow
tags: [Security Authentication, OIDC, Authentication, Login, SSO]
aliases: [OIDC, OIDC Authentication, SSO Login, Third-Party Login]
source: OIDC-Authentication-Flow.md
---

# OIDC Authentication Flow

This document explains WeKnora's current OIDC login capability and its actual invocation process, covering the complete front-end and back-end chain.

> OIDC authentication is the login method for multi-space scenarios in the Standard Edition; the [Lite Edition](../Project-Overview/Lite-vs-Standard-Edition.md) does not require it

## Overall Design

This project's OIDC login adopts the pattern of **the backend initiating authorization parameter generation, the backend receiving the callback and completing the code-to-token exchange, and the frontend receiving the final login result via URL hash**.

Core characteristics:

1. **The frontend is only responsible for initiating the redirect** and does not directly exchange tokens with the OIDC Provider
2. **The backend is responsible for exchanging the authorization code `code` for a token with the OIDC Provider**
3. After the backend obtains the OIDC user information, it looks up the local user; if one does not exist, it automatically creates a local account and default space
4. Ultimately, WeKnora's own local JWT is issued — the OIDC token is only used by the backend to exchange for user identity

> The logic for automatically creating users and spaces is related to the multi-space model described in [Shared Space Guide](../Security-Authentication/Shared-Spaces-Guide.md)

## Related Endpoints

| Endpoint | Description |
|------|------|
| `GET /api/v1/auth/oidc/config` | Get whether OIDC is enabled and the Provider's display name |
| `GET /api/v1/auth/oidc/url` | Generate the third-party login redirect URL |
| `GET /api/v1/auth/oidc/callback` | OIDC Provider callback address |

## Invocation Flow (4 Stages)

### Stage 1: Frontend Capability Discovery

The frontend calls `/auth/oidc/config` to decide whether to display the third-party login entry point.

### Stage 2: Browser Redirect for Authorization

The frontend calls `/auth/oidc/url` to obtain the authorization URL, then redirects to the OIDC Provider.

### Stage 3: Backend Completes Identity Exchange

The Provider calls back to the backend's `/auth/oidc/callback`; the backend exchanges the `code` for a token, retrieves user information, links or creates a local user, and issues a WeKnora JWT.

### Stage 4: Frontend Receives the Final Result

The backend issues a 302 redirect back to the frontend, passing the login result via `#oidc_result`; the frontend parses this uniformly in `App.vue`.

## Key Configuration Items

| Configuration Item | Description |
|--------|------|
| `OIDC_AUTH_ENABLE` | Whether to enable OIDC login |
| `OIDC_AUTH_CLIENT_ID` | OIDC Client ID |
| `OIDC_AUTH_CLIENT_SECRET` | OIDC Client Secret |
| `OIDC_AUTH_DISCOVERY_URL` | OIDC Discovery URL |
| `OIDC_AUTH_SCOPES` | Scope list, default `openid profile email` |

Minimum requirements when enabled: `client_id` + `client_secret` + (`discovery_url` or `authorization_endpoint + token_endpoint`)

## Local Integration Testing Example (Dex)

The project already provides a sample Dex configuration: `misc/dex-config.yaml`.

> Besides Dex, you can also use other Providers compliant with the OpenID Connect protocol, such as KeyCloak

## Notes

1. **`redirect_uri` must strictly match** the Provider's client configuration
2. **Email is the primary key for local account linking** — if the Provider does not return an email, login cannot be completed
3. **The first OIDC login automatically creates a user and default space**
4. **What is actually used to access the API is the local JWT**, not the OIDC access token

## Related Topics

- [Shared Space Guide](../Security-Authentication/Shared-Spaces-Guide.md) — User and organization management in multi-space scenarios
- [Lite vs Standard Edition](../Project-Overview/Lite-vs-Standard-Edition.md) — Lite Edition does not require OIDC (single space)
- [API Documentation Overview](../API-Reference/API-Documentation-Overview.md) — API authentication mechanism

---

## Backlinks

- [Home](../Home.md) — Wiki homepage navigation
- [Shared Space Guide](../Security-Authentication/Shared-Spaces-Guide.md) — Users and spaces created by OIDC can be used for shared spaces
- [Lite vs Standard Edition](../Project-Overview/Lite-vs-Standard-Edition.md) — Lite does not require OIDC (single space, no registration needed)
- [API Documentation Overview](../API-Reference/API-Documentation-Overview.md) — API authentication mechanism related to OIDC JWT
