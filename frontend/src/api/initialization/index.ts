import { get, post, put } from '../../utils/request';
import i18n from '@/i18n'

const t = (key: string) => i18n.global.t(key)

// GET /initialization/config/:kbId exposes credential presence, not values.
export interface ModelCredentialStatus {
    apiKey?: boolean;
}

export interface COSCredentialStatus {
    secretId?: boolean;
    secretKey?: boolean;
}

// Initial config data type
export interface InitializationConfig {
    llm: {
        source: string;
        modelName: string;
        baseUrl?: string;
        /** @deprecated Use credentials.apiKey from GET responses */
        apiKey?: string;
        credentials?: ModelCredentialStatus;
    };
    embedding: {
        source: string;
        modelName: string;
        baseUrl?: string;
        /** @deprecated Use credentials.apiKey from GET responses */
        apiKey?: string;
        dimension?: number; // Add embedding dimension field
        credentials?: ModelCredentialStatus;
    };
    rerank: {
        modelName: string;
        baseUrl: string;
        /** @deprecated Use credentials.apiKey from GET responses */
        apiKey?: string;
        enabled: boolean;
        credentials?: ModelCredentialStatus;
    };
    multimodal: {
        enabled: boolean;
        storageType: 'cos' | 'minio';
        vlm?: {
            modelName: string;
            baseUrl: string;
            /** @deprecated Use credentials.apiKey from GET responses */
            apiKey?: string;
            interfaceType?: string; // "ollama" or "openai"
            credentials?: ModelCredentialStatus;
        };
        cos?: {
            region: string;
            bucketName: string;
            appId: string;
            pathPrefix?: string;
            /** @deprecated Use credentials from GET responses */
            secretId?: string;
            /** @deprecated Use credentials from GET responses */
            secretKey?: string;
            credentials?: COSCredentialStatus;
        };
        minio?: {
            bucketName: string;
            pathPrefix?: string;
        };
    };
    documentSplitting: {
        chunkSize: number;
        chunkOverlap: number;
        separators: string[];
        // Adaptive chunking strategy. Empty / "legacy" = classic recursive splitter.
        // "auto" lets the backend profiler pick a tier; "heading" / "heuristic"
        // pin the tier explicitly. See backend chunker package for details.
        strategy?: string;
        // Cap chunk size in approx tokens. 0 = char-based budget only.
        tokenLimit?: number;
        // Language hints for heuristic patterns ("de", "en", "zh"). Empty = auto-detect.
        languages?: string[];
    };
    // Frontend-only hint for storage selection UI
    storageType?: 'cos' | 'minio';
    nodeExtract: {
        enabled: boolean,
        text: string,
        tags: string[],
        nodes: Node[],
        relations: Relation[]
    }
}

// Download task status type
export interface DownloadTask {
    id: string;
    modelName: string;
    status: 'pending' | 'downloading' | 'completed' | 'failed';
    progress: number;
    message: string;
    startTime: string;
    endTime?: string;
}

// Simplified knowledge base config update interface (model ID only)
export interface KBModelConfigRequest {
    llmModelId: string
    embeddingModelId: string
    vlm_config?: {
        enabled: boolean
        model_id?: string
        description_language?: string
        custom_instructions?: string
    }
    asr_config?: {
        enabled: boolean
        model_id?: string
        language?: string
    }
    documentSplitting: {
        chunkSize: number
        chunkOverlap: number
        separators: string[]
        parserEngineRules?: {
            file_types: string[]
            engine: string
            xlsx_first_row_as_header?: boolean
        }[]
        enableParentChild?: boolean
        parentChunkSize?: number
        childChunkSize?: number
        // Adaptive chunking strategy ("auto" | "heading" | "heuristic" | "legacy").
        // The backend uses pointer-based DTOs for these three fields:
        // - undefined / not set in payload → no change on server
        // - "" / 0 / [] explicitly sent     → clears the value
        // Send the field whenever the user has opened the editor — even
        // empty values — so the user can always reset back to defaults.
        strategy?: string
        // Approximate token budget per chunk; 0 = char-based.
        tokenLimit?: number
        // Language hints for heuristic patterns. Empty array = auto-detect.
        languages?: string[]
        tableMetadataInstructions?: string
    }
    multimodal: {
        enabled: boolean
    }
    /** Storage engine selection: "local" | "minio" | "cos" | "obs", etc., affects document upload and in-document image storage */
    storageBackendId?: string
    storageProvider?: string
    nodeExtract: {
        enabled: boolean
        text: string
        tags: string[]
        nodes: Node[]
        relations: Relation[]
        customInstructions?: string
    }
    questionGeneration?: {
        enabled: boolean
        questionCount: number
        customInstructions?: string
    }
}

export function updateKBConfig(kbId: string, config: KBModelConfigRequest): Promise<any> {
    return new Promise((resolve, reject) => {
        console.log('Starting KB config update (simplified)...', kbId, config);
        put(`/api/v1/initialization/config/${kbId}`, config)
            .then((response: any) => {
                console.log('KB config update completed', response);
                resolve(response);
            })
            .catch((error: any) => {
                console.error('Failed to update KB config:', error);
                reject(error.error || error);
            });
    });
}

// Update config by knowledge base ID (legacy, kept for compatibility)
export function initializeSystemByKB(kbId: string, config: InitializationConfig): Promise<any> {
    return new Promise((resolve, reject) => {
        console.log('Starting KB config update...', kbId, config);
        post(`/api/v1/initialization/initialize/${kbId}`, config)
            .then((response: any) => {
                console.log('KB config update completed', response);
                resolve(response);
            })
            .catch((error: any) => {
                console.error('Failed to update KB config:', error);
                reject(error.error || error);
            });
    });
}

// Check Ollama service status
export function checkOllamaStatus(): Promise<{ available: boolean; version?: string; error?: string; baseUrl?: string }> {
    return new Promise((resolve, reject) => {
        get('/api/v1/initialization/ollama/status')
            .then((response: any) => {
                resolve(response.data || { available: false });
            })
            .catch((error: any) => {
                console.error('Failed to check Ollama status:', error);
                resolve({ available: false, error: error.message || t('error.initialization.checkFailed') });
            });
    });
}

// Ollama model detail info interface
export interface OllamaModelInfo {
    name: string;
    size: number;
    digest: string;
    modified_at: string;
}

// List installed Ollama models (with details)
export function listOllamaModels(): Promise<OllamaModelInfo[]> {
    return new Promise((resolve, reject) => {
        get('/api/v1/initialization/ollama/models')
            .then((response: any) => {
                resolve((response.data && response.data.models) || []);
            })
            .catch((error: any) => {
                console.error('Failed to list Ollama models:', error);
                resolve([]);
            });
    });
}

// Check Ollama model status
export function checkOllamaModels(models: string[]): Promise<{ models: Record<string, boolean> }> {
    return new Promise((resolve, reject) => {
        post('/api/v1/initialization/ollama/models/check', { models })
            .then((response: any) => {
                resolve(response.data || { models: {} });
            })
            .catch((error: any) => {
                console.error('Failed to check Ollama models:', error);
                reject(error);
            });
    });
}

// Start Ollama model download (async)
export function downloadOllamaModel(modelName: string): Promise<{ taskId: string; modelName: string; status: string; progress: number }> {
    return new Promise((resolve, reject) => {
        post('/api/v1/initialization/ollama/models/download', { modelName })
            .then((response: any) => {
                resolve(response.data || { taskId: '', modelName, status: 'failed', progress: 0 });
            })
            .catch((error: any) => {
                console.error('Failed to start Ollama model download:', error);
                reject(error);
            });
    });
}

// Query download progress
export function getDownloadProgress(taskId: string): Promise<DownloadTask> {
    return new Promise((resolve, reject) => {
        get(`/api/v1/initialization/ollama/download/progress/${taskId}`)
            .then((response: any) => {
                resolve(response.data);
            })
            .catch((error: any) => {
                console.error('Failed to get download progress:', error);
                reject(error);
            });
    });
}

// Get all download tasks
export function listDownloadTasks(): Promise<DownloadTask[]> {
    return new Promise((resolve, reject) => {
        get('/api/v1/initialization/ollama/download/tasks')
            .then((response: any) => {
                resolve(response.data || []);
            })
            .catch((error: any) => {
                console.error('Failed to list download tasks:', error);
                reject(error);
            });
    });
}


export function getCurrentConfigByKB(kbId: string): Promise<InitializationConfig & { hasFiles: boolean }> {
    return new Promise((resolve, reject) => {
        get(`/api/v1/initialization/config/${kbId}`)
            .then((response: any) => {
                resolve(response.data || {});
            })
            .catch((error: any) => {
                console.error('Failed to get KB config:', error);
                reject(error);
            });
    });
}

// Common optional parameters shared by all "test connection" interfaces.
// customHeaders / extraConfig / interfaceType correspond to fields of the same name in the backend's ModelTestRequest,
// and are passed through to the actual model assembly process, ensuring the test connection and production calls follow exactly the same path.
interface BaseModelTestPayload {
    customHeaders?: Record<string, string>;
    extraConfig?: Record<string, string>;
    interfaceType?: string;
    /** Second key segment (e.g. LKEAP Rerank's Tencent Cloud SecretKey) */
    appSecret?: string;
}

// Check remote API model
export function checkRemoteModel(modelConfig: {
    modelName: string;
    baseUrl: string;
    apiKey?: string;
    provider?: string;
    // Pass modelId when editing an existing model; the backend will automatically bring the apiKey out from storage
    // (The frontend no longer echoes plaintext keys, so test connection must use this backfill path)
    modelId?: string;
} & BaseModelTestPayload): Promise<{
    available: boolean;
    message?: string;
}> {
    return new Promise((resolve, reject) => {
        post('/api/v1/initialization/remote/check', modelConfig)
            .then((response: any) => {
                resolve(response.data || {});
            })
            .catch((error: any) => {
                console.error('Failed to check remote model:', error);
                reject(error);
            });
    });
}

// Test whether an Embedding model (local/remote) is available
export function testEmbeddingModel(modelConfig: {
    source: 'local' | 'remote';
    modelName: string;
    baseUrl?: string;
    apiKey?: string;
    dimension?: number;
    supportsDimensionOverride?: boolean;
    provider?: string;
    modelId?: string;
} & BaseModelTestPayload): Promise<{ available: boolean; message?: string; dimension?: number }> {
    return new Promise((resolve, reject) => {
        post('/api/v1/initialization/embedding/test', modelConfig)
            .then((response: any) => {
                resolve(response.data || {});
            })
            .catch((error: any) => {
                console.error('Failed to test Embedding model:', error);
                reject(error);
            });
    });
}


export function checkRerankModel(modelConfig: {
    modelName: string;
    baseUrl: string;
    apiKey?: string;
    provider?: string;
    modelId?: string;
} & BaseModelTestPayload): Promise<{
    available: boolean;
    message?: string;
}> {
    return new Promise((resolve, reject) => {
        post('/api/v1/initialization/rerank/check', modelConfig)
            .then((response: any) => {
                resolve(response.data || {});
            })
            .catch((error: any) => {
                console.error('Failed to check Rerank model:', error);
                reject(error);
            });
    });
}

// Check ASR model connection (test via the /v1/audio/transcriptions endpoint)
export function checkASRModel(modelConfig: {
    modelName: string;
    baseUrl: string;
    apiKey?: string;
    provider?: string;
    modelId?: string;
} & BaseModelTestPayload): Promise<{
    available: boolean;
    message?: string;
}> {
    return new Promise((resolve, reject) => {
        post('/api/v1/initialization/asr/check', modelConfig)
            .then((response: any) => {
                resolve(response.data || {});
            })
            .catch((error: any) => {
                console.error('Failed to check ASR model:', error);
                reject(error);
            });
    });
}

export function testMultimodalFunction(testData: {
    image: File;
    vlm_model: string;
    vlm_base_url: string;
    vlm_api_key?: string;
    vlm_interface_type?: string;
    storage_type?: 'cos' | 'minio';
    // COS optional fields (required only when storage_type === 'cos')
    cos_secret_id?: string;
    cos_secret_key?: string;
    cos_region?: string;
    cos_bucket_name?: string;
    cos_app_id?: string;
    cos_path_prefix?: string;
    // MinIO optional fields
    minio_bucket_name?: string;
    minio_path_prefix?: string;
    chunk_size: number;
    chunk_overlap: number;
    separators: string[];
}): Promise<{
    success: boolean;
    caption?: string;
    ocr?: string;
    processing_time?: number;
    message?: string;
}> {
    return new Promise((resolve, reject) => {
        const formData = new FormData();
        formData.append('image', testData.image);
        formData.append('vlm_model', testData.vlm_model);
        formData.append('vlm_base_url', testData.vlm_base_url);
        if (testData.vlm_api_key) {
            formData.append('vlm_api_key', testData.vlm_api_key);
        }
        if (testData.vlm_interface_type) {
            formData.append('vlm_interface_type', testData.vlm_interface_type);
        }
        if (testData.storage_type) {
            formData.append('storage_type', testData.storage_type);
        }
        // Append COS fields only when storage_type is COS
        if (testData.storage_type === 'cos') {
            if (testData.cos_secret_id) formData.append('cos_secret_id', testData.cos_secret_id);
            if (testData.cos_secret_key) formData.append('cos_secret_key', testData.cos_secret_key);
            if (testData.cos_region) formData.append('cos_region', testData.cos_region);
            if (testData.cos_bucket_name) formData.append('cos_bucket_name', testData.cos_bucket_name);
            if (testData.cos_app_id) formData.append('cos_app_id', testData.cos_app_id);
            if (testData.cos_path_prefix) formData.append('cos_path_prefix', testData.cos_path_prefix);
        }
        // MinIO fields
        if (testData.minio_bucket_name) formData.append('minio_bucket_name', testData.minio_bucket_name);
        if (testData.minio_path_prefix) formData.append('minio_path_prefix', testData.minio_path_prefix);
        formData.append('chunk_size', testData.chunk_size.toString());
        formData.append('chunk_overlap', testData.chunk_overlap.toString());
        formData.append('separators', JSON.stringify(testData.separators));

        // Get auth token
        const token = localStorage.getItem('weknora_token');
        const headers: Record<string, string> = {};
        if (token) {
            headers['Authorization'] = `Bearer ${token}`;
        }

        // Cross-space access request header: attach directly, avoid short-circuiting "selectedTenantId
        // === defaultTenantId then don't attach" causing the header to silently drop in some edge cases.
        // Consistent behavior with utils/request.ts and api/chat/streame.ts.
        const selectedTenantId = localStorage.getItem('weknora_selected_tenant_id');
        if (selectedTenantId) {
            headers['X-Tenant-ID'] = selectedTenantId;
        }

        // Use native fetch because FormData needs to be sent
        fetch('/api/v1/initialization/multimodal/test', {
            method: 'POST',
            headers,
            body: formData
        })
            .then(response => response.json())
            .then((data: any) => {
                if (data.success) {
                    resolve(data.data || {});
                } else {
                    resolve({ success: false, message: data.message || t('error.initialization.testFailed') });
                }
            })
            .catch((error: any) => {
                console.error('Failed multimodal test:', error);
                reject(error);
            });
    });
}

// Text content relation extraction interface
export interface TextRelationExtractionRequest {
    text: string;
    tags: string[];
    model_id: string;
}

export interface Node {
    name: string;
    attributes: string[];
}

export interface Relation {
    node1: string;
    node2: string;
    type: string;
}

export interface TextRelationExtractionResponse {
    nodes: Node[];
    relations: Relation[];
}

// Text content relation extraction
export function extractTextRelations(request: TextRelationExtractionRequest): Promise<TextRelationExtractionResponse> {
    return new Promise((resolve, reject) => {
        post('/api/v1/initialization/extract/text-relation', request, { timeout: 60000 })
            .then((response: any) => {
                resolve(response.data || { nodes: [], relations: [] });
            })
            .catch((error: any) => {
                console.error('Failed to extract text relations:', error);
                reject(error);
            });
    });
}

export interface FabriTextRequest {
    tags: string[];
    model_id: string;
}

export interface FabriTextResponse {
    text: string;
}

// Text content generation
export function fabriText(request: FabriTextRequest): Promise<FabriTextResponse> {
    return new Promise((resolve, reject) => {
        post('/api/v1/initialization/extract/fabri-text', request)
            .then((response: any) => {
                resolve(response.data || { text: '' });
            })
            .catch((error: any) => {
                console.error('Failed to generate text:', error);
                reject(error);
            });
    });
}

export interface FabriTagRequest {
}

export interface FabriTagResponse {
    tags: string[];
}

// Tag generation
export function fabriTag(request: FabriTagRequest): Promise<FabriTagResponse> {
    return new Promise((resolve, reject) => {
        post('/api/v1/initialization/extract/fabri-tag', request)
            .then((response: any) => {
                resolve(response.data || { tags: [] as string[] });
            })
            .catch((error: any) => {
                console.error('Failed to generate tags:', error);
                reject(error);
            });
    });
}

// Model provider info type
export interface ModelProviderOption {
    value: string;        // provider identifier
    label: string;        // Display name
    description: string;  // Description
    defaultUrls: Record<string, string>;  // Default URL by model type
    modelTypes: string[]; // Supported model types
}

// Get model provider list
export function listModelProviders(modelType?: string): Promise<ModelProviderOption[]> {
    return new Promise((resolve, reject) => {
        const url = modelType
            ? `/api/v1/models/providers?model_type=${encodeURIComponent(modelType)}`
            : '/api/v1/models/providers';
        get(url)
            .then((response: any) => {
                resolve(response.data || []);
            })
            .catch((error: any) => {
                console.error('Failed to list model providers:', error);
                resolve([]); // Return empty array on failure, frontend can fall back to defaults
            });
    });
}
