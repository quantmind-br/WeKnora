<template>
  <div class="tenant-selector" ref="selectorRef">
    <div class="tenant-trigger" @click="toggleDropdown">
      <div class="tenant-info">
        <div class="tenant-label">{{ $t('tenant.currentTenant') }}</div>
        <div class="tenant-name-row">
          <span class="tenant-name">{{ currentTenantName }}</span>
          <t-icon name="swap" class="tenant-switch-icon" />
        </div>
      </div>
    </div>

    <Transition name="dropdown">
      <div v-if="showDropdown" class="tenant-dropdown" @click.stop>
        <div class="dropdown-header">
          <span class="dropdown-title">{{ $t('tenant.switchTenant') }}</span>
          <div class="search-box">
            <t-icon name="search" class="search-icon" />
            <input ref="searchInput" v-model="searchQuery" type="text" :placeholder="$t('tenant.searchPlaceholder')"
              class="search-input" @keydown.esc="closeDropdown" @input="handleSearchInput" />
            <t-icon v-if="searchQuery" name="close-circle-filled" class="clear-icon" @click="clearSearch" />
          </div>
        </div>

        <div class="tenant-list" ref="tenantListRef" @scroll="handleScroll">
          <div v-if="loading && tenants.length === 0" class="tenant-loading">
            <t-loading size="small" />
            <span>{{ $t('tenant.loading') }}</span>
          </div>

          <template v-else-if="tenants.length > 0">
            <div v-for="tenant in tenants" :key="tenant.id"
              :class="['tenant-item', { selected: isSelected(tenant.id) }]" @click="selectTenant(tenant.id)">
              <div class="tenant-item-content">
                <div class="tenant-item-avatar" :class="{ active: isSelected(tenant.id) }">
                  {{ tenant.name.charAt(0).toUpperCase() }}
                </div>
                <div class="tenant-item-info">
                  <span class="tenant-item-name">{{ tenant.name }}</span>
                  <span class="tenant-item-id">ID: {{ tenant.id }}</span>
                </div>
              </div>
              <t-icon v-if="isSelected(tenant.id)" name="check" size="16px" class="check-icon" />
            </div>
          </template>

          <div v-else class="tenant-empty">
            <span>{{ $t('tenant.noMatch') }}</span>
          </div>

          <div v-if="loading && tenants.length > 0" class="tenant-loading-more">
            <t-loading size="small" />
          </div>
        </div>

        <!-- The self-service creation entry stays consistent with the backend capability returned by /auth/me. -->
        <div v-if="authStore.canCreateTenant" class="tenant-create-action" @click="openCreateDialog">
          <t-icon name="add" class="tenant-create-icon" />
          <span class="tenant-create-label">{{ $t('tenant.create.action') }}</span>
        </div>
      </div>
    </Transition>

    <!-- Overlay layer -->
    <div v-if="showDropdown" class="tenant-overlay" @click="closeDropdown"></div>

    <!-- Create workspace dialog: reuses the shared component, used by both TenantSelector and UserMenu -->
    <CreateTenantDialog v-model:visible="createDialogVisible" @created="onTenantCreated" />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch, onUnmounted, nextTick } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { searchTenants, type TenantInfo } from '@/api/tenant'
import { useI18n } from 'vue-i18n'
import { MessagePlugin } from 'tdesign-vue-next'
import {
  navigateAfterTenantSwitch,
  persistLastActiveTenantPreference,
  stashTenantSwitchToast,
} from '@/utils/tenantSwitch'
import CreateTenantDialog from '@/components/CreateTenantDialog.vue'
import { useRoleLabel } from '@/composables/useRoleLabel'

const { t } = useI18n()
const authStore = useAuthStore()
const { formatRole } = useRoleLabel()

const showDropdown = ref(false)
const searchQuery = ref('')
const tenants = ref<TenantInfo[]>([])
const selectorRef = ref<HTMLElement | null>(null)
const tenantListRef = ref<HTMLElement | null>(null)
const searchInput = ref<HTMLInputElement | null>(null)

// Pagination-related
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)
const loading = ref(false)
const searchTimer = ref<number | null>(null)

const selectedTenantId = computed(() => authStore.selectedTenantId)
// home space id comes from user.tenant_id (assigned at registration, never changes). Don't read
// authStore.tenant.id — that's the currently active space, which switches with X-Tenant-ID; use it
// treating it as home would cause the "switch back to home" branch to misjudge; see the useHomeTenant() comment for details.
const defaultTenantId = computed(() =>
  authStore.user?.tenant_id ? Number(authStore.user.tenant_id) : null,
)

const currentTenantId = computed(() => {
  return selectedTenantId.value || defaultTenantId.value
})

const currentTenantName = computed(() => {
  if (!currentTenantId.value) return t('tenant.unknown')
  // First look it up from the currently loaded space list
  const tenant = tenants.value.find(t => t.id === currentTenantId.value)
  if (tenant) return tenant.name
  // If it's the selected space, use the saved space name
  if (selectedTenantId.value && authStore.selectedTenantName) {
    return authStore.selectedTenantName
  }
  // Finally, fall back to the default space name
  return authStore.tenant?.name || t('tenant.unknown')
})

const hasMore = computed(() => {
  return tenants.value.length < total.value
})

const isSelected = (tenantId: number) => {
  return currentTenantId.value === tenantId
}

const toggleDropdown = () => {
  showDropdown.value = !showDropdown.value
  if (showDropdown.value) {
    if (tenants.value.length === 0) {
      loadTenants()
    }
    nextTick(() => {
      searchInput.value?.focus()
    })
  }
}

const closeDropdown = () => {
  showDropdown.value = false
  searchQuery.value = ''
  currentPage.value = 1
  if (searchTimer.value) {
    clearTimeout(searchTimer.value)
    searchTimer.value = null
  }
}

const clearSearch = () => {
  searchQuery.value = ''
  currentPage.value = 1
  tenants.value = []
  total.value = 0
  loadTenants()
}

const selectTenant = (tenantId: number) => {
  // Found the selected space info
  const selectedTenant = tenants.value.find(t => t.id === tenantId)

  // Always write the override, so request.ts always attaches X-Tenant-ID to override the JWT; don't
  // clear it just because switching to home (see the same-named comment in UserMenu.switchToTenant). Server-side persisted
  // preference is still handled separately — clear last_active when switching to home, so the next clean re-login lands on home.
  const switchingToHome = tenantId === defaultTenantId.value
  // When switching to home, selectedTenant may not have home loaded into the list due to pagination / search,
  // so fall back to picking the name from memberships. Note: do not fall back to authStore.tenant?.name
  // — that's the name of the currently active space, which in sessions where active != home is the peer's name.
  const homeNameFallback = switchingToHome
    ? (authStore.memberships ?? []).find((m) => Number(m.tenant_id) === tenantId)?.tenant_name
      || null
    : null
  authStore.setSelectedTenant(tenantId, selectedTenant?.name || homeNameFallback || null)
  closeDropdown()
  const displayName = selectedTenant?.name
    || homeNameFallback
    || `#${tenantId}`
  // Cross-tenant superusers may not have a membership row in the target
  // tenant; in that case skip the role line rather than show a misleading
  // empty/raw value.
  const membership = (authStore.memberships ?? []).find((m) => Number(m.tenant_id) === tenantId)
  const roleLabel = membership ? formatRole(membership.role) : ''
  // The toast is shown by App.vue after reload (showing it directly here would get killed by the hard reload).
  stashTenantSwitchToast({
    name: displayName,
    role: roleLabel || undefined,
    roleEnum: membership?.role || undefined,
  })
  // Persist "last active tenant" preference (switching to home clears
  // it). Fire-and-forget, but race it against the existing 500ms grace
  // window so most writes finish before the hard reload tears the page
  // down. After switching spaces, navigate to a safe entry point under the new space (see tenantSwitch.ts comment).
  const persist = persistLastActiveTenantPreference(switchingToHome ? null : tenantId)
  Promise.race([persist, new Promise((r) => setTimeout(r, 500))])
    .finally(() => navigateAfterTenantSwitch())
}

const loadTenants = async (append = false) => {
  if (loading.value) return

  loading.value = true
  try {
    const keyword = searchQuery.value.trim()
    let tenantID: number | undefined = undefined

    // If it's purely numeric, search by it as both tenant_id and keyword
    // this way it can both exact-match the space ID and fuzzy-match spaces whose name contains the number
    if (keyword && /^\d+$/.test(keyword)) {
      tenantID = Number(keyword)
    }

    const response = await searchTenants({
      keyword: keyword || undefined,
      tenant_id: tenantID,
      page: currentPage.value,
      page_size: pageSize.value
    })

    if (response.success && response.data) {
      if (append) {
        tenants.value = [...tenants.value, ...response.data.items]
      } else {
        tenants.value = response.data.items
      }
      total.value = response.data.total
      authStore.setAllTenants(tenants.value)
    } else {
      MessagePlugin.error(response.message || t('tenant.loadTenantsFailed'))
    }
  } catch (error) {
    console.error('Failed to load tenants:', error)
    MessagePlugin.error(t('tenant.loadTenantsFailed'))
  } finally {
    loading.value = false
  }
}

const handleSearchInput = () => {
  if (searchTimer.value) {
    clearTimeout(searchTimer.value)
  }

  searchTimer.value = window.setTimeout(() => {
    currentPage.value = 1
    tenants.value = []
    total.value = 0
    loadTenants()
  }, 300)
}

const handleScroll = () => {
  if (!tenantListRef.value) return

  const { scrollTop, scrollHeight, clientHeight } = tenantListRef.value
  const isNearBottom = scrollHeight - scrollTop - clientHeight < 50

  if (isNearBottom && hasMore.value && !loading.value) {
    currentPage.value++
    loadTenants(true)
  }
}

// ---- Create new workspace ----
// The dialog is rendered by the shared component CreateTenantDialog; this only handles opening / receiving the creation result.
const createDialogVisible = ref(false)

const openCreateDialog = () => {
  closeDropdown()
  if (!authStore.canCreateTenant) {
    MessagePlugin.info(t('tenant.create.disabled'))
    return
  }
  createDialogVisible.value = true
}

const onTenantCreated = async (newTenant: TenantInfo) => {
  // Merge the new space into the current list and switch to it. Follows the same path as selectTenant:
  // setSelectedTenant + navigateAfterTenantSwitch. In the backend, X-Tenant-ID middle-
  // ware checks tenant_members; EnsureOwner has already written the owner row on the backend.
  tenants.value = [newTenant, ...tenants.value.filter(t => t.id !== newTenant.id)]
  total.value = total.value + 1
  authStore.setAllTenants(tenants.value)
  await authStore.refreshFromAuthMe()
  authStore.setSelectedTenant(newTenant.id, newTenant.name)
  // Newly-created tenant becomes the user's "last active" so re-login
  // lands here. Race against the existing grace window before reload.
  const persist = persistLastActiveTenantPreference(newTenant.id)
  Promise.race([persist, new Promise((r) => setTimeout(r, 300))])
    .finally(() => navigateAfterTenantSwitch())
}

onMounted(() => {
  // Preload the space list
  loadTenants()
})

onUnmounted(() => {
  if (searchTimer.value) {
    clearTimeout(searchTimer.value)
  }
})
</script>

<style scoped lang="less">
.tenant-selector {
  position: relative;
  margin: 0 0 12px;
}

.tenant-trigger {
  display: flex;
  align-items: center;
  padding: 10px 12px;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s;
  background: var(--td-bg-color-secondarycontainer);
  border: .5px solid var(--td-component-stroke);

  &:hover {
    background: var(--td-bg-color-container-hover);
    border-color: var(--td-component-border);
  }
}

.tenant-info {
  flex: 1;
  min-width: 0;
}

.tenant-label {
  font-size: 11px;
  color: var(--td-text-color-placeholder);
  margin-bottom: 2px;
  font-weight: 500;
}

.tenant-name-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}

.tenant-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--td-text-color-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  flex: 1;
}

.tenant-switch-icon {
  font-size: 14px;
  color: var(--td-brand-color);
  flex-shrink: 0;
}

.tenant-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 999;
}

.tenant-dropdown {
  position: absolute;
  top: calc(100% + 4px);
  left: 0;
  right: 0;
  background: var(--td-bg-color-container);
  border: .5px solid var(--td-component-stroke);
  border-radius: 10px;
  box-shadow: 0 6px 24px rgba(0, 0, 0, 0.12);
  z-index: 1000;
  overflow: hidden;
}

.dropdown-header {
  padding: 12px;
  border-bottom: .5px solid var(--td-component-stroke);
}

.dropdown-title {
  display: block;
  font-size: 12px;
  font-weight: 600;
  color: var(--td-text-color-secondary);
  margin-bottom: 8px;
}

.search-box {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 7px 10px;
  background: var(--td-bg-color-secondarycontainer);
  border-radius: 6px;
  border: .5px solid transparent;
  transition: all 0.2s;

  &:focus-within {
    background: var(--td-bg-color-container);
    border-color: var(--td-brand-color);
    box-shadow: 0 0 0 2px rgba(7, 192, 95, 0.1);
  }
}

.search-icon {
  font-size: 14px;
  color: var(--td-text-color-placeholder);
  flex-shrink: 0;
}

.search-input {
  flex: 1;
  border: none;
  outline: none;
  background: transparent;
  font-size: 13px;
  color: var(--td-text-color-primary);
  min-width: 0;

  &::placeholder {
    color: var(--td-text-color-placeholder);
  }
}

.clear-icon {
  font-size: 14px;
  color: var(--td-text-color-placeholder);
  cursor: pointer;
  flex-shrink: 0;
  transition: color 0.2s;

  &:hover {
    color: var(--td-text-color-secondary);
  }
}

.tenant-list {
  max-height: 280px;
  overflow-y: auto;
  padding: 6px;

  &::-webkit-scrollbar {
    width: 4px;
  }

  &::-webkit-scrollbar-track {
    background: transparent;
  }

  &::-webkit-scrollbar-thumb {
    background: var(--td-bg-color-secondarycontainer);
    border-radius: 2px;

    &:hover {
      background: var(--td-bg-color-component-disabled);
    }
  }
}

.tenant-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 10px;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s;
  margin-bottom: 2px;

  &:last-child {
    margin-bottom: 0;
  }

  &:hover {
    background: var(--td-bg-color-secondarycontainer);
  }

  &.selected {
    background: rgba(7, 192, 95, 0.08);

    .tenant-item-name {
      color: var(--td-brand-color);
      font-weight: 500;
    }
  }
}

.tenant-item-content {
  display: flex;
  align-items: center;
  gap: 10px;
  flex: 1;
  min-width: 0;
}

.tenant-item-avatar {
  width: 32px;
  height: 32px;
  border-radius: 6px;
  background: var(--td-bg-color-secondarycontainer);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  font-weight: 600;
  color: var(--td-text-color-secondary);
  flex-shrink: 0;
  transition: all 0.2s;

  &.active {
    background: linear-gradient(135deg, var(--td-brand-color) 0%, var(--td-brand-color-active) 100%);
    color: var(--td-text-color-anti);
  }
}

.tenant-item-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.tenant-item-name {
  font-size: 13px;
  color: var(--td-text-color-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.tenant-item-id {
  font-size: 11px;
  color: var(--td-text-color-placeholder);
}

.check-icon {
  color: var(--td-brand-color);
  flex-shrink: 0;
}

.tenant-loading,
.tenant-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 24px 12px;
  gap: 8px;
  color: var(--td-text-color-placeholder);
  font-size: 13px;
}

.tenant-loading-more {
  display: flex;
  justify-content: center;
  padding: 8px;
}

.tenant-create-action {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  margin: 4px 6px 6px;
  border-top: .5px solid var(--td-component-stroke);
  border-radius: 6px;
  cursor: pointer;
  color: var(--td-brand-color);
  font-size: 13px;
  font-weight: 500;
  transition: background 0.15s;

  &:hover {
    background: rgba(7, 192, 95, 0.08);
  }
}

.tenant-create-icon {
  font-size: 14px;
  flex-shrink: 0;
}

.tenant-create-label {
  flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

// Dropdown animation
.dropdown-enter-active,
.dropdown-leave-active {
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
}

.dropdown-enter-from,
.dropdown-leave-to {
  opacity: 0;
  transform: translateY(-6px);
}

.dropdown-enter-to,
.dropdown-leave-from {
  opacity: 1;
  transform: translateY(0);
}
</style>
