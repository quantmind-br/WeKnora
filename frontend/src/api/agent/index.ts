import { get, post, put, del } from "../../utils/request";

// Agent configuration
// Agent type preset ID under smart reasoning
// 'rag-qa'       : Classic document/FAQ chunked RAG
// 'wiki-qa'      : Wiki graph navigation Q&A
// 'hybrid-rag-wiki': Wiki + chunked hybrid retrieval
// 'custom'       : Fully custom (no preset applied)
export type AgentType = 'rag-qa' | 'wiki-qa' | 'hybrid-rag-wiki' | 'data-analysis' | 'custom';

export interface QuestionSuggestionConfig {
  starters: {
    enabled: boolean;
    mode: 'curated' | 'knowledge' | 'hybrid';
    items: string[];
    count: number;
  };
  follow_ups: {
    enabled: boolean;
    mode: 'generated' | 'knowledge' | 'hybrid';
    count: number;
    model_id?: string;
    additional_instruction?: string;
    categories: Array<'clarify' | 'deepen' | 'action'>;
    max_context_turns: number;
    suppress_on_fallback: boolean;
    suppress_when_answer_asks_question: boolean;
    knowledge_fallback: boolean;
    allow_regenerate: boolean;
  };
}

export interface CustomAgentConfig {
  // ===== Basic Settings =====
  agent_mode?: 'quick-answer' | 'smart-reasoning';  // Run mode: quick-answer=RAG mode, smart-reasoning=ReAct Agent mode
  // Type preset under smart reasoning mode, used to apply the "system prompt + tools + KB compatibility" combination with one click
  // Only effective when agent_mode === 'smart-reasoning'; ignored in quick-answer mode
  agent_type?: AgentType;
  system_prompt?: string;           // Unified system prompt (uses the {{web_search_status}} placeholder to dynamically control behavior)
  system_prompt_id?: string;        // Referenced prompt template ID (presets populate this field)
  context_template?: string;        // Context template (normal mode)

  // ===== Model Settings =====
  model_id?: string;
  rerank_model_id?: string;         // ReRank model ID
  temperature?: number;
  max_completion_tokens?: number;   // Maximum number of generated tokens (normal mode)
  thinking?: boolean;                      // Whether to enable thinking mode (for models that support extended thinking)
  citation_enabled?: boolean;        // Whether to output knowledge base/web source citations in the final answer (enabled by default)

  // ===== Agent Mode Settings =====
  max_iterations?: number;          // Maximum number of iterations
  llm_call_timeout?: number;        // LLM call timeout (seconds)
  allowed_tools?: string[];         // Allowed tools
  reflection_enabled?: boolean;     // Whether to enable reflection
  // MCP service selection mode: all=all enabled MCP services, selected=specified services, none=no MCP
  mcp_selection_mode?: 'all' | 'selected' | 'none';
  mcp_services?: string[];          // List of selected MCP service IDs
  // Wait timeout (seconds) when OAuth authorization is triggered during a conversation: the authorization prompt is automatically skipped once it expires.
  // When <=0, the server-side default timeout is used. Only applies to MCP services that use OAuth.
  mcp_auth_wait_timeout?: number;

  // ===== Skills Settings (Agent mode only) =====
  // Skills selection mode: all=all preinstalled, selected=specified, none=none used
  skills_selection_mode?: 'all' | 'selected' | 'none';
  selected_skills?: string[];       // List of selected skill names

  // ===== Knowledge Base Settings =====
  // Knowledge base selection mode: all=all knowledge bases, selected=specified knowledge bases, none=no knowledge base
  kb_selection_mode?: 'all' | 'selected' | 'none';
  knowledge_bases?: string[];
  // Whether to retrieve the knowledge base only on explicit @ mentions (default: false)
  // true: retrieve only when the user explicitly mentions the knowledge base/document via @
  // false: automatically retrieve the knowledge base based on kb_selection_mode
  retrieve_kb_only_when_mentioned?: boolean;

  // ===== Image Upload/Multimodal Settings =====
  image_upload_enabled?: boolean;    // Whether to enable image upload (default: false)
  vlm_model_id?: string;            // VLM model ID (used for image analysis)
  image_storage_provider?: string;   // Image storage provider
  audio_upload_enabled?: boolean;    // Whether to enable audio upload/ASR transcription (default: false)
  asr_model_id?: string;            // ASR model ID (used for audio transcription)
  // Attachment image understanding / scanned document OCR toggle (default: false, enabling it increases parsing time)
  attachment_image_understanding?: boolean;
  // Maximum pages for scanned document OCR (0 = use global default WEKNORA_CHAT_ATTACHMENT_OCR_MAX_PAGES)
  attachment_ocr_max_pages?: number;
  // Maximum time to wait for attachment parsing to complete in a single Q&A turn (seconds, 0 = use global default WEKNORA_CHAT_ATTACHMENT_WAIT_TIMEOUT_SEC)
  attachment_parse_wait_timeout_sec?: number;

  // ===== Chat attachment parser engine strategy =====
  // Select the parser engine by file type; priority: request parser_engine > agent rules > tenant rules > auto
  chat_parser_engine_rules?: { file_types: string[]; engine: string }[];

  // ===== File type restrictions =====
  // Supported file types (e.g. ["csv", "xlsx", "xls"])
  // Empty means all file types are supported
  supported_file_types?: string[];

  // ===== Web search settings =====
  web_search_enabled?: boolean;
  web_search_provider_id?: string;
  web_search_max_results?: number;

  // ===== Multi-turn conversation settings =====
  multi_turn_enabled?: boolean;     // Whether to enable multi-turn conversation
  history_turns?: number;           // Number of historical turns to retain

  // ===== Retrieval strategy settings =====
  embedding_top_k?: number;         // Vector recall TopK
  keyword_threshold?: number;       // Keyword recall threshold
  vector_threshold?: number;        // Vector recall threshold
  rerank_top_k?: number;            // Rerank TopK
  rerank_threshold?: number;        // Rerank threshold

  // ===== Advanced settings (mainly for normal mode) =====
  enable_query_expansion?: boolean; // Whether to enable query expansion
  enable_rewrite?: boolean;         // Whether to enable question rewriting
  rewrite_prompt_system?: string;   // Rewrite system prompt
  rewrite_prompt_user?: string;     // Rewrite user prompt template
  fallback_strategy?: 'fixed' | 'model'; // Fallback strategy
  fallback_response?: string;       // Fixed fallback reply
  fallback_prompt?: string;         // Fallback prompt (used during model generation)
  // Intent prompt: overrides the main system prompt for non-retrieval intents (greetings, chit-chat, etc.)
  intent_prompts?: Record<string, string>;

  // ===== Deprecated fields (kept for compatibility) =====
  welcome_message?: string;
  question_suggestions?: QuestionSuggestionConfig;
}

// Agent
export interface CustomAgent {
  id: string;
  name: string;
  description?: string;
  avatar?: string;
  is_builtin: boolean;
  tenant_id?: number;
  created_by?: string;
  // creator_name is batch-filled by the backend list endpoint, used only for the source badge on list cards.
  creator_name?: string;
  config: CustomAgentConfig;
  created_at?: string;
  updated_at?: string;
}

// Create agent request
export interface CreateAgentRequest {
  name: string;
  description?: string;
  avatar?: string;
  config?: CustomAgentConfig;
}

// Update agent request
export interface UpdateAgentRequest {
  name: string;
  description?: string;
  avatar?: string;
  config?: CustomAgentConfig;
}

// Built-in agent IDs (commonly used reserved constants, for easy reference in code)
export const BUILTIN_QUICK_ANSWER_ID = 'builtin-quick-answer';
export const BUILTIN_SMART_REASONING_ID = 'builtin-smart-reasoning';

// AgentMode constants
export const AGENT_MODE_QUICK_ANSWER = 'quick-answer';
export const AGENT_MODE_SMART_REASONING = 'smart-reasoning';

// Deprecated: Use BUILTIN_QUICK_ANSWER_ID instead
export const BUILTIN_AGENT_NORMAL_ID = BUILTIN_QUICK_ANSWER_ID;
// Deprecated: Use BUILTIN_SMART_REASONING_ID instead
export const BUILTIN_AGENT_AGENT_ID = BUILTIN_SMART_REASONING_ID;

// Get agent list (including built-in agents)
// disabled_own_agent_ids: "My" agent IDs disabled in the conversation dropdown for the current workspace, only affects this workspace
export function listAgents(params?: {
  /**
   * Optional creator filter; mirrors listKnowledgeBases. Built-in agents
   * (is_builtin=true) are always returned regardless of this filter so
   * the conversation dropdown never silently loses quick-answer /
   * smart-reasoning when a user picks "Created by me".
   */
  creator?: 'all' | 'mine' | 'others';
}) {
  const qs = params?.creator && params.creator !== 'all' ? `?creator=${params.creator}` : '';
  return get<{ data: CustomAgent[]; disabled_own_agent_ids?: string[] }>(`/api/v1/agents${qs}`);
}

// Get agent details
export function getAgentById(id: string) {
  return get<{ data: CustomAgent }>(`/api/v1/agents/${id}`);
}

// Create agent
export function createAgent(data: CreateAgentRequest) {
  return post<{ data: CustomAgent }>('/api/v1/agents', data);
}

// Update agent
export function updateAgent(id: string, data: UpdateAgentRequest) {
  return put<{ data: CustomAgent }>(`/api/v1/agents/${id}`, data);
}

// Delete agent
export function deleteAgent(id: string) {
  return del<{ success: boolean }>(`/api/v1/agents/${id}`);
}

// Duplicate agent
export function copyAgent(id: string) {
  return post<{ data: CustomAgent }>(`/api/v1/agents/${id}/copy`);
}

// Determine whether it's a built-in agent (via the agent.is_builtin field or ID prefix)
export function isBuiltinAgent(agentId: string): boolean {
  return agentId.startsWith('builtin-');
}

// Placeholder definition
export interface PlaceholderDefinition {
  name: string;
  label: string;
  description: string;
}

// Placeholder response
export interface PlaceholdersResponse {
  all: PlaceholderDefinition[];
  system_prompt: PlaceholderDefinition[];
  agent_system_prompt: PlaceholderDefinition[];
  context_template: PlaceholderDefinition[];
  rewrite_system_prompt: PlaceholderDefinition[];
  rewrite_prompt: PlaceholderDefinition[];
  fallback_prompt: PlaceholderDefinition[];
}

// Get placeholder definition
export function getPlaceholders() {
  return get<{ data: PlaceholdersResponse }>('/api/v1/agents/placeholders');
}

// ===== Agent type presets =====

// Backend kb_filter structure (see internal/types/agent_type_preset.go)
export interface AgentTypeKBFilter {
  any_of?: string[];   // KB must own at least one of these
  all_of?: string[];   // KB must own all of these
  none_of?: string[];  // KB must own none of these
}

// KB capability tags (JSON of backend types.KBCapabilities)
export interface KBCapabilities {
  vector: boolean;
  keyword: boolean;
  wiki: boolean;
  graph: boolean;
  faq: boolean;
}

// Preset "auto-fill" config payload: only includes fields overridden by the preset; other fields are left untouched
export interface AgentTypePresetConfig {
  system_prompt_id?: string;
  temperature?: number;
  max_iterations?: number;
  allowed_tools?: string[];
  retain_retrieval_history?: boolean;
  faq_priority_enabled?: boolean;
  web_search_enabled?: boolean;
  supported_file_types?: string[];
  kb_selection_mode?: 'all' | 'selected' | 'none';
}

export interface AgentTypePresetI18n {
  label: string;
  description: string;
}

export interface AgentTypePreset {
  id: AgentType;
  i18n: Record<string, AgentTypePresetI18n>;
  config?: AgentTypePresetConfig;     // Empty means "custom" type (no preset)
  kb_filter?: AgentTypeKBFilter;      // Empty means all KBs are selectable
}

// Fetch the list of type presets (for the editor)
export function getAgentTypePresets() {
  return get<{ data: AgentTypePreset[] }>('/api/v1/agents/type-presets');
}

// ===== IM channels =====

export interface IMChannel {
  id: string;
  tenant_id?: number;
  agent_id: string;
  // 'lark' is Feishu's international edition; it shares Feishu's credentials and modes.
  platform: 'wecom' | 'feishu' | 'lark' | 'slack' | 'telegram' | 'dingtalk' | 'mattermost' | 'wechat' | 'qqbot' | 'yunzhijia';
  name: string;
  enabled: boolean;
  mode: 'webhook' | 'websocket' | 'longpoll';
  output_mode: 'stream' | 'full';
  session_mode?: 'user' | 'thread';
  knowledge_base_id?: string;
  credentials: Record<string, any>;
  created_at?: string;
  updated_at?: string;
}

export function listIMChannels(agentId: string) {
  return get<{ data: IMChannel[] }>(`/api/v1/agents/${agentId}/im-channels`);
}

// Tenant-wide overview row. Credentials are intentionally omitted — use
// listIMChannels(agentId) when you need to edit a specific channel.
export interface IMChannelOverview {
  id: string;
  tenant_id: number;
  agent_id: string;
  agent_name: string; // empty string for built-in agents
  platform: IMChannel['platform'];
  name: string;
  enabled: boolean;
  mode: IMChannel['mode'];
  output_mode: IMChannel['output_mode'];
  session_mode?: IMChannel['session_mode'];
  bot_identity: string;
  created_at: string;
  updated_at: string;
}

export function listAllIMChannels() {
  return get<{ data: IMChannelOverview[] }>('/api/v1/im-channels');
}

export function createIMChannel(agentId: string, data: Partial<IMChannel>) {
  return post<{ data: IMChannel }>(`/api/v1/agents/${agentId}/im-channels`, data);
}

export function updateIMChannel(id: string, data: Partial<IMChannel>) {
  return put<{ data: IMChannel }>(`/api/v1/im-channels/${id}`, data);
}

export function deleteIMChannel(id: string) {
  return del<{ success: boolean }>(`/api/v1/im-channels/${id}`);
}

export function toggleIMChannel(id: string) {
  return post<{ data: IMChannel }>(`/api/v1/im-channels/${id}/toggle`);
}

// ===== Suggested questions =====

// Suggested questions
export interface SuggestedQuestion {
  question: string;
  source: 'faq' | 'document' | 'agent_config' | 'wiki';
  knowledge_base_id?: string;
}

// Get agent suggested questions
// Returns suggested questions based on the agent's associated knowledge base scope, used for quick-ask prompts in the frontend chat panel
export function getSuggestedQuestions(
  agentId: string,
  params?: {
    knowledge_base_ids?: string[];
    knowledge_ids?: string[];
    tag_scopes?: Array<{ knowledge_base_id: string; tag_ids: string[] }>;
    limit?: number;
  }
) {
  const query = new URLSearchParams();
  if (params?.knowledge_base_ids?.length) query.set('knowledge_base_ids', params.knowledge_base_ids.join(','));
  if (params?.knowledge_ids?.length) query.set('knowledge_ids', params.knowledge_ids.join(','));
  if (params?.tag_scopes?.length) query.set('tag_scopes', JSON.stringify(params.tag_scopes));
  if (params?.limit) query.set('limit', String(params.limit));
  const qs = query.toString();
  return get<{ data: { questions: SuggestedQuestion[] } }>(`/api/v1/agents/${agentId}/suggested-questions${qs ? '?' + qs : ''}`);
}
// ===== WeChat QR Code Login =====

export interface WeChatQRCodeResult {
  qrcode_url: string;
  qrcode: string;
}

export interface WeChatQRCodeStatus {
  status: 'wait' | 'scaned' | 'confirmed' | 'expired';
  credentials?: {
    bot_token: string;
    ilink_bot_id: string;
    ilink_user_id: string;
  };
  baseurl?: string;
}

export function getWeChatQRCode() {
  return post<{ data: WeChatQRCodeResult }>('/api/v1/wechat/qrcode');
}

export function pollWeChatQRCodeStatus(qrcode: string) {
  return post<{ data: WeChatQRCodeStatus }>('/api/v1/wechat/qrcode/status', { qrcode });
}
