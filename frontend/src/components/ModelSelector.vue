<template>
  <div class="model-selector">
    <t-select
      :value="selectedModelId"
      @change="handleModelChange"
      :placeholder="placeholderText"
      :disabled="disabled"
      :loading="loading"
      :status="status"
      :clearable="clearable"
      filterable
      style="width: 100%;"
    >
      <!-- Existing model options -->
      <t-option
        v-for="model in models"
        :key="model.id"
        :value="model.id"
        :label="modelDisplayName(model)"
      >
        <div class="model-option">
          <t-icon name="check-circle-filled" class="model-icon" />
          <span class="model-name">{{ modelDisplayName(model) }}</span>
          <span v-if="model.display_name" class="model-raw-name">{{ model.name }}</span>
          <t-tag v-if="model.is_builtin" size="small" theme="primary">{{ $t('model.builtinTag') }}</t-tag>
          <t-tag v-if="model.is_default" size="small" theme="success">{{ $t('model.defaultTag') }}</t-tag>
        </div>
      </t-option>
      
      <!-- Add model option (at the bottom) -->
      <t-option
        v-if="!disabled"
        value="__add_model__"
        class="add-model-option"
      >
        <div class="model-option add">
          <t-icon name="add" class="add-icon" />
          <span class="model-name">{{ $t('model.addModelInSettings') }}</span>
        </div>
      </t-option>
    </t-select>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { listModels, type ModelConfig } from '@/api/model'
import { MessagePlugin } from 'tdesign-vue-next'
import { useI18n } from 'vue-i18n'

interface Props {
  modelType: 'KnowledgeQA' | 'Embedding' | 'Rerank' | 'VLLM' | 'ASR'
  selectedModelId?: string
  disabled?: boolean
  placeholder?: string
  status?: 'default' | 'success' | 'warning' | 'error'
  clearable?: boolean
  // Optional: full list of models passed in externally; if provided, skip the API call
  allModels?: ModelConfig[]
}

const props = withDefaults(defineProps<Props>(), {
  disabled: false,
  placeholder: '',
  status: 'default',
  clearable: false,
})

const emit = defineEmits<{
  'update:selectedModelId': [value: string]
  'add-model': []
}>()

const models = ref<ModelConfig[]>([])
const loading = ref(false)
const { t } = useI18n()

const placeholderText = computed(() => {
  return props.placeholder || t('model.selectModelPlaceholder')
})

const modelDisplayName = (model: ModelConfig) => {
  const displayName = model.display_name?.trim()
  return displayName || model.name
}

// Watch for changes to allModels and auto-filter models of the current type
watch(() => props.allModels, (newModels) => {
  if (newModels && Array.isArray(newModels)) {
    models.value = newModels.filter(m => m.type === props.modelType)
  }
}, { immediate: true })

const selectedModel = computed(() => {
  if (!props.selectedModelId) return null
  return models.value.find(m => m.id === props.selectedModelId)
})

// Load the model list (only called when allModels is not provided)
const loadModels = async () => {
  // If allModels is provided externally, no need to load
  if (props.allModels) {
    return
  }
  
  loading.value = true
  try {
    const result = await listModels()
    // Filter models by type on the frontend
    if (result && Array.isArray(result)) {
      models.value = result.filter(m => m.type === props.modelType)
    } else {
      models.value = []
    }
  } catch (error) {
    console.error(t('model.loadFailed'), error)
    MessagePlugin.error(t('model.loadFailed'))
    models.value = []
  } finally {
    loading.value = false
  }
}

// Handle model selection changes
const handleModelChange = (value?: string) => {
  // If the selected option is "add model", trigger the add event instead of updating the selected value
  if (value === '__add_model__') {
    emit('add-model')
    return
  }
  emit('update:selectedModelId', value || '')
}

// Expose a refresh method to the parent component
defineExpose({
  refresh: loadModels
})

onMounted(() => {
  // Only load when allModels is not provided
  if (!props.allModels) {
    loadModels()
  }
})
</script>

<style lang="less" scoped>
.model-selector {
  width: 100%;
}

.model-option {
  display: flex;
  align-items: center;
  gap: 8px;
  
  .model-icon {
    font-size: 14px;
    color: var(--td-brand-color);
  }
  
  .add-icon {
    font-size: 14px;
    color: var(--td-brand-color);
  }
  
  .model-name {
    flex: 0 1 auto;
    min-width: 0;
    font-size: 13px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .model-raw-name {
    flex: 1;
    min-width: 0;
    font-size: 12px;
    color: var(--td-text-color-placeholder);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  
  &.add {
    .model-name {
      color: var(--td-brand-color);
      font-weight: 500;
    }
  }
}
</style>
