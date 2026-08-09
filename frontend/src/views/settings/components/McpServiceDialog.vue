<template>
  <SettingDrawer
    :visible="dialogVisible"
    :title="mode === 'add' ? t('mcpServiceDialog.addTitle') : t('mcpServiceDialog.editTitle')"
    :class="`mcp-drawer mcp-drawer--${formData.transport_type}`"
    :confirm-loading="submitting"
    @update:visible="(v: boolean) => dialogVisible = v"
    @confirm="handleSubmit"
    @cancel="handleClose"
  >
    <!--
      Header icon — same style as McpSettings list's .service-card__badge:
      transport_type determines the icon and container coloring. SSE is green, HTTP-Streamable is blue.
      The non-scoped block .mcp-drawer--{transport} injects background and text color; currentColor
      lets t-icon pick up the color too.
    -->
    <template #headerIcon>
      <t-icon :name="transportIcon" />
    </template>

    <!-- subtitle: transport type name + enabled-status mini chip -->
    <template #subtitle>
      <span>{{ transportLabel }}</span>
      <span
        class="subtitle-tag"
        :class="formData.enabled ? 'subtitle-tag--ok' : 'subtitle-tag--muted'"
      >
        {{ formData.enabled ? t('mcpSettings.enabled', 'Enabled') : t('mcpSettings.disabled', 'Disabled') }}
      </span>
    </template>

    <!--
      Test connection button moved to footer-left, same style as ModelEditorDialog/Storage/Parser/
      WebSearch drawer. Only works in edit mode (needs a service id to call the /test endpoint).
      In create mode the button is disabled with a hint to "save first to test".
    -->
    <template #footer-left>
      <t-button
        variant="outline"
        :loading="testing"
        :disabled="mode === 'add' || !props.service?.id"
        :title="mode === 'add' ? t('mcpServiceDialog.testAfterSaveHint', 'Save first to test the connection') : ''"
        @click="handleTestConnection"
      >
        <template #icon>
          <t-icon
            v-if="!testing && lastTestOk === true"
            name="check-circle-filled"
            class="status-icon available"
          />
          <t-icon
            v-else-if="!testing && lastTestOk === false"
            name="close-circle-filled"
            class="status-icon unavailable"
          />
        </template>
        {{ testing ? t('webSearchSettings.testing', 'Testing…') : t('mcpSettings.actions.test', 'Test connection') }}
      </t-button>
    </template>

    <t-form ref="formRef" :data="formData" :rules="rules" label-align="top">
      <!--
        Import from code: paste standard mcpServers JSON, parse it purely on the frontend and fill it back into the form.
        Does not auto-submit; the user reviews before clicking save.
      -->
      <section class="setting-drawer__section code-import">
        <button type="button" class="code-import__toggle" @click="codeImportOpen = !codeImportOpen">
          <t-icon :name="codeImportOpen ? 'chevron-down' : 'chevron-right'" />
          <span>{{ t('mcpServiceDialog.codeImport.toggle') }}</span>
        </button>
        <div v-if="codeImportOpen" class="code-import__body">
          <p class="form-desc">{{ t('mcpServiceDialog.codeImport.hint') }}</p>
          <p v-if="mode === 'edit'" class="form-desc code-import__warn">
            {{ t('mcpServiceDialog.codeImport.editOverwriteHint') }}
          </p>
          <t-textarea
            v-model="codeImportText"
            :autosize="{ minRows: 5, maxRows: 14 }"
            :placeholder="codeImportPlaceholder"
            class="code-import__textarea"
          />
          <p v-if="codeImportError" class="code-import__error">{{ codeImportError }}</p>
          <div class="code-import__actions">
            <t-button theme="primary" variant="outline" @click="handleCodeImport">
              {{ t('mcpServiceDialog.codeImport.parse') }}
            </t-button>
          </div>
        </div>
      </section>

      <!-- Section 1 — Basic Info -->
      <section class="setting-drawer__section">
        <h4 class="setting-drawer__section-title">{{ t('mcpServiceDialog.basicSection', 'Basic info') }}</h4>

        <div class="form-item">
          <label class="form-label required">{{ t('mcpServiceDialog.name') }}</label>
          <t-input v-model="formData.name" :placeholder="t('mcpServiceDialog.namePlaceholder')" />
        </div>

        <div class="form-item">
          <label class="form-label">{{ t('mcpServiceDialog.description') }}</label>
          <t-textarea
            v-model="formData.description"
            :autosize="{ minRows: 2, maxRows: 5 }"
            :placeholder="t('mcpServiceDialog.descriptionPlaceholder')"
          />
        </div>

        <div class="form-item">
          <label class="form-label">{{ t('mcpServiceDialog.enableService') }}</label>
          <div class="vision-toggle">
            <t-switch v-model="formData.enabled" />
            <span class="form-desc form-desc--inline">
              {{ t('mcpServiceDialog.enableServiceDesc', 'When disabled, this service will not be called') }}
            </span>
          </div>
        </div>
      </section>

      <!-- Section 2 — Connection Config (transport + url) -->
      <section class="setting-drawer__section">
        <h4 class="setting-drawer__section-title">{{ t('mcpServiceDialog.connectionSection', 'Connection settings') }}</h4>

        <div class="form-item">
          <label class="form-label required">{{ t('mcpServiceDialog.transportType') }}</label>
          <!-- Compact pill segmented, same style as ModelEditorDialog source switcher / Storage MinIO deployment mode -->
          <div class="source-options" role="radiogroup">
            <button
              type="button"
              class="source-option"
              :class="{ 'is-active': formData.transport_type === 'sse' }"
              @click="formData.transport_type = 'sse'"
            >
              <t-icon name="cast" class="source-option__icon" />
              <span class="source-option__label">SSE</span>
            </button>
            <button
              type="button"
              class="source-option"
              :class="{ 'is-active': formData.transport_type === 'http-streamable' }"
              @click="formData.transport_type = 'http-streamable'"
            >
              <t-icon name="link" class="source-option__icon" />
              <span class="source-option__label">HTTP Streamable</span>
            </button>
          </div>
        </div>

        <div class="form-item">
          <label class="form-label required">{{ t('mcpServiceDialog.serviceUrl') }}</label>
          <t-input v-model="formData.url" :placeholder="t('mcpServiceDialog.serviceUrlPlaceholder')" />
        </div>

        <!-- Custom request headers: appended to every MCP request's HTTP header (same interaction as model management) -->
        <div class="form-item">
          <div class="custom-headers-header">
            <label class="form-label" style="margin-bottom: 0">
              {{ t('mcpServiceDialog.customHeaders.label') }}
            </label>
            <t-button variant="text" size="small" theme="primary" @click="addCustomHeader">
              <template #icon><t-icon name="add" /></template>
              {{ t('mcpServiceDialog.customHeaders.add') }}
            </t-button>
          </div>
          <p class="form-desc custom-headers-desc">{{ t('mcpServiceDialog.customHeaders.desc') }}</p>
          <div v-if="formData.headers.length > 0" class="custom-headers-list">
            <div v-for="(item, idx) in formData.headers" :key="idx" class="custom-header-row">
              <t-input
                v-model="item.key"
                :placeholder="t('mcpServiceDialog.customHeaders.keyPlaceholder')"
                class="custom-header-key"
              />
              <t-input
                v-model="item.value"
                :placeholder="t('mcpServiceDialog.customHeaders.valuePlaceholder')"
                class="custom-header-value"
              />
              <t-button
                variant="text"
                shape="square"
                size="small"
                class="custom-header-remove"
                :aria-label="t('common.delete')"
                @click="removeCustomHeader(idx)"
              >
                <t-icon name="close" />
              </t-button>
            </div>
          </div>
        </div>
      </section>

      <!-- Section 3 — Auth Config (None / API Key / Bearer Token / OAuth) -->
      <section class="setting-drawer__section">
        <h4 class="setting-drawer__section-title">{{ t('mcpServiceDialog.authConfig') }}</h4>

        <div class="form-item">
          <label class="form-label">{{ t('mcpServiceDialog.authType', 'Authentication method') }}</label>
          <!-- Expandable pill segmented, same style as the transport type above, avoids re-opening a dropdown -->
          <div class="source-options" role="radiogroup">
            <button
              v-for="opt in authTypeOptions"
              :key="opt.value"
              type="button"
              class="source-option"
              :class="{ 'is-active': formData.auth_config.auth_type === opt.value }"
              @click="formData.auth_config.auth_type = opt.value as '' | 'api_key' | 'bearer' | 'oauth'"
            >
              <span class="source-option__label">{{ opt.label }}</span>
            </button>
          </div>
        </div>

        <!-- OAuth 2.0: zero-config (auto-discovery + dynamic client registration), authorized per user -->
        <template v-if="isOAuth">
          <div class="form-item">
            <label class="form-label">{{ t('mcpServiceDialog.oauthScopes', 'Scopes (optional, space-separated)') }}</label>
            <t-input v-model="oauthScopesText" :placeholder="t('mcpServiceDialog.optional')" />
          </div>
          <div class="form-item">
            <label class="form-label">{{ t('mcpServiceDialog.oauthAuthorization', 'Authorization status') }}</label>
            <div class="oauth-status">
              <t-tag v-if="oauthTokenState === 'authorized'" theme="success" variant="light">
                {{ t('mcpServiceDialog.oauthAuthorized', 'Authorized') }}
              </t-tag>
              <t-tag v-else-if="oauthTokenState === 'refreshable'" theme="primary" variant="light">
                {{ t('mcpServiceDialog.oauthRefreshable', 'Token expired; it will refresh automatically') }}
              </t-tag>
              <t-tag v-else theme="warning" variant="light">
                {{ t('mcpServiceDialog.oauthUnauthorized', 'Unauthorized') }}
              </t-tag>
              <t-button
                size="small"
                theme="primary"
                :loading="oauthAuthorizing || oauthChecking || submitting"
                @click="handleAuthorize"
              >
                {{ oauthTokenState === 'reauth_required' ? t('mcpServiceDialog.oauthAuthorize', 'Authorize') : t('mcpServiceDialog.oauthReauthorize', 'Re-authorize') }}
              </t-button>
              <t-button
                v-if="oauthTokenState !== 'reauth_required' && props.service?.id"
                size="small"
                theme="danger"
                variant="outline"
                @click="handleRevokeOAuth"
              >
                {{ t('mcpServiceDialog.oauthRevoke', 'Revoke authorization') }}
              </t-button>
            </div>
            <p class="form-desc">
              {{ t('mcpServiceDialog.oauthAuthorizeHint', 'Clicking Authorize saves the current config first, then starts authorization (per user).') }}
            </p>
          </div>
        </template>

        <!--
          Non-OAuth: in Edit mode credentials are managed by CredentialResource (a separate
          /credentials sub-resource call); in Create mode use a plain password input.
          Both fields are optional — the MCP service may not require auth at all.
        -->
        <!-- Credential header strategy: header name + secret value (other auth_types don't show the secret field) -->
        <template v-else-if="formData.auth_config.auth_type === 'api_key'">
          <!-- Header name (not secret): defaults to X-API-Key, fill Authorization for Bearer/bare-token scenarios. -->
          <div class="form-item">
            <label class="form-label">{{ t('mcpServiceDialog.apiKeyHeader', 'Header name') }}</label>
            <t-input
              v-model="formData.auth_config.api_key_header"
              placeholder="X-API-Key"
            />
            <p class="form-desc">{{ t('mcpServiceDialog.apiKeyHeaderDesc', 'Leave empty to use X-API-Key. For Bearer, set Authorization and put “Bearer <token>” in the secret; for a raw token, set Authorization and paste the token only.') }}</p>
          </div>

          <CredentialResource
            v-if="mode === 'edit' && props.service?.id"
            :api="credentialApi"
            :fields="credentialFields"
            :meta="credentialMeta"
          />
          <div v-else class="form-item">
            <label class="form-label">{{ t('mcpServiceDialog.credentialValue', 'Secret / Token') }}</label>
            <t-input
              v-model="formData.auth_config.api_key"
              type="password"
              :placeholder="t('mcpServiceDialog.optional')"
            >
              <template #prefix-icon><t-icon name="lock-on" /></template>
            </t-input>
          </div>
        </template>
      </section>

      <!-- Section 4 — Advanced Config (timeout/retry), switched to lightweight number inputs with unit suffixes,
           no longer using t-input-number's increment/decrement steppers (step buttons aren't needed here, users prefer typing directly). -->
      <section class="setting-drawer__section">
        <h4 class="setting-drawer__section-title">{{ t('mcpServiceDialog.advancedConfig') }}</h4>

        <div class="form-item">
          <label class="form-label">{{ t('mcpServiceDialog.timeoutSec') }}</label>
          <t-input
            v-model="advancedTimeoutText"
            type="number"
            :min="1"
            :max="300"
            placeholder="30"
            class="number-input"
            @blur="onAdvancedNumberBlur('timeout', 30, 1, 300)"
          >
            <template #suffix>
              <span class="number-input__unit">{{ t('mcpServiceDialog.unitSecond', 'sec') }}</span>
            </template>
          </t-input>
        </div>
        <div class="form-item">
          <label class="form-label">{{ t('mcpServiceDialog.retryCount') }}</label>
          <t-input
            v-model="advancedRetryCountText"
            type="number"
            :min="0"
            :max="10"
            placeholder="3"
            class="number-input"
            @blur="onAdvancedNumberBlur('retry_count', 3, 0, 10)"
          >
            <template #suffix>
              <span class="number-input__unit">{{ t('mcpServiceDialog.unitTimes', 'times') }}</span>
            </template>
          </t-input>
        </div>
        <div class="form-item">
          <label class="form-label">{{ t('mcpServiceDialog.retryDelaySec') }}</label>
          <t-input
            v-model="advancedRetryDelayText"
            type="number"
            :min="0"
            :max="60"
            placeholder="1"
            class="number-input"
            @blur="onAdvancedNumberBlur('retry_delay', 1, 0, 60)"
          >
            <template #suffix>
              <span class="number-input__unit">{{ t('mcpServiceDialog.unitSecond', 'sec') }}</span>
            </template>
          </t-input>
        </div>
      </section>

      <!-- Section 5 — Test Results (inline, avoids stacking another centered dialog on top of the drawer) -->
      <section v-if="testResult" ref="testResultSection" class="setting-drawer__section">
        <div class="test-result-header">
          <h4 class="setting-drawer__section-title">{{ t('mcpServiceDialog.testResultTitle', 'Test result') }}</h4>
          <t-button
            variant="text"
            theme="default"
            shape="square"
            size="small"
            class="test-result-close"
            @click="testResult = null"
          >
            <template #icon><t-icon name="close" /></template>
          </t-button>
        </div>
        <McpTestResultBody :result="testResult" :service-id="props.service?.id" />
      </section>
    </t-form>
  </SettingDrawer>
</template>

<script setup lang="ts">
import { ref, watch, computed, nextTick } from 'vue'
import { MessagePlugin } from 'tdesign-vue-next'
import type { FormInstanceFunctions, FormRule } from 'tdesign-vue-next'
import { useI18n } from 'vue-i18n'
import {
  createMCPService,
  updateMCPService,
  putMCPCredentials,
  deleteMCPCredentialField,
  testMCPService,
  getMCPOAuthAuthorizeURL,
  getMCPOAuthAuthorizationStatus,
  getMCPOAuthStatus,
  revokeMCPOAuthToken,
  MCP_OAUTH_CALLBACK_PATH,
  type MCPService,
  type McpCredentialField,
  type MCPTestResult,
  type MCPOAuthTokenState,
} from '@/api/mcp-service'
import SettingDrawer from '@/components/settings/SettingDrawer.vue'
import McpTestResultBody from './McpTestResultBody.vue'
import CredentialResource, {
  type CredentialFieldDef,
  type CredentialResourceApi,
} from '@/components/credentials/CredentialResource.vue'

interface Props {
  visible: boolean
  service: MCPService | null
  mode: 'add' | 'edit'
}

interface Emits {
  (e: 'update:visible', value: boolean): void
  (e: 'success'): void
  // Emitted after a successful create; carries the newly created service so
  // the parent can transition the drawer to edit mode without closing it.
  (e: 'created', service: MCPService): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const formRef = ref<FormInstanceFunctions>()
const submitting = ref(false)
const { t } = useI18n()
const codeImportPlaceholder = `{
  "mcpServers": {
    "my-server": {
      "url": "https://example.com/sse"
    }
  }
}`

const formData = ref({
  name: '',
  description: '',
  enabled: true,
  transport_type: 'sse' as 'sse' | 'http-streamable',
  url: '',
  // Custom HTTP headers attached to every MCP request — edited as key/value
  // rows here, serialised to a Record<string,string> on submit (top-level
  // `headers`). Independent of the auth strategy.
  headers: [] as { key: string; value: string }[],
  auth_config: {
    // Authentication strategy. UI exposes '' (none) | 'api_key' (credential
    // header) | 'oauth'. Legacy 'bearer' is normalised to 'api_key' on load.
    auth_type: '' as '' | 'api_key' | 'bearer' | 'oauth',
    // Only used in add-mode; in edit-mode the CredentialResource owns these.
    api_key: '',
    // Non-secret: header name for the api_key value (default X-API-Key).
    api_key_header: '',
    token: '',
    // OAuth-only, non-secret config.
    scopes: [] as string[],
    auth_server_metadata_url: '',
  },
  advanced_config: {
    timeout: 30,
    retry_count: 3,
    retry_delay: 1,
  },
})

// ---- Custom headers (key/value editor, same UX as ModelEditorDialog) ----
const addCustomHeader = () => {
  formData.value.headers.push({ key: '', value: '' })
}
const removeCustomHeader = (idx: number) => {
  formData.value.headers.splice(idx, 1)
}

// ---- Code import (paste standard mcpServers JSON) ----
const codeImportOpen = ref(false)
const codeImportText = ref('')
const codeImportError = ref('')

// Map a raw MCP server config object onto the form. Stdio is rejected.
function applyServerConfig(name: string, cfg: Record<string, unknown>) {
  if (cfg.command) {
    codeImportError.value = t('mcpServiceDialog.codeImport.errors.stdioUnsupported')
    return false
  }
  const url = typeof cfg.url === 'string' ? cfg.url.trim() : ''
  if (!url) {
    codeImportError.value = t('mcpServiceDialog.codeImport.errors.missingUrl')
    return false
  }

  // Transport: explicit type/transport wins, else guess from the URL shape.
  const rawType = String(cfg.type ?? cfg.transport ?? '').toLowerCase()
  let transport: 'sse' | 'http-streamable'
  if (rawType === 'sse') {
    transport = 'sse'
  } else if (['http', 'http-streamable', 'streamable-http', 'streamablehttp', 'streamable_http'].includes(rawType)) {
    transport = 'http-streamable'
  } else {
    transport = /\/sse\/?($|\?)/i.test(url) ? 'sse' : 'http-streamable'
  }

  // Auth: recognise Bearer / API-key headers; every other header lands in the
  // custom-headers editor so nothing is silently dropped.
  // Map a recognised auth header onto the unified credential strategy: the raw
  // header value is the secret (encrypted), carried in the given header name.
  // Bearer keeps its "Bearer " prefix inside the value; Authorization with a
  // raw token and X-API-Key all collapse to the same shape.
  let authType: '' | 'api_key' = ''
  let apiKey = ''
  let apiKeyHeader = ''
  const customHeaders: { key: string; value: string }[] = []
  const headers = (cfg.headers && typeof cfg.headers === 'object')
    ? (cfg.headers as Record<string, unknown>)
    : {}
  for (const [key, val] of Object.entries(headers)) {
    const lowerKey = key.toLowerCase()
    const strVal = typeof val === 'string' ? val : String(val ?? '')
    if (lowerKey === 'authorization' || ['x-api-key', 'api-key', 'apikey'].includes(lowerKey)) {
      authType = 'api_key'
      apiKey = strVal.trim()
      apiKeyHeader = lowerKey === 'x-api-key' ? '' : key
    } else {
      customHeaders.push({ key, value: strVal })
    }
  }

  formData.value.name = name || formData.value.name
  formData.value.url = url
  formData.value.transport_type = transport
  if (typeof cfg.description === 'string') formData.value.description = cfg.description
  formData.value.headers = customHeaders
  // No recognised auth header → none (custom headers carry the rest).
  formData.value.auth_config.auth_type = authType
  formData.value.auth_config.token = ''
  formData.value.auth_config.api_key = apiKey
  formData.value.auth_config.api_key_header = apiKeyHeader
  formData.value.auth_config.scopes = []

  return true
}

const handleCodeImport = () => {
  codeImportError.value = ''
  const raw = codeImportText.value.trim()
  if (!raw) {
    codeImportError.value = t('mcpServiceDialog.codeImport.errors.empty')
    return
  }

  let parsed: unknown
  try {
    parsed = JSON.parse(raw)
  } catch {
    codeImportError.value = t('mcpServiceDialog.codeImport.errors.invalidJson')
    return
  }
  if (!parsed || typeof parsed !== 'object') {
    codeImportError.value = t('mcpServiceDialog.codeImport.errors.noServer')
    return
  }

  const obj = parsed as Record<string, unknown>
  let name = ''
  let cfg: Record<string, unknown> | null = null
  let multiple = false

  const servers = obj.mcpServers
  if (servers && typeof servers === 'object') {
    const entries = Object.entries(servers as Record<string, unknown>)
    if (entries.length === 0) {
      codeImportError.value = t('mcpServiceDialog.codeImport.errors.noServer')
      return
    }
    multiple = entries.length > 1
    name = entries[0][0]
    cfg = entries[0][1] as Record<string, unknown>
  } else if (obj.url || obj.command) {
    // Bare single-server object (no mcpServers wrapper).
    cfg = obj
  } else {
    codeImportError.value = t('mcpServiceDialog.codeImport.errors.noServer')
    return
  }

  if (!cfg || typeof cfg !== 'object') {
    codeImportError.value = t('mcpServiceDialog.codeImport.errors.noServer')
    return
  }

  if (!applyServerConfig(name, cfg)) return

  formRef.value?.clearValidate()
  if (multiple) {
    MessagePlugin.info(
      t('mcpServiceDialog.codeImport.toasts.multipleServers', { name }) as string,
    )
  } else {
    MessagePlugin.success(t('mcpServiceDialog.codeImport.toasts.filled') as string)
  }
}

// Comma/space separated text binding for OAuth scopes.
const oauthScopesText = computed({
  get: () => (formData.value.auth_config.scopes || []).join(' '),
  set: (val: string) => {
    formData.value.auth_config.scopes = val
      .split(/[\s,]+/)
      .map((s) => s.trim())
      .filter(Boolean)
  },
})

const isOAuth = computed(() => formData.value.auth_config.auth_type === 'oauth')

// API Key / Bearer are unified into one "credential header" strategy now that
// the header name is configurable (Bearer is just Authorization + a "Bearer "
// value prefix). "None" stays as the default for services that need no auth or
// only the custom headers configured above.
const authTypeOptions = computed(() => [
  { value: '', label: t('mcpServiceDialog.authTypeNone', 'None / custom header') },
  { value: 'api_key', label: t('mcpServiceDialog.authTypeApiKey', 'API Key / Token') },
  { value: 'oauth', label: t('mcpServiceDialog.authTypeOAuth', 'OAuth 2.0 (authorize on first connect)') },
])

// ---- OAuth authorization state (edit mode only) ----
const oauthAuthorized = ref(false)
const oauthTokenState = ref<MCPOAuthTokenState>('reauth_required')
const oauthChecking = ref(false)
const oauthAuthorizing = ref(false)

async function refreshOAuthStatus() {
  if (props.mode !== 'edit' || !props.service?.id || !isOAuth.value) return
  oauthChecking.value = true
  try {
    const status = await getMCPOAuthAuthorizationStatus(props.service.id)
    oauthAuthorized.value = status.authorized
    oauthTokenState.value = status.state
  } catch (e) {
    console.error('Failed to query MCP OAuth status:', e)
  } finally {
    oauthChecking.value = false
  }
}

// Open the OAuth popup for an already-persisted service and poll for
// completion against that service id (independent of props, which may be
// re-bound by the parent after a save).
async function startAuthorize(serviceId: string) {
  if (!serviceId) return
  oauthAuthorizing.value = true
  try {
    const redirectUri = window.location.origin + MCP_OAUTH_CALLBACK_PATH
    // After the backend completes the exchange it bounces the popup here. The
    // app root is harmless; the popup is closed by the opener below once the
    // authorization status flips, so this page is only shown briefly.
    const frontendRedirect = window.location.origin + '/'
    const authorization = await getMCPOAuthAuthorizeURL(serviceId, {
      redirect_uri: redirectUri,
      frontend_redirect: frontendRedirect,
    })
    if (!authorization.authorizationUrl || !authorization.authorizationAttempt) {
      MessagePlugin.error(t('mcpServiceDialog.toasts.authorizeFailed', 'Failed to start authorization') as string)
      oauthAuthorizing.value = false
      return
    }
    const popup = window.open(authorization.authorizationUrl, 'mcp_oauth', 'width=600,height=720')
    // Poll for completion: either the popup closes or the status flips.
    const timer = window.setInterval(async () => {
      const closed = !popup || popup.closed
      try {
        oauthAuthorized.value = await getMCPOAuthStatus(serviceId, authorization.authorizationAttempt)
        if (oauthAuthorized.value) oauthTokenState.value = 'authorized'
      } catch { /* keep polling */ }
      if (oauthAuthorized.value || closed) {
        window.clearInterval(timer)
        oauthAuthorizing.value = false
        if (oauthAuthorized.value) {
          try { popup?.close() } catch { /* cross-origin close may throw */ }
          MessagePlugin.success(t('mcpServiceDialog.toasts.authorized', 'Authorization succeeded') as string)
        }
      }
    }, 1500)
  } catch (e) {
    console.error('Failed to start MCP OAuth authorization:', e)
    MessagePlugin.error(t('mcpServiceDialog.toasts.authorizeFailed', 'Failed to start authorization') as string)
    oauthAuthorizing.value = false
  }
}

// Authorize ALWAYS persists the current form first (create or update), so the
// backend sees auth_type=oauth before issuing the authorization request.
// Without this, a brand-new service or one just switched to OAuth fails with
// "MCP service is not configured to use OAuth". The drawer stays open the whole
// time — the parent re-binds the (now saved) service via the `created` event.
async function handleAuthorize() {
  const valid = await formRef.value?.validate()
  if (!valid) return

  submitting.value = true
  let serviceId = ''
  try {
    const hasId = !!props.service?.id
    const data = buildPayload(!hasId)
    if (hasId) {
      await updateMCPService(props.service!.id, data)
      serviceId = props.service!.id
      emit('created', { ...(props.service as MCPService), ...data } as MCPService)
    } else {
      const created = await createMCPService(data)
      serviceId = created.id
      emit('created', created)
    }
  } catch (e) {
    MessagePlugin.error(t('mcpServiceDialog.toasts.updateFailed') as string)
    console.error('Failed to save before authorize:', e)
    submitting.value = false
    return
  }
  submitting.value = false
  await startAuthorize(serviceId)
}

async function handleRevokeOAuth() {
  if (!props.service?.id) return
  try {
    await revokeMCPOAuthToken(props.service.id)
    oauthAuthorized.value = false
    oauthTokenState.value = 'reauth_required'
    MessagePlugin.success(t('mcpServiceDialog.toasts.revoked', 'Authorization revoked') as string)
  } catch (e) {
    console.error('Failed to revoke MCP OAuth token:', e)
    MessagePlugin.error(t('mcpServiceDialog.toasts.revokeFailed', 'Revoke failed') as string)
  }
}

// Header icon name + transport label, mirrored from McpSettings list cards
// so the list-card → drawer hand-off stays visually continuous.
const transportIcon = computed(() => {
  return formData.value.transport_type === 'http-streamable' ? 'link' : 'cast'
})

const transportLabel = computed(() => {
  return formData.value.transport_type === 'http-streamable' ? 'HTTP Streamable' : 'SSE'
})

// Field metadata for the credential subresource. Keep label keys local to
// MCP so other resources don't accidentally inherit "API Key" / "Bearer
// Token" labels via the shared component.
// The unified credential strategy stores its secret in the api_key field
// (placed verbatim into the configured header). None / OAuth carry no static
// secret, so the credential card is only shown for the api_key strategy.
const credentialFields = computed<CredentialFieldDef<McpCredentialField>[]>(() => {
  if (formData.value.auth_config.auth_type === 'api_key') {
    return [{ key: 'api_key', label: t('mcpServiceDialog.credentialValue', 'Secret / Token') }]
  }
  return []
})

// Adapter that binds the generic CredentialResource component to the MCP
// credential endpoints. Recomputed if the user opens a different service.
const credentialApi = computed<CredentialResourceApi<McpCredentialField>>(() => {
  const id = props.service?.id ?? ''
  return {
    save: async (patch) => {
      const meta = await putMCPCredentials(id, patch)
      return meta.fields
    },
    remove: async (field) => {
      await deleteMCPCredentialField(id, field)
    },
  }
})

// Initial "configured?" metadata read from the main service response. The
// component reads this on mount; subsequent state changes after save/remove
// are tracked locally by the component itself (and re-derived from this
// whenever the parent reloads the service).
const credentialMeta = computed(() => props.service?.credentials ?? {
  api_key: { configured: false },
  token: { configured: false },
})

const rules: Record<string, FormRule[]> = {
  name: [{ required: true, message: t('mcpServiceDialog.rules.nameRequired') as string, type: 'error' }],
  transport_type: [{ required: true, message: t('mcpServiceDialog.rules.transportRequired') as string, type: 'error' }],
  url: [
    {
      validator: (val: string) => {
        if (!val || val.trim() === '') {
          return { result: false, message: t('mcpServiceDialog.rules.urlRequired') as string, type: 'error' }
        }
        try {
          new URL(val)
          return { result: true, message: '', type: 'success' }
        } catch {
          return { result: false, message: t('mcpServiceDialog.rules.urlInvalid') as string, type: 'error' }
        }
      },
    },
  ],
}

const dialogVisible = computed({
  get: () => props.visible,
  set: (value) => emit('update:visible', value),
})

// ---- Test connection state (in-drawer) ----
const testing = ref(false)
// Tri-state icon hint on the test button: null=neutral, true=just succeeded,
// false=just failed. Cleared when transport/url change so a stale ✓/✗
// doesn't sit next to a config the user is now editing.
const lastTestOk = ref<boolean | null>(null)
// In-drawer test result, rendered inline (no centered dialog stacked on the
// drawer). Cleared when the target config changes so a stale result doesn't
// sit next to edited config.
const testResult = ref<MCPTestResult | null>(null)
const testResultSection = ref<HTMLElement | null>(null)

// Results area sits at the very bottom of the drawer, auto-scrolls into view once the test finishes, so users don't think nothing happened.
function scrollToTestResult() {
  void nextTick(() => {
    testResultSection.value?.scrollIntoView({ behavior: 'smooth', block: 'start' })
  })
}

watch(
  () => [formData.value.transport_type, formData.value.url],
  () => {
    lastTestOk.value = null
    testResult.value = null
  },
)

async function handleTestConnection() {
  if (!props.service?.id) return
  testing.value = true
  MessagePlugin.info({
    content: t('mcpSettings.toasts.testing', { name: props.service.name || '' }),
    duration: 0,
    closeBtn: false,
  })
  try {
    const result = await testMCPService(props.service.id)
    MessagePlugin.closeAll()
    const safe: MCPTestResult = result ?? {
      success: false,
      message: t('mcpSettings.toasts.noResponse') as string,
    }
    lastTestOk.value = safe.success === true
    testResult.value = safe
    const discoveredDescription = typeof safe.description === 'string' ? safe.description.trim() : ''
    if (safe.success === true && discoveredDescription) {
      formData.value.description = discoveredDescription
    }
    // Server told us it needs OAuth (RFC 9728): guide the user by switching the
    // auth strategy to OAuth. We intentionally do NOT prefill
    // auth_server_metadata_url — the discovered URL is the RFC 9728
    // protected-resource metadata (not the RFC 8414 authorization-server
    // metadata), and the client library discovers the right endpoints from the
    // base URL on its own.
    if (safe.oauth_required === true && formData.value.auth_config.auth_type !== 'oauth') {
      formData.value.auth_config.auth_type = 'oauth'
      MessagePlugin.warning(
        t('mcpServiceDialog.toasts.oauthRequired', 'This service requires OAuth. Switched to OAuth 2.0 automatically; save, then click Authorize.') as string,
      )
    }
    scrollToTestResult()
  } catch (error: any) {
    MessagePlugin.closeAll()
    const errorMessage =
      error?.response?.data?.error?.message ||
      error?.message ||
      (t('mcpSettings.toasts.testFailed') as string)
    console.error('Failed to test MCP service:', error)
    lastTestOk.value = false
    testResult.value = { success: false, message: errorMessage }
    scrollToTestResult()
  } finally {
    testing.value = false
  }
}

// ---- Advanced numeric inputs (text-bound proxies) ----
// We bind text instead of v-model directly to advanced_config.<n> so the
// user can clear the field and see the placeholder while typing. On blur
// we coerce, clamp, and write back; bad values fall back to the default.
const advancedTimeoutText = computed<string>({
  get: () => String(formData.value.advanced_config.timeout ?? ''),
  set: (v) => { formData.value.advanced_config.timeout = parseSloppyInt(v) ?? 30 },
})
const advancedRetryCountText = computed<string>({
  get: () => String(formData.value.advanced_config.retry_count ?? ''),
  set: (v) => { formData.value.advanced_config.retry_count = parseSloppyInt(v) ?? 3 },
})
const advancedRetryDelayText = computed<string>({
  get: () => String(formData.value.advanced_config.retry_delay ?? ''),
  set: (v) => { formData.value.advanced_config.retry_delay = parseSloppyInt(v) ?? 1 },
})

// Permissive int parser — keeps '' / NaN inputs as null instead of 0 so the
// field can stay visually empty while the user is still typing. Negative
// numbers and non-int chars are rejected (returns null).
function parseSloppyInt(raw: string): number | null {
  if (raw == null) return null
  const s = String(raw).trim()
  if (!s) return null
  const n = Number(s)
  if (!Number.isFinite(n)) return null
  return Math.trunc(n)
}

function onAdvancedNumberBlur(
  field: 'timeout' | 'retry_count' | 'retry_delay',
  fallback: number,
  min: number,
  max: number,
) {
  const cur = formData.value.advanced_config[field]
  if (cur == null || !Number.isFinite(cur)) {
    formData.value.advanced_config[field] = fallback
    return
  }
  // Clamp to [min, max] on blur — gives the input "settled" feedback even
  // though native type=number doesn't enforce its own min/max attribute
  // for typed values (only for stepper buttons).
  formData.value.advanced_config[field] = Math.min(max, Math.max(min, cur))
}

const resetForm = () => {
  formData.value = {
    name: '',
    description: '',
    enabled: true,
    transport_type: 'sse',
    url: '',
    headers: [],
    auth_config: { auth_type: '', api_key: '', api_key_header: '', token: '', scopes: [], auth_server_metadata_url: '' },
    advanced_config: { timeout: 30, retry_count: 3, retry_delay: 1 },
  }
  formRef.value?.clearValidate()
}

watch(
  () => props.service,
  (service) => {
    // Clear the previous test feedback when switching to a different service (or adding a new one), so stale ✓/✗ doesn't linger on the new form
    lastTestOk.value = null
    testResult.value = null
    // Also reset the code import area, so pasted content/errors left over from the previous service don't linger on the new form
    codeImportOpen.value = false
    codeImportText.value = ''
    codeImportError.value = ''
    if (service) {
      const transportType = service.transport_type === 'stdio' ? 'sse' : (service.transport_type || 'sse')
      formData.value = {
        name: service.name || '',
        description: service.description || '',
        enabled: service.enabled ?? true,
        transport_type: transportType as 'sse' | 'http-streamable',
        url: service.url || '',
        headers: service.headers
          ? Object.entries(service.headers).map(([key, value]) => ({ key, value: String(value) }))
          : [],
        // Credentials are owned by CredentialResource in edit mode, but reset
        // the local state too so a switch to add-mode starts clean.
        auth_config: {
          // Legacy 'bearer' collapses into the unified 'api_key' strategy; ''
          // (none) and the rest are preserved.
          auth_type: service.auth_config?.auth_type === 'bearer'
            ? 'api_key'
            : ((service.auth_config?.auth_type as '' | 'api_key' | 'oauth') || ''),
          api_key: '',
          api_key_header: service.auth_config?.api_key_header || '',
          token: '',
          scopes: service.auth_config?.scopes ? [...service.auth_config.scopes] : [],
          auth_server_metadata_url: service.auth_config?.auth_server_metadata_url || '',
        },
        advanced_config: {
          timeout: service.advanced_config?.timeout || 30,
          retry_count: service.advanced_config?.retry_count || 3,
          retry_delay: service.advanced_config?.retry_delay || 1,
        },
      }
      oauthAuthorized.value = false
      oauthTokenState.value = 'reauth_required'
      refreshOAuthStatus()
    } else {
      resetForm()
    }
  },
  { immediate: true },
)

// Build the request body from the form. `asCreate` controls whether initial
// secret credentials ride along (POST only — edits route secrets through the
// /credentials subresource).
function buildPayload(asCreate: boolean): Partial<MCPService> {
  // Custom headers: trim, drop blank rows, collapse to a Record. Always send
  // the field (even when empty) so removing the last header persists on edit.
  const headersMap: Record<string, string> = {}
  for (const item of formData.value.headers) {
    const key = (item.key ?? '').trim()
    const value = (item.value ?? '').trim()
    if (key && value) headersMap[key] = value
  }

  const data: Partial<MCPService> = {
    name: formData.value.name,
    description: formData.value.description,
    enabled: formData.value.enabled,
    transport_type: formData.value.transport_type,
    advanced_config: formData.value.advanced_config,
    url: formData.value.url || undefined,
    headers: headersMap,
  }

  // Non-secret auth config (strategy + OAuth params) flows through the main body.
  const auth: NonNullable<MCPService['auth_config']> = {
    auth_type: formData.value.auth_config.auth_type,
  }
  if (formData.value.auth_config.auth_type === 'api_key' && formData.value.auth_config.api_key_header) {
    auth.api_key_header = formData.value.auth_config.api_key_header.trim()
  }
  if (isOAuth.value) {
    auth.scopes = formData.value.auth_config.scopes
    if (formData.value.auth_config.auth_server_metadata_url) {
      auth.auth_server_metadata_url = formData.value.auth_config.auth_server_metadata_url
    }
  }
  if (asCreate && !isOAuth.value) {
    if (formData.value.auth_config.api_key) auth.api_key = formData.value.auth_config.api_key
    if (formData.value.auth_config.token) auth.token = formData.value.auth_config.token
  }
  data.auth_config = auth
  return data
}

const handleSubmit = async () => {
  const valid = await formRef.value?.validate()
  if (!valid) return

  submitting.value = true
  try {
    const data = buildPayload(props.mode === 'add')
    if (props.mode === 'add') {
      const created = await createMCPService(data)
      MessagePlugin.success(t('mcpServiceDialog.toasts.created'))
      // Keep the drawer open and hand back the new service so the parent can
      // flip it into edit mode in place — OAuth authorization and "test
      // connection" both need a saved service id, so transitioning here lets
      // the user do them immediately instead of save → reopen.
      emit('created', created)
    } else {
      await updateMCPService(props.service!.id, data)
      MessagePlugin.success(t('mcpServiceDialog.toasts.updated'))
      emit('success')
    }
  } catch (error) {
    MessagePlugin.error(
      props.mode === 'add'
        ? (t('mcpServiceDialog.toasts.createFailed') as string)
        : (t('mcpServiceDialog.toasts.updateFailed') as string),
    )
    console.error('Failed to save MCP service:', error)
  } finally {
    submitting.value = false
  }
}

const handleClose = () => {
  dialogVisible.value = false
}
</script>

<style scoped lang="less">
// ---- Drawer content — same convention as ModelEditorDialog ----
.form-item {
  margin-bottom: 0;
}

// ---- Custom request headers (same key/value rows as ModelEditorDialog) ----
.custom-headers-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.custom-headers-desc {
  margin: 4px 0 8px;
}

.custom-headers-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.custom-header-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.custom-header-key {
  flex: 0 0 38%;
}

.custom-header-value {
  flex: 1;
  min-width: 0;
}

.custom-header-remove {
  flex-shrink: 0;
  color: var(--td-text-color-placeholder);

  &:hover,
  &:focus-visible {
    color: var(--td-error-color);
  }
}

// ---- Import from code ----
.code-import {
  &__toggle {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    padding: 0;
    border: none;
    background: none;
    cursor: pointer;
    font-size: 14px;
    font-weight: 500;
    color: var(--td-text-color-primary);

    &:hover {
      color: var(--td-brand-color);
    }
  }

  &__body {
    margin-top: 12px;
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  &__warn {
    color: var(--td-warning-color);
  }

  &__textarea :deep(textarea) {
    font-family: var(--td-font-family-mono, monospace);
    font-size: 12px;
  }

  &__error {
    margin: 0;
    font-size: 12px;
    color: var(--td-error-color);
  }

  &__actions {
    display: flex;
    justify-content: flex-end;
  }
}

.oauth-status {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.test-result-header {
  display: flex;
  align-items: center;
  justify-content: space-between;

  .setting-drawer__section-title {
    margin-bottom: 0;
  }

  .test-result-close {
    flex-shrink: 0;
    color: var(--td-text-color-placeholder);
  }
}

.oauth-hint {
  margin: 0;
  font-size: 12px;
  color: var(--td-text-color-secondary);
  line-height: 1.5;
}

.form-label {
  display: block;
  margin-bottom: 6px;
  font-size: 13px;
  font-weight: 500;
  color: var(--td-text-color-primary);
  line-height: 1.4;

  &.required::before {
    content: '*';
    color: var(--td-error-color);
    margin-right: 4px;
    font-weight: 500;
    line-height: 1;
  }
}

.form-desc {
  margin: 4px 0 0 0;
  font-size: 12px;
  line-height: 1.5;
  color: var(--td-text-color-placeholder);

  &--inline {
    margin: 0;
  }
}

:deep(.t-input),
:deep(.t-select),
:deep(.t-textarea),
:deep(.t-input-number) {
  width: 100%;
  font-size: 13px;
}

// Hide t-form's default form-item container — use custom .form-item / .form-label instead
:deep(.t-form) .t-form-item {
  display: none;
}

// ---- Compact pill segmented (transport switch) ----
.source-options {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px;
  background: var(--td-bg-color-component);
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
}

.source-option {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 5px 12px;
  height: 28px;
  background: transparent;
  border: 1px solid transparent;
  border-radius: 6px;
  cursor: pointer;
  font-family: inherit;
  font-size: 13px;
  color: var(--td-text-color-secondary);
  line-height: 1;
  transition: all 0.15s ease;

  &:hover:not(.is-active) {
    color: var(--td-text-color-primary);
    background: var(--td-bg-color-container-hover);
  }

  &.is-active {
    background: var(--td-bg-color-container);
    border-color: var(--td-brand-color);
    color: var(--td-brand-color);
    font-weight: 500;
    box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04);
  }
}

.source-option__icon {
  font-size: 14px;
  flex-shrink: 0;
}

.source-option__label {
  white-space: nowrap;
}

.vision-toggle {
  display: flex;
  align-items: center;
  gap: 8px;
}

// ---- Advanced config number inputs: replaces t-input-number's stepper buttons, lighter weight ----
// Use a plain t-input + suffix unit + type=number. Native number inputs show
// a pair of spin buttons on Chrome; hidden in scoped styles to keep the visuals
// clean. Max/min are clamped via onBlur, not relying on native step limits.
.number-input {
  :deep(input::-webkit-outer-spin-button),
  :deep(input::-webkit-inner-spin-button) {
    -webkit-appearance: none;
    appearance: none;
    margin: 0;
  }

  // Firefox renders type=number as textfield style, which looks better
  :deep(input[type="number"]) {
    -moz-appearance: textfield;
    appearance: textfield;
  }
}

.number-input__unit {
  font-size: 12px;
  color: var(--td-text-color-placeholder);
  user-select: none;
}

// ---- Status icon for the footer-left test button ----
.status-icon {
  font-size: 16px;
  flex-shrink: 0;

  &.available {
    color: var(--td-brand-color);
  }

  &.unavailable {
    color: var(--td-error-color);
  }
}

// ---- Small tag in the subtitle ----
.subtitle-tag {
  display: inline-flex;
  align-items: center;
  padding: 0 6px;
  margin-left: 6px;
  height: 16px;
  font-size: 10px;
  font-weight: 500;
  border-radius: 3px;

  &--ok {
    color: var(--td-success-color);
    background: var(--td-success-color-light);
  }

  &--muted {
    color: var(--td-text-color-placeholder);
    background: var(--td-bg-color-component);
  }
}
</style>

<!--
  Non-scoped block: per-transport header-icon coloring. Mirrors the matching
  .service-card--{transport} .service-card__badge in McpSettings so the
  list-card → drawer hand-off stays visually continuous.
-->
<style lang="less">
.mcp-drawer--sse .setting-drawer__header-icon {
  background: rgba(17, 128, 83, 0.12);
  color: #118053;
}

.mcp-drawer--http-streamable .setting-drawer__header-icon {
  background: rgba(0, 82, 217, 0.1);
  color: #0052D9;
}
</style>
