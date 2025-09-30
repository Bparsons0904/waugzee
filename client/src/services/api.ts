/**
 * DEPRECATION NOTICE: This file provides low-level API access.
 *
 * ⚠️ DO NOT USE THIS DIRECTLY IN COMPONENTS ⚠️
 *
 * Instead, use TanStack Query hooks from @services/apiHooks:
 * - useApiQuery for GET requests
 * - useApiPut for PUT requests
 * - useApiPost for POST requests
 * - useApiPatch for PATCH requests
 * - useApiDelete for DELETE requests
 *
 * TanStack Query provides:
 * - Automatic caching and cache invalidation
 * - Loading and error states
 * - Optimistic updates
 * - Request deduplication
 * - Retry logic
 *
 * This file should only be used internally by apiHooks.ts or for special cases
 * that cannot be handled by TanStack Query.
 */

import { env } from "@services/env.service";
import axios, { AxiosError, AxiosRequestConfig } from "axios";

// Types
export interface ApiError {
  message: string;
  code?: string;
  details?: Record<string, unknown>;
}

export interface ApiResponse<T = unknown> {
  data?: T;
  error?: ApiError;
  message?: string;
}

export class ApiClientError extends Error {
  constructor(
    message: string,
    public status?: number,
    public code?: string,
    public details?: Record<string, unknown>
  ) {
    super(message);
    this.name = 'ApiClientError';
  }
}

export class NetworkError extends Error {
  constructor(message: string, public originalError?: Error) {
    super(message);
    this.name = 'NetworkError';
  }
}

// Create axios instance
const axiosClient = axios.create({
  baseURL: env.apiUrl + "/api",
  timeout: 10000,
  headers: {
    Accept: "application/json",
    "Content-Type": "application/json",
    "X-Client-Type": "solid",
  },
});

// Token getter function - set by AuthContext to avoid closure issues
let getAuthToken: (() => string | null) | null = null;

export const setTokenGetter = (getter: () => string | null) => {
  getAuthToken = getter;
};

// Request interceptor to add Authorization header
axiosClient.interceptors.request.use(
  (config) => {
    const token = getAuthToken?.();
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => Promise.reject(error),
);

// Response interceptor for error handling
axiosClient.interceptors.response.use(
  (response) => response,
  (error) => Promise.reject(handleApiError(error)),
);

// Error handling
const handleApiError = (error: AxiosError): ApiClientError | NetworkError => {
  if (error.response) {
    // Server responded with error status
    const response = error.response as { data?: { error?: ApiError } };
    const apiError = response.data?.error;

    return new ApiClientError(
      apiError?.message || error.message || "An error occurred",
      error.response.status,
      apiError?.code,
      apiError?.details,
    );
  } else if (error.request) {
    // Request was made but no response received
    return new NetworkError("Network error: No response received", error);
  } else {
    // Something else happened
    return new NetworkError(
      error.message || "An unexpected error occurred",
      error,
    );
  }
};

// Retry logic
interface RetryConfig {
  maxAttempts?: number;
  baseDelayMs?: number;
  maxDelayMs?: number;
  shouldRetry?: (error: Error) => boolean;
}

const defaultRetryConfig: RetryConfig = {
  maxAttempts: 3,
  baseDelayMs: 1000,
  maxDelayMs: 10000,
  shouldRetry: (error: Error) => {
    // Retry on network errors and 5xx server errors
    if (error instanceof NetworkError) return true;
    if (error instanceof ApiClientError) {
      return error.status >= 500 && error.status < 600;
    }
    return false;
  },
};

const sleep = (ms: number): Promise<void> => 
  new Promise(resolve => setTimeout(resolve, ms));

const retryRequest = async <T>(
  fn: () => Promise<T>,
  config: RetryConfig = defaultRetryConfig
): Promise<T> => {
  const { maxAttempts = 3, baseDelayMs = 1000, maxDelayMs = 10000, shouldRetry } = config;
  
  let lastError: Error;

  for (let attempt = 1; attempt <= maxAttempts; attempt++) {
    try {
      return await fn();
    } catch (error) {
      lastError = error as Error;

      if (attempt === maxAttempts || !shouldRetry?.(lastError)) {
        throw lastError;
      }

      const delay = Math.min(baseDelayMs * Math.pow(2, attempt - 1), maxDelayMs);
      await sleep(delay);
    }
  }

  throw lastError!;
};

// Core request function
const request = async <T>(
  method: string,
  url: string,
  data?: unknown,
  config?: AxiosRequestConfig
): Promise<T> => {
  const makeRequest = async (): Promise<T> => {
    const response = await axiosClient.request({
      method,
      url,
      data,
      ...config,
    });
    return response.data;
  };

  return retryRequest(makeRequest);
};

// Typed API methods
export const api = {
  get: <T>(url: string, config?: AxiosRequestConfig): Promise<T> =>
    request<T>("GET", url, undefined, config),

  post: <T>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> =>
    request<T>("POST", url, data, config),

  put: <T>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> =>
    request<T>("PUT", url, data, config),

  patch: <T>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> =>
    request<T>("PATCH", url, data, config),

  delete: <T>(url: string, config?: AxiosRequestConfig): Promise<T> =>
    request<T>("DELETE", url, undefined, config),
};

// Backwards compatibility exports (for gradual migration)
export const apiRequest = request;
export const getApi = api.get;
export const postApi = api.post;
export const putApi = api.put;
export const patchApi = api.patch;
export const deleteApi = api.delete;