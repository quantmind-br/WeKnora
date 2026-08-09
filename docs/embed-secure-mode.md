# Embed Security Mode

> **In one sentence**: the long-lived key (publish token `em_…`) stays only on **your own server**; the visitor's browser only ever holds a short-lived token (`ems_…`) valid for 30 minutes.

## Why use it?

| Approach | Where the publish token lives | Risk |
|------|-----------------|------|
| iframe / regular Widget | Embedded in the page HTML or URL hash | Anyone can copy it via "View Source" — effectively a public key |
| **Security Mode Widget** | Only in environment variables / secret management, server-side | The browser never gets the long-lived key; you can also verify the visitor is logged in first |

For production embeds facing the public, **Security Mode should be the default choice**.

## Two Kinds of Tokens

| Name | Format | Held by | Purpose |
|------|------|--------|------|
| Publish Token | `em_…` | Your server only | Exchanged with WeKnora for a short-lived token; visible in the admin panel under "Channel Keys" |
| Session Token | `ems_…` | Visitor's browser (inside the iframe) | Used to call chat, upload, and other embed APIs; expires in about 30 minutes, and the Widget refreshes it automatically |

## Workflow

```
Visitor's browser            Your backend (shop's server)           WeKnora
     │                              │                              │
     │ 1. Loads Widget              │                              │
     │    data-token-endpoint       │                              │
     │    (does not contain em_)    │                              │
     │─────────────────────────────►│                              │
     │                              │ 2. Verify visitor is logged in (optional) │
     │                              │ 3. POST .../embed/:id/exchange │
     │                              │    Authorization: Embed em_…  │
     │                              │─────────────────────────────►│
     │                              │◄──── session_token (ems_…) ───│
     │◄── 4. { token, expiresIn } ──│                              │
     │ 5. iframe chats using ems_   │                              │
```

This maps to the two code snippets under Admin Panel → "Embed Channels → Security Mode":

1. **Page script**: `data-token-endpoint="https://your-domain/weknora/embed-token"` (no `data-token`)
2. **Server-side endpoint**: uses the publish token to call exchange and returns `ems_…` to the frontend

## Integration Steps

### Step 1: Create a Channel in WeKnora

- Note down the **Channel ID** and **Publish Token** (`em_…`)
- Configure the domain allowlist (see below)
- Configure per-minute / per-day rate limits

### Step 2: Deploy the Token-Exchange Endpoint

Add a new HTTP endpoint (any path you like) to your own backend, with these requirements:

**Input**: a browser GET request (the Widget will `fetch` this address)

**You must**:

- Verify the caller is a legitimate visitor (session cookie, JWT, etc.); return `401` if not logged in
- Call the WeKnora exchange endpoint using the publish token
- On success, return JSON: `{ "token": "<ems_…>", "expiresIn": 1800 }`

**Contract for calling exchange**:

```http
POST https://<weknora-host>/api/v1/embed/<channel_id>/exchange
Authorization: Embed <publish token em_…>
Origin: https://<your-business-site>    ← must match the channel allowlist, otherwise 403
```

> Server-side `fetch` does not send an `Origin` header by default — you need to **set it manually** to match the allowlist.

### Step 3: Paste the Security Mode Widget Code

Change `data-token-endpoint` to the real URL from the previous step. A complete example can be copied from the "Security Mode" tab in the admin panel.

## Server-Side Examples

Replace `<WEKNORA_HOST>` and `<CHANNEL_ID>` below with actual values; put the publish token in the environment variable `WEKNORA_PUBLISH_TOKEN` — **do not** put it in the frontend.

### Node.js (Express)

```javascript
const WEKNORA_BASE = 'https://<WEKNORA_HOST>';
const CHANNEL_ID = '<CHANNEL_ID>';
const ALLOWED_ORIGIN = 'https://shop.example.com'; // must match the channel allowlist

app.get('/weknora/embed-token', async (req, res) => {
  const hasSession = Boolean(req.cookies?.session_id);
  const auth = req.headers.authorization || '';
  if (!hasSession && !auth.startsWith('Bearer ')) {
    return res.status(401).json({ error: 'unauthorized' });
  }

  const r = await fetch(`${WEKNORA_BASE}/api/v1/embed/${CHANNEL_ID}/exchange`, {
    method: 'POST',
    headers: {
      Authorization: 'Embed ' + process.env.WEKNORA_PUBLISH_TOKEN,
      Origin: ALLOWED_ORIGIN,
    },
  });
  const body = await r.json();
  if (!body?.data?.session_token) {
    return res.status(502).json({ error: 'mint failed' });
  }
  res.json({ token: body.data.session_token, expiresIn: body.data.expires_in });
});
```

### Go (net/http)

```go
func embedTokenHandler(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Authorization") == "" && r.Header.Get("Cookie") == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	req, _ := http.NewRequest(http.MethodPost,
		"https://<WEKNORA_HOST>/api/v1/embed/<CHANNEL_ID>/exchange", nil)
	req.Header.Set("Authorization", "Embed "+os.Getenv("WEKNORA_PUBLISH_TOKEN"))
	req.Header.Set("Origin", "https://shop.example.com") // must match the channel allowlist
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode >= 300 {
		http.Error(w, `{"error":"mint failed"}`, http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	var body struct {
		Data struct {
			SessionToken string `json:"session_token"`
			ExpiresIn    int    `json:"expires_in"`
		} `json:"data"`
	}
	if json.NewDecoder(resp.Body).Decode(&body) != nil || body.Data.SessionToken == "" {
		http.Error(w, `{"error":"mint failed"}`, http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"token": body.Data.SessionToken, "expiresIn": body.Data.ExpiresIn,
	})
}
```

The "Security Mode → Server-Side Example" tab in the admin panel generates a snippet with the real URL for your current channel ID.

## How Do I Fill In the Domain Allowlist?

| Request origin to allow | Allowlist example |
|------------------|------------|
| The origin the chat iframe is served from (the embed page) | `https://app.example.com` or `https://embed.example.com` |
| Your token-exchange backend (the `Origin` sent during exchange) | `https://shop.example.com` |

If the embed uses a [dedicated subdomain](./embed-subdomain.md), **add both** (the embed origin and your business backend's origin).

You can temporarily use `*` in development; **`*` is forbidden in production**.

## Go-Live Checklist

- [ ] The publish token is injected only via environment variables / a secret manager — never committed to Git, never bundled into the frontend
- [ ] The token-exchange endpoint verifies visitor identity
- [ ] The entire chain runs over HTTPS
- [ ] The allowlist includes both the embed origin and the origin used during exchange
- [ ] Rate limiting is configured; for sensitive agents, don't use regular mode, which exposes the token on the web page
- [ ] After rotating the publish token, the server-side environment variable is updated accordingly

## FAQ

| Symptom | Cause and fix |
|------|------------|
| exchange returns **401** / `publish token required` | The publish token is wrong, has been rotated, or you mistakenly used the `ems_` session token instead |
| exchange or the chat API returns **403** `origin not allowed` | The allowlist doesn't include the current request's `Origin`; remember to manually set the `Origin` header during server-side exchange |
| The iframe stays stuck "waiting for token" | The `token-endpoint` isn't returning `{ token, expiresIn }`, or CORS isn't allowing the Widget's origin to access your endpoint |
| The token-exchange endpoint returns **502** `mint failed` | WeKnora is unreachable, the channel has been disabled, or the exchange response format is wrong |
| Visitors can chat without any restriction | The token-exchange endpoint isn't performing login verification — add a session/JWT check before exchange |

## Related

- Optional: dedicated embed subdomain → [embed-subdomain.md](./embed-subdomain.md)
- Widget SDK comments: `frontend/public/weknora-widget.js`
- Code generation: `frontend/src/api/embed/index.ts`
