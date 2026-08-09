/**
 * Security utility class - prevents XSS attacks
 */

import DOMPurify from 'dompurify';
import type { Config, NodeHook } from 'dompurify';
import {
  domPurifySecurityHooks,
  domPurifySecurityOptions,
  markdownDomPurifyConfig,
  markdownDomPurifySecurityHooks,
} from './markdownDomPurify.ts';
import {
  buildProtectedFileRequest,
  isProtectedFileProxyPath,
  isProviderFileURL,
  PROVIDER_SCHEME_PATTERN,
  resolveProtectedFileAccess,
  type ProtectedFileAccessContext,
} from './protectedFileAccess.ts';

const PROVIDER_IMAGE_PLACEHOLDER = 'data:image/gif;base64,R0lGODlhAQABAIAAAAAAAP///ywAAAAAAQABAAACAUwAOw==';
const PROVIDER_IMG_SRC_RE = new RegExp(
  `<img\\b([^>]*?)\\ssrc=(["'])(${PROVIDER_SCHEME_PATTERN}):(?:\\/\\/|&#x2f;&#x2f;|&#47;&#47;)([^"']+)\\2([^>]*)>`,
  'gi',
);
const STORAGE_BACKEND_IMG_SRC_RE = new RegExp(
  `<img\\b([^>]*?)\\ssrc=(["'])storage:\\/\\/([0-9A-Za-z_-]+)\\/(${PROVIDER_SCHEME_PATTERN}):(?:\\/\\/|&#x2f;&#x2f;|&#47;&#47;)([^"']+)\\2([^>]*)>`,
  'gi',
);

type SecurityHooks = {
  beforeSanitizeElements: NodeHook;
  afterSanitizeElements: NodeHook;
};

function sanitizeWithSecurityHooks(
  html: string,
  config: Config,
  hooks: SecurityHooks,
): string {
  DOMPurify.addHook('beforeSanitizeElements', hooks.beforeSanitizeElements);
  DOMPurify.addHook('afterSanitizeElements', hooks.afterSanitizeElements);
  try {
    return DOMPurify.sanitize(html, config);
  } finally {
    DOMPurify.removeHook('afterSanitizeElements', hooks.afterSanitizeElements);
    DOMPurify.removeHook('beforeSanitizeElements', hooks.beforeSanitizeElements);
  }
}

// Configure DOMPurify's security policy
const DOMPurifyConfig = {
  // Allowed tags
  ALLOWED_TAGS: [
    'p', 'br', 'strong', 'em', 'u', 's', 'del', 'ins',
    'h1', 'h2', 'h3', 'h4', 'h5', 'h6',
    'ul', 'ol', 'li', 'blockquote', 'pre', 'code',
    'a', 'img', 'table', 'thead', 'tbody', 'tr', 'th', 'td',
    'div', 'span', 'figure', 'figcaption', 'details', 'summary', 'think', 'button',
    // Tags supported by Mermaid SVG
    'svg', 'g', 'path', 'rect', 'circle', 'ellipse', 'line', 'polygon',
    'polyline', 'text', 'tspan', 'defs', 'marker', 'filter', 'use',
    'clippath', 'lineargradient', 'radialgradient', 'stop', 'pattern',
    'image', 'foreignobject', 'desc', 'title', 'switch', 'symbol', 'mask',
    // Tags supported by KaTeX MathML
    'math', 'annotation', 'semantics', 'mo', 'mi', 'mn', 'msup', 'mrow', 'mfrac', 'msqrt', 'mroot', 'mstyle'
  ],
  // Allowed attributes
  ALLOWED_ATTR: [
    'href', 'title', 'alt', 'src', 'class', 'id', 'style', 'data-protected-src', 'data-img-loading',
    'target', 'rel', 'width', 'height', 'open',
    'type', 'aria-label', 'disabled', 'role', 'tabindex',
    // Attributes supported by Mermaid SVG
    'd', 'fill', 'stroke', 'stroke-width', 'stroke-linecap', 'stroke-linejoin',
    'stroke-dasharray', 'stroke-dashoffset', 'stroke-miterlimit', 'stroke-opacity',
    'fill-opacity', 'opacity', 'transform', 'viewbox', 'preserveaspectratio',
    'x', 'y', 'x1', 'y1', 'x2', 'y2', 'cx', 'cy', 'rx', 'ry', 'r',
    'dx', 'dy', 'text-anchor', 'dominant-baseline', 'font-family', 'font-size',
    'font-weight', 'font-style', 'letter-spacing', 'word-spacing',
    'marker-start', 'marker-mid', 'marker-end', 'markerunits', 'markerwidth',
    'markerheight', 'refx', 'refy', 'orient', 'points', 'offset',
    'gradientunits', 'gradienttransform', 'spreadmethod', 'stop-color', 'stop-opacity',
    'patternunits', 'patterntransform', 'clippathunits', 'maskunits',
    'filterunits', 'primitiveunits', 'xmlns', 'xmlns:xlink', 'xlink:href',
    'version', 'baseprofile', 'enable-background', 'overflow', 'visibility',
    'display', 'pointer-events', 'cursor', 'data-emit', 'direction',
    // Attributes supported by KaTeX MathML
    'mathvariant', 'encoding', 'aria-hidden'
  ],
  USE_PROFILES: { html: true, svg: true, mathMl: true },
  ...domPurifySecurityOptions,
};

/**
 * Safely sanitize HTML content
 * @param html The HTML string to sanitize
 * @returns The sanitized, safe HTML string
 */
export function sanitizeHTML(html: string): string {
  if (!html || typeof html !== 'string') {
    return '';
  }
  
  try {
    const preparedHTML = protectProviderImageSrcInHTML(html);
    return sanitizeWithSecurityHooks(
      preparedHTML,
      DOMPurifyConfig as unknown as Config,
      domPurifySecurityHooks,
    );
  } catch (error) {
    console.error('HTML sanitization failed:', error);
    // If sanitization fails, return escaped plain text
    return escapeHTML(html);
  }
}

/** Sanitize assistant markdown HTML (code/mermaid toolbars, KaTeX, SVG). */
export function sanitizeMarkdownHTML(html: string): string {
  if (!html || typeof html !== 'string') {
    return '';
  }

  try {
    const preparedHTML = protectProviderImageSrcInHTML(html);
    return sanitizeWithSecurityHooks(
      preparedHTML,
      markdownDomPurifyConfig as unknown as Config,
      markdownDomPurifySecurityHooks,
    );
  } catch (error) {
    console.error('Markdown HTML sanitization failed:', error);
    return escapeHTML(html);
  }
}

function buildProtectedImageTag(
  before: string,
  quote: string,
  protectedSrc: string,
  after: string,
): string {
  // A definitive 404 should not leave a skeleton behind. Streaming
  // re-renders call this function repeatedly, so remember the missing
  // source until the explicit end-of-stream retry clears the cache.
  if (protectedFileMissingSources.has(protectedSrc)) {
    return '';
  }
  // Reuse the already-hydrated blob if we have one, so repeated re-renders
  // (typewriter streaming) keep the same stable image instead of flashing
  // back to the placeholder every frame.
  const cachedBlobURL = protectedFileBlobBySource.get(protectedSrc);
  if (cachedBlobURL) {
    return `<img${before} src=${quote}${cachedBlobURL}${quote} data-protected-src=${quote}${protectedSrc}${quote}${after}>`;
  }
  // Not hydrated yet: render the 1x1 placeholder but tag it so CSS can give
  // it a stable skeleton box. Otherwise width:auto/height:auto collapse the
  // 1x1 gif to a ~1px line that violently jumps to full size once loaded.
  return `<img${before} src=${quote}${PROVIDER_IMAGE_PLACEHOLDER}${quote} data-protected-src=${quote}${protectedSrc}${quote} data-img-loading=${quote}1${quote}${after}>`;
}

export function protectProviderImageSrcInHTML(html: string): string {
  if (!html) return html;
  const withProviderImages = html.replace(
    PROVIDER_IMG_SRC_RE,
    (_m, before, quote, provider, restPathRaw, after) => {
      const restPath = decodeProviderURL(restPathRaw);
      return buildProtectedImageTag(before, quote, `${provider}://${restPath}`, after);
    },
  );
  return withProviderImages.replace(
    STORAGE_BACKEND_IMG_SRC_RE,
    (_m, before, quote, backendID, provider, restPathRaw, after) => {
      const restPath = decodeProviderURL(restPathRaw);
      return buildProtectedImageTag(
        before,
        quote,
        `storage://${backendID}/${provider}://${restPath}`,
        after,
      );
    },
  );
}

function decodeProviderURL(raw: string): string {
  return raw
    .trim()
    .replace(/&#x2f;/gi, '/')
    .replace(/&#47;/g, '/')
    .replace(/&amp;/g, '&')
    .replace(/&quot;/g, '"');
}

function providerSourceFromImageSrc(src: string): string | null {
  const decodedSrc = decodeProviderURL(src);
  if (isProviderFileURL(decodedSrc)) {
    return decodedSrc;
  }

  try {
    const baseURL = typeof window !== 'undefined' ? window.location.origin : 'http://localhost';
    const url = new URL(decodedSrc, baseURL);
    if (!isProtectedFileProxyPath(url.pathname)) {
      return null;
    }

    const filePath = (url.searchParams.get('file_path') || '').trim();
    return isProviderFileURL(filePath) ? filePath : null;
  } catch {
    return null;
  }
}

function normalizeProtectedImageElement(img: HTMLImageElement): string | null {
  const protectedSrc = providerSourceFromImageSrc(
    img.getAttribute('data-protected-src') || '',
  );
  const src = img.getAttribute('src') || '';
  const sourceURL = protectedSrc || providerSourceFromImageSrc(src);
  if (!sourceURL) {
    return null;
  }

  img.setAttribute('data-protected-src', sourceURL);
  if (!src.trim().startsWith('blob:')) {
    img.setAttribute('src', PROVIDER_IMAGE_PLACEHOLDER);
  }
  return sourceURL;
}

/**
 * Escape special HTML characters
 * @param text The text to escape
 * escaped text
 */
export function escapeHTML(text: string): string {
  if (!text || typeof text !== 'string') {
    return '';
  }
  
  const map: { [key: string]: string } = {
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    '"': '&quot;',
    "'": '&#x27;',
    '/': '&#x2F;',
    '`': '&#x60;',
    '=': '&#x3D;'
  };
  
  return text.replace(/[&<>"'`=\/]/g, (s) => map[s]);
}

/**
 * Validate whether the URL is safe
 * @param url URL to validate
 * @returns whether it is a safe URL
 */
export function isValidURL(url: string): boolean {
  if (!url || typeof url !== 'string') {
    return false;
  }
  const trimmed = url.trim();
  if (!trimmed) {
    return false;
  }

  // Allow relative in-site paths starting with / (e.g. local storage /files/images/xxx.jpg)
  if (trimmed.startsWith('/') && !trimmed.startsWith('//')) {
    return true;
  }

  // Allow provider:// form, to be authenticated and replaced with a blob URL by the frontend later
  if (isProviderFileURL(trimmed)) {
    return true;
  }
  
  try {
    const urlObj = new URL(trimmed);
    return ['http:', 'https:'].includes(urlObj.protocol);
  } catch {
    return false;
  }
}

/**
 * Safely process Markdown content
 * @param markdown Markdown text
 * @returns safe HTML string
 */
export function safeMarkdownToHTML(markdown: string): string {
  if (!markdown || typeof markdown !== 'string') {
    return '';
  }
  
  // First escape possible HTML tags
  const escapedMarkdown = markdown
    .replace(/<script\b[^<]*(?:(?!<\/script>)<[^<]*)*<\/script>/gi, '')
    .replace(/<iframe\b[^<]*(?:(?!<\/iframe>)<[^<]*)*<\/iframe>/gi, '')
    .replace(/<object\b[^<]*(?:(?!<\/object>)<[^<]*)*<\/object>/gi, '')
    .replace(/<embed\b[^<]*(?:(?!<\/embed>)<[^<]*)*<\/embed>/gi, '');
  
  return escapedMarkdown;
}

/**
 * Sanitize user input
 * @param input user input
 * @returns sanitized safe input
 */
export function sanitizeUserInput(input: string): string {
  if (!input || typeof input !== 'string') {
    return '';
  }
  
  // Remove control characters
  let cleaned = input.replace(/[\x00-\x1F\x7F-\x9F]/g, '');
  
  // Limit length
  if (cleaned.length > 10000) {
    cleaned = cleaned.substring(0, 10000);
  }
  
  return cleaned.trim();
}

/**
 * Validate whether the image URL is safe
 * @param url image URL
 * @returns whether it is a safe image URL
 */
export function isValidImageURL(url: string): boolean {
  if (!isValidURL(url)) {
    return false;
  }
  
  return true;
}

/**
 * Create a safe image element
 * @param src image source
 * @param alt alt text
 * @param title title
 * @returns safe image HTML
 */
export function createSafeImage(src: string, alt: string = '', title: string = ''): string {
  if (!isValidImageURL(src)) {
    return '';
  }
  
  // src is validated by isValidImageURL; keep URL structure unchanged.
  // Only escape quotes to avoid breaking attributes.
  const safeSrc = src.replace(/"/g, '&quot;');
  const safeAlt = escapeHTML(alt);
  const safeTitle = escapeHTML(title);
  
  return `<img src="${safeSrc}" alt="${safeAlt}" title="${safeTitle}" class="markdown-image" style="max-width: 100%; height: auto;">`;
}

type ProtectedFileLoadResult =
  | { status: 'loaded'; blobURL: string }
  | { status: 'missing' }
  | { status: 'failed' };

type ProtectedFileCacheState = {
  blobByRequest: Map<string, string>;
  blobBySource: Map<string, string>;
  missingSources: Set<string>;
  failures: Map<string, number>;
  inflight: Map<string, Promise<ProtectedFileLoadResult>>;
};

// Keep object URLs alive across Vite hot updates. A hot update replaces this
// module but not the page document, so module-local Maps would forget valid
// blob URLs and make already-loaded images fall back to the 1x1 placeholder.
const protectedFileCacheState = (() => {
  const fresh = (): ProtectedFileCacheState => ({
    blobByRequest: new Map(),
    blobBySource: new Map(),
    missingSources: new Set(),
    failures: new Map(),
    inflight: new Map(),
  });
  if (typeof window === 'undefined') return fresh();
  const scope = window as typeof window & {
    __weknoraProtectedFileCacheV1__?: ProtectedFileCacheState;
  };
  scope.__weknoraProtectedFileCacheV1__ ||= fresh();
  return scope.__weknoraProtectedFileCacheV1__;
})();

const protectedFileBlobCache = protectedFileCacheState.blobByRequest;
// Blob URL keyed by the protected source URL (e.g. `local://...`). Once an image
// has been hydrated, re-renders of the same markdown can emit the blob src
// directly instead of the placeholder. Without this, the typewriter re-renders
// the answer every frame, recreating each <img> as a placeholder that hydration
// only restores a microtask later — which reads as a per-frame flicker.
const protectedFileBlobBySource = protectedFileCacheState.blobBySource;
const protectedFileMissingSources = protectedFileCacheState.missingSources;
// Throttle retries of failed file fetches. During streaming the same markdown
// is re-rendered on every chunk, producing brand-new <img> elements (so the
// per-element `authHydrated` flag is reset each time). Without throttling a
// not-yet-generated file (404) would be re-requested on every chunk. We record
// the last failure time per URL and skip re-fetching within a cooldown window,
// while still allowing a later attempt once the file becomes available.
const protectedFileFailureCache = protectedFileCacheState.failures;
const protectedFileInflight = protectedFileCacheState.inflight;
const PROTECTED_FILE_RETRY_COOLDOWN_MS = 5000;

/**
 * Replace images proxied through /files in Markdown with ones fetched via an authenticated-header fetch before displaying.
 * Used to avoid exposing the token in the URL.
 */
/**
 * Clear failed-retry cooldown records. Called on scenarios like stream end, so images that got a 404 because the file wasn't generated yet
 * can retry loading immediately, without waiting for the cooldown window to end.
 */
export function clearProtectedFileFailureCache(): void {
  protectedFileFailureCache.clear();
  protectedFileMissingSources.clear();
}

function protectedImageSource(img: HTMLImageElement): string {
  return normalizeProtectedImageElement(img)
    || (img.getAttribute('data-protected-src') || '').trim()
    || (img.getAttribute('src') || '').trim();
}

function forEachProtectedImageWithSource(
  root: ParentNode,
  sourceURL: string,
  callback: (img: HTMLImageElement) => void,
): void {
  root.querySelectorAll<HTMLImageElement>('img[data-protected-src]').forEach((candidate) => {
    if (protectedImageSource(candidate) === sourceURL) callback(candidate);
  });
}

function removeMissingProtectedImages(root: ParentNode, sourceURL: string): void {
  forEachProtectedImageWithSource(root, sourceURL, (img) => {
    const parent = img.parentElement;
    img.remove();
    // Markdown emits a dedicated paragraph for a standalone image. Remove that
    // wrapper too so a missing image leaves no vertical placeholder/gap.
    if (parent?.tagName === 'P' && !parent.textContent?.trim() && parent.children.length === 0) {
      parent.remove();
    }
  });
}

function applyHydratedProtectedImage(root: ParentNode, sourceURL: string, blobURL: string): void {
  forEachProtectedImageWithSource(root, sourceURL, (img) => {
    img.src = blobURL;
    img.dataset.authHydrated = '1';
    img.removeAttribute('data-img-loading');
  });
}

/**
 * Fetch protected images (resource:// etc.) in the content through the corresponding authenticated file proxy,
 * then replace them for display with a blob URL.
 *
 * Which proxy to use is decided by {@link resolveProtectedFileAccess}: the default context registered at the app entry
 * (e.g. the embedded app's Embed plane) takes priority, and components only use
 * `access` to narrow the scope (e.g. knowledge base) within the same auth plane.
 */
export async function hydrateProtectedFileImages(
  root: ParentNode | null | undefined,
  access?: ProtectedFileAccessContext,
): Promise<void> {
  if (!root || typeof window === 'undefined') {
    return;
  }

  const images = root.querySelectorAll<HTMLImageElement>(
    'img[data-protected-src], img[src^="resource://"], img[src^="storage://"], img[src^="local://"], img[src^="minio://"], img[src^="cos://"], img[src^="tos://"], img[src^="s3://"], img[src^="oss://"], img[src^="ks3://"], img[src^="obs://"]',
  );
  if (!images.length) {
    return;
  }

  const resolvedAccess = resolveProtectedFileAccess(access);

  await Promise.all(Array.from(images).map(async (img) => {
    const normalizedSourceURL = normalizeProtectedImageElement(img);
    const protectedSrc = (img.getAttribute('data-protected-src') || '').trim();
    const src = (img.getAttribute('src') || '').trim();
    const sourceURL = normalizedSourceURL || protectedSrc || src;
    if (!sourceURL) {
      return;
    }
    if (img.dataset.authHydrated === '1') {
      return;
    }
    if (protectedFileMissingSources.has(sourceURL)) {
      removeMissingProtectedImages(root, sourceURL);
      return;
    }
    img.dataset.authHydrated = '1';

    // A null request means this source cannot be fetched under the current
    // access context (not a storage path, or the embed token has not arrived
    // yet). Leave the placeholder so a later pass can retry.
    const request = buildProtectedFileRequest(sourceURL, resolvedAccess);
    if (!request) {
      img.dataset.authHydrated = '0';
      return;
    }
    const { url: requestURL, headers } = request;

    const cachedBlobURL = protectedFileBlobCache.get(requestURL);
    if (cachedBlobURL) {
      applyHydratedProtectedImage(root, sourceURL, cachedBlobURL);
      return;
    }

    const lastFailure = protectedFileFailureCache.get(requestURL);
    if (lastFailure !== undefined && Date.now() - lastFailure < PROTECTED_FILE_RETRY_COOLDOWN_MS) {
      img.dataset.authHydrated = '0';
      return;
    }

    // Every component that references the same image awaits the shared task.
    // The previous Set-based de-dupe made later components return immediately;
    // only the component that started the fetch was updated, leaving all other
    // occurrences stuck on the transparent placeholder forever.
    let loadTask = protectedFileInflight.get(requestURL);
    if (!loadTask) {
      loadTask = (async (): Promise<ProtectedFileLoadResult> => {
        try {
          const resp = await fetch(requestURL, {
            method: 'GET',
            headers,
            credentials: 'include',
          });
          if (!resp.ok) {
            if (resp.status === 404) {
              protectedFileFailureCache.set(requestURL, Date.now());
              return { status: 'missing' };
            }
            throw new Error(`HTTP ${resp.status}`);
          }
          const blob = await resp.blob();
          const blobURL = URL.createObjectURL(blob);
          protectedFileBlobCache.set(requestURL, blobURL);
          protectedFileFailureCache.delete(requestURL);
          return { status: 'loaded', blobURL };
        } catch (error) {
          console.warn('[security] hydrateProtectedFileImages failed:', error);
          protectedFileFailureCache.set(requestURL, Date.now());
          return { status: 'failed' };
        } finally {
          protectedFileInflight.delete(requestURL);
        }
      })();
      protectedFileInflight.set(requestURL, loadTask);
    }

    const result = await loadTask;
    if (result.status === 'loaded') {
      protectedFileBlobBySource.set(sourceURL, result.blobURL);
      protectedFileMissingSources.delete(sourceURL);
      applyHydratedProtectedImage(root, sourceURL, result.blobURL);
      return;
    }
    if (result.status === 'missing') {
      protectedFileMissingSources.add(sourceURL);
      removeMissingProtectedImages(root, sourceURL);
      return;
    }
    if (result.status === 'failed') {
      img.dataset.authHydrated = '0';
    }
  }));
}
