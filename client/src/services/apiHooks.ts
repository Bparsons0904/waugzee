import {
  CLEANING_HISTORY_ENDPOINTS,
  PLAY_HISTORY_ENDPOINTS,
  STYLUS_ENDPOINTS,
} from "@constants/api.constants";
import type {
  LogCleaningRequest,
  LogCleaningResponse,
  LogPlayRequest,
  LogPlayResponse,
} from "@models/Release";
import type {
  AvailableStylusResponse,
  CreateCustomStylusRequest,
  CreateCustomStylusResponse,
  CreateUserStylusRequest,
  UpdateUserStylusRequest,
  UserStylusesResponse,
} from "@models/Stylus";
import {
  type UseMutationOptions,
  type UseMutationResult,
  type UseQueryOptions,
  type UseQueryResult,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/solid-query";
import type { AxiosRequestConfig } from "axios";
// import { api, ApiClientError } from "./api";
import type { Accessor } from "solid-js";
import { useToast } from "../context/ToastContext";
import { api } from "./api";

// Enhanced query options
export interface ApiQueryOptions<T> extends Omit<UseQueryOptions<T>, "queryKey" | "queryFn"> {
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
  const { enabled, ...restOptions } = options || {};

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
    enabled: typeof enabled === "function" ? enabled() : enabled,
    ...restOptions,
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
  const queryKey = [...baseQueryKey, "paginated", { page, limit, ...additionalParams }];
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

// Stylus API Hooks
export function useAvailableStyluses(options?: ApiQueryOptions<AvailableStylusResponse>) {
  return useApiGet<AvailableStylusResponse>(
    ["styluses", "available"],
    STYLUS_ENDPOINTS.AVAILABLE,
    undefined,
    options,
  );
}

// Claude we should not need this, we already have this is UserDataContext
export function useUserStyluses(options?: ApiQueryOptions<UserStylusesResponse>) {
  return useApiGet<UserStylusesResponse>(
    ["styluses", "user"],
    STYLUS_ENDPOINTS.USER_STYLUSES,
    undefined,
    options,
  );
}

export function useCreateUserStylus(
  options?: ApiMutationOptions<{ success: boolean }, CreateUserStylusRequest>,
) {
  return useApiPost<{ success: boolean }, CreateUserStylusRequest>(
    STYLUS_ENDPOINTS.CREATE,
    undefined,
    {
      invalidateQueries: [["styluses", "user"]],
      successMessage: "Stylus added to equipment successfully!",
      errorMessage: "Failed to add stylus to equipment. Please try again.",
      ...options,
    },
  );
}

export function useCreateCustomStylus(
  options?: ApiMutationOptions<CreateCustomStylusResponse, CreateCustomStylusRequest>,
) {
  return useApiPost<CreateCustomStylusResponse, CreateCustomStylusRequest>(
    STYLUS_ENDPOINTS.CUSTOM,
    undefined,
    {
      invalidateQueries: [
        ["styluses", "user"],
        ["styluses", "available"],
      ],
      successMessage: "Custom stylus created and added to equipment!",
      errorMessage: "Failed to create custom stylus. Please try again.",
      ...options,
    },
  );
}

export function useUpdateUserStylus(
  options?: ApiMutationOptions<{ success: boolean }, { id: string; data: UpdateUserStylusRequest }>,
) {
  return useApiPatch<{ success: boolean }, { id: string; data: UpdateUserStylusRequest }>(
    (variables) => STYLUS_ENDPOINTS.UPDATE(variables.id),
    undefined,
    {
      invalidateQueries: [["styluses", "user"]],
      successMessage: "Stylus updated successfully!",
      errorMessage: "Failed to update stylus. Please try again.",
      ...options,
    },
  );
}

export function useDeleteUserStylus(options?: ApiMutationOptions<void, string>) {
  return useApiMutation<void, string>("DELETE", (id) => STYLUS_ENDPOINTS.DELETE(id), undefined, {
    invalidateQueries: [["styluses", "user"]],
    successMessage: "Stylus removed from equipment successfully!",
    errorMessage: "Failed to remove stylus. Please try again.",
    ...options,
  });
}

// Play History API Hooks
export function useLogPlay(options?: ApiMutationOptions<LogPlayResponse, LogPlayRequest>) {
  return useApiPost<LogPlayResponse, LogPlayRequest>(PLAY_HISTORY_ENDPOINTS.CREATE, undefined, {
    successMessage: "Play logged successfully!",
    errorMessage: "Failed to log play. Please try again.",
    ...options,
  });
}

export function useDeletePlay(options?: ApiMutationOptions<void, string>) {
  return useApiMutation<void, string>(
    "DELETE",
    (id) => PLAY_HISTORY_ENDPOINTS.DELETE(id),
    undefined,
    {
      successMessage: "Play deleted successfully!",
      errorMessage: "Failed to delete play. Please try again.",
      ...options,
    },
  );
}

// Cleaning History API Hooks
export function useLogCleaning(
  options?: ApiMutationOptions<LogCleaningResponse, LogCleaningRequest>,
) {
  return useApiPost<LogCleaningResponse, LogCleaningRequest>(
    CLEANING_HISTORY_ENDPOINTS.CREATE,
    undefined,
    {
      successMessage: "Cleaning logged successfully!",
      errorMessage: "Failed to log cleaning. Please try again.",
      ...options,
    },
  );
}

export function useDeleteCleaning(options?: ApiMutationOptions<void, string>) {
  return useApiMutation<void, string>(
    "DELETE",
    (id) => CLEANING_HISTORY_ENDPOINTS.DELETE(id),
    undefined,
    {
      successMessage: "Cleaning deleted successfully!",
      errorMessage: "Failed to delete cleaning. Please try again.",
      ...options,
    },
  );
}
