<template>
  <div class="mcp-settings">
    <div class="section-header">
      <h2>{{ $t('mcpSettings.title') }}</h2>
      <p class="section-description">
        {{ $t('mcpSettings.description') }}
      </p>
    </div>

    <div v-if="loading" class="loading-container">
      <t-loading :text="$t('common.loading')" />
    </div>

    <template v-else>
      <div class="list-section-header">
        <h3>{{ $t('mcpSettings.configuredServices') }}</h3>
        <p>{{ $t('mcpSettings.manageAndTest') }}</p>
      </div>

      <div v-if="services.length === 0 && !authStore.hasRole('admin')" class="empty-state">
        <t-empty :description="$t('mcpSettings.empty')" />
      </div>

      <div v-else class="services-grid">
        <!-- Card with the same shape as ModelSettings / WebSearchSettings: transport badge on the left +
             title / subtitle / url three-line layout. The switch sits on the right of the title row, the three-dot menu only appears on hover.
             SettingCard currently has no other consumers, but the component is kept for future reuse. -->
        <div
          v-for="service in services"
          :key="service.id"
          class="service-card"
          :class="[
            `service-card--${service.transport_type || 'unknown'}`,
            {
              'service-card--builtin': service.is_builtin,
              'service-card--clickable': isServiceCardClickable(),
            },
          ]"
          :role="isServiceCardClickable() ? 'button' : undefined"
          :tabindex="isServiceCardClickable() ? 0 : undefined"
          @click="onServiceCardClick($event, service)"
          @keydown.enter="onServiceCardClick($event, service)"
        >
          <div class="service-card__badge" :aria-label="getTransportTypeLabel(service.transport_type)">
            <t-icon :name="getTransportTypeIcon(service.transport_type)" size="18px" />
          </div>
          <div class="service-card__body">
            <div class="service-card__header">
              <h3 class="service-card__title" :title="service.name">{{ service.name }}</h3>
              <!-- Single status badge: builtin takes priority (always enabled, cannot be turned off), otherwise use enabled. -->
              <span
                v-if="service.is_builtin"
                class="service-card__pill service-card__pill--warning"
              >
                {{ $t('mcpSettings.builtin') }}
              </span>
              <span
                v-else
                class="service-card__status"
                :class="service.enabled ? 'service-card__status--on' : 'service-card__status--off'"
              >
                <span class="service-card__status-dot" />
                {{ service.enabled ? $t('common.on') : $t('common.off') }}
              </span>
              <div
                v-if="(service.is_builtin ? getBuiltinServiceOptions() : getServiceOptions(service)).length > 0"
                class="service-card__actions"
                @click.stop
              >
                <t-dropdown
                  :options="service.is_builtin ? getBuiltinServiceOptions() : getServiceOptions(service)"
                  placement="bottom-right"
                  attach="body"
                  trigger="click"
                  @click="(data: any) => handleMenuAction({ value: data.value }, service)"
                >
                  <t-button variant="text" shape="square" size="small" class="service-card__more">
                    <t-icon name="ellipsis" />
                  </t-button>
                </t-dropdown>
              </div>
            </div>
            <div class="service-card__subtitle">
              <span class="service-card__type">{{ getTransportTypeLabel(service.transport_type) }}</span>
              <template v-if="service.description">
                <span class="service-card__sep">·</span>
                <span class="service-card__desc" :title="service.description">{{ service.description }}</span>
              </template>
            </div>
            <div v-if="service.url" class="service-card__url" :title="service.url">
              {{ service.url }}
            </div>
          </div>
        </div>
        <button
          v-if="authStore.hasRole('admin')"
          type="button"
          class="service-card service-card--add"
          @click="handleAdd"
        >
          <span class="service-card--add__icon" aria-hidden="true">
            <add-icon />
          </span>
          <span class="service-card--add__label">{{ $t('mcpSettings.addService') }}</span>
        </button>
      </div>
    </template>

    <!-- Add/Edit Drawer -->
    <McpServiceDialog
      v-model:visible="dialogVisible"
      :service="currentService"
      :mode="dialogMode"
      @success="handleDialogSuccess"
      @created="handleDialogCreated"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import { AddIcon } from 'tdesign-icons-vue-next'
import { useI18n } from 'vue-i18n'
import {
  listMCPServices,
  updateMCPService,
  deleteMCPService,
  type MCPService
} from '@/api/mcp-service'
import McpServiceDialog from './components/McpServiceDialog.vue'
import { useConfirmDelete } from '@/components/settings/useConfirmDelete'
import { useAuthStore } from '@/stores/auth'

const { t } = useI18n()
const authStore = useAuthStore()
const confirmDelete = useConfirmDelete()

const services = ref<MCPService[]>([])
const loading = ref(false)
const dialogVisible = ref(false)
const dialogMode = ref<'add' | 'edit'>('add')
const currentService = ref<MCPService | null>(null)

// Load MCP services
const loadServices = async () => {
  loading.value = true
  try {
    services.value = await listMCPServices()
  } catch (error) {
    MessagePlugin.error(t('mcpSettings.toasts.loadFailed'))
    console.error('Failed to load MCP services:', error)
  } finally {
    loading.value = false
  }
}

// Handle add button click
const handleAdd = () => {
  currentService.value = null
  dialogMode.value = 'add'
  dialogVisible.value = true
}

const isServiceCardClickable = () => authStore.hasRole('admin')

const onServiceCardClick = (event: Event, service: MCPService) => {
  if (!isServiceCardClickable()) return
  if (event.type === 'keydown') {
    const ke = event as KeyboardEvent
    if (ke.key !== 'Enter' && ke.key !== ' ') return
    ke.preventDefault()
  }
  const target = event.target as HTMLElement | null
  if (target?.closest('.service-card__actions')) return
  handleEdit(service)
}

// Handle edit button click
const handleEdit = (service: MCPService) => {
  currentService.value = { ...service }
  dialogMode.value = 'edit'
  dialogVisible.value = true
}

// Handle dialog success (edit-mode update): close + refresh.
const handleDialogSuccess = () => {
  dialogVisible.value = false
  loadServices()
}

// Handle first create: keep the drawer open and flip it to edit mode bound to
// the newly created service, so OAuth authorization and "test connection"
// (both of which need a saved service id) are usable right away. The list is
// refreshed in the background; we prefer the freshly-fetched record so the
// edit form sees server-side fields (e.g. credential metadata).
const handleDialogCreated = async (created: MCPService) => {
  await loadServices()
  const full = services.value.find((s) => s.id === created.id) || created
  currentService.value = { ...full }
  dialogMode.value = 'edit'
}

// Handle toggle enabled/disabled
const handleToggleEnabled = async (service: MCPService) => {
  if (!service || !service.id) return

  const originalState = service.enabled
  try {
    await updateMCPService(service.id, { enabled: service.enabled })
    MessagePlugin.success(service.enabled ? t('mcpSettings.toasts.enabled') : t('mcpSettings.toasts.disabled'))
  } catch (error) {
    service.enabled = originalState
    MessagePlugin.error(t('mcpSettings.toasts.updateStateFailed'))
    console.error('Failed to update MCP service:', error)
  }
}

// Handle delete button click
const handleDelete = (service: MCPService) => {
  if (!service || !service.id) return

  confirmDelete({
    body: t('mcpSettings.deleteConfirmBody', { name: service.name || t('mcpSettings.unnamed') }),
    onConfirm: async () => {
      try {
        await deleteMCPService(service.id)
        MessagePlugin.success(t('mcpSettings.toasts.deleted'))
        loadServices()
      } catch (error) {
        MessagePlugin.error(t('mcpSettings.toasts.deleteFailed'))
        console.error('Failed to delete MCP service:', error)
      }
    }
  })
}

// Get service options for dropdown menu. MCP service mutations are all
// Admin+ in the backend matrix, so non-Admins see an empty action menu.
// Test connection has moved to the edit drawer's footer, no longer in the outer menu — a single entry point reduces
// user confusion ("why are there two test entries, do they give the same result?").
const getServiceOptions = (service: MCPService) => {
  if (!authStore.hasRole('admin')) {
    return []
  }
  return [
    {
      content: service.enabled ? t('common.off') : t('common.on'),
      value: 'toggle',
    },
    { content: t('common.edit'), value: 'edit' },
    { content: t('common.delete'), value: 'delete', theme: 'error' as const }
  ]
}

// Builtin: edit only (also Admin+ only). Built-in service testing is also triggered via the drawer's footer,
// no longer exposing a "test connection" item in the outer menu.
const getBuiltinServiceOptions = () => {
  if (!authStore.hasRole('admin')) {
    return []
  }
  return [
    { content: t('common.edit'), value: 'edit' }
  ]
}

// Handle menu action. 'test' has been removed from the menu — testing now
// lives only in the editor drawer. We keep the switch's case list narrow
// so a stray 'test' from somewhere else falls through harmlessly.
const handleMenuAction = (data: { value: string }, service: MCPService) => {
  switch (data.value) {
    case 'toggle':
      // Flip the local model and reuse the toggle path so the API call,
      // optimistic UI, and rollback-on-failure all stay in one place.
      service.enabled = !service.enabled
      handleToggleEnabled(service)
      break
    case 'edit':
      handleEdit(service)
      break
    case 'delete':
      handleDelete(service)
      break
  }
}

// Get transport type icon. Reuses tdesign's built-in icon names; add to this when a new transport is introduced.
const getTransportTypeIcon = (transportType: string) => {
  switch (transportType) {
    case 'sse':
      return 'cast'
    case 'http-streamable':
      return 'link'
    case 'stdio':
      return 'code'
    default:
      return 'tools'
  }
}

// Get transport type label
const getTransportTypeLabel = (transportType: string) => {
  switch (transportType) {
    case 'sse':
      return 'SSE'
    case 'http-streamable':
      return 'HTTP Streamable'
    case 'stdio':
      return 'Stdio'
    default:
      return transportType
  }
}

onMounted(() => {
  loadServices()
})
</script>

<style scoped lang="less">
.mcp-settings {
  width: 100%;
}

.section-header {
  margin-bottom: 28px;

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
    line-height: 1.6;
  }
}

.loading-container {
  padding: 40px 0;
  text-align: center;
}

.list-section-header {
  margin-bottom: 16px;

  h3 {
    font-size: 16px;
    font-weight: 600;
    color: var(--td-text-color-primary);
    margin: 0 0 4px 0;
  }

  p {
    font-size: 13px;
    color: var(--td-text-color-placeholder);
    margin: 0;
    line-height: 1.5;
  }
}

.empty-state {
  padding: 80px 0;
  text-align: center;

  :deep(.t-empty__description) {
    font-size: 14px;
    color: var(--td-text-color-placeholder);
    margin-bottom: 16px;
  }
}

.services-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 12px;

  .service-card--add {
    width: 100%;
    height: 100%;
  }
}

// Transport-distinguished card. Same shape as ModelSettings / WebSearchSettings.
.service-card {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 14px 14px 14px 12px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 10px;
  background: var(--td-bg-color-container);
  transition: border-color 0.18s ease, box-shadow 0.18s ease;
  min-width: 0;

  &--builtin {
    background: var(--td-bg-color-secondarycontainer);
  }

  &--clickable {
    cursor: pointer;

    &:hover {
      border-color: var(--td-brand-color-3, var(--td-brand-color));
      box-shadow: 0 4px 14px rgba(15, 23, 42, 0.06);
    }

    &:focus-visible {
      outline: 2px solid var(--td-brand-color);
      outline-offset: 2px;
    }
  }

  &--builtin:not(.service-card--clickable):hover {
    box-shadow: none;
    border-color: var(--td-component-stroke);
  }

  &--add {
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 8px;
    min-height: 68px;
    border-style: dashed;
    background: transparent;
    color: var(--td-text-color-placeholder);
    cursor: pointer;
    font: inherit;
    text-align: center;

    &:hover,
    &:focus-visible {
      color: var(--td-brand-color);
      border-color: var(--td-brand-color);
      background: color-mix(in srgb, var(--td-brand-color) 6%, transparent);
      box-shadow: none;
    }

    &:focus-visible {
      outline: 2px solid var(--td-brand-color);
      outline-offset: 2px;
    }

    &__icon {
      display: flex;
      align-items: center;
      justify-content: center;
      width: 32px;
      height: 32px;
      border-radius: 8px;
      background: color-mix(in srgb, var(--td-brand-color) 10%, transparent);
      color: var(--td-brand-color);
      font-size: 18px;
    }

    &__label {
      font-size: 13px;
      font-weight: 500;
      line-height: 1.4;
    }
  }
}

.service-card__actions {
  flex-shrink: 0;
}

.service-card__badge {
  flex-shrink: 0;
  width: 36px;
  height: 36px;
  border-radius: 9px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-top: 1px;
  background: rgba(0, 82, 217, 0.1);
  color: #0052D9;
}

// Badge colors for the three transports: sse streaming → green, http-streamable → blue, stdio → orange
.service-card--sse .service-card__badge {
  background: rgba(17, 128, 83, 0.12);
  color: #118053;
}
.service-card--http-streamable .service-card__badge {
  background: rgba(0, 82, 217, 0.1);
  color: #0052D9;
}
.service-card--stdio .service-card__badge {
  background: rgba(184, 92, 0, 0.12);
  color: #B85C00;
}

.service-card__body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.service-card__header {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}

.service-card__title {
  flex: 1;
  min-width: 0;
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  line-height: 1.4;
  color: var(--td-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.service-card__pill {
  flex-shrink: 0;
  padding: 1px 6px;
  font-size: 11px;
  font-weight: 500;
  line-height: 16px;
  border-radius: 3px;

  &--warning {
    color: var(--td-warning-color-7, #B85C00);
    background: var(--td-warning-color-1, #FEF3E6);
  }
}

// On/Off status badge — uses a dot + text instead of t-switch, to avoid accidental taps; toggling the enabled state is
// triggered by the toggle item in the three-dot menu, and the actual API call goes through the same handleToggleEnabled path.
.service-card__status {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 1px 8px 1px 6px;
  font-size: 11px;
  font-weight: 500;
  line-height: 16px;
  border-radius: 10px;
  background: var(--td-bg-color-secondarycontainer);

  &--on {
    color: var(--td-success-color-7, #118053);

    .service-card__status-dot {
      background: var(--td-success-color, #118053);
    }
  }

  &--off {
    color: var(--td-text-color-placeholder);

    .service-card__status-dot {
      background: var(--td-gray-color-5);
    }
  }
}

.service-card__status-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
}

.service-card__more {
  flex-shrink: 0;
  color: var(--td-text-color-placeholder);
  padding: 2px;
  opacity: 0;
  transition: opacity 0.15s ease;

  &:hover,
  &:focus-visible {
    background: var(--td-bg-color-secondarycontainer);
    color: var(--td-text-color-primary);
  }
}

// switch always visible (it's the state anchor); the three-dot button only appears on hover/focus.
.service-card:hover .service-card__more,
.service-card:focus-within .service-card__more,
.service-card__actions:focus-within .service-card__more {
  opacity: 1;
}

.service-card__subtitle {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 4px;
  font-size: 12px;
  line-height: 1.4;
  color: var(--td-text-color-secondary);
  min-width: 0;
}

.service-card__type {
  font-weight: 500;
}

.service-card__sep {
  color: var(--td-text-color-placeholder);
}

.service-card__desc {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}

.service-card__url {
  font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, Consolas, monospace;
  font-size: 11px;
  line-height: 1.4;
  color: var(--td-text-color-placeholder);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  min-width: 0;
}
</style>
