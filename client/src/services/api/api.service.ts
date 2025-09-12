import { env } from "@services/env.service";
import axios, { AxiosError, AxiosResponse } from "axios";
import {
  ApiResponse,
  ApiClientError,
  NetworkError,
  RequestConfig,
} from "./apiTypes";
import { retryWithExponentialBackoff, dataRetryConfig } from "@utils/retry.utils";

export const apiClient = axios.create({
  baseURL: env.apiUrl + "/api",
  timeout: 10000,
  headers: {
    Accept: "application/json",
    "Content-Type": "application/json",
    "X-Client-Type": "solid",
  },
});

// Token getter function - will be set by AuthContext to avoid closure issues
let getAuthToken: (() => string | null) | null = null;

// Function to set the token getter (called by AuthContext)
export const setTokenGetter = (getter: () => string | null) => {
  getAuthToken = getter;
};

// Request interceptor to add Authorization header
apiClient.interceptors.request.use(
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
apiClient.interceptors.response.use(
  (response) => response,
  (error) => Promise.reject(handleApiError(error)),
);


// Enhanced error handling function
const handleApiError = (error: AxiosError): ApiClientError | NetworkError => {
  if (error.response) {
    // Server responded with error status
    const response = error.response as AxiosResponse<ApiResponse>;
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

// Generic API request function with retry logic
export const apiRequest = async <T>(
  method: "GET" | "POST" | "PUT" | "PATCH" | "DELETE",
  url: string,
  data?: unknown,
  config?: RequestConfig,
): Promise<T> => {
  const makeRequest = async (): Promise<T> => {
    try {
      const response = await apiClient.request({
        method,
        url,
        data,
        ...config,
      });

      return response.data;
    } catch (error) {
      if (error instanceof ApiClientError || error instanceof NetworkError) {
        throw error;
      }
      throw handleApiError(error as AxiosError);
    }
  };

  // Use retry logic for data requests
  return retryWithExponentialBackoff(makeRequest, dataRetryConfig);
};

// Convenience methods with better typing
export const getApi = async <T>(
  url: string,
  params?: Record<string, unknown>,
): Promise<T> => {
  return apiRequest<T>("GET", url, undefined, { params });
};

export const postApi = async <T, U = unknown>(
  url: string,
  data: U,
  config?: RequestConfig,
): Promise<T> => {
  return apiRequest<T>("POST", url, data, config);
};

export const putApi = async <T, U = unknown>(
  url: string,
  data: U,
  config?: RequestConfig,
): Promise<T> => {
  return apiRequest<T>("PUT", url, data, config);
};

export const patchApi = async <T, U = unknown>(
  url: string,
  data: U,
  config?: RequestConfig,
): Promise<T> => {
  return apiRequest<T>("PATCH", url, data, config);
};

export const deleteApi = async <T>(
  url: string,
  config?: RequestConfig,
): Promise<T> => {
  return apiRequest<T>("DELETE", url, undefined, config);
};
