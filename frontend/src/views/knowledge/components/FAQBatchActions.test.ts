import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'

const manager = readFileSync(new URL('./FAQEntryManager.vue', import.meta.url), 'utf8')
const batchBar = readFileSync(new URL('./FAQBatchBar.vue', import.meta.url), 'utf8')
const locales = [
  readFileSync(new URL('../../../i18n/locales/zh-CN.ts', import.meta.url), 'utf8'),
  readFileSync(new URL('../../../i18n/locales/en-US.ts', import.meta.url), 'utf8'),
  readFileSync(new URL('../../../i18n/locales/ko-KR.ts', import.meta.url), 'utf8'),
  readFileSync(new URL('../../../i18n/locales/ja-JP.ts', import.meta.url), 'utf8'),
  readFileSync(new URL('../../../i18n/locales/ru-RU.ts', import.meta.url), 'utf8'),
]

test('FAQ batch actions connect directly through component events instead of global events', () => {
  assert.match(manager, /import FAQBatchBar from '\.\/FAQBatchBar\.vue'/)
  assert.match(manager, /<FAQBatchBar[\s\S]*?@batch-tag="openBatchTagDialog"/)
  assert.match(manager, /@enable="handleBatchStatusChange\(true\)"/)
  assert.match(manager, /@disable="handleBatchStatusChange\(false\)"/)
  assert.match(manager, /@delete="handleBatchDelete"/)
  assert.doesNotMatch(manager, /faqMenuAction/)
  assert.doesNotMatch(manager, /faqSelectionChanged/)
})

test('the batch action bar shows the actions valid for the selection state and permissions', () => {
  assert.match(batchBar, /v-if="count > 0 && \(canEdit \|\| canManage\)"/)
  assert.match(batchBar, /v-if="canEdit && disabledCount > 0"/)
  assert.match(batchBar, /v-if="canEdit && enabledCount > 0"/)
  assert.match(batchBar, /<t-popconfirm v-if="canManage"/)
  assert.match(manager, /const canSelectEntries = computed\(\(\) => canEdit\.value \|\| canManage\.value\)/)
  assert.match(manager, /if \(!canSelectEntries\.value \|\| batchActionLoading\.value\) return/)
})

test('dangerous actions require confirmation and every batch request guards against duplicate submission', () => {
  assert.match(batchBar, /confirmBatchDelete/)
  assert.match(batchBar, /@confirm="emit\('delete'\)"/)
  assert.match(batchBar, /const actionLoading = computed/)
  assert.match(manager, /const batchDeleteLoading = ref\(false\)/)
  assert.match(manager, /const batchTagLoading = ref\(false\)/)
  assert.match(manager, /const batchStatusAction = ref<'enable' \| 'disable' \| null>\(null\)/)
  assert.match(manager, /finally \{\s*batchDeleteLoading\.value = false\s*\}/)
  assert.match(manager, /finally \{\s*batchTagLoading\.value = false\s*\}/)
  assert.match(manager, /finally \{\s*batchStatusAction\.value = null\s*\}/)
})

test('every supported language includes the FAQ batch action copy', () => {
  for (const locale of locales) {
    assert.match(locale, /batchEnable:/)
    assert.match(locale, /batchDisable:/)
    assert.match(locale, /batchDelete:/)
    assert.match(locale, /confirmBatchDelete:/)
    assert.match(locale, /batchDeleteSuccess:/)
  }
})
