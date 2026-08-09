// src/utils/request.js
import axios from "axios";
import { generateRandomString, MAX_FILE_SIZE_MB } from "./index";
import i18n from '@/i18n'
import { getApiBaseUrl } from './api-base';

const t = (key: string) => i18n.global.t(key)

// API base URL
const BASE_URL = getApiBaseUrl();


// Create Axios instance
const instance = axios.create({
  baseURL: BASE_URL, // Use the configured API base URL
  timeout: 30000, // Request timeout
  headers: {
    "Content-Type": "application/json",
    "X-Request-ID": `${generateRandomString(12)}`,
  },
});

// Current UI language for the Accept-Language header
function getCurrentLanguage(): string {
  return i18n.global.locale?.value || localStorage.getItem('locale') || 'en-US'
}


instance.interceptors.request.use(
  (config) => {
    const existingAuth = config.headers?.Authorization ?? config.headers?.authorization;
    const isEmbedAuth = typeof existingAuth === 'string' && existingAuth.startsWith('Embed ');
    const isEmbedPath = typeof config.url === 'string' && config.url.includes('/api/v1/embed/');

    // Embed channels use the Embed token; don't override it with a local JWT (or the debug page will get a 401)
    if (!isEmbedAuth) {
      const token = localStorage.getItem('weknora_token');
      if (token) {
        config.headers["Authorization"] = `Bearer ${token}`;
      }
    }
    
    // Add user language preference
    config.headers["Accept-Language"] = getCurrentLanguage();
    
    // Add cross-space access header: as long as setSelectedTenant has written an active space,
    // Every request must attach X-Tenant-ID. Early versions would short-circuit
    // "don't attach it when selectedTenantId === defaultTenantId" to reduce header size,
    // but this optimization gets triggered by any code that writes weknora_tenant as the active space (OIDC
    // callback, UserMenu loadUserInfo, router hydrate), causing subsequent requests to
    // silently lose the header, so the frontend "switched" but still actually runs in the home space — turning "only
    // the first batch of requests after switching carries X-Tenant-ID" into a permanent state.
    // The backend's IsTenantAccessible already allows the header to point to the home space (own tenant),
    // so attaching it unconditionally introduces no new risk.
    if (!isEmbedAuth && !isEmbedPath) {
      const selectedTenantId = localStorage.getItem('weknora_selected_tenant_id');
      if (selectedTenantId) {
        config.headers["X-Tenant-ID"] = selectedTenantId;
      }
    }
    
    config.headers["X-Request-ID"] = `${generateRandomString(12)}`;
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// Token refresh flag, prevents multiple requests from refreshing the token simultaneously
let isRefreshing = false;
let failedQueue: Array<{ resolve: Function; reject: Function }> = [];

// Share-link endpoints (/auth/invitations/lookup, /auth/register-by-invite)
// are reachable by anonymous users opening an invite link. A 401 from these
// must surface to the page (e.g. expired token), not trigger the
// refresh-then-redirect-to-login flow (issue #1617). '/auth/register' already
// covers '/auth/register-by-invite' via substring match.
const PUBLIC_AUTH_PATHS = ['/auth/auto-setup', '/auth/login', '/auth/register', '/auth/oidc/', '/auth/invitations/lookup', '/api/v1/embed/'];

function isPublicAuthRequest(url?: string): boolean {
  if (!url) return false;
  return PUBLIC_AUTH_PATHS.some(p => url.includes(p));
}

// Process queued requests
const processQueue = (error: any, token: string | null = null) => {
  failedQueue.forEach(({ resolve, reject }) => {
    if (error) {
      reject(error);
    } else {
      resolve(token);
    }
  });
  
  failedQueue = [];
};

function isEmbedPage(): boolean {
  if (typeof window === 'undefined') return false;
  return window.location.pathname.startsWith('/embed/');
}

function redirectToLogin() {
  if (typeof window === 'undefined') return;
  if (window.location.pathname === '/login') return;
  // Embed channel is authenticated via Embed token; anonymous access should not be kicked to the login page
  if (isEmbedPage()) return;
  window.location.href = '/login';
}

instance.interceptors.response.use(
  (response) => {
    // Handle logic based on business status code
    const { status, data } = response;
    if (status >= 200 && status < 300) {
      return data;
    } else {
      return Promise.reject(data);
    }
  },
  async (error: any) => {
    const originalRequest = error.config;
    
    if (!error.response) {
      return Promise.reject({ message: t('error.networkError') });
    }
    
    // 401s from public endpoints (auto-setup / login / register / oidc) skip the refresh logic and return the error directly
    if ((error.response.status === 401 || error.response.status === 403) && isPublicAuthRequest(originalRequest?.url)) {
      const { status, data } = error.response;
      const msg = typeof data === 'object'
        ? (typeof data?.error === 'string' ? data.error : (data?.error?.message || data?.message))
        : data;
      return Promise.reject({ status, message: msg || t('error.invalidCredentials') });
    }

    // Embed debug page/widget: reject immediately when there's no JWT, don't go through refresh → /login
    if (error.response.status === 401 && isEmbedPage()) {
      const { status, data } = error.response;
      const msg = typeof data === 'object'
        ? (typeof data?.error === 'string' ? data.error : (data?.error?.message || data?.message))
        : data;
      return Promise.reject({ status, message: msg || t('error.invalidCredentials') });
    }

    // If it's a 401 error and not a token refresh request, try refreshing the token
    if (error.response.status === 401 && !originalRequest._retry && !originalRequest.url?.includes('/auth/refresh')) {
      if (isRefreshing) {
        // If a token refresh is already in progress, queue the request
        return new Promise((resolve, reject) => {
          failedQueue.push({ resolve, reject });
        }).then(token => {
          originalRequest.headers['Authorization'] = 'Bearer ' + token;
          return instance(originalRequest);
        }).catch(err => {
          return Promise.reject(err);
        });
      }
      
      originalRequest._retry = true;
      isRefreshing = true;
      
      const refreshToken = localStorage.getItem('weknora_refresh_token');
      
      if (refreshToken) {
        try {
          // Dynamically import the refresh token API
          const { refreshToken: refreshTokenAPI } = await import('../api/auth/index');
          const response = await refreshTokenAPI(refreshToken);
          
          if (response.success && response.data) {
            const { token, refreshToken: newRefreshToken } = response.data;
            
            // Update the token in localStorage
            localStorage.setItem('weknora_token', token);
            localStorage.setItem('weknora_refresh_token', newRefreshToken);
            
            // Update the request header
            originalRequest.headers['Authorization'] = 'Bearer ' + token;
            
            // Process queued requests
            processQueue(null, token);
            
            return instance(originalRequest);
          } else {
            throw new Error(response.message || t('error.tokenRefreshFailed'));
          }
        } catch (refreshError) {
          // Refresh failed, clear all tokens and redirect to the login page
          localStorage.removeItem('weknora_token');
          localStorage.removeItem('weknora_refresh_token');
          localStorage.removeItem('weknora_user');
          localStorage.removeItem('weknora_tenant');
          
          processQueue(refreshError, null);
          
          redirectToLogin();
          
          return Promise.reject(refreshError);
        } finally {
          isRefreshing = false;
        }
      } else {
        // No refresh token, redirect directly to the login page
        localStorage.removeItem('weknora_token');
        localStorage.removeItem('weknora_user');
        localStorage.removeItem('weknora_tenant');
        
        redirectToLogin();
        
        return Promise.reject({ message: t('error.pleaseRelogin') });
      }
    }
    
    // Handle Nginx 413 Request Entity Too Large
    if (error.response.status === 413) {
      return Promise.reject({ 
        status: 413, 
        message: i18n.global.t('error.fileSizeExceeded', { size: MAX_FILE_SIZE_MB }),
        success: false
      });
    }

    const { status, data } = error.response;
    // Throw the HTTP status code along with it, so upper layers can easily detect scenarios like 401
    // Backend response format: { success: false, error: { code, message, details } }
    // Extract error.message as the top-level message, so the frontend can conveniently access it via error?.message
    let errorMessage: string | undefined;
    if (typeof data === 'object') {
      if (typeof data?.error === 'string') {
        errorMessage = data.error;
      } else if (data?.error?.message) {
        errorMessage = data.error.message;
      } else {
        errorMessage = data?.message;
      }
    } else if (typeof data === 'string') {
      errorMessage = data;
    }
    return Promise.reject({ 
      status, 
      message: errorMessage,
      ...(typeof data === 'object' ? data : {}) 
    });
  }
);

export function get<T = any>(url: string, config?: any): Promise<T> {
  return instance.get<T>(url, config) as unknown as Promise<T>;
}

export async function getDown(url: string): Promise<Blob> {
  const res = await instance.get<Blob>(url, {
    responseType: "blob",
  }) as unknown as Blob;
  return res
}

export function postUpload(
  url: string,
  data = {},
  onUploadProgress?: (progressEvent: any) => void,
  config: any = {},
): Promise<any> {
  return instance.post(url, data, {
    ...config,
    headers: {
      "Content-Type": "multipart/form-data",
      "X-Request-ID": `${generateRandomString(12)}`,
      ...(config.headers || {}),
    },
    onUploadProgress: onUploadProgress || config.onUploadProgress,
  }) as unknown as Promise<any>;
}

export function postChat<T = any>(url: string, data = {}): Promise<T> {
  return instance.post(url, data, {
    headers: {
      "Content-Type": "text/event-stream;charset=utf-8",
      "X-Request-ID": `${generateRandomString(12)}`,
    },
  }) as unknown as Promise<T>;
}

export function post<T = any>(url: string, data = {}, config?: any): Promise<T> {
  return instance.post<T>(url, data, config) as unknown as Promise<T>;
}

export function put<T = any>(url: string, data = {}, config?: any): Promise<T> {
  return instance.put<T>(url, data, config) as unknown as Promise<T>;
}

export function del<T = any>(url: string, data?: any): Promise<T> {
  return instance.delete<T>(url, { data }) as unknown as Promise<T>;
}
