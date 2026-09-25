import { createI18n } from 'vue-i18n'
import zhCN from './locales/zh-CN.ts'
import ruRU from './locales/ru-RU.ts'
import enUS from './locales/en-US.ts'
import koKR from './locales/ko-KR.ts'
import jaJP from './locales/ja-JP.ts'
import { BUILT_IN_DEFAULT, resolveDefaultLocale } from './resolveDefaultLocale.ts'

const messages = {
  'zh-CN': zhCN,
  'en-US': enUS,
  'ru-RU': ruRU,
  'ko-KR': koKR,
  'ja-JP': jaJP
}

const deploymentLocale = resolveDefaultLocale(
  window.__RUNTIME_CONFIG__?.DEFAULT_LOCALE,
  import.meta.env.VITE_DEFAULT_LOCALE,
)

// Migration: honor the English-default rollout. A locale saved before this
// release (typically a stale zh-CN) is reset to the deployment default once.
// Users can still switch languages in Settings; the choice is saved going forward.
const MIGRATED_KEY = 'weknora-locale-migrated-v2'
let savedLocale = localStorage.getItem('locale')
if (!localStorage.getItem(MIGRATED_KEY)) {
  localStorage.setItem(MIGRATED_KEY, '1')
  if (savedLocale) {
    savedLocale = deploymentLocale
    localStorage.setItem('locale', deploymentLocale)
  }
}

// User's explicit past choice wins; otherwise use the deployment default.
const i18n = createI18n({
  legacy: false,
  locale: savedLocale || deploymentLocale,
  fallbackLocale: BUILT_IN_DEFAULT,
  globalInjection: true,
  // Some translations intentionally embed `<strong>` markup (e.g. agent step summaries).
  // We render them via v-html with our own sanitization, so silence vue-i18n's HTML warning
  // to avoid flooding the console and slowing renders during history loads.
  warnHtmlMessage: false,
  messages
})

export default i18n
