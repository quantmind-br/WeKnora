/**
  * Wire references in the answer body to "files generated in the sandbox" into the artifact download path.
 *
  * The model references files it generated in the sandbox with Markdown image syntax (the prompt requires
  * `![caption](sandbox:file-name)`), and before persisting, the server rewrites that into the file's stable handle
  * `resource://<handle>`, the same reference form used by knowledge base images and chat attachments. Both forms
  * point at the same `Message.Artifacts`:
 *
  *   - `resource://<handle>` — the authoritative form after persistence; this is what history sessions read;
  *   - `sandbox:<file-name>` — the model's raw form, used while this round's streamed output is not yet rewritten.
 *
  * Because the handle form is identical to knowledge base images, one answer routinely contains both "retrieved knowledge base images"
  * and "skill-generated images". So when a handle does not match this message's artifact list we must return null
  * and hand it back to the default protected image rendering, rather than showing "File unavailable".
 *
  * Image artifacts are shown inline (fetched with auth, then swapped for a blob); other types (HTML charts, CSV,
  * documents, etc.) render as a card that, when clicked, opens the artifact preview in the sandbox side panel. Stuffing
  * a 1MB self-contained HTML iframe into the answer body is both slow and unsafe.
 *
  * Artifacts the user has deleted are tombstones in the list (with deleted_at) and render as a greyed-out, non-clickable
  * card. Callers must **not** filter tombstones out before passing the list in: once filtered, handles no longer match this answer's artifacts,
  * so they get treated as knowledge base images, go through protected image rendering, and end up as a broken image that never loads.
 */

import { escapeHTML } from './security.ts';
import { renderArtifactFileIcon } from './artifactFileIcon';

/** Minimal field set aligned with the backend artifactListItem / SSE publicArtifactViews. */
export interface ArtifactRefMeta {
  index: number;
  file_name: string;
  file_type?: string;
  /** `resource://<handle>`. Empty when the backend has no resource directory enabled; then only the file name can resolve it. */
  handle?: string;
  /**
    * History messages carry `Message.Artifacts` directly, where the storage reference is called `url`. It means the same
    * as handle; either one will do.
   */
  url?: string;
  /**
    * When the user deleted the file. Tombstone entries **must** stay in the list passed to the renderer: first, the index is
    * the download address, so removing one shifts every later file; second, only a matching handle tells us it belongs to this
    * answer, and a mismatch gets treated as a knowledge base image through protected image rendering, ending up as a broken image.
   */
  deleted_at?: string | null;
}

export interface ArtifactRefContext {
  sessionId: string;
  messageId: string;
}

export interface ArtifactRefLabels {
  /** Card subtitle, e.g. "Click to preview". */
  previewHint: string;
  /** Subtitle when the round has ended but the reference matches no artifact, e.g. "File unavailable". */
  missingHint: string;
  /** Subtitle when the user has deleted the file, e.g. "File deleted". */
  deletedHint: string;
}

const RESOURCE_HANDLE_RE = /^resource:\/\/([A-Za-z0-9_-]{22})$/;
const SANDBOX_NAME_RE = /^sandbox:(?:\/\/)?(.+)$/i;
const IMAGE_EXTENSIONS = new Set(['png', 'jpg', 'jpeg', 'gif', 'webp', 'bmp', 'avif']);
const TRANSPARENT_PIXEL =
  'data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///ywAAAAAAQABAAACAUwAOw==';

type ArtifactRef = { kind: 'handle'; handle: string } | { kind: 'name'; name: string };

function parseArtifactRef(href: string): ArtifactRef | null {
  const trimmed = (href || '').trim();
  if (!trimmed) return null;

  const handleMatch = trimmed.match(RESOURCE_HANDLE_RE);
  if (handleMatch) {
    return { kind: 'handle', handle: handleMatch[1] };
  }

  const nameMatch = trimmed.match(SANDBOX_NAME_RE);
  if (!nameMatch) return null;
  let name = nameMatch[1].trim();
  try {
    name = decodeURIComponent(name);
  } catch {
    // Keep it as-is: decodeURIComponent throws when the file name contains a bare %.
  }
  // The directory prefix carries no information; artifacts are indexed by file name.
  name = name.split('/').pop() || '';
  return name ? { kind: 'name', name } : null;
}

/**
  * Whether this link target may be a sandbox artifact reference.
 *
  * The handle form looks the same as knowledge base images, so true here only means "worth trying artifact resolution",
  * not that this message really has the file.
 */
export function isArtifactRefHref(href: string): boolean {
  return parseArtifactRef(href) !== null;
}

function artifactHandle(artifact: ArtifactRefMeta): string {
  const raw = (artifact.handle || artifact.url || '').trim();
  return raw.match(RESOURCE_HANDLE_RE)?.[1] || '';
}

const CODE_SPAN_OR_FENCE_RE = /(```[\s\S]*?```|~~~[\s\S]*?~~~|`[^`\n]*`)/g;

/**
  * Scan a Markdown link target and return the raw text inside the parentheses plus the closing parenthesis position.
 *
  * Parenthesis matching is used instead of a regex because skill-generated file names often contain parentheses and spaces
  * (`Tencent(00700) volume_838ccc.html`). marked truncates the target at the first space,
  * so the reference never matches an artifact; scanning by depth finds exactly the parenthesis that really closes the link.
 */
function scanLinkDestination(text: string, openIndex: number): { inner: string; end: number } | null {
  let depth = 1;
  for (let i = openIndex + 1; i < text.length; i += 1) {
    const ch = text[i];
    if (ch === '\n') return null;
    if (ch === '(') depth += 1;
    else if (ch === ')') {
      depth -= 1;
      if (depth === 0) return { inner: text.slice(openIndex + 1, i), end: i };
    }
  }
  return null;
}

/** Split the optional title part (`dest "title"`) off the target. */
function splitDestinationTitle(inner: string): { destination: string; title: string } {
  const match = inner.match(/^([\s\S]*?)(\s+(?:"[^"]*"|'[^']*'))$/);
  if (!match) return { destination: inner.trim(), title: '' };
  return { destination: match[1].trim(), title: match[2] };
}

/** What precedes `](` must be a `[...]` closed on the same line, otherwise this is not a link. */
function hasLinkLabelBefore(text: string, closeBracketIndex: number): boolean {
  for (let i = closeBracketIndex - 1; i >= 0; i -= 1) {
    const ch = text[i];
    if (ch === '\n') return false;
    if (ch === '[') return true;
  }
  return false;
}

function normalizeSegment(segment: string): string {
  if (!segment.includes('](')) return segment;

  let out = '';
  let cursor = 0;
  while (cursor < segment.length) {
    const relative = segment.slice(cursor).indexOf('](');
    if (relative < 0) break;
    const closeBracket = cursor + relative;
    const open = closeBracket + 1;

    if (!hasLinkLabelBefore(segment, closeBracket)) {
      out += segment.slice(cursor, open + 1);
      cursor = open + 1;
      continue;
    }
    const scanned = scanLinkDestination(segment, open);
    if (!scanned) {
      out += segment.slice(cursor, open + 1);
      cursor = open + 1;
      continue;
    }

    const { destination, title } = splitDestinationTitle(scanned.inner);
    const ref = parseArtifactRef(destination);
    if (!ref || ref.kind !== 'name') {
      out += segment.slice(cursor, scanned.end + 1);
      cursor = scanned.end + 1;
      continue;
    }

    // Percent-encode so marked sees a single whitespace-free token;
    // parseArtifactRef decodes it again on the way out.
    out += `${segment.slice(cursor, open + 1)}sandbox:${encodeURIComponent(ref.name)}${title})`;
    cursor = scanned.end + 1;
  }
  return out + segment.slice(cursor);
}

/**
  * Before marked parses, normalize the targets of `sandbox:` references into a single token without spaces.
 *
  * Without this step, marked splits a file name with spaces at the space, leaving only the first half as the target
  * and leaking the second half into the body text: exactly the "card name truncated + tail left as bare text" bug.
  * Examples inside code blocks are left as-is.
 */
export function normalizeSandboxArtifactRefs(markdown: string): string {
  if (!markdown || !markdown.includes('](')) return markdown;
  if (!/\]\(\s*sandbox:/i.test(markdown)) return markdown;

  const parts = markdown.split(CODE_SPAN_OR_FENCE_RE);
  for (let i = 0; i < parts.length; i += 2) {
    parts[i] = normalizeSegment(parts[i]);
  }
  return parts.join('');
}

/** Resolve a reference to a concrete artifact; returns null when it cannot be resolved (not collected yet / file name mismatch). */
export function resolveArtifactRef(
  href: string,
  artifacts: ArtifactRefMeta[] | undefined | null,
): ArtifactRefMeta | null {
  const ref = parseArtifactRef(href);
  if (!ref || !artifacts?.length) return null;

  if (ref.kind === 'handle') {
    return artifacts.find((item) => artifactHandle(item) === ref.handle) || null;
  }
  return artifacts.find((item) => (item.file_name || '').trim() === ref.name) || null;
}

function fileExtension(fileName: string): string {
  const base = (fileName || '').trim().toLowerCase();
  const dot = base.lastIndexOf('.');
  return dot > 0 ? base.slice(dot + 1) : '';
}

/**
  * Whether to render inline as an image. SVG is deliberately excluded: it is executable content, so it takes the card + sandboxed preview path.
 */
function rendersAsImage(artifact: ArtifactRefMeta): boolean {
  const ext = fileExtension(artifact.file_name);
  if (ext) return IMAGE_EXTENSIONS.has(ext);
  const type = (artifact.file_type || '').toLowerCase();
  return type.startsWith('image/') && !type.includes('svg');
}

// Blob URLs are cached per (session, message, index) and hung on window: Vite HMR replaces modules
// but does not rebuild the document, and a module-level Map would make already-loaded images fall back to the placeholder.
type ArtifactBlobState = { blobByKey: Map<string, string>; inflight: Map<string, Promise<string | null>> };

const artifactBlobState: ArtifactBlobState = (() => {
  const fresh = (): ArtifactBlobState => ({ blobByKey: new Map(), inflight: new Map() });
  if (typeof window === 'undefined') return fresh();
  const scope = window as typeof window & { __weknoraArtifactBlobCacheV1__?: ArtifactBlobState };
  scope.__weknoraArtifactBlobCacheV1__ ||= fresh();
  return scope.__weknoraArtifactBlobCacheV1__;
})();

function blobCacheKey(ctx: ArtifactRefContext, index: number): string {
  return `${ctx.sessionId}\u0000${ctx.messageId}\u0000${index}`;
}

// Same class name as chatMarkdownRenderer's streaming image skeleton, so the styles are shared.
const STREAMING_PLACEHOLDER =
  '<span class="streaming-image-loading"><span class="streaming-image-loading__skeleton"></span></span>';

/**
  * The card's three states:
  *   - `ready`   — normal, clickable to open the preview;
  *   - `pending` — the round has ended but the reference matches no artifact (the model referenced a file that does not exist);
  *   - `deleted` — the file really was generated, but the user deleted it and its bytes were reclaimed.
  * The last two are both non-clickable, but must be told apart: one "never existed", the other "you deleted it yourself".
 */
type ArtifactCardVariant = 'ready' | 'pending' | 'deleted';

function renderCard(
  fileName: string,
  hint: string,
  index: number | null,
  variant: ArtifactCardVariant,
): string {
  const safeName = escapeHTML(fileName);
  const safeHint = escapeHTML(hint);
  // The card must be an inline element: marked wraps images in <p>, and a block element would be hoisted
  // out of the paragraph by the HTML parser, breaking the body structure.
  const interactive = variant === 'ready' && index !== null
    ? ` data-artifact-index="${index}" role="button" tabindex="0"`
    : '';
  const state = variant === 'ready' ? '' : ` artifact-ref-card--${variant}`;
  return (
    `<span class="artifact-ref-card${state}"${interactive} title="${safeName}">`
    + `<span class="artifact-ref-card__icon" aria-hidden="true">${renderArtifactFileIcon(fileName)}</span>`
    + '<span class="artifact-ref-card__text">'
    + `<span class="artifact-ref-card__name">${safeName}</span>`
    + `<span class="artifact-ref-card__hint">${safeHint}</span>`
    + '</span></span>'
  );
}

function renderImage(
  artifact: ArtifactRefMeta,
  alt: string,
  ctx: ArtifactRefContext | null,
): string {
  const safeAlt = escapeHTML(alt || artifact.file_name || '');
  // If it was already fetched, use the blob directly: streaming re-renders rebuild <img>, otherwise every frame would flash back to the placeholder.
  const cached = ctx ? artifactBlobState.blobByKey.get(blobCacheKey(ctx, artifact.index)) : undefined;
  const src = cached || TRANSPARENT_PIXEL;
  const loading = cached ? '' : ' data-img-loading="1"';
  return (
    `<img class="markdown-image artifact-ref-image" src="${src}" alt="${safeAlt}"`
    + ` data-artifact-index="${artifact.index}"${loading}>`
  );
}

/**
  * Render a Markdown image/link target.
 *
  * Returns null when this is not a sandbox artifact reference; the caller should fall back to default rendering (plain images,
  * `resource://` protected images, external links, etc. are all unaffected).
  * Returns an empty string when the target is empty; the caller should not draw a broken image.
 */
export function renderArtifactReference(args: {
  href: string;
  alt?: string;
  artifacts?: ArtifactRefMeta[] | null;
  labels: ArtifactRefLabels;
  context?: ArtifactRefContext | null;
  /** This round's answer is still being generated. Artifacts are only collected when the round ends, so failing to resolve now is normal. */
  streaming?: boolean;
}): string | null {
  const href = (args.href || '').trim();
  if (!href) return '';
  const ref = parseArtifactRef(href);
  if (!ref) return null;

  const artifact = resolveArtifactRef(href, args.artifacts);
  if (!artifact) {
    // The handle does not match this message's artifacts, so it is some other protected file (a retrieved knowledge base image,
    // an attachment image...). Hand it back to default rendering; hydrateProtectedFileImages fetches it with auth.
    if (ref.kind === 'handle') return null;
    // The artifact list only arrives with the complete event at the end of the round, so it can never resolve while streaming.
    // Show a skeleton instead of a card here: it avoids flashing a half file name, and does not leave a "generating"
    // state behind in an answer that has actually finished.
    if (args.streaming) return STREAMING_PLACEHOLDER;
    // Still no match after the round ended means the model referenced a file that does not exist. Say so plainly instead of
    // continuing to show "generating".
    const fallbackName = ref.name || (args.alt || '').trim();
    if (!fallbackName) return '';
    return renderCard(fallbackName, args.labels.missingHint, null, 'pending');
  }

  // Bytes of files the user deleted have been reclaimed, so the download would 404. Image artifacts degrade to a card too: going through
  // renderImage would only fail to fetch and leave a broken image that never loads.
  if (artifact.deleted_at) {
    const name = artifact.file_name || (args.alt || '').trim();
    if (!name) return '';
    return renderCard(name, args.labels.deletedHint, null, 'deleted');
  }

  if (rendersAsImage(artifact)) {
    return renderImage(artifact, args.alt || '', args.context ?? null);
  }
  return renderCard(artifact.file_name, args.labels.previewHint, artifact.index, 'ready');
}

async function loadArtifactBlobURL(ctx: ArtifactRefContext, index: number): Promise<string | null> {
  const key = blobCacheKey(ctx, index);
  const cached = artifactBlobState.blobByKey.get(key);
  if (cached) return cached;

  let task = artifactBlobState.inflight.get(key);
  if (!task) {
    task = (async () => {
      try {
        // Imported lazily so parsing/rendering stays free of the axios
        // transport — those parts run in plain Node during unit tests.
        const { downloadArtifact } = await import('@/api/chat');
        const blob = await downloadArtifact(ctx.sessionId, ctx.messageId, index);
        const blobURL = URL.createObjectURL(blob);
        artifactBlobState.blobByKey.set(key, blobURL);
        return blobURL;
      } catch (error) {
        console.warn('[sandboxArtifactRefs] artifact image load failed:', error);
        return null;
      } finally {
        artifactBlobState.inflight.delete(key);
      }
    })();
    artifactBlobState.inflight.set(key, task);
  }
  return task;
}

/**
  * Replace artifact images in the body with blobs fetched with auth.
 *
  * Idempotent like hydrateProtectedFileImages: elements already swapped carry the authHydrated marker,
  * and concurrent requests for the same file share one Promise.
 */
export async function hydrateArtifactImages(
  root: ParentNode | null | undefined,
  ctx: ArtifactRefContext | null | undefined,
): Promise<void> {
  if (!root || !ctx?.sessionId || !ctx?.messageId) return;

  const images = Array.from(
    root.querySelectorAll<HTMLImageElement>('img.artifact-ref-image[data-artifact-index]'),
  ).filter((img) => img.dataset.authHydrated !== '1');
  if (!images.length) return;

  await Promise.all(images.map(async (img) => {
    const index = Number(img.getAttribute('data-artifact-index'));
    if (!Number.isInteger(index) || index < 0) return;
    img.dataset.authHydrated = '1';

    const blobURL = await loadArtifactBlobURL(ctx, index);
    if (!blobURL) {
      img.dataset.authHydrated = '0';
      return;
    }
    img.src = blobURL;
    img.removeAttribute('data-img-loading');
  }));
}

/**
  * Get the index of the artifact card activated by a click/keyboard event; returns null when the event is unrelated to a card.
 */
export function artifactIndexFromEventTarget(target: EventTarget | null): number | null {
  if (!(target instanceof Element)) return null;
  const card = target.closest('.artifact-ref-card[data-artifact-index]');
  if (!card) return null;
  const index = Number(card.getAttribute('data-artifact-index'));
  return Number.isInteger(index) && index >= 0 ? index : null;
}
