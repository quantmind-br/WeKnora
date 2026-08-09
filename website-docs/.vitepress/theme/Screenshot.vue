<script setup lang="ts">
import { ref, computed } from 'vue'
import { withBase } from 'vitepress'

const props = defineProps<{
  /** Image path relative to the site root, conventionally placed under public/screenshots/ */
  src: string
  /** Caption: describes what this image shows */
  caption: string
  /** Additional notes: which UI elements the screenshot should cover, for reference when adding the image */
  hint?: string
}>()

// The image may not have been added yet. Render a placeholder box by default, and only
// switch to the image once it actually loads successfully, so a broken-image icon doesn't
// flash before falling back when the image is missing.
const loaded = ref(false)
const resolved = computed(() => withBase(props.src))
</script>

<template>
  <figure class="wk-shot">
    <img
      v-show="loaded"
      class="wk-shot-img"
      :src="resolved"
      :alt="caption"
      @load="loaded = true"
    />
    <div v-if="!loaded" class="wk-shot-placeholder">
      <div class="wk-shot-placeholder-badge">Screenshot pending</div>
      <div class="wk-shot-placeholder-caption">{{ caption }}</div>
      <p v-if="hint" class="wk-shot-placeholder-hint">{{ hint }}</p>
      <code class="wk-shot-placeholder-path">website-docs/public{{ src }}</code>
    </div>
    <figcaption class="wk-shot-caption">{{ caption }}</figcaption>
  </figure>
</template>

<style scoped>
.wk-shot {
  margin: 24px 0;
}

.wk-shot-img {
  display: block;
  width: 100%;
  border: 1px solid var(--vp-c-divider);
  border-radius: 10px;
}

.wk-shot-placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 32px 24px;
  text-align: center;
  border: 1px dashed var(--vp-c-divider);
  border-radius: 10px;
  background: var(--vp-c-bg-soft);
}

.wk-shot-placeholder-badge {
  padding: 2px 10px;
  border-radius: 999px;
  font-size: 12px;
  line-height: 20px;
  color: var(--vp-c-text-2);
  background: var(--vp-c-default-soft);
}

.wk-shot-placeholder-caption {
  font-size: 15px;
  font-weight: 600;
  color: var(--vp-c-text-1);
}

.wk-shot-placeholder-hint {
  max-width: 46em;
  margin: 0;
  font-size: 13px;
  line-height: 1.7;
  color: var(--vp-c-text-2);
}

.wk-shot-placeholder-path {
  font-size: 12px;
  color: var(--vp-c-text-3);
}

.wk-shot-caption {
  margin-top: 8px;
  font-size: 13px;
  line-height: 1.6;
  text-align: center;
  color: var(--vp-c-text-2);
}
</style>
