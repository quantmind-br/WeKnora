<template>
  <div class="tenant-info">
    <div class="section-header">
      <h2>{{ $t('tenant.title') }}</h2>
      <p class="section-description">{{ $t('tenant.sectionDescription') }}</p>
    </div>

    <!-- Loading state -->
    <div v-if="loading" class="loading-inline">
      <t-loading size="small" />
      <span>{{ $t('tenant.loadingInfo') }}</span>
    </div>

    <!-- Error state -->
    <div v-else-if="error" class="error-inline">
      <t-alert theme="error" :message="error">
        <template #operation>
          <t-button size="small" @click="loadInfo">{{ $t('tenant.retry') }}</t-button>
        </template>
      </t-alert>
    </div>

    <!-- Content: info list + danger zone section, avoid mixing dashed lines with the setting-row bottom border -->
    <div v-else class="tenant-info-body">
      <div class="settings-group">
        <!-- Tenant ID -->
        <div class="setting-row">
          <div class="setting-info">
            <label>{{ $t('tenant.details.idLabel') }}</label>
            <p class="desc">{{ $t('tenant.details.idDescription') }}</p>
          </div>
          <div class="setting-control">
            <span class="info-value">{{ tenantInfo?.id || '-' }}</span>
          </div>
        </div>

        <!-- Tenant name -->
        <div class="setting-row">
          <div class="setting-info">
            <label>{{ $t('tenant.details.nameLabel') }}</label>
            <p class="desc">{{ $t('tenant.details.nameDescription') }}</p>
          </div>
          <div class="setting-control">
            <!-- Read-only state: display name + edit button (edit entry only visible to owner).
               In-place editing replaces the dialog: one fewer layer of visual interruption, consistent with other rows' display rhythm. -->
            <template v-if="!editing">
              <span class="info-value">{{ tenantInfo?.name || '-' }}</span>
              <t-button v-if="canEditTenant" theme="default" variant="text" shape="square" size="small"
                class="edit-btn" :title="$t('tenant.details.editName')" :aria-label="$t('tenant.details.editName')"
                @click="startEditName">
                <template #icon>
                  <t-icon name="edit" />
                </template>
              </t-button>
            </template>
            <!-- Edit state: input field + save/cancel. Enter to save, Esc to cancel. -->
            <div v-else class="inline-edit">
              <t-input v-model="editName" :placeholder="$t('tenant.details.editNamePlaceholder')" :maxlength="64"
                :disabled="saving" autofocus class="inline-edit-input" @enter="saveTenantName"
                @keydown="onEditKeydown" />
              <t-button theme="primary" size="small" :loading="saving" :disabled="!canSubmit" @click="saveTenantName">
                {{ $t('tenant.details.editNameConfirm') }}
              </t-button>
              <t-button theme="default" variant="outline" size="small" :disabled="saving" @click="cancelEditName">
                {{ $t('tenant.details.editNameCancel') }}
              </t-button>
            </div>
          </div>
        </div>

        <!-- Tenant description -->
        <div class="setting-row">
          <div class="setting-info">
            <label>{{ $t('tenant.details.descriptionLabel') }}</label>
            <p class="desc">{{ $t('tenant.details.descriptionDescription') }}</p>
          </div>
          <div class="setting-control">
            <!-- Read-only state: display description (placeholder when empty) + edit button (edit entry only visible to owner).
               Same "in-place edit" pattern as name, one fewer layer of dialog interruption. -->
            <template v-if="!editingDescription">
              <span class="info-value description-value" :class="{ 'is-empty': !tenantInfo?.description }">
                {{ tenantInfo?.description || $t('tenant.details.descriptionEmptyPlaceholder') }}
              </span>
              <t-button v-if="canEditTenant" theme="default" variant="text" shape="square" size="small"
                class="edit-btn" :title="$t('tenant.details.editDescription')"
                :aria-label="$t('tenant.details.editDescription')" @click="startEditDescription">
                <template #icon>
                  <t-icon name="edit" />
                </template>
              </t-button>
            </template>
            <!-- Edit state: textarea + save/cancel. Esc to cancel, Ctrl/⌘+Enter to save;
               Enter defaulting to newline in a textarea is more natural — it doesn't take over Enter for submit. -->
            <div v-else class="inline-edit inline-edit-description">
              <t-textarea v-model="editDescription"
                :placeholder="$t('tenant.details.editDescriptionPlaceholder')" :maxlength="512"
                :autosize="{ minRows: 2, maxRows: 6 }" :disabled="savingDescription" autofocus
                class="inline-edit-textarea" @keydown="onEditDescriptionKeydown" />
              <div class="inline-edit-actions">
                <t-button theme="primary" size="small" :loading="savingDescription"
                  :disabled="!canSubmitDescription" @click="saveTenantDescription">
                  {{ $t('tenant.details.editNameConfirm') }}
                </t-button>
                <t-button theme="default" variant="outline" size="small" :disabled="savingDescription"
                  @click="cancelEditDescription">
                  {{ $t('tenant.details.editNameCancel') }}
                </t-button>
              </div>
            </div>
          </div>
        </div>

        <!-- Tenant business -->
        <div v-if="tenantInfo?.business" class="setting-row">
          <div class="setting-info">
            <label>{{ $t('tenant.details.businessLabel') }}</label>
            <p class="desc">{{ $t('tenant.details.businessDescription') }}</p>
          </div>
          <div class="setting-control">
            <span class="info-value">{{ tenantInfo.business }}</span>
          </div>
        </div>

        <!-- Tenant status -->
        <div class="setting-row">
          <div class="setting-info">
            <label>{{ $t('tenant.details.statusLabel') }}</label>
            <p class="desc">{{ $t('tenant.details.statusDescription') }}</p>
          </div>
          <div class="setting-control">
            <t-tag :theme="getStatusTheme(tenantInfo?.status)" variant="light" size="small">
              {{ getStatusText(tenantInfo?.status) }}
            </t-tag>
          </div>
        </div>

        <!-- Tenant creation time -->
        <div class="setting-row">
          <div class="setting-info">
            <label>{{ $t('tenant.details.createdAtLabel') }}</label>
            <p class="desc">{{ $t('tenant.details.createdAtDescription') }}</p>
          </div>
          <div class="setting-control">
            <span class="info-value">{{ formatDate(tenantInfo?.created_at) }}</span>
          </div>
        </div>

        <!-- Storage quota -->
        <div v-if="tenantInfo?.storage_quota !== undefined" class="setting-row">
          <div class="setting-info">
            <label>{{ $t('tenant.storage.quotaLabel') }}</label>
            <p class="desc">{{ $t('tenant.storage.quotaDescription') }}</p>
          </div>
          <div class="setting-control">
            <span class="info-value">{{ formatBytes(tenantInfo.storage_quota) }}</span>
          </div>
        </div>

        <!-- Used storage -->
        <div v-if="tenantInfo?.storage_quota !== undefined" class="setting-row">
          <div class="setting-info">
            <label>{{ $t('tenant.storage.usedLabel') }}</label>
            <p class="desc">{{ $t('tenant.storage.usedDescription') }}</p>
          </div>
          <div class="setting-control">
            <span class="info-value">{{ formatBytes(tenantInfo.storage_used || 0) }}</span>
          </div>
        </div>

        <!-- Storage usage -->
        <div v-if="tenantInfo?.storage_quota !== undefined" class="setting-row">
          <div class="setting-info">
            <label>{{ $t('tenant.storage.usageLabel') }}</label>
            <p class="desc">{{ $t('tenant.storage.usageDescription') }}</p>
          </div>
          <div class="setting-control">
            <div class="usage-control">
              <span class="usage-text">{{ getUsagePercentage() }}%</span>
              <!-- t-progress: theme = shape (line/plump/circle); color uses status -->
              <t-progress :percentage="getUsagePercentage()" :show-info="false" size="small"
                :status="getUsagePercentage() > 80 ? 'warning' : 'success'" style="flex: 1;" />
            </div>
          </div>
        </div>

      </div>

      <aside v-if="showLeaveDangerZone" class="leave-space-panel" :aria-label="$t('tenant.leaveDangerZone.title')">
        <div class="leave-space-panel-inner">
          <div class="leave-space-panel-text">
            <div class="leave-space-panel-title">{{ $t('tenant.leaveDangerZone.title') }}</div>
            <p class="leave-space-panel-desc">{{ $t('tenant.leaveDangerZone.desc') }}</p>
          </div>
          <div class="leave-space-panel-action">
            <t-button theme="danger" variant="outline" size="medium" @click="confirmLeaveTenant">
              {{ $t('tenant.leaveDangerZone.button') }}
            </t-button>
          </div>
        </div>
      </aside>

      <aside v-if="showDeleteDangerZone" class="leave-space-panel delete-space-panel"
        :aria-label="$t('tenant.deleteDangerZone.title')">
        <div class="leave-space-panel-inner">
          <div class="leave-space-panel-text">
            <div class="leave-space-panel-title">{{ $t('tenant.deleteDangerZone.title') }}</div>
            <p class="leave-space-panel-desc">{{ $t('tenant.deleteDangerZone.desc') }}</p>
          </div>
          <div class="leave-space-panel-action">
            <t-button theme="danger" size="medium" @click="confirmDeleteTenant">
              {{ $t('tenant.deleteDangerZone.button') }}
            </t-button>
          </div>
        </div>
      </aside>

    </div>

    <t-dialog v-model:visible="deleteTenantVisible" :header="$t('tenant.deleteDangerZone.confirmTitle')"
      :confirm-btn="{
        content: $t('tenant.deleteDangerZone.confirm'),
        theme: 'danger',
        disabled: deleteConfirmName.trim() !== (tenantInfo?.name || ''),
        loading: deletingTenant,
      }" :cancel-btn="$t('common.cancel')" :close-on-overlay-click="!deletingTenant"
      :close-btn="!deletingTenant" @confirm="deleteCurrentTenant">
      <div class="delete-tenant-confirm">
        <p class="delete-tenant-confirm-body">
          {{ $t('tenant.deleteDangerZone.confirmBody', { name: tenantInfo?.name || '' }) }}
        </p>
        <p class="delete-tenant-confirm-hint">
          {{ $t('tenant.deleteDangerZone.confirmHint', { name: tenantInfo?.name || '' }) }}
        </p>
        <t-input v-model="deleteConfirmName" :placeholder="tenantInfo?.name || ''" :disabled="deletingTenant"
          clearable />
      </div>
    </t-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { DialogPlugin, MessagePlugin } from 'tdesign-vue-next'
import { getCurrentUser, type TenantInfo } from '@/api/auth'
import { deleteTenant as deleteTenantApi, updateTenant as updateTenantApi } from '@/api/tenant'
import {
  leaveTenant,
  fetchAllTenantMembers,
  type TenantMember,
  type TenantRole,
} from '@/api/tenant/members'
import { useAuthStore } from '@/stores/auth'
import { useI18n } from 'vue-i18n'
import { useRoleLabel, useHomeTenant } from '@/composables/useRoleLabel'
import {
  navigateAfterTenantSwitch,
  persistLastActiveTenantPreference,
  stashTenantSwitchToast,
} from '@/utils/tenantSwitch'

const { t, locale } = useI18n()
const { formatRole } = useRoleLabel()
const { homeTenantId } = useHomeTenant()
const authStore = useAuthStore()

// Reactive state
const tenantInfo = ref<TenantInfo | null>(null)
const loading = ref(true)
const error = ref('')

// Only the owner can change the space name (consistent with the g.Owner() guard in the backend's router.go;
// the server is always the final authority on permissions — this only decides whether the UI exposes the entry).
const canEditTenant = computed(() => authStore.hasRole('owner'))

/** Consistent with the original TenantMembers.vue: the last remaining Owner doesn't show "leave", to avoid misalignment with the server's last-owner check. */
const activeTenantNumericId = computed(() => Number(authStore.currentTenantId ?? 0))

const leaveMembersSnap = ref<TenantMember[]>([])
const leaveGateReady = ref(false)
const leaveGateLoading = ref(false)

const currentTenantRole = computed<TenantRole | ''>(() => (authStore.currentTenantRole || '') as TenantRole | '')

const canLeaveSpace = computed(() => {
  const r = currentTenantRole.value
  if (!r || !tenantInfo.value?.id) return false
  if (r !== 'owner') return true
  return leaveMembersSnap.value.filter((m) => m.role === 'owner').length > 1
})

/** Appears once the main content has loaded successfully, the `listMembers` gating rule is ready, and leaving is allowed. */
const showLeaveDangerZone = computed(() => {
  if (loading.value || error.value || !tenantInfo.value) return false
  if (!leaveGateReady.value || leaveGateLoading.value) return false
  if (!currentTenantRole.value) return false
  if (Number(tenantInfo.value.id) !== activeTenantNumericId.value) return false
  return canLeaveSpace.value
})

const showDeleteDangerZone = computed(() => {
  if (loading.value || error.value || !tenantInfo.value) return false
  if (Number(tenantInfo.value.id) !== activeTenantNumericId.value) return false
  return authStore.hasRole('owner')
})

async function evaluateLeaveGate(): Promise<void> {
  leaveGateReady.value = false
  leaveMembersSnap.value = []
  leaveGateLoading.value = false

  const infoId = tenantInfo.value?.id != null ? Number(tenantInfo.value.id) : 0
  if (!infoId || !activeTenantNumericId.value || infoId !== activeTenantNumericId.value) {
    leaveGateReady.value = true
    return
  }

  const role = currentTenantRole.value
  if (!role) {
    leaveGateReady.value = true
    return
  }
  if (role !== 'owner') {
    leaveGateReady.value = true
    return
  }

  leaveGateLoading.value = true
  try {
    leaveMembersSnap.value = await fetchAllTenantMembers(infoId)
  } finally {
    leaveGateLoading.value = false
    leaveGateReady.value = true
  }
}

function confirmLeaveTenant() {
  const tid = Number(tenantInfo.value?.id ?? 0)
  if (!tid) return

  const dlg = DialogPlugin.confirm({
    header: t('tenantMember.leave.confirmTitle'),
    body: t('tenantMember.leave.confirmBody'),
    confirmBtn: { content: t('tenantMember.leave.confirm'), theme: 'danger' },
    cancelBtn: t('common.cancel'),
    onConfirm: async () => {
      try {
        const resp = await leaveTenant(tid)
        if (resp.success) {
          MessagePlugin.success(t('tenantMember.leave.success'))
          authStore.logout()
          window.location.href = '/login'
        } else {
          MessagePlugin.error(resp.message || t('tenantMember.errors.generic'))
        }
      } catch (err: any) {
        const status = err?.status
        if (status === 409) {
          MessagePlugin.error(t('tenantMember.errors.lastOwner'))
        } else {
          MessagePlugin.error(err?.message || t('tenantMember.errors.generic'))
        }
      } finally {
        dlg.destroy()
      }
    },
    onClose: () => dlg.destroy(),
  })
}

function confirmDeleteTenant() {
  const tid = Number(tenantInfo.value?.id ?? 0)
  const tenantName = tenantInfo.value?.name || ''
  if (!tid || !tenantName) return
  deleteConfirmName.value = ''
  deleteTenantVisible.value = true
}

async function deleteCurrentTenant() {
  const tid = Number(tenantInfo.value?.id ?? 0)
  const tenantName = tenantInfo.value?.name || ''
  if (!tid || !tenantName) return
  if (deleteConfirmName.value.trim() !== tenantName) {
    MessagePlugin.warning(t('tenant.deleteDangerZone.nameMismatch'))
    return
  }
  try {
    deletingTenant.value = true
    const resp = await deleteTenantApi(tid)
    if (resp.success) {
      MessagePlugin.success(t('tenant.deleteDangerZone.success'))
      authStore.setMemberships(
        (authStore.memberships ?? []).filter((m) => m.tenant_id !== tid),
      )
      await authStore.refreshFromAuthMe()
      const next =
        authStore.memberships.find((m) => m.tenant_id === homeTenantId.value) ??
        authStore.memberships[0]
      if (next) {
        const switchingToHome =
          homeTenantId.value !== null && homeTenantId.value === next.tenant_id
        const name = next.tenant_name?.trim() || `#${next.tenant_id}`
        authStore.setSelectedTenant(next.tenant_id, name)
        stashTenantSwitchToast({
          name,
          role: formatRole(next.role) || undefined,
          roleEnum: next.role || undefined,
        })
        const persist = persistLastActiveTenantPreference(
          switchingToHome ? null : next.tenant_id,
        )
        await Promise.race([persist, new Promise((r) => setTimeout(r, 400))])
        navigateAfterTenantSwitch()
        return
      }
      authStore.logout()
      window.location.href = '/login'
    } else {
      MessagePlugin.error(resp.message || t('tenant.deleteDangerZone.failed'))
    }
  } catch (err: any) {
    MessagePlugin.error(err?.message || t('tenant.deleteDangerZone.failed'))
  } finally {
    deletingTenant.value = false
    deleteConfirmName.value = ''
    deleteTenantVisible.value = false
  }
}

watch(
  [() => tenantInfo.value?.id, () => authStore.currentTenantId, () => authStore.currentTenantRole],
  () => {
    if (!loading.value && tenantInfo.value && !error.value) {
      void evaluateLeaveGate()
    }
  },
)

// In-place editing of the space name: editing controls the toggle between inline read-only and edit states.
// Not using a dialog here since there's only one field — a popup would just interrupt the settings-browsing flow.
const editing = ref(false)
const editName = ref('')
const saving = ref(false)
const deleteConfirmName = ref('')
const deleteTenantVisible = ref(false)
const deletingTenant = ref(false)
const editNameTrimmed = computed(() => editName.value.trim())
// Condition for the save button to be clickable: non-empty, content changed, not currently saving.
// The backend's name field has neither a uniqueIndex nor duplicate-name validation, so no "already exists" check is done here;
// the backend service also only rejects empty on create, not on update — so it's enough for the frontend to just guard against empty.
const canSubmit = computed(
  () => !saving.value && !!editNameTrimmed.value && editNameTrimmed.value !== tenantInfo.value?.name,
)

const startEditName = () => {
  editName.value = tenantInfo.value?.name || ''
  editing.value = true
}

const cancelEditName = () => {
  if (saving.value) return
  editing.value = false
  editName.value = ''
}

// t-input doesn't bubble esc itself, so it's handled manually here (symmetric with the enter behavior).
const onEditKeydown = (_value: any, ctx: { e: KeyboardEvent }) => {
  if (ctx?.e?.key === 'Escape') {
    cancelEditName()
  }
}

// In-place editing of the space description: three states — editing / editValue / saving — symmetric with the name.
// Description can be empty (it's an optional field in the business logic), so the submit condition doesn't require non-empty, just that the content changed.
const editingDescription = ref(false)
const editDescription = ref('')
const savingDescription = ref(false)
const editDescriptionTrimmed = computed(() => editDescription.value.trim())
const canSubmitDescription = computed(
  () => !savingDescription.value && editDescriptionTrimmed.value !== (tenantInfo.value?.description || ''),
)

const startEditDescription = () => {
  editDescription.value = tenantInfo.value?.description || ''
  editingDescription.value = true
}

const cancelEditDescription = () => {
  if (savingDescription.value) return
  editingDescription.value = false
  editDescription.value = ''
}

// Enter in the textarea defaults to newline; submit is via Ctrl/⌘+Enter; Esc cancels.
const onEditDescriptionKeydown = (_value: any, ctx: { e: KeyboardEvent }) => {
  const e = ctx?.e
  if (!e) return
  if (e.key === 'Escape') {
    cancelEditDescription()
    return
  }
  if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
    e.preventDefault()
    void saveTenantDescription()
  }
}

const saveTenantDescription = async () => {
  if (!tenantInfo.value?.id) return
  const newDesc = editDescriptionTrimmed.value
  if (newDesc === (tenantInfo.value.description || '')) {
    editingDescription.value = false
    return
  }

  try {
    savingDescription.value = true
    const resp = await updateTenantApi(Number(tenantInfo.value.id), { description: newDesc })
    if (resp.success) {
      // Reflect it locally right away, instead of waiting on the /auth/me round trip. Unlike the name, the description doesn't appear in the space switcher etc.
      // in the top-level components, so there's no need to sync authStore.tenant / memberships.
      if (tenantInfo.value) {
        tenantInfo.value = { ...tenantInfo.value, description: newDesc }
      }
      MessagePlugin.success(t('tenant.details.editDescriptionSuccess'))
      editingDescription.value = false
    } else {
      MessagePlugin.error(resp.message || t('tenant.details.editDescriptionFailed'))
    }
  } catch (err: any) {
    MessagePlugin.error(err?.message || t('tenant.details.editDescriptionFailed'))
  } finally {
    savingDescription.value = false
  }
}

const saveTenantName = async () => {
  const newName = editNameTrimmed.value
  if (!newName) {
    MessagePlugin.warning(t('tenant.details.editNameRequired'))
    return
  }
  if (!tenantInfo.value?.id) return
  if (newName === tenantInfo.value.name) {
    editing.value = false
    return
  }

  try {
    saving.value = true
    const resp = await updateTenantApi(Number(tenantInfo.value.id), { name: newName })
    if (resp.success) {
      // Reflect it locally right away, instead of waiting on the /auth/me round trip; also refresh the tenant cache in the login state
      // (if the currently active space is the home tenant, the space switcher in the top bar etc. also gets updated).
      if (tenantInfo.value) {
        tenantInfo.value = { ...tenantInfo.value, name: newName }
      }
      if (authStore.tenant && String(authStore.tenant.id) === String(tenantInfo.value?.id)) {
        authStore.setTenant({ ...authStore.tenant, name: newName })
      }
      // tenant_name in memberships is the field the space switcher reads, so sync it too to avoid showing the old name.
      if (authStore.memberships?.length) {
        const next = authStore.memberships.map((m) =>
          String(m.tenant_id) === String(tenantInfo.value?.id)
            ? { ...m, tenant_name: newName }
            : m,
        )
        authStore.setMemberships(next)
      }
      MessagePlugin.success(t('tenant.details.editNameSuccess'))
      editing.value = false
    } else {
      MessagePlugin.error(resp.message || t('tenant.details.editNameFailed'))
    }
  } catch (err: any) {
    MessagePlugin.error(err?.message || t('tenant.details.editNameFailed'))
  } finally {
    saving.value = false
  }
}

// Methods
const loadInfo = async () => {
  try {
    loading.value = true
    error.value = ''

    const userResponse = await getCurrentUser()

    const data = userResponse?.data as { tenant?: TenantInfo } | undefined
    if ((userResponse as any).success && data?.tenant) {
      tenantInfo.value = data.tenant
    } else {
      error.value = userResponse.message || t('tenant.messages.fetchFailed')
    }
  } catch (err: any) {
    error.value = err?.message || t('tenant.messages.networkError')
  } finally {
    loading.value = false
  }
  // Must be evaluated after loading=false: otherwise the leave entry gets blocked by the loading condition in showLeaveDangerZone,
  // and in some environments the hydrated role arrives slightly later than the /auth/me response.
  if (tenantInfo.value && !error.value) {
    await evaluateLeaveGate()
  }
}

const getStatusText = (status: string | undefined) => {
  switch (status) {
    case 'active':
      return t('tenant.statusActive')
    case 'inactive':
      return t('tenant.statusInactive')
    case 'suspended':
      return t('tenant.statusSuspended')
    default:
      return t('tenant.statusUnknown')
  }
}

const getStatusTheme = (status: string | undefined) => {
  switch (status) {
    case 'active':
      return 'success'
    case 'inactive':
      return 'warning'
    case 'suspended':
      return 'danger'
    default:
      return 'default'
  }
}

const formatDate = (dateStr: string | undefined) => {
  if (!dateStr) return t('tenant.unknown')

  try {
    const date = new Date(dateStr)
    const formatter = new Intl.DateTimeFormat(locale.value || 'en-US', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit'
    })
    return formatter.format(date)
  } catch {
    return t('tenant.formatError')
  }
}

const formatBytes = (bytes: number) => {
  if (bytes === 0) return '0 B'

  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))

  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const getUsagePercentage = () => {
  if (!tenantInfo.value?.storage_quota || tenantInfo.value.storage_quota === 0) {
    return 0
  }

  const used = tenantInfo.value.storage_used || 0
  const percentage = (used / tenantInfo.value.storage_quota) * 100
  return Math.min(Math.round(percentage * 100) / 100, 100)
}

// Lifecycle
onMounted(() => {
  loadInfo()
})
</script>

<style lang="less" scoped>
.tenant-info {
  width: 100%;
}

.section-header {
  margin-bottom: 32px;

  h2 {
    font-size: 20px;
    font-weight: 600;
    color: var(--td-text-color-primary);
    margin: 0 0 8px 0;
  }

  .section-description {
    font-size: 14px;
    color: var(--td-text-color-secondary);
    margin: 0;
    line-height: 1.5;
  }
}

.loading-inline {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 40px 0;
  justify-content: center;
  color: var(--td-text-color-secondary);
  font-size: 14px;
}

.error-inline {
  padding: 20px 0;
}

.tenant-info-body {
  display: flex;
  flex-direction: column;
}

.settings-group {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.setting-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  padding: 20px 0;
  border-bottom: 1px solid var(--td-component-stroke);

  &:last-child {
    border-bottom: none;
  }
}

.setting-info {
  /* No longer flex:1: the label column is fixed to a reasonable max-content range (CJK labels are usually 4-6 characters,
     plus the desc text widening it), it doesn't participate in remaining-space distribution, avoiding long content squeezing it into vertical single-character wrapping.
     min-width as a fallback, so slightly longer desc text won't get compressed into one character per line either. */
  flex: 0 0 auto;
  width: max-content;
  min-width: 140px;
  max-width: 40%;
  padding-right: 24px;

  label {
    font-size: 15px;
    font-weight: 500;
    color: var(--td-text-color-primary);
    display: block;
    margin-bottom: 4px;
  }

  .desc {
    font-size: 13px;
    color: var(--td-text-color-secondary);
    margin: 0;
    line-height: 1.5;
  }
}

.setting-control {
  /* Conversely: the content column absorbs the remaining space, and is allowed to shrink + wrap internally, so long strings no longer blow out the row.
     Removed the original min-width:280px hard constraint (short content doesn't need that wide a display slot either). */
  flex: 1 1 auto;
  min-width: 0;
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 8px;

  .info-value {
    font-size: 14px;
    color: var(--td-text-color-primary);
    text-align: right;
    /* anywhere is more aggressive than break-word: even long strings with no spaces (like "WorkspaceDefault...")
       Also forces a line break so a single item doesn't stretch the whole row. */
    overflow-wrap: anywhere;
    min-width: 0;
  }

  .edit-btn {
    flex-shrink: 0;
  }
}

.inline-edit {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  justify-content: flex-end;
}

.inline-edit-input {
  /* In inline-edit mode the input can't fill the whole row, or the two right-side buttons would hug the edge;
     Just set a reasonable cap; anything beyond that falls back to t-input's own ellipsis. */
  max-width: 220px;
  flex: 1;
}

/* In-place editing of the description row: the textarea itself can wrap and expand, with the button moved below and right-aligned,
   Avoid squeezing the button narrow horizontally like the name row does. */
.inline-edit-description {
  flex-direction: column;
  align-items: stretch;
  gap: 8px;
  width: 100%;
  max-width: 360px;
}

.inline-edit-textarea {
  width: 100%;
}

.inline-edit-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

/* Read-only description: multi-line wrap allowed; an empty description shows placeholder-colored text hinting the user can click to edit.
.description-value {
  white-space: pre-wrap;
  word-break: break-word;

  &.is-empty {
    color: var(--td-text-color-placeholder);
  }
}

.leave-space-panel {
  margin-top: 4px;
}

.delete-space-panel {
  margin-top: 12px;
}

.leave-space-panel-inner {
  display: flex;
  flex-direction: row;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  padding: 16px 18px;
  border-radius: 10px;
  border: 1px solid var(--td-component-stroke);
  background-color: var(--td-bg-color-secondarycontainer);
  box-sizing: border-box;
}

.leave-space-panel-text {
  flex: 1;
  min-width: 0;
  max-width: min(65%, 28rem);
  padding-right: 8px;
}

.leave-space-panel-title {
  font-size: 15px;
  font-weight: 500;
  color: var(--td-text-color-primary);
  line-height: 1.4;
  margin-bottom: 4px;
}

.leave-space-panel-desc {
  margin: 0;
  font-size: 13px;
  line-height: 1.55;
  color: var(--td-text-color-secondary);
}

.leave-space-panel-action {
  flex-shrink: 0;
}

@media (max-width: 560px) {
  .leave-space-panel-inner {
    flex-direction: column;
    align-items: stretch;
  }

  .leave-space-panel-text {
    max-width: none;
    padding-right: 0;
  }

  .leave-space-panel-action {
    display: flex;
    justify-content: flex-end;
  }
}

.usage-control {
  //   width: 100%;
  //   display: flex;
  //   align-items: center;
  //   gap: 12px;

  .usage-text {
    font-size: 14px;
    font-weight: 500;
    color: var(--td-text-color-primary);
    min-width: 50px;
    text-align: right;
  }
}

.delete-tenant-confirm-body {
  margin: 0 0 10px;
  color: var(--td-text-color-primary);
  line-height: 1.6;
}

.delete-tenant-confirm-hint {
  margin: 0 0 12px;
  color: var(--td-text-color-secondary);
  line-height: 1.5;
}
</style>
