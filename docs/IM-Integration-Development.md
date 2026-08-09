# IM Integration Development Documentation

WeKnora's IM integration module connects enterprise instant messaging platforms (WeCom, Feishu, Lark, Slack, Telegram, DingTalk, Mattermost) to the WeKnora knowledge Q&A pipeline, enabling users to ask AI questions directly within IM and receive real-time streaming answers.

IM channels are bound to an Agent — one Agent can connect to multiple IM channels, and all configuration is managed through the frontend Agent editor and stored in the database.

## Table of Contents

- [Quick Integration Guide](#quick-integration-guide)
  - [WeCom Integration](#wecom-integration)
  - [Feishu Integration](#feishu-integration)
  - [Lark Integration](#lark-integration)
  - [Slack Integration](#slack-integration)
  - [Telegram Integration](#telegram-integration)
  - [DingTalk Integration](#dingtalk-integration)
  - [Mattermost Integration](#mattermost-integration)
- [Frontend Management](#frontend-management)
- [Architecture Overview](#architecture-overview)
- [Data Model](#data-model)
- [API Endpoints](#api-endpoints)
- [Core Concepts](#core-concepts)
- [Message Processing Flow](#message-processing-flow)
- [Interface Definitions](#interface-definitions)
- [Platform Adapter Details](#platform-adapter-details)
  - [WeCom](#wecom)
  - [Feishu and Lark](#feishu-and-lark)
  - [Slack](#slack-1)
  - [Telegram](#telegram-1)
  - [DingTalk](#dingtalk)
  - [Mattermost](#mattermost-1)
- [Slash Command System](#slash-command-system)
- [QA Queue and Rate Limiting](#qa-queue-and-rate-limiting)
- [Streaming Output Mechanism](#streaming-output-mechanism)
- [File Message Handling](#file-message-handling)
- [Key Parameters and Thresholds](#key-parameters-and-thresholds)
- [Error Handling](#error-handling)
- [Extending to New Platforms](#extending-to-new-platforms)

---

## Quick Integration Guide

### Prerequisites

- WeKnora is deployed and running
- At least one Agent (custom assistant) has been created
- The Agent has a configured model and knowledge base

> **Publicly reachable images are required for replies containing images**: for IM to display knowledge-base images, the IM side must be able to reach a publicly accessible HTTP image URL. Choose one of the following —
> (A) The storage backend itself is publicly reachable (object storage's public endpoint / `MINIO_ENDPOINT` set to a public address);
> (B) Configure `APP_EXTERNAL_URL` so images go through WeKnora's `/r/` short link (nginx already has the `/r/` proxy built in).
> For the default MinIO deployment, option (B) is simplest. See the `APP_EXTERNAL_URL` description in `.env.example` for details.

### WeCom Integration

WeCom offers two integration modes — choose based on your application type:

#### Method 1: WebSocket Mode (Smart Bot, Recommended)

> No public domain required — suitable for quick validation and intranet deployments.

**Step 1: Create a Smart Bot**

1. Log in to the [WeCom Workspace] (make sure you're on the latest version of WeCom) → **Smart Bot** → **Create Bot** → **Manual Creation** → **Switch to API Mode Creation** → **Select "Use Long Connection"**
2. Once created, on the bot details page, obtain:
   - **BotID** — the bot's unique identifier
   - **BotSecret** — the bot's secret key (click reset to regenerate)

**Step 2: Add an IM Channel in WeKnora**

1. Go to the Agent editor → select the **IM Integration** tab in the left navigation
2. Click **Add Channel**
3. Fill in the configuration:
   - **Platform**: select "WeCom"
   - **Channel Name**: a custom name for easy identification (e.g. "Customer Service Bot")
   - **Integration Mode**: select "WebSocket"
   - **Output Mode**: select "Streaming" (recommended)
   - **Bot ID**: enter the BotID obtained from WeCom
   - **Bot Secret**: enter the BotSecret obtained from WeCom
4. Click Save

**Step 3: Verification**

After saving, WeKnora will automatically establish a WebSocket long connection to WeCom. The following log entry indicates a successful connection:

```
[IM] WeCom WebSocket connecting (bot_id=xxx)...
```

At this point, sending a message to the bot in WeCom will get an AI reply.

---

#### Method 2: Webhook Mode (Custom Application)

> Requires a publicly reachable callback address — suitable for scenarios with an existing custom application.

**Step 1: Create a Custom Application**

1. Log in to the [WeCom Admin Console](https://work.weixin.qq.com/) → **App Management** → **Custom Apps** → **Create App**
2. Record the following information:
   - **CorpID** — found at the bottom of the **My Company** → **Company Info** page
   - **AgentID** — the AgentId (integer) on the app details page
   - **Secret** — the Secret on the app details page

**Step 2: Add an IM Channel in WeKnora**

1. Go to the Agent editor → **IM Integration** tab → **Add Channel**
2. Fill in the configuration:
   - **Platform**: select "WeCom"
   - **Integration Mode**: select "Webhook"
   - **Output Mode**: select "Streaming"
   - **Corp ID**: the enterprise ID
   - **Agent Secret**: the app's Secret
   - **Token**: custom or randomly generated (record it)
   - **EncodingAESKey**: custom or randomly generated (record it)
   - **Corp Agent ID**: the app's AgentID (integer)
3. After saving, the channel card will display a **callback address** in the format `https://your-domain/api/v1/im/callback/{channel_id}`
4. Copy that callback address

**Step 3: Configure WeCom to Receive Messages**

1. On the app details page → **Receive Messages** → **Set API Reception**
2. Fill in:
   - **URL**: paste the callback address copied in the previous step
   - **Token**: enter the Token you set in WeKnora
   - **EncodingAESKey**: enter the EncodingAESKey you set in WeKnora
3. Click Save — WeCom will send a GET verification request, and WeKnora will respond automatically

**Step 4: Configure Trusted Domains (Optional)**

If you need to use this in group chats, add trusted domains under the app details page → **Web Authorization and JS-SDK**.

---

### Feishu Integration

Feishu also offers two modes; the WebSocket mode has simpler configuration.

#### Method 1: WebSocket Mode (Recommended)

> No public domain required, no event encryption configuration needed.

**Step 1: Create a Feishu App**

1. Log in to the [Feishu Open Platform](https://open.feishu.cn/) → **Developer Console** → **Create Custom App**
2. On the **Credentials & Basic Info** page, obtain:
   - **App ID**
   - **App Secret**

**Step 2: Enable Permissions and Events**

1. **Add App Capabilities**: on the app details page → **Add App Capabilities** → add the **Bot** capability
2. **Configure Permissions**: in **Permission Management**, search for and enable the following permissions: your app → Permission Management → Batch Import, paste the JSON below (content unchanged from the original):
```json
{
  "scopes": {
    "tenant": [
      "aily:file:read",
      "aily:file:write",
      "application:application.app_message_stats.overview:readonly",
      "application:application:self_manage",
      "application:bot.menu:write",
      "cardkit:card:write",
      "contact:user.employee_id:readonly",
      "corehr:file:download",
      "docs:document.content:read",
      "event:ip_list",
      "im:chat",
      "im:chat.access_event.bot_p2p_chat:read",
      "im:chat.members:bot_access",
      "im:message",
      "im:message.group_at_msg:readonly",
      "im:message.group_msg",
      "im:message.p2p_msg:readonly",
      "im:message:readonly",
      "im:message:send_as_bot",
      "im:resource",
      "sheets:spreadsheet",
      "wiki:wiki:readonly"
    ],
    "user": [
      "aily:file:read",
      "aily:file:write",
      "im:chat.access_event.bot_p2p_chat:read"
    ]
  }
}
```
3. **Configure Event Subscriptions**:
   - Under **Events & Callbacks** → **Event Configuration**, choose the request method **Use long connection to receive events**
   - Add the event `im.message.receive_v1` (receive messages)

**Step 3: Publish the App**

Create a version and submit it for review under **Version Management & Release**. Once approved, users will be able to interact with the bot.

**Step 4: Add an IM Channel in WeKnora**

1. Go to the Agent editor → **IM Integration** → **Add Channel**
2. Fill in the configuration:
   - **Platform**: select "Feishu"
   - **Integration Mode**: select "WebSocket"
   - **Output Mode**: select "Streaming" (requires the cardkit:card permission to be enabled)
   - **App ID**: enter the App ID obtained from Feishu
   - **App Secret**: enter the App Secret obtained from Feishu
3. Save

The following log entry after startup indicates a successful connection:

```
[IM] Feishu WebSocket connecting (app_id=xxx)...
```

---

#### Method 2: Webhook Mode

> Requires a publicly reachable callback address.

**Prerequisite steps** are the same as above (create the app, enable permissions), plus:

**Step 1: Add an IM Channel in WeKnora**

1. Go to the Agent editor → **IM Integration** → **Add Channel**
2. Fill in the configuration:
   - **Platform**: select "Feishu"
   - **Integration Mode**: select "Webhook"
   - **App ID** / **App Secret**
   - **Verification Token**: obtained from the Feishu event subscription page
   - **Encrypt Key**: obtained from the Feishu event subscription page
3. After saving, copy the **callback address** shown on the channel card

**Step 2: Configure Feishu Event Subscription**

1. Under **Events & Callbacks** → **Event Configuration**, choose the request method **Send events to a developer server**
2. **Request URL**: paste the callback address copied from WeKnora
3. Add the event `im.message.receive_v1`
4. When you click Save, Feishu will send a URL verification request (challenge), which WeKnora will respond to automatically

---

### Lark Integration

Lark is the international version of Feishu. Both are the same product deployed on two mutually isolated clouds — **the IM API, event structures, credential fields, and integration modes are identical**, so WeKnora shares the same adapter code between Lark and Feishu.

However, "identical code" does not mean "identical configuration." There are three differences when integrating:

| | Feishu | Lark |
|---|---|---|
| Open Platform | <https://open.feishu.cn/> | <https://open.larksuite.com/> |
| **Platform** option when adding a channel | "Feishu" | "Lark (International Feishu)" |
| Permission list | JSON in the [Feishu Integration](#feishu-integration) section | JSON in the [Lark Permission Configuration](#lark-permission-configuration) section (**cannot reuse the Feishu one**) |

The steps for creating the app, adding the bot capability, subscribing to the `im.message.receive_v1` event, publishing a version, and filling in App ID / App Secret are the same as in the [Feishu Integration](#feishu-integration) section. **Only the permission configuration differs — see below.**

#### Lark Permission Configuration

In the Lark Open Platform, under **Permission Management → Batch Import**, paste the JSON below, which covers both the IM bot and the Knowledge Base (Wiki) data source functionality:

```json
{
  "scopes": {
    "tenant": [
      "im:message",
      "im:message:send_as_bot",
      "im:resource",
      "im:message.p2p_msg:readonly",
      "im:message.group_at_msg:readonly",
      "cardkit:card:write",
      "wiki:wiki:readonly",
      "drive:export:readonly",
      "drive:drive:readonly",
      "docx:document:readonly"
    ],
    "user": []
  }
}
```

> Do not reuse the JSON from the [Feishu Integration](#feishu-integration) section: `aily:file:*` and `corehr:file:download`
> do not exist in Lark, and the entire import will fail.

> **Note**: Feishu apps and Lark apps are not interchangeable. An app created at open.feishu.cn cannot be used for a Lark channel,
> and vice versa — credentials are only valid on the cloud where they were created; cross-cloud calls will fail authentication.

The following log entry after WebSocket mode starts up indicates a successful connection:

```
[IM] Lark WebSocket connecting (app_id=xxx)...
```

When the same App ID is registered separately on both clouds, WeKnora distinguishes them via `platform:app_id` (e.g. `feishu:cli_xxx`
and `lark:cli_xxx`), so the two channels can coexist without their sessions interfering with each other.

---

### Slack Integration

Slack offers two integration modes; the WebSocket (Socket Mode) mode is recommended since it requires no public domain.

#### Method 1: WebSocket Mode (Socket Mode, Recommended)

> No public domain required — suitable for quick validation and intranet deployments.

**Step 1: Create a Slack App**

1. Log in to [Slack API](https://api.slack.com/apps) → **Create New App** → **From scratch**
2. Fill in the App Name and select the Workspace to install it to.

**Step 2: Generate an App-Level Token**

1. On the app details page's left navigation, select **Basic Information**.
2. Scroll to the **App-Level Tokens** section and click **Generate Token and Scopes**.
3. Fill in a Token Name and add the `connections:write` scope.
4. Click Generate and copy the generated Token (starting with `xapp-`) — this is your **App Token**.

**Step 3: Enable Socket Mode**

1. In the left navigation, select **Socket Mode**.
2. Turn on the **Enable Socket Mode** toggle.

**Step 4: Configure Event Subscriptions**

1. In the left navigation, select **Event Subscriptions**.
2. Turn on the **Enable Events** toggle.
3. Expand **Subscribe to bot events** and add the following events:
   - `app_mention` (@ mentioning the bot in a channel)
   - `message.channels` (channel messages)
   - `message.groups` (private channel messages)
   - `message.im` (direct messages)
   - `message.mpim` (multi-person direct messages)
4. Click **Save Changes**.

**Step 5: Configure Permissions (OAuth & Permissions)**

1. In the left navigation, select **OAuth & Permissions**.
2. Scroll to **Scopes** → **Bot Token Scopes**, and make sure the following permissions are included (these are usually added automatically when adding events):
   - `app_mentions:read`
   - `channels:history`
   - `chat:write`
   - `groups:history`
   - `im:history`
   - `mpim:history`
   - `files:read` (for receiving files)
3. Scroll to the top and click **Install to Workspace**.
4. After authorizing, copy the **Bot User OAuth Token** (starting with `xoxb-`) — this is your **Bot Token**.

**Step 6: Add an IM Channel in WeKnora**

1. Go to the Agent editor → **IM Integration** → **Add Channel**
2. Fill in the configuration:
   - **Platform**: select "Slack"
   - **Integration Mode**: select "WebSocket"
   - **Output Mode**: select "Streaming"
   - **App Token**: enter the token starting with `xapp-`
   - **Bot Token**: enter the token starting with `xoxb-`
3. Save

The following log entry after startup indicates a successful connection:

```
[IM] Slack WebSocket connecting...
```

---

#### Method 2: Webhook Mode (Events API)

> Requires a publicly reachable callback address.

**Step 1: Create a Slack App and Obtain Credentials**

1. Log in to [Slack API](https://api.slack.com/apps) and create an app.
2. On the **Basic Information** page, scroll to the **App Credentials** section and copy the **Signing Secret**.
3. On the **OAuth & Permissions** page, configure the Bot Token Scopes (as above), install to Workspace, and copy the **Bot User OAuth Token** (Bot Token).

**Step 2: Add an IM Channel in WeKnora**

1. Go to the Agent editor → **IM Integration** → **Add Channel**
2. Fill in the configuration:
   - **Platform**: select "Slack"
   - **Integration Mode**: select "Webhook"
   - **Bot Token**: enter the token starting with `xoxb-`
   - **Signing Secret**: enter the Signing Secret
3. After saving, copy the **callback address** shown on the channel card.

**Step 3: Configure Event Subscriptions**

1. On the Slack App settings page's left navigation, select **Event Subscriptions**.
2. Turn on the **Enable Events** toggle.
3. Paste the callback address copied from WeKnora into the **Request URL** field. Slack will send a challenge request, which WeKnora will automatically respond to and verify.
4. Expand **Subscribe to bot events** and add the required events (as above).
5. Click **Save Changes**.

---

### Telegram Integration

Telegram offers two integration modes; the WebSocket (long polling) mode is recommended since it requires no public domain.

#### Method 1: WebSocket Mode (Long Polling, Recommended)

> No public domain required — suitable for quick validation and intranet deployments.

**Step 1: Create a Telegram Bot**

1. Search for [@BotFather](https://t.me/BotFather) in Telegram and start a conversation
2. Send `/newbot` and follow the prompts to fill in the bot name and username
3. Once created, obtain the **Bot Token** (in the format `123456:ABC-DEF1234ghIkl-zyx57W2v1u123ew11`)

**Step 2: Add an IM Channel in WeKnora**

1. Go to the Agent editor → **IM Integration** → **Add Channel**
2. Fill in the configuration:
   - **Platform**: select "Telegram"
   - **Integration Mode**: select "WebSocket"
   - **Output Mode**: select "Streaming" (recommended)
   - **Bot Token**: enter the token obtained from BotFather
3. Save

The following log entry after startup indicates a successful connection:

```
[IM] Telegram long polling connecting...
```

At this point, sending a message to the bot in Telegram will get an AI reply.

---

#### Method 2: Webhook Mode

> Requires a publicly reachable callback address (HTTPS).

**Step 1: Create a Telegram Bot and Obtain Credentials**

Same as above — create the bot via BotFather and obtain the Bot Token.

**Step 2: Add an IM Channel in WeKnora**

1. Go to the Agent editor → **IM Integration** → **Add Channel**
2. Fill in the configuration:
   - **Platform**: select "Telegram"
   - **Integration Mode**: select "Webhook"
   - **Bot Token**: enter the Bot Token
   - **Secret Token** (optional): a custom secret used to verify the `X-Telegram-Bot-Api-Secret-Token` header of callback requests
3. After saving, copy the **callback address** shown on the channel card

**Step 3: Configure the Webhook**

Set the webhook via the Telegram Bot API:

```bash
curl -X POST "https://api.telegram.org/bot<YOUR_BOT_TOKEN>/setWebhook" \
  -H "Content-Type: application/json" \
  -d '{"url": "<YOUR_CALLBACK_URL>", "secret_token": "<YOUR_SECRET_TOKEN>"}'
```

> Note: Telegram webhooks must use HTTPS.

---

### DingTalk Integration

DingTalk offers two integration modes; the Stream mode (WebSocket) is recommended since it requires no public domain.

#### Method 1: WebSocket Mode (Stream, Recommended)

> No public domain required — suitable for quick validation and intranet deployments.

**Step 1: Create a DingTalk Bot**

1. Log in to the [DingTalk Open Platform](https://open-dev.dingtalk.com/) → **App Development** → **Create App**
2. On the app details page → **Add App Capabilities** → add the **Bot** capability
3. On the **Credentials & Basic Info** page, obtain:
   - **Client ID** (AppKey)
   - **Client Secret** (AppSecret)

**Step 2: Configure the Bot**

1. On the app details page → **Bot** → set the message reception mode to **Stream Mode**
2. (Optional) If you want the streaming AI card effect, create an **AI Card Template** under **Interactive Cards** and record the **Card Template ID**

**Step 3: Add an IM Channel in WeKnora**

1. Go to the Agent editor → **IM Integration** → **Add Channel**
2. Fill in the configuration:
   - **Platform**: select "DingTalk"
   - **Integration Mode**: select "WebSocket"
   - **Output Mode**: select "Streaming" (recommended)
   - **Client ID**: enter the AppKey
   - **Client Secret**: enter the AppSecret
   - **Card Template ID** (optional): enter the AI card template ID; once enabled, streaming replies will be displayed as AI cards
3. Save

The following log entry after startup indicates a successful connection:

```
[IM] DingTalk Stream connecting...
```

> **About AI Cards**: once the Card Template ID is configured, streaming replies will show a real-time typing effect via DingTalk AI cards; without it, streaming content will be sent all at once after completion.

---

#### Method 2: Webhook Mode

> Requires a publicly reachable callback address.

**Step 1: Create a DingTalk Bot and Obtain Credentials**

Same as above — create the app and obtain the Client ID and Client Secret.

**Step 2: Add an IM Channel in WeKnora**

1. Go to the Agent editor → **IM Integration** → **Add Channel**
2. Fill in the configuration:
   - **Platform**: select "DingTalk"
   - **Integration Mode**: select "Webhook"
   - **Client ID**: enter the AppKey
   - **Client Secret**: enter the AppSecret
   - **Card Template ID** (optional): same as above
3. After saving, copy the **callback address** shown on the channel card

**Step 3: Configure DingTalk Event Subscription**

1. On the app details page → **Bot** → set the message reception mode to **HTTP Mode**
2. **Message Receiving Address**: paste the callback address copied from WeKnora
3. DingTalk will verify callback requests using an HmacSHA256 signature

---

### Mattermost Integration

Mattermost is a **self-hosted deployment**, and currently only supports **Webhook** mode: an **Outgoing Webhook** POSTs user messages to WeKnora, and the bot sends channel/thread replies back via the **REST API v4**. Similar to the Slack Events API, the callback must return `200` quickly, with the actual answer delivered asynchronously via a Bot Token API call.

> A publicly or internally reachable callback URL is required (the Mattermost server must be able to reach WeKnora's `/api/v1/im/callback/{channel_id}`). If WeKnora is on an internal network, configure [Trusted Internal Connections](https://docs.mattermost.com/configure/environment-configuration-settings.html) in Mattermost, or add the callback address to the allow list.

#### Step 1: Create a Bot Account and Obtain a Token

1. Create a dedicated bot account in Mattermost (or use a personal account's **Personal Access Token** — not recommended for production).
2. Grant the bot the permissions needed to post in the target channel/team, read files, etc. (if you need to save user-uploaded files to the knowledge base, it needs to be able to download attachments).
3. Under **Profile → Security → Personal Access Tokens** (or the bot's settings), generate a token and save it as the **Bot Token**.

#### Step 2: Create an Outgoing Webhook

1. Open **Product Menu → Integrations → Outgoing Webhooks** (if not visible, a system administrator needs to enable Outgoing Webhooks under **System Console → Integrations**).
2. Click **Add Outgoing Webhook** and fill in a title and description.
3. It is recommended to set **Content Type** to **application/json** (`application/x-www-form-urlencoded` is also supported — WeKnora can parse both).
4. Select the trigger channel, or configure **trigger words** (see Mattermost's documentation for how channel and trigger word rules combine).
5. Leave **Callback URLs** blank or use a placeholder for now; you'll get the real address after creating the channel in WeKnora.
6. After saving, copy the **Token** shown on the page (the outgoing webhook secret) — this is the **Outgoing Webhook Token**.

#### Step 3: Add an IM Channel in WeKnora

1. Go to the Agent editor → **IM Integration** → **Add Channel**.
2. Fill in the configuration:
   - **Platform**: select "Mattermost"
   - **Integration Mode**: fixed to **Webhook** (the WebSocket option is disabled once Mattermost is selected)
   - **Output Mode**: Streaming / Full (streaming is implemented by editing posts)
   - **Site URL**: the Mattermost site's root address, e.g. `https://mattermost.example.com` (no trailing `/`)
   - **Bot Token**: the token generated in the previous step
   - **Outgoing Webhook Token**: the token copied from the outgoing webhook page (used to validate callbacks, and also participates in `bot_identity` deduplication)
   - **Bot User ID** (optional): the bot's user ID in Mattermost; if filled in, callbacks triggered by the bot itself can be ignored, avoiding a self-reply loop
3. After saving, copy the **callback address** on the channel card, go back to the Mattermost outgoing webhook configuration, set the **Callback URLs** to that address, and save.

#### Step 4: Verify

Send a message in the bound channel that triggers the outgoing webhook, and you should receive a reply from WeKnora; by default the reply appears in the **thread of the triggering post** (aligned via Mattermost's `root_id` and the triggering `post_id`).

**Reference documentation:** [Outgoing webhooks](https://developers.mattermost.com/integrate/webhooks/outgoing/)

---

## Frontend Management

IM channels are managed in the **IM Integration** tab of the Agent editor (visible only in edit mode, not shown when creating a new Agent).

### Channel List

Each channel is displayed as a card containing:
- **Platform badge**: WeCom (green) / Feishu (teal) / Lark (blue) / Slack (purple) / Telegram (blue) / DingTalk (blue) / Mattermost (blue)
- **Channel name**: user-defined
- **Integration mode**: WebSocket / Webhook
- **Output mode**: Streaming / Full
- **Enable toggle**: instantly enable/disable the channel
- **Callback address**: shown in Webhook mode, one-click copy
- **Edit/Delete**: manage the channel configuration

### Channel Operations

- **Add Channel**: select platform → fill in credentials → select mode → save
- **Edit Channel**: name, mode, output mode, and credentials can be modified (platform cannot be changed)
- **Enable/Disable**: toggle instantly; disabled channels will not process messages
- **Delete Channel**: cannot be undone once deleted

---

## Architecture Overview

```
┌──────────────────────────────────────────────────────────────────────────────┐
│                              IM Integration Architecture                     │
│                                                                              │
│   ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐      │
│   │  WeCom   │  │  Feishu  │  │  Slack   │  │ Telegram │  │ DingTalk │  IM  │
│   └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘ layer│
│        │ WH/WS       │ WH/WS      │ WH/WS       │ WH/LP       │ WH/Stream  │
│   ─────┼─────────────┼────────────┼─────────────┼─────────────┼──────────   │
│        ▼             ▼            ▼             ▼             ▼              │
│   ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐      │
│   │  WeCom   │  │  Feishu  │  │  Slack   │  │ Telegram │  │ DingTalk │ Adap-│
│   │ Adapter  │  │ Adapter  │  │ Adapter  │  │ Adapter  │  │ Adapter  │ ter  │
│   └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘ layer│
│        │             │             │              │             │             │
│   ─────┼─────────────┼─────────────┼──────────────┼─────────────┼────────   │
│        └─────────────┼─────────────┼──────────────┘─────────────┘            │
│                      │                                                       │
│                 ┌────┴────────────┐                                            │
│                 │ Mattermost      │ Webhook-only                               │
│                 └────────┬────────┘                                            │
│                          ▼                                                    │
│                 ┌────────────────┐                                             │
│                 │ Mattermost     │                                             │
│                 │ Adapter        │                                             │
│                 └────────┬────────┘                                             │
│                          │                                                     │
│        ──────────────────┴────────────────────────────────────────────────────│
│                        ▼                                                     │
│   ┌──────────────────────────────────┐                                       │
│   │         im.Service               │     Service orchestration layer       │
│   │                                  │     · IM channel management (CRUD)    │
│   │  ┌────────────────────────────┐  │     · Adapter Factory (dynamic       │
│   │  │ CommandRegistry            │  │       creation)                       │
│   │  │ qaQueue (Worker Pool)      │  │     · Slash command dispatch          │
│   │  │ rateLimiter (sliding win.) │  │     · QA queue scheduling (bounded,   │
│   │  │ processedMsgs (dedup)      │  │       async)                          │
│   │  │ inflight (cancel tracking) │  │     · Sliding window rate limiting    │
│   │  └────────────────────────────┘  │     · Message deduplication           │
│   └──────────────┬───────────────────┘       (MessageID + TTL)              │
│                  │                          · Session mapping                │
│                  │                            (ChannelSession)                │
│                  │                          · Streaming/full-output routing  │
│   ───────────────┼───────────────────────────────────────────────────────    │
│                  ▼                                                           │
│   ┌──────────────────────────────────────┐                                   │
│   │     WeKnora Core (QA Pipeline)       │     Core layer                   │
│   │   SessionService · MessageService    │                                   │
│   │   TenantService  · AgentService      │                                   │
│   │   KnowledgeService (file saving)     │                                   │
│   └──────────────────────────────────────┘                                   │
└──────────────────────────────────────────────────────────────────────────────┘
```

**Design patterns:**

| Pattern | Purpose |
|------|------|
| Adapter Pattern | Unifies differences across IM platforms; each platform implements the `im.Adapter` interface |
| Factory Pattern | `AdapterFactory` dynamically creates Adapter instances from database channel configuration |
| Strategy Pattern | `StreamSender`, `FileDownloader` are optional interfaces, implemented as needed |
| Command Pattern | The `Command` interface + `CommandRegistry` implement a pluggable slash command system |
| Producer-Consumer | The `qaQueue` bounded queue + Worker Pool decouples message reception from QA execution |
| Event-Driven | `EventBus` decouples the QA pipeline from IM output, enabling real-time chunk pushes |

---

## Data Model

### im_channels Table

IM channel configuration is stored in the `im_channels` table, bound to an Agent:

```sql
CREATE TABLE im_channels (
    id                VARCHAR(36) PRIMARY KEY,
    tenant_id         BIGINT NOT NULL,
    agent_id          VARCHAR(36) NOT NULL,       -- bound Agent ID
    platform          VARCHAR(20) NOT NULL,       -- 'wecom' | 'feishu' | 'lark' | 'slack' | 'telegram' | 'dingtalk' | 'mattermost'
    name              VARCHAR(255) NOT NULL DEFAULT '',
    enabled           BOOLEAN NOT NULL DEFAULT true,
    mode              VARCHAR(20) NOT NULL DEFAULT 'websocket',  -- 'webhook' | 'websocket'
    output_mode       VARCHAR(20) NOT NULL DEFAULT 'stream',     -- 'stream' | 'full'
    knowledge_base_id VARCHAR(36),                -- optional; bind a knowledge base to receive file messages
    bot_identity      VARCHAR(255),               -- computed field, prevents duplicate bots
    credentials       JSONB NOT NULL DEFAULT '{}',               -- platform credentials
    created_at        TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at        TIMESTAMPTZ
);
```

**`credentials` field structure:**

| Platform | Mode | Fields |
|------|------|------|
| WeCom | WebSocket | `bot_id`, `bot_secret` |
| WeCom | Webhook | `corp_id`, `agent_secret`, `token`, `encoding_aes_key`, `corp_agent_id` |
| Feishu / Lark | WebSocket | `app_id`, `app_secret` |
| Feishu / Lark | Webhook | `app_id`, `app_secret`, `verification_token`, `encrypt_key` |
| Slack | WebSocket | `app_token`, `bot_token` |
| Slack | Webhook | `bot_token`, `signing_secret` |
| Telegram | WebSocket | `bot_token` |
| Telegram | Webhook | `bot_token`, `secret_token` (optional) |
| DingTalk | WebSocket | `client_id`, `client_secret`, `card_template_id` (optional) |
| DingTalk | Webhook | `client_id`, `client_secret`, `card_template_id` (optional) |
| Mattermost | Webhook (only supported mode) | `site_url`, `bot_token`, `outgoing_token` (required); `bot_user_id` (optional, filters the bot's own messages) |

The `mode` of a `mattermost` channel is fixed to `webhook` in the database (if not specified at creation time, the server and model hooks default it to `webhook`). `bot_identity` takes the form `mattermost:wh:{outgoing_token}`, used to prevent the same outgoing webhook from being bound to multiple channels.

### im_channel_sessions Table

Maps user sessions from an IM channel to WeKnora sessions:

```
(im_channel_id, Platform, UserID, ChatID, TenantID)  →  SessionID
```

Created automatically on first interaction; subsequent messages reuse the same session. The `/clear` command soft-deletes the session record, and it is recreated on the next message.

---

## API Endpoints

### IM Channel Management API (Authentication Required)

| Method | Path | Description |
|------|------|------|
| POST | `/api/v1/agents/:id/im-channels` | Create an IM channel |
| GET | `/api/v1/agents/:id/im-channels` | List all IM channels for an Agent |
| PUT | `/api/v1/im-channels/:id` | Update an IM channel |
| DELETE | `/api/v1/im-channels/:id` | Delete an IM channel |
| POST | `/api/v1/im-channels/:id/toggle` | Enable/disable an IM channel |

### IM Callback Endpoint (No Authentication, Platform Signature Verification)

| Method | Path | Description |
|------|------|------|
| GET/POST | `/api/v1/im/callback/:channel_id` | Generic callback (automatically routed to the corresponding Adapter based on channel_id) |

> In Webhook mode, each channel has a unique callback address `/api/v1/im/callback/{channel_id}`, which can be copied with one click from the channel card in the frontend. The callback route is registered **before** the authentication middleware, and is protected by platform signature verification instead.

---

## Core Concepts

### IMChannel

Each IM channel represents a binding between an IM platform bot and a WeKnora Agent. An Agent can be bound to multiple channels (e.g. connecting WeCom, Feishu, Slack, and Mattermost simultaneously), and multiple channels can also be created for the same platform (e.g. different WeCom bots).

A channel has a computed field, `BotIdentity`, derived from the platform type, mode, and core credentials, used to prevent the same bot from being created twice.

When a channel starts, the Service dynamically creates the corresponding Adapter instance via the `AdapterFactory`, based on the platform type and credentials.

### IncomingMessage — Unified Inbound Message

After decryption and parsing, messages from all platforms are normalized into an `IncomingMessage`, which smooths over platform differences:

```go
type IncomingMessage struct {
    Platform    Platform          // "wecom" | "feishu" | "lark" | "slack" | "telegram" | "dingtalk" | "mattermost"
    MessageType MessageType       // "text" | "file" | "image"
    UserID      string            // platform user identifier
    UserName    string            // display name (optional)
    ChatID      string            // group chat ID (empty for direct messages)
    ChatType    ChatType          // "direct" | "group"
    Content     string            // plain text content
    MessageID   string            // platform message ID (for deduplication)
    FileKey     string            // file identifier (file/image messages)
    FileName    string            // file name (file/image messages)
    FileSize    int64             // file size (bytes)
    Extra       map[string]string // platform-specific fields (e.g. req_id, aes_key)
}
```

### ReplyMessage — Unified Outbound Reply

```go
type ReplyMessage struct {
    Content    string            // Markdown text
    IsStreaming bool             // whether this is a streaming chunk
    IsFinal    bool             // whether this is the final chunk
    Extra      map[string]string // platform-specific fields
}
```

### ChannelSession — Session Mapping

Maps an IM channel (channel ID + user + group chat) to a WeKnora session, enabling conversational context continuity. Created automatically on first interaction; subsequent messages reuse the same session. Concurrent creation is handled via a unique constraint plus a fallback query. Stored in the `im_channel_sessions` table.

---

## Message Processing Flow

### Full Message Processing Flow

```
User sends a message in IM
        │
        ▼
┌─ HTTP Handler / WebSocket callback ─────────────┐
│  1. Look up channel configuration by channel_id  │
│  2. Get the corresponding Adapter                │
│  3. Signature verification (VerifyCallback)      │
│  4. URL verification handling                    │
│     (HandleURLVerification)                      │
│  5. Decrypt + parse → IncomingMessage             │
│     (ParseCallback)                               │
│  6. Immediately return HTTP 200 (async processing)│
└──────────────────────────┬──────────────────────-┘
                           │ goroutine
                           ▼
┌─ im.Service.HandleMessage ──────────────────────┐
│  1. Deduplication check (MessageID, 5-min TTL)   │
│  2. Content length validation (≤ 4096 runes,     │
│     truncated if exceeded)                       │
│  3. Slash command detection → dispatched to      │
│     CommandRegistry if matched                   │
│  4. Rate limit check (sliding window, 10/60s)    │
│  5. Get agent_id, tenant_id from channel config   │
│  6. Resolve/create ChannelSession                 │
│  7. Get the WeKnora Session                       │
│  8. Load Agent configuration (knowledge base,     │
│     model, etc.)                                  │
│  9. File message? → download and save to          │
│     knowledge base                                │
│ 10. Submit to qaQueue (bounded queue, async        │
│     execution)                                     │
└───────────┬─────────────────────────────────────┘
            │
            ▼
┌─ qaQueue Worker ────────────────────────────────┐
│  Pulls the request from the queue, records        │
│  inflight, determines streaming/full mode          │
└───────────┬─────────────────────┬───────────────┘
            │                     │
    Streaming mode ▼        Full mode ▼
┌────────────────────┐  ┌─────────────────────┐
│ handleMessageStream│  │ runQA (blocks until  │
│                    │  │ the full answer is    │
│ · StartStream      │  │ collected, then sends │
│ · EventBus         │  │ it all at once)       │
│   subscription     │  └─────────────────────┘
│ · 300ms batched     │
│   flushing          │
│ · Tool event display│
│ · UpdateStreamContent│
│ · FinalizeStream      │
│ · EndStream           │
└────────────────────┘
            │
            ▼
    Message persistence (user + assistant)
```

### Channel Lifecycle

```
Channel created/updated (frontend UI)
        │
        ▼
┌─ im.Service ──────────────────────────┐
│  1. Save the channel configuration to  │
│     the database                       │
│  2. If the channel is enabled:         │
│     a. AdapterFactory creates the      │
│        Adapter                         │
│     b. WebSocket mode: establish a     │
│        long connection                 │
│     c. Webhook mode: register the      │
│        callback handler                │
│  3. Maintain the channels map          │
│     (channel_id →                      │
│     channelState{Channel, Adapter})    │
└────────────────────────────────────────┘

On service startup:
  LoadAndStartChannels() → load all enabled channels from the DB → StartChannel() for each one

On channel disable/delete:
  StopChannel() → cancel the Adapter context → remove from the map
```

---

## Interface Definitions

### im.Adapter — Platform Adapter (Must Be Implemented)

```go
type Adapter interface {
    Platform() Platform
    VerifyCallback(c *gin.Context) error
    ParseCallback(c *gin.Context) (*IncomingMessage, error)
    SendReply(ctx context.Context, incoming *IncomingMessage, reply *ReplyMessage) error
    HandleURLVerification(c *gin.Context) bool
}
```

| Method | Responsibility |
|------|------|
| `Platform()` | Returns the platform identifier, used for routing and registration |
| `VerifyCallback()` | Verifies the signature/token of the callback request |
| `ParseCallback()` | Decrypts and parses the callback into an `IncomingMessage`; returns `nil` for non-message events |
| `SendReply()` | Sends a full reply via the platform API |
| `HandleURLVerification()` | Handles the platform's initial URL verification (called on first-time configuration) |

### im.StreamSender — Streaming Push (Optional)

```go
type StreamSender interface {
    StartStream(ctx context.Context, incoming *IncomingMessage) (streamID string, err error)
    UpdateStreamContent(ctx context.Context, incoming *IncomingMessage, streamID string, fullContent string) error
    FinalizeStream(ctx context.Context, incoming *IncomingMessage, streamID string, finalContent string) error
    EndStream(ctx context.Context, incoming *IncomingMessage, streamID string) error
}
```

- `UpdateStreamContent`: during streaming, replaces the message with the **currently visible full text** (replace semantics, not incremental appending)
- `FinalizeStream`: the final replacement before ending, typically the final display after collapsing thinking/tool progress (usually the plain answer)

Once this interface is implemented, the Service automatically routes to streaming mode. Setting the channel configuration `output_mode: "full"` forces this off.

### im.FileDownloader — File Download (Optional)

```go
type FileDownloader interface {
    DownloadFile(ctx context.Context, msg *IncomingMessage) (io.ReadCloser, string, error)
}
```

Once this interface is implemented, when a user sends a file/image message and the channel is configured with a `knowledge_base_id`, the Service automatically downloads the file and saves it to the specified knowledge base.

### im.AdapterFactory — Adapter Factory

```go
type AdapterFactory func(ctx context.Context, channel *IMChannel, msgHandler func(*IncomingMessage)) (Adapter, CancelFunc, error)
```

Each platform registers a factory function; the Service calls the factory to create an Adapter instance when starting a channel. The factory function decides which type of Adapter to create based on the channel's `mode` and `credentials`.

---

## Platform Adapter Details

### WeCom

Offers two connection modes, corresponding to two adapter implementations:

#### Webhook Mode (`WebhookAdapter`)

Suitable for **custom applications**, requires a publicly accessible callback address.

```
WeCom server ──HTTP POST──▶ /api/v1/im/callback/{channel_id}
                                      │
                              Decrypt (AES-256-CBC)
                              Parse XML → IncomingMessage
                                      │
                              After processing, call the WeCom REST API to reply
```

- **Encryption scheme:** AES-256-CBC, with the Key obtained by Base64-decoding `encoding_aes_key` (32 bytes); the IV is the first 16 bytes of the Key
- **Message format:** `random(16) + msg_len(4) + message + corp_id`, PKCS#7 padding
- **Signature verification:** SHA-1(`sort([token, timestamp, nonce, encrypt])`), constant-time comparison
- **Message types:** supports `text` (text) and `image` (image; PicUrl direct download or MediaId temporary media API)
- **Group reply:** first tries the `appchat/send` group chat API, falling back to a direct private message on failure
- **Reply method:** sends Markdown messages via the `/cgi-bin/message/send` interface

#### WebSocket Mode (`WSAdapter` + `LongConnClient`)

Suitable for **smart customer service bots**, requires no public domain — the client actively establishes a WebSocket long connection.

```
LongConnClient ══WebSocket══▶ wss://openws.work.weixin.qq.com
       │
  1. Send aibot_subscribe (bot_id + secret)
  2. Receive aibot_msg_callback message frames
  3. Reply via aibot_respond_msg
  4. Heartbeat every 30s to keep alive (ping/pong)
  5. Automatic reconnect on disconnect (exponential backoff 1s → 30s)
```

- **Authentication:** Bot ID + Bot Secret
- **Message types:** `text` (text), `image` (image), `file` (file), `voice` (voice, already converted to text server-side), `mixed` (mixed, text + image), `event` (server event)
- **File decryption:** attachments are decrypted with a per-message unique AES-256-CBC key (IV is the first 16 bytes of the key)
- **Streaming reply:** the accumulated full text is sent over WebSocket frames, with `finish=true` marking the end
- **Fault tolerance:** exponential backoff reconnection (base 1s, cap 30s), read timeout = 3 × heartbeat interval (90s)

#### Source Files

| File | Responsibility |
|------|------|
| `internal/im/wecom/webhook_adapter.go` | Webhook mode: callback decryption, signature verification, REST API replies, group sending, token caching, file downloads |
| `internal/im/wecom/ws_adapter.go` | WebSocket mode adapter shell, proxies to `LongConnClient` |
| `internal/im/wecom/longconn.go` | WebSocket client: connection management, heartbeat, frame protocol, automatic reconnection, multi-message-type parsing, file decryption |

---

### Feishu and Lark

The unified adapter supports both Webhook and WebSocket modes, and natively implements the `StreamSender` and `FileDownloader` interfaces.

#### One Adapter Shared Across Two Clouds

Feishu and Lark are the same product deployed on two isolated clouds, with identical APIs and event structures, so the `internal/im/feishu`
package serves both platforms, with `Region` determining which cloud to connect to:

```go
// internal/im/feishu/region.go
type Region struct {
    Platform           im.Platform  // "feishu" | "lark", written into IncomingMessage
    OpenBaseURL        string       // https://open.feishu.cn | https://open.larksuite.com
    Label              string       // log prefix [Feishu] / [Lark]
    ThinkingText       string       // streaming card placeholder text (Chinese / English)
    ImageFallbackLabel string       // fallback link text when image upload fails
}

var (
    RegionFeishu = Region{Platform: im.PlatformFeishu, OpenBaseURL: "https://open.feishu.cn", ...}
    RegionLark   = Region{Platform: im.PlatformLark,   OpenBaseURL: "https://open.larksuite.com", ...}
)
```

Registered as two platforms in the container:

```go
// internal/container/container.go
imService.RegisterAdapterFactory("feishu", feishu.NewFactory(feishu.RegionFeishu))
imService.RegisterAdapterFactory("lark", feishu.NewFactory(feishu.RegionLark))
```

All adapter API calls are concatenated with `region.OpenBaseURL` via `a.api(path, args...)`; in WebSocket mode,
`larkws.WithDomain(region.OpenBaseURL)` points to the corresponding cloud (the SDK connects to open.feishu.cn by default; Lark will fail authentication if not specified).

> **Cross-cloud resources are not interchangeable**: `tenant_access_token`, `image_key`, `file_key`, and `card_id` are all issued by a single
> app on a single cloud; using them across clouds or across apps will be rejected. The image upload cache is therefore partitioned with an `app_id` prefix.

#### APIs and Permissions Actually Used by the IM Adapter

The table below explains which permissions the adapter **actually** depends on. The permission identifiers in the table have been cross-checked
line by line against the official Lark API documentation; Feishu uses the same names.

> **The permission lists for the two clouds are not equivalent.** Identical APIs and event structures do not mean identical permission
> catalogs — some permissions that exist on the Feishu Open Platform do not exist on the Lark Open Platform. So refer to each cloud's
> respective configuration list separately — [Feishu Integration](#feishu-integration) and
> [Lark Permission Configuration](#lark-permission-configuration) — and do not copy-paste across clouds.
>
> The JSON in the Feishu section also covers the **Feishu data source connector** (Wiki sync), so it includes
> `wiki:wiki:readonly`, `docs:document.content:read`, `sheets:spreadsheet`, and other permissions not used by IM.

| Adapter call | Purpose | Required permission (any one suffices) |
|---|---|---|
| `POST /auth/v3/tenant_access_token/internal` | Exchange for a tenant access token | No permission required (uses App ID / Secret) |
| `POST /im/v1/messages/{id}/reply` | Reply to a message (primary path) | `im:message`, `im:message:send_as_bot`, or `im:message:send` |
| `POST /im/v1/messages` | Send a message (fallback when reply fails) | Same as above |
| `GET /im/v1/messages/{id}/resources/{key}` | Download a file/image sent by the user | `im:message` or `im:message:readonly` |
| `POST /im/v1/images` | Upload an image to obtain an `image_key` | `im:resource` or `im:resource:upload` |
| `POST /cardkit/v1/cards`<br>`PUT /cardkit/v1/cards/{id}/elements/{eid}/content`<br>`PATCH /cardkit/v1/cards/{id}/settings` | Streaming card (only needed for streaming output mode) | `cardkit:card:write` |
| Subscribe to event `im.message.receive_v1` (direct message) | Receive direct messages | `im:message.p2p_msg` or `im:message.p2p_msg:readonly` |
| Subscribe to event `im.message.receive_v1` (group @ mention) | Receive @-mention messages in groups | `im:message.group_at_msg` or `im:message.group_at_msg:readonly` |

Therefore, to fully enable IM functionality (direct messages + group @ mentions + files + images + streaming cards), a minimum of 5 permission
categories is required: receive direct messages, receive group @ mentions, send messages, read/write resources, and write cards. If streaming
output is disabled (output mode set to "Full"), `cardkit:card:write` can be omitted.

#### Webhook Mode

```
Feishu server ──HTTP POST──▶ /api/v1/im/callback/{channel_id}
                                   │
                           Decrypt (AES-256-CBC, optional)
                           Parse JSON → IncomingMessage
                                   │
                           Reply via the Feishu Open API
```

- **Encryption scheme:** AES-256-CBC, with the Key being `SHA-256(encrypt_key)` and the IV being the first 16 bytes of the ciphertext
- **Event filtering:** only `im.message.receive_v1` events are processed; other event types are ignored
- **Message types:** `text` (text), `file` (file), `image` (image), `post` (rich text, extracts title + structured content)
- **Group message handling:** the `@_user_xxx` mention prefix is automatically stripped

#### WebSocket Mode

Establishes a long connection via the official Feishu SDK (`github.com/larksuite/oapi-sdk-go`); event delivery is equivalent to Webhook mode, requires no public domain, and has built-in automatic reconnection.

#### Streaming Reply (CardKit v1)

Feishu's streaming output is based on **CardKit card streaming updates**, the officially recommended best practice:

```
StartStream:
  1. POST /cardkit/v1/cards              → create the card entity (streaming_mode: true)
  2. POST /im/v1/messages                → send the card message to the chat

UpdateStreamContent:
  3. PUT /cardkit/v1/cards/{id}/elements/{eid}/content  → update the element content (accumulated full text)

FinalizeStream:
  4. PUT /cardkit/v1/cards/{id}/elements/{eid}/content  → final visible content (usually the plain answer)

EndStream:
  5. PATCH /cardkit/v1/cards/{id}/settings  → set streaming_mode: false
```

Each `UpdateStreamContent` / `FinalizeStream` call sends the **accumulated full text**, not an increment, tracked by `feishuStreamState`, which maintains the full content and a strictly increasing `sequence` number.

**Think block handling:** `<think>...</think>` blocks in streaming output are converted into Feishu Markdown quote block format:

```
> 💭 **Thinking Process**
> [thinking content line 1]
> [thinking content line 2]
```

> **Known limitation**: `Region.ThinkingText` only localizes the card's initial placeholder text (Lark shows `Thinking...`).
> Once streaming output begins, the think block title comes from the shared `MarkdownThinkStyle`, which is currently in Chinese
> for all platforms (Lark, Slack, Telegram, Mattermost, etc.). Localizing it would require threading the locale through
> `FormatIMDisplayContent`, which is a cross-platform change and out of scope for the Lark integration.

**Orphaned stream cleanup:** a background goroutine scans every 1 minute for streaming cards that have been open for more than 5 minutes without being closed, and automatically calls `EndStream` to close them (preventing memory leaks).

#### Source Files

| File | Responsibility |
|------|------|
| `internal/im/feishu/region.go` | `Region` definition (Feishu / Lark domains, platform identifier, log prefix, localized text) |
| `internal/im/feishu/adapter.go` | Event parsing, CardKit streaming implementation, token caching, AES decryption, think block conversion, file downloads |
| `internal/im/feishu/longconn.go` | WebSocket long connection (wraps the Feishu SDK), event dispatch |
| `internal/im/feishu/factory.go` | Constructs the adapter and long-connection client based on `Region` |

---

### Slack

The unified adapter supports both Webhook and WebSocket (Socket Mode) modes, and natively implements the `StreamSender` interface.

#### Webhook Mode (Events API)

```
Slack server ──HTTP POST──▶ /api/v1/im/callback/{channel_id}
                                   │
                           Signature verification (HMAC-SHA256)
                           Parse JSON → IncomingMessage
                                   │
                           Reply via the Slack Web API
```

- **Signature verification:** uses `signing_secret` to perform HMAC-SHA256 signature verification on the request body, preventing forged requests.
- **Event filtering:** only `message` and `app_mention` events are processed; messages sent by the bot itself are ignored.
- **URL verification:** automatically handles Slack's `url_verification` challenge request.

#### WebSocket Mode (Socket Mode)

Establishes a long connection via `slack-go/slack/socketmode`; event delivery is equivalent to Webhook mode, requires no public domain, and has built-in automatic reconnection.

```
LongConnClient ══WebSocket══▶ wss://wss-primary.slack.com
       │
  1. Establish the connection using the App Token
  2. Receive Events API message frames
  3. Acknowledge messages (Ack)
  4. Reply via the Slack Web API
```

#### Streaming Reply

Slack's streaming output is implemented via message updates (chat.update):

```
StartStream:
  1. POST /chat.postMessage              → send the initial message, obtain the ts (timestamp)

UpdateStreamContent:
  2. POST /chat.update                   → update the message content based on ts (accumulated full text)

FinalizeStream:
  3. POST /chat.update                   → replace with the final visible content

EndStream:
  4. No special action needed
```

Each `UpdateStreamContent` / `FinalizeStream` call sends the **accumulated full text**, not an increment.

#### Source Files

| File | Responsibility |
|------|------|
| `internal/im/slack/adapter.go` | Event parsing, signature verification, streaming implementation, file downloads |
| `internal/im/slack/longconn.go` | WebSocket long connection (wraps slack-go Socket Mode) |

---

### Telegram

The unified adapter supports both Webhook and long polling modes, natively implementing the `StreamSender` and `FileDownloader` interfaces.

#### Webhook Mode

```
Telegram server ──HTTP POST──▶ /api/v1/im/callback/{channel_id}
                                   │
                           Secret Token verification (optional)
                           Parse JSON → IncomingMessage
                                   │
                           Reply via the Telegram Bot API
```

- **Signature verification:** constant-time comparison of the `X-Telegram-Bot-Api-Secret-Token` header against the configured `secret_token`
- **Message types:** `text` (text), `document` (file, downloaded via file_id), `photo` (image, automatically selects the largest size)
- **Group message handling:** the `@bot` mention prefix is automatically stripped

#### Long Polling Mode

Telegram's WebSocket mode actually uses long polling rather than a true WebSocket, continuously calling the `getUpdates` API to fetch new messages.

```
LongConnClient ──HTTP POST──▶ https://api.telegram.org/bot<token>/getUpdates
       │
  1. Send getUpdates (timeout=30s, long polling)
  2. Parse the returned Update list
  3. Update the offset to confirm processed messages
  4. Loop
  5. Back off 3 seconds on error and retry
```

- **Timeout settings:** 35s HTTP client timeout, 30s long polling timeout
- **Message acknowledgment:** processed messages are automatically confirmed via the `offset` parameter

#### Streaming Reply

Telegram's streaming output is implemented via message editing (editMessageText):

```
StartStream:
  1. POST sendMessage               → send the initial "Thinking..." message, obtain the message_id

UpdateStreamContent:
  2. POST editMessageText            → update the message content based on message_id (accumulated full text, plain text)

FinalizeStream:
  3. POST editMessageText            → replace with the final visible content (same as during streaming, plain text)

EndStream:
  4. Clean up stream state
```

Each `UpdateStreamContent` / `FinalizeStream` call sends the **accumulated full text**, not an increment, with a minimum edit interval of 500ms (to avoid triggering Telegram's rate limits). Both intermediate and final states use plain text, avoiding Markdown parsing failures.

**Think block handling:** `<think>...</think>` blocks in streaming output are converted into Telegram quote block format:

```
> 💭 *Thinking Process*
> [thinking content line 1]
> [thinking content line 2]
```

**Orphaned stream cleanup:** a background goroutine scans every 1 minute for streaming messages that have been open for more than 5 minutes without being closed, and automatically cleans them up (preventing memory leaks).

#### File Download

Telegram supports downloading file and image messages, via the following flow:
1. Call the `getFile` API to obtain the file path
2. Download the file via `https://api.telegram.org/file/bot<token>/<file_path>`

#### Source Files

| File | Responsibility |
|------|------|
| `internal/im/telegram/adapter.go` | Event parsing, Secret Token verification, streaming implementation (editMessage), file downloads |
| `internal/im/telegram/longconn.go` | Long polling client: getUpdates loop, offset management, error backoff |

---

### DingTalk

The unified adapter supports both Webhook and Stream modes, natively implementing the `StreamSender` interface. Supports optional AI card streaming output.

#### Webhook Mode

```
DingTalk server ──HTTP POST──▶ /api/v1/im/callback/{channel_id}
                                   │
                           HmacSHA256 signature verification
                           Parse JSON → IncomingMessage
                                   │
                           Reply via sessionWebhook or the OpenAPI
```

- **Signature verification:** uses `client_secret` to compute an HmacSHA256 signature over `timestamp + "\n" + secret`, Base64-encoded and compared against the `Sign` request header; the timestamp is valid for 1 hour
- **Reply method:** prefers replying via the `sessionWebhook` in the callback message body (suitable for group chats); falls back to the OpenAPI when unavailable or on failure
  - Group chat: `/v1.0/robot/groupMessages/send`
  - Direct message: `/v1.0/robot/oToMessages/batchSend`
- **Message format:** Markdown format (`sampleMarkdown`)
- **Authentication:** obtains an Access Token via `/v1.0/oauth2/accessToken`, with caching (refreshed 5 minutes before expiry)

#### Stream Mode (WebSocket)

Establishes a Stream connection via the official DingTalk SDK (`github.com/open-dingtalk/dingtalk-stream-sdk-go`); event delivery is equivalent to Webhook mode, requires no public domain, and has built-in automatic reconnection.

```
LongConnClient ══Stream══▶ DingTalk Stream service
       │
  1. Establish the connection using ClientID + ClientSecret
  2. Register the ChatBot callback handler
  3. Receive message events
  4. Reconnection and heartbeat built into the SDK
```

#### Streaming Reply (AI Cards)

DingTalk's streaming output is implemented via **AI interactive cards** (requires configuring `card_template_id`):

```
StartStream:
  1. POST /v1.0/card/instances/createAndDeliver  → create and deliver the AI card

UpdateStreamContent:
  2. PUT /v1.0/card/streaming                     → streaming update of the card content (accumulated full text)

FinalizeStream:
  3. PUT /v1.0/card/streaming                     → replace with the final visible content

EndStream:
  4. PUT /v1.0/card/streaming (isFinalize=true)   → mark the stream as finished
```

Each `UpdateStreamContent` / `FinalizeStream` call sends the **accumulated full text** (`isFull: true`), with a minimum update interval of 500ms.

**Fallback strategy without a card template:** when `card_template_id` is not configured, streaming content is accumulated in memory, and `EndStream` sends the complete reply all at once via `sessionWebhook` or the OpenAPI.

**Think block handling:** uses the same Markdown quote block format as Feishu (`MarkdownThinkStyle`):

```
> 💭 **Thinking Process**
> [thinking content line 1]
> [thinking content line 2]
```

**Orphaned stream cleanup:** a background goroutine scans every 1 minute for streams that have been open for more than 5 minutes without being closed, and automatically cleans them up to prevent memory leaks.

#### Source Files

| File | Responsibility |
|------|------|
| `internal/im/dingtalk/adapter.go` | Event parsing, HmacSHA256 signature verification, AI card streaming implementation, Access Token caching, OpenAPI calls |
| `internal/im/dingtalk/longconn.go` | Stream client (wraps the DingTalk SDK), ChatBot message dispatch |

---

### Mattermost

Only supports **Webhook** mode: a Mattermost **Outgoing Webhook** POSTs the request body to WeKnora's unified callback address; the adapter implements `Adapter`, `StreamSender`, and `FileDownloader` (using standard library HTTP calls to the REST API, no third-party SDK).

#### Webhook Mode (Outgoing Webhook)

```
Mattermost server ──HTTP POST──▶ /api/v1/im/callback/{channel_id}
                                        │
                                Validate that body's token = outgoing_token
                                Parse JSON or x-www-form-urlencoded
                                        │
                                Reply via REST API v4 (Bearer Bot Token)
```

- **Security validation:** the `token` in the request body must match the `outgoing_token` in the channel credentials (the factory validates that `outgoing_token` is non-empty when the channel is created).
- **Payload format:** supports both `application/json` and `application/x-www-form-urlencoded` (consistent with the [official documentation](https://developers.mattermost.com/integrate/webhooks/outgoing/)).
- **Bot filtering:** if `bot_user_id` is configured and `user_id` matches it, a `nil` message is returned to avoid self-triggering.
- **Thread reply:** `IncomingMessage.Extra` stores `thread_root_id` (uses the value of `root_id` if present, otherwise the triggering post's `post_id`); `root_id` is set when calling `SendReply` / `CreatePost`, so the reply appears in the same thread.
- **Message deduplication:** `MessageID` uses the triggering post's `post_id`.
- **File messages:** if the payload contains `file_ids`, the first file ID is used as `FileKey`; `DownloadFile` first calls `GET /api/v4/files/{id}/info` and then `GET /api/v4/files/{id}` to download the content.

#### Streaming Reply

Similar to Slack, the accumulated full text is shown by **editing the same post**:

```
StartStream:
  1. POST /api/v4/posts                    → create a placeholder post (e.g. "Thinking..."), obtain the post id

UpdateStreamContent:
  2. PUT /api/v4/posts/{post_id}/patch     → patch the message field (accumulated full text)

FinalizeStream:
  3. PUT /api/v4/posts/{post_id}/patch     → replace with the final visible content

EndStream:
  4. Clean up stream state
```

The streaming refresh interval is batched by the Service-side `streamFlushInterval` (300ms), reducing edit frequency and API load.

#### URL Verification

Mattermost outgoing webhooks have no Slack/Feishu-style challenge flow; `HandleURLVerification` always returns `false`.

#### Source Files

| File | Responsibility |
|------|------|
| `internal/im/mattermost/adapter.go` | Outgoing webhook parsing, token validation, posting/patching for streaming, file downloads |
| `internal/im/mattermost/client.go` | REST v4: `CreatePost`, `PatchPostMessage`, file info/download |
| `internal/im/mattermost/form_parse.go` | Form-encoded body and `file_ids` parsing helpers |

---

## Slash Command System

IM channels support slash commands — users type `/commandname` in chat to trigger them, bypassing the QA pipeline and not subject to rate limiting.

### Built-in Commands

| Command | Arguments | Description |
|------|------|------|
| `/help` | `[command name]` | Shows a list of all available commands; with an argument, shows detailed usage for the specified command |
| `/info` | — | Shows the name, persona/role settings, knowledge base list, and other information of the currently bound assistant |
| `/search` | `<keyword>` | Performs a hybrid search (vector + keyword) against the bound knowledge base, returning up to 5 raw text snippets without AI summarization |
| `/stop` | — | Cancels the currently queued or executing QA request |
| `/clear` | — | Clears the current conversation memory (soft-deletes the ChannelSession); a new session starts on the next message |

### Command Dispatch Flow

```
User message ──▶ HandleMessage
               │
               ├─ Starts with "/"?
               │      │
               │      ├─ Registered command → CommandRegistry.Parse → Command.Execute → reply with result
               │      │                                             │
               │      │                                     ActionClear → soft-delete the ChannelSession
               │      │                                     ActionStop  → cancel a queued/executing QA request
               │      │
               │      └─ LooksLikeCommand() = true but not registered
               │             → reply "Unknown command, send /help to see available commands"
               │         LooksLikeCommand() = false (e.g. "/api/v2/users")
               │             → treated as a normal message, enters the QA pipeline
               │
               └─ Normal message → rate limit check → qaQueue → QA pipeline
```

> `LooksLikeCommand()` checks whether the first token contains a `/` separator to distinguish command attempts from URL paths, avoiding false interception.

### Extending with Custom Commands

Implement the `im.Command` interface and register it with the `CommandRegistry` during Service initialization:

```go
type Command interface {
    Name() string        // command name (without "/")
    Description() string // one-line description, used for /help output
    Execute(ctx context.Context, cmdCtx *CommandContext, args []string) (*CommandResult, error)
}
```

**Design conventions:**
- Dependencies (DB, Services) are injected via the constructor, not placed in `CommandContext`
- User input errors are returned as friendly messages via `CommandResult`; `error` is used only for infrastructure failures (DB errors, network errors, etc.)
- Side-effect intentions (such as clearing a session) are declared via `CommandResult.Action` and executed by the Service
- Registering a duplicate command name will panic at startup, ensuring configuration errors are surfaced early

### Source Files

| File | Responsibility |
|------|------|
| `internal/im/command.go` | `Command` interface, `CommandAction`, `CommandContext` definitions |
| `internal/im/command_registry.go` | `CommandRegistry`: command registration, parsing, dispatch, `LooksLikeCommand` |
| `internal/im/cmd_help.go` | `/help` command implementation |
| `internal/im/cmd_info.go` | `/info` command implementation (shows Agent info, knowledge base list) |
| `internal/im/cmd_search.go` | `/search` command implementation (hybrid search, up to 5 results, content truncated to 200 runes) |
| `internal/im/cmd_stop.go` | `/stop` command implementation |
| `internal/im/cmd_clear.go` | `/clear` command implementation |

---

## QA Queue and Rate Limiting

### QA Queue (qaQueue)

A bounded worker pool queue manages QA requests, preventing concurrency overload:

```
Message ──▶ Enqueue ──▶ [ Waiting queue (≤50) ] ──▶ Worker Pool (5 workers) ──▶ QA pipeline
              │                                      │
              ├─ Queue full → reject and reply with a message │
              ├─ User queue limit exceeded (≤3) → reject       ├─ Wait timeout (>60s) → dropped and user notified
              └─ /stop → Remove(userKey) to cancel             └─ Executes QA normally
```

**Design highlights:**

- **Bounded queue**: maximum capacity 50, preventing unbounded memory growth
- **Per-user backpressure**: each user can have at most 3 requests queued at once, preventing a single user from flooding and filling the queue
- **Queue wait notification**: on successful enqueue when the queue is non-empty, replies "There are N messages ahead of you being processed, please wait"
- **Queue timeout**: a request waiting in the queue for more than 60 seconds is automatically dropped, with a timeout notification sent
- **Cancelable**: the `/stop` command cancels a queued request via `qaQueue.Remove(userKey)`, and cancels an executing request via the `context.CancelFunc` in the `inflight` map
- **Metrics monitoring**: every 30 seconds, outputs queue depth, active worker count, and enqueue/processed/rejected/timeout counts (only when there is activity)

### Sliding Window Rate Limiting (rateLimiter)

Before a message enters the QA queue, it is rate-limited using a sliding window keyed by `channelID:userID:chatID`:

| Parameter | Value | Description |
|------|------|------|
| Window size | 60s | Sliding time window |
| Max requests | 10 per window | Each user can send at most 10 messages into QA per minute |
| Cleanup interval | 1 min | Automatically clears expired entries, preventing memory leaks |

When the rate limit is exceeded, a notification message is sent in reply and the message does not enter the queue. Slash commands are not subject to rate limiting.

### Source Files

| File | Responsibility |
|------|------|
| `internal/im/qaqueue.go` | qaQueue: bounded queue, worker pool, QueueMetrics, metrics reporting |
| `internal/im/ratelimit.go` | slidingWindowLimiter: per-key sliding window rate limiting, concurrency-safe cleanup |

---

## Streaming Output Mechanism

Streaming mode collects content chunks produced by the QA pipeline in real time via the `EventBus`, pushing them in **batches every 300ms**, balancing latency against API rate limits:

```
QA pipeline ──chunk──chunk──chunk──▶ EventBus
                                    │
                              Flush every 300ms
                                    │
                        ┌───────────▼───────────┐
                        │ Accumulate content →   │
                        │ push a full replacement │
                        │ (not incremental, sends │
                        │ the full text each time)│
                        └───────────────────────┘
```

### Content Handling

- **Think block filtering/conversion**: `<think>...</think>` blocks are converted into Markdown quote blocks for display in Feishu and DingTalk, converted into quote blocks with an italic title in Telegram, and filtered out on other platforms
- **Tool event display**: Agent tool calls are shown in real time with their invocation status
  - In progress: `⏳ [tool name]` (wrapped inside a think block)
  - Succeeded: `✅ [tool name] · [summary]`
  - Failed: `⚠️ [tool name] failed`
  - Internal tools (thinking, todo_write, etc.) are not shown to the user
- **Empty content fallback**: if streaming produces no visible content, it falls back to full-reply mode (`fallbackNonStream`)
- **Full persistence**: the full content (including thinking) is persisted to the database, ensuring complete history

### Platform-Specific Streaming Handling

- **"Thinking..." placeholder**: Feishu and Telegram immediately show placeholder text right after streaming initialization, improving perceived responsiveness
- **Orphaned stream cleanup**: background goroutines for Feishu, Telegram, and DingTalk scan every `streamReaperInterval` (1 minute) for streams open longer than `streamOrphanTTL` (5 minutes) without being closed, and automatically close them to prevent memory leaks
- **Think block conversion**: Feishu and DingTalk convert `<think>` tags into Markdown quote blocks (`> 💭 **Thinking Process**`); Telegram uses an italic title (`> 💭 *Thinking Process*`)

---

## File Message Handling

When a user sends a file or image message in IM, if the channel is configured with `knowledge_base_id`, the Service automatically saves the file to the corresponding knowledge base:

```
User sends a file/image message
        │
        ▼
  Message type = file/image?
  Channel configured with knowledge_base_id?
  Does the Adapter implement FileDownloader?
        │ All satisfied
        ▼
  1. adapter.DownloadFile(msg) → io.ReadCloser + fileName
  2. Notify the user "Processing file..."
  3. knowledgeService.Save(file, knowledgeBaseID)
  4. Notify the user "File saved to knowledge base"
```

**File download methods by platform:**

| Platform | Method |
|------|------|
| Feishu | GetMessageResource API (via FileKey) |
| WeCom Webhook | Direct PicUrl download or MediaId temporary media API |
| WeCom WebSocket | Encrypted attachment URL + per-message AES key decryption |
| Telegram | getFile API to obtain the file path + HTTPS download (supports documents and images) |
| Mattermost | `GET /api/v4/files/{file_id}/info` + `GET /api/v4/files/{file_id}` (Bearer Bot Token) |

---

## Key Parameters and Thresholds

| Parameter | Value | Description |
|------|------|------|
| `qaTimeout` | 120s | Maximum execution time for the QA pipeline |
| `dedupTTL` | 5 min | Message deduplication ID retention duration |
| `dedupCleanupInterval` | 1 min | Deduplication cleanup cycle |
| `maxContentLength` | 4096 | Maximum message length (runes); truncated if exceeded |
| `streamFlushInterval` | 300ms | Streaming content batch flush interval |
| `defaultMaxQueueSize` | 50 | Maximum capacity of the QA queue |
| `defaultMaxPerUser` | 3 | Maximum queued requests per user |
| `defaultWorkers` | 5 | Number of concurrent QA workers |
| `queueTimeout` | 60s | Maximum wait time for a request in the queue |
| `rateLimitWindow` | 60s | Rate limit sliding window size |
| `rateLimitMaxRequests` | 10 | Maximum requests per user per window |
| `metricsLogInterval` | 30s | Queue metrics logging cycle |
| `streamOrphanTTL` | 5 min | Feishu/Telegram/DingTalk orphaned stream timeout |
| `streamReaperInterval` | 1 min | Feishu/Telegram/DingTalk orphaned stream cleanup scan cycle |
| Telegram edit interval | 500ms | Minimum interval between editMessageText calls (avoids rate limits) |
| Telegram long polling timeout | 30s | getUpdates timeout parameter |
| Telegram error backoff | 3s | Wait time after a getUpdates failure |
| DingTalk card update interval | 500ms | Minimum interval for AI card streaming updates |
| DingTalk signature validity | 1h | Valid window for the webhook callback signature timestamp |
| WeCom WS heartbeat | 30s | WebSocket keep-alive frequency |
| WeCom WS read timeout | 90s | 3 × heartbeat interval, tolerates one missed heartbeat |
| WeCom WS reconnect backoff | 1s → 30s | Exponential backoff, capped at 30 seconds |
| Token cache safety margin | 5 min | Token refreshed this long before expiry |

---

## Error Handling

| Scenario | Handling Strategy |
|------|---------|
| Streaming initialization failure | Automatically falls back to full-reply mode (`fallbackNonStream`) |
| QA pipeline exception | Replies "Sorry, an error occurred while processing your question, please try again later." |
| QA timeout (>120s) | Marks the message complete, replies with a timeout notice |
| Empty answer | Replies "Sorry, I'm unable to answer this question right now." |
| Empty streaming content | Falls back to full reply when no visible content is produced |
| WebSocket disconnect | Automatic reconnection with exponential backoff |
| Platform retry | Deduplicated by MessageID, automatically skipped within 5 minutes |
| Channel startup failure | Error logged, does not affect other channels |
| QA queue full | Request rejected with reply "There are currently many people in the queue, please try again later." |
| User queue limit exceeded | Request rejected with a notification (per-user limit ≤3) |
| Queue wait timeout | Automatically dropped after 60s, replies "Your message wait timed out, please resend." |
| Message rate-limited | Replies with a rate-limit notice when the sliding window limit (10) is exceeded |
| Feishu orphaned stream | Scanned every minute; streams open more than 5 minutes are automatically closed |
| Telegram/DingTalk orphaned stream | Same as Feishu, scanned and cleaned up automatically every minute |
| WeCom group reply failure | Falls back to a private message to the user when the appchat API fails |
| DingTalk AI card creation failure | Falls back to sessionWebhook or OpenAPI reply |
| DingTalk sessionWebhook unavailable | Falls back to the OpenAPI (separate endpoints for group chat/direct message) |

---

## Extending to New Platforms

Integrating a new IM platform only takes 3 steps:

### 1. Implement the `im.Adapter` Interface

Create an adapter under `internal/im/<platform>/`:

```go
package myplatform

type Adapter struct { /* platform configuration */ }

func (a *Adapter) Platform() im.Platform     { return "myplatform" }
func (a *Adapter) VerifyCallback(c *gin.Context) error { /* signature verification */ }
func (a *Adapter) ParseCallback(c *gin.Context) (*im.IncomingMessage, error) { /* parse the message */ }
func (a *Adapter) SendReply(ctx context.Context, incoming *im.IncomingMessage, reply *im.ReplyMessage) error { /* send the reply */ }
func (a *Adapter) HandleURLVerification(c *gin.Context) bool { /* URL verification */ }
```

Optional interfaces:
- Implement `im.StreamSender` to support streaming output
- Implement `im.FileDownloader` to support automatically saving file messages to the knowledge base

> You can refer to the existing Telegram (pure HTTP API) and DingTalk (SDK + AI cards) adapters as implementation references.

### 2. Register the Adapter Factory

Register the factory function in `registerIMAdapterFactories` in `internal/container/container.go`:

```go
imService.RegisterAdapterFactory("myplatform", func(ctx context.Context, channel *im.IMChannel, msgHandler func(*im.IncomingMessage)) (im.Adapter, im.CancelFunc, error) {
    creds := parseCredentials(channel.Credentials)
    appKey := getString(creds, "app_key")
    appSecret := getString(creds, "app_secret")

    adapter := myplatform.NewAdapter(appKey, appSecret)

    // WebSocket mode needs to start a long connection
    if channel.Mode == "websocket" {
        cancelCtx, cancel := context.WithCancel(ctx)
        go adapter.StartLongConn(cancelCtx, msgHandler)
        return adapter, func() { cancel() }, nil
    }

    return adapter, func() {}, nil
})
```

### 3. Add the Platform Option to the Frontend

In `IMChannelPanel.vue`:
- Add a platform radio option
- Add the credential form fields for that platform

Add the platform name translation to the i18n files.

The Service layer (`im.Service`) needs no modification at all — channel management, command dispatch, message orchestration, session management, QA scheduling, rate limiting, and streaming control are all handled uniformly by the Service.

---
