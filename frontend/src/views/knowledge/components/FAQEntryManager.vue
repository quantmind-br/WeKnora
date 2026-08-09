<template>
  <div class="faq-manager">
    <div class="faq-content">
      <!-- Header -->
      <div class="faq-header">
        <div class="faq-header-title">
          <div class="faq-title-row">
            <h2 class="faq-breadcrumb">
              <button type="button" class="breadcrumb-link" @click="handleNavigateToKbList">
                {{ $t('menu.knowledgeBase') }}
              </button>
              <t-icon name="chevron-right" class="breadcrumb-separator" />
              <KBSwitcherDropdown
                v-if="knowledgeList.length"
                :kb-list="knowledgeList"
                :current-kb-id="props.kbId"
                @select="(id) => handleKnowledgeDropdownSelect({ value: id })"
              >
                <button type="button" class="breadcrumb-link dropdown" :disabled="!props.kbId">
                  <template v-if="!kbInfo">
                    <t-skeleton animation="gradient" :row-col="[{ width: '120px', height: '20px' }]" />
                  </template>
                  <template v-else>
                    <span>{{ kbInfo.name }}</span>
                    <t-icon name="chevron-down" />
                  </template>
                </button>
              </KBSwitcherDropdown>
              <button v-else type="button" class="breadcrumb-link" :disabled="!props.kbId"
                @click="handleNavigateToCurrentKB">
                <template v-if="!kbInfo">
                  <t-skeleton animation="gradient" :row-col="[{ width: '120px', height: '20px' }]" />
                </template>
                <template v-else>
                  {{ kbInfo.name }}
                </template>
              </button>
              <t-icon name="chevron-right" class="breadcrumb-separator" />
              <span class="breadcrumb-current">{{ $t('knowledgeEditor.faq.title') }}</span>
            </h2>
            <div class="kb-title-actions">
              <KBInfoPopover
                v-if="kbInfo && !authStore.isLiteMode"
                :kb-info="kbInfo"
              />
              <t-tooltip v-if="canManage" :content="$t('knowledgeBase.settings')" placement="top">
                <button type="button" class="kb-settings-button" @click="handleOpenKBSettings">
                  <t-icon name="setting" size="16px" />
                </button>
              </t-tooltip>
              <!-- Import results: icon-only by default, hover / click to expand details -->
              <div v-if="showImportResultBadge" class="faq-import-host"
                :class="{ 'is-expanded': importResultExpanded }">
                <button type="button" class="faq-import-trigger"
                  :aria-label="$t('faqManager.import.recentResult')"
                  @click.stop="importResultExpanded = !importResultExpanded">
                  <t-icon name="check-circle-filled" size="16px" />
                </button>
                <div class="faq-import-panel">
                  <div class="faq-import-strip faq-import-strip--result faq-import-strip--panel">
                    <span class="faq-import-strip__text">{{ importResultSummary }}</span>
                    <t-tag size="small" variant="light"
                      :theme="importResult!.import_mode === 'append' ? 'primary' : 'warning'">
                      {{ importResult!.import_mode === 'append' ? $t('faqManager.import.appendMode') :
                        $t('faqManager.import.replaceMode') }}
                    </t-tag>
                    <t-button v-if="importResult!.failed_entries_url && importResult!.failed_count > 0"
                      variant="text" theme="danger" size="small" class="faq-import-strip__link"
                      @click="downloadFailedEntries">
                      {{ $t('faqManager.import.downloadReasons') }}
                    </t-button>
                    <span class="faq-import-strip__time">{{ formatImportTime(importResult!.imported_at) }}</span>
                    <button type="button" class="faq-import-strip__close" :aria-label="$t('common.close')"
                      @click="closeImportResult">
                      <t-icon name="close" size="14px" />
                    </button>
                  </div>
                </div>
              </div>
              <!-- Import in progress -->
              <div v-else-if="isImportInProgress && importState.taskStatus"
                class="faq-import-strip faq-import-strip--in-title"
                :class="`faq-import-strip--${importState.taskStatus.status}`">
                <t-icon :name="importProgressIcon" size="16px" class="faq-import-strip__icon"
                  :class="{ 'is-spinning': importState.taskStatus.status === 'running' }" />
                <span class="faq-import-strip__text">{{ importProgressText }}</span>
                <div class="faq-import-strip__bar">
                  <div class="faq-import-strip__bar-fill" :style="{ width: `${importState.taskStatus.progress}%` }" />
                </div>
                <span class="faq-import-strip__count">{{ importState.taskStatus.processed }}/{{
                  importState.taskStatus.total }}</span>
              </div>
            </div>
          </div>
          <p class="faq-subtitle">{{ $t('knowledgeEditor.faq.subtitle') }}</p>
        </div>
      </div>

      <div class="faq-main">
        <div class="faq-card-area">
          <!-- Search bar and tag filters -->
          <div class="faq-filter-bar">
            <t-input v-model.trim="entrySearchKeyword" :placeholder="$t('knowledgeEditor.faq.searchPlaceholder')"
              clearable class="faq-search-input" @clear="loadEntries()" @enter="loadEntries()">
              <template #prefix-icon>
                <t-icon name="search" size="16px" />
              </template>
            </t-input>
            <div class="faq-filter-bar__filters">
              <t-popup v-model:visible="tagFilterPanelVisible" trigger="click" placement="bottom-left"
                overlay-class-name="tag-filter-popup" :overlay-inner-style="{ padding: 0 }">
                <template #content>
                  <div class="tag-filter-panel" @click.stop>
                    <div class="tag-filter-panel__header">
                      <div class="tag-filter-panel__title">
                        <span>{{ $t('knowledgeBase.tagFilterTitle') }}</span>
                        <span class="tag-filter-panel__count">({{ sidebarCategoryCount }})</span>
                      </div>
                    </div>
                    <div class="tag-search-bar">
                      <t-input v-model.trim="tagSearchQuery" size="small"
                        :placeholder="$t('knowledgeBase.tagSearchPlaceholder')" clearable>
                        <template #prefix-icon>
                          <t-icon name="search" size="14px" />
                        </template>
                      </t-input>
                    </div>
                    <div class="tag-filter-panel__body">
                      <template v-if="tagLoading && !sidebarTags.length">
                        <div class="tag-filter-chips">
                          <div v-for="n in 8" :key="'skel-tag-' + n" class="tag-filter-chip-skeleton">
                            <t-skeleton animation="gradient"
                              :row-col="[{ width: '56px', height: '24px', type: 'rect' }]" />
                          </div>
                        </div>
                      </template>
                      <template v-else>
                        <div class="tag-filter-chips">
                          <button
                            v-for="tag in sidebarTags"
                            :key="tag.id"
                            type="button"
                            class="tag-filter-chip"
                            :class="{ active: isTagFilterActive(tag.id) }"
                            :title="`${tag.name} (${tag.chunk_count || 0})`"
                            @click="handleTagRowClick(tag.id)"
                          >
                            <span class="tag-filter-chip__label">{{ tag.name }}</span>
                            <span class="tag-filter-chip__count">{{ tag.chunk_count || 0 }}</span>
                          </button>
                        </div>
                        <div v-if="!sidebarTags.length" class="tag-empty-state">
                          {{ $t('knowledgeBase.tagEmptyResult') }}
                        </div>
                        <div v-if="tagHasMore" class="tag-load-more">
                          <t-button variant="text" size="small" :loading="tagLoadingMore" @click.stop="loadTags()">
                            {{ $t('tenant.loadMore') }}
                          </t-button>
                        </div>
                      </template>
                    </div>
                    <div v-if="canEdit" class="tag-filter-panel__footer">
                      <t-button variant="text" size="small" class="tag-manage-link" @click="openTagManageDrawer">
                        {{ $t('knowledgeBase.tagManageLink') }}
                      </t-button>
                    </div>
                  </div>
                </template>
                <div class="doc-filter-field">
                  <button type="button" class="doc-tag-filter-trigger doc-filter-field__control"
                    :class="{ open: tagFilterPanelVisible, 'is-placeholder': isTagFilterPlaceholder }"
                    :aria-label="$t('knowledgeBase.tagFilterTitle')"
                    :title="activeTagFilterTitle"
                    @mouseenter="tagFilterTriggerHover = true"
                    @mouseleave="tagFilterTriggerHover = false">
                    <span class="doc-tag-filter-trigger__prefix" aria-hidden="true">
                      <t-icon name="discount" size="16px" />
                    </span>
                    <span class="doc-tag-filter-trigger__label">{{ activeTagFilterLabel }}</span>
                    <span class="doc-tag-filter-trigger__suffix">
                      <span
                        v-if="showTagFilterClear"
                        class="t-input__suffix t-input__suffix-icon t-input__clear"
                        :aria-label="$t('common.clear')"
                        @click.stop="clearTagFilter"
                        @mousedown.stop
                      >
                        <t-icon name="close-circle-filled" class="t-input__suffix-clear" />
                      </span>
                      <t-icon
                        v-else
                        name="chevron-down"
                        size="16px"
                        class="doc-tag-filter-trigger__caret"
                        :class="{ open: tagFilterPanelVisible }"
                      />
                    </span>
                  </button>
                </div>
              </t-popup>
            </div>
            <div class="faq-filter-bar__trailing">
              <!-- Create: create group (new entry / import) -->
              <template v-if="faqCreateOptions.length">
                <t-tooltip :content="$t('knowledgeEditor.faq.createGroup')" placement="top">
                  <t-dropdown :options="faqCreateOptions" trigger="click" placement="bottom-right"
                    @click="handleFaqAction">
                    <t-button variant="text" theme="default" class="content-bar-icon-btn" size="small">
                      <template #icon><t-icon name="add" size="16px" /></template>
                    </t-button>
                  </t-dropdown>
                </t-tooltip>
              </template>
              <!-- Export -->
              <t-dropdown :options="faqExportOptions" trigger="click" placement="bottom-right"
                @click="handleFaqAction">
                <t-tooltip :content="$t('knowledgeEditor.faqExport.exportButton')" placement="top">
                  <t-button variant="text" theme="default" class="content-bar-icon-btn" size="small"
                    :loading="exportLoading">
                    <template #icon><t-icon name="download" size="16px" /></template>
                  </t-button>
                </t-tooltip>
              </t-dropdown>
              <!-- Search -->
              <t-tooltip :content="$t('knowledgeEditor.faq.searchTest')" placement="top">
                <t-button variant="text" theme="default" class="content-bar-icon-btn" size="small"
                  @click="handleFaqAction({ value: 'search' })">
                  <template #icon><t-icon name="search" size="16px" /></template>
                </t-button>
              </t-tooltip>
            </div>
          </div>
          <!-- Card List Container with Scroll -->
          <div ref="scrollContainer" class="faq-scroll-container" @scroll="handleScroll">
            <!-- FAQ skeleton screen -->
            <div v-if="loading && entries.length === 0" class="faq-skeleton-grid">
              <div v-for="n in 6" :key="'faq-skel-' + n" class="faq-card faq-card-skeleton">
                <div class="faq-card-header">
                  <t-skeleton animation="gradient" :row-col="[{ width: '80%', height: '16px' }]" />
                </div>
                <div class="faq-card-body">
                  <t-skeleton animation="gradient"
                    :row-col="[{ width: '100%', height: '13px' }, { width: '90%', height: '13px' }, { width: '60%', height: '13px' }]" />
                </div>
                <div class="faq-skel-footer">
                  <t-skeleton animation="gradient"
                    :row-col="[[{ width: '50px', height: '18px', type: 'rect' }, { width: '60px', height: '18px', type: 'rect' }]]" />
                </div>
              </div>
            </div>
            <!-- Card List -->
            <template v-else-if="entries.length > 0">
              <div ref="cardListRef" class="faq-card-list">
                <div v-for="entry in entries" :key="entry.id" class="faq-card"
                  :class="{ 'selected': selectedRowKeys.includes(entry.id) }"
                  @click="handleCardSelect(entry.id, !selectedRowKeys.includes(entry.id))">
                  <!-- Card Header -->
                  <div class="faq-card-header">
                    <div class="faq-header-top">
                      <div class="faq-question" :title="entry.standard_question">
                        {{ entry.standard_question }}
                      </div>
                      <div class="faq-card-actions">
                        <t-popup v-if="canManage" v-model="entry.showMore" overlayClassName="card-more-popup"
                          trigger="click" destroy-on-close placement="bottom-right"
                          @visible-change="(visible: boolean) => (entry.showMore = visible)">
                          <div class="card-more-btn" @click.stop>
                            <img class="more-icon" src="@/assets/img/more.png" alt="" />
                          </div>
                          <template #content>
                            <div class="popup-menu" @click.stop>
                              <div class="popup-menu-item" @click.stop="handleMenuEdit(entry)">
                                <t-icon class="menu-icon" name="edit" />
                                <span>{{ $t('common.edit') }}</span>
                              </div>
                              <div class="popup-menu-item delete" @click.stop="handleMenuDelete(entry)">
                                <t-icon class="menu-icon" name="delete" />
                                <span>{{ $t('common.delete') }}</span>
                              </div>
                            </div>
                          </template>
                        </t-popup>
                      </div>
                    </div>
                  </div>

                  <!-- Card Body -->
                  <div class="faq-card-body">
                    <!-- Similar Questions Section -->
                    <div v-if="entry.similar_questions?.length" class="faq-section similar">
                      <div class="faq-section-label clickable"
                        @click.stop="entry.similarCollapsed = !entry.similarCollapsed">
                        <span>{{ $t('knowledgeEditor.faq.similarQuestions') }}</span>
                        <span class="section-count">
                          ({{ entry.similar_questions.length }})
                        </span>
                        <t-icon :name="entry.similarCollapsed ? 'chevron-right' : 'chevron-down'"
                          class="collapse-icon" />
                      </div>
                      <Transition name="slide-down">
                        <div v-if="!entry.similarCollapsed" class="faq-tags">
                          <FAQTagTooltip v-for="question in entry.similar_questions" :key="question" :content="question"
                            type="similar" placement="top">
                            <t-tag size="small" variant="light-outline" class="question-tag">
                              {{ question }}
                            </t-tag>
                          </FAQTagTooltip>
                        </div>
                      </Transition>
                    </div>

                    <!-- Negative Questions Section -->
                    <div v-if="entry.negative_questions?.length" class="faq-section negative">
                      <div class="faq-section-label clickable"
                        @click.stop="entry.negativeCollapsed = !entry.negativeCollapsed">
                        <span>{{ $t('knowledgeEditor.faq.negativeQuestions') }}</span>
                        <span class="section-count">
                          ({{ entry.negative_questions.length }})
                        </span>
                        <t-icon :name="entry.negativeCollapsed ? 'chevron-right' : 'chevron-down'"
                          class="collapse-icon" />
                      </div>
                      <Transition name="slide-down">
                        <div v-if="!entry.negativeCollapsed" class="faq-tags">
                          <FAQTagTooltip v-for="question in entry.negative_questions" :key="question"
                            :content="question" type="negative" placement="top">
                            <t-tag size="small" theme="warning" variant="light-outline" class="question-tag">
                              {{ question }}
                            </t-tag>
                          </FAQTagTooltip>
                        </div>
                      </Transition>
                    </div>

                    <!-- Answers Section -->
                    <div class="faq-section answers">
                      <div class="faq-section-label clickable"
                        @click.stop="entry.answersCollapsed = !entry.answersCollapsed">
                        <span>{{ $t('knowledgeEditor.faq.answers') }}</span>
                        <span v-if="entry.answers?.length" class="section-count">
                          ({{ entry.answers.length }})
                        </span>
                        <t-icon :name="entry.answersCollapsed ? 'chevron-right' : 'chevron-down'"
                          class="collapse-icon" />
                      </div>
                      <Transition name="slide-down">
                        <div v-if="!entry.answersCollapsed" class="faq-tags">
                          <FAQTagTooltip v-for="answer in entry.answers" :key="answer" :content="answer" type="answer"
                            placement="top">
                            <t-tag size="small" theme="success" variant="light-outline" class="question-tag">
                              {{ answer }}
                            </t-tag>
                          </FAQTagTooltip>
                        </div>
                      </Transition>
                    </div>
                  </div>

                  <!-- Card Footer -->
                  <div class="faq-card-footer">
                    <div class="faq-card-tag" @click.stop>
                      <template v-if="canEdit && tagList.length">
                        <t-dropdown :options="tagDropdownOptions" trigger="click"
                          @click="(data: any) => handleEntryTagChange(entry.id, data.value as string)">
                          <t-tag size="small" variant="light-outline" class="faq-tag-chip">
                            <span class="tag-text">{{ getTagName(entry.tag_id) || $t('knowledgeBase.untagged') }}</span>
                          </t-tag>
                        </t-dropdown>
                      </template>
                      <template v-else>
                        <t-tag size="small" variant="light-outline" class="faq-tag-chip">
                          <span class="tag-text">{{ getTagName(entry.tag_id) || $t('knowledgeBase.untagged') }}</span>
                        </t-tag>
                      </template>
                    </div>
                    <div class="faq-card-status" @click.stop>
                      <!-- Temporarily hide the recommendation toggle
                      <t-tooltip
                        :content="entry.is_recommended ? $t('knowledgeEditor.faq.recommendedEnabled') : $t('knowledgeEditor.faq.recommendedDisabled')"
                        placement="top"
                      >
                        <div class="status-item-compact">
                          <t-switch
                            :key="`${entry.id}-recommended-${entry.is_recommended}`"
                            size="small"
                            :value="entry.is_recommended"
                            :loading="!!entryRecommendedLoading[entry.id]"
                            :disabled="!!entryRecommendedLoading[entry.id]"
                            @click.stop
                            @change="(value: boolean) => handleEntryRecommendedChange(entry, value)"
                          />
                          <span class="status-label">{{ $t('knowledgeEditor.faq.recommended') }}</span>
                        </div>
                      </t-tooltip>
                      -->
                      <t-tooltip
                        :content="entry.is_enabled ? $t('knowledgeEditor.faq.statusEnabled') : $t('knowledgeEditor.faq.statusDisabled')"
                        placement="top">
                        <div class="status-item-compact">
                          <t-switch :key="`${entry.id}-${entry.is_enabled}`" size="small" :value="entry.is_enabled"
                            :loading="!!entryStatusLoading[entry.id]"
                            :disabled="!!entryStatusLoading[entry.id] || !canEdit" @click.stop
                            @change="(value: boolean) => handleEntryStatusChange(entry, value)" />
                        </div>
                      </t-tooltip>
                    </div>
                  </div>
                </div>
              </div>
            </template>
            <template v-else-if="!loading">
              <div class="faq-empty-state">
                <div class="empty-content">
                  <t-icon name="file-add" size="48px" class="empty-icon" />
                  <div class="empty-text">{{ $t('knowledgeEditor.faq.emptyTitle') }}</div>
                  <div class="empty-desc">{{ $t('knowledgeEditor.faq.emptyDesc') }}</div>
                </div>
              </div>
            </template>
            <div v-if="loadingMore" class="faq-load-more">
              <t-loading size="small" :text="$t('common.loading')" />
            </div>
            <div v-if="hasMore === false && entries.length > 0" class="faq-no-more">
              {{ $t('common.noMoreData') }}
            </div>
          </div>
        </div>
      </div>
    </div>
    <!-- Editor Drawer -->
    <t-drawer v-model:visible="editorVisible"
      :header="editorMode === 'create' ? $t('knowledgeEditor.faq.editorCreate') : $t('knowledgeEditor.faq.editorEdit')"
      :close-btn="true" size="520px" placement="right" class="faq-editor-drawer" @close="handleEditorClose">
      <div class="faq-editor-drawer-content">
        <t-form ref="editorFormRef" :data="editorForm" :rules="editorRules" layout="vertical" :label-width="0"
          class="faq-editor-form">
          <div class="settings-group">
            <!-- Standard question -->
            <div class="setting-row vertical setting-row-primary">
              <div class="setting-info">
                <label class="required-label">
                  {{ $t('knowledgeEditor.faq.standardQuestion') }}
                  <span class="required-mark">*</span>
                </label>
                <p class="desc">{{ $t('knowledgeEditor.faq.standardQuestionDesc') }}</p>
              </div>
              <div class="setting-control">
                <t-input v-model="editorForm.standard_question" :maxlength="200" class="full-width-input" />
              </div>
            </div>

            <!-- Similar questions -->
            <div class="setting-row vertical setting-row-optional setting-row-similar">
              <div class="setting-info">
                <label class="optional-label">{{ $t('knowledgeEditor.faq.similarQuestions') }}</label>
                <p class="desc optional-desc">{{ $t('knowledgeEditor.faq.similarQuestionsDesc') }}</p>
              </div>
              <div class="setting-control">
                <div class="full-width-input-wrapper">
                  <t-input v-model="similarInput" :placeholder="$t('knowledgeEditor.faq.similarPlaceholder')"
                    @enter="addSimilar" class="full-width-input" />
                  <t-button theme="primary" variant="outline"
                    :disabled="!similarInput.trim() || editorForm.similar_questions.length >= 10" @click="addSimilar"
                    class="add-item-btn" size="small">
                    <t-icon name="add" size="16px" />
                  </t-button>
                </div>
                <div v-if="editorForm.similar_questions.length > 0" class="item-list">
                  <div v-for="(question, index) in editorForm.similar_questions" :key="index" class="item-row">
                    <div class="item-content">{{ question }}</div>
                    <t-button theme="default" variant="text" size="small" @click="removeSimilar(index)"
                      class="remove-item-btn">
                      <t-icon name="close" size="16px" />
                    </t-button>
                  </div>
                </div>
              </div>
            </div>

            <!-- Counterexamples -->
            <div class="setting-row vertical setting-row-optional setting-row-negative">
              <div class="setting-info">
                <label class="optional-label">{{ $t('knowledgeEditor.faq.negativeQuestions') }}</label>
                <p class="desc optional-desc">{{ $t('knowledgeEditor.faq.negativeQuestionsDesc') }}</p>
              </div>
              <div class="setting-control">
                <div class="full-width-input-wrapper">
                  <t-input v-model="negativeInput" :placeholder="$t('knowledgeEditor.faq.negativePlaceholder')"
                    @enter="addNegative" class="full-width-input" />
                  <t-button theme="primary" variant="outline"
                    :disabled="!negativeInput.trim() || editorForm.negative_questions.length >= 10" @click="addNegative"
                    class="add-item-btn" size="small">
                    <t-icon name="add" size="16px" />
                  </t-button>
                </div>
                <div v-if="editorForm.negative_questions.length > 0" class="item-list">
                  <div v-for="(question, index) in editorForm.negative_questions" :key="index"
                    class="item-row negative">
                    <div class="item-content">{{ question }}</div>
                    <t-button theme="default" variant="text" size="small" @click="removeNegative(index)"
                      class="remove-item-btn">
                      <t-icon name="close" size="16px" />
                    </t-button>
                  </div>
                </div>
              </div>
            </div>

            <!-- Answer -->
            <div class="setting-row vertical setting-row-primary setting-row-answer">
              <div class="setting-info">
                <label class="required-label">
                  {{ $t('knowledgeEditor.faq.answers') }}
                  <span class="required-mark">*</span>
                </label>
                <p class="desc">{{ $t('knowledgeEditor.faq.answersDesc') }}</p>
              </div>
              <div class="setting-control">
                <div class="textarea-container">
                  <div class="full-width-input-wrapper textarea-wrapper">
                    <t-textarea v-model="answerInput" :placeholder="$t('knowledgeEditor.faq.answerPlaceholder')"
                      :autosize="{ minRows: 3, maxRows: 6 }" class="full-width-textarea" @keydown.ctrl.enter="addAnswer"
                      @keydown.meta.enter="addAnswer" />
                    <t-button theme="primary" variant="outline"
                      :disabled="!answerInput.trim() || editorForm.answers.length >= 5" @click="addAnswer"
                      class="add-item-btn" size="small">
                      <t-icon name="add" size="16px" />
                    </t-button>
                  </div>
                  <div class="item-count">{{ editorForm.answers.length }}/5</div>
                </div>
                <div v-if="editorForm.answers.length > 0" class="item-list">
                  <div v-for="(answer, index) in editorForm.answers" :key="index" class="item-row answer-row">
                    <div class="item-content">{{ answer }}</div>
                    <t-button theme="default" variant="text" size="small" @click="removeAnswer(index)"
                      class="remove-item-btn">
                      <t-icon name="close" size="16px" />
                    </t-button>
                  </div>
                </div>
              </div>
            </div>

            <div class="setting-row vertical">
              <div class="setting-info">
                <label>{{ $t('knowledgeBase.tagLabel') }}</label>
                <p class="desc">{{ $t('knowledgeEditor.faq.tagDesc') }}</p>
              </div>
              <div class="setting-control">
                <t-select v-model="editorForm.tag_id" class="full-width-input" :options="tagSelectOptions" clearable
                  :placeholder="$t('knowledgeEditor.faq.tagPlaceholder')" />
              </div>
            </div>
          </div>
        </t-form>
      </div>

      <template #footer>
        <div class="faq-editor-drawer-footer">
          <t-button theme="default" variant="outline" @click="editorVisible = false">
            {{ $t('common.cancel') }}
          </t-button>
          <t-button theme="primary" @click="handleSubmitEntry" :loading="savingEntry">
            {{ editorMode === 'create' ? $t('knowledgeEditor.faq.editorCreate') : $t('common.save') }}
          </t-button>
        </div>
      </template>
    </t-drawer>

    <!-- Import Dialog -->
    <Teleport to="body">
      <Transition name="modal">
        <div v-if="importVisible" class="faq-import-overlay" @click.self="importVisible = false">
          <div class="faq-import-modal">
            <!-- Close button -->
            <button class="close-btn" @click="importVisible = false" :aria-label="$t('general.close')">
              <svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor">
                <path d="M15 5L5 15M5 5L15 15" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
              </svg>
            </button>

            <div class="faq-import-container">
              <div class="faq-import-header">
                <h2 class="import-title">{{ $t('knowledgeEditor.faqImport.title') }}</h2>
              </div>

              <div class="faq-import-content">
                <!-- Import mode selection -->
                <div class="import-form-item">
                  <label class="import-form-label required">{{ $t('knowledgeEditor.faqImport.modeLabel') }}</label>
                  <t-radio-group v-model="importState.mode" class="import-radio-group">
                    <t-radio-button value="append">{{ $t('knowledgeEditor.faqImport.appendMode') }}</t-radio-button>
                    <t-radio-button value="replace">{{ $t('knowledgeEditor.faqImport.replaceMode') }}</t-radio-button>
                  </t-radio-group>
                </div>

                <!-- File upload area -->
                <div class="import-form-item">
                  <div class="file-label-row">
                    <label class="import-form-label required">{{ $t('knowledgeEditor.faqImport.fileLabel') }}</label>
                    <t-dropdown :options="downloadExampleOptions" placement="bottom-right" trigger="click"
                      @click="handleDownloadExample" class="download-example-dropdown">
                      <t-button theme="default" variant="outline" size="small" class="download-example-btn">
                        <t-icon name="download" size="16px" />
                        <span>{{ $t('knowledgeEditor.faqImport.downloadExample') }}</span>
                      </t-button>
                    </t-dropdown>
                  </div>
                  <div class="file-upload-wrapper">
                    <input ref="fileInputRef" type="file" accept=".json,.csv,.xlsx,.xls" @change="handleFileChange"
                      class="file-input-hidden" />
                    <div class="file-upload-area" :class="{ 'has-file': importState.file }"
                      @click="fileInputRef?.click()" @dragover.prevent @dragenter.prevent
                      @drop.prevent="handleFileDrop">
                      <div class="file-upload-content">
                        <t-icon name="upload" size="32px" class="upload-icon" />
                        <div class="upload-text">
                          <span v-if="!importState.file" class="upload-primary-text">
                            {{ $t('knowledgeEditor.faqImport.clickToUpload') }}
                          </span>
                          <span v-else class="upload-file-name">
                            {{ importState.file.name }}
                          </span>
                          <span v-if="!importState.file" class="upload-secondary-text">
                            {{ $t('knowledgeEditor.faqImport.dragDropTip') }}
                          </span>
                        </div>
                      </div>
                    </div>
                    <p class="import-form-tip">{{ $t('knowledgeEditor.faqImport.fileTip') }}</p>
                  </div>
                </div>

                <!-- Preview area -->
                <div v-if="importState.preview.length" class="import-preview">
                  <div class="preview-header">
                    <t-icon name="file-view" size="16px" class="preview-icon" />
                    <span class="preview-title">
                      {{ $t('knowledgeEditor.faqImport.previewCount', { count: importState.preview.length }) }}
                    </span>
                  </div>
                  <div class="preview-list">
                    <div v-for="(item, index) in importState.preview.slice(0, 5)" :key="index" class="preview-item">
                      <span class="preview-index">{{ index + 1 }}</span>
                      <span class="preview-question">{{ item.standard_question }}</span>
                    </div>
                  </div>
                  <p v-if="importState.preview.length > 5" class="preview-more">
                    {{ $t('knowledgeEditor.faqImport.previewMore', { count: importState.preview.length - 5 }) }}
                  </p>
                </div>

              </div>

              <div class="faq-import-footer">
                <t-button theme="default" variant="outline" @click="handleCancelImport"
                  :disabled="importState.importing && importState.taskStatus?.status === 'running'">
                  {{ $t('common.cancel') }}
                </t-button>
                <t-button theme="primary" @click="handleImport" :loading="importState.importing && !importState.taskId"
                  :disabled="importState.taskStatus?.status === 'running'">
                  {{ importState.taskStatus?.status === 'success' ? $t('common.close') :
                    importState.taskStatus?.status === 'failed' ? $t('common.retry') :
                      $t('knowledgeEditor.faqImport.importButton') }}
                </t-button>
              </div>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>

    <!-- Batch Tag Dialog -->
    <Teleport to="body">
      <Transition name="modal">
        <div v-if="batchTagDialogVisible" class="batch-tag-overlay" @click.self="batchTagDialogVisible = false">
          <div class="batch-tag-modal">
            <!-- Close button -->
            <button class="batch-tag-close-btn" @click="batchTagDialogVisible = false"
              :aria-label="$t('general.close')">
              <svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor">
                <path d="M15 5L5 15M5 5L15 15" stroke="currentColor" stroke-width="2" stroke-linecap="round" />
              </svg>
            </button>

            <div class="batch-tag-container">
              <div class="batch-tag-header">
                <h2 class="batch-tag-title">{{ $t('knowledgeEditor.faq.batchUpdateTag') }}</h2>
              </div>

              <div class="batch-tag-content">
                <div class="batch-tag-tip">
                  <t-icon name="info-circle" size="16px" class="tip-icon" />
                  <span>{{ $t('knowledgeEditor.faq.batchUpdateTagTip', { count: selectedRowKeys.length }) }}</span>
                </div>
                <t-form layout="vertical" class="batch-tag-form">
                  <t-form-item :label="$t('knowledgeBase.tagLabel')">
                    <t-select v-model="batchTagValue" :options="tagSelectOptions"
                      :placeholder="$t('knowledgeBase.tagPlaceholder')" clearable filterable class="batch-tag-select">
                      <template #empty>
                        <div class="tag-select-empty">
                          {{ $t('knowledgeBase.noTags') }}
                        </div>
                      </template>
                    </t-select>
                  </t-form-item>
                </t-form>
              </div>

              <div class="batch-tag-footer">
                <t-button theme="default" variant="outline" @click="batchTagDialogVisible = false">
                  {{ $t('common.cancel') }}
                </t-button>
                <t-button theme="primary" @click="handleBatchTag">
                  {{ $t('common.confirm') }}
                </t-button>
              </div>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>

    <!-- Search Test Drawer -->
    <t-drawer v-model:visible="searchDrawerVisible" :header="$t('knowledgeEditor.faq.searchTestTitle')"
      :close-btn="true" size="420px" placement="right" class="faq-search-drawer">
      <div class="search-test-content">
        <t-form layout="vertical" class="search-form" :label-width="0">
          <div class="settings-group">
            <!-- Query text -->
            <div class="setting-row vertical search-first-row">
              <div class="setting-info">
                <label>{{ $t('knowledgeEditor.faq.queryLabel') }}</label>
                <p class="desc">{{ $t('knowledgeEditor.faq.queryPlaceholder') }}</p>
              </div>
              <div class="setting-control">
                <t-input v-model="searchForm.query" :placeholder="$t('knowledgeEditor.faq.queryPlaceholder')"
                  @enter="handleSearch" class="full-width-input" />
              </div>
            </div>

            <!-- Similarity threshold -->
            <div class="setting-row vertical">
              <div class="setting-info">
                <label>{{ $t('knowledgeEditor.faq.similarityThresholdLabel') }}</label>
                <p class="desc">{{ $t('knowledgeEditor.faq.vectorThresholdDesc') }}</p>
              </div>
              <div class="setting-control">
                <div class="slider-wrapper">
                  <t-slider v-model="searchForm.vectorThreshold" :min="0" :max="1" :step="0.1" :show-tooltip="true"
                    :format-tooltip="(val: number) => val.toFixed(2)" />
                  <div class="slider-value">{{ searchForm.vectorThreshold.toFixed(2) }}</div>
                </div>
              </div>
            </div>

            <!-- Match count -->
            <div class="setting-row vertical">
              <div class="setting-info">
                <label>{{ $t('knowledgeEditor.faq.matchCountLabel') }}</label>
                <p class="desc">{{ $t('knowledgeEditor.faq.matchCountDesc') }}</p>
              </div>
              <div class="setting-control">
                <div class="slider-wrapper">
                  <t-slider v-model="searchForm.matchCount" :min="1" :max="50" :step="1" :show-tooltip="true" />
                  <div class="slider-value">{{ searchForm.matchCount }}</div>
                </div>
              </div>
            </div>

            <!-- Search button -->
            <div class="setting-row vertical">
              <div class="setting-control">
                <t-button theme="primary" block :loading="searching" @click="handleSearch" class="search-button">
                  {{ searching ? $t('knowledgeEditor.faq.searching') : $t('knowledgeEditor.faq.searchButton') }}
                </t-button>
              </div>
            </div>
          </div>
        </t-form>

        <!-- Search Results -->
        <div v-if="searchResults.length > 0 || hasSearched" class="search-results">
          <div class="results-header">
            <span>{{ $t('knowledgeEditor.faq.searchResults') }} ({{ searchResults.length }})</span>
          </div>
          <div v-if="searchResults.length === 0" class="no-results">
            {{ $t('knowledgeEditor.faq.noResults') }}
          </div>
          <div v-else class="results-list">
            <div v-for="(result, index) in searchResults" :key="result.id" class="result-card"
              :class="{ 'expanded': result.expanded }">
              <div class="result-header" @click="toggleResult(result)">
                <div class="result-question-wrapper">
                  <div class="result-main">
                    <div class="result-question">
                      <span class="result-index">{{ index + 1 }}.</span>
                      {{ result.standard_question }}
                    </div>
                    <div v-if="result.matched_question && result.matched_question !== result.standard_question"
                      class="matched-question">
                      <span class="matched-label">{{ $t('knowledgeEditor.faq.matchedQuestion') }}:</span>
                      <span class="matched-text">{{ result.matched_question }}</span>
                    </div>
                  </div>
                  <div class="result-meta">
                    <t-tag size="small" variant="light-outline" class="score-tag">
                      {{ (result.score || 0).toFixed(3) }}
                    </t-tag>
                  </div>
                  <t-icon :name="result.expanded ? 'chevron-up' : 'chevron-down'" class="expand-icon" />
                </div>
              </div>
              <Transition name="slide-down">
                <div v-if="result.expanded" class="result-body">
                  <div v-if="result.answers?.length" class="result-section">
                    <div class="section-label">{{ $t('knowledgeEditor.faq.answers') }}</div>
                    <div class="result-tags">
                      <t-tooltip v-for="answer in result.answers" :key="answer" :content="answer" placement="top">
                        <t-tag size="small" theme="success" variant="light" class="answer-tag">
                          {{ answer }}
                        </t-tag>
                      </t-tooltip>
                    </div>
                  </div>
                  <div v-if="result.similar_questions?.length" class="result-section">
                    <div class="section-label">{{ $t('knowledgeEditor.faq.similarQuestions') }}</div>
                    <div class="result-tags">
                      <t-tooltip v-for="question in result.similar_questions" :key="question" :content="question"
                        placement="top">
                        <t-tag size="small" variant="light-outline" class="question-tag">
                          {{ question }}
                        </t-tag>
                      </t-tooltip>
                    </div>
                  </div>
                </div>
              </Transition>
            </div>
          </div>
        </div>
      </div>
    </t-drawer>

    <KbTagManageDrawer
      v-model:visible="tagManageDrawerVisible"
      :kb-id="props.kbId"
      :is-faq="true"
      @changed="onTagManageChanged"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, watch, onMounted, computed, nextTick, onUnmounted, h } from 'vue'
import { MessagePlugin, DialogPlugin, Icon as TIcon } from 'tdesign-vue-next'
import type { FormRules, FormInstanceFunctions } from 'tdesign-vue-next'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { useOrganizationStore } from '@/stores/organization'
import {
  listFAQEntries,
  upsertFAQEntries,
  createFAQEntry,
  updateFAQEntry,
  updateFAQEntryFieldsBatch,
  deleteFAQEntries,
  searchFAQEntries,
  exportFAQEntries,
  listKnowledgeTags,
  updateFAQEntryTagBatch,
  getKnowledgeBaseById,
  listKnowledgeBases,
  getFAQImportProgress,
  updateFAQImportResultDisplayStatus,
} from '@/api/knowledge-base'
import * as XLSX from 'xlsx'
import Papa from 'papaparse'
import FAQTagTooltip from '@/components/FAQTagTooltip.vue'
import KBInfoPopover from '@/components/KBInfoPopover.vue'
import KBSwitcherDropdown from '@/components/KBSwitcherDropdown.vue'
import KbTagManageDrawer from './KbTagManageDrawer.vue'
import { useUIStore } from '@/stores/ui'

interface FAQEntry {
  id: number
  chunk_id: string
  knowledge_id: string
  knowledge_base_id: string
  tag_id?: number
  is_enabled: boolean
  is_recommended: boolean
  standard_question: string
  similar_questions: string[]
  negative_questions: string[]
  answers: string[]
  updated_at: string
  showMore?: boolean
  score?: number
  match_type?: string
  matched_question?: string
  expanded?: boolean
  similarCollapsed?: boolean
  negativeCollapsed?: boolean
  answersCollapsed?: boolean
}

interface FAQEntryPayload {
  standard_question: string
  similar_questions: string[]
  negative_questions: string[]
  answers: string[]
  tag_id?: number
  tag_name?: string
  is_enabled?: boolean
  is_recommended?: boolean
}

const props = defineProps<{
  kbId: string
}>()

const { t } = useI18n()
const router = useRouter()
const uiStore = useUIStore()
const authStore = useAuthStore()
const orgStore = useOrganizationStore()

// Permission control: check if current user owns this KB or has edit/manage permission.
//
// isOwner used to compare kbInfo.tenant_id against the user's effective tenant id,
// which silently treated "any KB visible to me in my current tenant" as "I created
// it" — Viewer / Contributor in their home tenant ended up showing every FAQ
// CRUD entry on every KB and 403'ing when they clicked. Mirror the rule we settled
// on in KnowledgeBase.vue: explicit creator_id match, with role / org-share fallbacks
// inside canEdit / canManage. Legacy KBs with empty creator_id stay tenant-owned
// (Admin+ may manage).
const isOwner = computed(() => {
  if (!kbInfo.value) return false
  const creatorId = (kbInfo.value as any).creator_id || ''
  const userId = authStore.user?.id || ''
  if (!creatorId) return false
  return creatorId === userId
})

// Current KB's shared record (when accessed via organization share)
const currentSharedKb = computed(() =>
  orgStore.sharedKnowledgeBases.find((s) => s.knowledge_base?.id === props.kbId) ?? null,
)

// Accessed via organization share: presence in the sharedKnowledgeBases list
// means we reached this KB through a shared space, so the user's local tenant
// role is irrelevant — only the share grant counts. tenant_id comparison
// alone is unreliable (a user can be a member of both source and receiving
// tenants); share-list presence is the authoritative signal.
const isViaShare = computed(() => !!currentSharedKb.value)

// Can edit: when accessed via an organization share, ONLY the share grant
// counts — even if the current user happens to be the original creator of
// the KB. The backend's RBAC middleware authorizes based on the active
// tenant, not on creator_id, so a creator viewing their own KB from a
// different tenant context will be 403'd on write. Otherwise: KB creator
// (any role) or tenant Admin+ in the home tenant.
const canEdit = computed(() => {
  if (isViaShare.value) return orgStore.canEditKB(props.kbId, false)
  if (isOwner.value) return true
  if (authStore.hasRole('admin')) return true
  return orgStore.canEditKB(props.kbId, false)
})

// Can manage (delete, settings, share): same isViaShare-first rule. For
// shared KBs only an 'admin' share grant qualifies — editor/viewer (and
// even being the creator viewed via share) never grant delete/settings.
const canManage = computed(() => {
  if (isViaShare.value) return orgStore.canManageKB(props.kbId, false)
  if (isOwner.value) return true
  if (authStore.hasRole('admin')) return true
  return orgStore.canManageKB(props.kbId, false)
})

const faqExportOptions = computed(() => [
  { content: t('knowledgeEditor.faqExport.exportCSV'), value: 'export_csv' },
  { content: t('knowledgeEditor.faqExport.exportJSON'), value: 'export_json' },
])

// FAQ actions: create group (new entry + import)
const faqCreateOptions = computed(() => {
  if (!canEdit.value) return []
  return [
    { content: t('knowledgeEditor.faq.editorCreate'), value: 'create', prefixIcon: () => h(TIcon, { name: 'add', size: '16px' }) },
    { content: t('knowledgeEditor.faqImport.importButton'), value: 'import', prefixIcon: () => h(TIcon, { name: 'upload', size: '16px' }) },
  ]
})

// Handle FAQ actions
const handleFaqAction = (data: { value: string }) => {
  switch (data.value) {
    case 'create':
      openEditor()
      break
    case 'import':
      openImportDialog()
      break
    case 'search':
      searchDrawerVisible.value = true
      break
    case 'export_csv':
      handleExportCSV()
      break
    case 'export_json':
      handleExportJSON()
      break
    case 'export':
      handleExportCSV()
      break
  }
}

const loading = ref(true)
const loadingMore = ref(false)
const entries = ref<FAQEntry[]>([])
const entryStatusLoading = reactive<Record<number, boolean>>({})
const entryRecommendedLoading = reactive<Record<number, boolean>>({})
const selectedRowKeys = ref<number[]>([])
const scrollContainer = ref<HTMLElement | null>(null)
const cardListRef = ref<HTMLElement | null>(null)
const hasMore = ref(true)
const pageSize = 20
let currentPage = 1
const entrySearchKeyword = ref('')
let entrySearchDebounce: number | null = null

const tagList = ref<any[]>([])
const tagLoading = ref(false)
const selectedTagIds = ref<string[]>([])
const tagFilterPanelVisible = ref(false)
const tagFilterTriggerHover = ref(false)
const tagFilterCleared = ref(false)
const tagManageDrawerVisible = ref(false)
const overallFAQTotal = ref(0)
const tagSearchQuery = ref('')
const TAG_PAGE_SIZE = 20
const tagPage = ref(1)
const tagHasMore = ref(false)
const tagLoadingMore = ref(false)
const tagTotal = ref(0)
let tagSearchDebounce: number | null = null

const showTagFilterClear = computed(
  () => selectedTagIds.value.length > 0 && tagFilterTriggerHover.value,
)

const isTagFilterPlaceholder = computed(
  () => selectedTagIds.value.length === 0 && tagFilterCleared.value,
)

const tagMap = computed<Record<string, any>>(() => {
  const map: Record<string, any> = {}
  tagList.value.forEach((tag) => {
    map[tag.id] = tag
  })
  return map
})

// tagMapBySeqId uses seq_id as key for looking up by entry.tag_id
const tagMapBySeqId = computed<Record<number, any>>(() => {
  const map: Record<number, any> = {}
  tagList.value.forEach((tag) => {
    map[tag.seq_id] = tag
  })
  return map
})

const regularTags = computed(() => tagList.value)
const tagDropdownOptions = computed(() =>
  regularTags.value.map((tag: any) => ({ content: tag.name, value: String(tag.seq_id) })),
)
const tagSelectOptions = computed(() =>
  regularTags.value.map((tag: any) => ({ label: tag.name, value: tag.seq_id })),
)

const sidebarCategoryCount = computed(() => tagTotal.value || tagList.value.length)
const sidebarTags = computed(() => {
  const list = tagList.value
  const selectedIds = selectedTagIds.value
  if (!selectedIds.length) {
    return list
  }
  const missingSelected = selectedIds
    .filter((id) => !list.some((tag) => tag.id === id))
    .map((id) => tagMap.value[id])
    .filter(Boolean)
  if (!missingSelected.length) {
    return list
  }
  return [...missingSelected, ...list]
})

const activeTagFilterLabel = computed(() => {
  if (selectedTagIds.value.length === 0) {
    return tagFilterCleared.value
      ? t('knowledgeBase.tagFilterPlaceholder')
      : t('knowledgeBase.allTags')
  }
  if (selectedTagIds.value.length === 1) {
    const id = selectedTagIds.value[0]
    return tagMap.value[id]?.name || t('knowledgeBase.allTags')
  }
  return t('knowledgeBase.tagFilterMulti', { count: selectedTagIds.value.length })
})

const activeTagFilterTitle = computed(() => {
  if (selectedTagIds.value.length === 0) {
    return t('knowledgeBase.tagFilterTitle')
  }
  const names = selectedTagIds.value
    .map((id) => tagMap.value[id]?.name)
    .filter(Boolean)
  return names.length > 0 ? names.join('、') : t('knowledgeBase.tagFilterTitle')
})

const isTagFilterActive = (tagId: string) => selectedTagIds.value.includes(tagId)

const kbInfo = ref<any>(null)
const knowledgeList = ref<Array<{ id: string; name: string; type?: string }>>([])

const loadKnowledgeInfo = async (kbId: string) => {
  if (!kbId) {
    kbInfo.value = null
    return
  }
  try {
    const res: any = await getKnowledgeBaseById(kbId)
    kbInfo.value = res?.data || null
    return kbInfo.value
  } catch (error) {
    console.error('Failed to load knowledge base info:', error)
    kbInfo.value = null
    return null
  }
}

const loadKnowledgeList = async () => {
  try {
    const res: any = await listKnowledgeBases()
    const myKbs: typeof knowledgeList.value = (res?.data || []).map((item: any) => ({
      id: String(item.id),
      name: item.name,
      type: item.type,
    }))

    // Also include shared knowledge bases from orgStore
    const sharedKbs: typeof knowledgeList.value = (orgStore.sharedKnowledgeBases || [])
      .filter(s => s.knowledge_base != null)
      .map(s => ({
        id: String(s.knowledge_base.id),
        name: s.knowledge_base.name,
        type: s.knowledge_base.type,
      }))

    // Merge and deduplicate by id (my KBs take precedence)
    const myKbIds = new Set(myKbs.map(kb => kb.id))
    const uniqueSharedKbs = sharedKbs.filter(kb => !myKbIds.has(kb.id))

    knowledgeList.value = [...myKbs, ...uniqueSharedKbs]
  } catch (error) {
    console.error('Failed to load knowledge bases:', error)
  }
}

const editorVisible = ref(false)
const editorMode = ref<'create' | 'edit'>('create')
const currentEntryId = ref<number | null>(null)
const editorForm = reactive<FAQEntryPayload>({
  standard_question: '',
  similar_questions: [],
  negative_questions: [],
  answers: [],
  tag_id: undefined,
})
const editorFormRef = ref<FormInstanceFunctions>()
const savingEntry = ref(false)

// Input field state
const answerInput = ref('')
const similarInput = ref('')
const negativeInput = ref('')

const importVisible = ref(false)
const fileInputRef = ref<HTMLInputElement | null>(null)
const importState = reactive({
  mode: 'append' as 'append' | 'replace',
  file: null as File | null,
  preview: [] as FAQEntryPayload[],
  importing: false,
  taskId: null as string | null,
  taskStatus: null as {
    status: string
    progress: number
    total: number
    processed: number
    message?: string
    error?: string
  } | null,
  pollingInterval: null as ReturnType<typeof setInterval> | null,
})

// FAQ import result state (persisted)
type FAQImportResultView = {
  total_entries: number
  success_count: number
  failed_count: number
  skipped_count: number
  partial_failed_count: number
  merged_count: number
  added_count: number
  import_mode: string
  imported_at: string
  task_id: string
  processing_time: number
  message?: string
  failed_entries_url?: string
  success_entries?: Array<{
    index: number
    seq_id: number
    tag_id?: number
    tag_name?: string
    standard_question: string
  }>
  display_status: string
}

const importResult = ref<FAQImportResultView | null>(null)
const importResultExpanded = ref(false)

const showImportResultBadge = computed(() => (
  !!importResult.value
  && importResult.value.display_status === 'open'
  && !importState.taskId
))

const isImportInProgress = computed(() => {
  const status = importState.taskStatus?.status
  return !!importState.taskId && (status === 'running' || status === 'pending')
})

const importResultSummary = computed(() => {
  const result = importResult.value
  if (!result) return ''
  if (result.message?.trim()) {
    return result.message.trim()
  }
  const parts: string[] = []
  parts.push(`${t('faqManager.import.totalData')} ${result.total_entries}`)
  if (result.merged_count > 0) {
    if (result.added_count > 0) {
      parts.push(`${t('faqManager.import.added')} ${result.added_count}`)
    }
    parts.push(`${t('faqManager.import.merged')} ${result.merged_count}`)
  } else if (result.success_count > 0) {
    parts.push(`${t('faqManager.import.success')} ${result.success_count}`)
  }
  if (result.partial_failed_count > 0) {
    parts.push(`${t('faqManager.import.partialFailed')} ${result.partial_failed_count}`)
  }
  if (result.failed_count > 0) {
    parts.push(`${t('faqManager.import.failed')} ${result.failed_count}`)
  }
  if (result.skipped_count > 0) {
    parts.push(`${t('faqManager.import.skipped')} ${result.skipped_count}`)
  }
  return parts.join(' · ')
})

const importProgressTitle = computed(() => {
  const status = importState.taskStatus?.status
  if (status === 'running') return t('faqManager.import.importing')
  if (status === 'success') return t('faqManager.import.importDone')
  if (status === 'failed') return t('faqManager.import.importFailed')
  return t('faqManager.import.waiting')
})

const importProgressIcon = computed(() => {
  const status = importState.taskStatus?.status
  if (status === 'running') return 'loading'
  if (status === 'success') return 'check-circle-filled'
  if (status === 'failed') return 'error-circle-filled'
  return 'time-filled'
})

const importProgressText = computed(() => {
  const status = importState.taskStatus
  if (!status) return ''
  if (status.error) return status.error
  if (status.message?.trim()) return status.message.trim()
  return importProgressTitle.value
})

// Search test state
const searchDrawerVisible = ref(false)
const searching = ref(false)
const hasSearched = ref(false)
const searchResults = ref<FAQEntry[]>([])
const searchForm = reactive({
  query: '',
  vectorThreshold: 0.7,
  matchCount: 10,
})


const getTagName = (tagId?: number) => {
  if (!tagId) return t('knowledgeBase.untagged')
  return tagMapBySeqId.value[tagId]?.name || t('knowledgeBase.untagged')
}

const handleTagFilterChange = (tagIds: string[]) => {
  selectedTagIds.value = tagIds
  uiStore.clearSelectedTagIds()
  tagIds.forEach((id) => uiStore.toggleSelectedTagId(id))
}

const handleTagRowClick = (tagId: string) => {
  const next = new Set(selectedTagIds.value)
  if (next.has(tagId)) {
    next.delete(tagId)
  } else {
    next.add(tagId)
  }
  if (next.size > 0) {
    tagFilterCleared.value = false
  }
  handleTagFilterChange([...next])
}

const clearTagFilter = () => {
  tagFilterCleared.value = true
  handleTagFilterChange([])
}

const openTagManageDrawer = () => {
  tagFilterPanelVisible.value = false
  tagManageDrawerVisible.value = true
}

const onTagManageChanged = (payload?: { deletedTagId?: string }) => {
  if (!props.kbId) return
  void loadTags(true)
  if (payload?.deletedTagId && selectedTagIds.value.includes(payload.deletedTagId)) {
    selectedTagIds.value = selectedTagIds.value.filter((id) => id !== payload.deletedTagId)
    handleTagFilterChange([...selectedTagIds.value])
  }
  currentPage = 1
  entries.value = []
  selectedRowKeys.value = []
  void loadEntries()
}

const loadTags = async (reset = false) => {
  if (!props.kbId) {
    tagList.value = []
    tagTotal.value = 0
    tagHasMore.value = false
    tagPage.value = 1
    return
  }

  if (reset) {
    tagPage.value = 1
    tagList.value = []
    tagTotal.value = 0
    tagHasMore.value = false
  } else if (tagLoading.value || tagLoadingMore.value) {
    return
  }

  const currentTagPage = tagPage.value || 1
  tagLoading.value = currentTagPage === 1
  tagLoadingMore.value = currentTagPage > 1

  try {
    const res: any = await listKnowledgeTags(props.kbId, {
      page: currentTagPage,
      page_size: TAG_PAGE_SIZE,
      keyword: tagSearchQuery.value || undefined,
    })
    const pageData = (res?.data || {}) as {
      data?: any[]
      total?: number
    }
    const pageTags = (pageData.data || []).map((tag: any) => ({
      ...tag,
      id: String(tag.id),
    }))

    if (currentTagPage === 1) {
      tagList.value = pageTags
    } else {
      tagList.value = [...tagList.value, ...pageTags]
    }

    tagTotal.value = pageData.total || tagList.value.length
    tagHasMore.value = tagList.value.length < tagTotal.value
    if (tagHasMore.value) {
      tagPage.value = currentTagPage + 1
    }
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('common.operationFailed'))
  } finally {
    tagLoading.value = false
    tagLoadingMore.value = false
  }
}

const handleEntryTagChange = async (entryId: number, value?: string) => {
  if (!props.kbId) return
  const targetEntry = entries.value.find((item) => item.id === entryId)
  const previousTagId = targetEntry ? targetEntry.tag_id : undefined
  const normalizedValue = value ? Number(value) : null
  if (normalizedValue === previousTagId) {
    return
  }
  try {
    await updateFAQEntryTagBatch(props.kbId, { updates: { [entryId]: normalizedValue } })
    MessagePlugin.success(t('knowledgeEditor.messages.updateSuccess'))
    await loadEntries()
    await loadTags(true)
  } catch (error: any) {
    if (targetEntry) {
      targetEntry.tag_id = previousTagId
    }
    MessagePlugin.error(error?.message || t('common.operationFailed'))
  }
}

const handleNavigateToKbList = () => {
  router.push('/platform/knowledge-bases')
}

const handleNavigateToCurrentKB = () => {
  if (!props.kbId) return
  router.push(`/platform/knowledge-bases/${props.kbId}`)
}

const handleOpenKBSettings = () => {
  if (!props.kbId) {
    MessagePlugin.warning(t('knowledgeEditor.messages.missingId'))
    return
  }
  uiStore.openKBSettings(props.kbId)
}

const handleKnowledgeDropdownSelect = (data: { value: string }) => {
  if (!data?.value || data.value === props.kbId) return
  router.push(`/platform/knowledge-bases/${data.value}`)
}

const handleFaqMenuAction = (event: Event) => {
  const detail = (event as CustomEvent<{ action: string; kbId: string }>).detail
  if (!detail || detail.kbId !== props.kbId) return

  if (detail.action === 'create') {
    if (canEdit.value) openEditor()
  } else if (detail.action === 'import') {
    if (canEdit.value) openImportDialog()
  } else if (detail.action === 'search') {
    searchDrawerVisible.value = true
  } else if (detail.action === 'export') {
    // Export is usually allowed for viewers as well
    handleExportCSV()
  } else if (detail.action === 'batch') {
    // Bulk actions are handled via the dropdown menu in the left-side menu
    if (selectedRowKeys.value.length === 0) {
      MessagePlugin.warning(t('knowledgeEditor.faq.selectEntriesFirst'))
    }
  } else if (detail.action === 'batchTag') {
    if (canEdit.value && selectedRowKeys.value.length > 0) {
      openBatchTagDialog()
    }
  } else if (detail.action === 'batchEnable') {
    if (canEdit.value && selectedRowKeys.value.length > 0) {
      handleBatchStatusChange(true)
    }
  } else if (detail.action === 'batchDisable') {
    if (canEdit.value && selectedRowKeys.value.length > 0) {
      handleBatchStatusChange(false)
    }
  } else if (detail.action === 'batchDelete') {
    if (canManage.value && selectedRowKeys.value.length > 0) {
      handleBatchDelete()
    }
  }
}

const handleEntryStatusChange = async (entry: FAQEntry, value: boolean) => {
  if (!props.kbId) {
    return
  }
  const entryIndex = entries.value.findIndex(e => e.id === entry.id)
  if (entryIndex === -1) {
    return
  }
  // Get the actual object reference from the array to ensure the latest data is used
  const actualEntry = entries.value[entryIndex]
  const previous = actualEntry.is_enabled
  if (previous === value) {
    return
  }
  // Update the property directly, Vue 3's reactivity system should detect it
  actualEntry.is_enabled = value
  entryStatusLoading[entry.id] = true
  try {
    await updateFAQEntryFieldsBatch(props.kbId, { by_id: { [entry.id]: { is_enabled: value } } })
    MessagePlugin.success(t(value ? 'knowledgeEditor.faq.statusEnableSuccess' : 'knowledgeEditor.faq.statusDisableSuccess'))
  } catch (error: any) {
    // Roll back on failure
    actualEntry.is_enabled = previous
    MessagePlugin.error(error?.message || t('knowledgeEditor.faq.statusUpdateFailed'))
  } finally {
    entryStatusLoading[entry.id] = false
  }
}

const handleEntryRecommendedChange = async (entry: FAQEntry, value: boolean) => {
  if (entryRecommendedLoading[entry.id]) {
    return
  }
  const entryIndex = entries.value.findIndex(e => e.id === entry.id)
  if (entryIndex === -1) {
    return
  }
  const actualEntry = entries.value[entryIndex]
  const previous = actualEntry.is_recommended
  if (previous === value) {
    return
  }
  actualEntry.is_recommended = value
  entryRecommendedLoading[entry.id] = true
  try {
    await updateFAQEntryFieldsBatch(props.kbId, { by_id: { [entry.id]: { is_recommended: value } } })
    MessagePlugin.success(t(value ? 'knowledgeEditor.faq.recommendedEnableSuccess' : 'knowledgeEditor.faq.recommendedDisableSuccess'))
  } catch (error: any) {
    actualEntry.is_recommended = previous
    MessagePlugin.error(error?.message || t('knowledgeEditor.faq.recommendedUpdateFailed'))
  } finally {
    entryRecommendedLoading[entry.id] = false
  }
}

const editorRules: FormRules<FAQEntryPayload> = {
  standard_question: [
    { required: true, message: t('knowledgeEditor.messages.nameRequired') },
  ],
  answers: [
    {
      validator: (val: string[]) => Array.isArray(val) && val.length > 0,
      message: t('knowledgeEditor.faq.answerRequired'),
    },
  ],
}

const loadEntries = async (append = false) => {
  if (!props.kbId) return
  if (append) {
    loadingMore.value = true
  } else {
    loading.value = true
    currentPage = 1
    entries.value = []
    selectedRowKeys.value = []
    Object.keys(entryStatusLoading).forEach((key) => {
      delete entryStatusLoading[Number(key)]
    })
  }

  try {
    // If overallFAQTotal is not initialized, fetch it first (without tag_id filter)
    if (overallFAQTotal.value === 0 && !append) {
      const totalRes = await listFAQEntries(props.kbId, {
        page: 1,
        page_size: 1,
      })
      const totalData = (totalRes.data || {}) as { total: number }
      overallFAQTotal.value = totalData.total || 0
    }

    const res = await listFAQEntries(props.kbId, {
      page: currentPage,
      page_size: pageSize,
      tag_ids: selectedTagIds.value.length > 0 ? selectedTagIds.value.join(',') : undefined,
      keyword: entrySearchKeyword.value ? entrySearchKeyword.value.trim() : undefined,
    })
    const pageData = (res.data || {}) as {
      data: FAQEntry[]
      total: number
    }
    const newEntries = (pageData.data || []).map(entry => ({
      ...entry,
      showMore: false,
      similarCollapsed: true,  // Similar questions collapsed by default
      negativeCollapsed: true,  // Counterexamples collapsed by default
      answersCollapsed: true,   // Answer collapsed by default
      is_enabled: entry.is_enabled !== false,
    }))

    if (append) {
      entries.value = [...entries.value, ...newEntries]
    } else {
      entries.value = newEntries
    }
    // Determine whether there is more data
    hasMore.value = entries.value.length < (pageData.total || 0)
    currentPage++

    // Wait for the DOM to update before re-laying out
    await nextTick()
    arrangeCards()
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('common.operationFailed'))
  } finally {
    loading.value = false
    loadingMore.value = false

    // Check whether more loading is needed to fill the visible area
    // Delay execution to ensure arrangeCards' requestAnimationFrame has completed
    setTimeout(() => {
      checkAndLoadMore()
    }, 350)
  }
}

const handleScroll = () => {
  if (!scrollContainer.value || loadingMore.value || !hasMore.value) return

  const container = scrollContainer.value
  const scrollTop = container.scrollTop
  const scrollHeight = container.scrollHeight
  const clientHeight = container.clientHeight

  // Load more when scrolled within 200px of the bottom
  if (scrollTop + clientHeight >= scrollHeight - 200) {
    loadEntries(true)
  }
}

// Check whether the content fills the viewport; if not and more data is available, keep loading
const checkAndLoadMore = () => {
  if (!scrollContainer.value) return
  if (loadingMore.value || loading.value) return
  if (!hasMore.value) return

  const container = scrollContainer.value
  const scrollHeight = container.scrollHeight
  const clientHeight = container.clientHeight

  // If the content height is less than the container height plus a 50px buffer, there's likely no scrollbar or it's near the bottom, so keep loading
  if (scrollHeight <= clientHeight + 50) {
    loadEntries(true)
  }
}

const handleCardSelect = (entryId: number, checked: boolean) => {
  if (checked) {
    if (!selectedRowKeys.value.includes(entryId)) {
      selectedRowKeys.value.push(entryId)
    }
  } else {
    const index = selectedRowKeys.value.indexOf(entryId)
    if (index > -1) {
      selectedRowKeys.value.splice(index, 1)
    }
  }
}

const resetEditorForm = () => {
  editorForm.standard_question = ''
  editorForm.similar_questions = []
  editorForm.negative_questions = []
  editorForm.answers = []
  editorForm.tag_id = undefined
  answerInput.value = ''
  similarInput.value = ''
  negativeInput.value = ''
}

const openEditor = (entry?: FAQEntry) => {
  if (entry) {
    editorMode.value = 'edit'
    currentEntryId.value = entry.id
    editorForm.standard_question = entry.standard_question
    editorForm.similar_questions = [...(entry.similar_questions || [])]
    editorForm.negative_questions = [...(entry.negative_questions || [])]
    editorForm.answers = [...(entry.answers || [])]
    editorForm.tag_id = entry.tag_id || undefined
  } else {
    editorMode.value = 'create'
    currentEntryId.value = null
    resetEditorForm()
  }
  answerInput.value = ''
  similarInput.value = ''
  negativeInput.value = ''
  editorVisible.value = true
}

const handleEditorClose = () => {
  // Reset the form on close
  resetEditorForm()
  answerInput.value = ''
  similarInput.value = ''
  negativeInput.value = ''
  editorFormRef.value?.clearValidate?.()
}

// Add answer
const addAnswer = () => {
  const trimmed = answerInput.value.trim()
  if (trimmed && editorForm.answers.length < 5 && !editorForm.answers.includes(trimmed)) {
    editorForm.answers.push(trimmed)
    answerInput.value = ''
  }
}

// Delete answer
const removeAnswer = (index: number) => {
  editorForm.answers.splice(index, 1)
}

// Add similar question
const addSimilar = () => {
  const trimmed = similarInput.value.trim()
  if (trimmed && editorForm.similar_questions.length < 10 && !editorForm.similar_questions.includes(trimmed)) {
    editorForm.similar_questions.push(trimmed)
    similarInput.value = ''
  }
}

// Delete similar question
const removeSimilar = (index: number) => {
  editorForm.similar_questions.splice(index, 1)
}

// Add negative example
const addNegative = () => {
  const trimmed = negativeInput.value.trim()
  if (trimmed && editorForm.negative_questions.length < 10 && !editorForm.negative_questions.includes(trimmed)) {
    editorForm.negative_questions.push(trimmed)
    negativeInput.value = ''
  }
}

// Delete negative example
const removeNegative = (index: number) => {
  editorForm.negative_questions.splice(index, 1)
}

const handleSubmitEntry = async () => {
  if (!editorFormRef.value) return
  const result = await editorFormRef.value.validate?.()
  if (result !== true) return

  savingEntry.value = true
  try {
    const payload: FAQEntryPayload = {
      standard_question: editorForm.standard_question,
      similar_questions: [...editorForm.similar_questions],
      negative_questions: [...editorForm.negative_questions],
      answers: [...editorForm.answers],
      tag_id: editorForm.tag_id || undefined,
    }
    if (editorMode.value === 'create') {
      await createFAQEntry(props.kbId, payload)
      MessagePlugin.success(t('knowledgeEditor.messages.createSuccess'))
    } else if (currentEntryId.value) {
      await updateFAQEntry(props.kbId, currentEntryId.value, payload)
      MessagePlugin.success(t('knowledgeEditor.messages.updateSuccess'))
    }
    editorVisible.value = false
    await loadEntries()
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('common.operationFailed'))
  } finally {
    savingEntry.value = false
  }
}

const handleBatchDelete = async () => {
  if (!selectedRowKeys.value.length) return
  try {
    await deleteFAQEntries(props.kbId, selectedRowKeys.value)
    MessagePlugin.success(t('knowledgeEditor.faqImport.deleteSuccess'))
    selectedRowKeys.value = []
    await loadEntries()
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('common.operationFailed'))
  }
}

// Batch status update dialog
const batchTagDialogVisible = ref(false)
const batchTagValue = ref<string>('')

const openBatchTagDialog = () => {
  if (!selectedRowKeys.value.length) return
  batchTagValue.value = ''
  batchTagDialogVisible.value = true
}

const handleBatchTag = async () => {
  if (!selectedRowKeys.value.length || !props.kbId) return
  try {
    const updates: Record<number, number | null> = {}
    selectedRowKeys.value.forEach(id => {
      updates[id] = batchTagValue.value ? Number(batchTagValue.value) : null
    })
    await updateFAQEntryTagBatch(props.kbId, { updates })
    MessagePlugin.success(t('knowledgeEditor.messages.updateSuccess'))
    batchTagDialogVisible.value = false
    selectedRowKeys.value = []
    await loadEntries()
    await loadTags(true)
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('common.operationFailed'))
  }
}

const handleBatchStatusChange = async (isEnabled: boolean) => {
  if (!selectedRowKeys.value.length || !props.kbId) return
  try {
    const by_id: Record<number, { is_enabled: boolean }> = {}
    selectedRowKeys.value.forEach(id => {
      by_id[id] = { is_enabled: isEnabled }
    })
    await updateFAQEntryFieldsBatch(props.kbId, { by_id })
    MessagePlugin.success(t(isEnabled ? 'knowledgeEditor.faq.statusEnableSuccess' : 'knowledgeEditor.faq.statusDisableSuccess'))
    selectedRowKeys.value = []
    await loadEntries()
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('common.operationFailed'))
  }
}

const handleBatchRecommendedChange = async (isRecommended: boolean) => {
  if (!selectedRowKeys.value.length || !props.kbId) return
  try {
    const by_id: Record<number, { is_recommended: boolean }> = {}
    selectedRowKeys.value.forEach(id => {
      by_id[id] = { is_recommended: isRecommended }
    })
    await updateFAQEntryFieldsBatch(props.kbId, { by_id })
    MessagePlugin.success(t(isRecommended ? 'knowledgeEditor.faq.recommendedEnableSuccess' : 'knowledgeEditor.faq.recommendedDisableSuccess'))
    selectedRowKeys.value = []
    await loadEntries()
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('common.operationFailed'))
  }
}

const handleMenuEdit = (entry: FAQEntry) => {
  entry.showMore = false
  openEditor(entry)
}

const handleMenuDelete = async (entry: FAQEntry) => {
  entry.showMore = false
  try {
    await deleteFAQEntries(props.kbId, [entry.id])
    MessagePlugin.success(t('knowledgeEditor.faqImport.deleteSuccess'))
    await loadEntries()
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('common.operationFailed'))
  }
}

const openImportDialog = () => {
  // Disallow opening the import dialog while an import is in progress
  if (importState.taskStatus?.status === 'running') {
    MessagePlugin.warning(t('faqManager.import.importInProgress'))
    return
  }
  stopPolling()
  importVisible.value = true
  importState.file = null
  importState.preview = []
  importState.mode = 'append'
  // Note: don't clear taskId and taskStatus, so progress is still visible after closing the dialog
  importState.importing = false
}

const processFile = async (file: File) => {
  importState.file = file

  try {
    let parsed: FAQEntryPayload[] = []
    if (file.name.endsWith('.json')) {
      parsed = await parseJSONFile(file)
    } else if (file.name.endsWith('.csv')) {
      parsed = await parseCSVFile(file)
    } else if (file.name.endsWith('.xlsx') || file.name.endsWith('.xls')) {
      parsed = await parseExcelFile(file)
    } else {
      MessagePlugin.warning(t('knowledgeEditor.faqImport.unsupportedFormat'))
      importState.preview = []
      return
    }
    importState.preview = parsed
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('knowledgeEditor.faqImport.parseFailed'))
    importState.preview = []
  }
}

const handleFileChange = async (event: Event) => {
  const target = event.target as HTMLInputElement
  const file = target.files?.[0]
  if (!file) return
  await processFile(file)
}

const handleFileDrop = async (event: DragEvent) => {
  const file = event.dataTransfer?.files[0]
  if (!file) return
  await processFile(file)
}

const parseJSONFile = async (file: File): Promise<FAQEntryPayload[]> => {
  const text = await file.text()
  const data = JSON.parse(text)
  if (!Array.isArray(data)) {
    throw new Error(t('knowledgeEditor.faqImport.invalidJSON'))
  }
  return data.map(normalizePayload)
}

const parseCSVFile = async (file: File): Promise<FAQEntryPayload[]> => {
  const text = await file.text()

  // Use papaparse to parse CSV, automatically handling quotes, escaping, delimiters, etc.
  return new Promise((resolve, reject) => {
    Papa.parse(text, {
      header: true,
      skipEmptyLines: true,
      delimiter: '', // Auto-detect the delimiter (comma or tab)
      quoteChar: '"',
      escapeChar: '"',
      transformHeader: (header: string) => {
        // Strip parentheses and descriptions from field names, keeping only the core field name
        const cleaned = header.trim()
          .replace(/\([^)]*\)/g, '') // Remove parentheses and their contents
          .trim()
        // Don't lowercase Chinese field names; lowercase English field names
        return /[\u4e00-\u9fa5]/.test(cleaned) ? cleaned : cleaned.toLowerCase()
      },
      complete: (results) => {
        try {
          const payloads: FAQEntryPayload[] = []
          results.data.forEach((row: any) => {
            const record: Record<string, string> = {}
            // Convert row data into a record object
            Object.keys(row).forEach((key) => {
              record[key] = String(row[key] || '').trim()
            })

            const isDisabled = parseBooleanField(record['是否停用'] || record['Disabled (optional - default FALSE)'] || record['Disabled'], false)
            payloads.push(
              normalizePayload({
                standard_question: record['问题'] || record['Question (required)'] || record['Question'] || record['standard_question'] || record['question'] || '',
                answers: splitByDelimiter(record['机器人回答'] || record['Bot answers (required - separate multiple with ##)'] || record['Bot answers'] || record['answers']),
                similar_questions: splitByDelimiter(record['相似问题'] || record['Similar questions (optional - separate multiple with ##)'] || record['Similar questions'] || record['similar_questions']),
                negative_questions: splitByDelimiter(record['反例问题'] || record['Negative examples (optional - separate multiple with ##)'] || record['Negative examples'] || record['negative_questions']),
                tag_id: record['tag_id'] ? Number(record['tag_id']) : undefined,
                tag_name: record['标签'] || record['分类'] || record['Tag (required)'] || record['Tag'] || record['Category (required)'] || record['tag_name'] || '',
                is_enabled: isDisabled !== undefined ? !isDisabled : undefined, // Whether disabled: FALSE means enabled, TRUE means disabled, so negate it
              }),
            )
          })
          resolve(payloads)
        } catch (error) {
          reject(error)
        }
      },
      error: (error: Error) => {
        reject(new Error(`CSV parse failed: ${error.message}`))
      },
    })
  })
}

const parseExcelFile = async (file: File): Promise<FAQEntryPayload[]> => {
  const data = await file.arrayBuffer()
  const workbook = XLSX.read(data, { type: 'array' })
  const sheetName = workbook.SheetNames[0]
  const worksheet = workbook.Sheets[sheetName]
  // Use raw: false to ensure quotes and escaping are handled correctly
  const json = XLSX.utils.sheet_to_json<Record<string, string>>(worksheet, {
    defval: '',
    raw: false // Ensure string values are parsed correctly
  })
  return json.map((row) => {
    // Get the raw header (with parenthetical descriptions stripped)
    const normalizedRow: Record<string, string> = {}
    Object.keys(row).forEach((key) => {
      const normalizedKey = key.trim()
        .replace(/\([^)]*\)/g, '') // Remove parentheses and their contents
        .trim()
      // Don't lowercase Chinese field names; lowercase English field names
      const finalKey = /[\u4e00-\u9fa5]/.test(normalizedKey) ? normalizedKey : normalizedKey.toLowerCase()
      // Ensure the value is a string
      normalizedRow[finalKey] = String(row[key] || '').trim()
    })

    const isDisabled = parseBooleanField(normalizedRow['是否停用'] || normalizedRow['Disabled (optional - default FALSE)'] || normalizedRow['Disabled'], false)
    return normalizePayload({
      standard_question: normalizedRow['问题'] || normalizedRow['Question (required)'] || normalizedRow['Question'] || normalizedRow['standard_question'] || normalizedRow['question'] || '',
      answers: splitByDelimiter(normalizedRow['机器人回答'] || normalizedRow['Bot answers (required - separate multiple with ##)'] || normalizedRow['Bot answers'] || normalizedRow['answers']),
      similar_questions: splitByDelimiter(normalizedRow['相似问题'] || normalizedRow['Similar questions (optional - separate multiple with ##)'] || normalizedRow['Similar questions'] || normalizedRow['similar_questions']),
      negative_questions: splitByDelimiter(normalizedRow['反例问题'] || normalizedRow['Negative examples (optional - separate multiple with ##)'] || normalizedRow['Negative examples'] || normalizedRow['negative_questions']),
      tag_id: normalizedRow['tag_id'] ? Number(normalizedRow['tag_id']) : undefined,
      tag_name: normalizedRow['标签'] || normalizedRow['分类'] || normalizedRow['Tag (required)'] || normalizedRow['Tag'] || normalizedRow['Category (required)'] || normalizedRow['tag_name'] || '',
      is_enabled: isDisabled !== undefined ? !isDisabled : undefined, // Whether disabled: FALSE means enabled, TRUE means disabled, so negate it
    })
  })
}

const splitByDelimiter = (value?: string) => {
  if (!value) return []
  // Use only ## as the delimiter, to avoid incorrectly splitting on commas, semicolons, etc.
  const trimmedValue = value.trim()
  if (!trimmedValue) return []

  // If the ## delimiter is present, split on ##
  if (trimmedValue.includes('##')) {
    return trimmedValue
      .split('##')
      .map(item => item.trim())
      .filter(Boolean)
  }

  // If there's no ## delimiter, treat the whole value as a single answer
  return [trimmedValue]
}

// Parse boolean fields (supports multiple formats: TRUE/FALSE, true/false, 是/否, 1/0, etc.)
const parseBooleanField = (value?: string, defaultValue: boolean = true): boolean | undefined => {
  if (!value) return undefined
  const normalized = value.trim().toUpperCase()
  if (normalized === 'TRUE' || normalized === '1' || normalized === '是' || normalized === 'YES') {
    return true
  }
  if (normalized === 'FALSE' || normalized === '0' || normalized === '否' || normalized === 'NO') {
    return false
  }
  return defaultValue
}

const normalizePayload = (payload: Partial<FAQEntryPayload>): FAQEntryPayload => ({
  standard_question: payload.standard_question || '',
  answers: payload.answers?.filter(Boolean) || [],
  similar_questions: payload.similar_questions?.filter(Boolean) || [],
  negative_questions: payload.negative_questions?.filter(Boolean) || [],
  tag_id: payload.tag_id || undefined,
  tag_name: payload.tag_name || '',
  is_enabled: payload.is_enabled !== undefined ? payload.is_enabled : undefined,
})

const stopPolling = () => {
  if (importState.pollingInterval) {
    clearInterval(importState.pollingInterval)
    importState.pollingInterval = null
  }
}

const startPolling = (taskId: string) => {
  stopPolling()
  // Save taskId to localStorage so it can be restored after a refresh
  saveTaskIdToStorage(taskId)

  // Track the previously processed count, used to determine whether the list needs refreshing
  let lastProcessed = 0

  importState.pollingInterval = setInterval(async () => {
    try {
      const res: any = await getFAQImportProgress(taskId)
      const progressData = res?.data
      if (progressData) {
        // Extract status from the Redis progress data
        // status: "pending" -> "pending", "processing" -> "running", "completed" -> "success", "failed" -> "failed"
        let status = progressData.status
        if (status === 'processing') {
          status = 'running'
        } else if (status === 'completed') {
          status = 'success'
        }

        const progress = progressData.progress || 0
        const total = progressData.total || 0
        const processed = progressData.processed || 0
        const error = progressData.error || ''
        const message = progressData.message || ''

        importState.taskStatus = {
          status: status,
          progress: progress,
          total: total,
          processed: processed,
          message: message,
          error: error,
        }

        // Refresh the FAQ list on progress updates (refresh once every so many entries added)
        if (processed > lastProcessed) {
          lastProcessed = processed
          await loadEntries()
          await loadTags(true)
        }

        // Stop polling once the task completes or fails (but don't auto-close the progress bar — let the user close it manually)
        if (status === 'success' || status === 'failed') {
          stopPolling()
          if (status === 'success') {
            // Save the completed taskId for loading results later
            if (importState.taskId) {
              saveLastCompletedTaskId(importState.taskId)
            }
            MessagePlugin.success(progressData.message || t('knowledgeEditor.faqImport.importSuccess'))
            // Clear filters so the user can see all newly imported data
            selectedTagIds.value = []
            tagFilterCleared.value = false
            uiStore.clearSelectedTagIds()
            entrySearchKeyword.value = ''
            overallFAQTotal.value = 0  // Reset to trigger re-fetch
            await loadEntries()
            await loadTags(true)
            await loadImportResult() // Load the latest import result stats
            // Auto-close the progress bar 3 seconds after the task completes
            setTimeout(() => {
              if (importState.taskStatus?.status === 'success') {
                handleCloseProgress()
              }
            }, 3000)
          } else {
            MessagePlugin.error(error || t('common.operationFailed'))
            // Don't auto-close on failure — let the user see the error message
          }
        }
      }
    } catch (error: any) {
      console.error('Failed to poll task status:', error)
      // If the task doesn't exist or has expired, clear storage
      if (error?.response?.status === 404 || error?.message?.includes('not found')) {
        clearTaskIdFromStorage()
        stopPolling()
        importState.taskId = null
        importState.taskStatus = null
      }
    }
  }, 3000) // Poll every 3 seconds
}

const handleCancelImport = () => {
  stopPolling()
  importState.importing = false
  importState.taskId = null
  importState.taskStatus = null
  importVisible.value = false
  // Note: don't clear localStorage, since the task may still be in progress
}

const handleCloseProgress = () => {
  stopPolling()
  importState.taskId = null
  importState.taskStatus = null
  clearTaskIdFromStorage()
}

// localStorage-related functions
const getStorageKey = () => {
  return `faq_import_task_${props.kbId}`
}

const saveTaskIdToStorage = (taskId: string) => {
  if (!props.kbId) return
  try {
    localStorage.setItem(getStorageKey(), taskId)
  } catch (error) {
    console.error('Failed to save taskId to localStorage:', error)
  }
}

const getTaskIdFromStorage = (): string | null => {
  if (!props.kbId) return null
  try {
    return localStorage.getItem(getStorageKey())
  } catch (error) {
    console.error('Failed to get taskId from localStorage:', error)
    return null
  }
}

const clearTaskIdFromStorage = () => {
  if (!props.kbId) return
  try {
    localStorage.removeItem(getStorageKey())
  } catch (error) {
    console.error('Failed to clear taskId from localStorage:', error)
  }
}

// Restore import task state (for recovery after refresh)
const restoreImportTask = async () => {
  if (!props.kbId) return

  const savedTaskId = getTaskIdFromStorage()
  if (!savedTaskId) return

  try {
    // Query progress status in Redis
    const res: any = await getFAQImportProgress(savedTaskId)
    const progressData = res?.data

    if (progressData) {
      // Extract status from Redis progress data
      let status = progressData.status
      if (status === 'processing') {
        status = 'running'
      } else if (status === 'completed') {
        status = 'success'
      }

      const progress = progressData.progress || 0
      const total = progressData.total || 0
      const processed = progressData.processed || 0
      const error = progressData.error || ''

      importState.taskId = savedTaskId
      importState.taskStatus = {
        status: status,
        progress: progress,
        total: total,
        processed: processed,
        message: progressData.message || '',
        error: error,
      }

      // If the task is still in progress, resume polling
      if (status === 'pending' || status === 'running') {
        startPolling(savedTaskId)
      } else {
        // Task completed or failed, clear storage
        clearTaskIdFromStorage()
      }
    } else {
      // Task doesn't exist, clear storage
      clearTaskIdFromStorage()
    }
  } catch (error: any) {
    console.error('Failed to restore import task:', error)
    // If the task doesn't exist or has expired, clear storage
    if (error?.response?.status === 404 || error?.message?.includes('not found')) {
      clearTaskIdFromStorage()
    }
  }
}

// localStorage key for last completed task
const getLastCompletedTaskKey = () => {
  return `faq_import_last_completed_${props.kbId}`
}

const saveLastCompletedTaskId = (taskId: string) => {
  if (!props.kbId) return
  try {
    localStorage.setItem(getLastCompletedTaskKey(), taskId)
  } catch (error) {
    console.error('Failed to save last completed taskId:', error)
  }
}

const getLastCompletedTaskId = (): string | null => {
  if (!props.kbId) return null
  try {
    return localStorage.getItem(getLastCompletedTaskKey())
  } catch (error) {
    return null
  }
}

// Load persisted import result statistics
const loadImportResult = async () => {
  if (!props.kbId) return

  const lastTaskId = getLastCompletedTaskId()
  if (!lastTaskId) {
    importResult.value = null
    return
  }

  try {
    const res: any = await getFAQImportProgress(lastTaskId)
    const data = res?.data
    if (data && data.status === 'completed') {
      // Check the display_status returned by the backend; if it's close, don't show it
      if (data.display_status === 'close') {
        importResult.value = null
        return
      }
      // Map progress fields to importResult format
      importResult.value = {
        total_entries: data.total,
        success_count: data.success_count || 0,
        failed_count: data.failed_count || 0,
        skipped_count: data.skipped_count || 0,
        partial_failed_count: data.partial_failed_count || 0,
        merged_count: data.merged_count || 0,
        added_count: data.added_count || 0,
        message: data.message || '',
        import_mode: data.import_mode || 'append',
        imported_at: data.imported_at,
        task_id: data.task_id,
        failed_entries_url: data.failed_entries_url,
        success_entries: data.success_entries,
        display_status: data.display_status || 'open',
        processing_time: data.processing_time || 0,
      }
    } else {
      importResult.value = null
    }
  } catch (error) {
    console.error('Failed to load FAQ import result:', error)
    importResult.value = null
  }
}

// Close the import result statistics card
const closeImportResult = async () => {
  importResultExpanded.value = false
  if (!props.kbId) return
  try {
    await updateFAQImportResultDisplayStatus(props.kbId, 'close')
    if (importResult.value) {
      importResult.value.display_status = 'close'
    }
  } catch (error) {
    console.error('Failed to close import result:', error)
  }
}

// Download failed entry reasons
const downloadFailedEntries = () => {
  if (!importResult.value?.failed_entries_url) {
    MessagePlugin.warning(t('faqManager.import.noFailedRecords'))
    return
  }
  // Open the download link directly
  window.open(importResult.value.failed_entries_url, '_blank')
}

// Format import time
const formatImportTime = (timeStr?: string) => {
  if (!timeStr) return ''
  try {
    const date = new Date(timeStr)
    return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')} ${String(date.getHours()).padStart(2, '0')}:${String(date.getMinutes()).padStart(2, '0')}`
  } catch (e) {
    return timeStr
  }
}

const handleImport = async () => {
  if (!importState.file || !importState.preview.length) {
    MessagePlugin.warning(t('knowledgeEditor.faqImport.selectFile'))
    return
  }

  // If the task has completed or failed, close the dialog
  if (importState.taskStatus?.status === 'success' || importState.taskStatus?.status === 'failed') {
    if (importState.taskStatus.status === 'success') {
      handleCancelImport()
    } else {
      // Retry on failure
      importState.taskId = null
      importState.taskStatus = null
      importState.importing = false
    }
    return
  }

  importState.importing = true
  try {
    const res: any = await upsertFAQEntries(props.kbId, {
      entries: importState.preview,
      mode: importState.mode,
    })

    const taskId = res?.data?.task_id
    if (taskId) {
      importState.taskId = taskId
      importState.taskStatus = {
        status: 'pending',
        progress: 0,
        total: importState.preview.length,
        processed: 0,
        message: t('faqManager.import.progressHint'),
      }
      // Start polling task status
      startPolling(taskId)
      // Close the import dialog immediately; progress will be shown at the top of the list page
      importVisible.value = false
      // Reset import dialog state (but keep taskId and taskStatus for progress display)
      importState.file = null
      importState.preview = []
      importState.importing = false
    } else {
      // If no task ID is returned, it may be an older API version — use the synchronous approach
      MessagePlugin.success(t('knowledgeEditor.faqImport.importSuccess'))
      importVisible.value = false
      await loadEntries()
      importState.importing = false
    }
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('common.operationFailed'))
    importState.importing = false
    stopPolling()
  }
}

// Watch for changes in selected count and notify the left-side menu
watch(selectedRowKeys, (newKeys, oldKeys) => {
  const count = newKeys.length
  // Get status info for the selected entries
  const selectedEntries = entries.value.filter(entry => newKeys.includes(entry.id))
  const enabledCount = selectedEntries.filter(entry => entry.is_enabled !== false).length
  const disabledCount = count - enabledCount

  const event = new CustomEvent('faqSelectionChanged', {
    detail: {
      count,
      enabledCount,
      disabledCount
    }
  })
  window.dispatchEvent(event)
}, { immediate: true, deep: true })

// Clean up polling when the component unmounts
onUnmounted(() => {
  stopPolling()
})

// Download sample file option
const downloadExampleOptions = computed(() => [
  { content: t('knowledgeEditor.faqImport.downloadExampleJSON'), value: 'json' },
  { content: t('knowledgeEditor.faqImport.downloadExampleCSV'), value: 'csv' },
  { content: t('knowledgeEditor.faqImport.downloadExampleExcel'), value: 'excel' },
])

// Example data
const exampleData: FAQEntryPayload[] = [
  {
    standard_question: 'What is WeKnora?',
    answers: ['WeKnora is an intelligent knowledge base management system', 'It supports multiple knowledge base types and import methods'],
    similar_questions: ['What is WeKnora?', 'Introduce WeKnora'],
    negative_questions: ['This is not WeKnora', 'Unrelated to WeKnora'],
    tag_name: 'Product Intro',
  },
  {
    standard_question: 'How do I create a knowledge base?',
    answers: ['Click the "New Knowledge Base" button', 'Select a knowledge base type and fill in the relevant information', 'You can start using it once created'],
    similar_questions: ['How to create a knowledge base?', 'How do I create a new one?'],
    negative_questions: [],
    tag_name: 'User guide',
  },
]

// Download sample file
const handleDownloadExample = (data: { value: string }) => {
  const { value } = data
  switch (value) {
    case 'json':
      downloadJSONExample()
      break
    case 'csv':
      downloadCSVExample()
      break
    case 'excel':
      downloadExcelExample()
      break
  }
}

// Download JSON sample
const downloadJSONExample = () => {
  const jsonStr = JSON.stringify(exampleData, null, 2)
  const blob = new Blob([jsonStr], { type: 'application/json;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = 'faq_example.json'
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}

// Download CSV sample
const downloadCSVExample = () => {
  const headers = ['Tag (required)', 'Question (required)', 'Similar questions (optional - separate multiple with ##)', 'Negative examples (optional - separate multiple with ##)', 'Bot answers (required - separate multiple with ##)', 'Reply all (optional - default FALSE)', 'Disabled (optional - default FALSE)', 'Disable recommendations (optional - default False)']
  const rows = exampleData.map((item) => {
    return [
      item.tag_name || '', // Tag
      item.standard_question,
      item.similar_questions.join('##'),
      item.negative_questions.join('##'),
      item.answers.join('##'),
      'FALSE', // Whether to reply to all
      'FALSE', // Whether to disable
      'FALSE', // Whether to forbid being recommended
    ]
  })
  const csvContent = [
    headers.join('\t'), // Use tab as delimiter
    ...rows.map((row) => row.map((cell) => {
      // If it contains tabs, newlines, or quotes, it needs to be wrapped in quotes
      if (cell.includes('\t') || cell.includes('\n') || cell.includes('"')) {
        return `"${cell.replace(/"/g, '""')}"`
      }
      return cell
    }).join('\t')),
  ].join('\n')
  const blob = new Blob(['\ufeff' + csvContent], { type: 'text/csv;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = 'faq_example.csv'
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}

// Download Excel sample
const downloadExcelExample = () => {
  const worksheet = XLSX.utils.json_to_sheet(
    exampleData.map((item) => ({
      'Tag (required)': item.tag_name || '',
      'Question (required)': item.standard_question,
      'Similar questions (optional - separate multiple with ##)': item.similar_questions.join('##'),
      'Negative examples (optional - separate multiple with ##)': item.negative_questions.join('##'),
      'Bot answers (required - separate multiple with ##)': item.answers.join('##'),
      'Reply all (optional - default FALSE)': 'FALSE',
      'Disabled (optional - default FALSE)': 'FALSE',
      'Disable recommendations (optional - default False)': 'FALSE',
    })),
  )
  const workbook = XLSX.utils.book_new()
  XLSX.utils.book_append_sheet(workbook, worksheet, 'FAQ')
  XLSX.writeFile(workbook, 'faq_example.xlsx')
}

// Export FAQ data
const exportLoading = ref(false)
const downloadExportBlob = (blob: Blob, ext: 'csv' | 'json') => {
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `faq_export_${new Date().toISOString().slice(0, 10)}.${ext}`
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}
const handleExportFAQ = async (format: 'csv' | 'json') => {
  if (!props.kbId) {
    MessagePlugin.warning(t('knowledgeBase.selectKnowledgeBase'))
    return
  }

  exportLoading.value = true
  try {
    const blob = await exportFAQEntries(props.kbId, format)
    downloadExportBlob(blob, format)
    MessagePlugin.success(t('knowledgeEditor.faqExport.exportSuccess'))
  } catch (error: any) {
    console.error('Export failed:', error)
    MessagePlugin.error(t('knowledgeEditor.faqExport.exportFailed'))
  } finally {
    exportLoading.value = false
  }
}
const handleExportCSV = () => handleExportFAQ('csv')
const handleExportJSON = () => handleExportFAQ('json')

watch(
  () => props.kbId,
  async (newKbId) => {
    currentPage = 1
    hasMore.value = true
    selectedTagIds.value = []
    tagFilterCleared.value = false
    uiStore.clearSelectedTagIds()
    overallFAQTotal.value = 0
    tagSearchQuery.value = ''

    if (!newKbId) {
      kbInfo.value = null
      // When kbId changes, clear the previous task state
      stopPolling()
      importState.taskId = null
      importState.taskStatus = null
      clearTaskIdFromStorage()
      return
    }

    const info = await loadKnowledgeInfo(newKbId)
    if (!info || info.type !== 'faq') {
      return
    }

    loadEntries()
    loadTags(true)
    // Restore import task status (if it exists)
    await restoreImportTask()
  },
  { immediate: true },
)

watch(selectedTagIds, (newVal, oldVal) => {
  if (oldVal === undefined) return
  if (newVal.join(',') !== oldVal.join(',')) {
    currentPage = 1
    entries.value = []
    selectedRowKeys.value = []
    loadEntries()
  }
})

watch(tagSearchQuery, (newVal, oldVal) => {
  if (newVal === oldVal) return
  if (tagSearchDebounce) {
    window.clearTimeout(tagSearchDebounce)
  }
  tagSearchDebounce = window.setTimeout(() => {
    loadTags(true)
  }, 300)
})

// Watch for changes in the FAQ search keyword
watch(entrySearchKeyword, (newVal, oldVal) => {
  if (newVal === oldVal) return
  if (entrySearchDebounce) {
    window.clearTimeout(entrySearchDebounce)
  }
  entrySearchDebounce = window.setTimeout(() => {
    loadEntries()
  }, 300)
})

const handleSearch = async () => {
  if (!searchForm.query.trim()) {
    MessagePlugin.warning(t('knowledgeEditor.faq.queryPlaceholder'))
    return
  }

  searching.value = true
  hasSearched.value = true
  try {
    const res = await searchFAQEntries(props.kbId, {
      query_text: searchForm.query.trim(),
      vector_threshold: searchForm.vectorThreshold,
      match_count: searchForm.matchCount,
    })
    const results = (res.data || []).map((entry: FAQEntry) => ({
      ...entry,
      similarCollapsed: true,  // Similar questions collapsed by default
      negativeCollapsed: true,  // Counterexamples collapsed by default
      answersCollapsed: true,   // Answers collapsed by default
      expanded: false,
    })) as FAQEntry[]

    // Sort by score, descending
    searchResults.value = results.sort((a, b) => (b.score || 0) - (a.score || 0))
  } catch (error: any) {
    MessagePlugin.error(error?.message || t('common.operationFailed'))
    searchResults.value = []
  } finally {
    searching.value = false
  }
}

const getMatchTypeLabel = (matchType?: string) => {
  if (!matchType) return ''
  if (matchType === 'embedding') {
    return t('knowledgeEditor.faq.matchTypeEmbedding')
  }
  if (matchType === 'keywords') {
    return t('knowledgeEditor.faq.matchTypeKeywords')
  }
  return matchType
}

const toggleResult = (result: FAQEntry) => {
  result.expanded = !result.expanded
}

// Debounce function
let arrangeCardsTimer: ReturnType<typeof setTimeout> | null = null
const debounceArrangeCards = (delay = 100) => {
  if (arrangeCardsTimer) {
    clearTimeout(arrangeCardsTimer)
  }
  arrangeCardsTimer = setTimeout(() => {
    arrangeCards()
    arrangeCardsTimer = null
  }, delay)
}

// Waterfall layout function — optimized version to avoid flicker
const arrangeCards = () => {
  if (!cardListRef.value) return

  const cards = cardListRef.value.querySelectorAll('.faq-card') as NodeListOf<HTMLElement>
  if (cards.length === 0) return

  // Get container width and column count
  const containerWidth = cardListRef.value.offsetWidth
  const gap = 12 // Keep consistent with CSS gap
  let columnCount = 1

  // Compute column count based on container width (increase cards per row)
  if (containerWidth >= 2560) columnCount = 12
  else if (containerWidth >= 1920) columnCount = 10
  else if (containerWidth >= 1536) columnCount = 8
  else if (containerWidth >= 1280) columnCount = 6
  else if (containerWidth >= 1024) columnCount = 5
  else if (containerWidth >= 768) columnCount = 4
  else if (containerWidth >= 640) columnCount = 3

  const columnWidth = (containerWidth - (gap * (columnCount - 1))) / columnCount

  // Initialize the height array for each column
  const columnHeights = new Array(columnCount).fill(0)

  // Use requestAnimationFrame to optimize performance
  requestAnimationFrame(() => {
    // Set width first, keeping the current position unchanged
    cards.forEach((card) => {
      // Ensure the card is absolutely positioned
      if (card.style.position !== 'absolute') {
        card.style.position = 'absolute'
      }
      // Set width so the height can be calculated correctly
      card.style.width = `${columnWidth}px`
    })

    // Wait for the browser to recompute layout
    requestAnimationFrame(() => {
      // Compute the heights of all cards (without changing position)
      const cardHeights: number[] = []
      cards.forEach((card) => {
        const height = card.offsetHeight || card.getBoundingClientRect().height
        cardHeights.push(height)
      })

      // Compute new positions
      const newPositions: Array<{ top: number; left: number }> = []
      cardHeights.forEach((height) => {
        const shortestColumnIndex = columnHeights.indexOf(Math.min(...columnHeights))
        const top = columnHeights[shortestColumnIndex]
        const left = shortestColumnIndex * (columnWidth + gap)

        newPositions.push({ top, left })
        columnHeights[shortestColumnIndex] += height + gap
      })

      // Batch-update all card positions, using CSS transitions for smooth movement
      cards.forEach((card, index) => {
        const { top, left } = newPositions[index]
        const currentTop = parseFloat(card.style.top) || 0
        const currentLeft = parseFloat(card.style.left) || 0

        // If the position has changed, add a transition effect
        if (Math.abs(currentTop - top) > 1 || Math.abs(currentLeft - left) > 1) {
          // Use will-change to hint the browser to optimize
          card.style.willChange = 'top, left'
          card.style.transition = 'top 0.3s cubic-bezier(0.4, 0, 0.2, 1), left 0.3s cubic-bezier(0.4, 0, 0.2, 1)'
        }

        card.style.position = 'absolute'
        card.style.top = `${top}px`
        card.style.left = `${left}px`
        card.style.width = `${columnWidth}px`
      })

      // Set container height
      const maxHeight = Math.max(...columnHeights)
      if (cardListRef.value) {
        cardListRef.value.style.height = `${maxHeight}px`
        cardListRef.value.style.position = 'relative'
      }

      // Remove transition and will-change after the animation completes, to avoid affecting subsequent interactions
      setTimeout(() => {
        cards.forEach((card) => {
          card.style.transition = ''
          card.style.willChange = ''
        })
      }, 300)
    })
  })
}

// Listen for window resize events (debounced)
let resizeTimer: ReturnType<typeof setTimeout> | null = null
const handleResize = () => {
  if (resizeTimer) {
    clearTimeout(resizeTimer)
  }
  resizeTimer = setTimeout(() => {
    arrangeCards()
    // When the window grows, more items may need to load — delay execution to ensure layout is complete
    setTimeout(() => {
      checkAndLoadMore()
    }, 350)
    resizeTimer = null
  }, 150)
}

onMounted(async () => {
  // Ensure shared knowledge bases are loaded before loading the knowledge list
  orgStore.fetchSharedKnowledgeBases()
  loadKnowledgeList()
  window.addEventListener('resize', handleResize)
  window.addEventListener('faqMenuAction', handleFaqMenuAction as EventListener)
  // If a kbId already exists, restore the import task status
  if (props.kbId) {
    await restoreImportTask()
    await loadImportResult() // Load import results
  }
  // Proactively trigger a selected-count event once, to ensure the left menu receives the initial state
  nextTick(() => {
    const count = selectedRowKeys.value.length
    const selectedEntries = entries.value.filter(entry => selectedRowKeys.value.includes(entry.id))
    const enabledCount = selectedEntries.filter(entry => entry.is_enabled !== false).length
    const disabledCount = count - enabledCount
    window.dispatchEvent(new CustomEvent('faqSelectionChanged', {
      detail: {
        count,
        enabledCount,
        disabledCount
      }
    }))
  })
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
  window.removeEventListener('faqMenuAction', handleFaqMenuAction as EventListener)
  if (arrangeCardsTimer) {
    clearTimeout(arrangeCardsTimer)
  }
  if (resizeTimer) {
    clearTimeout(resizeTimer)
  }
})

// Watch for changes in entries, then re-layout
watch(() => entries.value.length, () => {
  nextTick(() => {
    arrangeCards()
  })
})

// Watch for collapse-state changes, then re-layout (debounced, with a callback after the animation completes)
watch(() => entries.value.map(e => ({
  id: e.id,
  similarCollapsed: e.similarCollapsed,
  negativeCollapsed: e.negativeCollapsed,
  answersCollapsed: e.answersCollapsed
})), () => {
  // Use nextTick to ensure the DOM has updated
  nextTick(() => {
    // Wait one render frame for the height change to take effect
    requestAnimationFrame(() => {
      // Wait one more render frame to ensure the height calculation is accurate
      requestAnimationFrame(() => {
        // Wait for the Transition animation to finish before laying out (slide-down animation duration is about 200ms)
        // Use debounce to avoid frequent calls
        debounceArrangeCards(250)
      })
    })
  })
}, { deep: true })
</script>

<style lang="less">
/* Dropdown menu styles have been unified into @/assets/dropdown-menu.less */
.tag-filter-popup {
  z-index: 5500 !important;
}

.tag-filter-popup .t-popup__content {
  padding: 0 !important;
  border-radius: 8px !important;
  background: var(--td-bg-color-container) !important;
  border: 0.5px solid var(--td-component-stroke) !important;
  box-shadow:
    0 0 0 0.5px rgba(0, 0, 0, 0.03),
    0 2px 4px rgba(0, 0, 0, 0.04),
    0 8px 24px rgba(0, 0, 0, 0.1) !important;
}

.tag-filter-panel {
  width: 320px;
  max-width: min(320px, calc(100vw - 32px));
  max-height: min(70vh, 480px);
  display: flex;
  flex-direction: column;
  padding: 12px 14px;
  box-sizing: border-box;
  font-size: 12px;
  color: var(--td-text-color-primary);
}

.tag-filter-panel__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
}

.tag-filter-panel__title {
  display: flex;
  align-items: baseline;
  gap: 6px;
  font-size: 14px;
  font-weight: 600;
}

.tag-filter-panel__count {
  font-size: 12px;
  color: var(--td-text-color-placeholder);
  font-weight: 400;
}

.tag-filter-panel .tag-search-bar {
  margin-bottom: 10px;
}

.tag-filter-panel__body {
  display: flex;
  flex-direction: column;
  gap: 8px;
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}

.tag-filter-chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.tag-filter-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 24px;
  padding: 0 8px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 4px;
  background: transparent;
  color: var(--td-text-color-secondary);
  font-size: 11px;
  cursor: pointer;
}

.tag-filter-chip.active {
  border-color: color-mix(in srgb, var(--td-brand-color) 35%, var(--td-component-stroke));
  color: var(--td-brand-color);
  background-color: color-mix(in srgb, var(--td-brand-color) 6%, transparent);
}

.tag-filter-chip__label {
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tag-filter-chip__count {
  font-size: 10px;
  color: var(--td-text-color-placeholder);
}

.tag-filter-panel__footer {
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px solid var(--td-component-stroke);
}

.tag-empty-state {
  text-align: center;
  padding: 10px 6px;
  color: var(--td-text-color-placeholder);
  font-size: 11px;
}

.tag-load-more {
  display: flex;
  justify-content: center;
  padding-top: 2px;
}
</style>
<style scoped lang="less">
.faq-manager {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.15s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

.faq-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  gap: 20px;
}

// Consistent with the list page: light gray rounded background, filter panel as a white card on the left
.faq-main {
  display: flex;
  flex: 1;
  min-height: 0;
  background: transparent;
  border: none;
}

.faq-card-area {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  min-height: 0;
  padding: 0;
  border: none;
  overflow: hidden;
  background: transparent;
}

.faq-filter-bar {
  padding: 0 0 12px 0;
  flex-shrink: 0;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px 12px;

  .faq-search-input {
    flex: 1 1 220px;
    min-width: 0;
    width: auto;
  }

  &__filters {
    flex: 0 0 auto;
    display: flex;
    align-items: center;
    gap: 12px;
    min-width: 0;

    :deep(.t-popup__reference) {
      display: block;
    }
  }

  &__trailing {
    flex: 0 0 auto;
    display: flex;
    align-items: center;
    gap: 4px;
    margin-left: auto;

    :deep(.content-bar-icon-btn) {
      color: var(--td-text-color-secondary);
      background: transparent;
      border: none;

      &:hover {
        color: var(--td-brand-color);
        background: var(--td-bg-color-secondarycontainer);
      }
    }
  }

  @media (max-width: 767px) {
    .faq-search-input {
      flex: 1 1 100%;
    }

    &__filters {
      flex: 1 1 auto;
      min-width: 0;
    }

    &__trailing {
      flex: 0 0 auto;
      margin-left: auto;
    }
  }

  .doc-filter-field {
    width: 140px;
    flex-shrink: 0;

    &__control {
      width: 100%;
    }
  }

  .doc-tag-filter-trigger {
    display: inline-flex;
    align-items: center;
    box-sizing: border-box;
    width: 100%;
    height: 32px;
    padding: 0 8px;
    border: 1px solid transparent;
    border-radius: var(--td-radius-default);
    background: var(--td-bg-color-secondarycontainer);
    color: var(--td-text-color-primary);
    font-family: var(--app-font-family);
    font-size: 14px;
    line-height: 1;
    cursor: pointer;
    transition: background 0.2s ease, border-color 0.2s ease;

    &:hover,
    &.open {
      background: var(--td-bg-color-secondarycontainer);
      border-color: transparent;
    }

    &.is-placeholder {
      color: var(--td-text-color-placeholder);
    }

    &__prefix {
      flex-shrink: 0;
      display: inline-flex;
      align-items: center;
      margin-right: var(--td-comp-margin-s);
      color: var(--td-text-color-placeholder);
    }

    &__label {
      flex: 1;
      min-width: 0;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
      text-align: left;
    }

    &__suffix {
      flex-shrink: 0;
      display: inline-flex;
      align-items: center;
      margin-left: var(--td-comp-margin-s);
    }

    &__caret {
      flex-shrink: 0;
      color: var(--td-text-color-placeholder);
      transition: transform 0.2s ease, color 0.2s ease;

      &.open {
        color: var(--td-brand-color);
        transform: rotate(180deg);
      }
    }
  }

  :deep(.t-input) {
    font-size: 13px;
    background-color: var(--td-bg-color-secondarycontainer);
    border-color: transparent;
    border-radius: 6px;
    box-shadow: none !important;

    &:hover,
    &:focus,
    &.t-is-focused {
      border-color: var(--td-brand-color);
      background-color: var(--td-bg-color-container);
      box-shadow: none !important;
    }
  }

  :deep(.t-input__prefix-icon) {
    margin-right: 0;
  }
}

:deep(.tag-menu) {
  display: flex;
  flex-direction: column;
}

:deep(.tag-menu-item) {
  display: flex;
  align-items: center;
  padding: 8px 16px;
  cursor: pointer;
  transition: all 0.2s ease;
  color: var(--td-text-color-primary);
  font-family: var(--app-font-family);
  font-size: 14px;
  font-weight: 400;

  .menu-icon {
    margin-right: 8px;
    font-size: 16px;
  }

  &:hover {
    background: var(--td-bg-color-secondarycontainer);
    color: var(--td-text-color-primary);
  }

  &.danger {
    color: var(--td-text-color-primary);

    &:hover {
      background: var(--td-error-color-light);
      color: var(--td-error-color);

      .menu-icon {
        color: var(--td-error-color);
      }
    }
  }
}

.faq-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 12px;
  flex-shrink: 0;

  .faq-header-title {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .faq-title-row {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-wrap: wrap;
    width: 100%;

    .faq-import-strip--in-title {
      margin-bottom: 0;
      flex: 0 1 auto;
      min-width: 0;
      max-width: min(420px, 40vw);

      .faq-import-strip__text {
        max-width: 220px;
      }
    }
  }

  .kb-title-actions {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    flex-shrink: 0;
  }

  .faq-breadcrumb {
    display: flex;
    align-items: center;
    gap: 6px;
    margin: 0;
    font-size: 20px;
    font-weight: 600;
    color: var(--td-text-color-primary);
  }

  .breadcrumb-link {
    border: none;
    background: transparent;
    padding: 4px 8px;
    margin: -4px -8px;
    font: inherit;
    color: var(--td-text-color-secondary);
    cursor: pointer;
    display: inline-flex;
    align-items: center;
    gap: 4px;
    border-radius: 6px;
    transition: all 0.12s ease;

    &:hover:not(:disabled) {
      color: var(--td-success-color);
      background: var(--td-bg-color-container);
    }

    &:disabled {
      cursor: not-allowed;
      color: var(--td-text-color-placeholder);
    }

    &.dropdown {
      padding-right: 6px;

      :deep(.t-icon) {
        font-size: 14px;
        transition: transform 0.12s ease;
      }

      &:hover:not(:disabled) {
        :deep(.t-icon) {
          transform: translateY(1px);
        }
      }
    }
  }

  .breadcrumb-separator {
    font-size: 14px;
    color: var(--td-text-color-placeholder);
  }

  .breadcrumb-current {
    color: var(--td-text-color-primary);
    font-weight: 600;
  }

  h2 {
    margin: 0;
    color: var(--td-text-color-primary);
    font-family: var(--app-font-family);
    font-size: 24px;
    font-weight: 600;
    line-height: 32px;
  }

  .faq-subtitle {
    margin: 0;
    color: var(--td-text-color-placeholder);
    font-family: var(--app-font-family);
    font-size: 14px;
    font-weight: 400;
    line-height: 20px;
  }
}


// Import results entry: icon only by default, expands on hover / click
.faq-import-host {
  position: relative;
  flex-shrink: 0;

  .faq-import-trigger {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    border: none;
    background: transparent;
    padding: 2px;
    margin: 0;
    color: var(--td-success-color);
    cursor: pointer;
    line-height: 1;
    transition: opacity 0.15s ease;

    &:hover {
      opacity: 0.75;
    }
  }

  .faq-import-panel {
    position: absolute;
    top: calc(100% + 8px);
    left: 0;
    z-index: 200;
    opacity: 0;
    visibility: hidden;
    pointer-events: none;
    transform: translateY(-4px);
    transition: opacity 0.15s ease, transform 0.15s ease, visibility 0.15s ease;
  }

  &:hover .faq-import-panel,
  &.is-expanded .faq-import-panel,
  &:focus-within .faq-import-panel {
    opacity: 1;
    visibility: visible;
    pointer-events: auto;
    transform: translateY(0);
  }

  .faq-import-strip--panel {
    margin-bottom: 0;
    padding: 8px 10px;
    font-size: 12px;
    white-space: nowrap;
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.1);

    .faq-import-strip__text {
      max-width: 360px;
    }
  }
}


// FAQ import notice bar (compact single line)
.faq-import-strip {
  display: inline-flex;
  align-items: center;
  width: fit-content;
  gap: 8px;
  max-width: 100%;
  margin-bottom: 10px;
  padding: 4px 8px 4px 10px;
  border-radius: 6px;
  font-size: 12px;
  line-height: 1.4;
  color: var(--td-text-color-secondary);
  background: var(--td-bg-color-secondarycontainer);
  border: 1px solid var(--td-component-stroke);

  &__icon {
    flex-shrink: 0;
    color: var(--td-text-color-placeholder);

    &.is-spinning {
      animation: faq-import-spin 1s linear infinite;
    }
  }

  &__text {
    flex: 0 1 auto;
    min-width: 0;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 520px;
  }

  &__bar {
    flex-shrink: 0;
    width: 72px;
    height: 4px;
    border-radius: 2px;
    background: rgba(0, 0, 0, 0.08);
    overflow: hidden;
  }

  &__bar-fill {
    height: 100%;
    border-radius: 2px;
    background: var(--td-brand-color);
    transition: width 0.3s ease;
  }

  &__count {
    flex-shrink: 0;
    font-size: 12px;
    font-variant-numeric: tabular-nums;
    color: var(--td-text-color-placeholder);
  }

  &__time {
    flex-shrink: 0;
    font-size: 12px;
    color: var(--td-text-color-placeholder);
    white-space: nowrap;
  }

  &__link {
    flex-shrink: 0;
    padding: 0 4px;
    height: auto;
    font-size: 12px;
  }

  &__close {
    display: inline-flex;
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
    transition: background 0.15s ease, color 0.15s ease;

    &:hover {
      background: rgba(0, 0, 0, 0.06);
      color: var(--td-text-color-secondary);
    }
  }

  &--result {
    .faq-import-strip__icon {
      color: var(--td-success-color);
    }
  }

  &--running {
    .faq-import-strip__icon {
      color: var(--td-brand-color);
    }
  }

  &--success {
    .faq-import-strip__icon {
      color: var(--td-success-color);
    }

    .faq-import-strip__bar-fill {
      background: var(--td-success-color);
    }
  }

  &--failed {
    border-color: rgba(227, 77, 89, 0.3);
    background: rgba(227, 77, 89, 0.06);

    .faq-import-strip__icon {
      color: var(--td-error-color);
    }

    .faq-import-strip__text {
      color: var(--td-error-color);
    }

    .faq-import-strip__bar-fill {
      background: var(--td-error-color);
    }
  }
}

@keyframes faq-import-spin {
  from {
    transform: rotate(0deg);
  }

  to {
    transform: rotate(360deg);
  }
}


.tag-filter-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;

  .tag-filter-label {
    color: var(--td-text-color-secondary);
    font-size: 14px;
  }
}


.kb-settings-button {
  width: 30px;
  height: 30px;
  border: none;
  border-radius: 50%;
  background: var(--td-bg-color-secondarycontainer);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: var(--td-text-color-secondary);
  cursor: pointer;
  transition: all 0.2s ease;
  padding: 0;

  &:hover:not(:disabled) {
    background: var(--td-success-color-light);
    color: var(--td-brand-color);
  }

  &:disabled {
    cursor: not-allowed;
    opacity: 0.4;
  }

  :deep(.t-icon) {
    font-size: 18px;
  }
}

// Scroll container
.faq-scroll-container {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  padding-right: 4px;
}

@keyframes contentFadeIn {
  from {
    opacity: 0;
    transform: translateY(6px);
  }

  to {
    opacity: 1;
    transform: translateY(0);
  }
}

.faq-skeleton-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 12px;
  width: 100%;
  animation: contentFadeIn 0.32s ease-out;
}

.faq-card-skeleton {
  cursor: default;
  height: auto;

  .faq-card-header {
    padding-bottom: 10px;
    border-bottom: 1px solid var(--td-component-stroke);
  }

  .faq-card-body {
    padding: 8px 0;
  }

  .faq-skel-footer {
    padding-top: 8px;
    border-top: 1px solid var(--td-component-stroke);
  }
}

// Card list style - uses absolute positioning for waterfall layout, next row fills gaps from previous row
.faq-card-list {
  position: relative;
  width: 100%;
  animation: contentFadeIn 0.32s ease-out;
  min-width: 0;
}

.faq-card {
  border: 1px solid var(--td-component-stroke);
  border-radius: 10px;
  background: var(--td-bg-color-container);
  padding: 10px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
  max-width: 100%;
  overflow: hidden;
  cursor: pointer;
  transition: border-color 0.2s ease, box-shadow 0.2s ease, background-color 0.2s ease;
  box-sizing: border-box;
  height: fit-content;

  &:hover {
    border-color: var(--td-brand-color);
    box-shadow: 0 2px 8px rgba(7, 192, 95, 0.1);
  }

  &.selected {
    border-color: var(--td-brand-color);
    background: var(--td-success-color-light);
    box-shadow: 0 2px 8px rgba(7, 192, 95, 0.15);
  }
}

.faq-card-header {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding-bottom: 10px;
  border-bottom: 1px solid var(--td-component-stroke);
  position: relative;
}

.faq-header-top {
  display: flex;
  align-items: flex-start;
  gap: 10px;
}

.faq-card-actions {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-left: auto;
  flex-shrink: 0;
}

.faq-header-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  padding-top: 5px;
  border-top: 1px dashed var(--td-component-stroke);
}

.faq-meta-item {
  display: inline-flex;
  align-items: baseline;
  gap: 5px;
  padding: 3px 8px;
  border-radius: 999px;
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);

  .meta-label {
    font-size: 11px;
    color: var(--td-text-color-secondary);
    font-weight: 500;
  }

  .meta-value {
    font-size: 12px;
    color: var(--td-text-color-primary);
    font-weight: 600;
  }
}

.faq-card-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 6px;
  padding: 8px 12px;
  margin: 0 -10px -10px;
  background: rgba(48, 50, 54, 0.02);
  border-top: 1px solid var(--td-component-stroke);
  flex-wrap: nowrap;
}

.faq-card-status {
  display: flex;
  align-items: center;
  gap: 5px;
  flex-shrink: 0;
  margin-left: auto;
}

.status-item {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 3px 8px;
  border-radius: 999px;
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  font-size: 11px;
  color: var(--td-text-color-secondary);
  font-family: var(--app-font-family);

  .status-icon {
    font-size: 13px;
    color: var(--td-text-color-placeholder);

    &.warning {
      color: var(--td-warning-color);
    }

    &.success {
      color: var(--td-success-color);
    }
  }
}

.status-item-compact {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 2px 4px;
  border-radius: 4px;
  background: transparent;
  border: none;
  cursor: pointer;
  transition: all 0.2s ease;

  &:hover {
    background: var(--td-bg-color-container-hover);
  }

  .status-icon {
    font-size: 16px;
    flex-shrink: 0;

    &.warning {
      color: var(--td-warning-color);
    }

    &.success {
      color: var(--td-success-color);
    }
  }

  :deep(.t-switch) {
    flex-shrink: 0;
  }
}

.faq-card-tag {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  flex: 1;
  min-width: 0;

  :deep(.t-tag) {
    display: inline-flex;
    align-items: center;
    cursor: pointer;
    max-width: 120px;
    height: 20px;
    border-radius: 4px;
    border-color: var(--td-component-stroke);
    color: var(--td-text-color-disabled);
    padding: 0 6px;
    background: var(--td-bg-color-container-hover);
    font-size: 11px;
    font-weight: 400;
    font-family: var(--app-font-family);
    transition: all 0.2s ease;

    &:hover {
      border-color: var(--td-brand-color);
      color: var(--td-brand-color-active);
      background: var(--td-success-color-light);
    }
  }
}

.faq-tag-chip {
  display: inline-flex;
  align-items: center;
  cursor: pointer;

  .tag-text {
    max-width: 100px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 11px;
    font-weight: 400;
    color: var(--td-text-color-disabled);
  }
}

.card-more-btn {
  display: flex;
  width: 28px;
  height: 28px;
  justify-content: center;
  align-items: center;
  border-radius: 6px;
  cursor: pointer;
  flex-shrink: 0;
  opacity: 0.6;

  &:hover {
    background: var(--td-bg-color-secondarycontainer);
    opacity: 1;
  }

  &.mobile {
    display: none;
  }

  .more-icon {
    width: 16px;
    height: 16px;
  }
}

/* card-menu style unified into @/assets/dropdown-menu.less, using the .popup-menu class */

.faq-question {
  flex: 1;
  color: var(--td-text-color-primary);
  font-family: var(--app-font-family);
  font-size: 15px;
  font-weight: 600;
  line-height: 1.5;
  word-break: break-word;
  min-width: 0;
  overflow: hidden;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
}

.faq-card-body {
  display: flex;
  flex-direction: column;
  gap: 6px;
  flex: 1;
  min-width: 0;
  overflow: hidden;
  contain: layout;
}

.faq-section {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
  overflow: hidden;

  .faq-section-label {
    color: var(--td-text-color-secondary);
    font-family: var(--app-font-family);
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    display: flex;
    align-items: center;
    gap: 5px;
    margin-bottom: 1px;

    &::before {
      content: '';
      width: 3px;
      height: 10px;
      background: var(--td-brand-color);
      border-radius: 2px;
      flex-shrink: 0;
    }

    &.clickable {
      cursor: pointer;
      user-select: none;
      padding: 2px 0;
      border-radius: 4px;

      &:hover {
        color: var(--td-text-color-primary);
        background: var(--td-bg-color-container);
        padding-left: 4px;
        padding-right: 4px;
        margin-left: -4px;
        margin-right: -4px;
      }
    }

    .collapse-icon {
      font-size: 13px;
      color: var(--td-text-color-placeholder);
      flex-shrink: 0;
      margin-left: auto; // Align arrow to the right
    }

    .section-count {
      color: var(--td-text-color-placeholder);
      font-weight: 400;
      margin-left: 4px;
    }
  }

  &.answers .faq-section-label::before {
    background: var(--td-brand-color);
  }

  &.similar .faq-section-label::before {
    background: var(--td-brand-color);
  }

  &.negative .faq-section-label::before {
    background: var(--td-warning-color);
  }
}

.faq-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
  min-height: 18px;
  min-width: 0;
  width: 100%;
  overflow: hidden;
  contain: layout style paint; // Optimize rendering performance

  // Ensure each tag has a max-width limit
  >* {
    max-width: 100%;
    min-width: 0;
    flex: 0 1 auto;
  }

  // Limit max width when tag is on its own line
  >*:first-child:last-child {
    max-width: 100%;
  }
}

.question-tag {
  font-size: 11px;
  padding: 3px 8px;
  max-width: 100%;
  min-width: 0;
  border-radius: 5px;
  font-family: var(--app-font-family);
  flex: 0 1 auto;

  :deep(.t-tag) {
    max-width: 100% !important;
    min-width: 0 !important;
    width: auto !important;
    display: inline-flex !important;
    align-items: center;
    vertical-align: middle;
    overflow: hidden !important;
    box-sizing: border-box;
    background: var(--td-bg-color-container);
    border-color: var(--td-component-stroke);
    color: var(--td-text-color-primary);
  }

  // Targets the span element inside TDesign tag
  :deep(.t-tag span),
  :deep(.t-tag > span) {
    display: block !important;
    overflow: hidden !important;
    text-overflow: ellipsis !important;
    white-space: nowrap !important;
    max-width: 100% !important;
    width: auto !important;
    line-height: 1.4;
    min-width: 0 !important;
  }
}

// Ensure the tag itself doesn't overflow the container
.faq-tags :deep(.t-tag) {
  max-width: 100%;
  min-width: 0;
  flex-shrink: 1;
}

.faq-tags :deep(.faq-tag-wrapper) {
  max-width: 100%;
  min-width: 0;
  flex-shrink: 1;
}

.empty-tip {
  color: var(--td-text-color-placeholder);
  font-size: 12px;
  font-style: italic;
  padding: 8px 0;
  font-family: var(--app-font-family);
}


.faq-load-more,
.faq-no-more {
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 24px 16px;
  color: var(--td-text-color-secondary);
  font-size: 13px;
  font-family: var(--app-font-family);
}

.faq-no-more {
  color: var(--td-text-color-placeholder);
  font-style: italic;
}

// Empty state style
.faq-empty-state {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 400px;
  padding: 60px 20px;

  .empty-content {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 16px;
    text-align: center;
    max-width: 400px;
  }

  .empty-icon {
    color: var(--td-text-color-disabled);
    opacity: 0.6;
  }

  .empty-text {
    color: var(--td-text-color-primary);
    font-family: var(--app-font-family);
    font-size: 18px;
    font-weight: 600;
    line-height: 28px;
  }

  .empty-desc {
    color: var(--td-text-color-secondary);
    font-family: var(--app-font-family);
    font-size: 14px;
    font-weight: 400;
    line-height: 22px;
  }
}

// Import dialog style - consistent with the create knowledge base dialog style
.faq-import-overlay {
  position: fixed;
  inset: 0;
  z-index: 1000;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
  backdrop-filter: blur(4px);
}

.faq-import-modal {
  position: relative;
  width: 100%;
  max-width: 600px;
  max-height: 90vh;
  background: var(--td-bg-color-container);
  border-radius: 12px;
  box-shadow: 0 6px 28px rgba(15, 23, 42, 0.08);
  overflow: hidden;
  display: flex;
  flex-direction: column;

  .close-btn {
    position: absolute;
    top: 20px;
    right: 20px;
    width: 32px;
    height: 32px;
    border: none;
    background: var(--td-bg-color-secondarycontainer);
    border-radius: 6px;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--td-text-color-secondary);
    transition: all 0.2s ease;
    z-index: 10;

    &:hover {
      background: var(--td-bg-color-secondarycontainer);
      color: var(--td-text-color-primary);
    }
  }
}

.faq-import-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.faq-import-header {
  padding: 24px 24px 16px;
  border-bottom: 1px solid var(--td-component-stroke);
  flex-shrink: 0;

  .import-title {
    margin: 0;
    font-family: var(--app-font-family);
    font-size: 18px;
    font-weight: 600;
    color: var(--td-text-color-primary);
  }
}

.faq-import-content {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 24px;
  min-height: 0;
  max-height: calc(90vh - 140px); // Subtract header and footer height

  // Custom scrollbar
  &::-webkit-scrollbar {
    width: 6px;
  }

  &::-webkit-scrollbar-track {
    background: var(--td-bg-color-secondarycontainer);
    border-radius: 3px;
  }

  &::-webkit-scrollbar-thumb {
    background: var(--td-bg-color-component-disabled);
    border-radius: 3px;
    transition: background 0.2s;

    &:hover {
      background: var(--td-brand-color);
    }
  }
}

.faq-import-footer {
  padding: 16px 24px;
  border-top: 1px solid var(--td-component-stroke);
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  flex-shrink: 0;
}

// Import form item
.import-form-item {
  margin-bottom: 24px;

  &:last-child {
    margin-bottom: 0;
  }
}

// File tag row
.file-label-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
  gap: 12px;
}

// Download sample button
.download-example-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  font-family: var(--app-font-family);
  font-size: 13px;
  font-weight: 500;
  padding: 6px 14px;
  border-radius: 6px;
  border: 1px solid var(--td-component-stroke);
  background: var(--td-bg-color-container);
  color: var(--td-text-color-primary);
  transition: all 0.2s ease;
  cursor: pointer;
  white-space: nowrap;

  &:hover {
    border-color: var(--td-brand-color);
    color: var(--td-brand-color);
    background: var(--td-success-color-light);
  }

  &:active {
    background: var(--td-success-color-light);
  }

  :deep(.t-icon) {
    font-size: 16px;
  }
}

// Import form label
.import-form-label {
  display: block;
  margin-bottom: 0;
  font-family: var(--app-font-family);
  font-size: 14px;
  font-weight: 500;
  color: var(--td-text-color-primary);
  letter-spacing: -0.2px;
  flex: 1;

  &.required::after {
    content: '*';
    color: var(--td-error-color);
    margin-left: 4px;
    font-weight: 600;
  }
}

// File upload wrapper
.file-upload-wrapper {
  width: 100%;
}

// Hidden file input
.file-input-hidden {
  position: absolute;
  width: 0;
  height: 0;
  opacity: 0;
  overflow: hidden;
  pointer-events: none;
}

// File upload area
.file-upload-area {
  position: relative;
  width: 100%;
  min-height: 120px;
  border: 2px dashed var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-secondarycontainer);
  cursor: pointer;
  transition: all 0.3s ease;
  display: flex;
  align-items: center;
  justify-content: center;

  &:hover {
    border-color: var(--td-brand-color);
    background: var(--td-success-color-light);
  }

  &.has-file {
    border-color: var(--td-brand-color);
    background: var(--td-success-color-light);
    border-style: solid;
  }
}

// File upload content
.file-upload-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  text-align: center;
}

.upload-icon {
  color: var(--td-brand-color);
  transition: transform 0.2s ease;
}

.file-upload-area:hover .upload-icon {
  transform: translateY(-2px);
}

.upload-text {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.upload-primary-text {
  font-family: var(--app-font-family);
  font-size: 14px;
  font-weight: 500;
  color: var(--td-text-color-primary);
}

.upload-secondary-text {
  font-family: var(--app-font-family);
  font-size: 12px;
  color: var(--td-text-color-secondary);
}

.upload-file-name {
  font-family: var(--app-font-family);
  font-size: 14px;
  font-weight: 500;
  color: var(--td-brand-color);
  word-break: break-all;
}

// Import form hint
.import-form-tip {
  margin-top: 8px;
  font-family: var(--app-font-family);
  font-size: 12px;
  color: var(--td-text-color-disabled);
  line-height: 18px;
}

// Preview area
.import-preview {
  margin-top: 20px;
  padding: 16px;
  background: var(--td-bg-color-secondarycontainer);
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
}

.preview-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--td-component-stroke);
}

.preview-icon {
  color: var(--td-brand-color);
  flex-shrink: 0;
}

.preview-title {
  font-family: var(--app-font-family);
  font-size: 14px;
  font-weight: 500;
  color: var(--td-text-color-primary);
}

.preview-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 8px;
}

.preview-item {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 10px 12px;
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: 6px;
  transition: all 0.2s ease;

  &:hover {
    border-color: var(--td-brand-color);
    box-shadow: 0 2px 4px rgba(7, 192, 95, 0.08);
  }
}

.preview-index {
  flex-shrink: 0;
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, var(--td-brand-color) 0%, var(--td-brand-color-active) 100%);
  color: var(--td-text-color-anti);
  border-radius: 4px;
  font-family: var(--app-font-family);
  font-size: 12px;
  font-weight: 600;
}

.preview-question {
  flex: 1;
  font-family: var(--app-font-family);
  font-size: 13px;
  color: var(--td-text-color-primary);
  line-height: 1.5;
  word-break: break-word;
}

.preview-more {
  margin-top: 8px;
  padding-top: 8px;
  border-top: 1px solid var(--td-component-stroke);
  font-family: var(--app-font-family);
  font-size: 12px;
  color: var(--td-text-color-secondary);
  text-align: center;
}

// Responsive layout is computed dynamically by JavaScript, no media query needed here

// Card menu popup style unified into @/assets/dropdown-menu.less

// FAQ editor drawer style
:deep(.faq-editor-drawer) {
  .t-drawer__body {
    padding: 20px;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
    height: 100%;
  }

  .t-drawer__header {
    padding: 20px 24px;
    border-bottom: 1px solid var(--td-component-stroke);
    font-family: var(--app-font-family);
    font-size: 18px;
    font-weight: 600;
    color: var(--td-text-color-primary);
  }

  .t-drawer__footer {
    padding: 16px 24px;
    border-top: 1px solid var(--td-component-stroke);
  }
}

.faq-editor-drawer-content {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  min-height: 0;

  // Custom scrollbar
  &::-webkit-scrollbar {
    width: 6px;
  }

  &::-webkit-scrollbar-track {
    background: var(--td-bg-color-secondarycontainer);
    border-radius: 3px;
  }

  &::-webkit-scrollbar-thumb {
    background: var(--td-bg-color-component-disabled);
    border-radius: 3px;
    transition: background 0.2s;

    &:hover {
      background: var(--td-brand-color);
    }
  }

  .editor-form {
    width: 100%;
  }
}

.faq-editor-drawer-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

// Full-width input wrapper - unified style
.full-width-input-wrapper {
  display: flex;
  gap: 8px;
  align-items: center;
  width: 100%;

  .full-width-input {
    flex: 1;
    min-width: 0;
  }

  .full-width-textarea {
    flex: 1;
    min-width: 0;

    :deep(.t-textarea__inner) {
      min-height: 80px;
    }
  }

  // textarea needs top alignment
  &.textarea-wrapper {
    align-items: flex-start;
  }

  .add-item-btn {
    flex-shrink: 0;
    width: 32px;
    height: 32px;
    min-width: 32px;
    padding: 0;
    font-family: var(--app-font-family);
    transition: all 0.2s ease;
    border-radius: 8px;
  }

  :deep(.add-item-btn) {
    background: var(--td-brand-color) !important;
    border: 1px solid var(--td-brand-color) !important;
    border-radius: 8px !important;
    color: var(--td-text-color-anti) !important;
    display: flex;
    align-items: center;
    justify-content: center;

    &:hover:not(:disabled) {
      background: var(--td-brand-color) !important;
      border-color: var(--td-brand-color-active) !important;
      transform: scale(1.05);
      box-shadow: 0 2px 8px rgba(7, 192, 95, 0.3);
    }

    &:active:not(:disabled) {
      background: var(--td-brand-color-active) !important;
      border-color: var(--td-brand-color-active) !important;
      transform: scale(0.98);
    }

    &:disabled {
      background: var(--td-bg-color-component-disabled) !important;
      border-color: var(--td-component-stroke) !important;
      color: var(--td-text-color-placeholder) !important;
      cursor: not-allowed;
      opacity: 0.6;
    }

    .t-icon {
      font-size: 16px;
    }
  }
}

.textarea-container {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 100%;
}

.item-count {
  font-size: 13px;
  color: var(--td-text-color-secondary);
  font-family: var(--app-font-family);
  font-weight: 500;
  text-align: right;
  padding-right: 40px;
  line-height: 1;
}

.item-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 100%;
  margin-top: 8px;
}


.item-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 14px;
  background: var(--td-bg-color-container);
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  transition: all 0.2s ease;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.04);
  position: relative;

  &.answer-row {
    align-items: flex-start;
    padding: 12px 14px;
  }

  &:hover {
    background: var(--td-bg-color-secondarycontainer);
    border-color: var(--td-brand-color);
    box-shadow: 0 2px 8px rgba(7, 192, 95, 0.12);
    transform: translateY(-1px);
  }

  &.negative {
    background: var(--td-warning-color-light);
    border-color: var(--td-warning-color-focus);

    &:hover {
      background: var(--td-warning-color-light);
      border-color: var(--td-warning-color);
      box-shadow: 0 2px 8px rgba(251, 191, 36, 0.15);
    }
  }

  .item-content {
    flex: 1;
    font-size: 14px;
    line-height: 1.6;
    color: var(--td-text-color-primary);
    font-family: var(--app-font-family);
    white-space: pre-wrap;
    word-break: break-word;
    padding: 0;
    font-weight: 400;
  }

  .remove-item-btn {
    flex-shrink: 0;
    color: var(--td-text-color-placeholder);
    padding: 0;
    width: 24px;
    height: 24px;
    min-width: 24px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 6px;
    transition: all 0.2s ease;
    background: transparent;
    border: none;
    cursor: pointer;

    &:hover {
      color: var(--td-error-color);
      background: var(--td-error-color-light);
    }

    &:active {
      background: var(--td-error-color-light);
    }

    :deep(.t-icon) {
      font-size: 14px;
    }
  }

  &.answer-row .remove-item-btn {
    margin-top: 0;
  }
}

.form-tip {
  margin-top: 6px;
  font-size: 12px;
  color: var(--td-text-color-disabled);
  font-family: var(--app-font-family);
}

// FAQ editor form style - fully follows the settings page
.faq-editor-form {
  width: 100%;

  // Hide Form's default structure
  :deep(.t-form__label) {
    display: none !important;
    width: 0 !important;
    padding: 0 !important;
    margin: 0 !important;
  }

  :deep(.t-form__controls) {
    margin-left: 0 !important;
    width: 100% !important;
  }

  :deep(.t-form__controls-content) {
    margin: 0 !important;
    padding: 0 !important;
    width: 100% !important;
    display: block !important;
  }

  :deep(.t-form-item) {
    margin-bottom: 0 !important;
    padding: 0 !important;
  }
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

  &.vertical {
    flex-direction: column;
    gap: 12px;

    .setting-control {
      width: 100%;
      max-width: 100%;
    }
  }

  // Emphasis style for primary fields (standard question, answer)
  &.setting-row-primary {
    padding: 20px 0;
    padding-left: 12px;
    position: relative;

    // Remove top spacing from the first one (standard question)
    &:first-child {
      padding-top: 0;
    }

    // Left color marker (both standard question and answer use green)
    &::before {
      content: '';
      position: absolute;
      left: 0;
      top: 20px;
      width: 3px;
      height: calc(100% - 40px);
      background: var(--td-brand-color);
      border-radius: 0 2px 2px 0;
    }

    &:first-child::before {
      top: 0;
      height: calc(100% - 20px);
    }
  }

  // Secondary style for optional fields (similar questions, counter-examples)
  &.setting-row-optional {
    padding-left: 12px;
    position: relative;

    // Left color marker
    &::before {
      content: '';
      position: absolute;
      left: 0;
      top: 20px;
      width: 3px;
      height: calc(100% - 40px);
      border-radius: 0 2px 2px 0;
    }

    .setting-info {
      .optional-label {
        color: var(--td-text-color-primary);
        font-weight: 500;
      }

      .optional-desc {
        color: var(--td-text-color-secondary);
      }
    }
  }

  // Blue marker for similar questions
  &.setting-row-similar::before {
    background: var(--td-brand-color);
  }

  // Orange marker for counter-examples
  &.setting-row-negative::before {
    background: var(--td-warning-color);
  }

  // Remove bottom border from answer
  &.setting-row-answer {
    border-bottom: none;
  }
}

.setting-info {
  flex: 1;
  max-width: 65%;
  padding-right: 24px;

  label {
    font-size: 15px;
    font-weight: 500;
    color: var(--td-text-color-primary);
    display: block;
    margin-bottom: 4px;
  }

  .required-label {
    font-size: 15px;
    font-weight: 600;
    color: var(--td-text-color-primary);
    display: inline-flex;
    align-items: center;
    gap: 4px;
    margin-bottom: 4px;
  }

  .required-mark {
    color: var(--td-error-color);
    font-weight: 600;
    font-size: 14px;
  }

  .optional-label {
    font-size: 15px;
    font-weight: 600;
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

  .optional-desc {
    font-size: 13px;
    color: var(--td-text-color-secondary);
  }
}

.setting-row.vertical .setting-info {
  max-width: 100%;
  padding-right: 0;
  width: 100%;
}

.setting-control {
  flex-shrink: 0;
  min-width: 280px;
  display: flex;
  justify-content: flex-end;
  align-items: center;
}

.setting-row.vertical .setting-control {
  width: 100%;
  max-width: 100%;
  min-width: unset;
  justify-content: flex-start;
  align-items: flex-start;
  flex-direction: column;
}

// Input in vertical layout should be full width
.setting-row.vertical .full-width-input {
  width: 100%;

  :deep(.t-input__wrap) {
    width: 100%;
  }
}

.setting-row.vertical .full-width-textarea {
  width: 100%;

  :deep(.t-textarea) {
    width: 100%;
  }
}

// Input component styles - consistent with the login page
:deep(.t-input) {
  font-family: var(--app-font-family);
  font-size: 14px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-container);
  transition: all 0.2s ease;

  &:hover {
    border-color: var(--td-brand-color);
  }

  &:focus-within {
    border-color: var(--td-brand-color);
    box-shadow: 0 0 0 3px rgba(7, 192, 95, 0.1);
  }

  .t-input__inner {
    border: none !important;
    box-shadow: none !important;
    outline: none !important;
    background: transparent;
    font-size: 14px;
    font-family: var(--app-font-family);
    padding: 6px 12px;
    color: var(--td-text-color-primary);

    &:focus {
      border: none !important;
      box-shadow: none !important;
      outline: none !important;
    }

    &::placeholder {
      color: var(--td-text-color-placeholder);
    }
  }

  .t-input__wrap {
    border: none !important;
    box-shadow: none !important;
  }
}

// Textarea component styles
:deep(.t-textarea) {
  font-family: var(--app-font-family);
  font-size: 14px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-container);
  transition: all 0.2s ease;

  &:hover {
    border-color: var(--td-brand-color);
  }

  &:focus-within {
    border-color: var(--td-brand-color);
    box-shadow: 0 0 0 3px rgba(7, 192, 95, 0.1);
  }

  .t-textarea__inner {
    border: none !important;
    box-shadow: none !important;
    outline: none !important;
    background: transparent;
    font-size: 14px;
    font-family: var(--app-font-family);
    line-height: 1.6;
    resize: vertical;
    padding: 6px 12px;
    color: var(--td-text-color-primary);

    &:focus {
      border: none !important;
      box-shadow: none !important;
      outline: none !important;
    }

    &::placeholder {
      color: var(--td-text-color-placeholder);
    }
  }
}

// Import dialog animation
.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.2s ease;
}

.modal-enter-active .faq-import-modal,
.modal-leave-active .faq-import-modal,
.modal-enter-active .batch-tag-modal,
.modal-leave-active .batch-tag-modal {
  transition: transform 0.2s ease, opacity 0.2s ease;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-enter-from .faq-import-modal,
.modal-leave-to .faq-import-modal,
.modal-enter-from .batch-tag-modal,
.modal-leave-to .batch-tag-modal {
  transform: scale(0.95);
  opacity: 0;
}

// Tag style improvements
.answer-tag {
  background: var(--td-brand-color)1a;
  color: var(--td-brand-color);
  border-color: var(--td-brand-color)33;
}

.question-tag {
  background: var(--td-bg-color-container);
  border-color: var(--td-component-stroke);
  color: var(--td-text-color-placeholder);
}

// Search test drawer styles - consistent with the editor drawer style
:deep(.faq-search-drawer) {
  .t-drawer__body {
    padding: 20px;
    display: flex;
    flex-direction: column;
    height: 100%;
  }

  .t-drawer__header {
    padding: 20px 24px;
    border-bottom: 1px solid var(--td-component-stroke);
    font-family: var(--app-font-family);
    font-size: 18px;
    font-weight: 600;
    color: var(--td-text-color-primary);
  }
}

.search-test-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding-right: 0;

  // Hide scrollbar but keep scroll functionality
  scrollbar-width: none; // Firefox
  -ms-overflow-style: none; // IE and Edge

  &::-webkit-scrollbar {
    display: none; // Chrome, Safari, Opera
  }
}

.search-form {
  flex-shrink: 0;

  :deep(.t-form__label) {
    display: none !important;
    width: 0 !important;
    padding: 0 !important;
    margin: 0 !important;
  }

  :deep(.t-form__controls) {
    margin-left: 0 !important;
    width: 100% !important;
  }

  :deep(.t-form__controls-content) {
    margin: 0 !important;
    padding: 0 !important;
    width: 100% !important;
    display: block !important;
  }

  :deep(.t-form-item) {
    margin-bottom: 0 !important;
    padding: 0 !important;
  }
}

.slider-wrapper {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  padding: 2px 0;
}

.search-form .setting-row {
  padding: 16px 0;
  border-bottom: 1px solid var(--td-component-stroke);

  &.search-first-row {
    padding-top: 0;
  }

  &:last-child {
    border-bottom: none;
    padding-bottom: 0;
  }

  .setting-info {
    max-width: 100%;
    padding-right: 0;
    margin-bottom: 8px;

    label {
      font-size: 14px;
      font-weight: 500;
      color: var(--td-text-color-primary);
      display: block;
      margin-bottom: 4px;
    }

    .desc {
      font-size: 12px;
      color: var(--td-text-color-secondary);
      margin: 0;
      line-height: 1.4;
    }
  }

  .setting-control {
    width: 100%;
    max-width: 100%;
    min-width: unset;
    justify-content: flex-start;
    align-items: flex-start;
    flex-direction: column;
  }
}

:deep(.slider-wrapper .t-slider) {
  flex: 1;
  min-width: 0;
}

.slider-value {
  flex-shrink: 0;
  min-width: 50px;
  text-align: right;
  font-family: var(--app-font-family);
  font-size: 14px;
  font-weight: 500;
  color: var(--td-text-color-primary);
  padding: 4px 8px;
  background: var(--td-bg-color-container);
  border-radius: 6px;
}

.search-button {
  height: 36px;
  border-radius: 8px;
  font-family: var(--app-font-family);
  font-size: 14px;
  font-weight: 500;
  transition: all 0.2s ease;

  &:hover:not(:disabled) {
    transform: translateY(-1px);
    box-shadow: 0 4px 12px rgba(7, 192, 95, 0.3);
  }

  &:active:not(:disabled) {
    transform: translateY(0);
  }
}

.search-results {
  display: flex;
  flex-direction: column;
  padding-top: 20px;
  padding-left: 0;
  width: 100%;
  box-sizing: border-box;
}

.results-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;
  margin-left: 0;
  margin-right: 0;
  padding-left: 0;
  font-family: var(--app-font-family);
  font-size: 14px;
  font-weight: 600;
  color: var(--td-text-color-primary);
  flex-shrink: 0;
  justify-content: flex-start;

  .t-icon {
    color: var(--td-brand-color);
  }
}

.no-results {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 48px 16px;
  color: var(--td-text-color-secondary);
  font-family: var(--app-font-family);
  font-size: 14px;
  text-align: center;
  background: var(--td-bg-color-container);
  border-radius: 8px;
  border: 1px dashed var(--td-component-stroke);
}

.results-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.result-card {
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-container);
  padding: 14px;
  transition: border-color 0.2s ease, box-shadow 0.2s ease;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.04);
  width: 100%;
  box-sizing: border-box;
  min-width: 0;
  overflow: visible;
  position: relative;

  &:hover {
    border-color: var(--td-brand-color);
    box-shadow: 0 2px 8px rgba(7, 192, 95, 0.12);
  }
}

.result-header {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 0;
  border-bottom: none;
  cursor: pointer;
  user-select: none;
  padding: 4px;
  margin: -4px;
  border-radius: 6px;
  position: relative;

  &:hover {
    background-color: var(--td-bg-color-container);
  }
}

.result-card.expanded .result-header {
  margin-bottom: 12px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--td-component-stroke);
  margin-left: -4px;
  margin-right: -4px;
  padding-left: 4px;
  padding-right: 4px;
}

.result-question-wrapper {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  width: 100%;
}

.result-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.result-question {
  font-family: var(--app-font-family);
  font-size: 14px;
  font-weight: 600;
  color: var(--td-text-color-primary);
  line-height: 1.6;
  word-break: break-word;
  display: flex;
  align-items: flex-start;
  gap: 6px;

  .result-index {
    flex-shrink: 0;
    color: var(--td-brand-color);
    font-weight: 600;
  }
}

.matched-question {
  display: flex;
  align-items: flex-start;
  gap: 4px;
  padding-left: 20px;
  font-size: 12px;
  line-height: 1.5;

  .matched-label {
    flex-shrink: 0;
    color: var(--td-warning-color);
    font-weight: 500;
  }

  .matched-text {
    color: var(--td-warning-color-active);
    background: linear-gradient(90deg, rgba(251, 191, 36, 0.15) 0%, rgba(251, 191, 36, 0.05) 100%);
    padding: 1px 6px;
    border-radius: 4px;
    word-break: break-word;
  }
}

.result-meta {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  flex-shrink: 0;
  margin-left: auto;
}

.expand-icon {
  flex-shrink: 0;
  font-size: 18px;
  color: var(--td-text-color-secondary);
  transition: transform 0.2s ease;
  cursor: pointer;

  &:hover {
    color: var(--td-brand-color);
  }
}

.score-tag,
.match-type-tag {
  font-size: 12px;
  padding: 4px 8px;
  border-radius: 6px;
  font-family: var(--app-font-family);
}

.result-body {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding-top: 12px;
  margin-top: 0;
  border-top: 1px solid var(--td-component-stroke);
  position: relative;
  width: 100%;
}

// Slide down animation - performance optimization
.slide-down-enter-active {
  transition: opacity 0.2s cubic-bezier(0.4, 0, 0.2, 1),
    transform 0.2s cubic-bezier(0.4, 0, 0.2, 1);
  overflow: hidden;
  will-change: opacity, transform;
}

.slide-down-leave-active {
  transition: opacity 0.2s cubic-bezier(0.4, 0, 0.2, 1),
    transform 0.2s cubic-bezier(0.4, 0, 0.2, 1);
  overflow: hidden;
  will-change: opacity, transform;
}

.slide-down-enter-from {
  opacity: 0;
  transform: translateY(-8px);
}

.slide-down-enter-to {
  opacity: 1;
  transform: translateY(0);
}

.slide-down-leave-from {
  opacity: 1;
  transform: translateY(0);
}

.slide-down-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}

.result-section {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

// Batch tag dialog styles - consistent with the import dialog style
.batch-tag-overlay {
  position: fixed;
  inset: 0;
  z-index: 1000;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
  backdrop-filter: blur(4px);
}

.batch-tag-modal {
  position: relative;
  width: 100%;
  max-width: 480px;
  background: var(--td-bg-color-container);
  border-radius: 12px;
  box-shadow: 0 6px 28px rgba(15, 23, 42, 0.08);
  overflow: hidden;
  display: flex;
  flex-direction: column;

  .batch-tag-close-btn {
    position: absolute;
    top: 20px;
    right: 20px;
    width: 32px;
    height: 32px;
    border: none;
    background: var(--td-bg-color-secondarycontainer);
    border-radius: 6px;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--td-text-color-secondary);
    transition: all 0.2s ease;
    z-index: 10;

    &:hover {
      background: var(--td-bg-color-secondarycontainer);
      color: var(--td-text-color-primary);
    }
  }
}

.batch-tag-container {
  display: flex;
  flex-direction: column;
  padding: 24px;
}

.batch-tag-header {
  margin-bottom: 24px;
  padding-right: 40px;

  .batch-tag-title {
    margin: 0;
    font-size: 20px;
    font-weight: 600;
    color: var(--td-text-color-primary);
    line-height: 1.4;
  }
}

.batch-tag-content {
  flex: 1;
  min-height: 0;
}

.batch-tag-tip {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 12px 16px;
  margin-bottom: 20px;
  background: var(--td-brand-color-light);
  border: 1px solid var(--td-brand-color-focus);
  border-radius: 8px;
  font-size: 14px;
  color: var(--td-brand-color);
  line-height: 1.5;

  .tip-icon {
    flex-shrink: 0;
    margin-top: 2px;
    color: var(--td-brand-color);
  }
}

.batch-tag-form {
  margin-top: 0;

  :deep(.t-form-item) {
    margin-bottom: 0;
  }

  :deep(.t-form-item__label) {
    font-size: 14px;
    font-weight: 500;
    color: var(--td-text-color-primary);
    margin-bottom: 8px;
  }
}

.batch-tag-select {
  width: 100%;
}

.batch-tag-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 24px;
  padding-top: 20px;
  border-top: 1px solid var(--td-component-stroke);
}

.tag-select-empty {
  padding: 8px 12px;
  text-align: center;
  color: var(--td-text-color-secondary);
  font-size: 14px;
}

.section-label {
  font-family: var(--app-font-family);
  font-size: 12px;
  font-weight: 600;
  color: var(--td-text-color-secondary);
  margin-bottom: 4px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.result-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  width: 100%;
  min-width: 0;
}

:deep(.result-tags .t-tag) {
  max-width: 100%;
  min-width: 0;
  word-break: break-word;
  overflow-wrap: break-word;
}

:deep(.result-tags .t-tag__text) {
  display: inline-block;
  max-width: 100%;
  word-break: break-word;
  overflow-wrap: break-word;
  white-space: normal;
  line-height: 1.4;
}
</style>
