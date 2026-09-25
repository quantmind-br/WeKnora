# WeChat Mini Program Client

The WeChat Mini Program provides a mobile entry point for knowledge base Q&A and web page import, connecting to an existing WeKnora service. The source code lives in `miniprogram/`. To use it, first configure the API address and API Key, then select a knowledge base:

- Import web page URLs into the selected knowledge base.
- Ask questions against the selected knowledge base.

## Backend Address and Authentication Configuration

The Mini Program **does not hard-code the backend address in the code**; all connection information is filled in by the user on the "Settings" page and stored under the local storage key `weknora_settings` via `wx.setStorageSync`, with a structure containing four fields:

```js
{
  baseUrl: "http://localhost:8080",   // default value written by app.js onLaunch
  apiKey: "",
  selectedKnowledgeBaseId: "",
  locale: "zh"                        // UI language: zh | en
}
```

- **Default value**: In `onLaunch`, `miniprogram/app.js` writes a default `baseUrl: "http://localhost:8080"`, an empty `apiKey`, and `locale: "zh"` if no local settings are found. This default is only for local development convenience — actual use requires changing it to a real address on the Settings page.
- **Read/write and normalization**: `miniprogram/utils/config.js` provides `getSettings()` / `saveSettings()`, strips leading/trailing whitespace and a trailing `/` via `normalizeBaseUrl()`, and maps any value other than `en` to `zh` via `normalizeLocale()`.
- **Authentication method is API Key**: all requests in `miniprogram/utils/request.js` uniformly carry the following request headers:
  - `X-API-Key: <the API Key entered by the user>` (obtained from WeKnora's "Settings → Publish & Integrations → API Integration", in the form `sk-...`);
  - `X-Request-ID: mp-<timestamp>-<random string>` (for server-side tracing);
  - `Content-Type: application/json`.
- **Pre-validation**: if either `baseUrl` or `apiKey` is missing, the request is directly rejected with an error Promise, with the message shown in the current language (e.g. "Please configure the WeKnora API base URL first." / "Please configure the WeKnora API key first."); `pages/index/index.js`'s `onShow` likewise uses this to display a prompt guiding the user to the Settings page.
- **AppID configuration**: copy `miniprogram/project.private.config.json.example` to `project.private.config.json` and fill in your own AppID (the example file's content is `{"appid": "your-wechat-mini-program-appid"}`). Note: the shared `project.config.json` currently contains an `appid` field; when using your own Mini Program, override it with the private configuration.

Backend interfaces invoked (all defined in `miniprogram/utils/request.js`):

| Function | Method and Path |
| --- | --- |
| `listKnowledgeBases()` | `GET /api/v1/knowledge-bases` |
| `createKnowledgeFromURL(kbId, url, enableMultimodel)` | `POST /api/v1/knowledge-bases/{kbId}/knowledge/url` |
| `createSession(kbId)` | `POST /api/v1/sessions` |
| `knowledgeChat(sessionId, query, kbId)` | `POST /api/v1/knowledge-chat/{sessionId}` |

## Language Switching and Local Settings

Since v0.8.2, the Settings page lets you choose "中文" or "English", defaulting to Chinese; the choice is saved in the local settings as `locale` (`zh` / `en`) and persists across restarts. After switching, the page text, navigation bar title (`wx.setNavigationBarTitle`), and bottom tab labels (`wx.setTabBarItem`) update immediately; the bottom tabs also carry icons (`assets/tab/*.png`). Strings are centralized in `utils/i18n.js`, and new pages should reuse that module.

The chat page currently parses the whole SSE text after the request completes and accumulates the `answer` fragments, without chunk-by-chunk real-time rendering. For formal release, the API domain must be added to the request legal domain list.

## Build and Release Process

The Mini Program requires no build step (native development, no build toolchain) — simply open it directly with WeChat DevTools:

1. **Import the project**: in WeChat DevTools, select "Import Project" and point the directory to the repository's `miniprogram/`. The tool will read `project.config.json` (project name "WeKnora Mini Program").
2. **Configure the AppID**: copy `miniprogram/project.private.config.json.example` to `project.private.config.json`, and replace `appid` with your actual Mini Program AppID. `project.private.config.json` is a personal private configuration file, is ignored by `.gitignore`, and should not be committed.
3. **Configure the backend connection**: after running, go to the **Settings** tab, fill in the API Base URL (e.g. `https://weknora.example.com`) and the WeKnora API Key (obtained from "Settings → Publish & Integrations → API Integration"), switch the UI language if needed, then save.
4. **Local debugging note**: `project.config.json` has `urlCheck: true` enabled, so by default DevTools will block requests to non-legal domains such as `localhost`. For local testing, you can check "Do not verify legal domains" in DevTools, or expose the WeKnora service via an HTTPS development domain.
5. **Release**: before formal release, you need to add the WeKnora API domain (which must be HTTPS) to the request legal domain list (request domain whitelist) in the Mini Program admin console on the WeChat Official Accounts Platform; then click "Upload" in DevTools to submit the code, and finally submit it for review and release in the admin console.

### Testing

`miniprogram/package.json` defines a single script:

```bash
cd miniprogram
npm test    # actually runs node --test ../tests/miniprogram/*.test.js
```

That is, it uses Node.js's built-in test runner to run the unit tests in the repository's `tests/miniprogram/miniprogram.test.js` (covering SSE parsing, address normalization, Chinese/English switching, request headers and bodies, and the knowledge base page's loading logic), without needing to install any dependencies.

### Base Library Compatibility

Page and `utils/` code does not use optional chaining `?.` or nullish coalescing `??`: some base library versions do not support these two syntaxes, which causes blank pages (v0.8.2 fixed the knowledge base page going blank for this reason). Text bindings in WXML also stay as flat fields so they render across base library versions. Keep this constraint when modifying the code.

## Implementation Reference

### Tech Stack

This client is a **native WeChat Mini Program**, without using cross-platform frameworks such as Taro / uni-app / mpvue, and with no npm runtime dependencies at all:

- `miniprogram/app.js` — the standard `App({...})` entry point; on `onLaunch` it writes default settings to local storage;
- `miniprogram/app.json` — standard Mini Program global configuration (`pages`, `window`, `tabBar`);
- `miniprogram/app.wxss` — global styles; pages each use the standard `js / wxml / wxss / json` four-file set;
- `miniprogram/package.json` — package name `weknora-miniprogram` (version `0.1.0`), with `description` "WeChat Mini Program plugin for WeKnora", **with no `dependencies`**, only a single test script (see "Testing" above);
- `miniprogram/project.config.json` — `compileType: "miniprogram"`, `libVersion: "latest"` (uses the latest base library), compilation options enable `es6`, `enhance`, `postcss`, `minified`, `minifyWXSS`, `minifyWXML`, and enable `urlCheck: true` (legal domain validation). This file currently contains an `appid` field; your own AppID is provided via a private configuration file (see "Build and Release Process").

Global window style: navigation bar title `WeKnora`, background color `#0d3b2a` (dark green), white text.

### Page List

`miniprogram/app.json` registers 3 pages, and all three together make up the bottom `tabBar` (selected color `#07c05f`, each item with a normal / selected icon pair). Tab labels default to Chinese and switch with the language at runtime:

| Page Path | tabBar label (Chinese / English) | Function |
| --- | --- | --- |
| `pages/index/index` | 知识库 / Knowledge | Home page. Checks whether baseUrl / API Key are already configured; if not, prompts the user with a one-tap jump to Settings; calls `GET /api/v1/knowledge-bases` to load the knowledge base list, and lets the user select a knowledge base via a `picker` or list tap (the selection is persisted to local storage); after entering a web page URL, calls `POST /api/v1/knowledge-bases/{id}/knowledge/url` to import that URL into the selected knowledge base (`enable_multimodel` is fixed to `false`) |
| `pages/chat/chat` | 问答 / Chat | Knowledge Q&A page. On the first question, lazily creates a session via `POST /api/v1/sessions` (carrying the selected `knowledge_base_id`), then calls `POST /api/v1/knowledge-chat/{sessionId}` to ask the question; the response body is SSE text, which the client parses with `utils/sse.js`, concatenating the chunks where `response_type === "answer"` and displaying the result as a whole (falling back to displaying the raw response if parsing fails) |
| `pages/settings/settings` | 设置 / Settings | Connection configuration page. Fill in the API Base URL and API Key (password input field), choose the UI language, and save to local storage under `weknora_settings` |

### utils/ Utility Modules

| File | Responsibility |
| --- | --- |
| `miniprogram/utils/config.js` | Persistence layer for settings: defines the storage key `STORAGE_KEY = "weknora_settings"`, provides `getSettings()`, `saveSettings()` (merge-style update), `normalizeBaseUrl()` (trims and strips a trailing slash), and `normalizeLocale()` |
| `miniprogram/utils/i18n.js` | Chinese/English string tables plus `t()`, `getLocale()` / `setLocale()`, `applyTabBar()` (updates bottom tab labels and icons), and `applyNavTitle()` (updates the navigation bar title) |
| `miniprogram/utils/request.js` | Promise-based HTTP wrapper built on `wx.request`: concatenates `baseUrl + path`, injects `X-API-Key` / `X-Request-ID` headers, applies unified 2xx status checking and error message extraction (preferring `error.message`, then `message`, falling back to `HTTP <status>`); also exports the 4 business API functions listed in the table above |
| `miniprogram/utils/sse.js` | Server-Sent Events text parser: `parseSSE(raw)` splits event blocks on blank lines and parses `event:` / `data:` lines; `collectAnswerFromSSE(raw)` JSON-parses each event's `data` and accumulates the `content` where `response_type === "answer"` to produce the final answer text. Note that the Mini Program side **does not do streaming rendering** — it waits until `wx.request` receives the complete SSE text and then parses and displays it all at once |

### Data Flow Overview

```mermaid
flowchart LR
    S["Settings page<br/>(baseUrl + API Key + language)"] -->|"wx.setStorageSync(weknora_settings)"| C["utils/config.js"]
    K["Knowledge page<br/>(pages/index)"] -->|"listKnowledgeBases / createKnowledgeFromURL"| R["utils/request.js<br/>(X-API-Key header)"]
    Q["Chat page<br/>(pages/chat)"] -->|"createSession / knowledgeChat"| R
    R -->|"wx.request"| B["WeKnora Backend<br/>/api/v1/*"]
    B -->|"SSE text"| P["utils/sse.js<br/>collectAnswerFromSSE"]
    P --> Q
    C --> R
```
