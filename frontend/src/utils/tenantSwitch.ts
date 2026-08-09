// Tenant-switch navigation helper.
//
// Switching the active tenant always lands the user on the platform's KB
// list. Previously it was "reload on the current path" + fallback to the KB list for certain sensitive paths, but even
// pages without a resource id (settings, agent list, etc.) often end up in an empty state after reload because there's no
// corresponding data in the new space—so the experience is basically the same as jumping to a fixed home page anyway. Just unify by jumping to the KB list,
// resetting all stores/SSE/requests at once via a single full navigation.

import { updateMyPreferences } from '@/api/auth'

const SAFE_FALLBACK_PATH = '/platform/knowledge-bases'

/**
 * Return the URL to navigate to after a tenant switch. Currently always returns the KB list
 * as the login page; the function signature is kept in case future routing-specific handling is needed.
 */
export function tenantSwitchTargetPath(_currentPath: string): string {
  return SAFE_FALLBACK_PATH
}

/**
 * Perform the post-switch navigation. Unified to jump to the KB list.
 */
export function navigateAfterTenantSwitch(): void {
  window.location.href = tenantSwitchTargetPath(window.location.pathname)
}

// Toast after a successful switch is carried across the hard reload: the caller stuffs the info in before reloading
// sessionStorage, consumed once when App.vue starts, then popped. Called directly before reload
// NotifyPlugin gets dismissed too fast to read.
const PENDING_TOAST_KEY = 'weknora_pending_tenant_switch_toast'

export interface PendingTenantSwitchToast {
  name: string
  /** Human-readable role label, e.g. "所有者" / "Owner". Drives the role chip text. */
  role?: string
  /** Raw role enum, e.g. "owner". Drives the role chip color + icon. */
  roleEnum?: string
}

export function stashTenantSwitchToast(payload: PendingTenantSwitchToast): void {
  try {
    sessionStorage.setItem(PENDING_TOAST_KEY, JSON.stringify(payload))
  } catch {
    // If writing to sessionStorage fails (private mode, etc.), silently give up — the toast is just a nice-to-have
  }
}

export function consumePendingTenantSwitchToast(): PendingTenantSwitchToast | null {
  try {
    const raw = sessionStorage.getItem(PENDING_TOAST_KEY)
    if (!raw) return null
    sessionStorage.removeItem(PENDING_TOAST_KEY)
    const parsed = JSON.parse(raw)
    if (parsed && typeof parsed.name === 'string') {
      return {
        name: parsed.name,
        role: typeof parsed.role === 'string' ? parsed.role : undefined,
        roleEnum: typeof parsed.roleEnum === 'string' ? parsed.roleEnum : undefined,
      }
    }
    return null
  } catch {
    return null
  }
}

/**
 * Persist the user's "last active tenant" preference server-side so a
 * fresh login (new device, new refresh token, cleared browser) drops
 * them back into this workspace instead of always bouncing to their
 * home tenant.
 *
 * Conventions:
 *   - Pass the target tenant id when switching to a peer tenant.
 *   - Pass `null` (which sends `0`) when switching back to the home
 *     tenant — that clears the preference and reverts the user to the
 *     "always start at home" default.
 *
 * Fire-and-forget: callers usually trigger a full-page reload right
 * after; this returns the in-flight promise so callers can race it
 * against a small budget if they want best-effort completion before
 * navigation. Failures are logged but never thrown — losing one
 * persist is recoverable on the next switch.
 */
export function persistLastActiveTenantPreference(
  tenantId: number | null,
): Promise<void> {
  const payload = { last_active_tenant_id: tenantId == null ? 0 : tenantId }
  return updateMyPreferences(payload)
    .then((res) => {
      if (!res.success) {
        console.warn('persistLastActiveTenantPreference: server rejected update', res.message)
      }
    })
    .catch((err) => {
      console.warn('persistLastActiveTenantPreference: network/serialization error', err)
    })
}
