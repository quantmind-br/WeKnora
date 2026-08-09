import { defineStore } from "pinia";
import { nextTick } from "vue";
import { BUILTIN_QUICK_ANSWER_ID, BUILTIN_SMART_REASONING_ID } from "@/api/agent";
import { getApiBaseUrl } from "@/utils/api-base";
import { isAgentStreamAgentId } from "@/utils/agent-mode";
import { loadAndReconcileSettings } from "@/stores/settingsStorage";

// Define settings interface
interface Settings {
  endpoint: string;
  apiKey: string;
  knowledgeBaseId: string;
  isAgentEnabled: boolean;
  agentConfig: AgentConfig;
  selectedKnowledgeBases: string[];  // Currently selected knowledge base ID list
  selectedFiles: string[]; // Currently selected file ID list
  selectedFileKbMap: Record<string, string>; // File ID -> knowledge base ID, used to fetch shared knowledge base files with kb_id after refresh
  selectedTags: Array<{ id: string; name: string; kbId: string; kbName?: string }>;
  selectedMCPServices: string[];
  selectedSkills: string[];
  selectedTools?: string[];
  modelConfig: ModelConfig;  // Model configuration
  ollamaConfig: OllamaConfig;  // Ollama configuration
  webSearchEnabled: boolean;  // Whether web search is enabled
  conversationModels: ConversationModels;
  selectedAgentId: string;  // Currently selected agent ID
  selectedAgentSourceTenantId: string | null;  // Source space ID when using a shared agent (used for backend model/KB/MCP resolution)
  autoCheckUpdate?: boolean; // Whether to automatically check for and download updates
}

// Agent configuration interface
interface AgentConfig {
  maxIterations: number;
  temperature: number;
  allowedTools: string[];
  system_prompt?: string;  // Unified system prompt (uses {{web_search_status}} placeholder)
}

interface ConversationModels {
  summaryModelId: string;
  rerankModelId: string;
  selectedChatModelId: string;  // Currently selected conversation model ID
}

// Single model item interface
interface ModelItem {
  id: string;  // Unique ID
  name: string;  // Display name
  source: 'local' | 'remote';  // Model source
  modelName: string;  // Model identifier
  baseUrl?: string;  // Remote API URL
  apiKey?: string;  // Remote API Key
  dimension?: number;  // Embedding-specific: vector dimension
  interfaceType?: 'ollama' | 'openai';  // VLLM-specific: interface type
  isDefault?: boolean;  // Whether this is the default model
}

// Model configuration interface - supports multiple models
interface ModelConfig {
  chatModels: ModelItem[];
  embeddingModels: ModelItem[];
  rerankModels: ModelItem[];
  vllmModels: ModelItem[];  // VLLM vision model
}

// Ollama configuration interface
interface OllamaConfig {
  baseUrl: string;  // Ollama service address
  enabled: boolean;  // Whether enabled
}

// Default settings
const defaultSettings: Settings = {
  endpoint: getApiBaseUrl(),
  apiKey: "",
  knowledgeBaseId: "",
  isAgentEnabled: false,
  agentConfig: {
    maxIterations: 5,
    temperature: 0.7,
    allowedTools: [],  // Empty by default, needs to be loaded from the backend via API
    system_prompt: "",
  },
  selectedKnowledgeBases: [],  // Empty array by default
  selectedFiles: [], // Empty array by default
  selectedFileKbMap: {},  // File ID -> knowledge base ID
  selectedTags: [],
  selectedMCPServices: [],
  selectedSkills: [],
  modelConfig: {
    chatModels: [],
    embeddingModels: [],
    rerankModels: [],
    vllmModels: []
  },
  ollamaConfig: {
    baseUrl: "http://localhost:11434",
    enabled: true
  },
  webSearchEnabled: false,  // Web search disabled by default
  conversationModels: {
    summaryModelId: "",
    rerankModelId: "",
    selectedChatModelId: "",  // Currently selected conversation model ID
  },
  selectedAgentId: BUILTIN_QUICK_ANSWER_ID,  // Quick Q&A mode selected by default
  selectedAgentSourceTenantId: null as string | null,  // Shared agent source space ID
  autoCheckUpdate: true,
};

export const useSettingsStore = defineStore("settings", {
  state: () => ({
    // Load settings from local storage, fall back to defaults if none
    settings: loadAndReconcileSettings(defaultSettings),
    // Snapshot the "global default" on entering the session; restore it on leaving. Non-persistent fields:
    // Refreshing the page is equivalent to re-entering the "enter session" flow, so it naturally re-snapshots.
    _defaultsSnapshot: null as Settings | null,
    /** Restoring the input bar from session.last_request_state, to avoid the agent-switch watcher overriding the KB selection */
    _isApplyingSessionState: false,
  }),

  getters: {
    // Whether the Agent is enabled
    isAgentEnabled: (state) => state.settings.isAgentEnabled || false,

    // Whether the built-in Quick Q&A is currently active (prefer selectedAgentId to avoid drifting from isAgentEnabled)
    isQuickAnswerMode: (state) =>
      (state.settings.selectedAgentId || BUILTIN_QUICK_ANSWER_ID) === BUILTIN_QUICK_ANSWER_ID,

    // Whether to use the Agent streaming pipeline (intelligent reasoning / custom Agent); Quick Q&A uses the RAG pipeline
    isAgentStreamMode: (state) =>
      isAgentStreamAgentId(
        state.settings.selectedAgentId,
        state.settings.isAgentEnabled || false,
      ),
    
    // Whether the Agent is ready (fully configured)
    // Requires: 1) allowed tools configured 2) conversation model set 3) rerank model set
    isAgentReady: (state) => {
      const config = state.settings.agentConfig || defaultSettings.agentConfig
      const models = state.settings.conversationModels || defaultSettings.conversationModels
      return Boolean(
        config.allowedTools && config.allowedTools.length > 0 &&
        models.summaryModelId && models.summaryModelId.trim() !== '' &&
        models.rerankModelId && models.rerankModelId.trim() !== ''
      )
    },
    
    // Whether normal mode (quick answer) is ready
    // Requires: 1) conversation model set 2) rerank model set
    isNormalModeReady: (state) => {
      const models = state.settings.conversationModels || defaultSettings.conversationModels
      return Boolean(
        models.summaryModelId && models.summaryModelId.trim() !== '' &&
        models.rerankModelId && models.rerankModelId.trim() !== ''
      )
    },
    
    // Get Agent configuration
    agentConfig: (state) => state.settings.agentConfig || defaultSettings.agentConfig,

    conversationModels: (state) => state.settings.conversationModels || defaultSettings.conversationModels,
    
    // Get model configuration
    modelConfig: (state) => state.settings.modelConfig || defaultSettings.modelConfig,
    
    // Whether web search is enabled
    isWebSearchEnabled: (state) => state.settings.webSearchEnabled || false,
    
    // Whether to automatically check for and download updates
    isAutoCheckUpdateEnabled: (state) => state.settings.autoCheckUpdate ?? true,

    // Currently selected agent ID
    selectedAgentId: (state) => state.settings.selectedAgentId || BUILTIN_QUICK_ANSWER_ID,
    // Shared agent source space ID (optional)
    selectedAgentSourceTenantId: (state) => state.settings.selectedAgentSourceTenantId ?? null,
  },

  actions: {
    // Save settings
    saveSettings(settings: Settings) {
      this.settings = { ...settings };
      // Save to localStorage
      localStorage.setItem("WeKnora_settings", JSON.stringify(this.settings));
    },

    // Get settings
    getSettings(): Settings {
      return this.settings;
    },

    // Get API endpoint
    getEndpoint(): string {
      return this.settings.endpoint || defaultSettings.endpoint;
    },

    // Get API Key
    getApiKey(): string {
      return this.settings.apiKey;
    },

    // Get knowledge base ID
    getKnowledgeBaseId(): string {
      return this.settings.knowledgeBaseId;
    },
    
    // Enable/disable Agent
    toggleAgent(enabled: boolean) {
      this.settings.isAgentEnabled = enabled;
      localStorage.setItem("WeKnora_settings", JSON.stringify(this.settings));
    },
    
    // Update Agent configuration
    updateAgentConfig(config: Partial<AgentConfig>) {
      this.settings.agentConfig = { ...this.settings.agentConfig, ...config };
      localStorage.setItem("WeKnora_settings", JSON.stringify(this.settings));
    },

    updateConversationModels(models: Partial<ConversationModels>) {
      const current = this.settings.conversationModels || defaultSettings.conversationModels;
      this.settings.conversationModels = { ...current, ...models };
      localStorage.setItem("WeKnora_settings", JSON.stringify(this.settings));
    },
    
    // Update model configuration
    updateModelConfig(config: Partial<ModelConfig>) {
      this.settings.modelConfig = { ...this.settings.modelConfig, ...config };
      localStorage.setItem("WeKnora_settings", JSON.stringify(this.settings));
    },
    
    // Add model
    addModel(type: 'chat' | 'embedding' | 'rerank' | 'vllm', model: ModelItem) {
      const key = `${type}Models` as keyof ModelConfig;
      const models = [...this.settings.modelConfig[key]] as ModelItem[];
      // If set as default, unset the default status of other models
      if (model.isDefault) {
        models.forEach(m => m.isDefault = false);
      }
      // If this is the first model, automatically set it as default
      if (models.length === 0) {
        model.isDefault = true;
      }
      models.push(model);
      this.settings.modelConfig[key] = models as any;
      localStorage.setItem("WeKnora_settings", JSON.stringify(this.settings));
    },
    
    // Update model
    updateModel(type: 'chat' | 'embedding' | 'rerank' | 'vllm', modelId: string, updates: Partial<ModelItem>) {
      const key = `${type}Models` as keyof ModelConfig;
      const models = [...this.settings.modelConfig[key]] as ModelItem[];
      const index = models.findIndex(m => m.id === modelId);
      if (index !== -1) {
        // If setting as default, unset the default status of other models
        if (updates.isDefault) {
          models.forEach(m => m.isDefault = false);
        }
        models[index] = { ...models[index], ...updates };
        this.settings.modelConfig[key] = models as any;
        localStorage.setItem("WeKnora_settings", JSON.stringify(this.settings));
      }
    },
    
    // Delete model
    deleteModel(type: 'chat' | 'embedding' | 'rerank' | 'vllm', modelId: string) {
      const key = `${type}Models` as keyof ModelConfig;
      let models = [...this.settings.modelConfig[key]] as ModelItem[];
      const deletedModel = models.find(m => m.id === modelId);
      models = models.filter(m => m.id !== modelId);
      // If the deleted model was the default, set the first one as default
      if (deletedModel?.isDefault && models.length > 0) {
        models[0].isDefault = true;
      }
      this.settings.modelConfig[key] = models as any;
      localStorage.setItem("WeKnora_settings", JSON.stringify(this.settings));
    },
    
    // Set default model
    setDefaultModel(type: 'chat' | 'embedding' | 'rerank' | 'vllm', modelId: string) {
      const key = `${type}Models` as keyof ModelConfig;
      const models = [...this.settings.modelConfig[key]] as ModelItem[];
      models.forEach(m => m.isDefault = (m.id === modelId));
      this.settings.modelConfig[key] = models as any;
      localStorage.setItem("WeKnora_settings", JSON.stringify(this.settings));
    },
    
    // Update Ollama configuration
    updateOllamaConfig(config: Partial<OllamaConfig>) {
      this.settings.ollamaConfig = { ...this.settings.ollamaConfig, ...config };
      localStorage.setItem("WeKnora_settings", JSON.stringify(this.settings));
    },
    
    // Select knowledge base (replace the entire list)
    selectKnowledgeBases(kbIds: string[]) {
      this.settings.selectedKnowledgeBases = kbIds;
      localStorage.setItem("WeKnora_settings", JSON.stringify(this.settings));
    },
    
    // Add a single knowledge base
    addKnowledgeBase(kbId: string) {
      if (!this.settings.selectedKnowledgeBases.includes(kbId)) {
        this.settings.selectedKnowledgeBases.push(kbId);
        localStorage.setItem("WeKnora_settings", JSON.stringify(this.settings));
      }
    },
    
    // Remove a single knowledge base
    removeKnowledgeBase(kbId: string) {
      this.settings.selectedKnowledgeBases = 
        this.settings.selectedKnowledgeBases.filter((id: string) => id !== kbId);
      localStorage.setItem("WeKnora_settings", JSON.stringify(this.settings));
    },
    
    // Clear knowledge base selection
    clearKnowledgeBases() {
      this.settings.selectedKnowledgeBases = [];
      localStorage.setItem("WeKnora_settings", JSON.stringify(this.settings));
    },
    
    // Get the selected knowledge base list
    getSelectedKnowledgeBases(): string[] {
      return this.settings.selectedKnowledgeBases || [];
    },
    
    // Enable/disable web search
    toggleWebSearch(enabled: boolean) {
      this.settings.webSearchEnabled = enabled;
      localStorage.setItem("WeKnora_settings", JSON.stringify(this.settings));
    },

    // Enable/disable automatic update checks
    toggleAutoCheckUpdate(enabled: boolean) {
      this.settings.autoCheckUpdate = enabled;
      localStorage.setItem("WeKnora_settings", JSON.stringify(this.settings));
    },

    // File selection actions
    addFile(fileId: string) {
      if (!this.settings.selectedFiles) this.settings.selectedFiles = [];
      if (!this.settings.selectedFiles.includes(fileId)) {
        this.settings.selectedFiles.push(fileId);
        localStorage.setItem("WeKnora_settings", JSON.stringify(this.settings));
      }
    },

    removeFile(fileId: string) {
      if (!this.settings.selectedFiles) return;
      this.settings.selectedFiles = this.settings.selectedFiles.filter((id: string) => id !== fileId);
      if (this.settings.selectedFileKbMap) delete this.settings.selectedFileKbMap[fileId];
      localStorage.setItem("WeKnora_settings", JSON.stringify(this.settings));
    },

    clearFiles() {
      this.settings.selectedFiles = [];
      this.settings.selectedFileKbMap = {};
      localStorage.setItem("WeKnora_settings", JSON.stringify(this.settings));
    },

    addTag(tag: { id: string; name: string; kbId: string; kbName?: string }) {
      if (!this.settings.selectedTags) this.settings.selectedTags = [];
      if (!this.settings.selectedTags.some(t => t.id === tag.id && t.kbId === tag.kbId)) {
        this.settings.selectedTags.push(tag);
        localStorage.setItem("WeKnora_settings", JSON.stringify(this.settings));
      }
    },

    removeTag(tagId: string, kbId?: string) {
      if (!this.settings.selectedTags) return;
      this.settings.selectedTags = this.settings.selectedTags.filter(t => !(t.id === tagId && (!kbId || t.kbId === kbId)));
      localStorage.setItem("WeKnora_settings", JSON.stringify(this.settings));
    },

    clearTags() {
      this.settings.selectedTags = [];
      localStorage.setItem("WeKnora_settings", JSON.stringify(this.settings));
    },

    addMCPService(serviceId: string) {
      if (!this.settings.selectedMCPServices) this.settings.selectedMCPServices = [];
      if (!this.settings.selectedMCPServices.includes(serviceId)) {
        this.settings.selectedMCPServices.push(serviceId);
        localStorage.setItem("WeKnora_settings", JSON.stringify(this.settings));
      }
    },

    removeMCPService(serviceId: string) {
      if (!this.settings.selectedMCPServices) return;
      this.settings.selectedMCPServices = this.settings.selectedMCPServices.filter(id => id !== serviceId);
      localStorage.setItem("WeKnora_settings", JSON.stringify(this.settings));
    },

    addSkill(skillName: string) {
      if (!this.settings.selectedSkills) this.settings.selectedSkills = [];
      if (!this.settings.selectedSkills.includes(skillName)) {
        this.settings.selectedSkills.push(skillName);
        localStorage.setItem("WeKnora_settings", JSON.stringify(this.settings));
      }
    },

    removeSkill(skillName: string) {
      if (!this.settings.selectedSkills) return;
      this.settings.selectedSkills = this.settings.selectedSkills.filter(name => name !== skillName);
      localStorage.setItem("WeKnora_settings", JSON.stringify(this.settings));
    },

    setFileKbMap(updates: Record<string, string>) {
      if (!this.settings.selectedFileKbMap) this.settings.selectedFileKbMap = {};
      Object.assign(this.settings.selectedFileKbMap, updates);
      localStorage.setItem("WeKnora_settings", JSON.stringify(this.settings));
    },

    removeFileKbId(fileId: string) {
      if (this.settings.selectedFileKbMap) delete this.settings.selectedFileKbMap[fileId];
      localStorage.setItem("WeKnora_settings", JSON.stringify(this.settings));
    },
    
    getSelectedFiles(): string[] {
      return this.settings.selectedFiles || [];
    },

    /**
     * Scope for suggested-questions API (KB / file / tag @mentions).
     * Leave `limit` undefined to let the backend apply the agent's configured
     * starter count; pass a number only to request a specific count.
     */
    getSuggestedQuestionsParams(limit?: number) {
      const selectedKBs = this.getSelectedKnowledgeBases();
      const selectedFiles = this.getSelectedFiles();
      const tags = this.settings.selectedTags || [];
      const tagScopes = Object.entries(tags.reduce<Record<string, string[]>>((scopes, tag) => {
        if (!tag.id || !tag.kbId) return scopes;
        (scopes[tag.kbId] ||= []).push(tag.id);
        return scopes;
      }, {})).map(([knowledge_base_id, ids]) => ({
        knowledge_base_id,
        tag_ids: [...new Set(ids)],
      }));
      return {
        // A tag's parent KB is only an ownership hint, not an explicit whole-KB
        // selection. Keep it in tag_scopes so the backend cannot widen a tag to
        // every document in that KB.
        knowledge_base_ids: selectedKBs.length > 0 ? selectedKBs : undefined,
        knowledge_ids: selectedFiles.length > 0 ? selectedFiles : undefined,
        tag_scopes: tagScopes.length > 0 ? tagScopes : undefined,
        limit,
      };
    },
    
    // Select agent (sourceTenantId is only passed when using a shared agent)
    selectAgent(agentId: string, sourceTenantId?: string | null) {
      this.settings.selectedAgentId = agentId;
      this.settings.selectedAgentSourceTenantId = (sourceTenantId != null && sourceTenantId !== "") ? sourceTenantId : null;
      // The agent configuration only determines whether web search capability is available, not whether it's used this turn — that's still up to the user.
      // Every time an agent is selected, it defaults to off; after that, only the user can turn it on from the input box.
      this.settings.webSearchEnabled = false;
      // Automatically switch Agent mode based on agent type
      if (agentId === BUILTIN_QUICK_ANSWER_ID) {
        this.settings.isAgentEnabled = false;
      } else if (agentId === BUILTIN_SMART_REASONING_ID) {
        this.settings.isAgentEnabled = true;
      }
      // For custom agents, this must be decided based on their configuration
      
      // Reset knowledge base and file selection state when switching agents
      // Because different agents are associated with different knowledge bases, the user's previous selection must be cleared
      this.settings.selectedKnowledgeBases = [];
      this.settings.selectedFiles = [];
      this.settings.selectedFileKbMap = {};
      this.settings.selectedTags = [];
      this.settings.selectedMCPServices = [];
      this.settings.selectedSkills = [];
      localStorage.setItem("WeKnora_settings", JSON.stringify(this.settings));
    },
    
    // Get the currently selected agent ID
    getSelectedAgentId(): string {
      return this.settings.selectedAgentId || BUILTIN_QUICK_ANSWER_ID;
    },

    // —— Session-level input state restoration —— //
    //
    // The agent / model / KB / web search / MCP selections in the input bar are held by this store and shared across sessions.
    // But what the user wants is: when opening an old session, see the exact state that was active when the request was originally sent.
    // Implementation strategy: when entering a session, temporarily stash the "current global defaults" into a non-persistent `_defaultsSnapshot`
    // field, then override the store with session.last_request_state; restore from the snapshot when leaving the session.
    // Snapshot isn't written to localStorage, since it only matters during the route lifetime of "currently in an old session";
    // Refreshing the page is equivalent to "re-entering the session" → re-snapshot + overwrite, without losing the user's global default.

    // Capture the current settings as "the default to restore once the session is left".
    // If a snapshot already exists, don't overwrite it, to avoid switching between sessions (B→B') mistaking an already-restored store for the default.
    snapshotAsDefaultsIfNeeded() {
      if (this._defaultsSnapshot) return;
      this._defaultsSnapshot = JSON.parse(JSON.stringify(this.settings));
    },

    // Restore the default (if a snapshot exists), used when leaving a session or switching across sessions.
    restoreDefaultsIfSnapshotted() {
      if (!this._defaultsSnapshot) return;
      this.settings = this._defaultsSnapshot;
      this._defaultsSnapshot = null;
      // Don't write to localStorage: the default value was already written to localStorage before the snapshot, so restoring here
      // just brings back the value already in localStorage; writing it again would only add pointless IO.
    },

    // Overwrite input-bar-related fields based on session.last_request_state.
    // Only touch the fields recorded this time, and do **not** clear other unrelated fields in the store (e.g. the model list).
    // If any field is missing, keep the store's current value — a "best-effort restore".
    applyLastRequestState(state: SessionLastRequestStatePayload | null | undefined) {
      if (!state) return;
      this._isApplyingSessionState = true;
      try {
        if (typeof state.agent_enabled === "boolean") {
          this.settings.isAgentEnabled = state.agent_enabled;
        }
        if (typeof state.agent_id === "string" && state.agent_id) {
          this.settings.selectedAgentId = state.agent_id;
          // Whether the last record was a self-owned agent or a shared agent — the server currently doesn't return sourceTenantId to distinguish this.
          // Unlike selectAgent(), this does **not** reset the KB/file selection — because right after this
          // we're going to overwrite it with the KB/file from state anyway, so there's no need to clear it first.
        }
        if (state.model_id !== undefined) {
          const current = this.settings.conversationModels || defaultSettings.conversationModels;
          this.settings.conversationModels = { ...current, selectedChatModelId: state.model_id || "" };
        }
        if (Array.isArray(state.knowledge_base_ids)) {
          this.settings.selectedKnowledgeBases = [...state.knowledge_base_ids];
        }
        if (Array.isArray(state.knowledge_ids)) {
          this.settings.selectedFiles = [...state.knowledge_ids];
          // selectedFileKbMap can't be rebuilt at this point (KB ownership isn't stored in state); leave it to the frontend to
          // lazily fetch as needed. Keep the store's current value, to avoid accidentally deleting file mappings the user just added.
        }
        if (Array.isArray(state.mentioned_items)) {
          const fromMentions = state.mentioned_items
            .filter(item => item.type === "tag" && item.id && item.kb_id)
            .map(item => ({ id: item.id, name: item.name || item.id, kbId: item.kb_id!, kbName: item.kb_name }));
          const covered = new Set(fromMentions.map(t => t.id));
          const orphanTagIds = (state.tag_ids || []).filter(id => id && !covered.has(id));
          if (orphanTagIds.length > 0 && Array.isArray(state.knowledge_base_ids) && state.knowledge_base_ids.length === 1) {
            const kbId = state.knowledge_base_ids[0];
            orphanTagIds.forEach(id => {
              fromMentions.push({ id, name: id, kbId, kbName: undefined });
            });
          }
          this.settings.selectedTags = fromMentions;
        } else if (Array.isArray(state.tag_ids)) {
          const existing = this.settings.selectedTags || [];
          this.settings.selectedTags = existing.filter(tag => state.tag_ids?.includes(tag.id));
        }
        if (Array.isArray(state.mcp_service_ids)) {
          this.settings.selectedMCPServices = [...state.mcp_service_ids];
        } else if (Array.isArray(state.mentioned_items)) {
          this.settings.selectedMCPServices = state.mentioned_items
            .filter(item => item.type === "mcp" && item.id)
            .map(item => item.id);
        }
        if (Array.isArray(state.skill_names)) {
          this.settings.selectedSkills = [...state.skill_names];
        } else if (Array.isArray(state.mentioned_items)) {
          this.settings.selectedSkills = state.mentioned_items
            .filter(item => item.type === "skill" && item.id)
            .map(item => item.skill_name || item.id);
        }
        if (typeof state.web_search_enabled === "boolean") {
          this.settings.webSearchEnabled = state.web_search_enabled;
        }
      } finally {
        // The reset must be deferred until after the next flush: the watcher on selectedAgentId defaults to
        // flush:'pre', which runs asynchronously; resetting synchronously here would mean that by the time the watcher actually runs, the flag
        // is already false, making the guard useless — the restored KB would still get overwritten by the agent config. Deferring to nextTick
        // guarantees the watcher triggered by this state change runs while the flag is still true.
        nextTick(() => {
          this._isApplyingSessionState = false;
        });
      }
      // Note: intentionally not writing to localStorage — state from an old session shouldn't pollute the "user default".
      // When leaving a session, restoreDefaultsIfSnapshotted syncs the full default value stored in localStorage
      // back into this.settings again.
    },
  },
});

// Backend sessions.last_request_state JSON shape (aligned with SessionLastRequestState).
// All fields are optional — historical sessions, or sessions before the first request of a new session, don't have this record.
export interface SessionLastRequestStatePayload {
  agent_id?: string;
  agent_enabled?: boolean;
  model_id?: string;
  knowledge_base_ids?: string[];
  knowledge_ids?: string[];
  tag_ids?: string[];
  mcp_service_ids?: string[];
  skill_names?: string[];
  mentioned_items?: Array<{
    id: string;
    name?: string;
    type: string;
    kb_id?: string;
    kb_name?: string;
    skill_name?: string;
  }>;
  web_search_enabled?: boolean;
}
