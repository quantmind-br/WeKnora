import assert from 'node:assert/strict'
import test from 'node:test'

import { normalizeAPIKeyKnowledgeBaseIDs } from './apiKeyScope.ts'

/**
 * Verifies that a fully authorized key's null/undefined scope becomes an empty array, so list rendering does not blank the page when it reads length.
 * Passes the empty values the server may return and expects an empty array meaning "all knowledge bases".
 */
test('normalizes missing API key knowledge base scope to an empty array', () => {
  assert.deepEqual(normalizeAPIKeyKnowledgeBaseIDs(null), [])
  assert.deepEqual(normalizeAPIKeyKnowledgeBaseIDs(undefined), [])
})

/**
 * Verifies that a scoped key's knowledge base IDs are fully copied and the result is not the original array, so the edit form cannot pollute the list data.
 * Passes two knowledge base IDs and expects a new array in the original order.
 */
test('copies configured API key knowledge base scope', () => {
  const ids = ['kb-1', 'kb-2']
  const normalized = normalizeAPIKeyKnowledgeBaseIDs(ids)

  assert.deepEqual(normalized, ids)
  assert.notEqual(normalized, ids)
  assert.deepEqual(normalizeAPIKeyKnowledgeBaseIDs([]), [])
})
