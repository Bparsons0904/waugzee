import {
  useQuery,
  useMutation,
  useQueryClient,
  UseQueryOptions,
  UseMutationOptions,
  UseQueryResult,
  UseMutationResult,
} from "@tanstack/solid-query";
// import { api, ApiClientError } from "./api";
import { Accessor } from "solid-js";
import { useToast } from "../context/ToastContext";
import { AxiosRequestConfig } from "axios";
import { api } from "./api";

// Enhanced query options
export interface ApiQueryOptions<T>
  extends Omit<UseQueryOptions<T>, "queryKey" | "queryFn"> {
  enabled?: boolean | Accessor<boolean>;
}

// Enhanced mutation options with common patterns
export interface ApiMutationOptions<T, V>
  extends Omit<UseMutationOptions<T, Error, V>, "mutationFn"> {
  invalidateQueries?: readonly (readonly unknown[])[];
  successMessage?: string | ((data: T, variables: V) => string);
  errorMessage?: string | ((error: Error) => string);
  onSuccess?: (data: T, variables: V, context: unknown) => void;
  onError?: (error: Error, variables: V, context: unknown) => void;
}

// Generic query hook
export function useApiQuery<T>(
  queryKey: readonly unknown[],
  url: string,
  config?: AxiosRequestConfig,
  options?: ApiQueryOptions<T>,
): UseQueryResult<T, Error> {
  return useQuery(() => ({
    queryKey,
    queryFn: () => api.get<T>(url, config),
    // TODO: Move these settings to the global config and reenable
    refetchOnWindowFocus: false,
    // retry: (failureCount, error) => {
    //   // Don't retry on client errors (4xx)
    //   if (error instanceof ApiClientError && error.status && error.status >= 400 && error.status < 500) {
    //     return false;
    //   }
    //   // Retry up to 3 times for network errors and server errors
    //   return failureCount < 3;
    // },
    staleTime: 5 * 60 * 1000, // 5 minutes
    ...options,
  }));
}

// Generic mutation hook
export function useApiMutation<T, V = unknown>(
  method: "POST" | "PUT" | "PATCH" | "DELETE",
  url: string | ((variables: V) => string), // Support dynamic URLs
  config?: AxiosRequestConfig,
  options?: ApiMutationOptions<T, V>,
): UseMutationResult<T, Error, V> {
  const queryClient = useQueryClient();
  const toast = useToast();

  return useMutation(() => ({
    mutationFn: (variables: V) => {
      const requestUrl = typeof url === "function" ? url(variables) : url;

      switch (method) {
        case "POST":
          return api.post<T>(requestUrl, variables, config);
        case "PUT":
          return api.put<T>(requestUrl, variables, config);
        case "PATCH":
          return api.patch<T>(requestUrl, variables, config);
        case "DELETE":
          return api.delete<T>(requestUrl, config);
        default:
          throw new Error(`Unsupported method: ${method}`);
      }
    },
    onSuccess: (data, variables, context) => {
      // Invalidate specified queries
      if (options?.invalidateQueries) {
        options.invalidateQueries.forEach((queryKey) => {
          queryClient.invalidateQueries({ queryKey });
        });
      }

      // Handle success toast
      if (options?.successMessage) {
        const message =
          typeof options.successMessage === "function"
            ? options.successMessage(data, variables)
            : options.successMessage;
        toast.showSuccess(message);
      }

      // Call original onSuccess
      options?.onSuccess?.(data, variables, context);
    },
    onError: (error, variables, context) => {
      // Handle error toast
      if (options?.errorMessage) {
        const message =
          typeof options.errorMessage === "function"
            ? options.errorMessage(error)
            : options.errorMessage;
        toast.showError(message);
      }

      // Call original onError
      options?.onError?.(error, variables, context);
    },
    ...options,
  }));
}

// Convenience hooks
export function useApiGet<T>(
  queryKey: readonly unknown[],
  url: string,
  params?: Record<string, unknown>,
  options?: ApiQueryOptions<T>,
) {
  const config = params ? { params } : undefined;
  return useApiQuery<T>(queryKey, url, config, options);
}

export function useApiPost<T, V = unknown>(
  url: string | ((variables: V) => string),
  config?: AxiosRequestConfig,
  options?: ApiMutationOptions<T, V>,
) {
  return useApiMutation<T, V>("POST", url, config, options);
}

export function useApiPut<T, V = unknown>(
  url: string | ((variables: V) => string),
  config?: AxiosRequestConfig,
  options?: ApiMutationOptions<T, V>,
) {
  return useApiMutation<T, V>("PUT", url, config, options);
}

export function useApiPatch<T, V = unknown>(
  url: string | ((variables: V) => string),
  config?: AxiosRequestConfig,
  options?: ApiMutationOptions<T, V>,
) {
  return useApiMutation<T, V>("PATCH", url, config, options);
}

export function useApiDelete<T>(
  url: string,
  config?: AxiosRequestConfig,
  options?: ApiMutationOptions<T, void>,
) {
  return useApiMutation<T, void>("DELETE", url, config, options);
}

// Paginated query hook for common pagination pattern
export function useApiPaginatedQuery<T>(
  baseQueryKey: readonly unknown[],
  url: string,
  page: number,
  limit: number = 10,
  additionalParams?: Record<string, unknown>,
  options?: ApiQueryOptions<T>,
) {
  const queryKey = [
    ...baseQueryKey,
    "paginated",
    { page, limit, ...additionalParams },
  ];
  const params = { page, limit, ...additionalParams };

  return useApiGet<T>(queryKey, url, params, options);
}

// Search hook with enabled condition
export function useApiSearch<T>(
  baseQueryKey: readonly unknown[],
  url: string,
  searchQuery: Accessor<string>,
  minLength: number = 3,
  options?: ApiQueryOptions<T>,
) {
  const queryKey = [...baseQueryKey, "search", searchQuery()];

  return useApiGet<T>(
    queryKey,
    url,
    { q: searchQuery() },
    {
      enabled: () => searchQuery().length >= minLength,
      ...options,
    },
  );
}
