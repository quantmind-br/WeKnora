<script setup lang="ts">
import { useData } from 'vitepress/client'
import { withBase } from 'vitepress'
import Illus from './Illus.vue'

const { theme } = useData()
const versionLabel = theme.value.weknoraVersion ?? 'unknown'

const stats = [
  { value: '25', unit: 'types', label: 'File formats: documents, web pages, scanned files, images, audio' },
  { value: '26', unit: '+', label: 'Model providers, all of which can also be swapped for local inference' },
  { value: '9', unit: '', label: 'Access points: Web, IM, extension, command line, MCP' },
  { value: '4', unit: '', label: 'Indexes active at once: vector, keyword, Wiki, graph' },
]

const schema = [
  {
    step: '01',
    name: 'Ingest',
    hint: 'Where content comes from',
    items: ['File upload', 'URL scraping', 'Feishu', 'Notion', 'Yuque', 'RSS'],
  },
  {
    step: '02',
    name: 'Understand',
    hint: 'Convert to structured text',
    items: ['Layout analysis', 'Scanned-file OCR', 'Table extraction', 'Image description', 'Audio transcription'],
  },
  {
    step: '03',
    name: 'Index',
    hint: 'All four paths can run at once',
    items: ['Adaptive chunking', 'Vector', 'Keyword', 'Wiki', 'Knowledge graph'],
  },
  {
    step: '04',
    name: 'Apply',
    hint: 'Capabilities exposed externally',
    items: ['Knowledge Q&A', 'Agent reasoning', 'Wiki site', 'Data analysis', 'FAQ'],
  },
]

const chain = [
  {
    step: '01',
    icon: 'read',
    title: 'Understand the document',
    desc: 'Scanned files go through OCR, illustrations are described by a vision model, audio is transcribed by a speech model, and merged Excel cells are automatically filled in. A dedicated docreader service restores PDF, Office, web pages, images, and audio into text with layout preserved, rather than extracting a single block of plain text.',
    href: '/03-features/03-document-parsing',
  },
  {
    step: '02',
    icon: 'chunk',
    title: 'Chunk the knowledge well',
    desc: 'The chunking method is automatically chosen among three tiers—heading-based, heuristic, and recursive—based on document characteristics. Parent-child chunking lets retrieval hit small chunks while feeding the model the full context. The same document can be written simultaneously into four indexes: vector, keyword, Wiki, and knowledge graph.',
    href: '/02-architecture/03-document-pipeline',
  },
  {
    step: '03',
    icon: 'retrieve',
    title: 'Find the right evidence',
    desc: 'Intent recognition and query rewriting run first; vector and BM25 recall run in parallel and are fused with RRF, then a rerank model orders the results. When the knowledge graph is enabled, a batch of entity-relationship evidence is added to cover questions like "what is the relationship between A and B."',
    href: '/03-features/05-retrieval-engines',
  },
  {
    step: '04',
    icon: 'answer',
    title: 'Give an answer that can be checked',
    desc: 'Ordinary questions are answered directly with single-round retrieval; complex tasks are handed to a ReAct Agent, which decides on its own how many retrieval rounds to run, which tools to call, and whether to run data analysis. The answer streams back with sources annotated segment by segment, and the source text can be opened for verification.',
    href: '/03-features/07-agent',
  },
]

const surfaces = [
  { icon: 'console', name: 'Web console', desc: 'A complete interface for knowledge base management, chat, Wiki browsing, and system configuration.' },
  { icon: 'extension', name: 'Chrome extension', desc: 'Q&A in a web page sidebar, with support for clipping page content and quick Markdown notes into the knowledge base.' },
  { icon: 'embed', name: 'Web embed widget', desc: 'A single script line adds a floating Q&A widget to your own site; visitors don\u2019t need to log in.' },
  { icon: 'desktop', name: 'Desktop client', desc: 'A standalone desktop app with its own backend and local storage; not yet officially released and must be built from source.' },
  { icon: 'bot', name: 'IM bots', desc: 'Official adapters for 10 platforms including WeCom, Feishu, DingTalk, and Slack.' },
  { icon: 'mobile', name: 'WeChat Mini Program', desc: 'A mobile entry point supporting saving web pages to the knowledge base and asking questions.' },
  { icon: 'cli', name: 'weknora CLI', desc: 'Document management, retrieval, and streaming Q&A with citations, with JSON output by default for easy scripting.' },
  { icon: 'api', name: 'REST API and Go SDK', desc: 'A complete /api/v1 interface; API keys support authorization scoped by capability and knowledge base.' },
  { icon: 'mcp', name: 'MCP Server', desc: 'Exposes WeKnora as an MCP tool for retrieval by clients such as Claude and Cursor.' },
]

const features = [
  {
    icon: 'wiki',
    title: 'Wiki mode',
    desc: 'A model extracts entities and concepts from documents to generate cross-linked pages with cited sources, automatically organized into a table of contents and a relationship graph. Suited to scenarios where materials are scattered and lack an overall index.',
    href: '/03-features/14-wiki',
    tag: 'Knowledge organization',
  },
  {
    icon: 'version',
    title: 'Chunk-level editing and version management',
    desc: 'Parsing results can be corrected directly at chunk granularity; the index rebuilds immediately on save, and every edit keeps a historical version with rollback support. Wiki pages also have version history and distinguish three edit sources: pipeline, Agent, and manual.',
    href: '/03-features/02-knowledge-base',
    tag: 'Maintainability',
  },
  {
    icon: 'channels',
    title: 'Unified multi-channel access',
    desc: 'The same Agent can be published simultaneously to 10 IM platforms, an embed widget on your own site, and a browser extension, all sharing the same session, permission, and knowledge scope configuration.',
    href: '/03-features/12-im-integration',
    tag: 'Access',
  },
  {
    icon: 'mcp',
    title: 'Two-way MCP integration',
    desc: 'Connects as a client to external MCP services, with support for OAuth authorization and per-tool manual approval; can also act as an MCP Server exposing retrieval capability externally.',
    href: '/03-features/08-mcp',
    tag: 'Tool ecosystem',
  },
  {
    icon: 'govern',
    title: 'Multi-space isolation and auditing',
    desc: 'Workspace-level data isolation and a four-tier role matrix; API keys can be narrowed by capability and knowledge base scope. Operations are written to an audit log, background tasks have a queue dashboard, and the Q&A pipeline supports Langfuse tracing.',
    href: '/03-features/20-platform-admin',
    tag: 'Enterprise deployment',
  },
  {
    icon: 'pluggable',
    title: 'Pluggable architecture',
    desc: 'Parsing engines, chunking strategies, retrieval engines, model providers, search engines, and storage backends are all wired in through registries and can be swapped via configuration; a knowledge base can bind separate vector store and storage instances.',
    href: '/06-development/03-extension-points',
    tag: 'Architecture',
  },
  {
    icon: 'graph',
    title: 'Knowledge-graph-enhanced retrieval',
    desc: 'On ingestion, a model extracts entities and relationships into a graph database; at query time, recall is supplemented by following those relationships, answering questions like "what is the relationship between A and B" that vector retrieval handles poorly. Well suited to relationship-dense material such as people, organizations, and contract terms.',
    href: '/03-features/09-knowledge-graph',
    tag: 'Retrieval',
  },
  {
    icon: 'sync',
    title: 'Continuous data source sync',
    desc: 'Feishu, Notion, Yuque, and RSS sync automatically on a schedule once credentials are bound: a full sync the first time, then incremental pulls by modification time; documents deleted at the source are taken down in sync, keeping the knowledge base from going stale over time.',
    href: '/03-features/10-datasource',
    tag: 'Data ingestion',
  },
  {
    icon: 'faq',
    title: 'Precise FAQ answers',
    desc: 'Questions with fixed answers, like return policy or reimbursement process, can be maintained directly as Q&A pairs, matched by "standard question + similar questions + counter-examples" rather than by document fragment. It can be retrieved by the same Agent as the document library, forming a sequence that checks the standard answer first before turning to the documents.',
    href: '/03-features/17-faq',
    tag: 'Answer quality',
  },
]

const map = [
  {
    index: '01',
    icon: 'start',
    title: 'Getting started',
    brief: 'Read these four in order to complete deployment and run your first Q&A.',
    items: [
      { text: 'Product introduction', link: '/01-getting-started/01-introduction' },
      { text: 'Installation & deployment', link: '/01-getting-started/02-installation' },
      { text: 'Quickstart', link: '/01-getting-started/03-quickstart' },
      { text: 'Configuration reference', link: '/01-getting-started/04-configuration' },
    ],
  },
  {
    index: '02',
    icon: 'arch',
    title: 'Architecture',
    brief: 'The full system picture, plus the two main pipelines: document ingestion and retrieval Q&A.',
    items: [
      { text: 'Overall architecture', link: '/02-architecture/01-overview' },
      { text: 'Go backend design', link: '/02-architecture/02-backend-design' },
      { text: 'Document ingestion pipeline', link: '/02-architecture/03-document-pipeline' },
      { text: 'Retrieval Q&A pipeline', link: '/02-architecture/04-rag-pipeline' },
      { text: 'Async task system', link: '/02-architecture/05-async-tasks' },
    ],
  },
  {
    index: '03',
    icon: 'modules',
    title: 'Feature modules',
    brief: 'Configuration options, behavior contracts, and implementation paths for twenty-one capabilities.',
    items: [
      { text: 'Tenants, users, and auth', link: '/03-features/01-tenant-auth' },
      { text: 'Knowledge base and knowledge management', link: '/03-features/02-knowledge-base' },
      { text: 'docreader document parsing service', link: '/03-features/03-document-parsing' },
      { text: 'Chunking mechanism', link: '/03-features/04-chunking' },
      { text: 'Retrieval engines and vector storage', link: '/03-features/05-retrieval-engines' },
      { text: 'Model management', link: '/03-features/06-models' },
      { text: 'Agent engine', link: '/03-features/07-agent' },
      { text: 'MCP integration', link: '/03-features/08-mcp' },
      { text: 'Knowledge graph', link: '/03-features/09-knowledge-graph' },
      { text: 'Data source import', link: '/03-features/10-datasource' },
      { text: 'Web search and page scraping', link: '/03-features/11-web-search' },
      { text: 'IM integration', link: '/03-features/12-im-integration' },
      { text: 'Web embed', link: '/03-features/13-embed-channel' },
      { text: 'Wiki capability', link: '/03-features/14-wiki' },
      { text: 'Evaluation capability', link: '/03-features/15-evaluation' },
      { text: 'Observability and auditing', link: '/03-features/16-observability' },
      { text: 'FAQ capability', link: '/03-features/17-faq' },
      { text: 'Session and chat experience', link: '/03-features/18-chat-experience' },
      { text: 'Storage backends', link: '/03-features/19-storage-backends' },
      { text: 'Platform administration and system admin', link: '/03-features/20-platform-admin' },
      { text: 'External access to images and files', link: '/03-features/21-file-access' },
    ],
  },
  {
    index: '04',
    icon: 'api',
    title: 'API reference',
    brief: 'About 360 endpoints, with permission requirements, parameter tables, and curl examples.',
    items: [
      { text: 'API overview', link: '/04-api/01-api-overview' },
      { text: 'Agent, MCP, and skills', link: '/04-api/02-api-agent-mcp' },
      { text: 'Auth and users', link: '/04-api/02-api-auth' },
      { text: 'IM, Embed, and files', link: '/04-api/02-api-channels' },
      { text: 'Sessions, messages, and chat', link: '/04-api/02-api-chat' },
      { text: 'FAQ and Wiki', link: '/04-api/02-api-faq-wiki' },
      { text: 'Infrastructure and data sources', link: '/04-api/02-api-infra' },
      { text: 'Knowledge base and knowledge', link: '/04-api/02-api-knowledge' },
      { text: 'Chunks and tags', link: '/04-api/02-api-chunks' },
      { text: 'Models and initialization', link: '/04-api/02-api-model-system' },
      { text: 'System and platform administration', link: '/04-api/02-api-system' },
      { text: 'Organizations and sharing', link: '/04-api/02-api-org' },
      { text: 'Tenants and members', link: '/04-api/02-api-tenant' },
    ],
  },
  {
    index: '05',
    icon: 'clients',
    title: 'Clients',
    brief: 'Seven clients: Web, CLI, SDK, Mini Program, desktop, browser extension, and Skill.',
    items: [
      { text: 'Web frontend', link: '/05-clients/01-frontend' },
      { text: 'CLI tool', link: '/05-clients/02-cli' },
      { text: 'Go SDK', link: '/05-clients/03-go-sdk' },
      { text: 'WeChat Mini Program', link: '/05-clients/04-miniprogram' },
      { text: 'Desktop client', link: '/05-clients/05-desktop' },
      { text: 'Chrome extension', link: '/05-clients/06-chrome-extension' },
      { text: 'Claw Skill', link: '/05-clients/07-claw-skill' },
    ],
  },
  {
    index: '06',
    icon: 'dev',
    title: 'Development guide',
    brief: 'Local dev environment, database migrations, and nine categories of pluggable extension points.',
    items: [
      { text: 'Development guide', link: '/06-development/01-dev-guide' },
      { text: 'Database and migrations', link: '/06-development/02-database-schema' },
      { text: 'Extension point guide', link: '/06-development/03-extension-points' },
    ],
  },
]

const deployments = [
  { icon: 'compose', name: 'Docker Compose', desc: 'Standard deployment, 12 optional profile combinations for infrastructure', note: '' },
  { icon: 'helm', name: 'Helm', desc: 'Kubernetes cluster orchestration, suited to production multi-replica setups', note: '' },
  {
    icon: 'lite',
    name: 'Lite (single binary / desktop app)',
    desc: 'SQLite + in-process queue, no Docker or external database required; available as both a CLI and a GUI',
    note: 'The desktop app is not yet officially released and must be built from source',
  },
]
</script>

<template>
  <div class="landing">
    <!-- ============================= Hero ============================= -->
    <section class="hero">
      <div class="shell hero-grid">
        <div class="hero-copy">
          <p class="eyebrow">Tencent open source · WeKnora {{ versionLabel }} · Official docs</p>
          <h1 class="display">
            An open-source knowledge base Q&A system
          </h1>
          <p class="lede">WeKnora brings material from PDFs, Word documents, web pages, and sources like Feishu / Notion / Yuque into a knowledge base, providing retrieval-augmented Q&A that cites traceable sources in its answers. Beyond basic Q&A, it also offers <strong>automatic Wiki generation</strong>, <strong>two-way ReAct Agent and MCP integration</strong>, <strong>knowledge-graph-enhanced retrieval</strong>, and, for teams, <strong>multi-space isolation, four-tier RBAC, scoped API keys, and audit logs</strong>. It supports full private deployment, and models can all be swapped for local inference.</p>
          <p class="lede lede-sub">This documentation covers deployment and configuration, feature descriptions, a reference for about 360 API endpoints, and extension points for further development.</p>
          <div class="actions">
            <a class="btn btn-solid" :href="withBase('/01-getting-started/01-introduction')">Start reading</a>
            <a class="btn btn-ghost" :href="withBase('/02-architecture/01-overview')">System architecture</a>
            <a
              class="btn btn-text"
              href="https://weknora.weixin.qq.com"
              target="_blank"
              rel="noreferrer"
            >
              Official website ↗
            </a>
            <a
              class="btn btn-text"
              href="https://github.com/Tencent/WeKnora"
              target="_blank"
              rel="noreferrer"
            >
              GitHub repository ↗
            </a>
          </div>
        </div>

        <aside class="hero-side">
          <div class="schema" aria-label="Capability layers">
            <p class="schema-title">Capability layers</p>
            <ol class="schema-layers">
              <li v-for="layer in schema" :key="layer.name" class="layer">
                <span class="layer-step">{{ layer.step }}</span>
                <div class="layer-body">
                  <p class="layer-head">
                    <span class="layer-name">{{ layer.name }}</span>
                    <span class="layer-hint">{{ layer.hint }}</span>
                  </p>
                  <div class="layer-items">
                    <span v-for="n in layer.items" :key="n" class="chip">{{ n }}</span>
                  </div>
                </div>
              </li>
            </ol>
            <p class="schema-note">Each layer's implementation can be swapped out, and unused capabilities can be disabled; models support local deployment.</p>
          </div>
        </aside>
      </div>

      <svg class="wave" viewBox="0 0 1440 120" preserveAspectRatio="none" aria-hidden="true">
        <path
          d="M0 74C180 40 360 34 540 52c180 18 300 44 480 42s240-30 420-52"
          fill="none"
          stroke="var(--wk-ink)"
          stroke-opacity="0.16"
          stroke-width="1.5"
        />
        <path
          d="M0 92C200 62 380 58 560 74c180 16 300 40 470 36s250-26 410-44"
          fill="none"
          stroke="var(--wk-gold)"
          stroke-opacity="0.45"
          stroke-width="1.2"
        />
        <path
          d="M0 108C220 84 400 82 580 94c180 12 320 32 500 28s260-22 360-34"
          fill="none"
          stroke="var(--wk-ink)"
          stroke-opacity="0.08"
          stroke-width="1"
        />
      </svg>
    </section>

    <!-- ============================ Panorama ============================ -->
    <section class="panorama">
      <div class="shell">
        <Illus name="flow" class="panorama-illus" />
        <p class="panorama-note">
          Material comes in from files, web pages, audio, and images, is parsed uniformly, and is written in parallel into four indexes—vector, keyword, Wiki, and knowledge graph;
          the same knowledge base and Agent then extend out into nine client types, so switching access points never means switching systems.
        </p>
      </div>
    </section>

    <!-- ============================ Stats ============================ -->
    <section class="stats">
      <div class="shell stats-row">
        <div v-for="s in stats" :key="s.label" class="stat">
          <span class="stat-value">{{ s.value }}<i>{{ s.unit }}</i></span>
          <span class="stat-label">{{ s.label }}</span>
        </div>
      </div>
    </section>

    <!-- ============================ Main pipeline ============================ -->
    <section class="chapter">
      <div class="shell">
        <header class="chapter-head">
          <span class="marker">Processing pipeline</span>
          <h2 class="chapter-title">From a single PDF to an answer with sources attached</h2>
          <p class="chapter-sub">When Q&A quality is poor, the problem is often not the model but this pipeline: a scanned file's text wasn't extracted, a table got chopped up, or the retrieved passage doesn't match the question. Each of the four stages below corresponds to one class of failure point, and every implementation can be swapped out as needed.</p>
        </header>

        <ol class="chain">
          <li v-for="c in chain" :key="c.step" class="chain-item">
            <a :href="withBase(c.href)">
              <span class="chain-head">
                <Illus :name="c.icon" class="chain-icon" />
                <span class="chain-step">{{ c.step }}</span>
              </span>
              <h3 class="chain-title">{{ c.title }}</h3>
              <p class="chain-desc">{{ c.desc }}</p>
            </a>
          </li>
        </ol>
      </div>
    </section>

    <!-- ============================ Featured capabilities ============================ -->
    <section class="chapter chapter-alt">
      <div class="shell">
        <header class="chapter-head">
          <span class="marker">Core capabilities</span>
          <h2 class="chapter-title">Beyond basic retrieval Q&A</h2>
          <p class="chapter-sub">The following capabilities are WeKnora's main areas of investment, and can serve as comparison points when evaluating technology choices.</p>
        </header>

        <div class="features">
          <a v-for="f in features" :key="f.title" class="feature" :href="withBase(f.href)">
            <span class="feature-head">
              <Illus :name="f.icon" class="feature-icon" />
              <span class="feature-tag">{{ f.tag }}</span>
            </span>
            <h3 class="feature-title">{{ f.title }}</h3>
            <p class="feature-desc">{{ f.desc }}</p>
          </a>
        </div>
      </div>
    </section>

    <!-- ============================ Access methods ============================ -->
    <section class="chapter">
      <div class="shell">
        <header class="chapter-head">
          <span class="marker">Access methods</span>
          <h2 class="chapter-title">Nine clients and integration points</h2>
          <p class="chapter-sub">The same knowledge base and Agent configuration can be accessed from browsers, IM, your own site, terminals, and external agents, with no need to rebuild for each entry point.</p>
        </header>

        <div class="surfaces">
          <div v-for="s in surfaces" :key="s.name" class="surface">
            <h3 class="surface-name">
              <Illus :name="s.icon" class="surface-icon" />
              {{ s.name }}
            </h3>
            <p class="surface-desc">{{ s.desc }}</p>
          </div>
        </div>

        <div class="surfaces-actions">
          <a class="btn btn-ghost" :href="withBase('/05-clients/01-frontend')">View client docs</a>
          <a class="btn btn-text" :href="withBase('/03-features/13-embed-channel')">Web embed ↗</a>
          <a class="btn btn-text" :href="withBase('/03-features/12-im-integration')">IM integration ↗</a>
        </div>
      </div>
    </section>

    <!-- ============================ Documentation map ============================ -->
    <section class="chapter chapter-alt">
      <div class="shell">
        <header class="chapter-head">
          <span class="marker">Documentation map</span>
          <h2 class="chapter-title">Six sections, fifty-three articles</h2>
          <p class="chapter-sub">Covers deployment onboarding, system architecture, feature descriptions, API reference, clients, and further development.</p>
        </header>

        <div class="map">
          <section
            v-for="m in map"
            :key="m.index"
            class="map-block"
            :class="{ 'map-block-wide': m.items.length > 8 }"
          >
            <div class="map-head">
              <span class="map-index">{{ m.index }}</span>
              <h3 class="map-title">
                <Illus :name="m.icon" class="map-icon" />
                {{ m.title }}
              </h3>
              <p class="map-brief">{{ m.brief }}</p>
            </div>
            <ul class="map-list">
              <li v-for="it in m.items" :key="it.link">
                <a :href="withBase(it.link)">{{ it.text }}</a>
              </li>
            </ul>
          </section>
        </div>
      </div>
    </section>

    <!-- ============================ Deployment ============================ -->
    <section class="chapter">
      <div class="shell deploy">
        <div class="deploy-copy">
          <span class="marker">Deployment</span>
          <h2 class="chapter-title">Deployment forms</h2>
          <p class="chapter-sub">Standard deployment: clone the code, change two secrets, and bring up the full set of services with one command; for local trials, choose Lite mode, which doesn't depend on PostgreSQL or Redis.</p>
          <ul class="deploy-list">
            <li v-for="d in deployments" :key="d.name">
              <span class="deploy-name">
                <Illus :name="d.icon" class="deploy-icon" />
                {{ d.name }}
              </span>
              <span class="deploy-desc">
                {{ d.desc }}
                <span v-if="d.note" class="deploy-note">{{ d.note }}</span>
              </span>
            </li>
          </ul>
          <a class="btn btn-ghost" :href="withBase('/01-getting-started/02-installation')">View installation & deployment</a>
        </div>

        <div class="deploy-code">
          <div class="code-bar">
            <span>Standard deployment · Docker Compose</span>
          </div>
          <pre><code><span class="c"># 1 Get the code</span>
git clone https://github.com/Tencent/WeKnora.git
cd WeKnora

<span class="c"># 2 Prepare configuration: at minimum change JWT_SECRET and SYSTEM_AES_KEY</span>
cp .env.example .env

<span class="c"># 3 Bring up all services (first run needs to pull images)</span>
docker compose up -d --pull always

<span class="c"># 4 Confirm services are ready</span>
docker compose ps
curl http://localhost:8080/health

<span class="c"># 5 Open the frontend (port 80 by default, changeable via FRONTEND_PORT)</span>
open http://localhost

<span class="c"># Stop: docker compose down</span></code></pre>
          <p class="deploy-code-note">
            The first time you open the frontend you'll land on the sign-up page; after signing up, configure the chat model and embedding model in the initialization wizard, and you can create a knowledge base and start asking questions. The backend API shares the same domain as the frontend, at <code>http://localhost/api/v1</code>.
            For full steps see the <a :href="withBase('/01-getting-started/03-quickstart')">quickstart</a>, and for other deployment forms and parameters see <a :href="withBase('/01-getting-started/02-installation')">installation & deployment</a>.
          </p>
        </div>
      </div>
    </section>

    <!-- ============================ Closing ============================ -->
    <footer class="closing">
      <div class="shell closing-inner">
        <div class="closing-brand">
          <svg width="34" height="26" viewBox="0 0 34 26" fill="none" aria-hidden="true">
            <path
              d="M20.6 3.2c.36-.5 1.16-.22 1.13.39l-.53 10.2-6.9-.05c-.6 0-.86-.75-.4-1.13L20.6 3.2z"
              fill="currentColor"
            />
            <path
              d="M1.5 18.4c6.4-1.9 12.2-1.1 18.1.35 4.3 1.05 8.2 1.6 12.9.1"
              stroke="currentColor"
              stroke-width="2.1"
              stroke-linecap="round"
            />
            <path
              d="M4.4 22.1c5.6-1.35 10.8-.7 16 .5 3.8.87 7.2 1.2 11.3.15"
              stroke="var(--wk-gold)"
              stroke-width="1.4"
              stroke-linecap="round"
            />
          </svg>
          <span>WeKnora</span>
        </div>
        <p class="closing-note">This documentation is compiled from the {{ versionLabel }} source in the repository. Source paths are all relative to the repository root, API paths carry the <code>/api/v1</code> prefix by default, and the keys in configuration examples are all placeholders.</p>
        <div class="closing-links">
          <a :href="withBase('/01-getting-started/01-introduction')">Getting started</a>
          <a :href="withBase('/04-api/01-api-overview')">API overview</a>
          <a :href="withBase('/06-development/03-extension-points')">Extension points</a>
          <a href="https://github.com/Tencent/WeKnora" target="_blank" rel="noreferrer">GitHub</a>
        </div>
        <p class="closing-copy">© Tencent WeKnora · MIT License</p>
      </div>
    </footer>
  </div>
</template>

<style scoped>
.landing {
  /* Matches the same order of magnitude as the doc pages' full-viewport content column, to avoid the homepage looking noticeably narrower */
  --shell: 1600px;
  color: var(--wk-ink);
}

.shell {
  max-width: var(--shell);
  margin: 0 auto;
  padding: 0 32px;
}

.marker {
  display: inline-block;
  font-family: var(--wk-font-mono);
  font-size: 10.5px;
  letter-spacing: 0.2em;
  text-transform: uppercase;
  color: var(--wk-gold);
  margin-bottom: 22px;
}

.marker::before {
  content: "";
  display: inline-block;
  width: 22px;
  height: 1px;
  background: var(--wk-gold);
  vertical-align: middle;
  margin-right: 12px;
}

/* ------------------------------- Hero ------------------------------- */

.hero {
  position: relative;
  padding: clamp(80px, 13vw, 168px) 0 clamp(96px, 12vw, 150px);
  overflow: hidden;
}

.hero::before {
  content: "";
  position: absolute;
  inset: 0;
  background:
    radial-gradient(120% 80% at 78% -10%, rgba(184, 134, 59, 0.09), transparent 62%),
    radial-gradient(90% 70% at 8% 0%, rgba(16, 31, 56, 0.06), transparent 60%);
  pointer-events: none;
}

.hero-grid {
  position: relative;
  display: grid;
  grid-template-columns: minmax(0, 1.15fr) minmax(0, 0.85fr);
  gap: 76px;
  align-items: center;
}

.eyebrow {
  font-family: var(--wk-font-mono);
  font-size: 11px;
  letter-spacing: 0.22em;
  text-transform: uppercase;
  color: var(--wk-ink-mute);
  margin: 0 0 34px;
}

.display {
  font-family: var(--wk-font-serif);
  font-weight: 500;
  font-size: clamp(34px, 4.1vw, 58px);
  line-height: 1.2;
  letter-spacing: -0.025em;
  margin: 0;
}

.lede {
  margin: 34px 0 0;
  max-width: 60ch;
  font-size: 16.5px;
  line-height: 1.85;
  color: var(--wk-ink-soft);
}

/* The second paragraph explains what this site itself is; toned down a notch so it doesn't compete with the positioning sentence for attention */
.lede-sub {
  margin-top: 14px;
  font-size: 15px;
  color: var(--wk-ink-mute);
}

.actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 14px;
  margin-top: 46px;
}

.btn {
  display: inline-flex;
  align-items: center;
  height: 46px;
  padding: 0 26px;
  border-radius: 2px;
  font-size: 14px;
  font-weight: 550;
  letter-spacing: 0.02em;
  text-decoration: none;
  transition: all 0.22s ease;
}

.btn-solid {
  background: var(--wk-ink);
  color: var(--wk-paper);
  border: 1px solid var(--wk-ink);
}

.btn-solid:hover {
  background: transparent;
  color: var(--wk-ink);
}

.btn-ghost {
  border: 1px solid var(--wk-rule);
  color: var(--wk-ink);
}

.btn-ghost:hover {
  border-color: var(--wk-gold);
  color: var(--wk-gold);
}

.btn-text {
  padding: 0 6px;
  color: var(--wk-ink-mute);
}

.btn-text:hover {
  color: var(--wk-ink);
}

/* System component overview */

/* Panorama: the full pipeline plus nine clients spans very wide horizontally,
   and squeezing it into the hero's right-column card would crowd it, so it gets its own full-width band */
.panorama {
  padding: clamp(18px, 3vw, 40px) 0 clamp(28px, 4vw, 56px);
}

.panorama-illus {
  display: block;
  width: 100%;
  max-width: 1180px;
  height: auto;
  margin: 0 auto;
  color: var(--wk-ink);
  opacity: 0.9;
}

.panorama-note {
  max-width: 860px;
  margin: 26px auto 0;
  text-align: center;
  font-size: 13.5px;
  line-height: 1.8;
  color: var(--wk-ink-mute);
}

/* Capability layers: an index column plus a connecting vertical axis express top-down flow,
   since previously only a small arrow between layers left readers unable to tell this was a pipeline */
.schema {
  padding: 26px 26px 22px;
  border: 1px solid var(--wk-rule);
  border-radius: 4px;
  background: color-mix(in srgb, var(--wk-paper-2) 60%, transparent);
}

.schema-title {
  margin: 0 0 20px;
  font-family: var(--wk-font-mono);
  font-size: 10.5px;
  letter-spacing: 0.16em;
  text-transform: uppercase;
  color: var(--wk-ink-mute);
}

.schema-layers {
  margin: 0;
  padding: 0;
  list-style: none;
}

.layer {
  position: relative;
  display: grid;
  grid-template-columns: 34px 1fr;
  gap: 14px;
  padding: 0 0 22px;
}

.layer:last-child {
  padding-bottom: 0;
}

/* The vertical axis connects each layer's index number; the last layer doesn't extend further down */
.layer:not(:last-child)::before {
  content: "";
  position: absolute;
  left: 11px;
  top: 24px;
  bottom: 4px;
  width: 1px;
  background: var(--wk-rule);
}

.layer-step {
  z-index: 1;
  display: grid;
  place-items: center;
  width: 23px;
  height: 23px;
  border: 1px solid var(--wk-rule);
  border-radius: 50%;
  background: var(--wk-paper);
  font-family: var(--wk-font-mono);
  font-size: 10px;
  color: var(--wk-ink-mute);
}

.layer-head {
  display: flex;
  align-items: baseline;
  gap: 10px;
  margin: 3px 0 10px;
}

.layer-name {
  font-size: 14px;
  font-weight: 600;
  letter-spacing: 0.02em;
  color: var(--wk-ink);
}

.layer-hint {
  font-size: 12px;
  color: var(--wk-ink-mute);
}

.layer-items {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.chip {
  font-size: 12px;
  line-height: 1.5;
  padding: 4px 10px;
  border: 1px solid var(--wk-rule);
  border-radius: 2px;
  color: var(--wk-ink-soft);
  background: color-mix(in srgb, var(--wk-paper) 70%, transparent);
  white-space: nowrap;
}

.schema-note {
  margin: 22px 0 0;
  padding-top: 16px;
  border-top: 1px solid var(--wk-rule-soft);
  font-size: 12px;
  line-height: 1.7;
  color: var(--wk-ink-mute);
}

.wave {
  position: absolute;
  left: 0;
  right: 0;
  bottom: -1px;
  width: 100%;
  height: 120px;
  pointer-events: none;
}

/* ------------------------------- Stats ------------------------------- */

.stats {
  border-top: 1px solid var(--wk-rule-soft);
  border-bottom: 1px solid var(--wk-rule-soft);
  background: var(--wk-paper-2);
}

.stats-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
}

@media (max-width: 900px) {
  .stats-row {
    grid-template-columns: repeat(2, 1fr);
  }
}

.stat {
  padding: 34px 0;
  text-align: center;
  border-left: 1px solid var(--wk-rule-soft);
}

.stat:first-child {
  border-left: none;
}

.stat-value {
  display: block;
  font-family: var(--wk-font-serif);
  font-size: 34px;
  line-height: 1;
  font-weight: 500;
  letter-spacing: -0.02em;
}

.stat-value i {
  font-style: normal;
  font-size: 0.6em;
  color: var(--wk-gold);
  margin-left: 2px;
}

.stat-label {
  display: block;
  /* Chinese text could break at any character; a width cutting off mid-word would previously split awkwardly.
     Relaxed by one notch and let the browser balance the two lines so word groups more easily land on the same line */
  max-width: 26ch;
  /* keep-all lets Chinese text break only at commas/colons, not mid-word */
  word-break: keep-all;
  text-wrap: balance;
  margin: 12px auto 0;
  font-size: 12.5px;
  line-height: 1.7;
  letter-spacing: 0.04em;
  color: var(--wk-ink-mute);
}

/* ------------------------------ Common chapter styles ------------------------------ */

.chapter {
  padding: clamp(72px, 9vw, 124px) 0;
}

.chapter-alt {
  background: var(--wk-paper-2);
  border-top: 1px solid var(--wk-rule-soft);
  border-bottom: 1px solid var(--wk-rule-soft);
}

.chapter-head {
  max-width: 780px;
  margin-bottom: 68px;
}

.chapter-title {
  font-family: var(--wk-font-serif);
  font-weight: 500;
  font-size: clamp(27px, 3.2vw, 40px);
  line-height: 1.3;
  letter-spacing: -0.02em;
  margin: 0;
  border: none;
  padding: 0;
}

.chapter-sub {
  margin: 20px 0 0;
  font-size: 15.5px;
  line-height: 1.85;
  color: var(--wk-ink-soft);
}

/* ------------------------------- Main pipeline ------------------------------- */

.chain {
  list-style: none;
  margin: 0;
  padding: 0;
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  border-top: 1px solid var(--wk-rule);
}

.chain-item {
  border-left: 1px solid var(--wk-rule-soft);
}

.chain-item:first-child {
  border-left: none;
}

.chain-item a {
  display: block;
  height: 100%;
  padding: 32px 28px 40px 0;
  text-decoration: none;
  color: inherit;
  transition: opacity 0.22s ease;
}

.chain-item:not(:first-child) a {
  padding-left: 28px;
}

.chain-item a:hover {
  opacity: 0.62;
}

.chain-head {
  display: flex;
  align-items: center;
  gap: 12px;
}

.chain-icon {
  width: 26px;
  height: 26px;
  flex: none;
  color: var(--wk-ink);
}

.chain-step {
  font-family: var(--wk-font-mono);
  font-size: 11px;
  letter-spacing: 0.18em;
  color: var(--wk-gold);
}

.chain-title {
  font-family: var(--wk-font-serif);
  font-size: 21px;
  font-weight: 500;
  margin: 16px 0 14px;
  letter-spacing: -0.01em;
}

.chain-desc {
  margin: 0;
  font-size: 14px;
  line-height: 1.8;
  color: var(--wk-ink-soft);
}

/* ------------------------------ Featured capabilities ------------------------------ */

.features {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  border-top: 1px solid var(--wk-rule);
}

.feature {
  display: block;
  padding: 30px 30px 36px 0;
  border-bottom: 1px solid var(--wk-rule-soft);
  border-left: 1px solid var(--wk-rule-soft);
  text-decoration: none;
  color: inherit;
  transition: opacity 0.22s ease;
}

.feature:nth-child(3n + 1) {
  border-left: none;
}

.feature:not(:nth-child(3n + 1)) {
  padding-left: 30px;
}

.feature:nth-last-child(-n + 3) {
  border-bottom: none;
}

.feature:hover {
  opacity: 0.62;
}

.feature-head {
  display: flex;
  align-items: center;
  gap: 11px;
}

.feature-icon {
  width: 24px;
  height: 24px;
  flex: none;
  color: var(--wk-ink);
}

.feature-tag {
  font-family: var(--wk-font-mono);
  font-size: 11px;
  letter-spacing: 0.16em;
  color: var(--wk-gold);
}

.feature-title {
  font-family: var(--wk-font-serif);
  font-size: 20px;
  font-weight: 500;
  margin: 14px 0 12px;
  letter-spacing: -0.01em;
}

.feature-desc {
  margin: 0;
  font-size: 14px;
  line-height: 1.8;
  color: var(--wk-ink-soft);
}

/* ------------------------------ Access methods ------------------------------ */

.surfaces {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 28px 56px;
}

.surface {
  padding-top: 18px;
  border-top: 1px solid var(--wk-rule-soft);
}

.surface-name {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 15px;
  font-weight: 600;
  margin: 0 0 8px;
  letter-spacing: 0.01em;
}

.surface-icon {
  width: 20px;
  height: 20px;
  flex: none;
  color: var(--wk-ink);
}

.surface-desc {
  margin: 0;
  font-size: 13.5px;
  line-height: 1.75;
  color: var(--wk-ink-soft);
}

.surfaces-actions {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 18px;
  margin-top: 44px;
}

/* ------------------------------ Documentation map ------------------------------ */

.map {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0 72px;
}

.map-block {
  padding: 40px 0;
  border-top: 1px solid var(--wk-rule);
}

.map-head {
  display: grid;
  grid-template-columns: 46px 1fr;
  align-items: baseline;
  row-gap: 10px;
}

.map-index {
  font-family: var(--wk-font-mono);
  font-size: 11px;
  letter-spacing: 0.14em;
  color: var(--wk-gold);
}

.map-title {
  display: flex;
  align-items: center;
  gap: 12px;
  font-family: var(--wk-font-serif);
  font-size: 24px;
  font-weight: 500;
  margin: 0;
  letter-spacing: -0.01em;
}

.map-icon {
  width: 22px;
  height: 22px;
  flex: none;
  color: var(--wk-ink);
}

.map-brief {
  grid-column: 2;
  margin: 0;
  font-size: 13.8px;
  line-height: 1.75;
  color: var(--wk-ink-mute);
}

.map-list {
  list-style: none;
  margin: 26px 0 0 46px;
  padding: 0;
  columns: 1;
}

.map-block-wide {
  grid-column: 1 / -1;
}

.map-block-wide .map-list {
  columns: 3;
  column-gap: 72px;
}

.map-block-wide .map-brief {
  max-width: 52ch;
}

.map-list li {
  border-bottom: 1px solid var(--wk-rule-soft);
}

.map-list a {
  display: block;
  padding: 11px 0;
  font-size: 14.5px;
  color: var(--wk-ink-soft);
  text-decoration: none;
  transition: color 0.18s ease, padding-left 0.18s ease;
}

.map-list a:hover {
  color: var(--wk-ink);
  padding-left: 8px;
}

/* ------------------------------- Deployment ------------------------------- */

.deploy {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1.05fr);
  gap: 72px;
  align-items: start;
}

.deploy-list {
  list-style: none;
  margin: 38px 0;
  padding: 0;
  border-top: 1px solid var(--wk-rule-soft);
}

.deploy-list li {
  display: grid;
  /* The name column needs to fit an icon plus "Lite (single binary / desktop app)", so it's a notch wider than before */
  grid-template-columns: 178px 1fr;
  gap: 16px;
  padding: 15px 0;
  border-bottom: 1px solid var(--wk-rule-soft);
}

.deploy-name {
  display: grid;
  grid-template-columns: 20px 1fr;
  align-items: start;
  gap: 10px;
  font-size: 14px;
  font-weight: 600;
}

.deploy-icon {
  width: 20px;
  height: 20px;
  margin-top: 1px;
  color: var(--wk-ink);
}

/* A qualifier like "not yet officially released" follows the description. Placing it in the name column would get cut off by the 150px narrow column,
   whereas here it stays adjacent to its item without breaking the wording. */
.deploy-note {
  display: inline-block;
  margin-left: 6px;
  padding: 1px 7px;
  border: 1px solid var(--wk-rule);
  border-radius: 2px;
  font-family: var(--wk-font-mono);
  font-size: 10.5px;
  line-height: 1.7;
  letter-spacing: 0.04em;
  white-space: nowrap;
  color: var(--wk-ink-mute);
}

.deploy-desc {
  font-size: 13.8px;
  line-height: 1.7;
  color: var(--wk-ink-mute);
}

.deploy-code {
  border: 1px solid var(--wk-rule);
  border-radius: 4px;
  overflow: hidden;
  background: #0d1626;
}

.code-bar {
  padding: 12px 20px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
  font-family: var(--wk-font-mono);
  font-size: 10.5px;
  letter-spacing: 0.18em;
  text-transform: uppercase;
  color: rgba(230, 236, 245, 0.45);
}

.deploy-code pre {
  margin: 0;
  padding: 24px 22px 28px;
  overflow-x: auto;
}

.deploy-code code {
  font-family: var(--wk-font-mono);
  font-size: 13px;
  line-height: 2;
  color: #dfe7f2;
}

.deploy-code .c {
  color: rgba(217, 169, 79, 0.75);
}

/* The code block itself only gets to "the services are up"; follow-up actions and further reading go in the line below,
   so readers don't mistake finishing the commands for being done */
.deploy-code-note {
  margin: 0;
  padding: 16px 22px 20px;
  border-top: 1px solid rgba(255, 255, 255, 0.08);
  font-size: 12.5px;
  line-height: 1.9;
  color: rgba(223, 231, 242, 0.62);
}

.deploy-code-note code {
  font-family: var(--wk-font-mono);
  font-size: 12px;
  color: rgba(223, 231, 242, 0.85);
}

.deploy-code-note a {
  color: var(--wk-gold-light);
  text-decoration: none;
  border-bottom: 1px solid rgba(217, 169, 79, 0.4);
}

.deploy-code-note a:hover {
  border-bottom-color: var(--wk-gold-light);
}

/* ------------------------------- Closing ------------------------------- */

.closing {
  border-top: 1px solid var(--wk-rule-soft);
  background: var(--wk-paper-2);
  padding: 70px 0 60px;
}

.closing-inner {
  display: grid;
  gap: 26px;
  justify-items: center;
  text-align: center;
}

.closing-brand {
  display: flex;
  align-items: center;
  gap: 12px;
  font-family: var(--wk-font-sans);
  font-size: 14px;
  font-weight: 600;
  letter-spacing: 0.18em;
  text-transform: uppercase;
}

.closing-note {
  max-width: 58ch;
  margin: 0;
  font-size: 13.5px;
  line-height: 1.9;
  color: var(--wk-ink-mute);
}

.closing-note code {
  font-family: var(--wk-font-mono);
  font-size: 0.9em;
}

.closing-links {
  display: flex;
  flex-wrap: wrap;
  gap: 28px;
}

.closing-links a {
  font-size: 13.5px;
  color: var(--wk-ink-soft);
  text-decoration: none;
  border-bottom: 1px solid transparent;
  padding-bottom: 2px;
  transition: all 0.2s ease;
}

.closing-links a:hover {
  color: var(--wk-ink);
  border-bottom-color: var(--wk-gold);
}

.closing-copy {
  margin: 0;
  font-family: var(--wk-font-mono);
  font-size: 11px;
  letter-spacing: 0.1em;
  color: var(--wk-ink-mute);
}

/* ------------------------------ Responsive ------------------------------ */

@media (max-width: 1080px) {
  .hero-grid {
    grid-template-columns: minmax(0, 1fr);
    gap: 56px;
  }
  .map-block-wide .map-list {
    columns: 2;
    column-gap: 48px;
  }
  .chain {
    grid-template-columns: repeat(2, 1fr);
  }
  .chain-item:nth-child(odd) {
    border-left: none;
  }
  .chain-item:nth-child(n + 3) {
    border-top: 1px solid var(--wk-rule-soft);
  }
  .chain-item a,
  .chain-item:not(:first-child) a {
    padding-left: 0;
  }
  .chain-item:nth-child(even) a {
    padding-left: 28px;
  }
  .features {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .feature,
  .feature:not(:nth-child(3n + 1)) {
    padding-left: 0;
    border-left: none;
  }
  .feature:nth-child(even) {
    padding-left: 30px;
    border-left: 1px solid var(--wk-rule-soft);
  }
  .feature:nth-last-child(-n + 3) {
    border-bottom: 1px solid var(--wk-rule-soft);
  }
  .feature:nth-last-child(-n + 2) {
    border-bottom: none;
  }
  .surfaces {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 24px 40px;
  }
  .map {
    grid-template-columns: minmax(0, 1fr);
    gap: 0;
  }
  .deploy {
    grid-template-columns: minmax(0, 1fr);
    gap: 48px;
  }
  .stats-row {
    grid-template-columns: repeat(3, 1fr);
  }
  .stat:nth-child(3n + 1) {
    border-left: none;
  }
  .stat:nth-child(n + 4) {
    border-top: 1px solid var(--wk-rule-soft);
  }
}

@media (max-width: 640px) {
  .shell {
    padding: 0 22px;
  }
  .display br {
    display: none;
  }
  .chain {
    grid-template-columns: minmax(0, 1fr);
  }
  .chain-item {
    border-left: none;
  }
  .chain-item:not(:first-child) {
    border-top: 1px solid var(--wk-rule-soft);
  }
  .chain-item:nth-child(even) a {
    padding-left: 0;
  }
  .surfaces {
    grid-template-columns: minmax(0, 1fr);
    gap: 20px;
  }
  .features {
    grid-template-columns: minmax(0, 1fr);
  }
  .feature,
  .feature:nth-child(even) {
    padding-left: 0;
    border-left: none;
    border-bottom: 1px solid var(--wk-rule-soft);
  }
  .feature:last-child {
    border-bottom: none;
  }
  .stats-row {
    grid-template-columns: repeat(2, 1fr);
  }
  .stat {
    border-left: 1px solid var(--wk-rule-soft);
  }
  .stat:nth-child(odd) {
    border-left: none;
  }
  .stat:nth-child(n + 3) {
    border-top: 1px solid var(--wk-rule-soft);
  }
  .map-block-wide .map-list {
    columns: 1;
  }
  .map-head {
    grid-template-columns: 1fr;
  }
  .map-brief {
    grid-column: 1;
  }
  .map-list {
    margin-left: 0;
  }
  .deploy-list li {
    grid-template-columns: 1fr;
    gap: 6px;
  }
}
</style>
