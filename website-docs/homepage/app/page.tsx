import { existsSync } from "node:fs";
import { join } from "node:path";
import Image from "next/image";
import { BrandLogo } from "./brand-logo";
import { Icon } from "./ui";
import { homeAssets } from "../../shared/header";
import { ProductGallery, type GalleryShot, type GallerySlide } from "./product-gallery";
import { Header, ProductVideo } from "./interactive";
import { clients, dataSources, modelProviders, IntegrationMark } from "./brands";
import s from "./home.module.css";

const repo = "https://github.com/Tencent/WeKnora";
const docs = "/docs/";
const guide = (path: string) => `${docs}${path}.html`;
const modes = [
  { number: "01", icon: "search", label: "RAG", title: "Answers you can check", description: "Combines semantic and keyword search to find relevant material; answers come with source citations you can open and verify against the original.", tags: ["Hybrid search", "Multimodal parsing", "Source citations"], link: "03-features/05-retrieval-engines" },
  { number: "02", icon: "agent", label: "Agent", title: "Tasks done with knowledge and tools", description: "The agent searches knowledge bases and the web, calls MCP tools and skills, processes files and runs scripts in a sandbox, operates the browser on your computer, and remembers the preferences you have confirmed across sessions.", tags: ["Multi-step reasoning", "Skills and sandbox", "Local browser", "MCP tools", "Long-term memory"], link: "03-features/07-agent" },
  { number: "03", icon: "wiki", label: "Wiki", title: "Documents organized into a wiki", description: "Generates interlinked wiki pages and a knowledge graph from source documents, with browsing, editing and version rollback.", tags: ["Automatic organization", "Knowledge graph", "Version rollback"], link: "03-features/14-wiki" },
];
const releaseExtras = ["Confluence / DingTalk Docs data sources", "Catalog of 27 model providers", "Bocha / Serply web search", "Japanese UI", "Whitelist-only egress"];

const productShotExists = (src: string) => existsSync(join(process.cwd(), "public", src));
type SlideSource = Omit<GallerySlide, "shots"> & ({ image: string; alt: string } | { shots: Omit<GalleryShot, "available">[] });
const withAvailability = (slide: SlideSource): GallerySlide => {
  const shots = "shots" in slide ? slide.shots : [{ image: slide.image, alt: slide.alt }];
  return { name: slide.name, icon: slide.icon, title: slide.title, description: slide.description, link: slide.link, external: slide.external, shots: shots.map(shot => ({ ...shot, available: productShotExists(`${homeAssets}/product/${shot.image}.png`) })) };
};
const releaseSlides = [
  { name: "Local browser", icon: "browser", shots: [
    { label: "Run a task", icon: "browser", image: "local-browser-task", alt: "WeKnora UI: a smart reasoning conversation operating the local browser, with the task preview and pause, resume and end controls shown in the conversation" },
    { label: "Connect the extension", icon: "plug", image: "browser-connection", alt: "WeKnora UI: the browser connection page in the Toolbox, with the BrowserSkill extension connected and the web actions the agent can perform listed" },
  ], title: "Operate the browser on your computer", description: "Through Tencent's open-source BrowserSkill extension, the agent opens pages and fills in forms in your own Chrome or Edge, and hands over to you for logins and CAPTCHAs.", link: guide("05-clients/09-local-browser"), external: { label: "BrowserSkill", href: "https://github.com/Tencent/BrowserSkill", logo: "browserskill.png" } },
  { name: "MCP Server", icon: "plug", image: "mcp-server-endpoint", title: "Publish knowledge bases to other AI tools", description: "Create an MCP endpoint for a space, and clients such as Claude and Cursor can connect to search the knowledge bases and ask questions.", alt: "WeKnora UI: connection details of an MCP endpoint, including the endpoint address and the mcpServers configuration for Cursor and Claude Desktop", link: guide("03-features/08-mcp") },
  { name: "Conversation control", icon: "branch", image: "chat-steer-queue", title: "Adjust a conversation while it runs", description: "Add requirements while an answer is in progress, or fork or rewind from any earlier question; generated files are all collected on the Artifacts page.", alt: "WeKnora UI: follow-up requirements queued above the input box while an answer is being generated", link: guide("03-features/18-chat-experience") },
].map(withAvailability);
const wikiSlides = [
  { name: "Knowledge graph", icon: "channels", image: "wiki-graph", title: "Follow the links to related knowledge", description: "See how pages relate to each other in the knowledge graph. Click an entry to read the related content.", alt: "Wiki knowledge graph: links between the daily expense reimbursement entry and related pages, with entry details on the right" },
  { name: "Page browser", icon: "wiki", image: "wiki-browser", title: "Organized by topic, with sources kept", description: "With Wiki enabled, people, products and concepts are extracted from knowledge base documents into pages with source citations, browsable by directory.", alt: "Wiki browser: a topic-organized directory, the annual leave page, related entries and citations of the original documents" },
  { name: "Revision history", icon: "history", image: "wiki-revision-history", title: "Edit anytime, trace every change", description: "Revise pages directly or let the agent help maintain them. View the revision diff and restore earlier content when needed.", alt: "Wiki revision history: the list of previous versions of the annual leave page, the content diff and the rollback entry" },
].map(withAvailability);
const sandboxSlides = [
  { name: "Session sandbox", icon: "sandbox", image: "skill-sandbox-chat", title: "Keep working in the same sandbox", description: "Docker, E2B and Cube are supported. Turns in the same session share one workspace, and generated files can be previewed and downloaded.", alt: "WeKnora UI: the agent generates a Word document from the knowledge base and opens the artifact preview next to the conversation", link: guide("03-features/22-skills-sandbox") },
  { name: "Desktop and terminal", icon: "monitor", shots: [
    { label: "Graphical desktop", icon: "monitor", image: "sandbox-desktop", alt: "WeKnora UI: the Desktop tab of the sandbox panel on the right of the conversation, showing the XFCE graphical desktop and file manager inside the sandbox" },
    { label: "Interactive terminal", icon: "terminal", image: "sandbox-terminal", alt: "WeKnora UI: the sandbox terminal on the right of the conversation, listing the Word files generated in the workspace output directory" },
  ], title: "Open the desktop and terminal to see every step", description: "Open the graphical desktop or interactive terminal beside the chat to follow each of the agent's steps, and take over yourself when needed.", link: guide("03-features/22-skills-sandbox") },
  { name: "Skill catalog", icon: "skills", image: "skill-catalog", title: "Install, manage and reuse skills", description: "Install skills from ClawHub, SkillHub, Git or ZIP, and manage and reuse them across the space.", alt: "WeKnora UI: the skill management page in the Toolbox, listing the docx, pptx and pdf skills and the sandboxes they are installed in", link: guide("03-features/22-skills-sandbox") },
].map(withAvailability);

export default function Home() {
  return <div className={s.site}>
    <a className={s.skipLink} href="#main">Skip to content</a>
    <Header />
    <main id="main">
      <section className={`${s.shell} ${s.hero}`} aria-labelledby="hero-title">
        <a href="#release" className={s.releaseLink}><span>v0.8.2</span> Local browser, MCP Server and conversation forking <Icon name="arrow" /></a>
        <div className={s.heroGrid}>
          <h1 id="hero-title">Find the answers,<br /><em>and put knowledge to work.</em></h1>
          <div className={s.heroAside}>
            <p className={s.eyebrow}>TENCENT OPEN SOURCE · WEKNORA</p>
            <p className={s.heroDescription}>Tencent&apos;s open-source enterprise knowledge management framework.<br />Brings team documents together for knowledge Q&amp;A, task execution and wiki curation.</p>
            <div className={s.actions}><a className={s.primary} href="#get-started">Get started <Icon name="arrow" /></a><a className={s.secondary} href={repo} target="_blank" rel="noreferrer"><Icon name="github" /> GitHub</a></div>
            <p className={s.heroNote}>RAG Q&amp;A / Agent reasoning / Automatic Wiki</p>
          </div>
        </div>
        <ProductVideo />
        <div className={s.trustBar}><span><Icon name="github" /> Tencent Open Source · MIT License</span><span><Icon name="server" /> Private deployment supported</span><span><Icon name="model" /> Your choice of models and storage</span></div>
      </section>
      <section id="capabilities" className={`${s.shell} ${s.section}`} aria-labelledby="capabilities-title">
        <div className={s.sectionHeading}><div><p className={s.eyebrow}>01 / KNOWLEDGE AT WORK</p><h2 id="capabilities-title">Knowledge Q&amp;A, task execution and automatic Wiki</h2></div><p>Use RAG to look things up, the Agent for multi-step tasks,<br />and the Wiki to organize knowledge. All three share the same knowledge bases.</p></div>
        <div className={s.modeGrid}>{modes.map(mode => <article className={s.mode} key={mode.label}>
          <div className={s.modeTop}><Icon name={mode.icon} /><span>{mode.number} / {mode.label.toUpperCase()}</span></div>
          <h3>{mode.title}</h3><p>{mode.description}</p><ul className={s.tags}>{mode.tags.map(tag => <li key={tag}>{tag}</li>)}</ul>
          <a className={s.textLink} href={mode.label === "Wiki" ? "#wiki" : guide(mode.link)}>Learn about {mode.label} <Icon name="arrow" /></a>
        </article>)}</div>
      </section>
      <section id="release" className={s.release} aria-labelledby="release-title"><div className={s.shell}>
        <div className={s.sectionHeading}><div><p className={s.eyebrow}>02 / INTRODUCING v0.8.2</p><h2 id="release-title">Agents operate the browser,<br />knowledge bases plug into other AI.</h2></div><a className={s.textLink} href={guide("07-releases/v0.8.2")}>Read the release notes <Icon name="arrow" /></a></div>
        <ProductGallery id="release-gallery" label="v0.8.2" slides={releaseSlides} />
        <div className={s.releaseExtras}><span>This release also includes</span>{releaseExtras.map(item => <p key={item}>{item}</p>)}</div>
      </div></section>
      <section id="skills-sandbox" className={`${s.release} ${s.sandboxTopic}`} aria-labelledby="skills-sandbox-title"><div className={s.shell}>
        <div className={s.sectionHeading}><div><p className={s.eyebrow}>03 / SKILLS &amp; SANDBOX</p><h2 id="skills-sandbox-title">Agents run skills<br />and produce files.</h2></div><a className={s.textLink} href={guide("03-features/22-skills-sandbox")}>Learn about skills and sandbox <Icon name="arrow" /></a></div>
        <ProductGallery id="sandbox-gallery" label="Skills and sandbox" slides={sandboxSlides} />
      </div></section>
      <section id="wiki" className={`${s.shell} ${s.section} ${s.wiki}`} aria-labelledby="wiki-title">
        <div className={s.sectionHeading}>
          <div><p className={s.eyebrow}>04 / AUTOMATIC WIKI</p><h2 id="wiki-title">Documents organized into a browsable Wiki.</h2></div>
          <a className={s.textLink} href={guide("03-features/14-wiki")}>Learn how to use the Wiki <Icon name="arrow" /></a>
        </div>
        <ProductGallery id="wiki-gallery" label="Wiki" slides={wikiSlides} />
      </section>
      <section id="ecosystem" className={`${s.shell} ${s.section}`} aria-labelledby="ecosystem-title">
        <div className={s.sectionHeading}><div><p className={s.eyebrow}>05 / INTEGRATIONS</p><h2 id="ecosystem-title">Data source and tool integrations</h2></div><p>Sync material from Feishu, Confluence, GitLab and other platforms,<br />then query and use it through IM, the browser extension, MCP or the API.</p></div>
        <div className={s.ecosystem}>
          <div className={s.ecosystemColumn}><Icon name="sources" /><h3>Import and sync material</h3><p>Upload files, import web pages, or connect external data sources.</p><div className={s.integrations}>{dataSources.map(item => <span key={item.name}><IntegrationMark item={item} /></span>)}</div><a className={s.textLink} href={guide("03-features/10-datasource")}>Data source integrations <Icon name="arrow" /></a></div>
          <div className={s.ecosystemCore}><BrandLogo /><span>Team knowledge bases and agents</span><div>Understand · Retrieve · Reason · Act</div></div>
          <div className={s.ecosystemColumn}><Icon name="channels" /><h3>Access from the tools you use</h3><p>Supports IM Q&amp;A, the browser extension, MCP clients and developer tool integrations.</p><div className={s.integrations}>{clients.map(item => <span key={item.name}><IntegrationMark item={item} /></span>)}</div><a className={s.textLink} href={guide("03-features/12-im-integration")}>Clients and channels <Icon name="arrow" /></a></div>
        </div>
        <div className={s.models}><span>Your choice of models · 27 built-in providers <a className={s.textLink} href={guide("03-features/06-models")}>View all <Icon name="arrow" /></a></span>{modelProviders.map(item => <p key={item.name}><IntegrationMark item={item} /></p>)}</div>
      </section>
      <section id="enterprise" className={s.enterprise} aria-labelledby="enterprise-title"><div className={s.shell}>
        <div className={s.sectionHeading}><div><p className={s.eyebrow}>06 / BUILT FOR YOUR TEAM</p><h2 id="enterprise-title">Private deployment,<br />permissions managed the way your team needs.</h2></div><p>Configure data storage and member permissions,<br />and review operation logs and task status.</p></div>
        <div className={s.enterpriseGrid}><article><Icon name="server" /><h3>Deployment and storage</h3><p>Supports Docker, Kubernetes and Helm, plus the Lite single-machine edition and a desktop app. Models, vector databases and storage backends can be swapped as needed, with support for local inference.</p></article><article><Icon name="shield" /><h3>Space and resource permissions</h3><p>Multi-space isolation and a four-role matrix. API Keys are scoped by capability and knowledge base, OIDC identity integration is supported, and the service can be restricted to whitelisted domains.</p></article><article><Icon name="trace" /><h3>Auditing and runtime monitoring</h3><p>Space audit logs, a runtime task-queue dashboard and Langfuse tracing help teams pinpoint issues and manage runtime status.</p></article></div>
      </div></section>
      <section id="get-started" className={`${s.shell} ${s.closing}`} aria-labelledby="closing-title">
        <div className={s.sectionHeading}><div><p className={s.eyebrow}>GET STARTED</p><h2 id="closing-title">Choose the way that works for you.</h2></div></div>
        <div className={s.startGrid}>
          <article className={s.startCard}>
            <div className={s.startLabel}><Image className={s.startBrand} src={`${homeAssets}/brands/wechat-dialog.png`} alt="WeChat Dialog Open Platform logo" width={32} height={32} /><span>Online</span></div>
            <h3>WeChat Dialog Open Platform</h3>
            <p>Manage knowledge bases online and connect Q&amp;A to Official Accounts, Mini Programs and other WeChat scenarios.</p>
            <a className={s.textLink} href="https://chatbot.weixin.qq.com/login" target="_blank" rel="noreferrer">Open the platform <Icon name="external" /></a>
          </article>
          <article className={s.startCard}>
            <div className={s.startLabel}><Image className={s.startBrand} src={`${homeAssets}/brands/tencent-cloud.ico`} alt="Tencent Cloud logo" width={32} height={32} /><span>Cloud</span></div>
            <h3>Tencent Cloud Lighthouse</h3>
            <p>Deploy WeKnora from an application template and run it on your own cloud server.</p>
            <a className={s.textLink} href="https://mc.tencent.com/s69nKCVz" target="_blank" rel="noreferrer">Deploy on Tencent Cloud <Icon name="external" /></a>
          </article>
          <article className={s.startCard}>
            <div className={s.startLabel}><BrandLogo /><span>Self-hosted</span></div>
            <h3>Your own environment</h3>
            <p>Deploy with Docker or Kubernetes and configure models, storage and networking yourself.</p>
            <a className={s.textLink} href={guide("01-getting-started/02-installation")}>Read the deployment docs <Icon name="arrow" /></a>
          </article>
        </div>
      </section>
    </main>
    <footer className={`${s.shell} ${s.footer}`}><a className={s.brand} href="/" aria-label="WeKnora home"><BrandLogo /></a><p>Tencent Open Source · MIT License</p><nav aria-label="Footer navigation"><a href={docs}>Docs</a><a href={repo} target="_blank" rel="noreferrer">GitHub <Icon name="external" /></a><a href={`${repo}/blob/main/CHANGELOG.md`} target="_blank" rel="noreferrer">Changelog</a></nav></footer>
  </div>;
}
