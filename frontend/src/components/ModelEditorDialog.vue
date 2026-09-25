<template>
  <SettingDrawer :visible="dialogVisible" :title="isEdit ? $t('model.editor.editTitle') : $t('model.editor.addTitle')"
    :description="getModalDescription()" :icon="modelTypeIcon" :confirm-loading="saving" :cancel-disabled="saving"
    :confirm-text="$t('model.editor.saveAndClose')"
    :close-on-overlay-click="!saving" :close-on-esc-keydown="!saving"
    :confirm-disabled="formData.provider === 'weknoracloud' && wkcCredentialState !== 'configured'"
    @update:visible="(v: boolean) => dialogVisible = v" @confirm="handleConfirm" @cancel="handleCancel">

    <template v-if="formData.source === 'remote'" #footer-left>
      <t-button variant="outline" @click="checkRemoteAPI" :loading="checking"
        :disabled="saving || !formData.modelName || (!formData.baseUrl && formData.provider !== 'weknoracloud') || (formData.provider === 'weknoracloud' && wkcCredentialState !== 'configured')">
        <template #icon>
          <t-icon v-if="!checking && remoteChecked && remoteAvailable" name="check-circle-filled"
            class="status-icon available" />
          <t-icon v-else-if="!checking && remoteChecked && !remoteAvailable" name="close-circle-filled"
            class="status-icon unavailable" />
        </template>
        {{ checking ? $t('model.editor.testing') : $t('model.editor.testConnection') }}
      </t-button>
      <span v-if="remoteChecked" :class="['connection-status', remoteAvailable ? 'success' : 'error']">
        {{ remoteAvailable ? $t('model.editor.connectionSuccess') : $t('model.editor.connectionFailed') }}
      </span>
    </template>

    <template #footer-extra>
      <div v-if="saveError" class="connection-result error" role="alert">
        <strong>{{ $t('modelSettings.toasts.saveFailed') }}</strong>
        <div class="connection-result__details" tabindex="0">{{ saveError }}</div>
      </div>
      <div v-if="formData.source === 'remote'" class="connection-feedback" aria-live="polite">
        <p class="connection-hint">{{ $t(isEdit ? 'model.editor.testDraftEditHint' : 'model.editor.testDraftHint') }}</p>
        <p v-if="remoteStale" class="connection-hint">{{ $t('model.editor.testStale') }}</p>
        <div v-if="remoteChecked && !remoteAvailable" class="connection-result error">
          <div class="connection-result__header">
            <strong>{{ $t('model.editor.connectionFailed') }}</strong>
            <t-button size="small" variant="text" @click="copyWithToast(remoteMessage, 'common.copied')">
              {{ $t('common.copy') }}
            </t-button>
          </div>
          <div class="connection-result__details" tabindex="0">{{ remoteMessage }}</div>
        </div>
      </div>
    </template>

    <t-form :inert="saving || undefined" ref="formRef" :data="formData" :rules="rules" layout="vertical">

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
              :loading="loadingProviders" filterable
              @change="handleProviderChange"
              :popup-props="{ overlayClassName: 'wk-popover provider-select-popup', overlayInnerStyle: matchTriggerWidth }">
              <!--
                已选值：图标 + 本地化名称（描述只在下拉里展示，避免输入框过长）。
                目录里已经没有这个 provider 时（厂商下线 / 旧数据）退回显示原始 id，
                否则输入框会整个空掉，看起来像"没选厂商"，但保存时仍会带上它。
              -->
              <template #valueDisplay>
                <span v-if="formData.provider" class="provider-value">
                  <img v-if="selectedProviderIcon" :src="selectedProviderIcon" class="provider-icon" alt="" />
                  <span class="provider-value__name">{{ selectedProviderDisplayLabel }}</span>
                </span>
              </template>
              <!--
                show-overflow-tooltip=false: TDesign by default shows a small bubble with the
                full label on hover, but here the options are already two lines (main name + description), so no
                truncation occurs, and the tooltip would just clash with the already-highlighted gray background. Turn it off directly.
              -->
              <t-option v-for="opt in providerOptions" :key="opt.value" :value="opt.value"
                :label="providerDisplayLabel(opt)" :show-overflow-tooltip="false">
                <div class="provider-option">
                  <img v-if="providerIcon(opt)" :src="providerIcon(opt)" class="provider-icon" alt="" />
                  <span v-else class="provider-icon provider-icon--placeholder" aria-hidden="true">
                    {{ providerDisplayLabel(opt).slice(0, 1) }}
                  </span>
                  <div class="provider-option__text">
                    <span class="provider-name">{{ providerDisplayLabel(opt) }}</span>
                    <span class="provider-desc">{{ providerDisplayDescription(opt) }}</span>
                  </div>
                </div>
              </t-option>
            </t-select>
            <p v-if="vendorDocLink" class="form-desc provider-doc-link">
              <a :href="vendorDocLink" target="_blank" rel="noopener noreferrer">
                {{ $t('model.editor.providerDocs', { provider: selectedProviderDisplayLabel }) }}
                <t-icon name="jump" size="12px" />
              </a>
            </p>
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

          <!--
            Model name: the provider's built-in catalog supplies candidates, while free input is still allowed (filterable + creatable).
            Picking a catalog model auto-fills context window / output limit / vision / dimension (empty fields only).
          -->
          <div class="form-item">
            <label class="form-label required">{{ $t('model.modelName') }}</label>
            <!--
              没有内置目录时用纯输入框：TDesign 的 select 在零选项时会隐藏整个
              浮层（hideEmptyPopup），连 creatable 的"创建"行也一并藏掉，于是
              用户能打字、但没有任何方式提交，失焦后输入直接丢失。自定义
              (OpenAI 兼容接口) 正是这种情况，所以那里根本填不进模型名。
            -->
            <t-input
              v-if="catalogModelOptions.length === 0"
              v-model="formData.modelName"
              :placeholder="getModelNamePlaceholder()"
              :disabled="formData.provider === 'weknoracloud' && wkcCredentialState !== 'configured'"
              clearable
              autocomplete="off"
              spellcheck="false"
            />
            <t-select
              v-else
              v-model="formData.modelName"
              filterable
              creatable
              clearable
              :placeholder="getModelNamePlaceholder()"
              :disabled="formData.provider === 'weknoracloud' && wkcCredentialState !== 'configured'"
              :popup-props="{ overlayClassName: 'wk-popover catalog-model-select-popup', overlayInnerStyle: matchTriggerWidth }"
              @create="handleCatalogModelCreate"
              @change="handleCatalogModelChange"
            >
              <t-option v-for="opt in catalogModelOptions" :key="opt.value" :value="opt.value" :label="opt.label"
                :show-overflow-tooltip="false">
                <div class="catalog-model-option">
                  <span class="catalog-model-option__name">{{ opt.label }}</span>
                  <span v-if="opt.value !== opt.label" class="catalog-model-option__id">{{ opt.value }}</span>
                  <span class="catalog-model-option__badges">
                    <span v-if="opt.contextWindow" class="catalog-badge">{{ opt.contextWindow }}</span>
                    <span v-if="opt.dimension" class="catalog-badge">{{ $t('model.editor.dimensionLabel') }} {{ opt.dimension }}</span>
                    <span v-if="opt.reasoning" class="catalog-badge catalog-badge--accent">{{ $t('model.editor.catalog.reasoning') }}</span>
                    <span v-if="opt.vision" class="catalog-badge catalog-badge--accent">{{ $t('model.editor.catalog.vision') }}</span>
                  </span>
                </div>
              </t-option>
            </t-select>
            <p v-if="catalogModelOptions.length > 0" class="form-desc">{{ $t('model.editor.catalog.hint') }}</p>
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
            <label class="form-label" :class="{ required: apiKeyRequired }">{{ apiKeyLabel }}</label>
            <!--
              Edit mode: credentials live behind the /credentials subresource
              of the model — managed by the shared CredentialResource card,
              which now renders an INPUT-LOOKING row (32px tall, same border
              + radius as t-input) so it sits flush with the Base URL field
              above and the custom request headers controls below — no more
              "card inside a card" feel.
              Create mode: the resource doesn't exist yet, so we render a
              plain password input with a leading lock icon; TDesign's password
              input provides the show/hide toggle.
            -->
            <CredentialResource v-if="isEdit && props.modelData?.id" :api="credentialApi" :fields="credentialFields"
              :meta="credentialMeta" @changed="invalidateConnectionTest()" />
            <t-input v-else v-model="formData.apiKey" type="password"
              :placeholder="apiKeyPlaceholder"
              class="api-key-input" autocomplete="new-password" spellcheck="false">
              <template #prefix-icon><t-icon name="lock-on" /></template>
            </t-input>
            <p v-if="apiKeyHint" class="form-desc">{{ apiKeyHint }}</p>
          </div>

          <!--
            Provider-declared secret extra field (e.g. SecretKey for LKEAP / Volcengine Rerank):
            create mode writes it to app_secret; edit mode is managed by the CredentialResource card above.
          -->
          <div v-if="secretExtraField && !isEdit && formData.provider !== 'weknoracloud'" class="form-item">
            <label class="form-label" :class="{ required: secretExtraField.required }">{{ extraFieldDisplayLabel(secretExtraField) }}</label>
            <t-input v-model="formData.appSecret" type="password"
              :placeholder="secretExtraField.placeholder || ''" autocomplete="new-password" spellcheck="false">
              <template #prefix-icon><t-icon name="lock-on" /></template>
            </t-input>
            <p v-if="secretExtraField.placeholder" class="form-desc">{{ secretExtraField.placeholder }}</p>
          </div>

          <!-- 厂商声明的其余额外字段：按 type 动态渲染，值存入 extra_config[key] -->
          <div v-for="field in plainExtraFields" :key="field.key" class="form-item">
            <label class="form-label" :class="{ required: field.required }">{{ extraFieldDisplayLabel(field) }}</label>
            <div v-if="field.type === 'boolean'" class="vision-toggle">
              <t-switch :model-value="extraConfigBool(field.key)"
                @update:model-value="(v: boolean) => setExtraConfig(field.key, v ? 'true' : 'false')" />
              <span v-if="extraFieldDisplayPlaceholder(field)" class="form-desc form-desc--inline">{{ extraFieldDisplayPlaceholder(field) }}</span>
            </div>
            <t-select v-else-if="field.type === 'select'" :model-value="formData.extraConfig[field.key] || ''"
              :placeholder="extraFieldDisplayPlaceholder(field)" clearable
              @update:model-value="(v: string) => setExtraConfig(field.key, v)">
              <t-option v-for="opt in (field.options || [])" :key="opt.value" :value="opt.value"
                :label="extraFieldDisplayOptionLabel(opt)" />
            </t-select>
            <t-input v-else-if="field.type === 'number'" :model-value="formData.extraConfig[field.key] || ''"
              type="number" :placeholder="extraFieldDisplayPlaceholder(field)"
              @update:model-value="(v: string | number) => setExtraConfig(field.key, String(v ?? ''))" />
            <t-input v-else-if="field.type === 'password'" :model-value="formData.extraConfig[field.key] || ''"
              type="password" :placeholder="extraFieldDisplayPlaceholder(field)" autocomplete="new-password" spellcheck="false"
              @update:model-value="(v: string) => setExtraConfig(field.key, v)">
              <template #prefix-icon><t-icon name="lock-on" /></template>
            </t-input>
            <t-input v-else :model-value="formData.extraConfig[field.key] || ''"
              :placeholder="extraFieldDisplayPlaceholder(field)"
              @update:model-value="(v: string) => setExtraConfig(field.key, v)" />
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
            接入诊断：由后端目录实时解析出的有效协议 / 思考格式 / 支持等级 /
            是否目录内 / 上下文窗口，只读。厂商、模型名、Base URL 变化后 400ms 防抖刷新。
            仅对话 / VLM 有意义——Embedding、ReRank、ASR 既没有思考等级也没有
            上下文窗口，展示出来只会让人以为这些值对它们生效。
          -->
          <div v-if="showResolvedPanel" class="form-item">
            <div class="resolved-panel" aria-live="polite">
              <div class="resolved-panel__header">
                <t-icon name="system-code" class="resolved-panel__icon" />
                <span class="resolved-panel__title">{{ $t('model.editor.resolved.title') }}</span>
                <t-icon v-if="resolving" name="loading" class="spinning resolved-panel__loading" />
              </div>
              <p v-if="!resolved && !resolving && !resolveFailed" class="form-desc">{{ $t('model.editor.resolved.empty') }}</p>
              <!-- 后端错误体形态不固定，取不到可读文案时只显示标题，不显示 [object Object] -->
              <p v-else-if="resolveFailed" class="form-desc form-desc--warn">
                {{ $t('model.editor.resolved.failed') }}<template v-if="resolveError">: {{ resolveError }}</template>
              </p>
              <dl v-else-if="resolved" class="resolved-panel__grid">
                <dt>{{ $t('model.editor.resolved.protocol') }}</dt>
                <dd><code>{{ resolved.api }}</code></dd>
                <dt>{{ $t('model.editor.resolved.catalog') }}</dt>
                <dd>
                  <span class="catalog-badge" :class="{ 'catalog-badge--accent': resolved.cataloged }">
                    {{ resolved.cataloged ? $t('model.editor.resolved.catalogedYes') : $t('model.editor.resolved.catalogedNo') }}
                  </span>
                </dd>
                <template v-if="resolved.url">
                  <dt>{{ $t('model.editor.resolved.endpoint') }}</dt>
                  <dd><code class="resolved-endpoint">{{ resolved.url }}</code></dd>
                </template>
                <template v-if="resolved.remote_model && resolved.remote_model !== formData.modelName">
                  <dt>{{ $t('model.editor.advanced.remoteModelName.label') }}</dt>
                  <dd><code>{{ resolved.remote_model }}</code></dd>
                </template>
                <template v-if="isChatLike">
                  <dt>{{ $t('model.editor.resolved.thinkingFormat') }}</dt>
                  <dd><code>{{ resolved.capabilities?.thinking_format || '-' }}</code></dd>
                  <dt>{{ $t('model.editor.resolved.thinkingLevels') }}</dt>
                  <dd>
                    <template v-if="resolvedThinkingLevels.length > 0">
                      <span v-for="level in resolvedThinkingLevels" :key="level" class="catalog-badge">
                        {{ $t(levelLabelKey(level)) }}
                      </span>
                    </template>
                    <span v-else class="form-desc form-desc--inline">{{ $t('model.editor.resolved.noThinking') }}</span>
                  </dd>
                  <template v-if="resolved.capabilities?.context_window">
                    <dt>{{ $t('model.editor.contextWindowLabel') }}</dt>
                    <dd>{{ formatTokenCount(resolved.capabilities.context_window) }}</dd>
                  </template>
                  <template v-if="resolved.capabilities?.max_output_tokens">
                    <dt>{{ $t('model.editor.maxOutputTokensLabel') }}</dt>
                    <dd>{{ formatTokenCount(resolved.capabilities.max_output_tokens) }}</dd>
                  </template>
                </template>
              </dl>
            </div>
          </div>

          <!--
            Connection test action moved to the drawer footer (footer-left
            slot above) so primary actions live in one row at the bottom.
          -->
        </section>
      </template>

      <!-- Section 3 — Advanced options (rendered only when there is content, to avoid an empty section showing a bottom divider) -->
      <section v-if="['embedding', 'chat', 'vllm'].includes(activeModelType) || formData.source === 'remote'" class="setting-drawer__section">
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

        <!-- Chat / VLM: context window. Agent compaction sizes itself from this. -->
        <div v-if="activeModelType === 'chat' || activeModelType === 'vllm'" class="form-item">
          <label class="form-label">{{ $t('model.editor.contextWindowLabel') }}</label>
          <t-input v-model.number="formData.contextWindow" type="number" :min="1024" :max="10000000"
            :placeholder="$t('model.editor.contextWindowPlaceholder', { value: DEFAULT_MODEL_CONTEXT_WINDOW })" />
          <p class="form-desc">{{ $t('model.editor.contextWindowDesc') }}</p>
        </div>

        <!-- Chat: supports vision toggle (VLLM models are inherently multimodal) -->
        <div v-if="activeModelType === 'chat'" class="form-item">
          <label class="form-label">{{ $t('model.editor.supportsVisionLabel') }}</label>
          <div class="vision-toggle">
            <t-switch v-model="formData.supportsVision" />
            <span class="form-desc form-desc--inline">{{ $t('model.editor.supportsVisionDesc') }}</span>
          </div>
        </div>

        <!-- Chat / VLM: max output tokens (catalog default when empty). -->
        <div v-if="activeModelType === 'chat' || activeModelType === 'vllm'" class="form-item">
          <label class="form-label">{{ $t('model.editor.maxOutputTokensLabel') }}</label>
          <t-input v-model.number="formData.maxOutputTokens" type="number" :min="1" :max="10000000"
            :placeholder="$t('model.editor.maxOutputTokensPlaceholder')" />
          <p class="form-desc">{{ $t('model.editor.maxOutputTokensDesc') }}</p>
        </div>

        <!--
          Background concurrency cap for this model. Only chat / embedding / vllm
          are gated by the governor (see internal/models/limiter), so we surface
          it just for those three. 0 = fall back to the global default.
        -->
        <div v-if="['embedding', 'chat', 'vllm'].includes(activeModelType)" class="form-item">
          <label class="form-label">{{ $t('model.editor.maxConcurrencyLabel') }}</label>
          <t-input v-model.number="formData.maxConcurrency" type="number" :min="0" :max="4096"
            :placeholder="$t('model.editor.maxConcurrencyPlaceholder')" />
          <p class="form-desc">{{ $t('model.editor.maxConcurrencyDesc') }}</p>
        </div>

        <!--
          高级：协议覆盖 / 远端模型名 / 目录 compat 覆盖，以及仅旧数据才显示的
          thinking_control 兼容选择。默认折叠，绝大多数用户无需触碰。
        -->
        <template v-if="formData.source === 'remote' && formData.provider !== 'weknoracloud'">
          <button type="button" class="advanced-toggle" :aria-expanded="advancedOpen" @click="advancedOpen = !advancedOpen">
            <t-icon name="chevron-right" class="toggle-arrow" :class="{ open: advancedOpen }" />
            <span>{{ $t('model.editor.advanced.toggle') }}</span>
          </button>

          <template v-if="advancedOpen">
            <!--
              extra_config.api 只选对话协议。embedding / rerank 行不读它：
              embedding 的协议覆盖写在下面的 compat JSON 里（"api"），取值是向量协议。
            -->
            <div v-if="isChatLike" class="form-item">
              <label class="form-label">{{ $t('model.editor.advanced.api.label') }}</label>
              <t-select :model-value="formData.extraConfig.api || ''" clearable
                @update:model-value="(v: string) => setExtraConfig('api', v)">
                <t-option value="" :label="$t('model.editor.advanced.api.auto')" />
                <t-option v-for="api in PROTOCOL_OPTIONS" :key="api" :value="api" :label="api" />
              </t-select>
              <p class="form-desc">{{ $t('model.editor.advanced.api.desc') }}</p>
            </div>

            <div class="form-item">
              <label class="form-label">{{ $t('model.editor.advanced.remoteModelName.label') }}</label>
              <t-input :model-value="formData.extraConfig.remote_model_name || ''"
                :placeholder="$t('model.editor.advanced.remoteModelName.placeholder')"
                @update:model-value="(v: string) => setExtraConfig('remote_model_name', v)" />
              <p class="form-desc">{{ $t('model.editor.advanced.remoteModelName.desc') }}</p>
            </div>

            <!-- 仅旧数据：extra_config.thinking_control 存在时才显示，可清空改走目录默认 -->
            <div v-if="showLegacyThinkingControl" class="form-item">
              <label class="form-label">{{ $t('model.editor.advanced.legacyThinking.label') }}</label>
              <t-select :model-value="formData.thinkingControl || ''"
                @update:model-value="(v: string) => formData.thinkingControl = v">
                <t-option value="" :label="$t('model.editor.advanced.legacyThinking.catalog')" />
                <t-option v-for="opt in LEGACY_THINKING_CONTROL_VALUES" :key="opt" :value="opt"
                  :label="opt === 'none' ? $t('model.editor.advanced.legacyThinking.none') : opt" />
              </t-select>
              <p class="form-desc form-desc--warn">{{ $t('model.editor.advanced.legacyThinking.desc') }}</p>
            </div>

            <div class="form-item">
              <label class="form-label">{{ $t('model.editor.advanced.compat.label') }}</label>
              <t-textarea v-model="formData.specCompat" :autosize="{ minRows: 3, maxRows: 10 }"
                :placeholder="$t('model.editor.advanced.compat.placeholder')" class="compat-textarea"
                :status="specCompatError ? 'error' : undefined" />
              <p v-if="specCompatError" class="form-desc form-desc--error">{{ $t('model.editor.advanced.compat.invalid') }}: {{ specCompatError }}</p>
              <p v-else class="form-desc">
                {{ $t('model.editor.advanced.compat.desc') }}
                <a :href="COMPAT_DOC_URL" target="_blank" rel="noopener noreferrer" class="compat-doc-link">
                  {{ $t('model.editor.advanced.compat.docLink') }}
                  <t-icon name="jump" size="12px" />
                </a>
              </p>
            </div>
          </template>
        </template>
      </section>

    </t-form>
  </SettingDrawer>
</template>

<script setup lang="ts">
import { ref, watch, computed, onUnmounted, nextTick } from 'vue'
import { MessagePlugin, DialogPlugin } from 'tdesign-vue-next'
import {
  checkOllamaModels, checkRemoteModel, testEmbeddingModel, checkRerankModel, checkASRModel, listOllamaModels,
  downloadOllamaModel, getDownloadProgress, checkOllamaStatus, resolveModelCatalog,
  type OllamaModelInfo, type ModelProviderOption, type ModelProviderExtraField,
  type ModelProviderExtraFieldOption, type ModelCatalogEntry,
  type ResolvedModelCatalog,
} from '@/api/initialization'
import {
  getWeKnoraCloudStatus,
  putModelCredentials,
  deleteModelCredentialField,
  type ModelCredentialField,
  type ModelSpecOverride,
} from '@/api/model'
import { useI18n } from 'vue-i18n'
import { useUIStore } from '@/stores/ui'
import { useModelProvidersStore } from '@/stores/modelProviders'
import {
  credentialLabelForModelType,
  extraFieldLabel,
  extraFieldOptionLabel,
  extraFieldPlaceholder,
  extraFieldsForModelType,
  pickLocalized,
  providerDescription,
  providerIcon,
  providerLabel,
} from '@/stores/modelProvidersState'
import { levelLabelKey, supportedLevels } from '@/utils/reasoningEffort'
import { DEFAULT_MODEL_CONTEXT_WINDOW, formatTokenCount } from '@/utils/contextWindow'
import { copyWithToast } from '@/utils/clipboard'
import SettingDrawer from '@/components/settings/SettingDrawer.vue'
import CredentialResource, {
  type CredentialFieldDef,
  type CredentialResourceApi,
} from '@/components/credentials/CredentialResource.vue'
import { shouldShowOllamaUnavailableTip } from '@/components/modelEditorSourceState'
import { WEKNORA_CLOUD_PROVIDER, WKC_MODEL_KINDS, WKC_MODEL_NAME_BY_KIND } from '@/utils/weknoraCloudModels'
import { docsUrl } from '@/utils/docsUrl'

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
  /** Chat/VLM context window (tokens). Empty/0 means use the default 200000. */
  contextWindow?: number
  /** Concurrency limit for this model in background tasks; 0/undefined means fall back to the global default. Only applies to chat/embedding/vllm. */
  maxConcurrency?: number
  /** 对话/VLM 单次输出上限（token）。空/0 表示使用目录默认。 */
  maxOutputTokens?: number
  /**
   * Legacy extra_config.thinking_control (none | enable_thinking | thinking_type
   * | chat_template_kwargs). Only rows saved by older UIs carry it; the catalog
   * decides the encoding otherwise. Empty string = drop the key on save.
   */
  thinkingControl?: string
  /**
   * Provider-specific extra_config entries: vendor-declared extra fields
   * (api_version, region, ...) plus the advanced overrides `api` and
   * `remote_model_name`. thinking_control lives in its own field above.
   */
  extraConfig: Record<string, string>
  /** parameters.spec.compat as JSON text (validated before save). */
  specCompat?: string
  /** Other parameters.spec fields preserved verbatim from the loaded row. */
  spec?: ModelSpecOverride | null
  // Custom HTTP request headers (similar to extra_headers in the OpenAI Python SDK)
  customHeaders?: CustomHeaderItem[]
  /** Second credential (provider-declared secret extra field, e.g. SecretKey for LKEAP / Volcengine Rerank); written to app_secret on creation */
  appSecret?: string
}

/** Protocols the backend accepts in extra_config.api (internal/models/api.API). */
const PROTOCOL_OPTIONS = [
  'openai-completions',
  'openai-responses',
  'anthropic-messages',
  'google-generative-ai',
] as const

/** Field reference for parameters.spec.compat, per protocol and model type. */
const COMPAT_DOC_URL = docsUrl('modelsCompat')

/** Legacy thinking_control values still honoured by catalog.Resolve. */
const LEGACY_THINKING_CONTROL_VALUES = ['none', 'enable_thinking', 'thinking_type', 'chat_template_kwargs'] as const

/** Keys of extra_config that are edited by dedicated controls, not the generic renderer. */
const RESERVED_EXTRA_CONFIG_KEYS = new Set(['thinking_control'])

type EditorModelType = 'chat' | 'embedding' | 'rerank' | 'vllm' | 'asr'

interface Props {
  visible: boolean
  modelType: EditorModelType
  modelData?: ModelFormData | null
  saveModel: (data: ModelFormData & { modelType?: EditorModelType }) => Promise<void>
}

const { t, locale } = useI18n()
const uiStore = useUIStore()
const providersStore = useModelProvidersStore()

const props = withDefaults(defineProps<Props>(), {
  visible: false,
  modelData: null
})

const emit = defineEmits<{
  'update:visible': [value: boolean]
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

// The provider list comes entirely from the backend catalog (cached per model type in the store); the frontend no longer maintains any provider table.
const loadingProviders = computed(() => providersStore.isLoading(activeModelType.value))

const loadProviders = async (force = false) => {
  try {
    await providersStore.ensureLoaded(activeModelType.value, force)
  } catch (error) {
    console.error('Failed to load providers from API', error)
  }
}

const providerOptions = computed<ModelProviderOption[]>(() => providersStore.providersFor(activeModelType.value))

const selectedProvider = computed<ModelProviderOption | undefined>(() => {
  const id = formData.value.provider
  if (!id) return undefined
  return providerOptions.value.find(p => p.value === id) || providersStore.providerById(id)
})

const currentLocale = computed(() => String(locale.value || ''))
const providerDisplayLabel = (p: ModelProviderOption) => providerLabel(p, currentLocale.value)
const providerDisplayDescription = (p: ModelProviderOption) => providerDescription(p, currentLocale.value)
const selectedProviderIcon = computed(() => providerIcon(selectedProvider.value))
/** Localized vendor name, or the raw stored id when the catalog no longer has it. */
const selectedProviderDisplayLabel = computed(() => (
  selectedProvider.value
    ? providerDisplayLabel(selectedProvider.value)
    : (formData.value.provider || '')
))
/**
 * Pin a select's dropdown to the width of its input.
 *
 * TDesign sizes a select popup as max(popup content, trigger)
 * (select-input/hooks/useOverlayInnerStyle), so one long option — a vendor
 * whose description lists half a dozen model ids — stretches the whole menu
 * past the field it belongs to, leaving a wide band of empty space and
 * pushing the tick mark far from the text. Passing a function here replaces
 * that matching outright (the hook keeps a function as-is), and the option
 * rows already ellipsize, so the long ones simply truncate.
 */
const matchTriggerWidth = (triggerElement: HTMLElement) => ({
  width: `${triggerElement.offsetWidth}px`,
})

const extraFieldDisplayLabel = (field: ModelProviderExtraField) => extraFieldLabel(field, currentLocale.value)
const extraFieldDisplayPlaceholder = (field: ModelProviderExtraField) =>
  extraFieldPlaceholder(field, currentLocale.value)
const extraFieldDisplayOptionLabel = (option: ModelProviderExtraFieldOption) =>
  extraFieldOptionLabel(option, currentLocale.value)

/**
 * Vendors whose API is not a bearer-token API name their first credential
 * themselves (LKEAP rerank takes a SecretId, Volcengine rerank an Access Key
 * ID). Falling back to the generic "API Key" wording is what led operators to
 * paste an `sk-` token into a signature field.
 */
const credentialLabel = computed(() =>
  credentialLabelForModelType(selectedProvider.value?.credentialLabels, activeModelType.value),
)
const apiKeyLabel = computed(() => {
  const label = credentialLabel.value
  return label ? pickLocalized(label.labels, currentLocale.value, label.label) : t('model.editor.apiKeyOptional')
})
const apiKeyRequired = computed(
  () => credentialLabel.value?.required === true || (!!selectedProvider.value?.requiresAuth && !isEdit.value),
)
const apiKeyHint = computed(() => {
  const label = credentialLabel.value
  if (!label?.hint) return ''
  return pickLocalized(label.hints, currentLocale.value, label.hint)
})

// 厂商额外字段：按当前模型类型过滤；secret 字段走 app_secret 凭证，其余进 extra_config。
const visibleExtraFields = computed<ModelProviderExtraField[]>(() =>
  extraFieldsForModelType(selectedProvider.value?.extraFields, activeModelType.value)
    .filter(field => !RESERVED_EXTRA_CONFIG_KEYS.has(field.key)),
)
/**
 * A credential-bearing vendor field. `secret` is the backend's own marker;
 * `type: 'password'` is treated the same way so a vendor that forgets the flag
 * still cannot get its value echoed back from GET /models into extra_config.
 */
const isSecretExtraField = (field: ModelProviderExtraField) => field.secret === true || field.type === 'password'
// Only one credential slot (app_secret) exists per model, so the first such
// field wins; vendors today declare at most one (LKEAP / Volcengine rerank).
const secretExtraField = computed<ModelProviderExtraField | undefined>(() =>
  visibleExtraFields.value.find(isSecretExtraField),
)
const plainExtraFields = computed<ModelProviderExtraField[]>(() =>
  visibleExtraFields.value.filter(field => !isSecretExtraField(field)),
)

/** extra_config keys that belong to the connection, not to the vendor. */
const VENDOR_NEUTRAL_EXTRA_CONFIG_KEYS = ['api', 'remote_model_name'] as const

/** What survives a vendor (or model-type) switch: everything else is vendor-specific. */
const keepVendorNeutralExtraConfig = (): Record<string, string> => {
  const keep: Record<string, string> = {}
  for (const key of VENDOR_NEUTRAL_EXTRA_CONFIG_KEYS) {
    const value = formData.value.extraConfig?.[key]
    if (value) keep[key] = value
  }
  return keep
}

const setExtraConfig = (key: string, value: string | null | undefined) => {
  const next = { ...(formData.value.extraConfig || {}) }
  const normalized = value == null ? '' : String(value)
  if (normalized === '') delete next[key]
  else next[key] = normalized
  formData.value.extraConfig = next
}

const extraConfigBool = (key: string) => {
  const raw = (formData.value.extraConfig?.[key] || '').trim().toLowerCase()
  return raw === 'true' || raw === '1' || raw === 'yes'
}

/** Pre-fill vendor defaults for fields the user has not touched. */
const applyExtraFieldDefaults = () => {
  for (const field of plainExtraFields.value) {
    if (!field.default) continue
    if ((formData.value.extraConfig?.[field.key] ?? '') === '') {
      setExtraConfig(field.key, field.default)
    }
  }
}

// 内置模型目录 → 模型名下拉候选（可自由输入）
interface CatalogModelOption {
  value: string
  label: string
  contextWindow: string
  dimension?: number
  reasoning: boolean
  vision: boolean
  entry?: ModelCatalogEntry
}

const catalogEntries = computed<ModelCatalogEntry[]>(() => {
  const entries = selectedProvider.value?.models || []
  // Managed aliases are already used by the cloud setup page. They have no
  // published limits, so the backend catalog is empty, but they can still be
  // offered by name without inventing context windows or embedding dimensions.
  if (formData.value.provider === WEKNORA_CLOUD_PROVIDER && entries.length === 0) {
    const kind = WKC_MODEL_KINDS.find(kind => kind === activeModelType.value)
    if (kind) {
      const name = WKC_MODEL_NAME_BY_KIND[kind]
      return [{ id: name, name, type: kind === 'vllm' ? 'chat' : kind, input: kind === 'vllm' ? ['text', 'image'] : ['text'] }]
    }
  }
  // The list is already scoped: providers are fetched per model type, so the
  // backend returned exactly the entries that type can use. Filtering again
  // here on entry.type was wrong for 视觉 — a VLM entry is a chat model that
  // accepts images, so it arrives typed "chat" and every one of them was
  // dropped, leaving the picker empty for every vendor.
  return entries
})

const catalogModelOptions = computed<CatalogModelOption[]>(() => {
  const options: CatalogModelOption[] = catalogEntries.value.map(entry => ({
    value: entry.id,
    label: entry.name || entry.id,
    contextWindow: entry.context_window ? formatTokenCount(entry.context_window) : '',
    dimension: entry.dimension || undefined,
    reasoning: !!entry.reasoning || (entry.thinking_levels?.length ?? 0) > 0,
    vision: Array.isArray(entry.input) && entry.input.includes('image'),
    entry,
  }))
  const current = (formData.value.modelName || '').trim()
  if (current && !options.some(o => o.value === current)) {
    options.unshift({ value: current, label: current, contextWindow: '', reasoning: false, vision: false })
  }
  return options
})

const findCatalogEntry = (name: string) => catalogEntries.value.find(m => m.id === name)

/**
 * Where to read about what is configured right now.
 *
 * Prefer the page the selected model's facts were taken from — that is the
 * page listing its context window, thinking levels and price — and fall back
 * to the vendor's own site when the model is not in the catalog. Both come
 * from the catalog, so a new vendor gets the link without a UI change.
 */
const vendorDocLink = computed(() => {
  const provider = selectedProvider.value
  if (!provider) return ''
  const entry = findCatalogEntry((formData.value.modelName || '').trim())
  const url = entry?.source || provider.website || ''
  return /^https?:\/\//i.test(url) ? url : ''
})

/**
 * What applyCatalogEntry filled in for the model currently selected.
 *
 * Switching vendor has to take those values back — a context window from the
 * previous vendor's model is exactly the "填大会导致压缩不触发、上游直接拒绝"
 * case this field warns about — but it must not touch a number the operator
 * typed. Remembering what was filled, and only clearing a field that still
 * holds it, separates the two.
 */
const catalogFilled = ref<Partial<ModelFormData>>({})

/** Fill blank capability fields from a catalog entry the user just picked. */
const applyCatalogEntry = (entry: ModelCatalogEntry) => {
  if (activeModelType.value === 'chat' || activeModelType.value === 'vllm') {
    if (!formData.value.contextWindow && entry.context_window) {
      formData.value.contextWindow = entry.context_window
      catalogFilled.value.contextWindow = entry.context_window
    }
    if (!formData.value.maxOutputTokens && entry.max_output_tokens) {
      formData.value.maxOutputTokens = entry.max_output_tokens
      catalogFilled.value.maxOutputTokens = entry.max_output_tokens
    }
  }
  if (activeModelType.value === 'chat' && !formData.value.supportsVision && Array.isArray(entry.input) && entry.input.includes('image')) {
    formData.value.supportsVision = true
    catalogFilled.value.supportsVision = true
  }
  if (activeModelType.value === 'embedding' && !formData.value.dimension && entry.dimension) {
    formData.value.dimension = entry.dimension
    catalogFilled.value.dimension = entry.dimension
  }
}

const handleCatalogModelCreate = (value: string | number | boolean | bigint) => {
  formData.value.modelName = String(value ?? '').trim()
}

const handleCatalogModelChange = (value: unknown) => {
  const name = typeof value === 'string' ? value.trim() : ''
  if (!name) return
  const entry = findCatalogEntry(name)
  if (entry) applyCatalogEntry(entry)
}

// 接入诊断（只读）：厂商 / 模型名 / Base URL / 协议覆盖变化后 400ms 防抖调用目录解析
const resolved = ref<ResolvedModelCatalog | null>(null)
const resolving = ref(false)
const resolveFailed = ref(false)
/** Human-readable detail of the last failure; '' when the error carried none. */
const resolveError = ref('')
let resolveTimer: ReturnType<typeof setTimeout> | null = null
let resolveRevision = 0

/** 目录解析出的协议 / 思考等级 / 上下文窗口只对对话与 VLM 模型有意义。 */
const isChatLike = computed(() => activeModelType.value === 'chat' || activeModelType.value === 'vllm')

const showResolvedPanel = computed(() =>
  isChatLike.value
  && formData.value.source === 'remote'
  && !!formData.value.provider
  && formData.value.provider !== 'weknoracloud',
)

const resolvedThinkingLevels = computed(() => supportedLevels(resolved.value?.capabilities))

const runResolve = async () => {
  const revision = ++resolveRevision
  const provider = (formData.value.provider || '').trim()
  // Embedding / ReRank / ASR never render the panel, so do not spend a
  // request (and do not leave a stale chat result behind) for them.
  if (!props.visible || !showResolvedPanel.value || !provider) {
    resolved.value = null
    resolving.value = false
    resolveFailed.value = false
    resolveError.value = ''
    return
  }
  resolving.value = true
  try {
    const result = await resolveModelCatalog({
      provider,
      spec: buildSpec(),
      model: formData.value.modelName || '',
      base_url: formData.value.baseUrl || '',
      model_type: activeModelType.value,
      api: formData.value.extraConfig?.api || '',
      thinking_control: formData.value.thinkingControl || '',
      remote_model_name: formData.value.extraConfig?.remote_model_name || '',
      // Vendor fields decide the request too — Azure's api_version picks
      // between the v1 data plane and the dated deployments path — so the
      // preview has to see them or it describes a different request than
      // the one this row will make. Secret fields never travel in a query
      // string; plainExtraFields already excludes them.
      ...Object.fromEntries(
        plainExtraFields.value
          .map(field => [field.key, formData.value.extraConfig?.[field.key] || ''])
          .filter(([, value]) => !!value),
      ),
    })
    if (revision !== resolveRevision) return
    resolved.value = result
    resolveFailed.value = false
    resolveError.value = ''
  } catch (error: any) {
    if (revision !== resolveRevision) return
    resolved.value = null
    resolveFailed.value = true
    const detail = typeof error === 'string'
      ? error
      : (error?.message || error?.error?.message || error?.error)
    resolveError.value = typeof detail === 'string' ? detail.trim() : ''
  } finally {
    if (revision === resolveRevision) resolving.value = false
  }
}

const scheduleResolve = () => {
  if (resolveTimer) clearTimeout(resolveTimer)
  resolveTimer = setTimeout(() => {
    resolveTimer = null
    void runResolve()
  }, 400)
}

// 高级折叠区
const advancedOpen = ref(false)
/** Set when the loaded row carried thinking_control, so the select stays visible after clearing it. */
const legacyThinkingControlLoaded = ref(false)
const showLegacyThinkingControl = computed(() =>
  activeModelType.value === 'chat' && !!(formData.value.thinkingControl || legacyThinkingControlLoaded.value),
)

const specCompatError = computed(() => {
  const text = (formData.value.specCompat || '').trim()
  if (!text) return ''
  try {
    const parsed = JSON.parse(text)
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
      return t('model.editor.advanced.compat.mustBeObject')
    }
    return ''
  } catch (error: any) {
    return error?.message || 'invalid JSON'
  }
})

// Match the parent save path: retain row metadata, replace the edited compat.
const buildSpec = (): ModelSpecOverride => {
  if (specCompatError.value) throw new Error(specCompatError.value)
  const spec: ModelSpecOverride = { ...(formData.value.spec || {}) }
  const text = (formData.value.specCompat || '').trim()
  if (text) spec.compat = JSON.parse(text)
  else delete spec.compat
  return spec
}

const dialogVisible = computed({
  get: () => props.visible,
  set: (val) => {
    if (!saving.value) emit('update:visible', val)
  }
})

/** Populating the form from modelData; ignore programmatic change side effects from the vendor/source controls */
const hydratingForm = ref(false)

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

// Credential resource binding for the shared <CredentialResource> component.
// A vendor-declared secret extra field (LKEAP / Volcengine rerank SecretKey)
// is stored as the app_secret credential, so it shows up here in edit mode.
const credentialFields = computed<CredentialFieldDef<ModelCredentialField>[]>(() => {
  const fields: CredentialFieldDef<ModelCredentialField>[] = [
    { key: 'api_key', label: t('model.editor.apiKeyOptional') as string },
  ]
  if (formData.value.provider === 'weknoracloud') {
    fields.push({ key: 'app_secret', label: 'App Secret' })
  } else if (secretExtraField.value) {
    fields.push({ key: 'app_secret', label: extraFieldDisplayLabel(secretExtraField.value) })
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
const apiKeyPlaceholder = computed(() => {
  const label = credentialLabel.value
  if (label?.placeholder) return pickLocalized(label.placeholders, currentLocale.value, label.placeholder)
  return t('model.editor.apiKeyPlaceholder')
})

const formRef = ref()
const saving = ref(false)
const saveError = ref('')

// Settings itself listens on window for Escape. Capture it while saving so
// the parent cannot unmount this editor before the request finishes.
const handleSaveEscape = (event: KeyboardEvent) => {
  if (props.visible && saving.value && (event.key === 'Escape' || event.code === 'Escape')) {
    event.preventDefault()
    event.stopImmediatePropagation()
  }
}
watch(() => props.visible && saving.value, (locked) => {
  if (locked) window.addEventListener('keydown', handleSaveEscape, true)
  else window.removeEventListener('keydown', handleSaveEscape, true)
}, { flush: 'sync' })
// Toggles the create-mode API key input between masked and plain text. Lets
// the user proofread a freshly pasted secret without losing the password
// affordance for everyday use. Reset every time the drawer closes (see
// reset block in the visible watcher) so we never leak the previous value
// across editor sessions.
const modelChecked = ref(false)
const modelAvailable = ref(false)
const checking = ref(false)
const remoteChecked = ref(false)
const remoteAvailable = ref(false)
const remoteMessage = ref('')
const remoteStale = ref(false)
let connectionRevision = 0
let applyingDetectedDimension = false

// Invalidate pending responses too, even when the user changes a field back.
const invalidateConnectionTest = (showStale = true) => {
  connectionRevision++
  remoteStale.value = showStale && (remoteStale.value || checking.value || remoteChecked.value)
  checking.value = false
  remoteChecked.value = false
  remoteAvailable.value = false
  remoteMessage.value = ''
  dimensionChecked.value = false
  dimensionSuccess.value = false
  dimensionMessage.value = ''
}

const applyDetectedDimension = (dimension: number) => {
  applyingDetectedDimension = true
  try {
    formData.value.dimension = dimension
  } finally {
    applyingDetectedDimension = false
  }
}
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
  contextWindow: undefined,
  maxConcurrency: undefined,
  maxOutputTokens: undefined,
  thinkingControl: '',
  extraConfig: {},
  specCompat: '',
  spec: null,
  customHeaders: [],
  appSecret: '',
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

// Triggers for the integration diagnostics: visibility / source / provider / model name / Base URL / advanced overrides / model type
watch(
  () => [
    props.visible, formData.value.source, formData.value.provider, formData.value.modelName,
    formData.value.baseUrl, formData.value.extraConfig?.api, formData.value.extraConfig?.remote_model_name,
    formData.value.thinkingControl, activeModelType.value,
 formData.value.specCompat, JSON.stringify(formData.value.spec),
  ],
  () => {
    if (!props.visible) return
    scheduleResolve()
  },
)

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
  }
  remoteChecked.value = false
  remoteAvailable.value = false
  remoteMessage.value = ''

  await loadProviders()
  const supported = providerOptions.value.some(p => p.value === formData.value.provider)
  if (!supported) {
    formData.value.provider = 'generic'
    formData.value.baseUrl = ''
    // 只丢厂商相关的 extra_config；协议 / 远端模型名是用户对这次接入的选择，
    // 与厂商无关，和 handleProviderChange 保持同一套规则。
    formData.value.extraConfig = keepVendorNeutralExtraConfig()
    formData.value.appSecret = ''
  } else {
    handleProviderChange(formData.value.provider || 'generic')
  }
}

// Watch for visible changes and initialize the form
watch(() => props.visible, (val) => {
  if (val) {
    // Check Ollama service status
    checkOllamaServiceStatus()

    // Load the Model Provider list from the API (and fill in extra-field defaults for new rows).
    // Catalogs can be published by another administrator while this page is
    // open, so refresh this type without dropping other types' cached lists.
    loadProviders(true).then(() => {
      if (props.visible && !isEdit.value) applyExtraFieldDefaults()
    })
    advancedOpen.value = false

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
        const loadedExtra: Record<string, string> = {}
        for (const [key, value] of Object.entries(props.modelData.extraConfig || {})) {
          if (RESERVED_EXTRA_CONFIG_KEYS.has(key)) continue
          if (value != null && String(value) !== '') loadedExtra[key] = String(value)
        }
        const loadedSpec = props.modelData.spec || null
        formData.value = {
          ...props.modelData,
          apiKey: '',
          appSecret: '',
          extraConfig: loadedExtra,
          thinkingControl: props.modelData.thinkingControl || '',
          spec: loadedSpec,
          specCompat: props.modelData.specCompat
            ?? (loadedSpec?.compat ? JSON.stringify(loadedSpec.compat, null, 2) : ''),
          customHeaders: Array.isArray(props.modelData.customHeaders)
            ? props.modelData.customHeaders.map(h => ({ key: h.key, value: h.value }))
            : [],
        }
        legacyThinkingControlLoaded.value = !!props.modelData.thinkingControl
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
    } finally {
      nextTick(() => {
        hydratingForm.value = false
      })
    }
  }
})

watch(
  () => [
    activeModelType.value, props.modelData?.id, formData.value.source,
    formData.value.provider, formData.value.modelName, formData.value.baseUrl,
    formData.value.apiKey, formData.value.appSecret, formData.value.customHeaders,
    formData.value.dimension, formData.value.supportsDimensionOverride,
    formData.value.extraConfig, formData.value.thinkingControl,
 formData.value.specCompat, formData.value.spec,
  ],
  () => {
    if (!applyingDetectedDimension) invalidateConnectionTest(props.visible && !hydratingForm.value)
  },
  { deep: true, flush: 'sync' },
)

watch(() => props.visible, () => {
  invalidateConnectionTest(false)
  saveError.value = ''
}, { flush: 'sync' })

// Reset the form
const resetForm = () => {
  legacyThinkingControlLoaded.value = false
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
    contextWindow: undefined,
    maxConcurrency: undefined,
    maxOutputTokens: undefined,
    thinkingControl: '',
    extraConfig: {},
    specCompat: '',
    spec: null,
    customHeaders: [],
    appSecret: '',
  }
  resolved.value = null
  resolveFailed.value = false
  resolveError.value = ''
  modelChecked.value = false
  modelAvailable.value = false
  remoteChecked.value = false
  remoteAvailable.value = false
  remoteMessage.value = ''
  dimensionChecked.value = false
  dimensionSuccess.value = false
  dimensionMessage.value = ''
}

// Handle vendor selection changes (auto-fill default URL)
/**
 * Drop a model name the new vendor does not serve, along with whatever the
 * catalog filled in for it.
 *
 * The same id does exist at several vendors — deepseek-v4-pro is sold by
 * DeepSeek, Aliyun, Volcengine and the gateways — so switching between them
 * should keep the selection. Anything else is a name from the previous
 * vendor: left in place it is saved verbatim, resolves as an uncatalogued
 * model and fails at the first call.
 */
const resetModelSelectionForVendor = () => {
  const current = (formData.value.modelName || '').trim()
  if (!current || findCatalogEntry(current)) return

  formData.value.modelName = ''
  // Take back only the values applyCatalogEntry put there; a number the
  // operator typed is theirs and survives the switch.
  const filled = catalogFilled.value
  if (filled.contextWindow && formData.value.contextWindow === filled.contextWindow) {
    formData.value.contextWindow = undefined
  }
  if (filled.maxOutputTokens && formData.value.maxOutputTokens === filled.maxOutputTokens) {
    formData.value.maxOutputTokens = undefined
  }
  if (filled.dimension && formData.value.dimension === filled.dimension) {
    formData.value.dimension = undefined
  }
  if (filled.supportsVision && formData.value.supportsVision) {
    formData.value.supportsVision = false
  }
  catalogFilled.value = {}
  modelChecked.value = false
  modelAvailable.value = false
  dimensionChecked.value = false
  dimensionSuccess.value = false
  dimensionMessage.value = ''
}

const handleProviderChange = (value: string) => {
  const provider = providerOptions.value.find(opt => opt.value === value)
  if (provider?.defaultUrls) {
    // Get the corresponding default URL based on the current model type
    const defaultUrl = provider.defaultUrls[activeModelType.value]
    if (defaultUrl) {
      formData.value.baseUrl = defaultUrl
    }
  }
  // Reset validation status: it describes the previous vendor's connectivity, which is unrelated to the new vendor. Keep it outside the defaultUrls
  // check, otherwise switching to a vendor without a default URL would leave a stale "connection OK" result behind.
  remoteChecked.value = false
  remoteAvailable.value = false
  remoteMessage.value = ''
  // WeKnoraCloud: check credential status
  if (value === 'weknoracloud') {
    checkWkcCredentialStatus()
  }
  if (hydratingForm.value) return
  resetModelSelectionForVendor()
  // Switching vendor: drop the previous vendor's fields (keep advanced overrides such as protocol / remote model name), then fill in the new vendor's defaults
  formData.value.extraConfig = keepVendorNeutralExtraConfig()
  formData.value.appSecret = ''
  applyExtraFieldDefaults()
}

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
  if (checking.value || saving.value) return
  if (!formData.value.modelName || formData.value.source !== 'local' || activeModelType.value !== 'embedding') {
    return
  }

  const revision = ++connectionRevision
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

    if (revision !== connectionRevision) return
    dimensionChecked.value = true
    dimensionSuccess.value = result.available || false

    if (result.available && result.dimension) {
      applyDetectedDimension(result.dimension)
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
    if (revision !== connectionRevision) return
    console.error('Ollama dimension check failed:', error)
    dimensionChecked.value = true
    dimensionSuccess.value = false
    dimensionMessage.value = t('model.editor.dimensionFailed')
    MessagePlugin.error(dimensionMessage.value)
  } finally {
    if (revision === connectionRevision) checking.value = false
  }
}

/** extra_config exactly as it will be persisted: trimmed, empty values dropped. */
const buildExtraConfig = (): Record<string, string> => {
  const out: Record<string, string> = {}
  for (const [key, value] of Object.entries(formData.value.extraConfig || {})) {
    const trimmed = (value ?? '').toString().trim()
    if (key && trimmed) out[key] = trimmed
  }
  const legacy = (formData.value.thinkingControl || '').trim()
  if (legacy && activeModelType.value === 'chat' && formData.value.source === 'remote') {
    out.thinking_control = legacy
  }
  return out
}

// Check the Remote API connection (calls different interfaces depending on the model type)
const checkRemoteAPI = async () => {
  if (checking.value || saving.value) return
  if (!formData.value.modelName || (!formData.value.baseUrl && formData.value.provider !== 'weknoracloud')) {
    MessagePlugin.warning(t('model.editor.fillModelAndUrl'))
    return
  }

  const revision = ++connectionRevision
  checking.value = true
  remoteStale.value = false
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

    // extra_config 与真正保存时完全一致（厂商字段 + 高级覆盖 + 旧 thinking_control），
    // 使测试连接走与生产调用相同的目录解析路径。
    const extraConfig = buildExtraConfig()
    const extraPayload = {
      spec: buildSpec(),
      ...(Object.keys(extraConfig).length > 0 ? { extraConfig } : {}),
      ...(formData.value.appSecret?.trim() ? { appSecret: formData.value.appSecret.trim() } : {}),
    }

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
          ...extraPayload,
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
          ...extraPayload,
        })
        // If the test succeeds and returns a dimension, auto-fill it
        if (revision !== connectionRevision) return
        if (result.available && result.dimension) applyDetectedDimension(result.dimension)
        break

      case 'rerank':
        result = await checkRerankModel({
          modelName: formData.value.modelName,
          baseUrl: formData.value.baseUrl || '',
          apiKey: formData.value.apiKey || '',
          provider: formData.value.provider,
          ...idPayload,
          ...headerPayload,
          ...extraPayload,
        })
        break

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
          ...extraPayload,
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
          ...extraPayload,
        })
        break

      default:
        MessagePlugin.error(t('model.editor.unsupportedModelType'))
        return
    }

    if (revision !== connectionRevision) return
    remoteChecked.value = true
    remoteAvailable.value = result.available || false
    remoteMessage.value = result.available
      ? t('model.editor.connectionSuccess')
      : result.message || t('model.editor.connectionFailed')
  } catch (error: any) {
    if (revision !== connectionRevision) return
    remoteChecked.value = true
    remoteAvailable.value = false
    remoteMessage.value = error?.message || t('model.editor.connectionConfigError')
  } finally {
    if (revision === connectionRevision) checking.value = false
  }
}

// Confirm and save
const handleConfirm = async () => {
  if (saving.value) return
  saving.value = true
  if (checking.value) invalidateConnectionTest()
  saveError.value = ''
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

    if (formData.value.source === 'remote' && formData.value.provider !== 'weknoracloud') {
      // Provider-declared required extra fields
      for (const field of plainExtraFields.value) {
        if (field.required && !(formData.value.extraConfig?.[field.key] || '').trim()) {
          MessagePlugin.warning(t('model.editor.validation.extraFieldRequired', { name: extraFieldDisplayLabel(field) }))
          return
        }
      }
      if (!isEdit.value && secretExtraField.value?.required && !(formData.value.appSecret || '').trim()) {
        MessagePlugin.warning(t('model.editor.validation.extraFieldRequired', { name: extraFieldDisplayLabel(secretExtraField.value) }))
        return
      }
      if (specCompatError.value) {
        advancedOpen.value = true
        MessagePlugin.warning(`${t('model.editor.advanced.compat.invalid')}: ${specCompatError.value}`)
        return
      }
    }

    // Run form validation
    const validation = await formRef.value?.validate()
    if (validation !== undefined && validation !== true) return

    // Credential removal in edit mode is handled inline by the
    // CredentialResource card (it confirms + DELETEs to /credentials), so
    // the main save flow no longer needs to confirm or handle clear flags.

    // If adding new and no id, generate one
    if (!formData.value.id) {
      formData.value.id = generateId()
    }

    await props.saveModel({
      ...formData.value,
      extraConfig: buildExtraConfig(),
      ...(isEdit.value ? {} : { modelType: activeModelType.value }),
    })
    emit('update:visible', false)
    invalidateConnectionTest(false)
    // Reset draft after successful save, so the next new-model dialog starts blank
    resetForm()
    lastOpenedModelId.value = null
    // Remove the success toast here, handled centrally by the parent component
  } catch (error: any) {
    saveError.value = error?.message || t('modelSettings.toasts.saveFailed')
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
  window.removeEventListener('keydown', handleSaveEscape, true)
  invalidateConnectionTest(false)
  if (resolveTimer) {
    clearTimeout(resolveTimer)
    resolveTimer = null
  }
  resolveRevision++
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
})

// Watch for model name changes and clear dimension-detection state
watch(() => formData.value.modelName, () => {
  dimensionChecked.value = false
  dimensionSuccess.value = false
  dimensionMessage.value = ''
})

// Cancel (triggered by clicking the bottom "Cancel" button; clicking the overlay/ESC does not trigger it, preserving the draft)
const handleCancel = () => {
  if (saving.value) return
  resetForm()
  lastOpenedModelId.value = null
  dialogVisible.value = false
}
</script>

<style lang="less" scoped>
.provider-doc-link {
  margin-top: 6px;
}

.provider-doc-link a,
.compat-doc-link {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  color: var(--td-text-color-link);
  text-decoration: none;

  &:hover {
    text-decoration: underline;
  }
}

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
  font-size: var(--app-text-md);
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

.model-type-options,
.source-options {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

// 「模型类型」和「模型来源」都是单选，就用同一种按钮。模型来源原本是灰底轨道
// 的 segmented：#e7e7e7 的轨道在浅色表单里是一整块深灰，白色药丸又浮不起来，
// 而且紧挨着的模型类型是另一套长相。取消轨道后两组自然成为一族。
.model-type-option,
.source-option {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-family: inherit;
  padding: 6px 12px;
  min-height: 32px;
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-md);
  background: var(--td-bg-color-container);
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-md);
  line-height: 1.4;
  cursor: pointer;
  transition: border-color var(--app-motion-fast) ease, color var(--app-motion-fast) ease, background var(--app-motion-fast) ease;

  &__icon {
    font-size: var(--app-text-lg);
    flex-shrink: 0;
  }

  &__label {
    white-space: nowrap;
  }

  &:hover:not(.is-active) {
    border-color: var(--td-brand-color-3);
    color: var(--td-text-color-primary);
  }

  // 选中态与下面的「模型来源」分段一致：白底 + 主题色描边 + 主题色文字。
  // 原先还铺了一层 10% 的主题色底，五个按钮里那一块是整屏最重的色块，而且
  // 和下拉里刚去掉的整行绿底是同一个毛病。
  &.is-active {
    border-color: var(--td-brand-color);
    background: var(--td-bg-color-container);
    color: var(--td-brand-color);
    font-weight: 500;
  }

  &:focus-visible {
    outline: 2px solid var(--td-brand-color);
    outline-offset: 2px;
  }
}



.source-option {
  // 外观来自上面那条共用规则；这里只补禁用态（Ollama 未就绪 / rerank 不支持）。
  &.is-disabled {
    cursor: not-allowed;
    opacity: 0.45;
  }
}

// Input styles: only adjust font size on the outer .t-input, avoid adding borders again on the inner wrap/inner
// and border-radius, which would create the visual illusion of "nested rounded containers"
:deep(.t-input),
:deep(.t-select),
:deep(.t-textarea),
:deep(.t-input-number) {
  width: 100%;
  font-size: var(--app-text-md);
}

// Vendor selector styles — moved to a non-scoped block, since the t-select popup renders under body
// See .provider-option styles at the end of the file

// Checkbox
:deep(.t-checkbox) {
  font-size: var(--app-text-md);

  .t-checkbox__label {
    font-size: var(--app-text-md);
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
  border-radius: var(--app-radius-md);

  .test-message {
    font-size: var(--app-text-md);
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
    font-size: var(--app-text-md);
    border-radius: var(--app-radius-sm);
    flex-shrink: 0;
  }

  .status-icon {
    font-size: var(--app-text-xl);
    flex-shrink: 0;

    &.available {
      color: var(--td-brand-color);
    }

    &.unavailable {
      color: var(--td-error-color);
    }
  }
}

.connection-status {
  font-size: var(--app-text-sm);
  &.success { color: var(--td-brand-color-active); }
  &.error { color: var(--td-error-color); }
}

.connection-hint {
  margin: 0 0 8px;
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-sm);
  line-height: 1.5;
}

.connection-result {
  margin-bottom: 8px;
  padding: 10px 12px;
  border: 1px solid var(--td-error-color-3);
  border-radius: var(--td-radius-default);
  background: var(--td-error-color-1);
  color: var(--td-error-color);
  font-size: var(--app-text-sm);
  text-align: left;

  &__header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 8px;
  }

  &__details {
    max-height: min(160px, 20vh);
    overflow: auto;
    white-space: pre-wrap;
    overflow-wrap: anywhere;
    line-height: 1.5;
    user-select: text;
  }
}

// Status icon variant used inside the footer button.
.status-icon {
  font-size: var(--app-text-xl);
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
  border-radius: var(--app-radius-md);
  font-size: var(--app-text-md);
  color: var(--td-text-color-secondary);
  line-height: 1.5;

  // Theming via tokens so the warn/ok states track light/dark switches
  // instead of fighting hardcoded `#fff7ed` etc.
  &--ok {
    background: var(--td-success-color-light);
    border: 1px solid var(--td-success-color-focus);
  }

  &--warn {
    background: var(--td-warning-color-light);
    border: 1px solid var(--td-warning-color-focus);
    border-left: 3px solid var(--td-warning-color);
  }

  .hint-icon {
    font-size: var(--app-text-xl);
    flex-shrink: 0;
    margin-top: 2px;

    &--ok {
      color: var(--td-success-color);
    }

    &--warn {
      color: var(--td-warning-color);
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
    font-size: var(--app-text-base);
    color: var(--td-brand-color);
    flex-shrink: 0;
  }

  .download-icon {
    font-size: var(--app-text-base);
    color: var(--td-brand-color);
    flex-shrink: 0;
  }

  .model-name {
    flex: 1;
    font-size: var(--app-text-md);
    color: var(--td-text-color-primary);
  }

  .model-size {
    font-size: var(--app-text-sm);
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
    animation: wk-spin 1s linear infinite;
    font-size: var(--app-text-base);
    color: var(--td-brand-color);
  }

  .progress-text {
    font-size: var(--app-text-sm);
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
      background: linear-gradient(90deg, color-mix(in srgb, var(--td-brand-color) 8%, transparent), color-mix(in srgb, var(--td-brand-color) 15%, transparent));
      transition: width var(--app-motion-slow) ease;
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
  font-size: var(--app-text-md);
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
  font-size: var(--app-text-sm);
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
    border-radius: var(--app-radius-sm);
    transition: all 0.18s ease;

    &:hover {
      background: var(--td-error-color-light);
      color: var(--td-error-color);
    }
  }
}

.form-desc {
  margin: 4px 0 0 0;
  font-size: var(--app-text-sm);
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

  &--error {
    color: var(--td-error-color);
  }
}

// 已选厂商：图标 + 名称，嵌在 t-select 的 valueDisplay 里
.provider-value {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  max-width: 100%;

  &__name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.provider-icon {
  width: 18px;
  height: 18px;
  flex-shrink: 0;
  border-radius: 4px;
  object-fit: contain;

  &--placeholder {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    font-size: var(--app-text-xs);
    font-weight: 600;
    color: var(--td-text-color-secondary);
    background: var(--td-bg-color-secondarycontainer);
  }
}

// 目录内 / 推理 / 视觉 等弱化徽标（与卡片上的 chip 同调）
.catalog-badge {
  display: inline-flex;
  align-items: center;
  padding: 0 6px;
  height: 18px;
  border-radius: 4px;
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-secondary);
  font-size: var(--app-text-xs);
  line-height: 18px;
  white-space: nowrap;

  & + & {
    margin-left: 4px;
  }

  &--accent {
    background: var(--td-brand-color-light);
    color: var(--td-brand-color);
  }
}

// 接入诊断面板：只读、浅底，避免与可编辑字段混淆
.resolved-panel {
  padding: 10px 12px;
  border: 1px dashed var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-container-hover);

  &__header {
    display: flex;
    align-items: center;
    gap: 6px;
    margin-bottom: 6px;
  }

  &__icon {
    font-size: var(--app-text-base);
    color: var(--td-text-color-secondary);
  }

  &__title {
    font-size: var(--app-text-sm);
    font-weight: 500;
    color: var(--td-text-color-secondary);
  }

  &__loading {
    font-size: var(--app-text-md);
    color: var(--td-text-color-placeholder);
    animation: spin 1s linear infinite;
  }

  &__grid {
    display: grid;
    grid-template-columns: max-content minmax(0, 1fr);
    column-gap: 12px;
    row-gap: 4px;
    margin: 0;
    font-size: var(--app-text-sm);

    dt {
      color: var(--td-text-color-placeholder);
      white-space: nowrap;
      line-height: 18px;
    }

    dd {
      margin: 0;
      min-width: 0;
      color: var(--td-text-color-primary);
      line-height: 18px;
      overflow-wrap: anywhere;
    }

    code {
      font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
      font-size: var(--app-text-sm);
    }
  }
}

// 高级折叠：与知识库分块设置里的 advanced-toggle 保持同一视觉
.advanced-toggle {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 0;
  margin: 0;
  background: transparent;
  border: none;
  cursor: pointer;
  font-family: inherit;
  font-size: var(--app-text-md);
  font-weight: 500;
  color: var(--td-text-color-secondary);
  user-select: none;

  &:hover {
    color: var(--td-text-color-primary);
  }

  &:focus-visible {
    outline: 2px solid var(--td-brand-color-focus);
    outline-offset: 2px;
    border-radius: 4px;
  }

  .toggle-arrow {
    font-size: var(--app-text-base);
    transition: transform 0.15s ease;

    &.open {
      transform: rotate(90deg);
    }
  }
}

.compat-textarea {
  :deep(textarea) {
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    font-size: var(--app-text-sm);
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
  border-radius: var(--app-radius-md);
  font-size: var(--app-text-md);

  .tip-icon {
    color: var(--td-error-color);
    font-size: var(--app-text-xl);
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
    font-size: var(--app-text-md);
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
    border-radius: var(--app-radius-xs);
    transition: all var(--app-motion-base) ease;

    &:hover {
      background: color-mix(in srgb, var(--td-brand-color) 8%, transparent) !important;
      color: var(--td-brand-color-active) !important;
    }

    &:active {
      background: color-mix(in srgb, var(--td-brand-color) 12%, transparent) !important;
    }

    .t-icon {
      font-size: var(--app-text-base) !important;
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
    font-size: var(--app-text-md);
  }
}
</style>

<!-- Non-scoped style: t-select popup renders under body, scoped styles can't override it -->
<style lang="less">
.catalog-model-select-popup {
  padding: 4px;

  .t-select-option {
    height: auto !important;
    padding: 8px 10px;
    border-radius: var(--app-radius-sm);
    margin: 2px 0;
  }
}

.catalog-model-option {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  width: 100%;

  &__name {
    font-size: var(--app-text-md);
    color: var(--td-text-color-primary);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__id {
    font-size: var(--app-text-xs);
    color: var(--td-text-color-placeholder);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  &__badges {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    margin-left: auto;
    flex-shrink: 0;

    .catalog-badge {
      display: inline-flex;
      align-items: center;
      padding: 0 6px;
      height: 18px;
      border-radius: 4px;
      background: var(--td-bg-color-secondarycontainer);
      color: var(--td-text-color-secondary);
      font-size: var(--app-text-xs);
      line-height: 18px;
      white-space: nowrap;

      &--accent {
        background: var(--td-brand-color-light);
        color: var(--td-brand-color);
      }
    }
  }
}

.provider-select-popup {
  // Give the container some breathing room: keep options from touching the popup's rounded corners
  padding: 4px;

  // TDesign hard-codes max-height: 300px on the select popup, but the options here are two lines
  // (about 56px), so exactly five show — there are twenty-odd more vendors below, but macOS overlay
  // scrollbars stay invisible until you scroll, so it looks like "that's all there is". Raise it so eight or nine fit
  // on screen and keep the scrollbar visible, so it is obvious the list continues. &.wk-popover only exists to beat
  // TDesign's own selector, which has the same two-level specificity.
  &.wk-popover .t-popup__content {
    max-height: min(480px, 60vh);
    // Scrolling itself is restored for every skinned select in
    // assets/theme/tdesign-overrides.less; this only makes the bar visible,
    // because macOS overlay scrollbars stay hidden until something moves and
    // a capped list then looks complete.
    scrollbar-color: var(--td-component-border) transparent;
    scrollbar-width: thin;
  }

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
    border-radius: var(--app-radius-sm);
    margin: 2px 0;
    outline: none;
    transition: background-color var(--app-motion-fast) ease;

    &:focus,
    &:focus-visible {
      outline: none;
    }


  }

  // The selected state is not defined here: assets/theme/tdesign-overrides.less already replaces TDesign's
  // default full-row --td-brand-color-light fill with a neutral background + a small theme-colored
  // check mark, consistent across the whole site. Defining another version here would only make this dropdown
  // mismatch the "Model type / Model source" filter rows. The selected row's main name still gets a touch of the theme color, as an anchor in the two-line layout.
  .t-select-option.t-is-selected .provider-name {
    color: var(--td-brand-color);
  }

  .provider-option {
    display: flex;
    align-items: center;
    gap: 10px;
    width: 100%;
    min-width: 0;

    .provider-icon {
      width: 18px;
      height: 18px;
      flex-shrink: 0;
      border-radius: 4px;
      object-fit: contain;

      &--placeholder {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        font-size: var(--app-text-xs);
        font-weight: 600;
        color: var(--td-text-color-secondary);
        background: var(--td-bg-color-secondarycontainer);
      }
    }

    &__text {
      display: flex;
      flex-direction: column;
      gap: 2px;
      // 允许收缩，描述才能在浮层宽度内省略（浮层宽度由 matchTriggerWidth 钉死）。
      min-width: 0;
      flex: 1;
    }

    .provider-name {
      font-size: var(--app-text-md);
      font-weight: 500;
      color: var(--td-text-color-primary);
      line-height: 20px;
    }

    .provider-desc {
      font-size: var(--app-text-sm);
      color: var(--td-text-color-placeholder);
      line-height: 18px;
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
    }
  }
}
</style>
