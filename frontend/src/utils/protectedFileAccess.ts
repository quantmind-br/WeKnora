/**
 * Access context for protected files (provider:// / resource:// etc.).
 *
 * The backend splits file proxying by access subject; auth models differ across them:
 * - `/files`                                → login session Bearer + X-Tenant-ID
 * - `/api/v1/knowledge-bases/:id/files`     → knowledge base access permission (cross-tenant shared library)
 * - `/api/v1/sessions/:id/messages/:mid/files` → session message ownership + shared agent permission
 * - `/api/v1/embed/:channel_id/files`       → embedded guest's Embed token
 *
 * Which proxy to pick depends on the current request's auth plane, not on which component is rendering the image.
 * This converges that decision into a single source of truth, so rendering components only need to declare scope instead of each building their own URL.
 */

export const PROVIDER_SCHEME_PATTERN = 'resource|local|minio|cos|tos|s3|oss|ks3|obs';

const PROVIDER_FILE_SCHEME_RE = new RegExp(`^(${PROVIDER_SCHEME_PATTERN}):\\/\\/\\S+$`, 'i');
const STORAGE_BACKEND_FILE_SCHEME_RE = new RegExp(
  `^storage:\\/\\/[0-9A-Za-z_-]+\\/(${PROVIDER_SCHEME_PATTERN}):\\/\\/\\S+$`,
  'i',
);

const KB_FILE_PROXY_PATH_RE = /^\/api\/v1\/knowledge-bases\/[^/]+\/files$/;
const EMBED_FILE_PROXY_PATH_RE = /^\/api\/v1\/embed\/[^/]+\/files$/;
const MESSAGE_FILE_PROXY_PATH_RE = /^\/api\/v1\/sessions\/[^/]+\/messages\/[^/]+\/files$/;

export type ProtectedFileAccessContext =
  /** Logged-in user: Bearer + selected tenant. */
  | { mode: 'tenant' }
  /** Embedded guest: only holds an Embed token, no Bearer, no tenant context. */
  | { mode: 'embed'; channelId: string; token: string }
  /** Knowledge base scope: logged-in user reads objects owned by other tenants within a shared library. */
  | { mode: 'knowledgeBase'; kbId: string }
  /** Message scope: logged-in user reads source-space resources within a shared agent's reply. */
  | { mode: 'message'; sessionId: string; messageId: string };

export interface ProtectedFileRequest {
  url: string;
  headers: Record<string, string>;
}

const TENANT_ACCESS: ProtectedFileAccessContext = { mode: 'tenant' };

interface ProtectedFileAccessState {
  current: ProtectedFileAccessContext;
}

// Also hung on window like the blob cache: Vite HMR replaces the module but doesn't rebuild the document,
// module-level variables lose the context registered when the embedded app started up.
const accessState: ProtectedFileAccessState = (() => {
  const fresh = (): ProtectedFileAccessState => ({ current: TENANT_ACCESS });
  if (typeof window === 'undefined') return fresh();
  const scope = window as typeof window & {
    __weknoraProtectedFileAccessV1__?: ProtectedFileAccessState;
  };
  scope.__weknoraProtectedFileAccessV1__ ||= fresh();
  return scope.__weknoraProtectedFileAccessV1__;
})();

/**
 * Registers the default access context for the current document. Called once by the app entry point (the embedded app registers it after obtaining
 * channelId/token), after which all protected file requests automatically go through the corresponding proxy.
 */
export function setDefaultProtectedFileAccess(
  access: ProtectedFileAccessContext | null,
): void {
  accessState.current = access ?? TENANT_ACCESS;
}

export function getDefaultProtectedFileAccess(): ProtectedFileAccessContext {
  return accessState.current;
}

/**
 * Merges the default context with the scope passed in by the component.
 *
 * The default context carries the auth plane (embedded guest vs. logged-in user); a component-level override can only
 * refine the scope within the same plane. An embedded guest has no Bearer, so letting `knowledgeBase` override
 * override the embed plane would send the request to a proxy that requires login state and return 401.
 */
export function resolveProtectedFileAccess(
  override?: ProtectedFileAccessContext | null,
): ProtectedFileAccessContext {
  const fallback = accessState.current;
  if (fallback.mode === 'embed') return fallback;
  if (!override) return fallback;
  if (override.mode === 'knowledgeBase' && !override.kbId.trim()) return fallback;
  if (override.mode === 'message' && (!override.sessionId.trim() || !override.messageId.trim())) {
    return fallback;
  }
  return override;
}

/** Whether it's a storage path that needs to be fetched via proxy (provider:// or storage://<backend>/provider://). */
export function isProviderFileURL(url: string): boolean {
  const trimmed = url.trim();
  return PROVIDER_FILE_SCHEME_RE.test(trimmed) || STORAGE_BACKEND_FILE_SCHEME_RE.test(trimmed);
}

/** Whether it's a path for one of the protected file proxies. */
export function isProtectedFileProxyPath(pathname: string): boolean {
  return (
    pathname === '/files'
    || KB_FILE_PROXY_PATH_RE.test(pathname)
    || MESSAGE_FILE_PROXY_PATH_RE.test(pathname)
    || EMBED_FILE_PROXY_PATH_RE.test(pathname)
  );
}

function tenantRequestHeaders(): Record<string, string> {
  const headers: Record<string, string> = {};
  try {
    const token = (localStorage.getItem('weknora_token') || '').trim();
    if (token) {
      headers['Authorization'] = `Bearer ${token}`;
    }

    const selectedTenantId = (localStorage.getItem('weknora_selected_tenant_id') || '').trim();
    if (selectedTenantId) {
      // Always attach when a selected tenant is set. Same rationale as
      // utils/request.ts / api/chat/streame.ts: the
      // "selectedTenantId === defaultTenantId → skip" short-circuit
      // silently drops the header whenever any code path writes the
      // active tenant into weknora_tenant, leaving authenticated file
      // fetches landing on the home tenant.
      headers['X-Tenant-ID'] = selectedTenantId;
    }
  } catch {
    // ignore localStorage read errors
  }
  return headers;
}

/**
 * Builds a proxy request for a storage path. Returning null means the current context can't issue this request
 * (not a storage path, or the embed context hasn't gotten a token yet); the caller should skip and retry later.
 */
export function buildProtectedFileRequest(
  sourceURL: string,
  access: ProtectedFileAccessContext,
): ProtectedFileRequest | null {
  const filePath = sourceURL.trim();
  if (!isProviderFileURL(filePath)) return null;

  const query = new URLSearchParams({ file_path: filePath }).toString();

  if (access.mode === 'embed') {
    const channelId = access.channelId.trim();
    const token = access.token.trim();
    // An embedded guest has only one credential, the Embed token; if it's missing, falling back to /files just gets a 401,
    // better to skip and wait for the next hydration after bootstrap completes.
    if (!channelId || !token) return null;
    return {
      url: `/api/v1/embed/${encodeURIComponent(channelId)}/files?${query}`,
      headers: { Authorization: `Embed ${token}` },
    };
  }

  if (access.mode === 'knowledgeBase') {
    return {
      url: `/api/v1/knowledge-bases/${encodeURIComponent(access.kbId.trim())}/files?${query}`,
      headers: tenantRequestHeaders(),
    };
  }

  if (access.mode === 'message') {
    return {
      url: `/api/v1/sessions/${encodeURIComponent(access.sessionId.trim())}/messages/${encodeURIComponent(access.messageId.trim())}/files?${query}`,
      headers: tenantRequestHeaders(),
    };
  }

  return { url: `/files?${query}`, headers: tenantRequestHeaders() };
}
