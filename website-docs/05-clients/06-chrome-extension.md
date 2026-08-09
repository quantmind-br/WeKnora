# Chrome Extension (Knowledge Management Assistant)

The browser extension solves the problem of "seeing something useful but being too lazy to copy-paste it into the knowledge base." Once installed, you can ask questions directly against your knowledge base from the sidebar on any webpage, and also clip the current page into it.

The extension itself is available on the [Chrome Web Store](https://chromewebstore.google.com/detail/jpemjbopikggjlmikmclgbmkhhopjdgd) (named "Knowledge Management Assistant") and works together with your self-hosted WeKnora service — it doesn't come with its own backend; all data is written into your own instance.

<Screenshot
  src="/screenshots/chrome-extension.png"
  caption="Chrome extension: web page sidebar Q&A and content clipping"
  hint="Show the extension sidebar expanded on a web page (the Q&A panel and knowledge base selector), along with region selection during clipping." />

## What It Can Do

| Capability | Description |
| --- | --- |
| Knowledge base Q&A | A sidebar chat panel that lets you switch between multiple knowledge bases, with three answer modes — quick, deep, and precise — so you can ask questions while browsing without interrupting your current work |
| Web page clipping | Save the page URL, have the AI intelligently extract the main content, or manually select a region, and write it into a specified knowledge base |
| Markdown quick notes | A built-in editor for jotting down ideas on the fly, saved to the knowledge base with one click |
| Keyboard shortcuts | Actions like asking questions or opening the sidebar can all be bound to custom shortcuts |

## Setup

The WeKnora interface has a guided setup page: "Settings → Integrations → Chrome Extension," which directly displays your current instance's API address along with a copy button. Steps:

1. **Get your API credentials**: In "Settings → API Info," copy the API Key and API address. It's recommended to create a dedicated key and scope its permissions as needed (at minimum it needs retrieval and ingestion-related permissions — see the API Key section of [Tenants, Users, and Authentication & Authorization](../03-features/01-tenant-auth.md));
2. **For the desktop version, fix the port first**: When using the WeKnora desktop version, set a fixed port (e.g. 37841) in the API Info settings. Otherwise, the port changes on every startup and the extension won't be able to connect;
3. **Install the extension**: Install it from the Chrome Web Store;
4. **Connect within the extension**: Open the extension settings, select "Enterprise/Developer" mode, and enter the API address and API Key.

Once configured, it's recommended to have it list your knowledge bases or ask a test question to confirm both the credentials and network connectivity are working.

## Troubleshooting

| Symptom | Things to check |
| --- | --- |
| Extension reports it can't connect | Whether the API address is reachable from the machine running the browser (in-container addresses and `localhost` won't work for remote deployments); whether the desktop version has a fixed port set |
| 401 / 403 | Whether the API Key has been revoked; whether the key's permissions cover retrieval and ingestion; if the key is scoped to specific knowledge bases, whether the target base is on the list |
| Nothing shows up in the knowledge base after clipping | Check the parsing status in the knowledge base's document list — `processing` means it's still parsing; failure reasons are shown in the document detail view |

## Related

- Credentials and permissions: [Tenants, Users, and Authentication & Authorization](../03-features/01-tenant-auth.md)
- Ingestion pipeline for clipped content: [Document Ingestion Pipeline](../02-architecture/03-document-pipeline.md)
- Other integration methods: [Claw Skill](07-claw-skill.md), [CLI Tool](02-cli.md), [Go SDK](03-go-sdk.md)
