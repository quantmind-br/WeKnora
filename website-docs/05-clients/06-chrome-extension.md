# Chrome Extension (Knowledge Management Assistant)

The "Knowledge Management Assistant" Chrome extension lets you ask questions against your knowledge base from a sidebar on any webpage, clip the current page, and save Markdown notes.

The extension can be installed from the [Chrome Web Store](https://chromewebstore.google.com/detail/jpemjbopikggjlmikmclgbmkhhopjdgd) and needs to connect to an existing WeKnora service. Imported content is stored in the instance it is connected to.

<Screenshot
  src="/screenshots/chrome-extension.png"
  caption="Chrome extension: web page sidebar Q&A and content clipping"
  hint="Show the extension sidebar expanded on a web page (the Q&A panel and knowledge base selector), along with region selection during clipping." />

## Main Features {#what-it-can-do}

| Capability | Description |
| --- | --- |
| Knowledge base Q&A | A sidebar chat panel that lets you switch between multiple knowledge bases, with three answer modes — quick, deep, and precise — so you can ask questions while browsing without interrupting your current work |
| Web page clipping | Save the page URL, have the AI intelligently extract the main content, or manually select a region, and write it into a specified knowledge base |
| Markdown quick notes | Record content with the built-in editor and save it to the knowledge base |
| Keyboard shortcuts | Actions like asking questions or opening the sidebar can all be bound to custom shortcuts |

## Installation and Connection {#setup}

See the connection guide under "Settings → Publish & Integrations → Chrome Extension" and copy the current instance's API address. Follow these steps to connect:

1. **Get your API credentials**: In "Settings → Publish & Integrations → API Integration," copy the API Key and API address (the "Open API Info" button on the guide page jumps straight there). It's recommended to create a dedicated key and scope its permissions as needed (at minimum it needs retrieval and ingestion-related permissions — see the API Key section of [Tenants, Users, and Authentication & Authorization](../03-features/01-tenant-auth.md));
2. **For the desktop version, fix the port first**: When using the WeKnora desktop version, set a fixed port (e.g. 37841) on the "API Integration" page. Otherwise, the port changes on every startup and the extension won't be able to connect;
3. **Install the extension**: Install it from the Chrome Web Store;
4. **Connect within the extension**: Open the extension settings, select "Enterprise/Developer" mode, and enter the API address and API Key.

After connecting, list your knowledge bases and ask a test question to confirm that the credentials and network are working.

## Troubleshooting

| Symptom | Things to check |
| --- | --- |
| Extension reports it can't connect | Whether the API address is reachable from the machine running the browser (in-container addresses and `localhost` won't work for remote deployments); whether the desktop version has a fixed port set |
| 401 / 403 | Whether the API Key has been revoked; whether the key's permissions cover retrieval and ingestion; if the key is scoped to specific knowledge bases, whether the target base is on the list |
| Nothing shows up in the knowledge base after clipping | Check the parsing status in the knowledge base's document list — `processing` means it's still parsing; failure reasons are shown in the document detail view |

## Related Documentation {#related}

- Credentials and permissions: [Tenants, Users, and Authentication & Authorization](../03-features/01-tenant-auth.md)
- Ingestion pipeline for clipped content: [Document Ingestion Pipeline](../02-architecture/03-document-pipeline.md)
- Other integration methods: [Claw Skill](07-claw-skill.md), [CLI Tool](02-cli.md), [Go SDK](03-go-sdk.md)
