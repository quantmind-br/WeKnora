import 'vitepress'

declare module 'vitepress/dist/client/theme-default/config' {
  export interface ThemeConfig {
    /** WeKnora release version shown on the docs site (from the VERSION file at the repo root) */
    weknoraVersion?: string
  }
}
