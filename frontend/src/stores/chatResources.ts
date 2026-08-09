import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { listKnowledgeBases, getKnowledgeBaseById } from '@/api/knowledge-base'
import { listAgents, type CustomAgent } from '@/api/agent'
import { listModels, type ModelConfig } from '@/api/model'
import { listWebSearchProviders, type WebSearchProviderEntity } from '@/api/web-search-provider'
import { useOrganizationStore } from '@/stores/organization'

/** Space-level resource cache TTL */
const CACHE_TTL_MS = 60_000

type ResourceKey = 'knowledgeBases' | 'agents' | 'models' | 'webSearchProviders'

export type ListCreatorFilter = 'all' | 'mine' | 'others'

function isKbModelReady(kb: any): boolean {
  if (!kb.summary_model_id || kb.summary_model_id === '') return false
  const strategy = kb.indexing_strategy
  const needsEmbedding = !strategy || strategy.vector_enabled || strategy.keyword_enabled
  if (needsEmbedding && (!kb.embedding_model_id || kb.embedding_model_id === '')) return false
  return true
}

export const useChatResourcesStore = defineStore('chatResources', () => {
  const rawKnowledgeBases = ref<any[]>([])
  const agents = ref<CustomAgent[]>([])
  const disabledOwnAgentIds = ref<string[]>([])
  const allModels = ref<ModelConfig[]>([])
  const webSearchProviders = ref<WebSearchProviderEntity[]>([])

  const loadedAt = ref<Partial<Record<ResourceKey, number>>>({})
  const inflight = new Map<ResourceKey, Promise<void>>()
  // List requests with creator==='all' are deduplicated separately: first-screen platform prefetch and onMounted on the chat page
  // May be triggered concurrently; without dedup while the cache hasn't been written yet, listKnowledgeBases / listAgents would be hit repeatedly.
  let kbAllInflight: Promise<any[]> | null = null
  let agentsAllInflight: Promise<{ data: CustomAgent[]; disabled_own_agent_ids: string[] }> | null = null
  // Generation counter: when force and non-force run concurrently, the handle gets overwritten by the later one, and the old request uses this on completion to determine
  // whether it's still the latest, to avoid mistakenly clearing an in-flight handle.
  let kbAllGen = 0
  let agentsAllGen = 0

  const agentKbCache = new Map<string, { at: number; data: any[] }>()
  const agentKbInflight = new Map<string, Promise<any[]>>()
  const kbDetailCache = new Map<string, { at: number; data: any }>()
  const kbDetailInflight = new Map<string, Promise<any | null>>()

  const validKnowledgeBases = computed(() => rawKnowledgeBases.value.filter(isKbModelReady))
  const chatModels = computed(() => allModels.value.filter((m) => m.type === 'KnowledgeQA'))

  function isFresh(key: ResourceKey): boolean {
    const at = loadedAt.value[key]
    return !!at && Date.now() - at < CACHE_TTL_MS
  }

  async function runOnce(key: ResourceKey, force: boolean, loader: () => Promise<void>): Promise<void> {
    if (!force && isFresh(key)) return
    const existing = inflight.get(key)
    if (existing) return existing
    const p = loader().finally(() => {
      inflight.delete(key)
    })
    inflight.set(key, p)
    return p
  }

  /**
   * Knowledge base list (supports creator filtering). When creator=all, writes to the cache for reuse on the chat page.
   */
  async function fetchKnowledgeBasesForList(
    params?: { creator?: ListCreatorFilter },
    force = false,
  ): Promise<any[]> {
    const creator = params?.creator ?? 'all'
    // Lists with creator filtering are list-page-only, not cached, and pass the request straight through.
    if (creator !== 'all') {
      const res: any = await listKnowledgeBases({ creator })
      return res?.data && Array.isArray(res.data) ? res.data : []
    }

    if (!force && isFresh('knowledgeBases')) {
      return rawKnowledgeBases.value
    }
    if (!force && kbAllInflight) return kbAllInflight

    const gen = ++kbAllGen
    kbAllInflight = (async () => {
      try {
        const res: any = await listKnowledgeBases()
        const data = res?.data && Array.isArray(res.data) ? res.data : []
        rawKnowledgeBases.value = data
        loadedAt.value.knowledgeBases = Date.now()
        const orgStore = useOrganizationStore()
        await orgStore.fetchSharedKnowledgeBases({ force })
        return data
      } finally {
        if (kbAllGen === gen) kbAllInflight = null
      }
    })()
    return kbAllInflight
  }

  async function ensureKnowledgeBases(force = false): Promise<void> {
    await fetchKnowledgeBasesForList({ creator: 'all' }, force)
  }

  /**
   * Agent list (supports creator filtering). When creator=all, writes to the cache.
   */
  async function fetchAgentsForList(
    params?: { creator?: ListCreatorFilter },
    force = false,
  ): Promise<{ data: CustomAgent[]; disabled_own_agent_ids: string[] }> {
    const creator = params?.creator ?? 'all'
    const orgStore = useOrganizationStore()

    // Lists with creator filtering aren't cached, but shared agents still need to be refreshed (consistent with the full-fetch path).
    if (creator !== 'all') {
      const [agentsRes] = await Promise.all([
        listAgents({ creator }),
        orgStore.fetchSharedAgents({ force }),
      ])
      const res = agentsRes as { data?: CustomAgent[]; disabled_own_agent_ids?: string[] }
      return { data: res.data || [], disabled_own_agent_ids: res.disabled_own_agent_ids || [] }
    }

    if (!force && isFresh('agents')) {
      return { data: agents.value, disabled_own_agent_ids: disabledOwnAgentIds.value }
    }
    if (!force && agentsAllInflight) return agentsAllInflight

    const gen = ++agentsAllGen
    agentsAllInflight = (async () => {
      try {
        const [agentsRes] = await Promise.all([
          listAgents(),
          orgStore.fetchSharedAgents({ force }),
        ])
        const res = agentsRes as { data?: CustomAgent[]; disabled_own_agent_ids?: string[] }
        const data = res.data || []
        agents.value = data
        disabledOwnAgentIds.value = res.disabled_own_agent_ids || []
        loadedAt.value.agents = Date.now()
        return { data, disabled_own_agent_ids: res.disabled_own_agent_ids || [] }
      } finally {
        if (agentsAllGen === gen) agentsAllInflight = null
      }
    })()
    return agentsAllInflight
  }

  async function ensureAgents(force = false): Promise<void> {
    await fetchAgentsForList({ creator: 'all' }, force)
  }

  async function ensureModels(force = false): Promise<void> {
    return runOnce('models', force, async () => {
      const models = await listModels()
      allModels.value = Array.isArray(models) ? models : []
      loadedAt.value.models = Date.now()
    })
  }

  /** @deprecated use ensureModels; kept as an alias for the chat input bar to call */
  async function ensureChatModels(force = false): Promise<void> {
    return ensureModels(force)
  }

  async function ensureWebSearchProviders(force = false): Promise<void> {
    return runOnce('webSearchProviders', force, async () => {
      const response = await listWebSearchProviders()
      const providers = (response as any)?.data
      webSearchProviders.value = Array.isArray(providers) ? providers : []
      loadedAt.value.webSearchProviders = Date.now()
    })
  }

  /** Prefetch in parallel the space-level resources commonly used by the chat input bar and list pages */
  async function prefetchChatInput(force = false): Promise<void> {
    const orgStore = useOrganizationStore()
    await Promise.all([
      ensureKnowledgeBases(force),
      ensureAgents(force),
      ensureModels(force),
      ensureWebSearchProviders(force),
      orgStore.fetchOrganizations({ force }),
    ])
  }

  async function ensureAgentKnowledgeBases(agentId: string, sourceTenantId?: string, force = false): Promise<any[]> {
    const cacheKey = `${agentId}:${sourceTenantId || 'current'}`
    const cached = agentKbCache.get(cacheKey)
    if (!force && cached && Date.now() - cached.at < CACHE_TTL_MS) {
      return cached.data
    }
    const existing = agentKbInflight.get(cacheKey)
    if (existing) return existing

    const p = (async () => {
      try {
        const res: any = await listKnowledgeBases({
          agent_id: agentId,
          agent_source_tenant_id: sourceTenantId,
        })
        const list = res?.data && Array.isArray(res.data) ? res.data : []
        agentKbCache.set(cacheKey, { at: Date.now(), data: list })
        return list
      } finally {
        agentKbInflight.delete(cacheKey)
      }
    })()
    agentKbInflight.set(cacheKey, p)
    return p
  }

  /** Single knowledge base detail (shared by sidebar + detail page, deduplicates concurrent requests) */
  async function fetchKnowledgeBaseById(kbId: string, force = false): Promise<any | null> {
    if (!kbId) return null
    const cached = kbDetailCache.get(kbId)
    if (!force && cached && Date.now() - cached.at < CACHE_TTL_MS) {
      return cached.data
    }
    const existing = kbDetailInflight.get(kbId)
    if (existing) return existing

    const p = (async () => {
      try {
        const res: any = await getKnowledgeBaseById(kbId)
        const data = res?.data ?? null
        if (data) {
          kbDetailCache.set(kbId, { at: Date.now(), data })
        }
        return data
      } catch {
        return null
      } finally {
        kbDetailInflight.delete(kbId)
      }
    })()
    kbDetailInflight.set(kbId, p)
    return p
  }

  function invalidateKnowledgeBaseDetail(kbId?: string) {
    if (kbId) {
      kbDetailCache.delete(kbId)
      kbDetailInflight.delete(kbId)
    } else {
      kbDetailCache.clear()
      kbDetailInflight.clear()
    }
  }

  function invalidate(...keys: ResourceKey[]) {
    if (keys.length === 0) {
      loadedAt.value = {}
      rawKnowledgeBases.value = []
      agents.value = []
      disabledOwnAgentIds.value = []
      allModels.value = []
      webSearchProviders.value = []
      agentKbCache.clear()
      // Also discard all inflight handles at the same time, otherwise a still-in-flight request would write stale data back into the cache after invalidation.
      inflight.clear()
      agentKbInflight.clear()
      kbAllInflight = null
      agentsAllInflight = null
      invalidateKnowledgeBaseDetail()
      return
    }
    keys.forEach((k) => {
      delete loadedAt.value[k]
      inflight.delete(k)
    })
    if (keys.includes('knowledgeBases')) {
      agentKbCache.clear()
      agentKbInflight.clear()
      kbAllInflight = null
      invalidateKnowledgeBaseDetail()
    }
    if (keys.includes('agents')) {
      agentsAllInflight = null
    }
  }

  return {
    rawKnowledgeBases,
    validKnowledgeBases,
    agents,
    disabledOwnAgentIds,
    allModels,
    chatModels,
    webSearchProviders,
    isFresh,
    fetchKnowledgeBasesForList,
    fetchAgentsForList,
    ensureKnowledgeBases,
    ensureAgents,
    ensureModels,
    ensureChatModels,
    ensureWebSearchProviders,
    ensureAgentKnowledgeBases,
    prefetchChatInput,
    fetchKnowledgeBaseById,
    invalidateKnowledgeBaseDetail,
    invalidate,
  }
})
