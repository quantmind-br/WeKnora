import { readFileSync, readdirSync } from 'node:fs'
import { resolve } from 'node:path'
import { defineConfig, type DefaultTheme } from 'vitepress'
import { withMermaid } from 'vitepress-plugin-mermaid'
import './theme-config.d.ts'
import { repoVersionLabel } from './version'

const root = resolve(import.meta.dirname, '..')

const sections: { dir: string; label: string; newestFirst?: boolean }[] = [
  { dir: '01-getting-started', label: 'Quick Start' },
  { dir: '02-architecture', label: 'Architecture' },
  { dir: '03-features', label: 'Features' },
  { dir: '04-api', label: 'API Reference' },
  { dir: '05-clients', label: 'Clients' },
  { dir: '06-development', label: 'Developer Guide' },
  // One page per release (v0.8.2.md), newest release first
  { dir: '07-releases', label: 'Releases', newestFirst: true },
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

/** 版本号按数字比较，v0.10.0 排在 v0.9.0 之后 */
function compareVersions(a: string, b: string): number {
  const parts = (f: string) => f.replace(/^v|\.md$/g, '').split('.').map(Number)
  const [x, y] = [parts(a), parts(b)]
  for (let i = 0; i < Math.max(x.length, y.length); i++) {
    if ((x[i] ?? 0) !== (y[i] ?? 0)) return (x[i] ?? 0) - (y[i] ?? 0)
  }
  return 0
}

function itemsOf(dir: string, newestFirst = false): DefaultTheme.SidebarItem[] {
  const files = readdirSync(resolve(root, dir)).filter((f) => f.endsWith('.md'))
  return (newestFirst ? files.sort(compareVersions).reverse() : files.sort())
    .map((f) => ({
      text: itemText(dir, f),
      link: `/${dir}/${f.replace(/\.md$/, '')}`,
    }))
}

const sidebar: DefaultTheme.SidebarItem[] = sections.map((s) => ({
  text: s.label,
  collapsed: false,
  items: itemsOf(s.dir, s.newestFirst),
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

export default withMermaid(
  defineConfig({
    title: 'WeKnora',
    titleTemplate: ':title · WeKnora Docs',
    description: 'Official WeKnora documentation: deployment, configuration, feature guides, API reference and development',
    lang: 'zh-CN',
    base: '/docs/',
    cleanUrls: true,
    appearance: { storageKey: 'vitepress-theme-appearance' },
    lastUpdated: true,
    srcExclude: ['README.md', 'MIGRATION.md', 'homepage/**', 'shared/**', 'scripts/**', 'deploy/**', 'static-site/**', 'releases/**'],
    metaChunk: true,
    transformPageData(pageData) {
      // The shared masthead replaces the default documentation navbar.
      pageData.frontmatter.navbar = false
    },

    head: [
      ['link', { rel: 'icon', href: '/docs/favicon.ico', type: 'image/x-icon' }],
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
      logoLink: { link: '/', target: '_self' },
      siteTitle: 'WeKnora',

      nav: [],

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
