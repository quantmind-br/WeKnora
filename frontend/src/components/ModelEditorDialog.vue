<template>
  <SettingDrawer :visible="dialogVisible" :title="isEdit ? $t('model.editor.editTitle') : $t('model.editor.addTitle')"
    :description="getModalDescription()" :icon="modelTypeIcon" :confirm-loading="saving"
    :confirm-disabled="formData.provider === 'weknoracloud' && wkcCredentialState !== 'configured'"
    @update:visible="(v: boolean) => dialogVisible = v" @confirm="handleConfirm" @cancel="handleCancel">

    <!--
      Footer-left slot: connection-test button lives here so it sits next to
      Save/Cancel — primary actions all aligned along the bottom of the
      drawer. Avoids the "test, then scroll back down to save" dance.
      Mirrors the pattern used in WebSearchSettings' provider drawer.
    -->
    <template v-if="formData.source === 'remote'" #footer-left>
      <t-button variant="outline" @click="checkRemoteAPI" :loading="checking"
        :disabled="!formData.modelName || (!formData.baseUrl && formData.provider !== 'weknoracloud') || (formData.provider === 'weknoracloud' && wkcCredentialState !== 'configured')">
        <template #icon>
          <t-icon v-if="!checking && remoteChecked && remoteAvailable" name="check-circle-filled"
            class="status-icon available" />
          <t-icon v-else-if="!checking && remoteChecked && !remoteAvailable" name="close-circle-filled"
            class="status-icon unavailable" />
        </template>
        {{ checking ? $t('model.editor.testing') : $t('model.editor.testConnection') }}
      </t-button>
      <span v-if="remoteChecked" :class="['footer-test-message', remoteAvailable ? 'success' : 'error']"
        :title="remoteMessage">
        {{ remoteMessage }}
      </span>
    </template>

    <t-form ref="formRef" :data="formData" :rules="rules" layout="vertical">

      <section v-if="!isEdit" class="setting-drawer__section">
        <h4 class="setting-drawer__section-title">{{ $t('model.editor.sectionType') }}</h4>
        <div class="model-type-options" role="radiogroup" :aria-label="$t('model.editor.typeLabel')">
          <button
            v-for="opt in modelTypeChoices"
            :key="opt.value"
            type="button"
            class="model-type-option"
            :class="{ 'is-active': activeModelType === opt.value }"
            role="radio"
            :aria-checked="activeModelType === opt.value"
            @click="selectModelType(opt.value)"
          >
            <t-icon :name="opt.icon" class="model-type-option__icon" />
            <span class="model-type-option__label">{{ opt.label }}</span>
          </button>
        </div>
      </section>

      <!--
        Section 1 — Model source + model name (source directly determines the fields below, so they're placed in one section)
      -->
      <section class="setting-drawer__section">
        <h4 class="setting-drawer__section-title">{{ $t('model.editor.sectionSource') }}</h4>

        <div class="form-item">
          <!--
            The section title already says "Model Source", so no need to repeat the label here,
            Present the segmented control directly as the section's first content, avoiding a "double title" feel.
          -->
          <div class="source-options" role="radiogroup" :aria-label="$t('model.editor.sourceLabel')">
            <button
              type="button"
              class="source-option"
              :class="{ 'is-active': formData.source === 'remote' }"
              role="radio"
              :aria-checked="formData.source === 'remote'"
              @click="formData.source = 'remote'"
            >
              <t-icon name="cloud" class="source-option__icon" />
              <span class="source-option__label">{{ $t('model.editor.sourceRemote') }}</span>
            </button>
            <button
              type="button"
              class="source-option"
              :class="{ 'is-active': formData.source === 'local', 'is-disabled': ollamaServiceStatus === false || activeModelType === 'rerank' }"
              :disabled="ollamaServiceStatus === false || activeModelType === 'rerank'"
              role="radio"
              :aria-checked="formData.source === 'local'"
              @click="formData.source = 'local'"
            >
              <t-icon name="server" class="source-option__icon" />
              <span class="source-option__label">{{ $t('model.editor.sourceLocal') }}</span>
            </button>
          </div>

          <!-- Hint message for ReRank models not supporting Ollama -->
          <div v-if="activeModelType === 'rerank'" class="ollama-unavailable-tip rerank-tip">
            <t-icon name="info-circle-filled" class="tip-icon info" />
            <span class="tip-text">{{ $t('model.editor.ollamaNotSupportRerank') }}</span>
          </div>

          <!-- Hint message when Ollama is unavailable -->
          <div v-else-if="shouldShowOllamaUnavailableTip(formData.source, activeModelType, ollamaServiceStatus)"
            class="ollama-unavailable-tip">
            <t-icon name="error-circle-filled" class="tip-icon" />
            <span class="tip-text">{{ $t('model.editor.ollamaUnavailable') }}</span>
            <t-button variant="text" size="small" @click="goToOllamaSettings" class="tip-link">
              <template #icon><t-icon name="jump" /></template>
              {{ $t('model.editor.goToOllamaSettings') }}
            </t-button>
          </div>
        </div>

        <!-- Ollama local model selector -->
        <div v-if="formData.source === 'local'" class="form-item">
          <label class="form-label required">{{ $t('model.modelName') }}</label>
          <div class="model-select-row">
            <t-select v-model="formData.modelName" :loading="loadingOllamaModels" :class="{ 'downloading': downloading }"
              :style="downloading ? `--progress: ${downloadProgress}%` : ''" filterable :filter="handleModelFilter"
              :placeholder="$t('model.searchPlaceholder')" @focus="loadOllamaModels"
              @visible-change="handleDropdownVisibleChange">
              <!-- Downloaded models -->
              <t-option v-for="model in filteredOllamaModels" :key="model.name" :value="model.name" :label="model.name">
                <div class="model-option">
                  <t-icon name="check-circle-filled" class="downloaded-icon" />
                  <span class="model-name">{{ model.name }}</span>
                  <span class="model-size">{{ formatModelSize(model.size) }}</span>
                </div>
              </t-option>

              <!-- Download new model option (shown only when the search term is not in the list) -->
              <t-option v-if="showDownloadOption" :value="`__download__${searchKeyword}`"
                :label="$t('model.editor.downloadLabel', { keyword: searchKeyword })" class="download-option">
                <div class="model-option download">
                  <t-icon name="download" class="download-icon" />
                  <span class="model-name">{{ $t('model.editor.downloadLabel', { keyword: searchKeyword }) }}</span>
                </div>
              </t-option>

              <!-- Download progress suffix -->
              <template v-if="downloading" #suffix>
                <div class="download-suffix">
                  <t-icon name="loading" class="spinning" />
                  <span class="progress-text">{{ downloadProgress.toFixed(1) }}%</span>
                </div>
              </template>
            </t-select>

            <!-- Refresh button -->
            <t-button variant="text" size="small" :loading="loadingOllamaModels" @click="refreshOllamaModels"
              class="refresh-btn">
              <t-icon name="refresh" />
              {{ $t('model.editor.refreshList') }}
            </t-button>
          </div>
        </div>
      </section>

      <!-- Remote API configuration -->
      <template v-if="formData.source === 'remote'">
        <section class="setting-drawer__section">
          <h4 class="setting-drawer__section-title">{{ $t('model.editor.sectionProvider') }}</h4>

          <!-- Vendor selector -->
          <div class="form-item">
            <label class="form-label">{{ $t('model.editor.providerLabel') }}</label>
            <t-select v-model="formData.provider" :placeholder="$t('model.editor.providerPlaceholder')"
              @change="handleProviderChange" :popup-props="{ overlayClassName: 'provider-select-popup' }">
              <!--
                show-overflow-tooltip=false: TDesign by default shows a small bubble with the
                full label on hover, but here the options are already two lines (main name + description), so no
                truncation occurs, and the tooltip would just clash with the already-highlighted gray background. Turn it off directly.
              -->
              <t-option v-for="opt in providerOptions" :key="opt.value" :value="opt.value" :label="opt.label"
                :show-overflow-tooltip="false">
                <div class="provider-option">
                  <span class="provider-name">{{ opt.label }}</span>
                  <span class="provider-desc">{{ opt.description }}</span>
                </div>
              </t-option>
            </t-select>
          </div>

          <!-- WeKnoraCloud hint message -->
          <template v-if="formData.provider === 'weknoracloud'">
            <!-- Credentials configured -->
            <div v-if="wkcCredentialState === 'configured'" class="weknoracloud-hint weknoracloud-hint--ok">
              <t-icon name="check-circle-filled" class="hint-icon hint-icon--ok" />
              <div>
                {{ $t('settings.weknoraCloud.modelHintConfigured') }}
                <a href="https://developers.weixin.qq.com/doc/aispeech/knowledge/atomic_capability/atomic_interface.html"
                  target="_blank" rel="noopener noreferrer" class="doc-link">
                  {{ $t('settings.weknoraCloud.modelHintDocsLink') }}
                  <t-icon name="link" class="link-icon" />
                </a>
              </div>
            </div>

            <!-- Not configured / invalid -->
            <div v-else-if="wkcCredentialState !== 'loading'" class="weknoracloud-hint weknoracloud-hint--warn">
              <t-icon name="error-circle-filled" class="hint-icon hint-icon--warn" />
              <div style="flex: 1;">
                <template v-if="wkcCredentialState === 'expired'">
                  {{ $t('settings.weknoraCloud.credentialExpired') }}
                </template>
                <template v-else>
                  {{ $t('settings.weknoraCloud.credentialUnconfigured') }}
                </template>
                <div style="margin-top: 8px;">
                  <t-button variant="text" size="small" @click="goToWeKnoraCloudSettings"
                    style="padding: 0; height: auto;">
                    <template #icon><t-icon name="jump" /></template>
                    {{ $t('settings.weknoraCloud.goToSettings') }}
                  </t-button>
                </div>
              </div>
            </div>

            <!-- Loading -->
            <div v-else class="weknoracloud-hint">
              <t-icon name="loading" class="spinning hint-icon hint-icon--loading" />
              <span>{{ $t('settings.weknoraCloud.checkingStatus') }}</span>
            </div>
          </template>

          <!-- Model name -->
          <div class="form-item">
            <label class="form-label required">{{ $t('model.modelName') }}</label>
            <t-input v-model="formData.modelName" :placeholder="getModelNamePlaceholder()"
              :disabled="formData.provider === 'weknoracloud' && wkcCredentialState !== 'configured'" />
          </div>

          <div class="form-item">
            <label class="form-label">{{ $t('model.editor.displayNameLabel') }}</label>
            <t-input v-model="formData.displayName" :placeholder="$t('model.editor.displayNamePlaceholder')" />
            <p class="form-desc">{{ $t('model.editor.displayNameDesc') }}</p>
          </div>

          <div v-if="formData.provider !== 'weknoracloud'" class="form-item">
            <label class="form-label required">{{ $t('model.editor.baseUrlLabel') }}</label>
            <t-input v-model="formData.baseUrl" :placeholder="getBaseUrlPlaceholder()" />
          </div>

          <div v-if="formData.provider !== 'weknoracloud'" class="form-item">
            <label class="form-label">{{
              isSignedRerank ? signedRerankAccessKeyLabel : $t('model.editor.apiKeyOptional')
            }}</label>
            <!--
              Edit mode: credentials live behind the /credentials subresource
              of the model — managed by the shared CredentialResource card,
              which now renders an INPUT-LOOKING row (32px tall, same border
              + radius as t-input) so it sits flush with the Base URL field
              above and the custom request headers controls below — no more
              "card inside a card" feel.
              Create mode: the resource doesn't exist yet, so we render a
              plain password input with a leading lock icon and a trailing
              show/hide eye toggle.
            -->
            <CredentialResource v-if="isEdit && props.modelData?.id" :api="credentialApi" :fields="credentialFields"
              :meta="credentialMeta" />
            <t-input v-else v-model="formData.apiKey" :type="showApiKey ? 'text' : 'password'"
              :placeholder="isSignedRerank ? signedRerankAccessKeyPlaceholder : apiKeyPlaceholder"
              class="api-key-input" autocomplete="off" spellcheck="false">
              <template #prefix-icon><t-icon name="lock-on" /></template>
              <template #suffix-icon>
                <t-icon
                  :name="showApiKey ? 'browse-off' : 'browse'"
                  class="api-key-toggle"
                  :aria-label="showApiKey ? 'Hide' : 'Show'"
                  @click.stop="showApiKey = !showApiKey"
                />
              </template>
            </t-input>
            <p v-if="isSignedRerank" class="form-desc">{{ signedRerankCredentialHint }}</p>
          </div>

          <!-- AK/SK Rerank creation mode: SecretKey (edit mode is managed by CredentialResource) -->
          <div v-if="isSignedRerank && !isEdit" class="form-item">
            <label class="form-label required">{{ signedRerankSecretKeyLabel }}</label>
            <t-input v-model="formData.appSecret" type="password"
              :placeholder="signedRerankSecretKeyPlaceholder" autocomplete="off" spellcheck="false">
              <template #prefix-icon><t-icon name="lock-on" /></template>
            </t-input>
          </div>

          <div v-if="isLkeapRerank" class="form-item">
            <label class="form-label">{{ $t('model.editor.lkeap.regionLabel') }}</label>
            <t-input v-model="formData.lkeapRegion" :placeholder="$t('model.editor.lkeap.regionPlaceholder')" />
            <p class="form-desc">{{ $t('model.editor.lkeap.regionDesc') }}</p>
          </div>

          <!-- Custom HTTP header (similar to extra_headers in the OpenAI Python SDK) -->
          <div v-if="formData.provider !== 'weknoracloud'" class="form-item">
            <div class="custom-headers-header">
              <label class="form-label" style="margin-bottom: 0;">{{ $t('model.editor.customHeadersLabel') }}</label>
              <t-button variant="text" size="small" theme="primary" @click="addCustomHeader">
                <template #icon><t-icon name="add" /></template>
                {{ $t('model.editor.customHeadersAdd') }}
              </t-button>
            </div>
            <p class="form-desc custom-headers-desc">{{ $t('model.editor.customHeadersDesc') }}</p>
            <div v-if="formData.customHeaders && formData.customHeaders.length > 0" class="custom-headers-list">
              <div v-for="(item, idx) in formData.customHeaders" :key="idx" class="custom-header-row">
                <t-input v-model="item.key" :placeholder="$t('model.editor.customHeadersKeyPlaceholder')"
                  class="custom-header-key" />
                <t-input v-model="item.value" :placeholder="$t('model.editor.customHeadersValuePlaceholder')"
                  class="custom-header-value" />
                <t-button variant="text" shape="square" size="small" class="custom-header-remove"
                  @click="removeCustomHeader(idx)" :aria-label="$t('common.delete')">
                  <t-icon name="close" />
                </t-button>
              </div>
            </div>
          </div>

          <!--
            Connection test action moved to the drawer footer (footer-left
            slot above) so primary actions live in one row at the bottom.
          -->
        </section>
      </template>

      <!-- Section 3 — Advanced options (rendered only when there is content, to avoid an empty section showing a bottom divider) -->
      <section v-if="['embedding', 'chat', 'vllm'].includes(activeModelType)" class="setting-drawer__section">
        <h4 class="setting-drawer__section-title">{{ $t('model.editor.sectionAdvanced') }}</h4>

        <!-- Embedding-specific: dimension -->
        <div v-if="activeModelType === 'embedding'" class="form-item">
          <label class="form-label">{{ $t('model.editor.dimensionLabel') }}</label>
          <div class="dimension-control">
            <t-input v-model.number="formData.dimension" type="number" :min="128" :max="4096"
              :placeholder="$t('model.editor.dimensionPlaceholder')"
              :disabled="!formData.supportsDimensionOverride || (formData.source === 'local' && checking)" />
            <!-- Ollama local model: auto-detect dimension button -->
            <t-button v-if="formData.source === 'local' && formData.modelName" variant="text" size="small"
              :loading="checking" @click="checkOllamaDimension" class="dimension-check-btn">
              <t-icon name="refresh" />
              {{ $t('model.editor.checkDimension') }}
            </t-button>
          </div>
          <p v-if="dimensionChecked && dimensionMessage" class="dimension-hint" :class="{ success: dimensionSuccess }">
            {{ dimensionMessage }}
          </p>
        </div>

        <div v-if="activeModelType === 'embedding'" class="form-item">
          <label class="form-label">{{ $t('model.editor.dimensionOverrideLabel') }}</label>
          <div class="vision-toggle">
            <t-switch v-model="formData.supportsDimensionOverride" />
            <span class="form-desc form-desc--inline">{{ $t('model.editor.dimensionOverrideDesc') }}</span>
          </div>
        </div>

        <!-- Chat: supports vision toggle (VLLM models are inherently multimodal) -->
        <div v-if="activeModelType === 'chat'" class="form-item">
          <label class="form-label">{{ $t('model.editor.supportsVisionLabel') }}</label>
          <div class="vision-toggle">
            <t-switch v-model="formData.supportsVision" />
            <span class="form-desc form-desc--inline">{{ $t('model.editor.supportsVisionDesc') }}</span>
          </div>
        </div>

        <!-- Chat + remote API: thinking mode parameter format -->
        <div v-if="showThinkingControlField" class="form-item">
          <label class="form-label">{{ $t('model.editor.thinkingControlLabel') }}</label>
          <t-select
            v-model="formData.thinkingControl"
            :key="`thinking-${formData.id}-${formData.thinkingControl}`"
            :popup-props="{ overlayClassName: 'thinking-control-select-popup' }"
            @change="onThinkingControlManualPick"
          >
            <t-option
              v-for="opt in thinkingControlOptions"
              :key="opt.value"
              :value="opt.value"
              :label="opt.label"
              :show-overflow-tooltip="false"
            >
              <div class="thinking-control-option">
                <span class="thinking-control-option__title">{{ opt.label }}</span>
                <span class="thinking-control-option__hint">{{ opt.hint }}</span>
              </div>
            </t-option>
          </t-select>
          <p class="form-desc">{{ $t('model.editor.thinkingControlDesc') }}</p>
        </div>

        <!--
          Background concurrency cap for this model. Only chat / embedding / vllm
          are gated by the governor (see internal/models/limiter), so we surface
          it just for those three. 0 = fall back to the global default.
        -->
        <div class="form-item">
          <label class="form-label">{{ $t('model.editor.maxConcurrencyLabel') }}</label>
          <t-input v-model.number="formData.maxConcurrency" type="number" :min="0" :max="4096"
            :placeholder="$t('model.editor.maxConcurrencyPlaceholder')" />
          <p class="form-desc">{{ $t('model.editor.maxConcurrencyDesc') }}</p>
        </div>
      </section>

    </t-form>
  </SettingDrawer>
</template>

<script setup lang="ts">
import { ref, watch, computed, onUnmounted, nextTick } from 'vue'
import { MessagePlugin, DialogPlugin } from 'tdesign-vue-next'
import { checkOllamaModels, checkRemoteModel, testEmbeddingModel, checkRerankModel, checkASRModel, listOllamaModels, downloadOllamaModel, getDownloadProgress, checkOllamaStatus, listModelProviders, type OllamaModelInfo, type ModelProviderOption } from '@/api/initialization'
import {
  getWeKnoraCloudStatus,
  putModelCredentials,
  deleteModelCredentialField,
  type ModelCredentialField,
} from '@/api/model'
import { useI18n } from 'vue-i18n'
import { useUIStore } from '@/stores/ui'
import {
  defaultThinkingControl,
  resolveThinkingControl,
  type ThinkingControlValue,
} from '@/utils/thinkingControl'
import SettingDrawer from '@/components/settings/SettingDrawer.vue'
import CredentialResource, {
  type CredentialFieldDef,
  type CredentialResourceApi,
} from '@/components/credentials/CredentialResource.vue'
import { shouldShowOllamaUnavailableTip } from '@/components/modelEditorSourceState'

interface CustomHeaderItem {
  key: string
  value: string
}

interface ModelFormData {
  id: string
  name: string
  source: 'local' | 'remote'
  provider?: string // Provider identifier: openai, aliyun, zhipu, generic, etc.
  modelName: string
  displayName?: string
  baseUrl?: string
  apiKey?: string
  dimension?: number
  supportsDimensionOverride?: boolean
  interfaceType?: 'ollama' | 'openai'
  isDefault: boolean
  supportsVision?: boolean
  /** Concurrency limit for this model in background tasks; 0/undefined means fall back to the global default. Only applies to chat/embedding/vllm. */
  maxConcurrency?: number
  /** extra_config.thinking_control — how agent thinking on/off maps to API fields. */
  thinkingControl?: string
  // Custom HTTP request headers (similar to extra_headers in the OpenAI Python SDK)
  customHeaders?: CustomHeaderItem[]
  /** LKEAP Rerank: Tencent Cloud SecretKey (written to app_secret on creation) */
  appSecret?: string
  /** LKEAP Rerank: region, e.g. ap-guangzhou */
  lkeapRegion?: string
}

type EditorModelType = 'chat' | 'embedding' | 'rerank' | 'vllm' | 'asr'

interface Props {
  visible: boolean
  modelType: EditorModelType
  modelData?: ModelFormData | null
}

const { t, te } = useI18n()
const uiStore = useUIStore()

const props = withDefaults(defineProps<Props>(), {
  visible: false,
  modelData: null
})

const emit = defineEmits<{
  'update:visible': [value: boolean]
  'confirm': [data: ModelFormData & { modelType?: EditorModelType }]
}>()

const draftModelType = ref<EditorModelType>(props.modelType)

const isEdit = computed(() => !!props.modelData)

const activeModelType = computed(() => (
  isEdit.value ? props.modelType : draftModelType.value
))

const modelTypeChoices = computed(() => ([
  { value: 'chat' as const, label: t('modelSettings.typeShort.chat'), icon: 'chat' },
  { value: 'embedding' as const, label: t('modelSettings.typeShort.embedding'), icon: 'chart-bubble' },
  { value: 'rerank' as const, label: t('modelSettings.typeShort.rerank'), icon: 'filter-sort' },
  { value: 'vllm' as const, label: t('modelSettings.typeShort.vllm'), icon: 'image' },
  { value: 'asr' as const, label: t('modelSettings.typeShort.asr'), icon: 'sound' },
]))

// Provider list returned by the API
const apiProviderOptions = ref<ModelProviderOption[]>([])
const loadingProviders = ref(false)

// Hardcoded fallback Provider configuration (used when the API is unavailable)
const fallbackProviderOptions = computed(() => [
  {
    value: 'openai',
    label: t('model.editor.providers.openai.label'),
    defaultUrls: {
      chat: 'https://api.openai.com/v1',
      embedding: 'https://api.openai.com/v1',
      rerank: 'https://api.openai.com/v1',
      vllm: 'https://api.openai.com/v1',
      asr: 'https://api.openai.com/v1'
    },
    description: t('model.editor.providers.openai.description'),
    modelTypes: ['chat', 'embedding', 'vllm', 'asr']
  },
  {
    value: 'azure_openai',
    label: t('model.editor.providers.azure_openai.label'),
    defaultUrls: {
      chat: 'https://{resource}.openai.azure.com',
      embedding: 'https://{resource}.openai.azure.com',
      vllm: 'https://{resource}.openai.azure.com',
      asr: 'https://{resource}.openai.azure.com'
    },
    description: t('model.editor.providers.azure_openai.description'),
    modelTypes: ['chat', 'embedding', 'vllm', 'asr']
  },
  {
    value: 'aliyun',
    label: t('model.editor.providers.aliyun.label'),
    defaultUrls: {
      chat: 'https://dashscope.aliyuncs.com/compatible-mode/v1',
      embedding: 'https://dashscope.aliyuncs.com/compatible-mode/v1',
      rerank: 'https://dashscope.aliyuncs.com/api/v1/services/rerank/text-rerank/text-rerank',
      vllm: 'https://dashscope.aliyuncs.com/compatible-mode/v1'
    },
    description: t('model.editor.providers.aliyun.description'),
    modelTypes: ['chat', 'embedding', 'rerank', 'vllm']
  },
  {
    value: 'zhipu',
    label: t('model.editor.providers.zhipu.label'),
    defaultUrls: {
      chat: 'https://open.bigmodel.cn/api/paas/v4',
      embedding: 'https://open.bigmodel.cn/api/paas/v4/embeddings',
      vllm: 'https://open.bigmodel.cn/api/paas/v4'
    },
    description: t('model.editor.providers.zhipu.description'),
    modelTypes: ['chat', 'embedding', 'vllm']
  },
  {
    value: 'openrouter',
    label: t('model.editor.providers.openrouter.label'),
    defaultUrls: {
      chat: 'https://openrouter.ai/api/v1',
      embedding: 'https://openrouter.ai/api/v1'
    },
    description: t('model.editor.providers.openrouter.description'),
    modelTypes: ['chat', 'embedding']
  },
  {
    value: 'requesty',
    label: t('model.editor.providers.requesty.label'),
    defaultUrls: {
      chat: 'https://router.requesty.ai/v1',
      embedding: 'https://router.requesty.ai/v1'
    },
    description: t('model.editor.providers.requesty.description'),
    modelTypes: ['chat', 'embedding']
  },
  {
    value: 'gemini',
    label: t('model.editor.providers.gemini.label'),
    defaultUrls: {
      chat: 'https://generativelanguage.googleapis.com/v1beta/openai',
      embedding: 'https://generativelanguage.googleapis.com/v1beta'
    },
    description: t('model.editor.providers.gemini.description'),
    modelTypes: ['chat', 'embedding']
  },
  {
    value: 'siliconflow',
    label: t('model.editor.providers.siliconflow.label'),
    defaultUrls: {
      chat: 'https://api.siliconflow.cn/v1',
      embedding: 'https://api.siliconflow.cn/v1',
      rerank: 'https://api.siliconflow.cn/v1'
    },
    description: t('model.editor.providers.siliconflow.description'),
    modelTypes: ['chat', 'embedding', 'rerank']
  },
  {
    value: 'jina',
    label: t('model.editor.providers.jina.label'),
    defaultUrls: {
      embedding: 'https://api.jina.ai/v1',
      rerank: 'https://api.jina.ai/v1'
    },
    description: t('model.editor.providers.jina.description'),
    modelTypes: ['embedding', 'rerank']
  },
  {
    value: 'nvidia',
    label: t('model.editor.providers.nvidia.label'),
    defaultUrls: {
      chat: 'https://integrate.api.nvidia.com/v1',
      embedding: 'https://integrate.api.nvidia.com/v1',
      rerank: 'https://ai.api.nvidia.com/v1/retrieval/nvidia/reranking',
      vllm: 'https://integrate.api.nvidia.com/v1',
    },
    description: t('model.editor.providers.nvidia.description'),
    modelTypes: ['chat', 'embedding', 'rerank', 'vllm']
  },
  {
    value: 'novita',
    label: t('model.editor.providers.novita.label'),
    defaultUrls: {
      chat: 'https://api.novita.ai/openai/v1',
      embedding: 'https://api.novita.ai/openai/v1',
      vllm: 'https://api.novita.ai/openai/v1',
    },
    description: t('model.editor.providers.novita.description'),
    modelTypes: ['chat', 'embedding', 'vllm']
  },
  {
    value: 'generic',
    label: t('model.editor.providers.generic.label'),
    defaultUrls: {},
    description: t('model.editor.providers.generic.description'),
    modelTypes: ['chat', 'embedding', 'rerank', 'vllm', 'asr']
  },
])

// Fetch the Provider list from the API
const loadProviders = async () => {
  loadingProviders.value = true
  try {
    const providers = await listModelProviders(activeModelType.value)
    if (providers.length > 0) {
      apiProviderOptions.value = providers
    }
  } catch (error) {
    console.error('Failed to load providers from API, using fallback', error)
  } finally {
    loadingProviders.value = false
  }
}

// Provider list filtered by the current model type
// Prefer the defaultUrls/modelTypes data returned by the API, but use i18n for label/description
const providerOptions = computed(() => {
  // When API data is available, use the API's structural data + i18n's display text
  if (apiProviderOptions.value.length > 0) {
    return apiProviderOptions.value.map(p => ({
      ...p,
      label: te(`model.editor.providers.${p.value}.label`)
        ? t(`model.editor.providers.${p.value}.label`)
        : p.label,
      description: te(`model.editor.providers.${p.value}.description`)
        ? t(`model.editor.providers.${p.value}.description`)
        : p.description,
    }))
  }
  // Fall back to hardcoded values, filtered by modelTypes
  return fallbackProviderOptions.value.filter(p =>
    p.modelTypes.includes(activeModelType.value)
  )
})

const dialogVisible = computed({
  get: () => props.visible,
  set: (val) => emit('update:visible', val)
})

const showThinkingControlField = computed(() =>
  activeModelType.value === 'chat' && formData.value.source === 'remote',
)

const resolvedThinkingControl = (): ThinkingControlValue =>
  defaultThinkingControl(
    formData.value.provider || '',
    formData.value.modelName || '',
  )

/** Whether the user manually changed the thinking parameter format (if changed, stop auto-overriding until the provider is switched) */
const thinkingControlManual = ref(false)
/** Populating the form from modelData; ignore programmatic change side effects from the vendor/source controls */
const hydratingForm = ref(false)

const onThinkingControlManualPick = () => {
  thinkingControlManual.value = true
}

const syncThinkingControlToForm = (force = false) => {
  if (!showThinkingControlField.value) return
  if (!force && !isEdit.value && thinkingControlManual.value) return
  formData.value.thinkingControl = resolvedThinkingControl()
}

const applyThinkingControlFromModelData = () => {
  if (!props.modelData || activeModelType.value !== 'chat' || formData.value.source !== 'remote') return
  thinkingControlManual.value = !!props.modelData.thinkingControl
  formData.value.thinkingControl = resolveThinkingControl(
    props.modelData.thinkingControl,
    formData.value.provider || props.modelData.provider || '',
    formData.value.modelName || props.modelData.modelName || '',
  )
}

const thinkingControlOptions = computed(() => {
  const keys = ['none', 'chatTemplateKwargs', 'enableThinking', 'thinkingType'] as const
  const values = ['none', 'chat_template_kwargs', 'enable_thinking', 'thinking_type'] as const
  return keys.map((key, i) => ({
    value: values[i],
    label: t(`model.editor.thinkingControl.${key}.label`),
    hint: t(`model.editor.thinkingControl.${key}.hint`),
  }))
})

// Header icon for the SettingDrawer — uses the same TDesign icon name table
// as the model card list, so the drawer's leading badge visually matches the
// card the user just clicked on.
const modelTypeIcon = computed(() => {
  const map: Record<string, string> = {
    chat: 'chat',
    embedding: 'chart-bubble',
    rerank: 'filter-sort',
    vllm: 'image',
    asr: 'sound',
  }
  return map[activeModelType.value] || 'setting'
})

const isLkeapRerank = computed(
  () => activeModelType.value === 'rerank' && formData.value.provider === 'lkeap',
)
const isVolcengineRerank = computed(
  () => activeModelType.value === 'rerank' && formData.value.provider === 'volcengine',
)
const isSignedRerank = computed(
  () => isLkeapRerank.value || isVolcengineRerank.value,
)
const signedRerankAccessKeyLabel = computed(() => (
  isVolcengineRerank.value
    ? t('model.editor.volcengine.accessKeyLabel')
    : t('model.editor.lkeap.secretIdLabel')
))
const signedRerankAccessKeyPlaceholder = computed(() => (
  isVolcengineRerank.value
    ? t('model.editor.volcengine.accessKeyPlaceholder')
    : t('model.editor.lkeap.secretIdPlaceholder')
))
const signedRerankSecretKeyLabel = computed(() => (
  isVolcengineRerank.value
    ? t('model.editor.volcengine.secretKeyLabel')
    : t('model.editor.lkeap.secretKeyLabel')
))
const signedRerankSecretKeyPlaceholder = computed(() => (
  isVolcengineRerank.value
    ? t('model.editor.volcengine.secretKeyPlaceholder')
    : t('model.editor.lkeap.secretKeyPlaceholder')
))
const signedRerankCredentialHint = computed(() => (
  isVolcengineRerank.value
    ? t('model.editor.volcengine.rerankCredentialHint')
    : t('model.editor.lkeap.rerankCredentialHint')
))

// Credential resource binding for the shared <CredentialResource> component.
const credentialFields = computed<CredentialFieldDef<ModelCredentialField>[]>(() => {
  const fields: CredentialFieldDef<ModelCredentialField>[] = [
    {
      key: 'api_key',
      label: (isSignedRerank.value
        ? signedRerankAccessKeyLabel.value
        : t('model.editor.apiKeyOptional')) as string,
    },
  ]
  if (formData.value.provider === 'weknoracloud') {
    fields.push({ key: 'app_secret', label: 'App Secret' })
  } else if (isSignedRerank.value) {
    fields.push({ key: 'app_secret', label: signedRerankSecretKeyLabel.value as string })
  }
  return fields
})

const credentialApi = computed<CredentialResourceApi<ModelCredentialField>>(() => {
  const id = props.modelData?.id ?? ''
  return {
    save: async (patch) => {
      const meta = await putModelCredentials(id, patch)
      return meta.fields
    },
    remove: async (field) => {
      await deleteModelCredentialField(id, field)
    },
  }
})

// Initial credential metadata. ModelSettings.convertToLegacyFormat
// preserves `credentials` from the main ListModels response so the card
// renders the correct "Configured" state on dialog open.
const credentialMeta = computed(() => (props.modelData as any)?.credentials ?? {
  api_key: { configured: false },
  app_secret: { configured: false },
})

// Placeholder hint for the create-mode API key input. Edit mode replaces
// this input entirely with a <CredentialResource> card.
const apiKeyPlaceholder = computed(() => t('model.editor.apiKeyPlaceholder'))

const formRef = ref()
const saving = ref(false)
// Toggles the create-mode API key input between masked and plain text. Lets
// the user proofread a freshly pasted secret without losing the password
// affordance for everyday use. Reset every time the drawer closes (see
// reset block in the visible watcher) so we never leak the previous value
// across editor sessions.
const showApiKey = ref(false)
const modelChecked = ref(false)
const modelAvailable = ref(false)
const checking = ref(false)
const remoteChecked = ref(false)
const remoteAvailable = ref(false)
const remoteMessage = ref('')
const dimensionChecked = ref(false)
const dimensionSuccess = ref(false)
const dimensionMessage = ref('')

// Ollama model status
const ollamaModelList = ref<OllamaModelInfo[]>([])
const loadingOllamaModels = ref(false)
const searchKeyword = ref('')
const downloading = ref(false)
const downloadProgress = ref(0)
const currentDownloadModel = ref('')
let downloadInterval: any = null

// Ollama service status
const ollamaServiceStatus = ref<boolean | null>(null)
const checkingOllamaStatus = ref(false)

// WeKnoraCloud credential status
const wkcCredentialState = ref<'loading' | 'unconfigured' | 'configured' | 'expired'>('loading')

const checkWkcCredentialStatus = async () => {
  wkcCredentialState.value = 'loading'
  try {
    const status = await getWeKnoraCloudStatus()
    if (status.needs_reinit) {
      wkcCredentialState.value = 'expired'
    } else if (status.has_models) {
      wkcCredentialState.value = 'configured'
    } else {
      wkcCredentialState.value = 'unconfigured'
    }
  } catch {
    wkcCredentialState.value = 'unconfigured'
  }
}

const goToWeKnoraCloudSettings = async () => {
  emit('update:visible', false)
  if (uiStore.showSettingsModal) {
    uiStore.closeSettings()
    await nextTick()
  }
  uiStore.openSettings('weknoracloud')
}

const formData = ref<ModelFormData>({
  id: '',
  name: '',
  source: 'remote',
  provider: 'generic',
  modelName: '',
  displayName: '',
  baseUrl: '',
  apiKey: '',
  dimension: undefined,
  supportsDimensionOverride: false,
  interfaceType: 'ollama',
  isDefault: false,
  supportsVision: false,
  maxConcurrency: undefined,
  thinkingControl: defaultThinkingControl('generic', ''),
  customHeaders: [],
  appSecret: '',
  lkeapRegion: 'ap-guangzhou',
})

const rules = computed(() => ({
  modelName: [
    { required: true, message: t('model.editor.validation.modelNameRequired') },
    {
      validator: (val: string) => {
        if (!val || !val.trim()) {
          return { result: false, message: t('model.editor.validation.modelNameEmpty') }
        }
        if (val.trim().length > 100) {
          return { result: false, message: t('model.editor.validation.modelNameMax') }
        }
        return { result: true }
      },
      trigger: 'blur'
    }
  ],
  baseUrl: [
    {
      required: true,
      message: t('model.editor.validation.baseUrlRequired'),
      trigger: 'blur'
    },
    {
      validator: (val: string) => {
        if (!val || !val.trim()) {
          return { result: false, message: t('model.editor.validation.baseUrlEmpty') }
        }
        // Simple URL format validation
        try {
          new URL(val.trim())
          return { result: true }
        } catch {
          return { result: false, message: t('model.editor.validation.baseUrlInvalid') }
        }
      },
      trigger: 'blur'
    }
  ]
}))

// Get the dialog description text
const getModalDescription = () => {
  const key = `model.editor.description.${activeModelType.value}` as const
  return t(key) || t('model.editor.description.default')
}

// Get the model name placeholder
const getModelNamePlaceholder = () => {
  if (activeModelType.value === 'vllm') {
    return formData.value.source === 'local'
      ? t('model.editor.modelNamePlaceholder.localVllm')
      : t('model.editor.modelNamePlaceholder.remoteVllm')
  }
  if (activeModelType.value === 'asr') {
    return t('model.editor.modelNamePlaceholder.remoteAsr')
  }
  return formData.value.source === 'local'
    ? t('model.editor.modelNamePlaceholder.local')
    : t('model.editor.modelNamePlaceholder.remote')
}

const getBaseUrlPlaceholder = () => {
  if (activeModelType.value === 'vllm') {
    return t('model.editor.baseUrlPlaceholderVllm')
  }
  if (activeModelType.value === 'asr') {
    return t('model.editor.baseUrlPlaceholderAsr')
  }
  return t('model.editor.baseUrlPlaceholder')
}

// Check Ollama service status
const checkOllamaServiceStatus = async () => {
  console.log('Checking Ollama service status...')
  checkingOllamaStatus.value = true
  try {
    const result = await checkOllamaStatus()
    ollamaServiceStatus.value = result.available
    console.log('Ollama service status check finished:', result.available)
  } catch (error) {
    console.error('Failed to check Ollama service status:', error)
    ollamaServiceStatus.value = false
  } finally {
    checkingOllamaStatus.value = false
  }

  // When Ollama is unavailable, default to remote for new scenarios
  if (ollamaServiceStatus.value === false && !isEdit.value && formData.value.source === 'local') {
    formData.value.source = 'remote'
  }
}

// Open the Ollama settings dialog
const goToOllamaSettings = async () => {
  console.log('Clicking the Go-to-Ollama-Settings button')
  // Close the current dialog
  emit('update:visible', false)

  // Close the settings dialog first (if already open)
  if (uiStore.showSettingsModal) {
    uiStore.closeSettings()
    // Wait for the DOM to update
    await nextTick()
  }

  // Open the settings dialog and jump directly to Ollama settings
  console.log('Calling uiStore.openSettings')
  uiStore.openSettings('ollama')
  console.log('uiStore.openSettings call finished')
}

// modelData id from the last time it was opened: used to determine switching models/adding new vs. consecutive opens of the same "add" action
const lastOpenedModelId = ref<string | null>(null)

const selectModelType = async (type: EditorModelType) => {
  if (isEdit.value || draftModelType.value === type) return
  draftModelType.value = type

  if (type === 'rerank') {
    formData.value.source = 'remote'
  }
  if (type !== 'embedding') {
    formData.value.dimension = undefined
    formData.value.supportsDimensionOverride = false
    dimensionChecked.value = false
    dimensionSuccess.value = false
    dimensionMessage.value = ''
  }
  if (type !== 'chat') {
    formData.value.supportsVision = false
    thinkingControlManual.value = false
  }
  remoteChecked.value = false
  remoteAvailable.value = false
  remoteMessage.value = ''

  await loadProviders()
  const supported = providerOptions.value.some(p => p.value === formData.value.provider)
  if (!supported) {
    formData.value.provider = 'generic'
    formData.value.baseUrl = ''
  } else {
    handleProviderChange(formData.value.provider || 'generic')
  }
  if (showThinkingControlField.value && !isEdit.value) {
    thinkingControlManual.value = false
    syncThinkingControlToForm(true)
  }
}

// Watch for visible changes and initialize the form
watch(() => props.visible, (val) => {
  if (val) {
    // Check Ollama service status
    checkOllamaServiceStatus()

    // Load the Model Provider list from the API
    loadProviders()

    // Clear leftover validation/detection results from the previous open each time, to avoid affecting the editing of a different model
    // Directly show the previous "connection successful" state
    modelChecked.value = false
    modelAvailable.value = false
    remoteChecked.value = false
    remoteAvailable.value = false
    remoteMessage.value = ''
    dimensionChecked.value = false
    dimensionSuccess.value = false
    dimensionMessage.value = ''

    const currentId = props.modelData?.id ?? null
    draftModelType.value = props.modelType

    hydratingForm.value = true
    try {
      if (props.modelData) {
        // Edit: always overwrite with the latest modelData. apiKey field is left blank — in
        // edit mode the credential is owned by the <CredentialResource> card,
        // not by this form's apiKey field.
        formData.value = {
          ...props.modelData,
          apiKey: '',
          customHeaders: Array.isArray(props.modelData.customHeaders)
            ? props.modelData.customHeaders.map(h => ({ key: h.key, value: h.value }))
            : [],
        }
        applyThinkingControlFromModelData()
      } else if (lastOpenedModelId.value !== null || !formData.value.id) {
        // Previously editing some model, or first-time add → reset to blank
        resetForm()
      }
      // Otherwise: two consecutive "add" opens (closed via overlay click/ESC in between) → keep the previous input

      lastOpenedModelId.value = currentId

      // ReRank models are forced to use the remote source (Ollama doesn't support ReRank)
      if (activeModelType.value === 'rerank') {
        formData.value.source = 'remote'
      }

      // If the current provider is WeKnoraCloud, check the credential status
      if (formData.value.provider === 'weknoracloud') {
        checkWkcCredentialStatus()
      }

      if (showThinkingControlField.value && !isEdit.value) {
        thinkingControlManual.value = false
        syncThinkingControlToForm(true)
      }
    } finally {
      nextTick(() => {
        hydratingForm.value = false
      })
    }
  }
})

// Reset the form
const resetForm = () => {
  thinkingControlManual.value = false
  formData.value = {
    id: generateId(),
    name: '', // Field kept but unused; modelName is used when saving
    source: 'remote',
    provider: 'generic',
    modelName: '',
    displayName: '',
    baseUrl: '',
    apiKey: '',
    dimension: undefined, // Not filled by default — let the user enter it manually or fetch it via the detect button
    supportsDimensionOverride: false,
    interfaceType: undefined,
    isDefault: false,
    supportsVision: false,
    maxConcurrency: undefined,
    thinkingControl: defaultThinkingControl('generic', ''),
    customHeaders: [],
    appSecret: '',
    lkeapRegion: 'ap-guangzhou',
  }
  modelChecked.value = false
  modelAvailable.value = false
  remoteChecked.value = false
  remoteAvailable.value = false
  remoteMessage.value = ''
  dimensionChecked.value = false
  dimensionSuccess.value = false
  dimensionMessage.value = ''
  showApiKey.value = false
}

// Handle vendor selection changes (auto-fill default URL)
const handleProviderChange = (value: string) => {
  const provider = providerOptions.value.find(opt => opt.value === value)
  if (provider && provider.defaultUrls) {
    // Get the corresponding default URL based on the current model type
    const defaultUrl = provider.defaultUrls[activeModelType.value]
    if (defaultUrl) {
      formData.value.baseUrl = defaultUrl
    }
    if (value === 'lkeap' && activeModelType.value === 'rerank' && !formData.value.modelName?.trim()) {
      formData.value.modelName = 'lke-reranker-base'
    }
    if (value === 'volcengine' && activeModelType.value === 'rerank' && !formData.value.modelName?.trim()) {
      formData.value.modelName = 'doubao-seed-rerank'
    }
    // Reset validation status
    remoteChecked.value = false
    remoteAvailable.value = false
    remoteMessage.value = ''
  }
  // WeKnoraCloud: check credential status
  if (value === 'weknoracloud') {
    checkWkcCredentialStatus()
  }
  if (hydratingForm.value) return
  if (activeModelType.value !== 'chat' || formData.value.source !== 'remote') return
  if (!isEdit.value) {
    thinkingControlManual.value = false
    syncThinkingControlToForm(true)
    return
  }
  // When editing, only follow the default if the user actively switches vendor
  thinkingControlManual.value = false
  syncThinkingControlToForm(true)
}

watch(
  () => [formData.value.source, formData.value.provider, formData.value.modelName] as const,
  ([source, provider, modelName], [prevSource, prevProvider, prevModelName]) => {
    if (hydratingForm.value || isEdit.value) return
    if (activeModelType.value !== 'chat' || source !== 'remote') return
    if (source === prevSource && provider === prevProvider && modelName === prevModelName) return

    const providerChanged = provider !== prevProvider

    if (providerChanged) {
      thinkingControlManual.value = false
      syncThinkingControlToForm(true)
      return
    }
    if (!thinkingControlManual.value) {
      syncThinkingControlToForm(true)
      return
    }
    const prevDefault = defaultThinkingControl(prevProvider || '', prevModelName || '')
    if (formData.value.thinkingControl === prevDefault) {
      syncThinkingControlToForm(true)
    }
  },
)

// Watch for source changes, reset validation state (merged into the watch below)

// Generate unique ID
const generateId = () => {
  return `model_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`
}

// Custom HTTP Header editing
const addCustomHeader = () => {
  if (!Array.isArray(formData.value.customHeaders)) {
    formData.value.customHeaders = []
  }
  formData.value.customHeaders.push({ key: '', value: '' })
}

const removeCustomHeader = (idx: number) => {
  if (!Array.isArray(formData.value.customHeaders)) return
  formData.value.customHeaders.splice(idx, 1)
}

// Filtered model list
const filteredOllamaModels = computed(() => {
  if (!searchKeyword.value) return ollamaModelList.value
  return ollamaModelList.value.filter(model =>
    model.name.toLowerCase().includes(searchKeyword.value.toLowerCase())
  )
})

// Whether to show the "Download model" option
const showDownloadOption = computed(() => {
  if (!searchKeyword.value.trim()) return false
  // Check if the search term already exists in the model list
  const exists = ollamaModelList.value.some(model =>
    model.name.toLowerCase() === searchKeyword.value.toLowerCase()
  )
  return !exists
})

// Custom filter logic (captures the search keyword)
const handleModelFilter = (filterWords: string) => {
  searchKeyword.value = filterWords
  return true // Let TDesign use our filteredOllamaModels
}

// Load the Ollama model list
const loadOllamaModels = async () => {
  // Only load when the local source is selected
  if (formData.value.source !== 'local') return

  loadingOllamaModels.value = true
  try {
    const models = await listOllamaModels()
    ollamaModelList.value = models
  } catch (error) {
    console.error(t('model.editor.loadModelListFailed'), error)
    MessagePlugin.error(t('model.editor.loadModelListFailed'))
  } finally {
    loadingOllamaModels.value = false
  }
}

// Refresh the model list
const refreshOllamaModels = async () => {
  ollamaModelList.value = [] // Clear it to force a reload
  await loadOllamaModels()
  MessagePlugin.success(t('model.editor.listRefreshed'))
}

// Watch for dropdown visibility changes
const handleDropdownVisibleChange = (visible: boolean) => {
  if (!visible) {
    searchKeyword.value = ''
  }
}

// Format model size
const formatModelSize = (bytes: number): string => {
  if (!bytes || bytes === 0) return ''
  const gb = bytes / (1024 * 1024 * 1024)
  return gb >= 1 ? `${gb.toFixed(1)} GB` : `${(bytes / (1024 * 1024)).toFixed(0)} MB`
}

// Check model status (Ollama local model)
const checkModelStatus = async () => {
  if (!formData.value.modelName || formData.value.source !== 'local') {
    return
  }

  try {
    // Call the real Ollama API to check if the model exists
    const result = await checkOllamaModels([formData.value.modelName])
    modelChecked.value = true
    modelAvailable.value = result.models[formData.value.modelName] || false
  } catch (error) {
    console.error('Failed to check model status:', error)
    modelChecked.value = false
    modelAvailable.value = false
  }
}

// Check the dimension of the local Ollama Embedding model
const checkOllamaDimension = async () => {
  if (!formData.value.modelName || formData.value.source !== 'local' || activeModelType.value !== 'embedding') {
    return
  }

  checking.value = true
  dimensionChecked.value = false
  dimensionMessage.value = ''

  try {
    const result = await testEmbeddingModel({
      source: 'local',
      modelName: formData.value.modelName,
      dimension: formData.value.dimension,
      supportsDimensionOverride: formData.value.supportsDimensionOverride ?? false,
    })

    dimensionChecked.value = true
    dimensionSuccess.value = result.available || false

    if (result.available && result.dimension) {
      formData.value.dimension = result.dimension
      dimensionMessage.value = t('model.editor.dimensionDetected', { value: result.dimension })
      MessagePlugin.success(dimensionMessage.value)
    } else {
      if (result.message) {
        console.debug('Backend dimension message:', result.message)
      }
      dimensionMessage.value = t('model.editor.dimensionFailed')
      MessagePlugin.warning(dimensionMessage.value)
    }
  } catch (error: any) {
    console.error('Ollama dimension check failed:', error)
    dimensionChecked.value = true
    dimensionSuccess.value = false
    dimensionMessage.value = t('model.editor.dimensionFailed')
    MessagePlugin.error(dimensionMessage.value)
  } finally {
    checking.value = false
  }
}

// Check the Remote API connection (calls different interfaces depending on the model type)
const checkRemoteAPI = async () => {
  if (!formData.value.modelName || (!formData.value.baseUrl && formData.value.provider !== 'weknoracloud')) {
    MessagePlugin.warning(t('model.editor.fillModelAndUrl'))
    return
  }

  checking.value = true
  remoteChecked.value = false
  remoteMessage.value = ''

  try {
    let result: any

    // Convert the Key-Value array of custom Headers from the form into the map expected by the backend.
    // Consistent with saving in ModelSettings.vue — empty lines are dropped automatically, ensuring the test connection and the actually saved
    // production call use exactly the same set of Headers.
    const customHeaders: Record<string, string> = {}
    if (Array.isArray(formData.value.customHeaders)) {
      for (const item of formData.value.customHeaders) {
        const key = (item?.key ?? '').trim()
        const value = (item?.value ?? '').trim()
        if (key && value) customHeaders[key] = value
      }
    }
    // Only include the field when non-empty, to avoid empty objects showing up in the URL query / logs
    const headerPayload = Object.keys(customHeaders).length > 0
      ? { customHeaders }
      : {}

    // Call different validation interfaces depending on the model type
    // In edit mode, apiKey is managed independently by <CredentialResource> and is not in formData.
    // Pass modelId through to the backend so it can automatically fall back to the stored decrypted value when apiKey is empty,
    // avoiding a case where "test connection fails immediately because no apiKey was sent."
    const idPayload = isEdit.value && props.modelData?.id
      ? { modelId: props.modelData.id as string }
      : {}

    switch (activeModelType.value) {
      case 'chat':
        // Chat model (KnowledgeQA)
        result = await checkRemoteModel({
          modelName: formData.value.modelName,
          baseUrl: formData.value.baseUrl || '',
          apiKey: formData.value.apiKey || '',
          provider: formData.value.provider,
          ...idPayload,
          ...headerPayload,
        })
        break

      case 'embedding':
        // Embedding model
        result = await testEmbeddingModel({
          source: 'remote',
          modelName: formData.value.modelName,
          baseUrl: formData.value.baseUrl || '',
          apiKey: formData.value.apiKey || '',
          dimension: formData.value.dimension,
          supportsDimensionOverride: formData.value.supportsDimensionOverride ?? false,
          provider: formData.value.provider,
          ...idPayload,
          ...headerPayload,
        })
        // If the test succeeds and returns a dimension, auto-fill it
        if (result.available && result.dimension) {
          formData.value.dimension = result.dimension
          MessagePlugin.info(t('model.editor.remoteDimensionDetected', { value: result.dimension }))
        }
        break

      case 'rerank': {
        const signedRerankExtra = isSignedRerank.value
          ? {
              ...(isLkeapRerank.value
                ? {
                    extraConfig: {
                      region: (formData.value.lkeapRegion || 'ap-guangzhou').trim(),
                    },
                  }
                : {}),
              ...(formData.value.appSecret?.trim()
                ? { appSecret: formData.value.appSecret.trim() }
                : {}),
            }
          : {}
        result = await checkRerankModel({
          modelName: formData.value.modelName,
          baseUrl: formData.value.baseUrl || '',
          apiKey: formData.value.apiKey || '',
          provider: formData.value.provider,
          ...idPayload,
          ...headerPayload,
          ...signedRerankExtra,
        })
        break
      }

      case 'vllm':
        // VLLM model (multimodal)
        // VLLM uses checkRemoteModel for the basic connection test
        result = await checkRemoteModel({
          modelName: formData.value.modelName,
          baseUrl: formData.value.baseUrl || '',
          apiKey: formData.value.apiKey || '',
          provider: formData.value.provider,
          ...idPayload,
          ...headerPayload,
        })
        break

      case 'asr':
        // ASR model (speech recognition) — uses the dedicated ASR test endpoint (/v1/audio/transcriptions)
        result = await checkASRModel({
          modelName: formData.value.modelName,
          baseUrl: formData.value.baseUrl || '',
          apiKey: formData.value.apiKey || '',
          provider: formData.value.provider,
          ...idPayload,
          ...headerPayload,
        })
        break

      default:
        MessagePlugin.error(t('model.editor.unsupportedModelType'))
        return
    }

    remoteChecked.value = true
    remoteAvailable.value = result.available || false
    // Previously, the backend's error message was only dropped into console.debug here, so users could only
    // see the generic "Connection failed" toast, with no way to tell if it was a 401 / 404 / model not found,
    // or something else. Changed to: show a generic i18n message on success; on failure, display the backend's
    // specific reason directly (already wrapped in backend's classifyConnectionError with
    // a readable Chinese hint + the raw SDK error), for easier troubleshooting.
    if (result.available) {
      remoteMessage.value = t('model.editor.connectionSuccess')
      MessagePlugin.success(remoteMessage.value)
    } else {
      remoteMessage.value = result.message || t('model.editor.connectionFailed')
      console.debug('Backend message:', result.message)
      MessagePlugin.error(remoteMessage.value)
    }
  } catch (error: any) {
    console.error('Remote API check failed:', error)
    remoteChecked.value = true
    remoteAvailable.value = false
    // Backend 4xx/5xx (e.g. SSRF validation failure) land here. The axios interceptor lifts the backend's
    // { error: { message: "..." } } into error.message, which already contains
    // a readable hint + reason — display it directly, far more useful than a generic "please check the config".
    remoteMessage.value = error?.message || t('model.editor.connectionConfigError')
    MessagePlugin.error(remoteMessage.value)
  } finally {
    checking.value = false
  }
}

// Confirm and save
const handleConfirm = async () => {
  try {
    // Manually validate required fields
    if (!formData.value.modelName || !formData.value.modelName.trim()) {
      MessagePlugin.warning(t('model.editor.validation.modelNameRequired'))
      return
    }

    if (formData.value.modelName.trim().length > 100) {
      MessagePlugin.warning(t('model.editor.validation.modelNameMax'))
      return
    }

    // If type is remote and not WeKnoraCloud, baseUrl is required
    if (formData.value.source === 'remote' && formData.value.provider !== 'weknoracloud') {
      if (!formData.value.baseUrl || !formData.value.baseUrl.trim()) {
        MessagePlugin.warning(t('model.editor.remoteBaseUrlRequired'))
        return
      }

      // Validate Base URL format
      try {
        new URL(formData.value.baseUrl.trim())
      } catch {
        MessagePlugin.warning(t('model.editor.validation.baseUrlInvalid'))
        return
      }
    }

    // Run form validation
    await formRef.value?.validate()

    // Credential removal in edit mode is handled inline by the
    // CredentialResource card (it confirms + DELETEs to /credentials), so
    // the main save flow no longer needs to confirm or handle clear flags.

    saving.value = true

    // If adding new and no id, generate one
    if (!formData.value.id) {
      formData.value.id = generateId()
    }

    emit('confirm', {
      ...formData.value,
      ...(isEdit.value ? {} : { modelType: activeModelType.value }),
    })
    dialogVisible.value = false
    // Reset draft after successful save, so the next new-model dialog starts blank
    resetForm()
    lastOpenedModelId.value = null
    // Remove the success toast here, handled centrally by the parent component
  } catch (error) {
    console.error('Form validation failed:', error)
  } finally {
    saving.value = false
  }
}

// Watch for model selection changes (handles download logic and auto dimension-detection hint)
watch(() => formData.value.modelName, async (newValue, oldValue) => {
  if (!newValue) return

  // Handle download logic
  if (newValue.startsWith('__download__')) {
    // Extract model name
    const modelName = newValue.replace('__download__', '')

    // Reset selection (avoid showing the __download__ prefix)
    formData.value.modelName = ''

    // Start download
    await startDownload(modelName)
    return
  }

  // If it's an embedding model, the selected item is a local Ollama model, and the model name actually changed
  if (activeModelType.value === 'embedding' &&
    formData.value.source === 'local' &&
    newValue !== oldValue &&
    oldValue !== '') {
    // Hint the user that dimensions can be detected
    MessagePlugin.info(t('model.editor.dimensionHint'))
  }
})

// Start downloading the model
const startDownload = async (modelName: string) => {
  downloading.value = true
  downloadProgress.value = 0
  currentDownloadModel.value = modelName

  try {
    // Kick off the download
    const result = await downloadOllamaModel(modelName)
    const taskId = result.taskId

    MessagePlugin.success(t('model.editor.downloadStarted', { name: modelName }))

    // Poll download progress
    downloadInterval = setInterval(async () => {
      try {
        const progress = await getDownloadProgress(taskId)
        downloadProgress.value = progress.progress

        if (progress.status === 'completed') {
          // Download complete
          clearInterval(downloadInterval)
          downloadInterval = null
          downloading.value = false

          MessagePlugin.success(t('model.editor.downloadCompleted', { name: modelName }))

          // Refresh the model list
          await loadOllamaModels()

          // Auto-select the newly downloaded model
          formData.value.modelName = modelName

          // Reset state
          downloadProgress.value = 0
          currentDownloadModel.value = ''

        } else if (progress.status === 'failed') {
          // Download failed
          clearInterval(downloadInterval)
          downloadInterval = null
          downloading.value = false
          MessagePlugin.error(progress.message || t('model.editor.downloadFailed', { name: modelName }))
          downloadProgress.value = 0
          currentDownloadModel.value = ''
        }
      } catch (error) {
        console.error('Failed to fetch download progress:', error)
      }
    }, 1000) // Poll once per second

  } catch (error: any) {
    downloading.value = false
    downloadProgress.value = 0
    currentDownloadModel.value = ''
    console.error('Download start failed:', error)
    MessagePlugin.error(t('model.editor.downloadStartFailed'))
  }
}

// Clear the timer on component unmount
onUnmounted(() => {
  if (downloadInterval) {
    clearInterval(downloadInterval)
  }
})

// Watch for source changes and clear all state
watch(() => formData.value.source, () => {
  // Reset validation state
  modelChecked.value = false
  modelAvailable.value = false
  remoteChecked.value = false
  remoteAvailable.value = false
  remoteMessage.value = ''
  dimensionChecked.value = false
  dimensionSuccess.value = false
  dimensionMessage.value = ''

  // Clear download state
  searchKeyword.value = ''
  if (downloadInterval) {
    clearInterval(downloadInterval)
    downloadInterval = null
  }
  downloading.value = false
  downloadProgress.value = 0
  currentDownloadModel.value = ''

  if (
    !hydratingForm.value
    && !isEdit.value
    && formData.value.source === 'remote'
    && activeModelType.value === 'chat'
  ) {
    thinkingControlManual.value = false
    syncThinkingControlToForm(true)
  }
})

// Watch for model name changes and clear dimension-detection state
watch(() => formData.value.modelName, () => {
  dimensionChecked.value = false
  dimensionSuccess.value = false
  dimensionMessage.value = ''
})

// Cancel (triggered by clicking the bottom "Cancel" button; clicking the overlay/ESC does not trigger it, preserving the draft)
const handleCancel = () => {
  resetForm()
  lastOpenedModelId.value = null
  dialogVisible.value = false
}
</script>

<style lang="less" scoped>
// Empty out the native t-form-item container (this component uses custom .form-item + hand-written labels)
:deep(.t-form) {
  .t-form-item {
    display: none;
  }
}

// Form item styles
.form-item {
  // No bottom margin — vertical rhythm is owned by the parent
  // .setting-drawer__section's `gap`. That keeps the spacing inside a section
  // tight and the gap between sections visually distinct.
  margin-bottom: 0;
}

.form-label {
  display: block;
  margin-bottom: 6px;
  font-size: 13px;
  font-weight: 500;
  color: var(--td-text-color-primary);
  line-height: 1.4;

  // TDesign-style required marker: leading asterisk before the label text,
  // matching the rest of the app's <t-form-item required ...> appearance.
  &.required::before {
    content: '*';
    color: var(--td-error-color);
    margin-right: 4px;
    font-weight: 500;
    line-height: 1;
  }
}

.model-type-options {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.model-type-option {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  min-height: 32px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-container);
  color: var(--td-text-color-secondary);
  font-size: 13px;
  line-height: 1.4;
  cursor: pointer;
  transition: border-color 0.15s ease, color 0.15s ease, background 0.15s ease;

  &__icon {
    font-size: 15px;
    flex-shrink: 0;
  }

  &__label {
    white-space: nowrap;
  }

  &:hover:not(.is-active) {
    border-color: var(--td-brand-color-3, var(--td-brand-color));
    color: var(--td-text-color-primary);
  }

  &.is-active {
    border-color: var(--td-brand-color);
    background: color-mix(in srgb, var(--td-brand-color) 10%, transparent);
    color: var(--td-brand-color);
    font-weight: 500;
  }

  &:focus-visible {
    outline: 2px solid var(--td-brand-color);
    outline-offset: 2px;
  }
}

// Model source segmented control: compact single-line pill-shaped segments. The container itself is a light rounded bar,
// The selected button stands out via a solid background + theme-color outline; unselected state is nearly transparent, saving vertical space.
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

  &:hover:not(.is-disabled):not(.is-active) {
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

  &.is-disabled {
    cursor: not-allowed;
    opacity: 0.45;
  }
}

.source-option__icon {
  font-size: 14px;
  flex-shrink: 0;
}

.source-option__label {
  white-space: nowrap;
}

// Input styles: only adjust font size on the outer .t-input, avoid adding borders again on the inner wrap/inner
// and border-radius, which would create the visual illusion of "nested rounded containers"
:deep(.t-input),
:deep(.t-select),
:deep(.t-textarea),
:deep(.t-input-number) {
  width: 100%;
  font-size: 13px;
}

// Vendor selector styles — moved to a non-scoped block, since the t-select popup renders under body
// See .provider-option styles at the end of the file

// Checkbox
:deep(.t-checkbox) {
  font-size: 13px;

  .t-checkbox__label {
    font-size: 13px;
    color: var(--td-text-color-primary);
  }
}

// API Key input: leading lock icon + trailing clickable "show/hide" eye icon
// TDesign shows the prefix-icon in gray by default — left as is; the eye on suffix
// Use the placeholder color, switch to the main text color on hover to avoid stealing focus.
.api-key-input {
  :deep(.t-input__prefix) {
    color: var(--td-text-color-placeholder);
  }

  :deep(.t-input__suffix) {
    color: var(--td-text-color-placeholder);
  }

  .api-key-toggle {
    cursor: pointer;
    transition: color 0.15s ease;
    font-size: 16px;

    &:hover {
      color: var(--td-text-color-primary);
    }
  }
}

// API test area — soft card style: use a light background + dashed border to frame "action + feedback" as one block,
// so users visually treat it as a standalone "action unit" rather than another plain field.
// (Legacy style kept: only used when a branch still renders the test block inline; currently RemoteAPI
// testing has moved to the SettingDrawer footer-left slot, the main flow no longer goes through this block.)
.api-test-section {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  background: var(--td-bg-color-container-hover);
  border: 1px dashed var(--td-component-stroke);
  border-radius: 8px;

  .test-message {
    font-size: 13px;
    line-height: 1.5;
    flex: 1;

    &.success {
      color: var(--td-brand-color-active);
    }

    &.error {
      color: var(--td-error-color);
    }
  }

  :deep(.t-button) {
    min-width: 88px;
    height: 32px;
    font-size: 13px;
    border-radius: 6px;
    flex-shrink: 0;
  }

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
}

// Connection-test message rendered next to the test button in the drawer
// footer. Truncates with ellipsis so a long backend error doesn't push
// Save/Cancel off-screen — the full text is in the title attribute.
.footer-test-message {
  font-size: 12px;
  line-height: 1.4;
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;

  &.success {
    color: var(--td-brand-color-active);
  }

  &.error {
    color: var(--td-error-color);
  }
}

// Status icon variant used inside the footer button.
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

// WeKnoraCloud hint message
.weknoracloud-hint {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 12px 14px;
  border-radius: 8px;
  font-size: 13px;
  color: var(--td-text-color-secondary);
  line-height: 1.5;

  // Theming via tokens so the warn/ok states track light/dark switches
  // instead of fighting hardcoded `#fff7ed` etc.
  &--ok {
    background: var(--td-success-color-light);
    border: 1px solid var(--td-success-color-focus);
  }

  &--warn {
    background: var(--td-warning-color-light, #fff7ed);
    border: 1px solid var(--td-warning-color-focus, #fed7aa);
    border-left: 3px solid var(--td-warning-color, #f97316);
  }

  .hint-icon {
    font-size: 16px;
    flex-shrink: 0;
    margin-top: 2px;

    &--ok {
      color: var(--td-success-color);
    }

    &--warn {
      color: var(--td-warning-color, #f97316);
    }

    &--loading {
      color: var(--td-text-color-placeholder);
    }
  }
}

// Ollama model selector style
.model-option {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 4px 0;

  .downloaded-icon {
    font-size: 14px;
    color: var(--td-brand-color);
    flex-shrink: 0;
  }

  .download-icon {
    font-size: 14px;
    color: var(--td-brand-color);
    flex-shrink: 0;
  }

  .model-name {
    flex: 1;
    font-size: 13px;
    color: var(--td-text-color-primary);
  }

  .model-size {
    font-size: 12px;
    color: var(--td-text-color-placeholder);
    margin-left: auto;
  }

  &.download {
    .model-name {
      color: var(--td-brand-color);
      font-weight: 500;
    }
  }
}

// Download progress suffix style
.download-suffix {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 0 4px;

  .spinning {
    animation: spin 1s linear infinite;
    font-size: 14px;
    color: var(--td-brand-color);
  }

  .progress-text {
    font-size: 12px;
    font-weight: 500;
    color: var(--td-brand-color);
  }
}

// Progress bar effect for the selection box while downloading
:deep(.t-select.downloading) {
  .t-input {
    position: relative;
    overflow: hidden;

    &::before {
      content: '';
      position: absolute;
      left: 0;
      top: 0;
      bottom: 0;
      width: var(--progress, 0%);
      background: linear-gradient(90deg, rgba(7, 192, 95, 0.08), rgba(7, 192, 95, 0.15));
      transition: width 0.3s ease;
      z-index: 0;
      border-radius: 5px 0 0 5px;
    }

    .t-input__inner,
    input {
      position: relative;
      z-index: 1;
      background: transparent !important;
    }
  }
}

.model-select-row {
  display: flex;
  align-items: center;
  gap: 8px;

  .t-select {
    flex: 1;
  }
}

.refresh-btn {
  flex-shrink: 0;
}

@keyframes spin {
  from {
    transform: rotate(0deg);
  }

  to {
    transform: rotate(360deg);
  }
}

// Dimension control style
.dimension-control {
  display: flex;
  align-items: center;
  gap: 8px;

  :deep(.t-input) {
    flex: 1;
  }
}

.dimension-check-btn {
  flex-shrink: 0;
}

.dimension-hint {
  margin: 8px 0 0 0;
  font-size: 13px;
  line-height: 1.5;
  color: var(--td-error-color);

  &.success {
    color: var(--td-brand-color);
  }
}

// Custom HTTP Header area
.custom-headers-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 6px;
}

.custom-headers-desc {
  margin: 0 0 10px 0;
  font-size: 12px;
  line-height: 1.5;
  color: var(--td-text-color-placeholder);
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

  .custom-header-key {
    flex: 0 0 38%;
  }

  .custom-header-value {
    flex: 1;
  }

  // Ghost icon button — matches the model-card "more" affordance: invisible
  // until hover/focus, then a subtle background pops in. Avoids painting a
  // permanent red splotch next to every header row.
  .custom-header-remove {
    flex-shrink: 0;
    width: 32px;
    height: 32px;
    padding: 0;
    color: var(--td-text-color-placeholder);
    border-radius: 6px;
    transition: all 0.18s ease;

    &:hover {
      background: var(--td-error-color-light);
      color: var(--td-error-color);
    }
  }
}

.form-desc {
  margin: 4px 0 0 0;
  font-size: 12px;
  line-height: 1.5;
  color: var(--td-text-color-placeholder);

  // Inline with switches/checkboxes — drops the top margin so the label and
  // helper text sit on the same baseline.
  &--inline {
    margin: 0;
  }

  &--recommend {
    color: var(--td-brand-color);
  }

  &--warn {
    color: var(--td-warning-color);
  }
}

.vision-toggle {
  display: flex;
  align-items: center;
  gap: 8px;
}

// Ollama unavailable hint style
.ollama-unavailable-tip {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 12px;
  padding: 10px 12px;
  background: var(--td-error-color-light);
  border: 1px solid var(--td-error-color-focus);
  border-radius: 8px;
  font-size: 13px;

  .tip-icon {
    color: var(--td-error-color);
    font-size: 16px;
    flex-shrink: 0;
    margin-right: 2px;

    &.info {
      color: var(--td-brand-color);
    }
  }

  .tip-text {
    color: var(--td-error-color);
    flex: 1;
    line-height: 1.5;
  }

  // ReRank hint uses the theme green style, consistent with the main page
  &.rerank-tip {
    background: var(--td-success-color-light);
    border: 1px solid var(--td-success-color-focus);
    border-left: 3px solid var(--td-brand-color);

    .tip-text {
      color: var(--td-success-color);
    }
  }

  :deep(.tip-link) {
    color: var(--td-brand-color);
    font-size: 13px;
    font-weight: 500;
    padding: 4px 6px 4px 10px !important;
    min-height: auto !important;
    height: auto !important;
    line-height: 1.4 !important;
    text-decoration: none;
    white-space: nowrap;
    display: inline-flex !important;
    align-items: center !important;
    gap: 1px;
    border-radius: 4px;
    transition: all 0.2s ease;

    &:hover {
      background: rgba(7, 192, 95, 0.08) !important;
      color: var(--td-brand-color-active) !important;
    }

    &:active {
      background: rgba(7, 192, 95, 0.12) !important;
    }

    .t-icon {
      font-size: 14px !important;
      margin: 0 !important;
      line-height: 1 !important;
      display: inline-flex !important;
      align-items: center !important;
    }
  }
}

// Destructive-action checkbox for "Remove this credential". Styled to match
// the pattern used in McpServiceDialog so the two dialogs read identically.
.clear-credential {
  display: inline-flex;
  margin-top: 8px;

  :deep(.t-checkbox__label) {
    color: var(--td-error-color);
    font-size: 13px;
  }
}
</style>

<!-- Non-scoped style: t-select popup renders under body, scoped styles can't override it -->
<style lang="less">
.thinking-control-select-popup {
  min-width: 22rem;
  max-width: min(28rem, calc(100vw - 2rem));
  padding: 4px;

  .t-select-option {
    height: auto !important;
    padding: 8px 10px;
    border-radius: 6px;
    margin: 2px 0;
    white-space: normal;
  }
}

.thinking-control-option {
  display: flex;
  flex-direction: column;
  gap: 2px;
  line-height: 1.35;
  min-width: 0;

  &__title {
    font-size: 13px;
    color: var(--td-text-color-primary);
  }

  &__hint {
    font-size: 12px;
    color: var(--td-text-color-placeholder);
    word-break: break-word;
  }
}

.provider-select-popup {
  // Give the container some breathing room: keep options from touching the popup's rounded corners
  padding: 4px;

  // TDesign attaches an overflow tooltip to t-select-option by default (floats on the right
  // to show the full label). Our option layout is "main name + secondary description" on two lines, which never
  // triggers truncation, so the tooltip is just visual noise → hide the popup's built-in tooltip directly.
  + .t-popup .t-tooltip,
  ~ .t-popup .t-tooltip {
    display: none !important;
  }

  .t-select-option {
    height: auto !important;
    padding: 8px 10px;
    border-radius: 6px;
    margin: 2px 0;
    outline: none;
    transition: background-color 0.15s ease;

    &:focus,
    &:focus-visible {
      outline: none;
    }

    // hover state: use a light brand color instead of strong gray, consistent with the theme tone
    &:hover:not(.t-is-selected) {
      background-color: var(--td-bg-color-container-hover);
    }
  }

  // active state: a slightly lighter background + a theme-colored bar on the left as affordance, no longer a fully filled gray background
  .t-select-option.t-is-selected {
    background-color: var(--td-brand-color-light);
    color: var(--td-text-color-primary);
    font-weight: 500;
    position: relative;

    &::before {
      content: '';
      position: absolute;
      left: 0;
      top: 8px;
      bottom: 8px;
      width: 3px;
      background: var(--td-brand-color);
      border-radius: 0 2px 2px 0;
    }

    .provider-name {
      color: var(--td-brand-color);
    }
  }

  .provider-option {
    display: flex;
    flex-direction: column;
    gap: 2px;
    width: 100%;
    min-width: 0;

    .provider-name {
      font-size: 13px;
      font-weight: 500;
      color: var(--td-text-color-primary);
      line-height: 20px;
    }

    .provider-desc {
      font-size: 12px;
      color: var(--td-text-color-placeholder);
      line-height: 18px;
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
    }
  }
}
</style>
