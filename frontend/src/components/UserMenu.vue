<template>
  <div class="user-menu" :class="{ 'user-menu--collapsed': uiStore.sidebarCollapsed }" ref="menuRef">
    <!-- User button -->
    <div class="user-button" data-guide="user-menu" @click="toggleMenu">
      <div class="user-avatar">
        <img v-if="userAvatar" :src="userAvatar" :alt="$t('common.avatar')" />
        <span v-else class="avatar-placeholder">{{ userInitial }}</span>
      </div>
      <template v-if="!uiStore.sidebarCollapsed">
        <div class="user-info">
          <!-- Multi-space / superuser: space name on the first line, username · role on the second. Single space: nickname + email. -->
          <template v-if="showTenantIdentityLine">
            <div class="user-tenant-name" :title="activeTenantName">{{ activeTenantName }}</div>
            <div class="user-tenant-meta">
              <span v-if="userName && userName !== activeTenantName" class="user-tenant-meta-name">{{ userName }}</span>
              <span v-if="(userName && userName !== activeTenantName) && currentRoleLabel"
                class="user-tenant-meta-sep">·</span>
              <t-icon v-if="currentRoleIcon" :name="currentRoleIcon" size="12px" class="user-tenant-meta-icon" />
              <span v-if="currentRoleLabel" class="user-tenant-meta-role">{{ currentRoleLabel }}</span>
            </div>
          </template>
          <template v-else>
            <div class="user-name">{{ userName }}</div>
            <div class="user-email">{{ userEmail }}</div>
          </template>
        </div>
        <t-icon :name="menuVisible ? 'chevron-up' : 'chevron-down'" class="dropdown-icon" />
      </template>
    </div>

    <!-- Dropdown menu -->
    <Transition name="dropdown">
      <div v-if="menuVisible" class="user-dropdown" @click.stop>
        <!-- Popover menu: account (avatar + nickname) / current space (name + permissions); bottom sidebar style unchanged. -->
        <div v-if="userName" class="dropdown-user-header is-clickable" role="button" tabindex="0"
          @click="handleQuickNav('userprofile')" @keydown.enter.prevent="handleQuickNav('userprofile')"
          @keydown.space.prevent="handleQuickNav('userprofile')">
          <div class="dropdown-user-avatar">
            <img v-if="userAvatar" :src="userAvatar" :alt="$t('common.avatar')" />
            <span v-else class="dropdown-user-avatar-placeholder">{{ userInitial }}</span>
          </div>
          <div class="dropdown-user-meta">
            <div class="dropdown-user-name-row">
              <span class="dropdown-user-name">{{ userName }}</span>
              <t-tooltip :content="$t('newUserGuide.reopen')" placement="top">
                <button type="button" class="dropdown-guide-btn" :aria-label="$t('newUserGuide.reopen')"
                  @click.stop="reopenGuide">
                  <t-icon name="help-circle" size="14px" />
                </button>
              </t-tooltip>
            </div>
            <span v-if="userEmail" class="dropdown-user-email">{{ userEmail }}</span>
          </div>
        </div>

        <div v-if="userName && !authStore.isLiteMode" ref="tenantMenuItemRef" class="dropdown-tenant-panel" :class="{
          'is-open': tenantSubmenuOpen,
          'is-clickable': showTenantSwitcher,
        }" @mouseenter="showTenantSwitcher && showTenantSubmenu()"
          @mouseleave="showTenantSwitcher && scheduleHideTenantSubmenu()">
          <t-icon name="system-sum" class="menu-icon" aria-hidden="true" />
          <div class="dropdown-tenant-panel-main">
            <span class="dropdown-tenant-panel-name" :title="activeTenantName || userName">
              {{ activeTenantName || userName }}
            </span>
            <div v-if="currentRoleLabel" class="dropdown-tenant-panel-role">
              <t-icon v-if="currentRoleIcon" :name="currentRoleIcon" size="12px"
                class="dropdown-tenant-panel-role-icon" />
              <span>{{ currentRoleLabel }}</span>
            </div>
          </div>
          <t-icon v-if="showTenantSwitcher" name="swap" class="dropdown-tenant-panel-trail"
            :title="$t('tenant.switcher.menuLabel')" />
        </div>
        <div class="menu-divider"></div>
        <!-- Account and space are the core context of the avatar menu; infrastructure-type settings are all consolidated under "All Settings". -->
        <div class="menu-item" @click="handleQuickNav('general')">
          <t-icon name="user" class="menu-icon" />
          <span>{{ $t('general.personalSettings') }}</span>
        </div>
        <div v-if="!authStore.isLiteMode" class="menu-item" @click="handleQuickNav('tenant')">
          <t-icon name="user-circle" class="menu-icon" />
          <span>{{ $t('settings.workspaceSettings') }}</span>
        </div>
        <!-- "Manage"-type shortcuts are only shown to users who actually have write permission. The read-only roster and model list
             are still accessible from "All Settings", avoiding showing viewers a management entry they can't actually use. -->
        <div v-if="canManageMembers" class="menu-item" @click="handleQuickNav('members')">
          <t-icon name="usergroup" class="menu-icon" />
          <span>{{ $t('tenantMember.title') }}</span>
        </div>
        <div v-if="canManageModels" class="menu-item" @click="handleQuickNav('models')">
          <t-icon name="control-platform" class="menu-icon" />
          <span>{{ $t('settings.modelManagement') }}</span>
        </div>
        <div class="menu-divider"></div>
        <div class="menu-item" @click="handleSettings">
          <t-icon name="setting" class="menu-icon" />
          <span>{{ $t('general.allSettings') }}</span>
        </div>
        <!--
          System administration entry — visible only to users with the
          platform-wide is_system_admin flag. Hidden for everyone else,
          including tenant Owners. Real authorisation lives server-side
          (RequireSystemAdmin middleware); this is UI gating only.
        -->
        <div v-if="authStore.isSystemAdmin" class="menu-item" @click="handleSystemAdmin">
          <t-icon name="server" class="menu-icon" />
          <span>{{ $t('settings.navGroups.systemAdministration') }}</span>
        </div>
        <div class="menu-divider"></div>
        <div class="menu-item" @click="openDocs">
          <t-icon name="help-circle" class="menu-icon" />
          <span class="menu-text-with-icon">
            <span>{{ $t('general.helpAndDocs') }}</span>
            <svg class="menu-external-icon" viewBox="0 0 16 16" aria-hidden="true">
              <path fill="currentColor"
                d="M12.667 8a.667.667 0 0 1 .666.667v4a2.667 2.667 0 0 1-2.666 2.666H4.667a2.667 2.667 0 0 1-2.667-2.666V5.333a2.667 2.667 0 0 1 2.667-2.666h4a.667.667 0 1 1 0 1.333h-4a1.333 1.333 0 0 0-1.333 1.333v7.334A1.333 1.333 0 0 0 4.667 13.333h6a1.333 1.333 0 0 0 1.333-1.333v-4A.667.667 0 0 1 12.667 8Zm2.666-6.667v4a.667.667 0 0 1-1.333 0V3.276l-5.195 5.195a.667.667 0 0 1-.943-.943l5.195-5.195h-2.057a.667.667 0 0 1 0-1.333h4a.667.667 0 0 1 .666.666Z" />
            </svg>
          </span>
        </div>
        <div class="menu-item" :title="$t('common.githubStarTip')" @click="openGithub">
          <t-icon name="logo-github" class="menu-icon" />
          <span class="menu-text-with-icon">
            <span>{{ $t('common.github') }}</span>
            <t-icon name="star-filled" class="menu-github-star-icon" size="16px" aria-hidden="true" />
            <svg class="menu-external-icon" viewBox="0 0 16 16" aria-hidden="true">
              <path fill="currentColor"
                d="M12.667 8a.667.667 0 0 1 .666.667v4a2.667 2.667 0 0 1-2.666 2.666H4.667a2.667 2.667 0 0 1-2.667-2.666V5.333a2.667 2.667 0 0 1 2.667-2.666h4a.667.667 0 1 1 0 1.333h-4a1.333 1.333 0 0 0-1.333 1.333v7.334A1.333 1.333 0 0 0 4.667 13.333h6a1.333 1.333 0 0 0 1.333-1.333v-4A.667.667 0 0 1 12.667 8Zm2.666-6.667v4a.667.667 0 0 1-1.333 0V3.276l-5.195 5.195a.667.667 0 0 1-.943-.943l5.195-5.195h-2.057a.667.667 0 0 1 0-1.333h4a.667.667 0 0 1 .666.666Z" />
            </svg>
          </span>
        </div>
        <template v-if="!authStore.isLiteMode">
          <div class="menu-divider"></div>
          <div class="menu-item danger" @click="handleLogout">
            <t-icon name="logout" class="menu-icon" />
            <span>{{ $t('auth.logout') }}</span>
          </div>
        </template>
      </div>
    </Transition>

    <!-- Tenant switcher floating panel — shares the same teleport rationale
         as the IM submenu. Data comes from authStore.memberships, kept fresh via
         GET /auth/me when the submenu opens (throttled) and after invite/create. -->
    <Teleport to="body">
      <div v-if="tenantSubmenuOpen" class="tenant-submenu-floating" :style="tenantSubmenuStyle"
        @mouseenter="showTenantSubmenu" @mouseleave="scheduleHideTenantSubmenu">
        <div class="tenant-submenu-header">
          {{ $t('tenant.switcher.menuLabel') }}
        </div>
        <div class="tenant-submenu-list">
          <div v-for="m in switchableMemberships" :key="m.tenant_id" class="tenant-submenu-item"
            :class="{ 'is-current': isCurrentTenant(m.tenant_id) }" @click="switchToTenant(m)">
            <div class="tenant-submenu-item-avatar" :class="{ 'is-current': isCurrentTenant(m.tenant_id) }">
              {{ tenantInitial(m) }}
              <!-- Home marker: add a small home icon to the bottom-right corner of the avatar on the home tenant row. Compared to a separate "mine"
                   pill on the meta line, this is more space-efficient,
                   and keeps the badge column aligned across rows. -->
              <span v-if="isHomeTenant(m.tenant_id)" class="tenant-submenu-item-home-dot"
                :title="$t('tenant.switcher.homeTooltip')">
                <t-icon name="home" size="9px" />
              </span>
            </div>
            <!-- Two-line layout: the first line is the tenant name (takes up all remaining width, so it isn't truncated by badges
                 — previously, when the home and current badges were on the same line, long tenant names
                 got squeezed into an ellipsis); the second line is the role (with role icon) + "current" badge.
                 The home badge has moved to the corner of the tenant name's initial-letter avatar, no longer taking extra space on the meta
                 line, avoiding misaligned badge column widths. -->
            <div class="tenant-submenu-item-info">
              <span class="tenant-submenu-item-name">{{ tenantDisplayName(m) }}</span>
              <div class="tenant-submenu-item-meta">
                <span class="tenant-submenu-item-role">
                  <t-icon v-if="roleIcon(m.role)" :name="roleIcon(m.role)" size="12px"
                    class="tenant-submenu-item-role-icon" />
                  {{ formatRole(m.role) }}
                </span>
                <span v-if="isCurrentTenant(m.tenant_id)" class="tenant-submenu-item-badge">{{
                  $t('tenant.switcher.currentBadge') }}</span>
              </div>
            </div>
          </div>
          <div v-if="switchableMemberships.length === 0" class="tenant-submenu-empty">
            {{ $t('tenant.switcher.empty') }}
          </div>
        </div>
        <!-- Self-service creation entry point stays consistent with the backend capabilities returned by /auth/me. -->
        <div v-if="authStore.canCreateTenant" class="tenant-submenu-create" @click="openCreateTenantDialog">
          <t-icon name="add" class="tenant-submenu-create-icon" />
          <span class="tenant-submenu-create-label">{{ $t('tenant.create.action') }}</span>
        </div>
      </div>
    </Teleport>

    <!-- Create workspace dialog -->
    <CreateTenantDialog v-model:visible="createTenantDialogVisible" @created="onTenantCreated" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useUIStore } from '@/stores/ui'
import { useAuthStore } from '@/stores/auth'
import { MessagePlugin } from 'tdesign-vue-next'
import { getCurrentUser, logout as logoutApi, userInfoFromApi } from '@/api/auth'
import { useI18n } from 'vue-i18n'
import CreateTenantDialog from '@/components/CreateTenantDialog.vue'
import {
  navigateAfterTenantSwitch,
  persistLastActiveTenantPreference,
  stashTenantSwitchToast,
} from '@/utils/tenantSwitch'
import type { TenantInfo } from '@/api/tenant'
import { useRoleLabel, useHomeTenant } from '@/composables/useRoleLabel'
import { getRootZoom, rectToCssPx, cssViewportSize } from '@/utils/zoom'
import { openNewUserGuide } from '@/config/contextualGuides'
import { SETTINGS_MANAGEMENT_SHORTCUT_MIN_ROLE } from '@/config/settingsAccess'

const { t } = useI18n()

const router = useRouter()
const uiStore = useUIStore()
const authStore = useAuthStore()
const { formatRole, roleIcon } = useRoleLabel()
const { homeTenantId, isHomeTenantActive, isHomeTenant } = useHomeTenant()

// Space name / current role shown in the top user card: updates live with the tenant switcher.
// activeTenantName prefers the name selected in the switcher (including fallback to the home tenant name),
// so single-space users can also see their own home tenant name correctly.
const activeTenantName = computed(() => {
  return (
    authStore.selectedTenantName ||
    authStore.tenant?.name ||
    ''
  )
})
const currentRoleLabel = computed(() => formatRole(authStore.currentTenantRole))
const currentRoleIcon = computed(() => roleIcon(authStore.currentTenantRole))

// Single-space users (memberships <= 1 and not superuser) = always home + owner, so the third
// line would just repeat the user-email info, which isn't worth the visual space; only render for multi-space / superuser
// users. Lite mode has no RBAC concept, so it's always hidden.
const showTenantIdentityLine = computed(() => {
  if (authStore.isLiteMode) return false
  if (authStore.canAccessAllTenants) return true
  return (authStore.memberships ?? []).length > 1
})

// Quick-access entries use "management capability" rather than the page's minimum visible role: the member roster and model list allow
// viewer browsing, but the "Manage" entry in the avatar menu only serves roles that can actually perform management actions.
const canManageMembers = computed(() =>
  authStore.canAccessAllTenants || authStore.hasRole(SETTINGS_MANAGEMENT_SHORTCUT_MIN_ROLE.members),
)
const canManageModels = computed(() =>
  authStore.canAccessAllTenants ||
  authStore.isSystemAdmin ||
  authStore.hasRole(SETTINGS_MANAGEMENT_SHORTCUT_MIN_ROLE.models),
)

const menuRef = ref<HTMLElement>()
const tenantMenuItemRef = ref<HTMLElement>()
const menuVisible = ref(false)
const tenantSubmenuOpen = ref(false)
const tenantSubmenuStyle = ref<Record<string, string>>({})
let tenantSubmenuHideTimer: ReturnType<typeof setTimeout> | null = null

// User info
const userInfo = ref({
  username: t('common.defaultUser'),
  email: 'user@example.com',
  avatar: ''
})

const userName = computed(() => userInfo.value.username)
const userEmail = computed(() => userInfo.value.email)
const userAvatar = computed(() => userInfo.value.avatar)

// First letter of the username (shown when there's no avatar)
const userInitial = computed(() => {
  return userName.value.charAt(0).toUpperCase()
})

// Show switch menu
const toggleMenu = () => {
  menuVisible.value = !menuVisible.value
}

// Quick navigation to a specific settings section
const handleQuickNav = (section: string) => {
  menuVisible.value = false
  uiStore.openSettings()
  router.push({ path: '/platform/settings', query: { section } })
}

// Open settings
const handleSettings = () => {
  menuVisible.value = false
  uiStore.openSettings()
  router.push('/platform/settings')
}

// Open the platform administration group inside the standard Settings
// modal. Global settings is the group's landing page; task queues, platform
// API keys and the audit log remain available beside it in the settings nav.
const handleSystemAdmin = () => {
  menuVisible.value = false
  uiStore.openSettings('system-global')
  router.push({ path: '/platform/settings', query: { section: 'system-global' } })
}

// Hover-driven submenu controls. A small hide delay tolerates the pointer
// slipping off briefly onto the gap between menu item and submenu pane.
const closeAll = () => {
  tenantSubmenuOpen.value = false
  menuVisible.value = false
}

// ---------- Create new tenant ----------
// When a regular user clicks "+ Create new workspace" at the bottom of the space submenu → opens CreateTenantDialog →
// the backend writes one owner tenant_members row → switch straight to the new space. Reuses the same
// setSelectedTenant + navigateAfterTenantSwitch chain as switchToTenant, avoiding token
// still pointing at the old space and causing SSE / store inconsistency.
const createTenantDialogVisible = ref(false)

const openCreateTenantDialog = () => {
  closeAll()
  if (!authStore.canCreateTenant) {
    MessagePlugin.info(t('tenant.create.disabled'))
    return
  }
  createTenantDialogVisible.value = true
}

const onTenantCreated = async (newTenant: TenantInfo) => {
  await authStore.refreshFromAuthMe()
  authStore.setSelectedTenant(newTenant.id, newTenant.name)
  const persist = persistLastActiveTenantPreference(newTenant.id)
  Promise.race([persist, new Promise((r) => setTimeout(r, 300))])
    .finally(() => navigateAfterTenantSwitch())
}

// ---------- Tenant switcher submenu ----------
//
// Same hover-driven submenu pattern; data comes from
// authStore.memberships (refreshed from /auth/me when the submenu opens and
// after membership-changing actions). PR 4 of #1303 relaxed the X-Tenant-ID
// gate in middleware/auth.go to accept active membership rows, so flipping
// authStore.selectedTenantId here is enough — the next page reload re-issues
// every request with the new header and the server resolves the role server-side.
type Membership = {
  tenant_id: number
  tenant_name?: string
  role: string
}

// switchableMemberships is the curated list shown in the dropdown. We keep
// the active tenant in there (with a "Current" badge) so the user has a
// single place to glance at "where am I right now"; clicking the current
// row is a no-op (handled in switchToTenant).
const switchableMemberships = computed<Membership[]>(() => {
  return authStore.memberships ?? []
})

// Rendered whenever the user has at least one membership — even single-
// tenant users need this submenu to discover the "create new workspace"
// entry at the bottom. Multi-tenant users additionally use it to switch
// between memberships. Cross-tenant superusers keep using the sidebar
// TenantSelector for the "any tenant in the system" case, so we don't
// double-show that here.
const showTenantSwitcher = computed(() => {
  return switchableMemberships.value.length >= 1
})

const isCurrentTenant = (id: number) => {
  const active = authStore.effectiveTenantId
  return active != null && Number(active) === Number(id)
}

const tenantDisplayName = (m: Membership) =>
  m.tenant_name && m.tenant_name.trim() !== '' ? m.tenant_name : `#${m.tenant_id}`

const tenantInitial = (m: Membership) => {
  const name = tenantDisplayName(m).trim()
  return (name.charAt(0) || '?').toUpperCase()
}

const switchToTenant = (m: Membership) => {
  if (isCurrentTenant(m.tenant_id)) {
    closeAll()
    return
  }
  // Always write the active space into selectedTenantId, so request.ts always attaches X-Tenant-ID.
  // In the old implementation, "clear override when switching back to home" made requests fall back to the space encoded in the JWT,
  // but the JWT, in sessions where last_active != home, happens to be the peer space (see
  // userService.resolveLoginTenantID), so switching back to home actually didn't move at all.
  // Server-side persisted preference still distinguishes home/peer: for home it clears last_active,
  // so the next clean re-login correctly lands on home.
  const home = homeTenantId.value
  const switchingToHome = home !== null && home === m.tenant_id
  authStore.setSelectedTenant(m.tenant_id, tenantDisplayName(m))
  closeAll()
  // The toast is shown by App.vue after reload (showing it here directly would get wiped by the hard reload).
  stashTenantSwitchToast({
    name: tenantDisplayName(m),
    role: formatRole(m.role) || undefined,
    roleEnum: m.role || undefined,
  })
  // Persist "last active tenant" preference (switching to home clears
  // it). Hard reload so every cached store / open SSE stream / in-flight
  // request gets re-keyed under the new tenant; navigateAfterTenantSwitch
  // redirects to the platform home so tenant-scoped resource paths don't
  // white-screen. Race the persist against the existing 400ms grace
  // window so most writes complete before the page tears down.
  const persist = persistLastActiveTenantPreference(switchingToHome ? null : m.tenant_id)
  Promise.race([persist, new Promise((r) => setTimeout(r, 400))])
    .finally(() => navigateAfterTenantSwitch())
}

let lastTenantSubmenuMembershipRefresh = 0
const TENANT_SUBMENU_MEMBERSHIP_REFRESH_MS = 2000

const showTenantSubmenu = () => {
  if (tenantSubmenuHideTimer) {
    clearTimeout(tenantSubmenuHideTimer)
    tenantSubmenuHideTimer = null
  }
  positionTenantSubmenu()
  tenantSubmenuOpen.value = true
  clampFloatingToViewport('.tenant-submenu-floating', tenantSubmenuStyle)
  const now = Date.now()
  if (now - lastTenantSubmenuMembershipRefresh >= TENANT_SUBMENU_MEMBERSHIP_REFRESH_MS) {
    lastTenantSubmenuMembershipRefresh = now
    void authStore.refreshFromAuthMe()
  }
}

const scheduleHideTenantSubmenu = () => {
  if (tenantSubmenuHideTimer) clearTimeout(tenantSubmenuHideTimer)
  tenantSubmenuHideTimer = setTimeout(() => {
    tenantSubmenuOpen.value = false
    tenantSubmenuHideTimer = null
  }, 180)
}

const positionTenantSubmenu = () => {
  const el = tenantMenuItemRef.value
  if (!el) return
  // Submenu is rendered with `position: fixed` under the root zoom — see
  // `.tenant-submenu-floating` styles. Anchor coords come from a visual-pixel
  // rect; normalize to CSS pixels before writing them back to CSS.
  const zoom = getRootZoom()
  const rect = rectToCssPx(el.getBoundingClientRect(), zoom)
  const { width: vw } = cssViewportSize(zoom)
  const PANEL_WIDTH = 264
  const GAP = 8
  const MARGIN = 8

  let left = rect.right + GAP
  if (left + PANEL_WIDTH + MARGIN > vw) {
    left = Math.max(MARGIN, rect.left - PANEL_WIDTH - GAP)
  }

  const top = Math.max(MARGIN, rect.top)

  tenantSubmenuStyle.value = {
    left: `${left}px`,
    top: `${top}px`,
  }
}

// Anchor the floating submenu just to the right of the hovered menu item,
// clamped to the viewport so it stays visible near the screen edge.
const clampFloatingToViewport = (selector: string, target: { value: Record<string, string> }) => {
  requestAnimationFrame(() => {
    const panel = document.querySelector(selector) as HTMLElement | null
    if (!panel) return
    const MARGIN = 8
    // `offsetHeight` and `target.value.top` are CSS pixels; `innerHeight` is
    // visual pixels under root zoom. Normalize the latter to keep the
    // comparison in one coordinate system.
    const { height: vh } = cssViewportSize()
    const h = panel.offsetHeight
    const currentTop = parseFloat(target.value.top || '0') || 0
    const maxTop = vh - h - MARGIN
    if (currentTop > maxTop) {
      target.value = { ...target.value, top: `${Math.max(MARGIN, maxTop)}px` }
    }
  })
}

const reopenGuide = () => {
  menuVisible.value = false
  openNewUserGuide()
}

const openDocs = () => {
  menuVisible.value = false
  window.open('https://github.com/Tencent/WeKnora/tree/main/docs', '_blank')
}

// Open GitHub
const openGithub = () => {
  menuVisible.value = false
  window.open('https://github.com/Tencent/WeKnora', '_blank')
}

// Log out
const handleLogout = async () => {
  menuVisible.value = false

  try {
    // Call the backend API to log out
    await logoutApi()
  } catch (error) {
    // Continue with local cleanup even if the API call fails
    console.error('Logout API call failed:', error)
  }

  // Clear all state and local storage
  authStore.logout()

  MessagePlugin.success(t('auth.logout'))

  // Redirect to the login page
  router.push('/login')
}

// Load user info
const loadUserInfo = async () => {
  try {
    const response = await getCurrentUser()
    if (response.success && response.data && response.data.user) {
      const user = response.data.user
      userInfo.value = {
        username: user.username || t('common.info'),
        email: user.email || 'user@example.com',
        avatar: user.avatar || ''
      }
      // Also update the user info in authStore, ensuring it includes can_access_all_tenants /
      // is_system_admin and all other fields. MUST go through the userInfoFromApi factory — historically
      // hand-writing a field whitelist here meant every new user field had to be synced across 5 setUser calls;
      // missing one spot caused is_system_admin to be silently reset to undefined when user.value's fields
      // were overwritten by loadUserInfo on mount after entering platform
      // (also polluting localStorage) — the system admin entry, triggered by hovering the workspace,
      // only appeared after refreshFromAuthMe. For new fields, only modify userInfoFromApi.
      authStore.setUser(userInfoFromApi(user))
      // If space info is returned, also update the space info; tenantless users (/auth/me
      // must be explicitly cleared, otherwise it will leave behind the workspace snapshot from the previous account/previous session.
      if (response.data.tenant) {
        authStore.setTenant({
          id: String(response.data.tenant.id),
          name: response.data.tenant.name,
          owner_id: user.id,
          created_at: response.data.tenant.created_at,
          updated_at: response.data.tenant.updated_at
        })
      } else {
        authStore.setTenant(null)
      }
      const membershipsSync = response.data.memberships
      if (Array.isArray(membershipsSync)) {
        authStore.setMemberships(membershipsSync)
      }
      const canCreateTenant = response.data.capabilities?.can_create_tenant
      if (typeof canCreateTenant === 'boolean') {
        authStore.setCanCreateTenant(canCreateTenant)
      }
    }
  } catch (error) {
    console.error('Failed to load user info:', error)
  }
}

// Click outside to close the menu
const handleClickOutside = (e: MouseEvent) => {
  const target = e.target as Node
  if (menuRef.value && menuRef.value.contains(target)) return
  // Tenant submenu is teleported to body, so it's not inside menuRef.
  const tenantFloating = document.querySelector('.tenant-submenu-floating')
  if (tenantFloating && tenantFloating.contains(target)) return
  menuVisible.value = false
  tenantSubmenuOpen.value = false
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
  loadUserInfo()
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
})
</script>

<style lang="less" scoped>
.user-menu {
  position: relative;
  width: 100%;

  &--collapsed {
    .user-button {
      justify-content: center;
      padding: 6px 3px;
      gap: 0;
    }

    .user-dropdown {
      left: calc(100% + 8px);
      bottom: 0;
      right: auto;
      /* Align with the dropdown visible width when the sidebar is expanded (aside width 260px) */
      min-width: 260px;
    }
  }
}

.user-button {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 6px;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
  background: transparent;

  &:hover {
    background: var(--td-bg-color-container-hover);
  }

  &:active {
    transform: scale(0.98);
  }
}

.user-avatar {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  overflow: hidden;
  flex-shrink: 0;
  background: linear-gradient(135deg, var(--td-brand-color) 0%, var(--td-brand-color-active) 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: width 0.2s ease, height 0.2s ease;

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .avatar-placeholder {
    color: var(--td-text-color-anti);
    font-size: 12px;
    font-weight: 600;
    line-height: 1;
  }
}

.user-info {
  flex: 1;
  min-width: 0;
  text-align: left;
  display: flex;
  flex-direction: column;
  gap: 2px;
  justify-content: center;

  .user-name {
    font-size: 14px;
    font-weight: 500;
    color: var(--td-text-color-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .user-email {
    font-size: 12px;
    color: var(--td-text-color-secondary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .user-tenant-name {
    font-size: 14px;
    font-weight: 600;
    letter-spacing: -0.01em;
    color: var(--td-text-color-primary);
    line-height: 1.35;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .user-tenant-meta {
    display: flex;
    align-items: center;
    gap: 4px;
    margin-top: 0;
    min-width: 0;
    font-size: 12px;
    line-height: 1.35;
    color: var(--td-text-color-secondary);

    .user-tenant-meta-name {
      flex: 0 1 auto;
      min-width: 0;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    .user-tenant-meta-sep {
      flex-shrink: 0;
      color: var(--td-text-color-placeholder);
    }

    .user-tenant-meta-icon {
      flex-shrink: 0;
      color: inherit;
    }

    .user-tenant-meta-role {
      flex-shrink: 0;
    }
  }
}

.dropdown-icon {
  font-size: 16px;
  color: var(--td-text-color-secondary);
  flex-shrink: 0;
  transition: transform 0.2s;
}

.user-dropdown {
  position: absolute;
  bottom: 100%;
  /* Relative to .user-menu: widened left/right by left/right; right edge inset with a positive value to avoid fully overlapping the sidebar content's right boundary */
  left: -4px;
  right: -5px;
  margin-bottom: 6px;
  background: var(--td-bg-color-container);
  border-radius: 8px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.12);
  border: 1px solid var(--td-component-stroke);
  overflow: hidden;
  z-index: 1000;
}

// Top of dropdown — account section: the 24px avatar center aligns vertically with the 16px menu icon center below;
// margin-left −4px, gap 6px keep the nickname start aligned with the menu text (12 + 24 + 6 − 4 = 38)
.dropdown-user-header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 9px 12px;
  min-width: 0;

  &.is-clickable {
    cursor: pointer;
    transition: background-color 0.15s ease;

    &:hover,
    &:focus-visible {
      background: var(--td-bg-color-container-hover);
      outline: none;
    }
  }

  .dropdown-user-avatar {
    width: 24px;
    height: 24px;
    margin-left: -4px;
    border-radius: 50%;
    overflow: hidden;
    flex-shrink: 0;
    background: linear-gradient(135deg, var(--td-brand-color) 0%, var(--td-brand-color-active) 100%);
    display: flex;
    align-items: center;
    justify-content: center;

    img {
      width: 100%;
      height: 100%;
      object-fit: cover;
    }

    .dropdown-user-avatar-placeholder {
      color: var(--td-text-color-anti);
      font-size: 12px;
      font-weight: 600;
      line-height: 1;
    }
  }

  .dropdown-user-meta {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 0;
    justify-content: center;
  }

  .dropdown-user-name-row {
    display: flex;
    align-items: center;
    gap: 2px;
    min-width: 0;
  }

  .dropdown-user-name {
    flex: 1;
    min-width: 0;
    font-size: 14px;
    font-weight: 500;
    color: var(--td-text-color-primary);
    line-height: 1.35;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .dropdown-user-email {
    min-width: 0;
    font-size: 12px;
    line-height: 1.35;
    color: var(--td-text-color-secondary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .dropdown-guide-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    width: 20px;
    height: 20px;
    margin: 0;
    padding: 0;
    border: none;
    border-radius: 4px;
    background: transparent;
    color: var(--td-text-color-placeholder);
    cursor: pointer;
    transition: background-color 0.2s ease, color 0.2s ease;

    &:hover {
      background: var(--td-bg-color-container-hover);
      color: var(--td-text-color-secondary);
    }
  }
}

// Dropdown — current workspace: same alignment as .menu-item below (16px icon slot on the left + text column + action icon on the right)
.dropdown-tenant-panel {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 9px 12px;
  border-top: 1px solid var(--td-component-stroke);
  background: transparent;
  transition: background 0.15s ease;
  min-width: 0;

  >.menu-icon {
    font-size: 16px;
    color: var(--td-text-color-secondary);
    flex-shrink: 0;
  }

  &.is-clickable {
    cursor: pointer;

    &:hover,
    &.is-open {
      background: var(--td-bg-color-container-hover);

      .dropdown-tenant-panel-trail {
        color: var(--td-text-color-secondary);
      }
    }
  }

  .dropdown-tenant-panel-main {
    flex: 1 1 auto;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 1px;
  }

  .dropdown-tenant-panel-name {
    font-size: 14px;
    font-weight: 500;
    color: var(--td-text-color-primary);
    line-height: 1.35;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .dropdown-tenant-panel-trail {
    flex-shrink: 0;
    font-size: 16px;
    color: var(--td-text-color-placeholder);
    transition: color 0.15s ease;
  }

  .dropdown-tenant-panel-role {
    display: flex;
    align-items: center;
    gap: 4px;
    font-size: 12px;
    line-height: 1.35;
    color: var(--td-text-color-secondary);
    min-width: 0;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;

    .dropdown-tenant-panel-role-icon {
      flex-shrink: 0;
      color: inherit;
    }
  }
}

.menu-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 9px 12px;
  cursor: pointer;
  transition: all 0.2s;
  font-size: 14px;
  color: var(--td-text-color-primary);

  &:hover {
    background: var(--td-bg-color-container-hover);
  }

  &.danger {
    color: var(--td-error-color);

    &:hover {
      background: var(--td-error-color-light);
    }

    .menu-icon {
      color: var(--td-error-color);
    }
  }

  // Menu item containing the right-side submenu
  &--submenu {
    position: relative;

    .menu-item-label {
      flex: 1;
    }

    .menu-chevron {
      font-size: 16px;
      color: var(--td-text-color-placeholder);
      flex-shrink: 0;
      transition: transform 0.15s;
    }

    &.is-open {
      background: var(--td-bg-color-container-hover);

      .menu-chevron {
        color: var(--td-text-color-secondary);
      }
    }
  }

  .menu-icon {
    font-size: 16px;
    color: var(--td-text-color-secondary);

    &.svg-icon {
      width: 16px;
      height: 16px;
      flex-shrink: 0;
    }

    &--emoji {
      width: 16px;
      height: 16px;
      display: inline-flex;
      align-items: center;
      justify-content: center;
      font-size: 15px;
      line-height: 1;
      flex-shrink: 0;
      color: inherit;
    }
  }

  .menu-text-with-icon {
    flex: 1;
    display: flex;
    align-items: center;
    gap: 6px;
    color: inherit;
    min-width: 0;

    >span:first-of-type {
      display: inline-flex;
      align-items: center;
      min-width: 0;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
  }

  .menu-new-badge {
    flex-shrink: 0;
    font-size: 10px;
    font-weight: 600;
    line-height: 1.2;
    padding: 2px 5px;
    border-radius: 4px;
    background: var(--td-brand-color-light);
    color: var(--td-brand-color);
    letter-spacing: 0.02em;
  }

  .menu-github-star-icon {
    flex-shrink: 0;
    color: var(--td-warning-color);
  }

  .menu-external-icon {
    width: 16px;
    height: 16px;
    color: var(--td-text-color-disabled);
    flex-shrink: 0;
    transition: color 0.2s ease;
    pointer-events: none;
  }

  &:hover .menu-external-icon {
    color: var(--td-brand-color);
  }
}

.menu-divider {
  height: 1px;
  background: var(--td-component-stroke);
  margin: 3px 0;
}

// Divider right after the account/space block: slightly tighten the spacing above it
.dropdown-user-header+.menu-divider,
.dropdown-tenant-panel+.menu-divider {
  margin-top: 1px;
}

// Dropdown animation
.dropdown-enter-active,
.dropdown-leave-active {
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
}

.dropdown-enter-from,
.dropdown-leave-to {
  opacity: 0;
  transform: translateY(8px);
}

.dropdown-enter-to,
.dropdown-leave-from {
  opacity: 1;
  transform: translateY(0);
}

</style>

<style lang="less">
// Tenant switcher submenu — teleported to <body>.
// All styling for the panel itself lives here (not in a child component) so
// the markup in UserMenu.vue stays self-contained.
.tenant-submenu-floating {
  position: fixed;
  z-index: 1100;
  width: 264px;
  max-height: 340px;
  display: flex;
  flex-direction: column;
  background: var(--td-bg-color-container);
  border: 0.5px solid var(--td-component-stroke);
  border-radius: 10px;
  box-shadow: 0 6px 24px rgba(0, 0, 0, 0.12);
  // Pointer bridge so the user can slide off the menu item onto the panel
  // without hitting the gap and triggering mouseleave-hide.
  padding-left: 2px;
  overflow: hidden;

  .tenant-submenu-header {
    padding: 8px 12px 6px;
    font-size: 12px;
    font-weight: 600;
    color: var(--td-text-color-secondary);
    border-bottom: 0.5px solid var(--td-component-stroke);
  }

  .tenant-submenu-list {
    overflow-y: auto;
    padding: 4px;
  }

  .tenant-submenu-item {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 7px 8px;
    border-radius: 6px;
    cursor: pointer;
    transition: background 0.15s;

    &:hover {
      background: var(--td-bg-color-secondarycontainer);
    }

    &.is-current {
      background: var(--td-bg-color-secondarycontainer);
      cursor: default;

      .tenant-submenu-item-name {
        color: var(--td-text-color-primary);
        font-weight: 600;
      }
    }
  }

  .tenant-submenu-item-avatar {
    width: 28px;
    height: 28px;
    border-radius: 6px;
    background: var(--td-bg-color-secondarycontainer);
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 13px;
    font-weight: 600;
    color: var(--td-text-color-secondary);
    flex-shrink: 0;

    &.is-current {
      background: linear-gradient(135deg, var(--td-brand-color) 0%, var(--td-brand-color-active) 100%);
      color: var(--td-text-color-anti);
    }
  }

  .tenant-submenu-item-info {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .tenant-submenu-item-name {
    font-size: 13px;
    color: var(--td-text-color-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  // Second row: role + badge, laid out inline together. The badge is shrunk to a secondary position,
  // letting the first row's tenant name take up the full width (previously long names were squeezed into an ellipsis by the badge).
  .tenant-submenu-item-meta {
    display: flex;
    align-items: center;
    gap: 6px;
    flex-wrap: wrap;
    min-width: 0;
  }

  .tenant-submenu-item-role {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    font-size: 11px;
    color: var(--td-text-color-placeholder);

    .tenant-submenu-item-role-icon {
      flex-shrink: 0;
      // Color inherits the role text color, avoiding drawing visual attention
      color: inherit;
    }
  }

  .tenant-submenu-item-badge {
    flex-shrink: 0;
    font-size: 10px;
    font-weight: 600;
    line-height: 1.2;
    padding: 2px 6px;
    border-radius: 4px;
    background: var(--td-bg-color-component);
    color: var(--td-text-color-secondary);
  }

  // The Home indicator is changed to a small dot overlaid on the bottom-right of the avatar, taking no extra space in the meta row, so
  // the badge column stays aligned across rows; when the user switches to a non-home tenant, this small icon still shows at a glance
  // which row is "my home space."
  .tenant-submenu-item-avatar {
    position: relative;
  }

  .tenant-submenu-item-home-dot {
    position: absolute;
    right: -3px;
    bottom: -3px;
    width: 14px;
    height: 14px;
    border-radius: 50%;
    background: var(--td-bg-color-container);
    color: var(--td-text-color-secondary);
    border: 1.5px solid var(--td-bg-color-container);
    display: flex;
    align-items: center;
    justify-content: center;
    pointer-events: none;
    box-shadow: 0 0 0 0.5px var(--td-success-color-light);
  }

  .tenant-submenu-empty {
    padding: 12px 10px;
    text-align: center;
    font-size: 12px;
    color: var(--td-text-color-placeholder);
  }

  .tenant-submenu-create {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 8px 10px;
    margin: 3px 4px 5px;
    border-top: .5px solid var(--td-component-stroke);
    border-radius: 6px;
    cursor: pointer;
    color: var(--td-brand-color);
    font-size: 14px;
    font-weight: 500;
    transition: background 0.15s;

    &:hover {
      background: rgba(7, 192, 95, 0.08);
    }

    .tenant-submenu-create-icon {
      font-size: 16px;
      flex-shrink: 0;
    }

    .tenant-submenu-create-label {
      flex: 1;
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
      font-size: 12px;
    }
  }
}
</style>
