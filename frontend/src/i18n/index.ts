import { createI18n } from 'vue-i18n'
import zhCN from './locales/zh-CN.ts'
import ruRU from './locales/ru-RU.ts'
import enUS from './locales/en-US.ts'
import koKR from './locales/ko-KR.ts'

const messages = {
  'zh-CN': zhCN,
  'en-US': enUS,
  'ru-RU': ruRU,
  'ko-KR': koKR
}

// Migration: honor the English-default rollout. If a stale zh-CN was saved
// before this release, default to English on first load. Users can still switch
// languages in Settings; the choice is saved going forward.
const savedLocale = localStorage.getItem('locale')
const MIGRATED_KEY = 'weknora-locale-migrated-v2'
let resolvedLocale: string
if (localStorage.getItem(MIGRATED_KEY)) {
  resolvedLocale = savedLocale || 'en-US'
} else {
  localStorage.setItem(MIGRATED_KEY, '1')
  resolvedLocale = 'en-US'
  if (savedLocale) localStorage.setItem('locale', 'en-US')
}

const i18n = createI18n({
  legacy: false,
  locale: resolvedLocale,
  fallbackLocale: 'en-US',
  globalInjection: true,
  // Some translations intentionally embed `<strong>` markup (e.g. agent step summaries).
  // We render them via v-html with our own sanitization, so silence vue-i18n's HTML warning
  // to avoid flooding the console and slowing renders during history loads.
  warnHtmlMessage: false,
  messages
})

export default i18n