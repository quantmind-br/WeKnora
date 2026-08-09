<template>
  <div class="kb-list-container">
    <ListSpaceSidebar v-if="!authStore.isLiteMode" v-model="spaceSelection" :count-all="allKnowledgeBases"
      :count-mine="kbs.length" :count-by-org="effectiveSharedCountByOrg" :count-favorites="kbFavoritesCount"
      :count-recents="kbRecentsCount" />
    <div class="kb-list-content">
      <div class="header" style="--wails-draggable: drag">
        <div class="header-title" style="--wails-draggable: drag">
          <div class="title-row" style="--wails-draggable: drag">
            <h2 style="--wails-draggable: drag">{{ $t('knowledgeBase.title') }}</h2>
            <t-tooltip v-if="authStore.hasRole('contributor')" :content="$t('knowledgeList.create')" placement="bottom">
              <t-button variant="text" theme="default" size="small" class="header-action-btn"
                data-guide="kb-list-create" style="--wails-draggable: no-drag" @click="handleCreateKnowledgeBase">
                <template #icon><t-icon name="folder-add" size="16px" /></template>
              </t-button>
            </t-tooltip>
          </div>
          <p class="header-subtitle" style="--wails-draggable: drag">{{ $t('knowledgeList.subtitle') }}</p>
        </div>
      </div>
      <div class="kb-list-main">
        <!-- creator filter intentionally removed from chrome: every card
             already shows its creator via ResourceOriginBadge / avatar, so
             a dedicated horizontal switch added more noise than signal.
             The backend `?creator=mine|others` param and the URL-state
             field are kept so a future "filter by member" entry point
             (e.g. clicking an avatar) can deep-link without re-plumbing. -->


        <!-- Knowledge base not initialized notice -->
        <div v-if="hasUninitializedKbs" class="warning-banner">
          <t-icon name="info-circle" size="16px" />
          <span>{{ $t('knowledgeList.uninitializedBanner') }}</span>
        </div>

        <!-- Upload progress notice -->
        <div v-if="uploadSummaries.length" class="upload-progress-panel">
          <div v-for="summary in uploadSummaries" :key="summary.kbId" class="upload-progress-item">
            <div class="upload-progress-icon">
              <t-icon :name="summary.completed === summary.total ? 'check-circle-filled' : 'upload'" size="20px" />
            </div>
            <div class="upload-progress-content">
              <div class="progress-title">
                {{
                  summary.completed === summary.total
                    ? $t('knowledgeList.uploadProgress.completedTitle', { name: summary.kbName })
                    : $t('knowledgeList.uploadProgress.uploadingTitle', { name: summary.kbName })
                }}
              </div>
              <div class="progress-subtitle">
                {{
                  summary.completed === summary.total
                    ? $t('knowledgeList.uploadProgress.completedDetail', { total: summary.total })
                    : $t('knowledgeList.uploadProgress.detail', { completed: summary.completed, total: summary.total })
                }}
              </div>
              <div class="progress-subtitle secondary">
                {{
                  summary.completed === summary.total
                    ? $t('knowledgeList.uploadProgress.refreshing')
                    : $t('knowledgeList.uploadProgress.keepPageOpen')
                }}
              </div>
              <div v-if="summary.hasError" class="progress-subtitle error">
                {{ $t('knowledgeList.uploadProgress.errorTip') }}
              </div>
              <div class="progress-bar">
                <div class="progress-bar-inner" :style="{ width: summary.progress + '%' }"></div>
              </div>
            </div>
          </div>
        </div>

        <!-- Skeleton screen placeholder -->
        <div v-if="loading && kbs.length === 0" class="kb-card-wrap">
          <div v-for="n in 6" :key="'skel-' + n" class="kb-card kb-card-skeleton">
            <div class="card-header">
              <t-skeleton animation="gradient" :row-col="[{ width: '60%', height: '20px' }]" />
            </div>
            <div class="card-content">
              <t-skeleton animation="gradient"
                :row-col="[{ width: '100%', height: '14px' }, { width: '80%', height: '14px' }]" />
            </div>
            <div class="card-bottom">
              <t-skeleton animation="gradient"
                :row-col="[[{ width: '28px', height: '28px', type: 'rect' }, { width: '28px', height: '28px', type: 'rect' }]]" />
            </div>
          </div>
        </div>

        <!-- Card grid: All / Favorites / Recent — share the same card template,
             can switch views by relying only on the filteredKnowledgeBases slice -->
        <div
          v-if="(spaceSelection === 'all' || spaceSelection === 'favorites' || spaceSelection === 'recents') && filteredKnowledgeBases.length > 0"
          class="kb-card-wrap">
          <!-- Pinned group title -->
          <div
            v-if="filteredKnowledgeBases[0] && filteredKnowledgeBases[0].isMine && filteredKnowledgeBases[0].is_pinned"
            class="kb-section-header kb-section-header-pinned" role="button" tabindex="0"
            @click="toggleKbSection('pinned')"
            @keydown.enter.prevent="toggleKbSection('pinned')"
            @keydown.space.prevent="toggleKbSection('pinned')">
            <t-icon name="pin-filled" size="14px" />
            <span>{{ $t('knowledgeList.sections.pinned') }}</span>
            <span class="kb-section-count">{{ filteredKbSectionCounts.pinned }}</span>
            <t-icon class="kb-section-toggle" :name="isKbSectionCollapsed('pinned') ? 'chevron-right' : 'chevron-down'"
              size="14px" />
          </div>
          <!-- All: my knowledge bases + knowledge bases shared with me.
               The "Pinned" group is now handled by the top header. The remaining sections (Created by me / This space ·
               View-only / Shared with me) each render their own title; the former "Other" transition title
               is meaningless under per-user pinned models now, removed to avoid overlapping with specific subsection titles. -->
          <template v-for="(kb, index) in filteredKnowledgeBases" :key="kb.id">
            <!-- Created by me: show a title before the first non-pinned "Created by me" card, displayed consistently
                 regardless of whether a "Pinned" section exists above. Same as "This space · View-only"
                 Only appears in the contributor view — the admin/owner view never had it to begin with
                 Any section header standing alone would feel unbalanced. -->
            <div v-if="showShareGroupHeaders
              && kb.isMine
              && isMyKb(kb as KB)
              && !kb.is_pinned
              && (index === 0
                || (filteredKnowledgeBases[index - 1] as any).is_pinned)" class="kb-section-header" role="button"
              tabindex="0" @click="toggleKbSection('mine')"
              @keydown.enter.prevent="toggleKbSection('mine')"
              @keydown.space.prevent="toggleKbSection('mine')">
              <t-icon name="user" size="14px" />
              <span>{{ $t('knowledgeList.sections.mine') }}</span>
              <span class="kb-section-count">{{ filteredKbSectionCounts.mine }}</span>
              <t-icon class="kb-section-toggle" :name="isKbSectionCollapsed('mine') ? 'chevron-right' : 'chevron-down'"
                size="14px" />
            </div>
            <!-- This space · view only: created by a colleague in this space, not editable by the current contributor.
                 The current card must be non-pinned (otherwise it belongs under "Pinned"), and the previous card must either
                 not exist, or be "shared with me", or be created by me, or be a pinned card
                 (the pinned → non-pinned transition also needs this header). -->
            <div v-if="showShareGroupHeaders
              && kb.isMine
              && !isMyKb(kb as KB)
              && !kb.is_pinned
              && (index === 0
                || !filteredKnowledgeBases[index - 1].isMine
                || isMyKb(filteredKnowledgeBases[index - 1] as KB)
                || (filteredKnowledgeBases[index - 1] as any).is_pinned)" class="kb-section-header" role="button"
              tabindex="0" @click="toggleKbSection('tenantOthers')"
              @keydown.enter.prevent="toggleKbSection('tenantOthers')"
              @keydown.space.prevent="toggleKbSection('tenantOthers')">
              <t-icon :name="tenantSectionIconName" size="14px" />
              <span>{{ $t(tenantSectionLabelKey) }}</span>
              <span class="kb-section-count">{{ filteredKbSectionCounts.tenantOthers }}</span>
              <t-icon class="kb-section-toggle"
                :name="isKbSectionCollapsed('tenantOthers') ? 'chevron-right' : 'chevron-down'" size="14px" />
            </div>
            <!-- Shared with me · editable: first transition from "Mine (incl. colleagues')" to shared + editable -->
            <div v-if="showShareGroupHeaders
              && !kb.isMine
              && isSharedKbEditable((kb as any).permission)
              && (index === 0 || filteredKnowledgeBases[index - 1].isMine)" class="kb-section-header" role="button"
              tabindex="0" @click="toggleKbSection('sharedEditable')"
              @keydown.enter.prevent="toggleKbSection('sharedEditable')"
              @keydown.space.prevent="toggleKbSection('sharedEditable')">
              <t-icon name="usergroup-add" size="14px" />
              <t-icon name="edit-1" size="12px" class="kb-section-subicon" />
              <span>{{ $t('knowledgeList.sections.sharedEditable') }}</span>
              <span class="kb-section-count">{{ filteredKbSectionCounts.sharedEditable }}</span>
              <t-icon class="kb-section-toggle"
                :name="isKbSectionCollapsed('sharedEditable') ? 'chevron-right' : 'chevron-down'" size="14px" />
            </div>
            <!-- Shared with me · view only: transition from "editable shared / mine" to viewer-shared -->
            <div v-if="showShareGroupHeaders
              && !kb.isMine
              && !isSharedKbEditable((kb as any).permission)
              && (index === 0
                || filteredKnowledgeBases[index - 1].isMine
                || isSharedKbEditable((filteredKnowledgeBases[index - 1] as any).permission))"
              class="kb-section-header" role="button" tabindex="0" @click="toggleKbSection('sharedReadonly')"
              @keydown.enter.prevent="toggleKbSection('sharedReadonly')"
              @keydown.space.prevent="toggleKbSection('sharedReadonly')">
              <t-icon name="usergroup-add" size="14px" />
              <t-icon name="browse" size="12px" class="kb-section-subicon" />
              <span>{{ $t('knowledgeList.sections.sharedReadonly') }}</span>
              <span class="kb-section-count">{{ filteredKbSectionCounts.sharedReadonly }}</span>
              <t-icon class="kb-section-toggle"
                :name="isKbSectionCollapsed('sharedReadonly') ? 'chevron-right' : 'chevron-down'" size="14px" />
            </div>
            <!-- My knowledge base cards -->
            <div v-if="kb.isMine" v-show="!isKbSectionCollapsed(kbSectionOf(kb))" class="kb-card" :class="{
              'uninitialized': !isInitialized(kb),
              'kb-type-document': (kb.type || 'document') === 'document',
              'kb-type-faq': kb.type === 'faq',
              'highlight-flash': highlightedKbId !== null && highlightedKbId === kb.id
            }"
              :ref="el => { if (highlightedKbId !== null && highlightedKbId === kb.id && el) highlightedCardRef = el as HTMLElement }"
              @click="handleCardClick(kb)">
              <!-- Favorite button: floats top-right; via .card-header's padding-right
                   leave room for the "More" button, to avoid the two buttons overlapping. -->
              <button type="button" class="kb-favorite-star" :class="{ 'is-favorited': isKbFavorited(kb.id) }"
                @click.stop="toggleFavoriteKb(kb.id, $event)">
                <t-icon :name="isKbFavorited(kb.id) ? 'star-filled' : 'star'" size="14px" />
              </button>
              <!-- Card header -->
              <div class="card-header">
                <span class="card-title" :title="kb.name">
                  <KbWikiBadge v-if="isWikiKb(kb)" />
                  <span class="card-title-text">{{ kb.name }}</span>
                </span>
                <!-- The card menu always exists when the card is visible: pin
                     is now per-user and available to anyone who can see the KB
                     (backend route only requires KB read access). Settings /
                     Delete are mutations, so they stay behind canManageKBCard. -->
                <t-popup overlayClassName="card-more-popup" trigger="click" destroy-on-close
                  placement="bottom-right">
                  <div class="more-wrap" @click.stop>
                    <img class="more-icon" src="@/assets/img/more.png" alt="" />
                  </div>
                  <template #content>
                    <div class="popup-menu" @click.stop>
                      <div class="popup-menu-item" @click.stop="handleTogglePinById(kb.id)">
                        <t-icon class="menu-icon" :name="kb.is_pinned ? 'pin-filled' : 'pin'" />
                        <span>{{ kb.is_pinned ? $t('knowledgeList.pin.unpin') : $t('knowledgeList.pin.pin') }}</span>
                      </div>
                      <div v-if="canDuplicateKBCard(kb)" class="popup-menu-item"
                        @click.stop="handleDuplicateById(kb.id)">
                        <t-icon class="menu-icon" name="file-copy" />
                        <span>{{ $t('knowledgeList.menu.duplicate') }}</span>
                      </div>
                      <template v-if="canManageKBCard(kb)">
                        <div class="popup-menu-item" @click.stop="handleSettingsById(kb.id)">
                          <t-icon class="menu-icon" name="setting" />
                          <span>{{ $t('knowledgeBase.settings') }}</span>
                        </div>
                        <div class="popup-menu-item delete" @click.stop="handleDeleteById(kb.id)">
                          <t-icon class="menu-icon" name="delete" />
                          <span>{{ $t('common.delete') }}</span>
                        </div>
                      </template>
                    </div>
                  </template>
                </t-popup>
              </div>

              <!-- Card content -->
              <div class="card-content">
                <div class="card-description">
                  {{ kb.description || $t('knowledgeBase.noDescription') }}
                </div>
              </div>

              <!-- Card footer -->
              <div class="card-bottom">
                <div class="bottom-left">
                  <div class="feature-badges">
                    <t-tooltip
                      :content="kb.type === 'faq' ? $t('knowledgeEditor.basic.typeFAQ') : $t('knowledgeEditor.basic.typeDocument')"
                      placement="top">
                      <div class="feature-badge"
                        :class="{ 'type-document': (kb.type || 'document') === 'document', 'type-faq': kb.type === 'faq' }">
                        <t-icon :name="kb.type === 'faq' ? 'chat-bubble-help' : 'folder'" size="14px" />
                        <span class="badge-count">{{ kb.type === 'faq' ? (kb.chunk_count || 0) : (kb.knowledge_count ||
                          0) }}</span>
                        <t-icon v-if="kb.isProcessing" name="loading" size="12px" class="processing-icon" />
                      </div>
                    </t-tooltip>
                    <t-tooltip v-if="kb.extract_config?.enabled" :content="$t('knowledgeList.features.knowledgeGraph')"
                      placement="top">
                      <div class="feature-badge kg">
                        <t-icon name="relation" size="14px" />
                      </div>
                    </t-tooltip>
                    <t-tooltip v-if="kb.vlm_config?.enabled" :content="$t('knowledgeList.features.multimodal')"
                      placement="top">
                      <div class="feature-badge multimodal">
                        <t-icon name="image" size="14px" />
                      </div>
                    </t-tooltip>
                    <t-tooltip v-if="kb.question_generation_config?.enabled"
                      :content="$t('knowledgeList.features.questionGeneration')" placement="top">
                      <div class="feature-badge question">
                        <t-icon name="help-circle" size="14px" />
                      </div>
                    </t-tooltip>
                    <t-tooltip v-if="kb.share_count && kb.share_count > 0"
                      :content="$t('knowledgeList.sharedToOrgs', { count: kb.share_count })" placement="top">
                      <div class="feature-badge shared">
                        <t-icon name="share" size="14px" />
                      </div>
                    </t-tooltip>
                  </div>
                </div>
                <div v-if="!authStore.isLiteMode && showKbOriginBadge(kb)" class="bottom-right">
                  <ResourceOriginBadge :variant="kbOriginVariant(kb)" :creator-name="kb.creator_name" />
                </div>
              </div>
            </div>

            <!-- Shared knowledge base cards -->
            <div v-else v-show="!isKbSectionCollapsed(kbSectionOf(kb))" class="kb-card shared-kb-card" :class="{
              'kb-type-document': (kb.type || 'document') === 'document',
              'kb-type-faq': kb.type === 'faq'
            }" @click="handleSharedKbClickFromAll(kb)">
              <button type="button" class="kb-favorite-star" :class="{ 'is-favorited': isKbFavorited(kb.id) }"
                @click.stop="toggleFavoriteKb(kb.id, $event)">
                <t-icon :name="isKbFavorited(kb.id) ? 'star-filled' : 'star'" size="14px" />
              </button>
              <!-- Card header -->
              <div class="card-header">
                <span class="card-title" :title="kb.name">
                  <KbWikiBadge v-if="isWikiKb(kb)" />
                  <span class="card-title-text">{{ kb.name }}</span>
                </span>
                <t-tooltip :content="$t('knowledgeList.menu.viewDetails')" placement="top">
                  <button type="button" class="shared-detail-trigger" @click.stop="openSharedDetailFromAll(kb)"
                    :aria-label="$t('knowledgeList.menu.viewDetails')">
                    <t-icon name="info-circle" size="16px" />
                  </button>
                </t-tooltip>
              </div>

              <!-- Card content -->
              <div class="card-content">
                <div class="card-description">
                  {{ kb.description || $t('knowledgeBase.noDescription') }}
                </div>
              </div>

              <!-- Card footer -->
              <div class="card-bottom">
                <div class="bottom-left">
                  <div class="feature-badges">
                    <t-tooltip
                      :content="kb.type === 'faq' ? $t('knowledgeEditor.basic.typeFAQ') : $t('knowledgeEditor.basic.typeDocument')"
                      placement="top">
                      <div class="feature-badge"
                        :class="{ 'type-document': (kb.type || 'document') === 'document', 'type-faq': kb.type === 'faq' }">
                        <t-icon :name="kb.type === 'faq' ? 'chat-bubble-help' : 'folder'" size="14px" />
                        <span class="badge-count">{{ kb.type === 'faq' ? (kb.chunk_count || '-') : (kb.knowledge_count
                          || '-')
                        }}</span>
                      </div>
                    </t-tooltip>
                    <t-tooltip v-if="kb.extract_config?.enabled" :content="$t('knowledgeList.features.knowledgeGraph')"
                      placement="top">
                      <div class="feature-badge kg">
                        <t-icon name="relation" size="14px" />
                      </div>
                    </t-tooltip>
                    <t-tooltip
                      v-if="kb.vlm_config?.enabled || (kb.storage_provider_config?.provider && kb.storage_provider_config.provider !== 'local')"
                      :content="$t('knowledgeList.features.multimodal')" placement="top">
                      <div class="feature-badge multimodal">
                        <t-icon name="image" size="14px" />
                      </div>
                    </t-tooltip>
                    <t-tooltip v-if="kb.question_generation_config?.enabled"
                      :content="$t('knowledgeList.features.questionGeneration')" placement="top">
                      <div class="feature-badge question">
                        <t-icon name="help-circle" size="14px" />
                      </div>
                    </t-tooltip>
                  </div>
                </div>
                <div class="bottom-right">
                  <t-tooltip :content="kb.org_name" placement="top">
                    <div class="org-source">
                      <img src="@/assets/img/organization-green.svg" class="org-source-icon" alt=""
                        aria-hidden="true" />
                      <span>{{ kb.org_name }}</span>
                    </div>
                  </t-tooltip>
                </div>
              </div>
            </div>
          </template>
        </div>

        <div v-if="spaceSelection === 'mine' && sortedMineKbs.length > 0" class="kb-card-wrap">
          <!-- Pinned group header -->
          <div v-if="sortedMineKbs[0] && sortedMineKbs[0].is_pinned" class="kb-section-header kb-section-header-pinned"
            role="button" tabindex="0" @click="toggleKbSection('pinned')"
            @keydown.enter.prevent="toggleKbSection('pinned')"
            @keydown.space.prevent="toggleKbSection('pinned')">
            <t-icon name="pin-filled" size="14px" />
            <span>{{ $t('knowledgeList.sections.pinned') }}</span>
            <span class="kb-section-count">{{ mineKbSectionCounts.pinned }}</span>
            <t-icon class="kb-section-toggle" :name="isKbSectionCollapsed('pinned') ? 'chevron-right' : 'chevron-down'"
              size="14px" />
          </div>
          <!-- My knowledge bases. "Pinned" is handled by the top header; every other section shows its own
               header — see the comment in the "All" tab for the same case. -->
          <template v-for="(kb, index) in sortedMineKbs" :key="kb.id">
            <!-- Created by me: show the header before the first non-pinned card I created, whether or not
                 a "Pinned" section is present above — align with "This space · view only" — see
                 the comment in the "All" tab for the same case. -->
            <div v-if="showShareGroupHeaders
              && isMyKb(kb)
              && !kb.is_pinned
              && (index === 0 || sortedMineKbs[index - 1].is_pinned)" class="kb-section-header" role="button"
              tabindex="0" @click="toggleKbSection('mine')"
              @keydown.enter.prevent="toggleKbSection('mine')"
              @keydown.space.prevent="toggleKbSection('mine')">
              <t-icon name="user" size="14px" />
              <span>{{ $t('knowledgeList.sections.mine') }}</span>
              <span class="kb-section-count">{{ mineKbSectionCounts.mine }}</span>
              <t-icon class="kb-section-toggle" :name="isKbSectionCollapsed('mine') ? 'chevron-right' : 'chevron-down'"
                size="14px" />
            </div>
            <!-- This space · view only: the current non-pinned colleague KB, and the previous card must either not exist,
                 or be created by me, or be a pinned card (pinned → non-pinned transition). -->
            <div v-if="showShareGroupHeaders
              && !isMyKb(kb)
              && !kb.is_pinned
              && (index === 0
                || isMyKb(sortedMineKbs[index - 1])
                || sortedMineKbs[index - 1].is_pinned)" class="kb-section-header" role="button" tabindex="0"
              @click="toggleKbSection('tenantOthers')"
              @keydown.enter.prevent="toggleKbSection('tenantOthers')"
              @keydown.space.prevent="toggleKbSection('tenantOthers')">
              <t-icon :name="tenantSectionIconName" size="14px" />
              <span>{{ $t(tenantSectionLabelKey) }}</span>
              <span class="kb-section-count">{{ mineKbSectionCounts.tenantOthers }}</span>
              <t-icon class="kb-section-toggle"
                :name="isKbSectionCollapsed('tenantOthers') ? 'chevron-right' : 'chevron-down'" size="14px" />
            </div>
            <div v-show="!isKbSectionCollapsed(kbSectionOf(kb))" class="kb-card" :class="{
              'uninitialized': !isInitialized(kb),
              'kb-type-document': (kb.type || 'document') === 'document',
              'kb-type-faq': kb.type === 'faq',
              'highlight-flash': highlightedKbId !== null && highlightedKbId === kb.id
            }"
              :ref="el => { if (highlightedKbId !== null && highlightedKbId === kb.id && el) highlightedCardRef = el as HTMLElement }"
              @click="handleCardClick(kb)">
              <button type="button" class="kb-favorite-star" :class="{ 'is-favorited': isKbFavorited(kb.id) }"
                @click.stop="toggleFavoriteKb(kb.id, $event)">
                <t-icon :name="isKbFavorited(kb.id) ? 'star-filled' : 'star'" size="14px" />
              </button>
              <!-- Card header -->
              <div class="card-header">
                <span class="card-title" :title="kb.name">
                  <KbWikiBadge v-if="isWikiKb(kb)" />
                  <span class="card-title-text">{{ kb.name }}</span>
                </span>
                <!-- See the matching block in the "all" tab template for why
                     this is no longer gated by canManageKBCard. -->
                <t-popup v-model="kb.showMore" overlayClassName="card-more-popup"
                  :on-visible-change="onVisibleChange" trigger="click" destroy-on-close placement="bottom-right">
                  <div variant="outline" class="more-wrap" @click.stop="openMore(index)"
                    :class="{ 'active-more': currentMoreIndex === index }">
                    <img class="more-icon" src="@/assets/img/more.png" alt="" />
                  </div>
                  <template #content>
                    <div class="popup-menu" @click.stop>
                      <div class="popup-menu-item" @click.stop="handleTogglePin(kb)">
                        <t-icon class="menu-icon" :name="kb.is_pinned ? 'pin-filled' : 'pin'" />
                        <span>{{ kb.is_pinned ? $t('knowledgeList.pin.unpin') : $t('knowledgeList.pin.pin') }}</span>
                      </div>
                      <div v-if="canDuplicateKBCard(kb)" class="popup-menu-item" @click.stop="handleDuplicate(kb)">
                        <t-icon class="menu-icon" name="file-copy" />
                        <span>{{ $t('knowledgeList.menu.duplicate') }}</span>
                      </div>
                      <template v-if="canManageKBCard(kb)">
                        <div class="popup-menu-item" @click.stop="handleSettings(kb)">
                          <t-icon class="menu-icon" name="setting" />
                          <span>{{ $t('knowledgeBase.settings') }}</span>
                        </div>
                        <div class="popup-menu-item delete" @click.stop="handleDelete(kb)">
                          <t-icon class="menu-icon" name="delete" />
                          <span>{{ $t('common.delete') }}</span>
                        </div>
                      </template>
                    </div>
                  </template>
                </t-popup>
              </div>

              <!-- Card content -->
              <div class="card-content">
                <div class="card-description">
                  {{ kb.description || $t('knowledgeBase.noDescription') }}
                </div>
              </div>

              <!-- Card footer -->
              <div class="card-bottom">
                <div class="bottom-left">
                  <div class="feature-badges">
                    <t-tooltip
                      :content="kb.type === 'faq' ? $t('knowledgeEditor.basic.typeFAQ') : $t('knowledgeEditor.basic.typeDocument')"
                      placement="top">
                      <div class="feature-badge"
                        :class="{ 'type-document': (kb.type || 'document') === 'document', 'type-faq': kb.type === 'faq' }">
                        <t-icon :name="kb.type === 'faq' ? 'chat-bubble-help' : 'folder'" size="14px" />
                        <span class="badge-count">{{ kb.type === 'faq' ? (kb.chunk_count || 0) : (kb.knowledge_count ||
                          0) }}</span>
                        <t-icon v-if="kb.isProcessing" name="loading" size="12px" class="processing-icon" />
                      </div>
                    </t-tooltip>
                    <t-tooltip v-if="kb.extract_config?.enabled" :content="$t('knowledgeList.features.knowledgeGraph')"
                      placement="top">
                      <div class="feature-badge kg">
                        <t-icon name="relation" size="14px" />
                      </div>
                    </t-tooltip>
                    <t-tooltip
                      v-if="kb.vlm_config?.enabled || (kb.storage_provider_config?.provider && kb.storage_provider_config.provider !== 'local')"
                      :content="$t('knowledgeList.features.multimodal')" placement="top">
                      <div class="feature-badge multimodal">
                        <t-icon name="image" size="14px" />
                      </div>
                    </t-tooltip>
                    <t-tooltip v-if="kb.question_generation_config?.enabled"
                      :content="$t('knowledgeList.features.questionGeneration')" placement="top">
                      <div class="feature-badge question">
                        <t-icon name="help-circle" size="14px" />
                      </div>
                    </t-tooltip>
                    <!-- Sharing status icon -->
                    <t-tooltip v-if="(kb.share_count ?? 0) > 0"
                      :content="$t('knowledgeList.sharedToOrgs', { count: kb.share_count ?? 0 })" placement="top">
                      <div class="feature-badge shared">
                        <t-icon name="share" size="14px" />
                      </div>
                    </t-tooltip>
                  </div>
                </div>
                <div v-if="!authStore.isLiteMode && showKbOriginBadge(kb)" class="bottom-right">
                  <ResourceOriginBadge :variant="kbOriginVariant(kb)" :creator-name="kb.creator_name" />
                </div>
              </div>
            </div>
          </template>
        </div>

        <!-- Collaboration / shared-with-me aggregate view removed: shared KBs now show under "All" or within a specific space -->

        <!-- Filter by space: all knowledge bases in this space (including ones I shared) -->
        <div v-if="spaceSelectionOrgId && spaceKbsLoading" class="kb-list-main-loading">
          <t-loading size="medium" text="" />
        </div>
        <div v-else-if="spaceSelectionOrgId && sortedSpaceKbsList.length > 0" class="kb-card-wrap">
          <template v-for="(shared, index) in sortedSpaceKbsList"
            :key="'shared-' + (shared.share_id || `agent-${shared.knowledge_base?.id}-${shared.source_from_agent?.agent_id || ''}`)">
            <!-- Shared by me: entries in this space that I created and shared myself; header only on the first is_mine entry -->
            <div v-if="showShareGroupHeaders && shared.is_mine && index === 0" class="kb-section-header"
              role="button" tabindex="0" @click="toggleKbSection('sharedByMe')"
              @keydown.enter.prevent="toggleKbSection('sharedByMe')"
              @keydown.space.prevent="toggleKbSection('sharedByMe')">
              <t-icon name="share" size="14px" />
              <span>{{ $t('knowledgeList.sections.sharedByMe') }}</span>
              <span class="kb-section-count">{{ spaceKbSectionCounts.sharedByMe }}</span>
              <t-icon class="kb-section-toggle"
                :name="isKbSectionCollapsed('sharedByMe') ? 'chevron-right' : 'chevron-down'" size="14px" />
            </div>
            <!-- Shared with me · editable: first entry into "shared + editable" from "mine" -->
            <div v-if="showShareGroupHeaders
              && !shared.is_mine
              && isSharedKbEditable(shared.permission)
              && (index === 0 || sortedSpaceKbsList[index - 1].is_mine)" class="kb-section-header"
              role="button" tabindex="0" @click="toggleKbSection('sharedEditable')"
              @keydown.enter.prevent="toggleKbSection('sharedEditable')"
              @keydown.space.prevent="toggleKbSection('sharedEditable')">
              <t-icon name="usergroup-add" size="14px" />
              <t-icon name="edit-1" size="12px" class="kb-section-subicon" />
              <span>{{ $t('knowledgeList.sections.sharedEditable') }}</span>
              <span class="kb-section-count">{{ spaceKbSectionCounts.sharedEditable }}</span>
              <t-icon class="kb-section-toggle"
                :name="isKbSectionCollapsed('sharedEditable') ? 'chevron-right' : 'chevron-down'" size="14px" />
            </div>
            <!-- Shared with me · view only: first entry into "viewer" from "editable shared / mine" -->
            <div v-if="showShareGroupHeaders
              && !shared.is_mine
              && !isSharedKbEditable(shared.permission)
              && (index === 0
                || sortedSpaceKbsList[index - 1].is_mine
                || isSharedKbEditable(sortedSpaceKbsList[index - 1].permission))" class="kb-section-header"
              role="button" tabindex="0" @click="toggleKbSection('sharedReadonly')"
              @keydown.enter.prevent="toggleKbSection('sharedReadonly')"
              @keydown.space.prevent="toggleKbSection('sharedReadonly')">
              <t-icon name="usergroup-add" size="14px" />
              <t-icon name="browse" size="12px" class="kb-section-subicon" />
              <span>{{ $t('knowledgeList.sections.sharedReadonly') }}</span>
              <span class="kb-section-count">{{ spaceKbSectionCounts.sharedReadonly }}</span>
              <t-icon class="kb-section-toggle"
                :name="isKbSectionCollapsed('sharedReadonly') ? 'chevron-right' : 'chevron-down'" size="14px" />
            </div>
            <div v-show="!isSpaceKbCollapsed(shared)" class="kb-card shared-kb-card" :class="{
              'kb-type-document': (shared.knowledge_base.type || 'document') === 'document',
              'kb-type-faq': shared.knowledge_base.type === 'faq'
            }" @click="handleSharedKbClick(shared)">
              <!-- Card header -->
              <div class="card-header">
                <span class="card-title" :title="shared.knowledge_base.name">
                  <KbWikiBadge v-if="isWikiKb(shared.knowledge_base)" />
                  <span class="card-title-text">{{ shared.knowledge_base.name }}</span>
                </span>
                <t-tooltip v-if="!shared.is_mine" :content="$t('knowledgeList.menu.viewDetails')" placement="top">
                  <button type="button" class="shared-detail-trigger" @click.stop="openSharedDetail(shared)"
                    :aria-label="$t('knowledgeList.menu.viewDetails')">
                    <t-icon name="info-circle" size="16px" />
                  </button>
                </t-tooltip>
              </div>

              <!-- Card content -->
              <div class="card-content">
                <div class="card-description">
                  {{ shared.knowledge_base.description || $t('knowledgeBase.noDescription') }}
                </div>
              </div>

              <!-- Card footer -->
              <div class="card-bottom">
                <div class="bottom-left">
                  <div class="feature-badges">
                    <t-tooltip
                      :content="shared.knowledge_base.type === 'faq' ? $t('knowledgeEditor.basic.typeFAQ') : $t('knowledgeEditor.basic.typeDocument')"
                      placement="top">
                      <div class="feature-badge"
                        :class="{ 'type-document': (shared.knowledge_base.type || 'document') === 'document', 'type-faq': shared.knowledge_base.type === 'faq' }">
                        <t-icon :name="shared.knowledge_base.type === 'faq' ? 'chat-bubble-help' : 'folder'"
                          size="14px" />
                        <span class="badge-count">{{ shared.knowledge_base.type === 'faq' ?
                          (shared.knowledge_base.chunk_count ??
                            '-') : (shared.knowledge_base.knowledge_count ?? '-') }}</span>
                      </div>
                    </t-tooltip>
                  </div>
                </div>
              </div>
            </div>
          </template>
        </div>

        <!-- Empty state for "All": keep the "New knowledge base" CTA, since this is the genuine empty-space-with-no-KBs case -->
        <div v-if="spaceSelection === 'all' && filteredKnowledgeBases.length === 0 && !loading" class="empty-state">
          <img class="empty-img" src="@/assets/img/upload.svg" alt="">
          <span class="empty-txt">{{ $t('knowledgeList.empty.title') }}</span>
          <span class="empty-desc">{{ $t('knowledgeList.empty.description') }}</span>
          <t-button v-if="authStore.hasRole('contributor')" class="kb-create-btn empty-state-btn"
            data-guide="kb-list-create" @click="handleCreateKnowledgeBase">
            <template #icon><t-icon name="folder-add" /></template>
            {{ $t('knowledgeList.create') }}
          </t-button>
        </div>

        <!-- Empty state for "Favorites": no create button — "no favorites" ≠ "no knowledge bases",
             The correct guidance is "go star it," not "create another one." -->
        <div v-if="spaceSelection === 'favorites' && filteredKnowledgeBases.length === 0 && !loading"
          class="empty-state">
          <t-icon name="star" size="48px" class="empty-icon" />
          <span class="empty-txt">{{ $t('knowledgeList.empty.favoritesTitle') }}</span>
          <span class="empty-desc">{{ $t('knowledgeList.empty.favoritesDescription') }}</span>
        </div>

        <!-- Recent empty state: similarly, the guidance is "go open one." -->
        <div v-if="spaceSelection === 'recents' && filteredKnowledgeBases.length === 0 && !loading" class="empty-state">
          <t-icon name="history" size="48px" class="empty-icon" />
          <span class="empty-txt">{{ $t('knowledgeList.empty.recentsTitle') }}</span>
          <span class="empty-desc">{{ $t('knowledgeList.empty.recentsDescription') }}</span>
        </div>

        <!-- My knowledge bases empty state -->
        <div v-if="spaceSelection === 'mine' && kbs.length === 0 && !loading" class="empty-state">
          <img class="empty-img" src="@/assets/img/upload.svg" alt="">
          <span class="empty-txt">{{ $t('knowledgeList.empty.title') }}</span>
          <span class="empty-desc">{{ $t('knowledgeList.empty.description') }}</span>
          <t-button v-if="authStore.hasRole('contributor')" class="kb-create-btn empty-state-btn"
            data-guide="kb-list-create" @click="handleCreateKnowledgeBase">
            <template #icon><t-icon name="folder-add" /></template>
            {{ $t('knowledgeList.create') }}
          </t-button>
        </div>

        <!-- Space knowledge bases empty state -->
        <div v-if="spaceSelectionOrgId && !spaceKbsLoading && spaceKbsList.length === 0" class="empty-state">
          <img class="empty-img" src="@/assets/img/upload.svg" alt="">
          <span class="empty-txt">{{ $t('knowledgeList.empty.sharedTitle') }}</span>
          <span class="empty-desc">{{ $t('knowledgeList.empty.sharedDescription') }}</span>
        </div>
      </div>
    </div>

    <!-- Delete confirmation dialog -->
    <t-dialog v-model:visible="deleteVisible" dialogClassName="del-knowledge-dialog" :closeBtn="false" :cancelBtn="null"
      :confirmBtn="null">
      <div class="circle-wrap">
        <div class="dialog-header">
          <img class="circle-img" src="@/assets/img/circle.png" alt="">
          <span class="circle-title">{{ $t('knowledgeList.delete.confirmTitle') }}</span>
        </div>
        <span class="del-circle-txt">
          {{ $t('knowledgeList.delete.confirmMessage', { name: deletingKb?.name ?? '' }) }}
        </span>
        <div class="circle-btn">
          <span class="circle-btn-txt" @click="deleteVisible = false">{{ $t('common.cancel') }}</span>
          <span class="circle-btn-txt confirm" @click="confirmDelete">{{ $t('knowledgeList.delete.confirmButton')
          }}</span>
        </div>
      </div>
    </t-dialog>

    <!-- Knowledge base editor (unified create/edit component) -->
    <KnowledgeBaseEditorModal :visible="uiStore.showKBEditorModal" :mode="uiStore.kbEditorMode"
      :kb-id="uiStore.currentKBId || undefined" :initial-type="uiStore.kbEditorType"
      @update:visible="(val) => val ? null : uiStore.closeKBEditor()" @success="handleKBEditorSuccess" />

    <!-- Share knowledge base dialog -->
    <ShareKnowledgeBaseDialog v-model:visible="shareDialogVisible" :knowledge-base-id="sharingKbId"
      :knowledge-base-name="sharingKbName" @shared="handleShareSuccess" />

    <!-- Right side: shared knowledge base details panel -->
    <Teleport to="body">
      <Transition name="shared-detail-drawer">
        <div v-if="sharedDetailPanelVisible && currentSharedKbForDetail" class="shared-detail-drawer-overlay"
          @click.self="closeSharedDetailPanel">
          <div class="shared-detail-drawer">
            <div class="shared-detail-drawer-header">
              <h3 class="shared-detail-drawer-title">{{ $t('knowledgeList.detail.title') }}</h3>
              <button type="button" class="shared-detail-drawer-close" @click="closeSharedDetailPanel"
                :aria-label="$t('general.close')">
                <t-icon name="close" size="20px" />
              </button>
            </div>
            <div class="shared-detail-drawer-body">
              <div class="shared-detail-row">
                <span class="shared-detail-label">{{ $t('knowledgeBase.name') }}</span>
                <span class="shared-detail-value">{{ currentSharedKbForDetail.knowledge_base.name }}</span>
              </div>
              <div class="shared-detail-row">
                <span class="shared-detail-label">{{ $t('knowledgeList.detail.sourceType') }}</span>
                <span class="shared-detail-value shared-detail-source-type">
                  {{ currentSharedKbForDetail.source_from_agent ? $t('knowledgeList.detail.sourceTypeAgent') :
                    $t('knowledgeList.detail.sourceTypeKbShare') }}
                </span>
              </div>
              <div class="shared-detail-row">
                <span class="shared-detail-label">{{ currentSharedKbForDetail.source_from_agent ?
                  $t('knowledgeList.detail.sourceFromAgent') : $t('knowledgeList.detail.sourceOrg') }}</span>
                <span class="shared-detail-value shared-detail-org">
                  <img src="@/assets/img/organization-green.svg" class="shared-detail-org-icon" alt=""
                    aria-hidden="true" />
                  {{ currentSharedKbForDetail.source_from_agent ? currentSharedKbForDetail.source_from_agent.agent_name
                    :
                    currentSharedKbForDetail.org_name }}
                </span>
              </div>
              <div v-if="currentSharedKbForDetail.source_from_agent" class="shared-detail-row">
                <span class="shared-detail-label">{{ $t('knowledgeList.detail.agentKbStrategy') }}</span>
                <span class="shared-detail-value">
                  {{ agentKbStrategyText(currentSharedKbForDetail.source_from_agent?.kb_selection_mode ?? '') }}
                </span>
              </div>
              <div class="shared-detail-row">
                <span class="shared-detail-label">{{ $t('knowledgeList.detail.sharedAt') }}</span>
                <span class="shared-detail-value">{{ formatStringDate(new Date(currentSharedKbForDetail.shared_at))
                }}</span>
              </div>
              <div class="shared-detail-row">
                <span class="shared-detail-label">{{ $t('knowledgeList.detail.myPermission') }}</span>
                <t-tag size="small"
                  :theme="currentSharedKbForDetail.permission === 'admin' ? 'primary' : currentSharedKbForDetail.permission === 'editor' ? 'warning' : 'default'">
                  {{ $t(`organization.role.${currentSharedKbForDetail.permission}`) }}
                </t-tag>
              </div>
            </div>
            <div class="shared-detail-drawer-footer">
              <t-button theme="default" variant="outline" @click="closeSharedDetailPanel">{{ $t('common.close')
              }}</t-button>
              <t-button theme="primary" class="go-to-kb-btn" @click="goToSharedKbFromPanel">
                <t-icon name="browse" />
                {{ $t('knowledgeList.detail.goToKb') }}
              </t-button>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>

    <ContextualGuide tour="kbList" :when="showKbListContextualGuide" />
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref, computed, watch, nextTick } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { MessagePlugin, Icon as TIcon } from 'tdesign-vue-next'
import { deleteKnowledgeBase, duplicateKnowledgeBase, togglePinKnowledgeBase } from '@/api/knowledge-base'
import { useChatResourcesStore } from '@/stores/chatResources'
import { formatStringDate } from '@/utils/index'
import { useUIStore } from '@/stores/ui'
import { useAuthStore } from '@/stores/auth'
import { useOrganizationStore } from '@/stores/organization'
import { listOrganizationSharedKnowledgeBases, type SharedKnowledgeBase, type OrganizationSharedKnowledgeBaseItem, type SourceFromAgentInfo } from '@/api/organization'
import { mergeAllScopeKnowledgeBases, type OwnedKnowledgeBase, type SharedKnowledgeBaseLike } from './kbListMerge'
import KnowledgeBaseEditorModal from './KnowledgeBaseEditorModal.vue'
import KbWikiBadge from './components/KbWikiBadge.vue'
import ShareKnowledgeBaseDialog from '@/components/ShareKnowledgeBaseDialog.vue'
import ListSpaceSidebar from '@/components/ListSpaceSidebar.vue'
import ResourceOriginBadge from '@/components/ResourceOriginBadge.vue'
import { shouldShowResourceOriginBadge } from '@/utils/card-list-badge'
import ContextualGuide from '@/components/ContextualGuide.vue'
import { isContextualGuideDone, markContextualGuideDone } from '@/config/contextualGuides'
import { useTenantModelReadiness } from '@/composables/useTenantModelReadiness'
import { useI18n } from 'vue-i18n'
import { useListUrlState } from '@/composables/useListUrlState'
import { useResourcePins } from '@/composables/useResourcePins'

const router = useRouter()
const route = useRoute()
const uiStore = useUIStore()
const authStore = useAuthStore()
const { loaded: modelsReadyLoaded, isReadyForDocumentKb } = useTenantModelReadiness()
const orgStore = useOrganizationStore()
const chatResources = useChatResourcesStore()
const { t } = useI18n()

// Left-side space selection: defaults based on the current role.
// A Viewer typically owns 0 KB in that space, so "Mine" shows an empty state and hides the shared KBs,
// which is very misleading; so Viewer defaults to "all" (both mine and shared with me are shown).
// Contributor and above mainly manage KBs they created, so it still defaults to "mine".
//
// State lives in `?scope=` so links are shareable/bookmarkable; the
// composable handles two-way sync with the URL. We keep "mine" as the
// stored value (not "workspace") for back-compat with any external link
// that might point at the old query — its display label is rebranded
// via ListSpaceSidebar's workspaceLabel computed.
const defaultScope: 'all' | 'mine' = authStore.hasRole('contributor') ? 'mine' : 'all'
const { scope: spaceSelection, creator: creatorFilter } = useListUrlState({
  defaultScope,
  defaultCreator: 'all',
})

// Per-user favorites + recents (localStorage-backed). isFavorite & touchRecent
// are wired into card render and click handlers below.
const pins = useResourcePins()
const kbFavoritesCount = computed(
  () => pins.favorites.value.filter((e) => e.type === 'kb').length
)
const kbRecentsCount = computed(
  () => pins.recents.value.filter((e) => e.type === 'kb').length
)

interface KB {
  id: string;
  name: string;
  description?: string;
  updated_at?: string;
  created_at?: string;
  pinned_at?: string;
  embedding_model_id?: string;
  summary_model_id?: string;
  type?: 'document' | 'faq';
  showMore?: boolean;
  vlm_config?: { enabled?: boolean; model_id?: string };
  extract_config?: { enabled?: boolean };
  storage_provider_config?: { provider?: string };
  storage_config?: { provider?: string; bucket_name?: string }; // legacy
  question_generation_config?: { enabled?: boolean; question_count?: number };
  knowledge_count?: number;
  chunk_count?: number;
  isProcessing?: boolean;
  processing_count?: number;
  share_count?: number;
  is_pinned?: boolean;
  // creator_id is the owner-id matched against authStore.user.id when
  // gating the per-card more-menu (Settings / Delete). Empty for legacy
  // KBs created before PR 5; those fall back to the role gate.
  creator_id?: string;
  // creator_name is filled in by the backend list API and is only used for the tooltip on the source badge in the card's bottom-right corner.
  creator_name?: string;
}

const kbs = ref<KB[]>([])
const loading = ref(false)
const deleteVisible = ref(false)
const deletingKb = ref<KB | null>(null)
const currentMoreIndex = ref<number>(-1)
const highlightedKbId = ref<string | null>(null)
const highlightedCardRef = ref<HTMLElement | null>(null)
const uploadTasks = ref<UploadTaskState[]>([])
const uploadCleanupTimers = new Map<string, ReturnType<typeof setTimeout>>()
let uploadRefreshTimer: ReturnType<typeof setTimeout> | null = null
const UPLOAD_CLEANUP_DELAY = 10000

// Share dialog state
const shareDialogVisible = ref(false)
const sharingKbId = ref('')
const sharingKbName = ref('')

// Shared knowledge bases (everything cross-tenant shared to me, including
// viewer-only). Used by the per-space views and the "all" aggregate so
// readers still see read-only shares — those are valid resources, just
// not editable.
const sharedKbs = computed<SharedKnowledgeBase[]>(() => orgStore.sharedKnowledgeBases || [])

const allKnowledgeBases = computed(() => kbs.value.length + sharedKbs.value.length)

// The currently selected item is a space ID (not "all", not "mine", not a pseudo-scope like favorites/recent)
// NB: keep the reserved-scope list in sync with ListSpaceSidebar's
// non-org buckets — otherwise a new pseudo-scope (e.g. "favorites")
// falls through here and triggers the per-space code paths, which
// renders an extra "no shared KB" empty state on top of the real view.
const RESERVED_SCOPES = new Set(['all', 'mine', 'favorites', 'recents'])
const spaceSelectionOrgId = computed(() => {
  const s = spaceSelection.value
  return !!s && !RESERVED_SCOPES.has(s)
})

// Knowledge bases shared with me in the current space (old: others' shares only; kept for compatibility)
const sharedKbsByOrg = computed(() => {
  const orgId = spaceSelection.value
  if (orgId === 'all' || orgId === 'mine') return []
  return sharedKbs.value.filter(s => s.organization_id === orgId)
})

// Space view: all knowledge bases in this space (including ones I shared), request the new API when a space is selected
const spaceKbsList = ref<OrganizationSharedKnowledgeBaseItem[]>([])
const spaceKbsLoading = ref(false)

// Stable ordering for the "Workspace" view: within this space, "created by me" comes first, "created by teammates" comes after;
// Within sub-segments, preserve the server's pinned-priority order. For the contributor view, add "This space · view only"
// The group heading is inserted right at the transition point; other roles don't see the heading, so a pure ordering change is harmless.
// Ordering for the 「本空间」 tab:
//   1. pinned KBs (mine or teammate), newest pin first
//   2. my non-pinned KBs
// 3. teammate non-pinned KBs (rendered under the「本空间 · 仅查看」header)
//
// Pin is per-user as of migration 000050, so a teammate-created KB that
// the caller has personally pinned must float into the pinned section
// even though it would otherwise live in the teammate sub-group. The
// previous version only bucketed by isMyKb and silently demoted these
// pinned-but-teammate KBs.
const sortedMineKbs = computed<KB[]>(() => {
  return [...kbs.value].sort((a, b) => {
    const ap = a.is_pinned ? 0 : 1
    const bp = b.is_pinned ? 0 : 1
    if (ap !== bp) return ap - bp
    if (a.is_pinned && b.is_pinned) {
      const at = a.pinned_at ? Date.parse(a.pinned_at as string) : 0
      const bt = b.pinned_at ? Date.parse(b.pinned_at as string) : 0
      if (at !== bt) return bt - at
    }
    const am = isMyKb(a) ? 0 : 1
    const bm = isMyKb(b) ? 0 : 1
    if (am !== bm) return am - bm
    const ac = a.created_at ? Date.parse(a.created_at as string) : 0
    const bc = b.created_at ? Date.parse(b.created_at as string) : 0
    return bc - ac
  })
})

// Stable ordering under the space view: KBs I created (is_mine) come first, and the remaining shared portion is then sorted by
// editable / view-only — this keeps the space list's visual order consistent with the "All" view.
const sortedSpaceKbsList = computed(() => {
  return [...spaceKbsList.value].sort((a, b) => {
    const aMine = a.is_mine ? 0 : 1
    const bMine = b.is_mine ? 0 : 1
    if (aMine !== bMine) return aMine - bMine
    const aE = isSharedKbEditable(a.permission) ? 0 : 1
    const bE = isSharedKbEditable(b.permission) ? 0 : 1
    return aE - bE
  })
})
const spaceCountByOrg = ref<Record<string, number>>({})

// Number of shared knowledge bases per space (used for sidebar display): prefer the per-space total returned by the API, otherwise use the "shared with me" count
const sharedCountByOrg = computed<Record<string, number>>(() => {
  const map: Record<string, number> = {}
  sharedKbs.value.forEach(s => {
    const id = s.organization_id
    if (!id) return
    map[id] = (map[id] || 0) + 1
  })
    ; (orgStore.organizations || []).forEach(org => {
      if (map[org.id] === undefined) map[org.id] = 0
    })
  return map
})
const effectiveSharedCountByOrg = computed<Record<string, number>>(() => {
  const base = sharedCountByOrg.value
  const merged = { ...base }
  Object.keys(spaceCountByOrg.value).forEach(orgId => {
    merged[orgId] = spaceCountByOrg.value[orgId]
  })
  return merged
})

// Favorites / Recents views: hydrate pin entries by id against every KB
// the user can already see in this page (own + cross-tenant shared). KBs
// the user no longer has access to (deleted / share revoked) are dropped
// silently — the pin survives until the next mutation, which keeps the
// composable simple at the cost of harmless ghost entries.
//
// Order:
//   - favorites: most recently starred first (PinEntry.ts desc)
//   - recents: most recently opened first (also ts desc, already sorted)
const kbResourceIndex = computed(() => {
  const map = new Map<string, { kb: any; isMine: boolean; shared?: SharedKnowledgeBase }>()
  for (const kb of kbs.value) {
    map.set(kb.id, { kb, isMine: true })
  }
  for (const shared of sharedKbs.value) {
    if (!shared.knowledge_base) continue
    if (!map.has(shared.knowledge_base.id)) {
      map.set(shared.knowledge_base.id, { kb: shared.knowledge_base, isMine: false, shared })
    }
  }
  return map
})

const favoritesList = computed(() => {
  return pins.favorites.value
    .filter((e) => e.type === 'kb')
    .map((e) => {
      const entry = kbResourceIndex.value.get(e.id)
      if (!entry) return null
      if (entry.isMine) {
        return { ...entry.kb, isMine: true as const, _pinTs: e.ts }
      }
      const s = entry.shared!
      return {
        ...entry.kb,
        isMine: false as const,
        permission: s.permission,
        shared_at: s.shared_at,
        share_id: s.share_id,
        org_name: s.org_name,
        _pinTs: e.ts,
      } as any
    })
    .filter((x): x is NonNullable<typeof x> => x !== null)
})

const recentsList = computed(() => {
  return pins.recents.value
    .filter((e) => e.type === 'kb')
    .map((e) => {
      const entry = kbResourceIndex.value.get(e.id)
      if (!entry) return null
      if (entry.isMine) {
        return { ...entry.kb, isMine: true as const, _pinTs: e.ts }
      }
      const s = entry.shared!
      return {
        ...entry.kb,
        isMine: false as const,
        permission: s.permission,
        shared_at: s.shared_at,
        share_id: s.share_id,
        org_name: s.org_name,
        _pinTs: e.ts,
      } as any
    })
    .filter((x): x is NonNullable<typeof x> => x !== null)
})

// Editable permission: editor / admin. Viewer goes into the "view only" group.
// Judge by share-level permission (not space role) — someone who got viewer access across spaces
// genuinely can't edit that KB even if I'm the owner of this space; conversely, someone who got editor access across spaces
// genuinely can edit it even if I'm only a contributor in this space.
const EDITABLE_PERMS = new Set(['admin', 'editor'])
function isSharedKbEditable(perm: string | undefined): boolean {
  return !!perm && EDITABLE_PERMS.has(perm)
}

// Whether to show the "editable / view-only" sub-grouping in the shared section: only meaningful for the middle tiers (contributor / editor).
// Viewer is read-only anyway, and admin/owner view manages everything uniformly, so grouping would just fragment things.
// This is purely UI presentation — permissions are enforced by the backend as the source of truth; don't treat this as a security boundary.
// The group heading applies to all roles — Pinned / Created by me / This space · view only / Shared with me
// are all objective information based on "creator + source" and don't depend on the current user's write permission.
// Originally shown only to contributors, to avoid implying a permission distinction ("view only") in front of admin/owner —
// but in practice admin/owner also want to distinguish cards they created from ones created by teammates, so
// it's now enabled for everyone. If we ever need to strip the permission connotation from the heading, just change the i18n copy;
// no need to touch this computed again.
const showShareGroupHeaders = computed(() => true)

// Group heading for KBs in the same space created by someone other than the current user.
// Contributor / viewer have no write permission on these KBs in this space, so it's labeled "view only";
// admin/owner actually have edit permission across the whole space, so "view only" would repeatedly mislead them into thinking
// They didn't cannot fix it — this part is really "KB others in workspace made", by ownership
// rather than permission is more accurate to label.
const tenantSectionLabelKey = computed(() =>
  authStore.hasRole('admin')
    ? 'knowledgeList.sections.tenantOthers'
    : 'knowledgeList.sections.tenantReadonly'
)

// Icon aligns with text above: admin/owner sees "this space · other members", by ownership
// split makes sense; usergroup (multi-person) fits; contributor/viewer sees "view only",
// keep browse (eye) icon to convey "can view, can't edit" semantics.
const tenantSectionIconName = computed(() =>
  authStore.hasRole('admin') ? 'usergroup' : 'browse'
)

// Group collapse: ephemeral, only applies for current session, not persisted to localStorage/server.
// Using a "collapsed set" instead of an "expanded set" is because default is fully expanded — an empty Set
// means the initial fully-expanded state, avoiding having to maintain a default every time a new section is added.
type KbSectionKey = 'pinned' | 'mine' | 'tenantOthers' | 'sharedByMe' | 'sharedEditable' | 'sharedReadonly'
const collapsedKbSections = ref<Set<KbSectionKey>>(new Set())
const isKbSectionCollapsed = (key: KbSectionKey) => collapsedKbSections.value.has(key)
const toggleKbSection = (key: KbSectionKey) => {
  // Reassigning a new Set is to change the ref's .value identity and trigger template re-render;
  // direct .add/.delete works fine on Vue 3's reactive Set too, but ref(Set)'s
  // inner proxy behavior varies slightly across versions — full replacement is safest.
  const next = new Set(collapsedKbSections.value)
  if (next.has(key)) next.delete(key)
  else next.add(key)
  collapsedKbSections.value = next
}
// Determines which group a KB belongs to — same logic used by several v-if checks in the template,
// extracted for reuse when v-show-ing cards, to avoid rebuilding the v-if for all 5 groups.
//
// Input has two shapes:
// Elements from filteredKnowledgeBases explicitly carry an `isMine` flag (see
// the spread in filteredKnowledgeBases; cross-space shared items get isMine=false).
// Elements from sortedMineKbs are raw KBs, with no isMine or permission field.
// Cross-space shared entries always have `permission`, same-space entries never do, so "no permission"
// is the same-space safety signal. Combined: check isMine first, then fall back to whether permission exists.
const kbSectionOf = (kb: any): KbSectionKey => {
  if (kb?.is_pinned) return 'pinned'
  const isOwnTenant = kb?.isMine === true || (kb?.isMine !== false && kb?.permission == null)
  if (isOwnTenant) return isMyKb(kb) ? 'mine' : 'tenantOthers'
  return isSharedKbEditable(kb?.permission) ? 'sharedEditable' : 'sharedReadonly'
}

// The space-filtered view (sortedSpaceKbsList) entries have a different shape: is_mine directly marks
// "shared by me", the rest follow sharedEditable / sharedReadonly based on permission.
const spaceKbSectionOf = (shared: any): KbSectionKey => {
  if (shared?.is_mine) return 'sharedByMe'
  return isSharedKbEditable(shared?.permission) ? 'sharedEditable' : 'sharedReadonly'
}
const isSpaceKbCollapsed = (shared: any): boolean => isKbSectionCollapsed(spaceKbSectionOf(shared))

// How many cards each group actually has — just reuse the group-determination function. Showing
// "(N)" on the group title lets users see at a glance how much gets hidden when collapsed, and makes it easy to verify filter results.
const emptyKbCounts = (): Record<KbSectionKey, number> => ({
  pinned: 0, mine: 0, tenantOthers: 0, sharedByMe: 0, sharedEditable: 0, sharedReadonly: 0,
})
const filteredKbSectionCounts = computed<Record<KbSectionKey, number>>(() => {
  const c = emptyKbCounts()
  filteredKnowledgeBases.value.forEach(kb => { c[kbSectionOf(kb)]++ })
  return c
})
const mineKbSectionCounts = computed<Record<KbSectionKey, number>>(() => {
  const c = emptyKbCounts()
  sortedMineKbs.value.forEach(kb => { c[kbSectionOf(kb)]++ })
  return c
})
const spaceKbSectionCounts = computed<Record<KbSectionKey, number>>(() => {
  const c = emptyKbCounts()
  sortedSpaceKbsList.value.forEach(shared => { c[spaceKbSectionOf(shared)]++ })
  return c
})

// Filtered knowledge bases: All = mine + all shared; Mine = mine only
//
// Favorites / Recents reuse the same render path as `all` — they're just
// pre-filtered, pre-ordered slices, so the existing kb-card / shared
// kb-card templates render them with zero extra markup. Order is
// preserved via the upstream array (pins order is ts-desc).
const filteredKnowledgeBases = computed(() => {
  if (spaceSelection.value === 'favorites') {
    return favoritesList.value
  }
  if (spaceSelection.value === 'recents') {
    return recentsList.value
  }
  if (spaceSelection.value === 'mine') {
    return kbs.value.map(kb => ({ ...kb, isMine: true as const }))
  }
  if (spaceSelection.value !== 'all') {
    return []
  }
  // The "All" scope merges own + shared KBs. The card template keys each
  // row by `kb.id`, so the same KB surfacing twice — owned *and* shared
  // back, or shared into the caller's view through two different orgs —
  // produced duplicate `v-for` keys and blanked the list once there were
  // ≥2 entries (#795). mergeAllScopeKnowledgeBases de-duplicates by KB id
  // (owned wins; most-privileged share kept) while preserving the existing
  // pinned → mine → teammate → shared(editable-first) ordering.
  return mergeAllScopeKnowledgeBases(
    kbs.value as unknown as OwnedKnowledgeBase[],
    sharedKbs.value as unknown as SharedKnowledgeBaseLike[],
    authStore.user?.id,
  ) as unknown as Array<(KB & { isMine: true }) | (SharedKnowledgeBase['knowledge_base'] & { isMine: false; permission: string; shared_at: string; share_id: string } & any)>
})

const showKbListEmpty = computed(() => {
  if (loading.value) return false
  if (!authStore.hasRole('contributor')) return false
  if (spaceSelection.value === 'all' && filteredKnowledgeBases.value.length === 0) return true
  if (spaceSelection.value === 'mine' && kbs.value.length === 0) return true
  return false
})

const showKbListContextualGuide = computed(
  () => showKbListEmpty.value && !uiStore.showKBEditorModal,
)

interface UploadTaskState {
  uploadId: string
  kbId: string
  fileName?: string
  progress: number
  status: 'uploading' | 'success' | 'error'
  error?: string
}

interface UploadSummary {
  kbId: string
  kbName: string
  total: number
  completed: number
  progress: number
  hasError: boolean
}

const applyKbListData = (data: any[]) => {
  kbs.value = data.map((kb: any) => ({
    ...kb,
    updated_at: kb.updated_at ? formatStringDate(new Date(kb.updated_at)) : '',
    showMore: false,
    isProcessing: kb.is_processing || false,
    processing_count: kb.processing_count || 0
  }))
}

const fetchList = (force = false) => {
  loading.value = true
  // The creator filter only applies to the caller's own tenant KBs (the
  // first call). Shared KBs are inherently "not mine" so we don't filter
  // them server-side; the segmented control is also hidden whenever the
  // user is browsing the shared / per-space scopes.
  return Promise.all([
    chatResources.fetchKnowledgeBasesForList({ creator: creatorFilter.value }, force).then(applyKbListData),
    orgStore.fetchSharedKnowledgeBases({ force }),
    orgStore.fetchOrganizations({ force }),
  ]).finally(() => { loading.value = false }).then(() => {
    // Per-space KB counts are already returned by GET /organizations' resource_counts, stored in orgStore.resourceCounts
    const counts = orgStore.resourceCounts?.knowledge_bases?.by_organization
    if (counts) spaceCountByOrg.value = { ...counts }
  })
}

// When a space is selected, request all KBs within that space (including ones I've shared)
watch(spaceSelection, (val) => {
  // Stale URL guard: an older "协作" view used scope=shared; that view
  // was removed, so normalize back to "all" instead of letting the
  // value fall through to the per-space fetch branch (which would 404
  // on the string "shared").
  if (val === 'shared') {
    spaceSelection.value = 'all'
    return
  }
  if (val === 'all' || val === 'mine' || val === 'favorites' || val === 'recents' || !val) {
    spaceKbsList.value = []
    return
  }
  spaceKbsLoading.value = true
  listOrganizationSharedKnowledgeBases(val).then((res) => {
    if (res.success && res.data) {
      spaceKbsList.value = res.data
      spaceCountByOrg.value = { ...spaceCountByOrg.value, [val]: res.data.length }
    } else {
      spaceKbsList.value = []
    }
  }).finally(() => {
    spaceKbsLoading.value = false
  })
}, { immediate: true })

// Refetch when the creator filter flips. We re-pull the whole list rather
// than filtering in-memory so the server stays the single source of truth
// (and we don't need to worry about stale share_count or pagination later).
watch(creatorFilter, () => {
  fetchList(true)
})

onMounted(() => {
  fetchList().then(() => {
    // Check whether the route params contain a knowledge base ID that needs highlighting
    const highlightKbId = route.query.highlightKbId as string
    if (highlightKbId) {
      triggerHighlightFlash(highlightKbId)
      // Drop the transient highlight param but preserve other state
      // (scope / creator / q) so refreshing doesn't reset the user's view.
      const { highlightKbId: _drop, ...rest } = route.query
      router.replace({ query: rest })
    }
  })

  window.addEventListener('knowledgeFileUploadStart', handleUploadStartEvent as EventListener)
  window.addEventListener('knowledgeFileUploadProgress', handleUploadProgressEvent as EventListener)
  window.addEventListener('knowledgeFileUploadComplete', handleUploadCompleteEvent as EventListener)
  window.addEventListener('knowledgeFileUploaded', handleUploadFinishedEvent as EventListener)
})

onUnmounted(() => {
  window.removeEventListener('knowledgeFileUploadStart', handleUploadStartEvent as EventListener)
  window.removeEventListener('knowledgeFileUploadProgress', handleUploadProgressEvent as EventListener)
  window.removeEventListener('knowledgeFileUploadComplete', handleUploadCompleteEvent as EventListener)
  window.removeEventListener('knowledgeFileUploaded', handleUploadFinishedEvent as EventListener)

  uploadCleanupTimers.forEach(timer => clearTimeout(timer))
  uploadCleanupTimers.clear()
  if (uploadRefreshTimer) {
    clearTimeout(uploadRefreshTimer)
    uploadRefreshTimer = null
  }
})

// Watch for route changes to handle highlight requests from navigation from other pages
watch(() => route.query.highlightKbId, (newKbId) => {
  if (newKbId && typeof newKbId === 'string' && kbs.value.length > 0) {
    triggerHighlightFlash(newKbId)
    const { highlightKbId: _drop, ...rest } = route.query
    router.replace({ query: rest })
  }
})

const openMore = (index: number) => {
  // Only track the currently open index, used for showing active styling
  // The dialog's open/close is managed automatically via v-model
  currentMoreIndex.value = index
}

const onVisibleChange = (visible: boolean) => {
  // Reset the index when the dialog closes
  if (!visible) {
    currentMoreIndex.value = -1
  }
}

const handleSettings = (kb: KB) => {
  // Manually close the dialog
  kb.showMore = false
  goSettings(kb.id)
}

// canManageKBCard mirrors KnowledgeBase.vue's `canManage`, gating the
// destructive items of the per-card menu — Settings, Delete — so a
// Viewer cannot click into them for a KB they don't own. The server
// still rejects the call (PR 5 guards every such mutation with
// OwnedKBOrAdmin) but the UI shouldn't surface buttons the user has
// no authority to use.
//
// The pin item is intentionally NOT gated by this predicate any more:
// pin state is per (user, kb) as of migration 000050 and the backend
// route only requires KB read access, so anyone who can see the card
// should be able to pin it for themselves.
//
// Legacy KBs created before PR 5 have an empty creator_id; treat
// those as tenant-owned (Admin+ may manage) so existing KBs aren't
// suddenly unmanageable for everyone.
function canManageKBCard(kb: KB): boolean {
  const userId = authStore.user?.id || ''
  if (kb.creator_id && userId && kb.creator_id === userId) return true
  return authStore.hasRole('admin')
}

function canDuplicateKBCard(kb: any): boolean {
  return authStore.hasRole('contributor') && kb.isMine !== false
}

// isMyKb is only used to toggle the badge in the bottom-right corner of the card between "created by me" and "created by other member in the same space".
// Different from canManageKBCard: manage permission falls back to admin, the badge purely matches by creator.
// When creator_id is empty (old KBs from before the PR 5 RBAC migration), always treat as tenant-owned — to avoid
// incorrectly marking old KBs shared across the whole space as "created by me".
function isMyKb(kb: { creator_id?: string }): boolean {
  const userId = authStore.user?.id || ''
  return !!(kb.creator_id && userId && kb.creator_id === userId)
}

// kbOriginVariant determines the display form of the badge in the card's bottom-right corner:
// - Created by myself: mine (green "Created by me")
// - Created by another member in the same space: creator variant — shows only the creator's name. The user is always in
// Browsing within a certain workspace (the top TenantSelector already marks the space identity); re-labeling the space name again in the bottom right
// would be redundant information; contributor / admin / owner / viewer
// see consistent badges. When the creator can't be resolved, the creator variant automatically falls back to
// the resourceOrigin.tenant text ("this space"), so no empty label appears.
function kbOriginVariant(kb: { creator_id?: string }): 'mine' | 'creator' {
  return isMyKb(kb) ? 'mine' : 'creator'
}

function showKbOriginBadge(kb: { creator_id?: string; creator_name?: string }): boolean {
  return shouldShowResourceOriginBadge({
    section: kbSectionOf(kb),
    variant: kbOriginVariant(kb),
    creatorName: kb.creator_name,
    showSectionHeaders: showShareGroupHeaders.value,
  })
}

// Handle settings by ID (used for knowledge bases under the "All" tab)
const handleSettingsById = (id: string) => {
  goSettings(id)
}

// Handle deletion by ID (used for knowledge bases under the "All" tab)
const handleDeleteById = (id: string) => {
  const kb = kbs.value.find(k => k.id === id)
  if (kb) {
    deletingKb.value = kb
    deleteVisible.value = true
  }
}

const handleTogglePin = async (kb: KB) => {
  kb.showMore = false
  try {
    const res: any = await togglePinKnowledgeBase(kb.id)
    if (res.success) {
      MessagePlugin.success(
        res.data.is_pinned ? t('knowledgeList.pin.pinSuccess') : t('knowledgeList.pin.unpinSuccess')
      )
      fetchList(true)
    }
  } catch {
    MessagePlugin.error(t('knowledgeList.pin.failed'))
  }
}

const handleTogglePinById = async (id: string) => {
  try {
    const res: any = await togglePinKnowledgeBase(id)
    if (res.success) {
      MessagePlugin.success(
        res.data.is_pinned ? t('knowledgeList.pin.pinSuccess') : t('knowledgeList.pin.unpinSuccess')
      )
      fetchList(true)
    }
  } catch {
    MessagePlugin.error(t('knowledgeList.pin.failed'))
  }
}

const handleDuplicate = async (kb: KB) => {
  kb.showMore = false
  await duplicateKB(kb.id)
}

const handleDuplicateById = async (id: string) => {
  await duplicateKB(id)
}

const duplicateKB = async (id: string) => {
  try {
    const res: any = await duplicateKnowledgeBase(id)
    if (res?.success) {
      const newKbId = res.data?.target_id || res.data?.knowledge_base?.id
      MessagePlugin.success(t('knowledgeList.messages.duplicateSuccess'))
      await fetchList(true)
      if (newKbId) {
        triggerHighlightFlash(newKbId)
      }
    } else {
      MessagePlugin.error(res?.message || t('knowledgeList.messages.duplicateFailed'))
    }
  } catch (e: any) {
    MessagePlugin.error(e?.message || t('knowledgeList.messages.duplicateFailed'))
  }
}

const handleShare = (kb: KB) => {
  // Manually close the dialog
  kb.showMore = false
  sharingKbId.value = kb.id
  sharingKbName.value = kb.name
  shareDialogVisible.value = true
}

const handleShareSuccess = () => {
  // Refresh the list after a successful share
  fetchList(true)
}

const handleSharedKbClick = (sharedKb: SharedKnowledgeBase) => {
  pins.touchRecent('kb', sharedKb.knowledge_base.id)
  // Navigate to the shared knowledge base detail page
  router.push(`/platform/knowledge-bases/${sharedKb.knowledge_base.id}`)
}

// Handle clicks on shared knowledge base cards in the "All" tab (go straight into the knowledge base)
const handleSharedKbClickFromAll = (kb: any) => {
  pins.touchRecent('kb', kb.id)
  router.push(`/platform/knowledge-bases/${kb.id}`)
}

// Right-side detail panel: shared knowledge base details (both direct shares and agent-sourced)
type SharedKbDetailItem = SharedKnowledgeBase & { is_mine?: boolean; source_from_agent?: SourceFromAgentInfo }
const sharedDetailPanelVisible = ref(false)
const currentSharedKbForDetail = ref<SharedKbDetailItem | null>(null)

const closeSharedDetailPanel = () => {
  sharedDetailPanelVisible.value = false
  currentSharedKbForDetail.value = null
}

// Open the right-side detail panel (shared card in the "All" tab)
const openSharedDetailFromAll = (kb: any) => {
  const sharedKb = sharedKbs.value.find(s => s.knowledge_base.id === kb.id)
  if (sharedKb) {
    currentSharedKbForDetail.value = sharedKb
    sharedDetailPanelVisible.value = true
  }
}

// Open the right-side detail panel (space tab: direct share or agent-sourced)
const openSharedDetail = (sharedKb: SharedKbDetailItem) => {
  currentSharedKbForDetail.value = sharedKb
  sharedDetailPanelVisible.value = true
}

// Agent policy text for the knowledge base (used when the drawer's "source" is an agent)
const agentKbStrategyText = (mode: string) => {
  if (mode === 'all') return t('knowledgeList.detail.agentKbStrategyAll')
  if (mode === 'selected') return t('knowledgeList.detail.agentKbStrategySelected')
  return t('knowledgeList.detail.agentKbStrategyNone')
}

// Enter the knowledge base from the right-side panel
const goToSharedKbFromPanel = () => {
  if (currentSharedKbForDetail.value) {
    router.push(`/platform/knowledge-bases/${currentSharedKbForDetail.value.knowledge_base.id}`)
    closeSharedDetailPanel()
  }
}

const handleDelete = (kb: KB) => {
  // Manually close the dialog
  kb.showMore = false
  deletingKb.value = kb
  deleteVisible.value = true
}

const confirmDelete = () => {
  if (!deletingKb.value) return

  deleteKnowledgeBase(deletingKb.value.id).then((res: any) => {
    if (res.success) {
      MessagePlugin.success(t('knowledgeList.messages.deleted'))
      deleteVisible.value = false
      deletingKb.value = null
      fetchList(true)
    } else {
      MessagePlugin.error(res.message || t('knowledgeList.messages.deleteFailed'))
    }
  }).catch((e: any) => {
    MessagePlugin.error(e?.message || t('knowledgeList.messages.deleteFailed'))
  })
}

const isInitialized = (kb: KB) => {
  // LLM (summary) model is always required
  if (!kb.summary_model_id || kb.summary_model_id === '') return false
  // Embedding model only required when RAG indexing is enabled (vector or keyword)
  const strategy = (kb as any).indexing_strategy
  const needsEmbedding = !strategy || strategy.vector_enabled || strategy.keyword_enabled
  if (needsEmbedding && (!kb.embedding_model_id || kb.embedding_model_id === '')) return false
  return true
}

const isWikiKb = (kb: unknown) =>
  !!(kb as { indexing_strategy?: { wiki_enabled?: boolean } } | null | undefined)?.indexing_strategy?.wiki_enabled

// Compute whether there are any uninitialized knowledge bases
const hasUninitializedKbs = computed(() => {
  return kbs.value.some(kb => !isInitialized(kb))
})

const getKbDisplayName = (kbId: string) => {
  const target = kbs.value.find(kb => kb.id === kbId)
  if (target?.name) return target.name
  return t('knowledgeList.uploadProgress.unknownKb', { id: kbId }) as string
}

const uploadSummaries = computed<UploadSummary[]>(() => {
  if (!uploadTasks.value.length) return []
  const grouped: Record<string, UploadTaskState[]> = {}
  uploadTasks.value.forEach(task => {
    const kbKey = String(task.kbId)
    if (!grouped[kbKey]) grouped[kbKey] = []
    grouped[kbKey].push(task)
  })
  return Object.entries(grouped).map(([kbId, tasks]) => {
    const total = tasks.length
    const completed = tasks.filter(task => task.status !== 'uploading').length
    const progressSum = tasks.reduce((sum, task) => sum + (task.progress ?? 0), 0)
    const avgProgress = total === 0 ? 0 : Math.min(100, Math.max(0, Math.round(progressSum / total)))
    const hasError = tasks.some(task => task.status === 'error')
    return {
      kbId,
      kbName: getKbDisplayName(kbId),
      total,
      completed,
      progress: avgProgress,
      hasError
    }
  }).sort((a, b) => a.kbName.localeCompare(b.kbName))
})

const clampProgress = (value: number) => Math.min(100, Math.max(0, Math.round(value)))

const addUploadTask = (task: UploadTaskState) => {
  uploadTasks.value = [
    ...uploadTasks.value.filter(item => item.uploadId !== task.uploadId),
    task,
  ]
}

const patchUploadTask = (uploadId: string, patch: Partial<UploadTaskState>) => {
  const index = uploadTasks.value.findIndex(task => task.uploadId === uploadId)
  if (index === -1) return
  const nextTasks = [...uploadTasks.value]
  nextTasks[index] = { ...nextTasks[index], ...patch }
  uploadTasks.value = nextTasks
}

const removeUploadTask = (uploadId: string) => {
  uploadTasks.value = uploadTasks.value.filter(task => task.uploadId !== uploadId)
  const timer = uploadCleanupTimers.get(uploadId)
  if (timer) {
    clearTimeout(timer)
    uploadCleanupTimers.delete(uploadId)
  }
}

const scheduleUploadTaskCleanup = (uploadId: string) => {
  const existing = uploadCleanupTimers.get(uploadId)
  if (existing) {
    clearTimeout(existing)
  }
  const timer = setTimeout(() => {
    removeUploadTask(uploadId)
  }, UPLOAD_CLEANUP_DELAY)
  uploadCleanupTimers.set(uploadId, timer)
}

type UploadEventDetail = {
  uploadId: string
  kbId?: string | number
  fileName?: string
  progress?: number
  status?: UploadTaskState['status']
  error?: string
}

const ensureUploadTaskEntry = (detail?: UploadEventDetail) => {
  if (!detail?.uploadId) return null
  const existing = uploadTasks.value.find(task => task.uploadId === detail.uploadId)
  if (existing) return existing
  if (!detail.kbId) return null
  const initialProgress = typeof detail.progress === 'number' ? clampProgress(detail.progress) : 0
  const newTask: UploadTaskState = {
    uploadId: detail.uploadId,
    kbId: String(detail.kbId),
    fileName: detail.fileName,
    progress: initialProgress,
    status: detail.status || 'uploading',
    error: detail.error
  }
  addUploadTask(newTask)
  return newTask
}

const handleCardClick = (kb: KB) => {
  // Track this open in the per-user "recent" list before navigating —
  // matches the user mental model "this is what I last worked on".
  pins.touchRecent('kb', kb.id)
  if (isInitialized(kb)) {
    goDetail(kb.id)
  } else {
    goSettings(kb.id)
  }
}

// toggleFavoriteKb is the click handler for the star icon rendered on
// each card. Stops propagation so it doesn't bubble into the card's
// own @click which would open the KB.
const toggleFavoriteKb = (kbId: string, evt?: Event) => {
  evt?.stopPropagation()
  pins.toggleFavorite('kb', kbId)
}
const isKbFavorited = (kbId: string) => pins.isFavorite('kb', kbId)

const goDetail = (id: string) => {
  router.push(`/platform/knowledge-bases/${id}`)
}

const goSettings = (id: string) => {
  // Open settings using a modal
  uiStore.openKBSettings(id)
}

// Create a knowledge base
const handleCreateKnowledgeBase = () => {
  markContextualGuideDone('kbList')
  // Still open the creation wizard when there's no model, and land on the model configuration page; the user can add a model within the wizard without navigating to system settings first
  const initialSection =
    modelsReadyLoaded.value && !isReadyForDocumentKb.value ? 'models' : undefined
  uiStore.openCreateKB('document', initialSection)
}

// Knowledge base editor success callback (fired on create or edit success)
const handleKBEditorSuccess = (kbId: string) => {
  console.log('[KnowledgeBaseList] knowledge operation success:', kbId)
  const shouldOpenDetailForUploadGuide = !isContextualGuideDone('kbDetail')
  // Editing from the list page must also invalidate the single-KB detail cache, otherwise the sidebar / detail page still shows stale info for 60s
  chatResources.invalidateKnowledgeBaseDetail(kbId)
  fetchList(true).then(() => {
    if (shouldOpenDetailForUploadGuide && kbId && !uiStore.showKBEditorModal) {
      goDetail(kbId)
    }
    // If the highlight ID came from a route parameter, trigger the flash effect
    if (route.query.highlightKbId === kbId) {
      triggerHighlightFlash(kbId)
      const { highlightKbId: _drop, ...rest } = route.query
      router.replace({ query: rest })
    }
  })
}

// Trigger the highlight flash effect
const triggerHighlightFlash = (kbId: string) => {
  highlightedKbId.value = kbId
  nextTick(() => {
    if (highlightedCardRef.value) {
      // Scroll to the highlighted card
      highlightedCardRef.value.scrollIntoView({
        behavior: 'smooth',
        block: 'center'
      })
    }
    // Clear the highlight after 3 seconds
    setTimeout(() => {
      highlightedKbId.value = null
    }, 3000)
  })
}

const handleUploadStartEvent = (event: Event) => {
  const detail = (event as CustomEvent<UploadEventDetail>).detail
  if (!detail?.uploadId || !detail?.kbId) return
  addUploadTask({
    uploadId: detail.uploadId,
    kbId: String(detail.kbId),
    fileName: detail.fileName,
    progress: typeof detail.progress === 'number' ? clampProgress(detail.progress) : 0,
    status: 'uploading'
  })
}

const handleUploadProgressEvent = (event: Event) => {
  const detail = (event as CustomEvent<UploadEventDetail>).detail
  if (!detail?.uploadId || typeof detail.progress !== 'number') return
  if (!ensureUploadTaskEntry(detail)) return
  patchUploadTask(detail.uploadId, {
    progress: clampProgress(detail.progress)
  })
}

const handleUploadCompleteEvent = (event: Event) => {
  const detail = (event as CustomEvent<UploadEventDetail>).detail
  if (!detail?.uploadId) return
  const progress = typeof detail.progress === 'number'
    ? clampProgress(detail.progress)
    : 100
  if (!ensureUploadTaskEntry({ ...detail, progress })) return
  patchUploadTask(detail.uploadId, {
    status: detail.status || 'success',
    progress,
    error: detail.error
  })
  scheduleUploadTaskCleanup(detail.uploadId)
}

const handleUploadFinishedEvent = (event: Event) => {
  const detail = (event as CustomEvent<{ kbId?: string | number }>).detail
  if (!detail?.kbId) return
  if (uploadRefreshTimer) {
    clearTimeout(uploadRefreshTimer)
  }
  uploadRefreshTimer = setTimeout(() => {
    fetchList(true)
    uploadRefreshTimer = null
  }, 800)
}
</script>

<style scoped lang="less">
.kb-list-container {
  margin: 0;
  height: 100%;
  box-sizing: border-box;
  flex: 1;
  display: flex;
  position: relative;
  min-height: 0;
}

.kb-list-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  padding: 20px 0 0 28px;
}

.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
  padding-right: 28px;

  .header-title {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .title-row {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  h2 {
    margin: 0;
    color: var(--td-text-color-primary);
    font-family: var(--app-font-family);
    font-size: 24px;
    font-weight: 600;
    line-height: 32px;
  }

}

.kb-create-btn {
  background: linear-gradient(135deg, var(--td-brand-color) 0%, #00a67e 100%);
  border: none;
  color: var(--td-text-color-anti);

  &:hover {
    background: linear-gradient(135deg, var(--td-brand-color) 0%, var(--td-brand-color-active) 100%);
  }
}

.kb-list-main {
  flex: 1;
  min-width: 0;
  overflow-y: auto;
  overflow-x: hidden;
  // No top padding, so the sticky group header (top: 0) can sit flush against the very top of the container;
  // Bottom padding is kept to avoid the last row of cards touching the edge.
  padding: 0 28px 8px 0;
  scrollbar-width: auto;
  scrollbar-color: auto;
}

.kb-list-main-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 200px;
  padding: 12px;
  background: var(--td-bg-color-container);
}

.shared-by-me-badge {
  display: inline-flex;
  align-items: center;
  padding: 2px 6px;
  background: rgba(7, 192, 95, 0.1);
  border-radius: 4px;
  font-size: 12px;
  color: var(--td-brand-color);
  margin-left: 6px;
}

.header-subtitle {
  margin: 0;
  color: var(--td-text-color-placeholder);
  font-family: var(--app-font-family);
  font-size: 14px;
  font-weight: 400;
  line-height: 20px;
}

.header-action-btn {
  padding: 0 !important;
  min-width: 28px !important;
  width: 28px !important;
  height: 28px !important;
  display: inline-flex !important;
  align-items: center !important;
  justify-content: center !important;
  background: var(--td-bg-color-secondarycontainer) !important;
  border: 1px solid var(--td-component-stroke) !important;
  border-radius: 6px !important;
  color: var(--td-text-color-secondary);
  cursor: pointer;
  box-shadow: inset 0 1px 0 color-mix(in srgb, var(--td-bg-color-container) 72%, transparent);
  transition: background 0.2s, border-color 0.2s, color 0.2s;

  &:hover {
    background: var(--td-bg-color-secondarycontainer) !important;
    border-color: var(--td-component-stroke) !important;
    color: var(--td-text-color-primary);
  }

  :deep(.t-icon),
  :deep(.btn-icon-wrapper) {
    color: var(--td-brand-color);
  }
}

// Tab switch styling (superseded by the left-side menu, kept for compatibility)
.kb-tabs {
  display: flex;
  align-items: center;
  gap: 24px;
  border-bottom: 1px solid var(--td-component-stroke);
  margin-bottom: 20px;

  .tab-item {
    padding: 12px 0;
    cursor: pointer;
    color: var(--td-text-color-secondary);
    font-family: var(--app-font-family);
    font-size: 14px;
    font-weight: 400;
    user-select: none;
    position: relative;
    transition: color 0.2s ease;

    &:hover {
      color: var(--td-text-color-primary);
    }

    &.active {
      color: var(--td-brand-color);
      font-weight: 500;

      &::after {
        content: '';
        position: absolute;
        bottom: -1px;
        left: 0;
        right: 0;
        height: 2px;
        background: var(--td-brand-color);
        border-radius: 1px;
      }
    }
  }
}


// Shared knowledge base card styling
// Share badge (green by default for document type, positioned top-right)
.shared-badge {
  position: absolute;
  top: 8px;
  right: 14px;
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 2px 8px;
  background: rgba(7, 192, 95, 0.1);
  border-radius: 4px;
  font-size: 12px;
  color: var(--td-brand-color);
  font-weight: 500;

  .t-icon {
    color: var(--td-brand-color);
  }
}

// Source organization (space icon + space name)
.org-source {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 3px 8px;
  background: rgba(7, 192, 95, 0.06);
  border-radius: 6px;
  font-size: 12px;
  line-height: 1.4;
  color: var(--td-text-color-secondary);
  max-width: 140px;
  transition: background-color 0.15s ease;

  span {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-weight: 500;
  }

  .org-source-icon {
    width: 14px;
    height: 14px;
    flex-shrink: 0;
    vertical-align: middle;
  }

  .t-icon {
    color: var(--td-brand-color);
    flex-shrink: 0;
  }
}

// "Mine" knowledge base tag (same style set as .org-source: gray text + green tag + light green background)
.personal-source {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 3px 8px;
  background: rgba(7, 192, 95, 0.06);
  border-radius: 6px;
  font-size: 11px;
  line-height: 1.4;
  color: var(--td-text-color-secondary);
  font-weight: 500;
  transition: background-color 0.15s ease;

  span {
    font-weight: 500;
  }

  .t-icon {
    color: var(--td-brand-color);
    flex-shrink: 0;
  }
}

.shared-kb-card {
  position: relative;

  // Shared knowledge bases display different styles depending on type
  &.kb-type-document {
    background: linear-gradient(135deg, var(--td-bg-color-container) 0%, rgba(7, 192, 95, 0.04) 100%) !important;

    &:hover {
      border-color: var(--td-brand-color) !important;
      box-shadow: 0 4px 12px rgba(7, 192, 95, 0.12) !important;
      background: linear-gradient(135deg, var(--td-bg-color-container) 0%, rgba(7, 192, 95, 0.08) 100%) !important;
    }

    &::after {
      background: linear-gradient(135deg, rgba(7, 192, 95, 0.08) 0%, transparent 100%) !important;
    }
  }

  &.kb-type-faq {
    background: linear-gradient(135deg, var(--td-bg-color-container) 0%, rgba(0, 82, 217, 0.04) 100%) !important;

    &:hover {
      border-color: var(--td-brand-color) !important;
      box-shadow: 0 4px 12px rgba(0, 82, 217, 0.12) !important;
      background: linear-gradient(135deg, var(--td-bg-color-container) 0%, rgba(0, 82, 217, 0.08) 100%) !important;
    }

    &::after {
      background: linear-gradient(135deg, rgba(0, 82, 217, 0.08) 0%, transparent 100%) !important;
    }

    // FAQ-type share badge uses blue
    .shared-badge {
      background: rgba(0, 82, 217, 0.1);
      color: var(--td-brand-color);

      .t-icon {
        color: var(--td-brand-color);
      }
    }
  }

  .org-tag {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    font-size: 12px;
    border-color: rgba(0, 82, 217, 0.15);
    color: var(--td-brand-color);
    background: rgba(0, 82, 217, 0.04);
    font-weight: 500;
    padding: 2px 8px;
    border-radius: 4px;
    max-width: fit-content;
  }
}


.warning-banner {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  margin-bottom: 20px;
  background: var(--td-warning-color-light);
  border: 1px solid var(--td-warning-color-focus);
  border-radius: 6px;
  color: var(--td-warning-color);
  font-family: var(--app-font-family);
  font-size: 14px;

  .t-icon {
    color: var(--td-warning-color);
    flex-shrink: 0;
  }
}

.upload-progress-panel {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-bottom: 20px;
}

.upload-progress-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  background: var(--td-bg-color-container);
}

.upload-progress-icon {
  color: var(--td-brand-color);
  display: flex;
  align-items: center;
  justify-content: center;
}

.upload-progress-content {
  flex: 1;
}

.progress-title {
  color: var(--td-text-color-primary);
  font-family: var(--app-font-family);
  font-size: 14px;
  font-weight: 600;
  line-height: 22px;
  margin-bottom: 2px;
}

.progress-subtitle {
  color: var(--td-text-color-secondary);
  font-family: var(--app-font-family);
  font-size: 12px;
  line-height: 18px;
}

.progress-subtitle.secondary {
  color: var(--td-text-color-placeholder);
  margin-top: 2px;
}

.progress-subtitle.error {
  color: var(--td-error-color);
  margin-top: 4px;
}

.progress-bar {
  width: 100%;
  height: 6px;
  border-radius: 999px;
  background: var(--td-bg-color-secondarycontainer);
  margin-top: 10px;
  overflow: hidden;
}

.progress-bar-inner {
  height: 100%;
  background: linear-gradient(90deg, var(--td-brand-color-active) 0%, var(--td-brand-color) 100%);
  transition: width 0.2s ease;
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

.kb-card-wrap {
  display: grid;
  gap: 12px;
  grid-template-columns: 1fr;
  animation: contentFadeIn 0.32s ease-out;
}

.kb-section-header {
  grid-column: 1 / -1;
  display: flex;
  align-items: center;
  gap: 6px;
  // The whole row is only used to fill the background for the sticky effect; the click event is triggered via bubbling from child elements, to avoid accidental collapse
  // when clicking large empty space to the right of the title. Keyboard tab/enter is unaffected by pointer-events.
  pointer-events: none;

  & > * {
    pointer-events: auto;
  }
  // Sticks to the top of the scroll container (.kb-list-main) when scrolling down. z-index must be higher than the card's own
  // hover shadow / decoration layer; the background must be opaque, otherwise cards would show through from below.
  position: sticky;
  top: 0;
  z-index: 5;
  background: var(--td-bg-color-container);
  // Use box-shadow to "extend" the background upward by another 8px, sealing off any gap between the sticky element and the top of the container
  // subpixel gap (border-radius rounded corner triangles, browser subpixel rendering during scroll, etc.
  // all cause the card to leak 1-2px here). The second shadow adds a bit more below, to avoid
  // cards spilling into the grid-gap area.
  box-shadow: 0 -8px 0 0 var(--td-bg-color-container),
    0 4px 0 0 var(--td-bg-color-container);
  padding: 6px 4px 6px 0;
  color: var(--td-text-color-secondary);
  font-family: var(--app-font-family);
  font-size: 13px;
  font-weight: 600;
  line-height: 20px;
  cursor: pointer;
  user-select: none;
  outline: none;

  &:hover {
    color: var(--td-text-color-primary);
  }

  &:focus-visible {
    box-shadow: 0 0 0 2px var(--td-brand-color-focus, rgba(0, 82, 217, 0.2));
  }

  // Icons inherit the section header's text color so the whole row
  // (icon + label) reads as one muted secondary tone. The pinned
  // modifier no longer overrides this either — uniform appearance
  // is intentional; the icon shape alone is enough to flag which
  // section the user is looking at.
  .t-icon {
    color: inherit;
  }

  .kb-section-toggle {
    margin-left: 4px;
    opacity: 0.7;
    transition: opacity 0.15s ease;
  }

  // the two subgroups shared with me use one shared main icon usergroup-add, plus a sub-icon
  // (edit / browse) to distinguish permissions. The sub-icon hugs the main icon on the left, so it still reads as one "group" overall.
  .kb-section-subicon {
    margin-left: -4px;
    opacity: 0.75;
  }

  // how many cards are actually in the group. Use 13px main font size, same color, reduced opacity, to avoid competing visually with the title,
  // while also giving a light background so it stays readable on light containers.
  .kb-section-count {
    margin-left: 2px;
    padding: 0 6px;
    border-radius: 8px;
    background: var(--td-bg-color-secondarycontainer);
    color: var(--td-text-color-secondary);
    font-size: 11px;
    line-height: 16px;
    font-weight: 500;
  }

  &:hover .kb-section-toggle {
    opacity: 1;
  }
}

.kb-card {
  border: 1px solid var(--td-component-stroke);
  border-radius: 8px;
  overflow: hidden;
  box-sizing: border-box;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
  background: var(--td-bg-color-container);
  position: relative;
  cursor: pointer;
  transition: all 0.25s ease;
  padding: 12px 14px;
  display: flex;
  flex-direction: column;
  height: 136px;
  min-height: 136px;

  &.kb-card-skeleton {
    cursor: default;

    .card-header {
      margin-bottom: 12px;
    }

    .card-content {
      flex: 1;
    }

    .card-bottom {
      margin-top: auto;
    }
  }

  &:hover {
    border-color: var(--td-brand-color);
    box-shadow: 0 4px 12px rgba(7, 192, 95, 0.12);
  }

  &.uninitialized {
    opacity: 0.9;
  }

  // document type style
  &.kb-type-document {
    background: linear-gradient(135deg, var(--td-bg-color-container) 0%, rgba(7, 192, 95, 0.04) 100%);

    &:hover {
      border-color: var(--td-brand-color);
      background: linear-gradient(135deg, var(--td-bg-color-container) 0%, rgba(7, 192, 95, 0.08) 100%);
    }

    // top-right decoration
    &::after {
      content: '';
      position: absolute;
      top: 0;
      right: 0;
      width: 60px;
      height: 60px;
      background: linear-gradient(135deg, rgba(7, 192, 95, 0.08) 0%, transparent 100%);
      border-radius: 0 12px 0 100%;
      pointer-events: none;
      z-index: 0;
    }
  }

  // Q&A type style
  &.kb-type-faq {
    background: linear-gradient(135deg, var(--td-bg-color-container) 0%, rgba(0, 82, 217, 0.04) 100%);

    &:hover {
      border-color: var(--td-brand-color);
      box-shadow: 0 4px 12px rgba(0, 82, 217, 0.12);
      background: linear-gradient(135deg, var(--td-bg-color-container) 0%, rgba(0, 82, 217, 0.08) 100%);
    }

    // top-right decoration
    &::after {
      content: '';
      position: absolute;
      top: 0;
      right: 0;
      width: 60px;
      height: 60px;
      background: linear-gradient(135deg, rgba(0, 82, 217, 0.08) 0%, transparent 100%);
      border-radius: 0 12px 0 100%;
      pointer-events: none;
      z-index: 0;
    }
  }

  .kb-favorite-star {
    // floats at the card's top-right corner. The card itself has padding, so the "more" button sits at the end of the header flex
    // and naturally falls within the padding, offset a bit from the star at the zero position.
    position: absolute;
    top: 0;
    right: 0;
    z-index: 3;
    width: 24px;
    height: 24px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: transparent;
    border: none;
    border-radius: 6px;
    color: var(--td-text-color-secondary);
    cursor: pointer;
    opacity: 0;
    transition: opacity 0.15s ease, background 0.15s ease, color 0.15s ease;

    &:hover {
      background: var(--td-bg-color-secondarycontainer);
      color: var(--td-warning-color, #e37318);
    }

    &.is-favorited {
      opacity: 1;
      color: var(--td-warning-color, #e37318);
    }
  }

  // Reveal the star on card hover; favorited state forces it visible.
  &:hover .kb-favorite-star {
    opacity: 1;
  }

  // ensure content stays above the decoration
  .card-header,
  .card-content,
  .card-bottom {
    position: relative;
    z-index: 1;
  }

  .card-header {
    margin-bottom: 6px;
  }

  .card-title {
    font-size: 15px;
    line-height: 22px;
  }

  .card-content {
    margin-bottom: 6px;
  }

  .card-description {
    font-size: 12px;
    line-height: 17px;
  }

  .card-bottom {
    padding-top: 6px;
  }

  .more-wrap {
    width: 28px;
    height: 28px;

    .more-icon {
      width: 16px;
      height: 16px;
    }
  }

  .card-more-btn {
    width: 28px;
    height: 28px;
  }
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 4px;
  margin-bottom: 6px;

  .card-title {
    flex: 1;
    font-size: 15px;
    font-weight: 600;
    color: var(--td-text-color-primary);
    letter-spacing: 0.01em;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    display: flex;
    align-items: center;
    gap: 6px;
    min-width: 0;
  }

  .card-title-text {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .card-more-btn {
    flex-shrink: 0;
    width: 24px;
    height: 24px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 6px;
    color: var(--td-text-color-placeholder);
    cursor: pointer;
    transition: all 0.2s;

    &:hover {
      background: var(--td-bg-color-container-hover);
      color: var(--td-text-color-secondary);
    }
  }

  .permission-tag {
    flex-shrink: 0;
  }
}

.card-title {
  color: var(--td-text-color-primary);
  font-family: var(--app-font-family);
  font-size: 15px;
  font-weight: 600;
  line-height: 22px;
  letter-spacing: 0.01em;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
  min-width: 0;
}

.more-wrap {
  display: flex;
  width: 24px;
  height: 24px;
  justify-content: center;
  align-items: center;
  border-radius: 6px;
  cursor: pointer;
  flex-shrink: 0;
  transition: all 0.2s ease;
  opacity: 0;

  .kb-card:hover & {
    opacity: 0.6;
  }

  &:hover {
    background: var(--td-bg-color-container-hover);
    opacity: 1 !important;
  }

  &.active-more {
    background: var(--td-bg-color-container-hover);
    opacity: 1 !important;
  }

  .more-icon {
    width: 14px;
    height: 14px;
  }
}

.card-content {
  flex: 1;
  min-height: 0;
  margin-bottom: 8px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

/* unify the three list cards: description font */
.card-description {
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  overflow: hidden;
  color: var(--td-text-color-secondary);
  font-family: var(--app-font-family);
  font-size: 12px;
  font-weight: 400;
  line-height: 18px;
}

.card-bottom {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: auto;
  padding-top: 8px;
  border-top: .5px solid var(--td-component-stroke);
}

.bottom-left {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  min-width: 0;
}

.bottom-right {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;

  .card-time {
    font-size: 12px;
    color: var(--td-text-color-placeholder);
  }
}

.feature-badges {
  display: flex;
  align-items: center;
  gap: 4px;
}

.feature-badge {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border-radius: 5px;
  cursor: default;
  transition: background 0.2s ease;

  &.type-document {
    background: rgba(7, 192, 95, 0.08);
    color: var(--td-brand-color-active);
    width: auto;
    padding: 0 6px;
    gap: 3px;

    &:hover {
      background: rgba(7, 192, 95, 0.12);
    }

    .badge-count {
      font-size: 11px;
      font-weight: 500;
    }

    .processing-icon {
      animation: spin 1s linear infinite;
    }
  }

  &.type-faq {
    background: rgba(0, 82, 217, 0.08);
    color: var(--td-brand-color);
    width: auto;
    padding: 0 6px;
    gap: 3px;

    &:hover {
      background: rgba(0, 82, 217, 0.12);
    }

    .badge-count {
      font-size: 11px;
      font-weight: 500;
    }

    .processing-icon {
      animation: spin 1s linear infinite;
    }
  }

  &.kg {
    background: rgba(124, 77, 255, 0.08);
    color: var(--td-brand-color);

    &:hover {
      background: rgba(124, 77, 255, 0.12);
    }
  }

  &.multimodal {
    background: rgba(255, 152, 0, 0.08);
    color: var(--td-warning-color);

    &:hover {
      background: rgba(255, 152, 0, 0.12);
    }
  }

  &.question {
    background: rgba(0, 150, 136, 0.08);
    color: var(--td-success-color);

    &:hover {
      background: rgba(0, 150, 136, 0.12);
    }
  }

  &.shared {
    background: rgba(0, 82, 217, 0.08);
    color: var(--td-brand-color);

    &:hover {
      background: rgba(0, 82, 217, 0.12);
    }
  }

  &.role-admin {
    background: rgba(7, 192, 95, 0.1);
    color: var(--td-brand-color-active);

    &:hover {
      background: rgba(7, 192, 95, 0.15);
    }
  }

  &.role-editor {
    background: rgba(255, 152, 0, 0.1);
    color: var(--td-warning-color);

    &:hover {
      background: rgba(255, 152, 0, 0.15);
    }
  }

  &.role-viewer {
    background: var(--td-bg-color-container-hover);
    color: var(--td-text-color-secondary);

    &:hover {
      background: rgba(0, 0, 0, 0.08);
    }
  }
}

@keyframes spin {
  from {
    transform: rotate(0deg);
  }

  to {
    transform: rotate(360deg);
  }
}

@keyframes highlightFlash {
  0% {
    border-color: var(--td-brand-color);
    box-shadow: 0 0 0 0 rgba(7, 192, 95, 0.4);
    transform: scale(1);
  }

  50% {
    border-color: var(--td-brand-color);
    box-shadow: 0 0 0 8px rgba(7, 192, 95, 0);
    transform: scale(1.02);
  }

  100% {
    border-color: var(--td-brand-color);
    box-shadow: 0 0 0 0 rgba(7, 192, 95, 0);
    transform: scale(1);
  }
}

.kb-card.highlight-flash {
  animation: highlightFlash 0.6s ease-in-out 3;
  border-color: var(--td-brand-color) !important;
  box-shadow: 0 0 12px rgba(7, 192, 95, 0.3) !important;
}

.card-time {
  color: var(--td-text-color-placeholder);
  font-family: var(--app-font-family);
  font-size: 12px;
  font-weight: 400;
}


.empty-state {
  flex: 1;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  padding: 60px 20px;

  .empty-img {
    width: 162px;
    height: 162px;
    margin-bottom: 20px;
  }

  .empty-txt {
    color: var(--td-text-color-placeholder);
    font-family: var(--app-font-family);
    font-size: 16px;
    font-weight: 600;
    line-height: 26px;
    margin-bottom: 8px;
  }

  .empty-desc {
    color: var(--td-text-color-disabled);
    font-family: var(--app-font-family);
    font-size: 14px;
    font-weight: 400;
    line-height: 22px;
    margin-bottom: 0;
  }

  .empty-state-btn {
    margin-top: 20px;
  }
}

// responsive layout
@media (min-width: 900px) {
  .kb-card-wrap {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (min-width: 1250px) {
  .kb-card-wrap {
    grid-template-columns: repeat(3, 1fr);
  }
}

@media (min-width: 1600px) {
  .kb-card-wrap {
    grid-template-columns: repeat(4, 1fr);
  }
}

@media (min-width: 1900px) {
  .kb-card-wrap {
    grid-template-columns: repeat(5, 1fr);
  }
}

@media (min-width: 2200px) {
  .kb-card-wrap {
    grid-template-columns: repeat(6, 1fr);
  }
}

// delete confirmation dialog style
:deep(.del-knowledge-dialog) {
  padding: 0px !important;
  border-radius: 6px !important;

  .t-dialog__header {
    display: none;
  }

  .t-dialog__body {
    padding: 16px;
  }

  .t-dialog__footer {
    padding: 0;
  }
}

:deep(.t-dialog__position.t-dialog--top) {
  padding-top: 40vh !important;
}

.circle-wrap {
  .dialog-header {
    display: flex;
    align-items: center;
    margin-bottom: 8px;
  }

  .circle-img {
    width: 20px;
    height: 20px;
    margin-right: 8px;
  }

  .circle-title {
    color: var(--td-text-color-primary);
    font-family: var(--app-font-family);
    font-size: 16px;
    font-weight: 600;
    line-height: 24px;
  }

  .del-circle-txt {
    color: var(--td-text-color-placeholder);
    font-family: var(--app-font-family);
    font-size: 14px;
    font-weight: 400;
    line-height: 22px;
    display: inline-block;
    margin-left: 29px;
    margin-bottom: 21px;
  }

  .circle-btn {
    height: 22px;
    width: 100%;
    display: flex;
    justify-content: flex-end;
  }

  .circle-btn-txt {
    color: var(--td-text-color-primary);
    font-family: var(--app-font-family);
    font-size: 14px;
    font-weight: 400;
    line-height: 22px;
    cursor: pointer;

    &:hover {
      opacity: 0.8;
    }
  }

  .confirm {
    color: var(--td-error-color);
    margin-left: 40px;

    &:hover {
      opacity: 0.8;
    }
  }
}
</style>

<style lang="less">
/* dropdown menu styles have been unified into @/assets/dropdown-menu.less */

// shared knowledge base card: details trigger (replaces the three dots, uses a "view details" link style)
.shared-detail-trigger {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 8px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--td-brand-color);
  font-size: 13px;
  font-family: var(--app-font-family);
  cursor: pointer;
  transition: background 0.2s ease, color 0.2s ease;

  .t-icon {
    flex-shrink: 0;
  }

  &:hover {
    background: rgba(7, 192, 95, 0.08);
    color: var(--td-brand-color);
  }
}

// right-side slide-out: shared knowledge base details panel
.shared-detail-drawer-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.4);
  z-index: 1000;
  display: flex;
  justify-content: flex-end;
}

.shared-detail-drawer {
  width: 360px;
  max-width: 90vw;
  height: 100%;
  background: var(--td-bg-color-container);
  box-shadow: -4px 0 24px rgba(0, 0, 0, 0.12);
  display: flex;
  flex-direction: column;
  font-family: var(--app-font-family);
}

.shared-detail-drawer-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 20px 24px;
  border-bottom: 1px solid var(--td-component-stroke);
  flex-shrink: 0;
}

.shared-detail-drawer-title {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--td-text-color-primary);
}

.shared-detail-drawer-close {
  width: 32px;
  height: 32px;
  border: none;
  border-radius: 6px;
  background: var(--td-bg-color-secondarycontainer);
  color: var(--td-text-color-secondary);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.2s ease, color 0.2s ease;

  &:hover {
    background: var(--td-bg-color-secondarycontainer);
    color: var(--td-text-color-primary);
  }
}

.shared-detail-drawer-body {
  flex: 1;
  overflow-y: auto;
  padding: 24px;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.shared-detail-drawer-body .shared-detail-row {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.shared-detail-drawer-body .shared-detail-label {
  font-size: 12px;
  color: var(--td-text-color-secondary);
  line-height: 1.4;
}

.shared-detail-drawer-body .shared-detail-value {
  font-size: 14px;
  color: var(--td-text-color-primary);
  line-height: 1.5;
  word-break: break-word;

  &.shared-detail-source-type {
    font-weight: 500;
    color: var(--td-text-color-primary);
  }

  &.shared-detail-org {
    display: inline-flex;
    align-items: center;
    gap: 6px;
  }
}

.shared-detail-drawer-body .shared-detail-org-icon {
  width: 14px;
  height: 14px;
  flex-shrink: 0;
}

.shared-detail-drawer-footer {
  padding: 16px 24px;
  border-top: 1px solid var(--td-component-stroke);
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  flex-shrink: 0;
  background: var(--td-bg-color-container);

  .go-to-kb-btn .t-button__text {
    display: inline-flex;
    align-items: center;
    gap: 4px;
  }
}

// right-side slide-in animation
.shared-detail-drawer-enter-active,
.shared-detail-drawer-leave-active {
  transition: opacity 0.25s ease;

  .shared-detail-drawer {
    transition: transform 0.25s ease;
  }
}

.shared-detail-drawer-enter-from,
.shared-detail-drawer-leave-to {
  opacity: 0;

  .shared-detail-drawer {
    transform: translateX(100%);
  }
}

// create dialog style improvements
.create-kb-dialog {
  .t-form-item__label {
    font-family: var(--app-font-family);
    font-size: 14px;
    font-weight: 500;
    color: var(--td-text-color-primary);
  }

  .t-input,
  .t-textarea {
    font-family: var(--app-font-family);
  }

}
</style>
