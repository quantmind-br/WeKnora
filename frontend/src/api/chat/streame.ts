import { fetchEventSource } from '@microsoft/fetch-event-source';
import { ref, onUnmounted } from 'vue';
import { generateRandomString } from '@/utils/index';
import i18n from '@/i18n';
import { getApiBaseUrl } from '@/utils/api-base';
import {
  sanitizeStreamRequestBody,
  type StreamRequestMeta,
} from '@/utils/chatRequestDebug';



interface StreamOptions {
  // Request method (defaults to POST)
  method?: 'GET' | 'POST'
  // Request headers
  headers?: Record<string, string>
  // Request body auto-serialization
  body?: Record<string, any>
  // Streaming render interval (ms)
  chunkInterval?: number
}

export function useStream() {
  // Reactive state
  const output = ref('')              // Displayed content
  const isStreaming = ref(false)      // Stream status
  const isLoading = ref(false)        // Initial load
  const error = ref<string | null>(null)// Error message
  const lastStreamRequest = ref<StreamRequestMeta | null>(null)
  let controller = new AbortController()
  let streamGeneration = 0

  // Streaming render buffer
  let buffer: string[] = []
  let renderTimer: number | null = null

  // Start streaming request
  const startStream = async (params: { session_id: any; query: any; knowledge_base_ids?: string[]; knowledge_ids?: string[]; tag_ids?: string[]; agent_enabled?: boolean; agent_id?: string; agent_source_tenant_id?: string | number; web_search_enabled?: boolean; summary_model_id?: string; mcp_service_ids?: string[]; skill_names?: string[]; mentioned_items?: Array<{id: string; name: string; type: string; kb_type?: string; kb_id?: string; kb_name?: string; service_id?: string; skill_name?: string}>; images?: Array<{data: string}>; attachment_uploads?: Array<{data: string; file_name: string; file_size: number}>; attachment_ids?: string[]; suggestion_attribution?: { suggestion_set_id: string; question_id: string }; method: string; url: string; embed_token?: string; embed_session_sig?: string; embed_visitor_id?: string }) => {
    const myGeneration = ++streamGeneration
    // Reset state
    output.value = '';
    error.value = null;
    isStreaming.value = true;
    isLoading.value = true;

    // Get API configuration
    const apiUrl = getApiBaseUrl();
    
    const embedToken = params.embed_token;
    const token = embedToken || localStorage.getItem('weknora_token');
    if (!token) {
      error.value = i18n.global.t('error.tokenNotFound');
      stopStream();
      return;
    }

    // Cross-space access header: as long as setSelectedTenant has written the active space, attach
    // X-Tenant-ID. Earlier versions would short-circuit "don't attach when selectedTenantId ===
    // defaultTenantId" to reduce header size, but any code that writes weknora_tenant
    // as the active space (OIDC sync / UserMenu loadUserInfo / router
    // hydrate) makes the two equal, causing subsequent streaming requests to silently drop the header and fall back to the
    // home space, resulting in the SSE endpoint returning 404. Just attach it directly — the backend's
    // IsTenantAccessible also allows the header to point to the user's own space.
    const selectedTenantId = localStorage.getItem('weknora_selected_tenant_id');
    const tenantIdHeader: string | null = selectedTenantId || null;

    // TTFB instrumentation: record the moment we kick off the request so
    // we can compare it with the first answer chunk we receive from the
    // server. This makes it possible to correlate the frontend-observed
    // latency with the backend "TTFB:first_answer_chunk" log line by
    // matching on X-Request-ID.
    const sentAt = performance.now();
    const requestID = generateRandomString(12);
    let firstAnswerLogged = false;

    try {
      let url =
        params.method == "POST"
          ? `${apiUrl}${params.url}/${params.session_id}`
          : `${apiUrl}${params.url}/${params.session_id}?message_id=${params.query}`;
      console.log(`[TTFB] request:start request_id=${requestID} url=${url} sent_at=${Date.now()}`);
      
      // Prepare POST body with required fields for agent-chat
      // knowledge_base_ids array and agent_enabled can update Session's SessionAgentConfig
      const postBody: any = { 
        query: params.query,
        agent_enabled: params.agent_enabled !== undefined ? params.agent_enabled : true
      };
      // Always include knowledge_base_ids for agent-chat (already validated above)
      if (params.knowledge_base_ids !== undefined && params.knowledge_base_ids.length > 0) {
        postBody.knowledge_base_ids = params.knowledge_base_ids;
      }
      // Include knowledge_ids if provided
      if (params.knowledge_ids !== undefined && params.knowledge_ids.length > 0) {
        postBody.knowledge_ids = params.knowledge_ids;
      }
      // Include agent_id if provided (backend resolves shared agent and tenant from share relation)
      if (params.agent_id) {
        postBody.agent_id = params.agent_id;
      }
      if (params.agent_source_tenant_id) {
        postBody.agent_source_tenant_id = Number(params.agent_source_tenant_id);
      }
      // Include web_search_enabled if provided
      if (params.web_search_enabled !== undefined) {
        postBody.web_search_enabled = params.web_search_enabled;
      }
      // Include summary_model_id if provided (for non-Agent mode)
      if (params.summary_model_id) {
        postBody.summary_model_id = params.summary_model_id;
      }
      // Include mcp_service_ids if provided (for Agent mode)
      if (params.mcp_service_ids !== undefined && params.mcp_service_ids.length > 0) {
        postBody.mcp_service_ids = params.mcp_service_ids;
      }
      if (params.skill_names !== undefined && params.skill_names.length > 0) {
        postBody.skill_names = params.skill_names;
      }
      if (params.tag_ids !== undefined && params.tag_ids.length > 0) {
        postBody.tag_ids = params.tag_ids;
      }
      // Include mentioned_items if provided (for displaying @mentions in chat)
      if (params.mentioned_items !== undefined && params.mentioned_items.length > 0) {
        postBody.mentioned_items = params.mentioned_items;
      }
      // Include images if provided (base64 data URIs for multimodal chat)
      if (params.images !== undefined && params.images.length > 0) {
        postBody.images = params.images;
      }
      // Include attachment_uploads if provided (documents, audio, etc.)
      if (params.attachment_uploads !== undefined && params.attachment_uploads.length > 0) {
        postBody.attachment_uploads = params.attachment_uploads;
      }
	  if (params.attachment_ids !== undefined && params.attachment_ids.length > 0) {
		postBody.attachment_ids = params.attachment_ids;
	  }
      if (params.suggestion_attribution) {
        postBody.suggestion_attribution = params.suggestion_attribution;
      }
      postBody.channel = embedToken ? "embed" : "web";

      lastStreamRequest.value = {
        requestId: requestID,
        url,
        method: params.method,
        body: params.method === 'POST' ? sanitizeStreamRequestBody(postBody) : null,
        sentAt: Date.now(),
      };
      
      await fetchEventSource(url, {
        method: params.method,
        headers: {
          "Content-Type": "application/json",
          "Authorization": embedToken ? `Embed ${embedToken}` : `Bearer ${token}`,
          "Accept-Language": i18n.global.locale?.value || localStorage.getItem('locale') || 'en-US',
          "X-Request-ID": requestID,
          ...(!embedToken && tenantIdHeader ? { "X-Tenant-ID": tenantIdHeader } : {}),
          ...(params.embed_session_sig ? { "X-Embed-Session": params.embed_session_sig } : {}),
          ...(params.embed_visitor_id ? { "X-Embed-Visitor": params.embed_visitor_id } : {}),
        },
        body:
          params.method == "POST"
            ? JSON.stringify(postBody)
            : null,
        signal: controller.signal,
        openWhenHidden: true,

        onopen: async (res) => {
          if (!res.ok) throw new Error(`HTTP ${res.status}`);
          console.log(`[TTFB] response:headers request_id=${requestID} elapsed_ms=${(performance.now() - sentAt).toFixed(1)}`);
          isLoading.value = false;
        },

        onmessage: (ev) => {
          if (myGeneration !== streamGeneration) return
          const parsed = JSON.parse(ev.data);
          // Log first answer chunk for end-to-end TTFB measurement.
          // Filter by event type so non-answer events (references, tool
          // calls, etc.) don't count as the "first token" arrival.
          if (!firstAnswerLogged && (parsed?.response_type === 'answer' || parsed?.type === 'answer')) {
            firstAnswerLogged = true;
            console.log(`[TTFB] response:first_answer request_id=${requestID} elapsed_ms=${(performance.now() - sentAt).toFixed(1)}`);
          }
          buffer.push(parsed); // Store data into buffer
          // Execute custom processing
          if (chunkHandler) {
            chunkHandler(parsed);
          }
        },

        onerror: (err) => {
          throw new Error(`${i18n.global.t('error.streamFailed')}: ${err}`);
        },

        onclose: () => {
          stopStream();
        },
      });
    } catch (err) {
      error.value = err instanceof Error ? err.message : String(err)
      stopStream()
    }
  }

  let chunkHandler: ((data: any) => void) | null = null
  // Register chunk handler
  const onChunk = (handler: (data: any) => void) => {
    chunkHandler = handler
  }


  // Stop stream
  const stopStream = () => {
    streamGeneration++
    controller.abort();
    controller = new AbortController(); // Reset controller (for re-initiating if needed)
    isStreaming.value = false;
    isLoading.value = false;
  }

  // Auto-cleanup on component unmount
  onUnmounted(stopStream)

  return {
    output,          // Displayed content
    isStreaming,     // Whether currently streaming
    isLoading,       // Initial connection state
    error,
    lastStreamRequest,
    onChunk,
    startStream,     // Start stream
    stopStream       // Manually stop
  }
}
