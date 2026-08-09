import { listModels, type ModelConfig } from '@/api/model'

export interface TenantModelReadiness {
  chatCount: number
  embeddingCount: number
  hasChat: boolean
  hasEmbedding: boolean
  /** Whether the models required for vector/keyword search are complete when the knowledge base has them enabled by default */
  isReadyForDocumentKb: boolean
  /** Creating an agent requires at least a chat model */
  isReadyForAgent: boolean
}

export function evaluateTenantModelReadiness(models: ModelConfig[]): TenantModelReadiness {
  const chatCount = models.filter((m) => m.type === 'KnowledgeQA').length
  const embeddingCount = models.filter((m) => m.type === 'Embedding').length
  const hasChat = chatCount > 0
  const hasEmbedding = embeddingCount > 0
  return {
    chatCount,
    embeddingCount,
    hasChat,
    hasEmbedding,
    isReadyForDocumentKb: hasChat && hasEmbedding,
    isReadyForAgent: hasChat,
  }
}

export async function fetchTenantModelReadiness(): Promise<TenantModelReadiness> {
  try {
    const models = await listModels()
    return evaluateTenantModelReadiness(models || [])
  } catch {
    return evaluateTenantModelReadiness([])
  }
}
