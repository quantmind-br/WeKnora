import { readFileSync, readdirSync } from 'node:fs'
import { resolve } from 'node:path'
import { defineConfig, type DefaultTheme } from 'vitepress'
import { withMermaid } from 'vitepress-plugin-mermaid'
import './theme-config.d.ts'
import { repoVersionLabel } from './version'

const root = resolve(import.meta.dirname, '..')

const sections: { dir: string; label: string }[] = [
  { dir: '01-getting-started', label: 'Quick Start' },
  { dir: '02-architecture', label: 'Architecture' },
  { dir: '03-features', label: 'Features' },
  { dir: '04-api', label: 'API Reference' },
  { dir: '05-clients', label: 'Clients' },
  { dir: '06-development', label: 'Developer Guide' },
]

/** Sidebar item text: take the body H1 and strip redundant prefixes/suffixes */
function itemText(dir: string, file: string): string {
  const raw = readFileSync(resolve(root, dir, file), 'utf-8')
  const heading = raw.match(/^#\s+(.+)$/m)?.[1] ?? file.replace(/\.md$/, '')
  return heading
    .replace(/^API Reference[:：]\s*/, '')
    .replace(/\s*[（(][^（()）]*[)）]\s*/g, ' ')
    .replace(/\s{2,}/g, ' ')
    .trim()
}

function itemsOf(dir: string): DefaultTheme.SidebarItem[] {
  return readdirSync(resolve(root, dir))
    .filter((f) => f.endsWith('.md'))
    .sort()
    .map((f) => ({
      text: itemText(dir, f),
      link: `/${dir}/${f.replace(/\.md$/, '')}`,
    }))
}

const sidebar: DefaultTheme.SidebarItem[] = sections.map((s) => ({
  text: s.label,
  collapsed: false,
  items: itemsOf(s.dir),
}))

/** Local search tokenizes by whitespace by default; a whole CJK run would be treated as one word, so degrade to per-character splitting here */
function tokenize(text: string): string[] {
  const tokens: string[] = []
  for (const part of text.split(/[\s\n\r#%*,=/:;?[\]{}()&+\-!'"$·、，。：；？！（）【】《》…—]+/)) {
    if (!part) continue
    if (/[\u4e00-\u9fa5]/.test(part)) {
      tokens.push(part, ...part.split(''))
    } else {
      tokens.push(part)
    }
  }
  return tokens
}

const repo = 'https://github.com/Tencent/WeKnora'
const site = 'https://weknora.weixin.qq.com'

export default withMermaid(
  defineConfig({
    title: 'WeKnora',
    titleTemplate: ':title · WeKnora Docs',
    description: 'Official WeKnora documentation: deployment, configuration, feature guides, API reference and development',
    lang: 'zh-CN',
    base: '/docs/',
    cleanUrls: true,
    lastUpdated: true,
    srcExclude: ['README.md'],
    metaChunk: true,

    head: [
      ['link', { rel: 'icon', href: '/favicon.svg', type: 'image/svg+xml' }],
      ['meta', { name: 'theme-color', content: '#101f38' }],
      ['meta', { property: 'og:type', content: 'website' }],
      ['meta', { property: 'og:title', content: 'WeKnora Docs' }],
      [
        'meta',
        {
          property: 'og:description',
          content: 'Turn PDFs, Word docs, web pages and Feishu / Notion / Yuque materials into a knowledge base with cited answers',
        },
      ],
    ],

    markdown: {
      theme: { light: 'github-light', dark: 'github-dark' },
      lineNumbers: false,
      toc: { level: [2, 3] },
      config(md) {
        // Wrap tables in a scroll container. The default theme sets <table> itself to display:block
        // for horizontal scrolling; the side effect is tables shrink to their content, so narrow tables
        // end up slimmer than the prose column. Moving the scroll to the wrapper lets tables use display:table + width:100% and fill the prose column uniformly.
        md.renderer.rules.table_open = () => '<div class="wk-table">\n<table>\n'
        md.renderer.rules.table_close = () => '</table>\n</div>\n'
      },
    },

    themeConfig: {
      logo: { light: '/logo-mark.svg', dark: '/logo-mark-dark.svg', alt: 'WeKnora' },
      siteTitle: 'WeKnora',

      nav: [
        { text: 'Quick Start', link: '/01-getting-started/01-introduction', activeMatch: '/01-getting-started/' },
        { text: 'Architecture', link: '/02-architecture/01-overview', activeMatch: '/02-architecture/' },
        { text: 'Features', link: '/03-features/01-tenant-auth', activeMatch: '/03-features/' },
        { text: 'API', link: '/04-api/01-api-overview', activeMatch: '/04-api/' },
        { text: 'Clients', link: '/05-clients/01-frontend', activeMatch: '/05-clients/' },
        { text: 'Developer', link: '/06-development/01-dev-guide', activeMatch: '/06-development/' },
        { text: 'Website', link: site },
      ],

      weknoraVersion: repoVersionLabel,

      sidebar,

      socialLinks: [{ icon: 'github', link: repo }],

      outline: { level: [2, 3], label: 'On This Page' },

      docFooter: { prev: 'Previous', next: 'Next' },
      returnToTopLabel: 'Back to top',
      sidebarMenuLabel: 'Contents',
      darkModeSwitchLabel: 'Appearance',
      lightModeSwitchTitle: 'Switch to light mode',
      darkModeSwitchTitle: 'Switch to dark mode',

      lastUpdated: {
        text: 'Last updated',
        formatOptions: { dateStyle: 'medium', timeStyle: undefined },
      },

      editLink: {
        pattern: `${repo}/edit/main/website-docs/:path`,
        text: 'Edit this page on GitHub',
      },

      search: {
        provider: 'local',
        options: {
          translations: {
            button: { buttonText: 'Search docs', buttonAriaLabel: 'Search docs' },
            modal: {
              displayDetails: 'Expand details',
              resetButtonTitle: 'Clear',
              backButtonTitle: 'Back',
              noResultsText: 'No results found',
              footer: {
                selectText: 'Select',
                navigateText: 'Navigate',
                closeText: 'Close',
              },
            },
          },
          miniSearch: {
            options: { tokenize },
            searchOptions: {
              fuzzy: 0.2,
              prefix: true,
              boost: { title: 4, text: 2, titles: 1 },
            },
          },
        },
      },

      footer: {
        message: `Built from the WeKnora ${repoVersionLabel} source · MIT License`,
        copyright: '© Tencent WeKnora',
      },
    },

    mermaid: {
      theme: 'base',
      fontFamily:
        '"PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", ui-sans-serif, sans-serif',
      themeVariables: {
        primaryColor: '#eef1f6',
        primaryTextColor: '#101f38',
        primaryBorderColor: '#9aa8bd',
        lineColor: '#7d8ba1',
        secondaryColor: '#faf7ef',
        tertiaryColor: '#f6f7f9',
        fontSize: '14px',
      },
    },
  }),
)
