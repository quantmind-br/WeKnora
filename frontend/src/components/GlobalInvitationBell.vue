<template>
  <!-- Global top-right "pending invitations" bell.
       - Same logic as the bell in the original UserMenu: only renders when pendingInvitationCount > 0,
         An empty inbox scenario doesn't take up corner pixels.
       - Fixed position, z-index far below t-drawer's default 2500; business page right-side drawers (FAQ, KB debug,
         Tenant audit, SettingDrawer, etc.) will naturally cover the bell when they pop up, no need to hide it explicitly.
       - Clicking the bell reuses the same MyInvitationsDialog, behavior stays the same as before. -->
  <template v-if="pendingInvitationCount > 0">
    <t-badge :count="pendingInvitationCount" :max-count="99" :offset="[6, 4]"
      class="global-invitation-bell">
      <button type="button" class="global-invitation-bell__btn"
        :title="$t('tenantInvitation.inboxTooltip')" @click="openDialog">
        <t-icon name="notification" size="18px" />
      </button>
    </t-badge>
  </template>
  <MyInvitationsDialog v-model:visible="dialogVisible" />
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useAuthStore } from '@/stores/auth'
import MyInvitationsDialog from '@/components/MyInvitationsDialog.vue'

const authStore = useAuthStore()

const pendingInvitationCount = computed(() => authStore.pendingInvitationCount)

const dialogVisible = ref(false)
const openDialog = () => {
  dialogVisible.value = true
}
</script>

<style lang="less" scoped>
.global-invitation-bell {
  position: fixed;
  top: 12px;
  right: 16px;
  /* Far below TDesign drawer's default z-index (2500), ensuring business page right-side drawers cover the bell properly when they pop up.
     Higher than normal page content (usually 0~10), avoiding being covered by list cards. */
  z-index: 100;
}

.global-invitation-bell__btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  padding: 0;
  border: 1px solid var(--td-component-stroke);
  border-radius: var(--app-radius-lg);
  /* Use the container background instead of transparent, so the bell stays visible against different page background colors when floating over content. */
  background: var(--td-bg-color-container);
  color: var(--td-text-color-secondary);
  cursor: pointer;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.04);
  transition: background-color 0.18s ease, color 0.18s ease, box-shadow 0.18s ease;

  &:hover {
    background-color: var(--td-bg-color-secondarycontainer);
    color: var(--td-brand-color);
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
  }

  &:focus-visible {
    outline: 2px solid var(--td-brand-color-focus);
    outline-offset: 1px;
  }
}
</style>
