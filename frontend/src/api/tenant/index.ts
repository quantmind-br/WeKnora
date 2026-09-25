import { del, get, post, put } from '@/utils/request'
import i18n from '@/i18n'

const t = (key: string) => i18n.global.t(key)

// Space info endpoint
export interface TenantInfo {
  id: number
  name: string
  description?: string
  status?: string
  business?: string
  storage_quota?: number
  storage_used?: number
  created_at: string
  updated_at: string
}

export type APIPrincipalMode = 'tenant' | 'direct_header' | 'signed_token'

export interface APIPrincipalConfig {
  mode: APIPrincipalMode
  direct_header_name: string
  signed_token_header_name: string
  require_direct_header: boolean
  // The server never returns the plaintext secret; only its presence.
  has_hmac_secret: boolean
}

export interface UpdateAPIPrincipalConfigPayload {
  mode: APIPrincipalMode
  direct_header_name?: string
  signed_token_header_name?: string
  require_direct_header?: boolean
  hmac_secret?: string
}

export interface CreateAPIPrincipalTestTokenPayload {
  external_user_id: string
  expires_in_seconds?: number
}

export interface APIPrincipalTestToken {
  token: string
  header_name: string
  expires_in_seconds: number
  expires_at_unix: number
  external_user_id: string
}

// Bounded per-key grants for non-full-access API keys.
//  - 'retrieve': read/search knowledge-base data within scope
//  - 'chat': run the conversation flow (sessions + agent listing + self identity)
//  - 'read_agents': list/read agents without chat or authoring
//  - 'ingest': write content into allowed knowledge bases (docs/chunks/FAQ/tags/wiki)
//  - 'manage_kbs': manage the KB lifecycle (create/copy/duplicate/update/delete + config)
//  - 'manage_agents': create/update/delete/copy agents
//  - 'message_history': search/read tenant chat-history metadata
//  - 'manage_models': manage tenant model definitions, checks, and credentials
//  - 'manage_mcp_services': manage MCP services, credentials, tool policies, and OAuth state
//  - 'manage_datasources': manage data-source connectors and sync jobs
//  - 'manage_channels': manage embed and IM channels
//  - 'manage_vector_stores': manage vector stores and parser/storage checks
//  - 'manage_web_search': manage web-search providers
//  - 'run_evaluations': run/read evaluation jobs
//  - 'manage_members': manage tenant members and invitations
//  - 'manage_spaces': manage organization/space collaboration
//  - 'manage_tenant_settings': read/update tenant integration settings (API principal mode, headers, tenant KV)
export type TenantAPIKeyCapability =
  | 'retrieve'
  | 'chat'
  | 'read_agents'
  | 'ingest'
  | 'manage_kbs'
  | 'manage_agents'
  | 'message_history'
  | 'manage_models'
  | 'manage_mcp_services'
  | 'manage_datasources'
  | 'manage_channels'
  | 'manage_vector_stores'
  | 'manage_storage_backends'
  | 'manage_web_search'
  | 'run_evaluations'
  | 'manage_members'
  | 'manage_spaces'
  | 'manage_tenant_settings'
  | 'system_tenants_read'
  | 'system_tenants_manage'
  | 'system_settings_read'
  | 'system_settings_manage'
  | 'system_runtime_read'
  | 'system_runtime_manage'
  | 'system_audit_read'

export interface TenantAPIKey {
  id: number
  scope_type?: 'tenant' | 'platform'
  name: string
  api_key: string
  full_access: boolean
  knowledge_base_ids: string[] | null
  capabilities?: TenantAPIKeyCapability[]
  last_used_at?: string
  expires_at?: string
  created_at: string
}

export interface CreatedTenantAPIKey extends TenantAPIKey {
  token?: string
}

export interface CreateTenantAPIKeyPayload {
  name: string
  full_access?: boolean
  knowledge_base_ids?: string[]
  capabilities?: TenantAPIKeyCapability[]
  expires_at_unix?: number
}

export interface UpdateTenantAPIKeyPayload {
  name: string
  full_access: boolean
  knowledge_base_ids: string[]
  capabilities: TenantAPIKeyCapability[]
  expires_at_unix?: number
}

// Search space parameters
export interface SearchTenantsParams {
  keyword?: string
  tenant_id?: number
  page?: number
  page_size?: number
}

// Search space response
export interface SearchTenantsResponse {
  success: boolean
  data?: {
    items: TenantInfo[]
    total: number
    page: number
    page_size: number
  }
  message?: string
}

/**
 * Get the full list of spaces (requires cross-space access permission)
 * @deprecated use searchTenants instead, which supports pagination and search
 */
export async function listAllTenants(): Promise<{ success: boolean; data?: { items: TenantInfo[] }; message?: string }> {
  try {
    const response = await get('/api/v1/tenants/all')
    return response as unknown as { success: boolean; data?: { items: TenantInfo[] }; message?: string }
  } catch (error: any) {
    return {
      success: false,
      message: error.message || t('error.tenant.listFailed')
    }
  }
}

export async function getAPIPrincipalConfig(
  tenantId: number,
): Promise<{ success: boolean; data?: APIPrincipalConfig; message?: string }> {
  try {
    const response = await get(`/api/v1/tenants/${tenantId}/api-principal-config`)
    return response as unknown as { success: boolean; data?: APIPrincipalConfig; message?: string }
  } catch (error: any) {
    return {
      success: false,
      message: error.message || t('error.tenant.getApiPrincipalConfigFailed'),
    }
  }
}

export async function updateAPIPrincipalConfig(
  tenantId: number,
  payload: UpdateAPIPrincipalConfigPayload,
): Promise<{ success: boolean; data?: APIPrincipalConfig; message?: string }> {
  try {
    const response = await put(`/api/v1/tenants/${tenantId}/api-principal-config`, payload)
    return response as unknown as { success: boolean; data?: APIPrincipalConfig; message?: string }
  } catch (error: any) {
    return {
      success: false,
      message: error.message || t('error.tenant.updateApiPrincipalConfigFailed'),
    }
  }
}

export async function createAPIPrincipalTestToken(
  tenantId: number,
  payload: CreateAPIPrincipalTestTokenPayload,
): Promise<{ success: boolean; data?: APIPrincipalTestToken; message?: string }> {
  try {
    const response = await post(`/api/v1/tenants/${tenantId}/api-principal-test-token`, payload)
    return response as unknown as { success: boolean; data?: APIPrincipalTestToken; message?: string }
  } catch (error: any) {
    return {
      success: false,
      message: error.message || t('error.tenant.createApiPrincipalTestTokenFailed'),
    }
  }
}

export async function listTenantAPIKeys(
  tenantId: number,
): Promise<{ success: boolean; data?: TenantAPIKey[]; message?: string }> {
  try {
    const response = await get(`/api/v1/tenants/${tenantId}/api-keys`)
    return response as unknown as { success: boolean; data?: TenantAPIKey[]; message?: string }
  } catch (error: any) {
    return {
      success: false,
      message: error.message || t('error.tenant.listApiKeysFailed'),
    }
  }
}

export async function createTenantAPIKey(
  tenantId: number,
  payload: CreateTenantAPIKeyPayload,
): Promise<{ success: boolean; data?: CreatedTenantAPIKey; message?: string }> {
  try {
    const response = await post(`/api/v1/tenants/${tenantId}/api-keys`, payload)
    return response as unknown as { success: boolean; data?: CreatedTenantAPIKey; message?: string }
  } catch (error: any) {
    return {
      success: false,
      message: error.message || t('error.tenant.createApiKeyFailed'),
    }
  }
}

/** 更新已创建租户 API Key 的授权范围和其他可配置属性。 */
export async function updateTenantAPIKey(
  tenantId: number,
  keyId: number,
  payload: UpdateTenantAPIKeyPayload,
): Promise<{ success: boolean; data?: TenantAPIKey; message?: string }> {
  try {
    const response = await put(`/api/v1/tenants/${tenantId}/api-keys/${keyId}`, payload)
    return response as unknown as { success: boolean; data?: TenantAPIKey; message?: string }
  } catch (error: any) {
    return {
      success: false,
      message: error.message || t('integrations.api.updateApiKeyScopeFailed'),
    }
  }
}

export async function deleteTenantAPIKey(
  tenantId: number,
  keyId: number,
): Promise<{ success: boolean; message?: string }> {
  try {
    const response = await del(`/api/v1/tenants/${tenantId}/api-keys/${keyId}`)
    return response as unknown as { success: boolean; message?: string }
  } catch (error: any) {
    return {
      success: false,
      message: error.message || t('error.tenant.deleteApiKeyFailed'),
    }
  }
}

/**
 * Update space info (currently exposes editing entry points for the name and description fields only).
 * The backend `PUT /tenants/:id` uses pointer fields to distinguish "not passed" from "explicitly empty string"; columns not passed are left
 * unchanged; here too, `name` / `description` are passed selectively as needed, independent of each other.
 * Permission: owner (consistent with the g.Owner() guard in router.go).
 */
export async function updateTenant(
  tenantId: number,
  payload: { name?: string; description?: string },
): Promise<{ success: boolean; data?: TenantInfo; message?: string }> {
  try {
    const response = await put(`/api/v1/tenants/${tenantId}`, payload)
    return response as unknown as { success: boolean; data?: TenantInfo; message?: string }
  } catch (error: any) {
    return {
      success: false,
      message: error.message || t('error.tenant.updateFailed'),
    }
  }
}

/**
 * Delete the current workspace. Permission: owner.
 */
export async function deleteTenant(
  tenantId: number,
): Promise<{ success: boolean; message?: string }> {
  try {
    const response = await del(`/api/v1/tenants/${tenantId}`)
    return response as unknown as { success: boolean; message?: string }
  } catch (error: any) {
    return {
      success: false,
      message: error.message || t('error.tenant.deleteFailed'),
    }
  }
}

/**
 * Create a new workspace (callable by any logged-in user).
 * Backend automatically sets the caller as the new space's Owner and fills in a default storage_quota
 * Waiting on server-side field; the API Key is created manually by the user on the integrations page.
 * Route: POST /api/v1/tenants (g.CrossTenant() is not attached on the router; used for self-service scenarios).
 */
export async function createTenant(
  payload: { name: string; description?: string },
): Promise<{ success: boolean; data?: TenantInfo; message?: string }> {
  try {
    const response = await post('/api/v1/tenants', payload)
    return response as unknown as { success: boolean; data?: TenantInfo; message?: string }
  } catch (error: any) {
    const code = error?.error?.code ?? error?.code
    return {
      success: false,
      message: code === 2005
        ? t('tenant.create.disabled')
        : (error.message || t('error.tenant.createFailed')),
    }
  }
}

/**
 * Search spaces (supports pagination, keyword search, and space ID filtering)
 */
export async function searchTenants(params: SearchTenantsParams = {}): Promise<SearchTenantsResponse> {
  try {
    const queryParams = new URLSearchParams()
    if (params.keyword) {
      queryParams.append('keyword', params.keyword)
    }
    if (params.tenant_id) {
      queryParams.append('tenant_id', String(params.tenant_id))
    }
    if (params.page) {
      queryParams.append('page', String(params.page))
    }
    if (params.page_size) {
      queryParams.append('page_size', String(params.page_size))
    }
    
    const queryString = queryParams.toString()
    const url = `/api/v1/tenants/search${queryString ? '?' + queryString : ''}`
    const response = await get(url)
    return response as unknown as SearchTenantsResponse
  } catch (error: any) {
    return {
      success: false,
      message: error.message || t('error.tenant.searchFailed')
    }
  }
}
