<template>
    <div class="main" ref="dropzone" :style="{ '--sidebar-width': `${uiStore.sidebarDisplayWidth}px` }">
        <Menu></Menu>
        <div v-if="isRouterAlive" class="platform-route-outlet">
            <RouterView />
        </div>
        <div class="upload-mask" v-show="ismask">
            <UploadMask></UploadMask>
        </div>
        <!-- Global settings modal, used by all platform sub-routes -->
        <Settings />
        <!-- Global command palette (⌘K), persists across platform routes -->
        <GlobalCommandPalette />
        <!-- Global top-right "pending invitations" bell. Fixed position, z-index below the drawer; business pages
             are naturally covered when the right drawer opens; only rendered when there are pending invitations. -->
        <GlobalInvitationBell />
        <!-- Knowledge base file upload progress overlay: the upload queue lives in the store, so switching pages doesn't interrupt it -->
        <UploadTasksPanel />
        <!-- Guided onboarding with overlay: auto-starts on first visit, can be reopened from the help button next to the nickname at the top of the user menu -->
        <NewUserGuide />
    </div>
</template>
<script setup lang="ts">
import Menu from '@/components/menu.vue'
import { ref, onMounted, onUnmounted, nextTick, provide, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router'
import UploadMask from '@/components/upload-mask.vue'
import Settings from '@/views/settings/Settings.vue'
import GlobalCommandPalette from '@/components/GlobalCommandPalette.vue'
import GlobalInvitationBell from '@/components/GlobalInvitationBell.vue'
import UploadTasksPanel from '@/components/upload-tasks/UploadTasksPanel.vue'
import NewUserGuide from '@/components/NewUserGuide.vue'
import { useCommandPaletteStore } from '@/stores/commandPalette'
import { useChatResourcesStore } from '@/stores/chatResources'
import { useUIStore } from '@/stores/ui'
import { getKnowledgeBaseById } from '@/api/knowledge-base/index'
import { MessagePlugin } from 'tdesign-vue-next'
import { useI18n } from 'vue-i18n'
import { collectDroppedFiles } from './collectDroppedFiles'

const route = useRoute();
const router = useRouter();
const commandPaletteStore = useCommandPaletteStore();
const uiStore = useUIStore();
let ismask = ref(false)
const { t } = useI18n();

const isRouterAlive = ref(true)
const reloadApp = () => {
    isRouterAlive.value = false
    nextTick(() => {
        isRouterAlive.value = true
    })
}
provide('app:reload', reloadApp)

// Only intercept Cmd/Ctrl+R when running in the Wails desktop client:
// The desktop client has no browser address bar, so a full page reload would show a blank screen — use a frontend soft refresh instead.
// Don't intercept in browsers (including the Web version / non-Lite deployments); let the browser do a real full-page refresh,
// otherwise the left menu, global settings, Pinia store, etc. won't reset along with the "refresh".
// @ts-ignore
const isWailsDesktop = typeof window !== 'undefined' && !!(window as any).runtime?.EventsOn

const handleGlobalKeyDown = (e: KeyboardEvent) => {
    if (!isWailsDesktop) return
    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'r') {
        e.preventDefault()
        reloadApp()
    }
}

// Counter for tracking drag enter/leave, to work around child elements triggering dragleave
let dragCounter = 0;

// Get the current knowledge base ID
const getCurrentKbId = (): string | null => {
    return (route.params as any)?.kbId as string || null
}

const CHAT_DROP_ROUTE_NAMES = new Set(['chat', 'globalCreatChat', 'kbCreatChat']);

const isChatDropRoute = () => {
    return CHAT_DROP_ROUTE_NAMES.has(String(route.name || ''));
}

// Check knowledge base initialization status
const checkKnowledgeBaseInitialization = async (): Promise<boolean> => {
    const currentKbId = getCurrentKbId();
    
    if (!currentKbId) {
        MessagePlugin.error(t('knowledgeBase.missingId'));
        return false;
    }
    
    try {
        const kbResponse = await getKnowledgeBaseById(currentKbId);
        const kb = kbResponse.data;
        
        if (!kb.summary_model_id) {
            MessagePlugin.warning(t('knowledgeBase.notInitialized'));
            return false;
        }
        const strategy = kb.indexing_strategy;
        const needsEmbedding = !strategy || strategy.vector_enabled || strategy.keyword_enabled;
        if (needsEmbedding && !kb.embedding_model_id) {
            MessagePlugin.warning(t('knowledgeBase.notInitialized'));
            return false;
        }
        return true;
    } catch (error) {
        MessagePlugin.error(t('knowledgeBase.getInfoFailed'));
        return false;
    }
}


// isFileDrag distinguishes an OS file drag (the only thing the global upload
// drop zone cares about) from an in-app element drag such as the wiki
// folder/page drag-and-drop. Element drags carry only "text/*" types, never
// "Files", so we bail out and let the originating component handle the drop.
const isFileDrag = (event: DragEvent): boolean => {
    const types = event.dataTransfer?.types
    if (!types) return false
    return Array.from(types).includes('Files')
}

const shouldHandleGlobalFileDrag = (event: DragEvent): boolean => {
    if (!isFileDrag(event)) return false;
    // Keep the browser from opening dropped files, even outside upload pages.
    event.preventDefault();
    // Settings and its teleported skill drawers own their uploads. This runs
    // in document capture, before a local drop handler can stop propagation.
    const enabled = !uiStore.showSettingsModal && (
        isChatDropRoute() || (route.name === 'knowledgeBaseDetail' && !!getCurrentKbId())
    );
    if (!enabled) {
        dragCounter = 0;
        ismask.value = false;
    }
    return enabled;
}

// Global drag event handling
const handleGlobalDragEnter = (event: DragEvent) => {
    if (!shouldHandleGlobalFileDrag(event)) return;
    dragCounter++;
    if (event.dataTransfer) {
        event.dataTransfer.effectAllowed = 'all';
    }
    ismask.value = true;
}

const handleGlobalDragOver = (event: DragEvent) => {
    if (!shouldHandleGlobalFileDrag(event)) return;
    if (event.dataTransfer) {
        event.dataTransfer.dropEffect = 'copy';
    }
}

const handleGlobalDragLeave = (event: DragEvent) => {
    if (!shouldHandleGlobalFileDrag(event)) return;
    dragCounter--;
    if (dragCounter === 0) {
        ismask.value = false;
    }
}

const handleGlobalDrop = async (event: DragEvent) => {
    if (!shouldHandleGlobalFileDrag(event)) return;
    dragCounter = 0;
    ismask.value = false;

    const droppedFiles = await collectDroppedFiles(event);
    if (droppedFiles.length === 0) {
        MessagePlugin.warning(t('knowledgeBase.dragFileNotText'));
        return;
    }

    if (isChatDropRoute()) {
        event.stopPropagation();
        window.dispatchEvent(new CustomEvent('weknora:chat-file-drop', {
            detail: { files: droppedFiles }
        }));
        return;
    }
    
    const isInitialized = await checkKnowledgeBaseInitialization();
    if (!isInitialized) {
        return;
    }

    window.dispatchEvent(new CustomEvent('weknora:knowledge-file-drop', {
        detail: { kbId: getCurrentKbId(), files: droppedFiles }
    }));
}

// Add global event listeners on component mount
onMounted(() => {
    document.addEventListener('dragenter', handleGlobalDragEnter, true);
    document.addEventListener('dragover', handleGlobalDragOver, true);
    document.addEventListener('dragleave', handleGlobalDragLeave, true);
    document.addEventListener('drop', handleGlobalDrop, true);
    if (isWailsDesktop) {
        window.addEventListener('keydown', handleGlobalKeyDown);
        // @ts-ignore
        window.runtime.EventsOn('app:reload', () => {
            reloadApp()
        })
    }
    // Support opening the global command palette via a URL query parameter, e.g. the old path
    // /platform/knowledge-search?q=foo carries ?cmdk=foo after redirect
    maybeOpenCmdkFromRoute()
    // Prefetch chat input bar resources in the background, reused from cache when entering creatChat / chat
    void useChatResourcesStore().prefetchChatInput()
});

// Watch for route changes, to handle the ?cmdk= parameter on SPA-internal navigations
watch(() => route.query.cmdk, () => {
    maybeOpenCmdkFromRoute()
})

function maybeOpenCmdkFromRoute() {
    if (!('cmdk' in route.query)) return
    const q = String(route.query.cmdk ?? '')
    commandPaletteStore.openPalette(q)
    // Clear the query to avoid retriggering on back/refresh
    const newQuery = { ...route.query }
    delete (newQuery as any).cmdk
    router.replace({ path: route.path, query: newQuery, hash: route.hash })
}

// Remove global event listeners on component unmount
onUnmounted(() => {
    document.removeEventListener('dragenter', handleGlobalDragEnter, true);
    document.removeEventListener('dragover', handleGlobalDragOver, true);
    document.removeEventListener('dragleave', handleGlobalDragLeave, true);
    document.removeEventListener('drop', handleGlobalDrop, true);
    if (isWailsDesktop) {
        window.removeEventListener('keydown', handleGlobalKeyDown);
        // @ts-ignore
        if (window.runtime?.EventsOff) {
            // @ts-ignore
            window.runtime.EventsOff('app:reload')
        }
    }
    dragCounter = 0;
});
</script>
<style lang="less">
.main {
    display: flex;
    align-items: stretch;
    width: 100%;
    height: 100%;
    min-width: 600px;
    min-height: 0;
    /* Unified full-page background, for visual continuity between the left menu and right content area */
    background: var(--td-bg-color-container);
}

/* Right-side route area: fills the remaining width and full column height, and passes min-height:0 down to child pages for internal flex scrolling */
.platform-route-outlet {
    flex: 1;
    min-width: 0;
    min-height: 0;
    display: flex;
    flex-direction: column;
    overflow: hidden;
}

.upload-mask {
    background-color: rgba(255, 255, 255, 0.8);
    position: fixed;
    width: 100%;
    height: 100%;
    z-index: 999;
    display: flex;
    justify-content: center;
    align-items: center;
}

img {
    -webkit-user-drag: none;
    -khtml-user-drag: none;
    -moz-user-drag: none;
    -o-user-drag: none;
    user-drag: none;
}
</style>
