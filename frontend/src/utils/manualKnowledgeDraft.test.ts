import assert from 'node:assert/strict'
import test from 'node:test'
import { buildManualDraft, deriveManualTitle } from './manualKnowledgeDraft'

const labels = { emptyAnswer: '(no content)', sourcesHeading: 'Sources' }

test('a chat answer loses its <kb/> tags and gains a source list', () => {
  const answer = [
    '## A binary view of technology',
    '',
    '> **"People fall into a binary view of good versus bad."** <kb doc="Interview.md" chunk_id="230a27cb-45c8-4ba9-9ba1-babea70bcd95" kb_id="441ee604-a04d-4c9a-a4bd-1c4e198e6226" />',
    '',
    'He then takes Zig as an example <kb doc="Interview.md" chunk_id="ef18491c-34f0-4563-bb4a-062cacbdfd95" kb_id="441ee604-a04d-4c9a-a4bd-1c4e198e6226" />',
  ].join('\n')

  const draft = buildManualDraft(answer, labels)

  assert.ok(!draft.includes('<kb'), draft)
  assert.ok(!draft.includes('chunk_id'), draft)
  assert.ok(draft.includes('> **"People fall into a binary view of good versus bad."**'))
  assert.ok(draft.endsWith('**Sources**\n\n- Interview.md\n'), draft)
})

test('web citations survive as real Markdown links', () => {
  const draft = buildManualDraft('See <web url="https://ziglang.org" title="Zig" /> website', labels)
  assert.equal(draft, 'See [Zig](https://ziglang.org) website')
})

test('a tag alone on its line leaves no blank line behind', () => {
  const draft = buildManualDraft('First paragraph\n<kb doc="A.md" chunk_id="1" />\nSecond paragraph', labels)
  assert.equal(draft, 'First paragraph\nSecond paragraph\n\n---\n\n**Sources**\n\n- A.md\n')
})

test('code blocks keep their own spacing while citations are stripped', () => {
  const answer = '```js\nfn( 1 ) ;\n\n\nconst x = 2\n```\n\nNote <kb doc="A.md" chunk_id="1" />'
  const draft = buildManualDraft(answer, labels)
  assert.ok(draft.startsWith('```js\nfn( 1 ) ;\n\n\nconst x = 2\n```'), draft)
  assert.ok(draft.includes('\n\nNote\n'), draft)
})

test('an answer without citations is passed through untouched', () => {
  assert.equal(buildManualDraft('Plain text answer', labels), 'Plain text answer')
  assert.equal(buildManualDraft('   ', labels), '(no content)')
})

test('a long question is cut on a clause boundary, with no trailing ellipsis', () => {
  const title = deriveManualTitle(
    '互联网上的人们常常陷入对技术好坏的二元看法，这种看法如何影响了技术的多样性和创新？',
    'Chat excerpt',
  )
  assert.equal(title, '互联网上的人们常常陷入对技术好坏的二元看法')
})

test('a short question keeps its wording, minus the question mark', () => {
  assert.equal(deriveManualTitle('Zig 为什么值得关注？', 'Chat excerpt'), 'Zig 为什么值得关注')
  assert.equal(deriveManualTitle('  ', 'Chat excerpt'), 'Chat excerpt')
})

test('a long question with no boundary is cut at the limit', () => {
  const title = deriveManualTitle('长'.repeat(60), 'Chat excerpt')
  assert.equal(title, '长'.repeat(32))
})

test('an English question is cut on a word boundary', () => {
  const title = deriveManualTitle('How does a binary view of technology hurt diversity?', 'Chat excerpt')
  assert.equal(title, 'How does a binary view of')
})
