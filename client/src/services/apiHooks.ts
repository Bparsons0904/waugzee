import {
  ADMIN_ENDPOINTS,
  CLEANING_HISTORY_ENDPOINTS,
  HISTORY_ENDPOINTS,
  PLAY_HISTORY_ENDPOINTS,
  RECOMMENDATION_ENDPOINTS,
  STYLUS_ENDPOINTS,
} from "@constants/api.constants";
import type { DownloadStatusResponse, StoredFilesResponse } from "@models/Admin";
import type {
  LogBothRequest,
  LogBothResponse,
  LogCleaningRequest,
  LogCleaningResponse,
  LogPlayRequest,
  LogPlayResponse,
  UpdateCleaningRequest,
  UpdateCleaningResponse,
  UpdatePlayRequest,
  UpdatePlayResponse,
} from "@models/Release";
import type {
  AvailableStylusResponse,
  CreateCustomStylusRequest,
  CreateCustomStylusResponse,
  CreateUserStylusRequest,
  UpdateUserStylusRequest,
} from "@models/Stylus";
import {
  type Query,
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
export interface ApiQueryOptions<T>
  extends Omit<
    UseQueryOptions<T, Error, T, readonly unknown[]>,
    "queryKey" | "queryFn" | "enabled"
  > {
  enabled?: boolean | Accessor<boolean>;
  refetchInterval?: number | false | ((query: Query<T, Error>) => number | false | undefined);
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

  const {
    onSuccess: userOnSuccess,
    onError: userOnError,
    invalidateQueries,
    successMessage,
    errorMessage,
    ...restOptions
  } = options || {};

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
      if (invalidateQueries) {
        invalidateQueries.forEach((queryKey) => {
          queryClient.invalidateQueries({ queryKey });
        });
      }

      // Handle success toast
      if (successMessage) {
        const message =
          typeof successMessage === "function" ? successMessage(data, variables) : successMessage;
        toast.showSuccess(message);
      }

      // Call user's onSuccess callback
      userOnSuccess?.(data, variables, context);
    },
    onError: (error, variables, context) => {
      // Handle error toast
      if (errorMessage) {
        const message = typeof errorMessage === "function" ? errorMessage(error) : errorMessage;
        toast.showError(message);
      }

      // Call user's onError callback
      userOnError?.(error, variables, context);
    },
    ...restOptions,
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

export function useCreateUserStylus(
  options?: ApiMutationOptions<{ success: boolean }, CreateUserStylusRequest>,
) {
  return useApiPost<{ success: boolean }, CreateUserStylusRequest>(
    STYLUS_ENDPOINTS.CREATE,
    undefined,
    {
      invalidateQueries: [["styluses", "user"], ["user"]],
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
      invalidateQueries: [["styluses", "user"], ["styluses", "available"], ["user"]],
      successMessage: "Custom stylus created and added to equipment!",
      errorMessage: "Failed to create custom stylus. Please try again.",
      ...options,
    },
  );
}

export function useUpdateUserStylus(
  options?: ApiMutationOptions<{ success: boolean }, { id: string; data: UpdateUserStylusRequest }>,
) {
  return useApiPut<{ success: boolean }, { id: string; data: UpdateUserStylusRequest }>(
    (variables) => STYLUS_ENDPOINTS.UPDATE(variables.id),
    undefined,
    {
      invalidateQueries: [["styluses", "user"], ["user"]],
      successMessage: "Stylus updated successfully!",
      errorMessage: "Failed to update stylus. Please try again.",
      ...options,
    },
  );
}

export function useDeleteUserStylus(options?: ApiMutationOptions<void, string>) {
  return useApiMutation<void, string>("DELETE", (id) => STYLUS_ENDPOINTS.DELETE(id), undefined, {
    invalidateQueries: [["styluses", "user"], ["user"]],
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

export function useUpdatePlay(
  id: string,
  options?: ApiMutationOptions<UpdatePlayResponse, UpdatePlayRequest>,
) {
  return useApiPut<UpdatePlayResponse, UpdatePlayRequest>(
    PLAY_HISTORY_ENDPOINTS.UPDATE(id),
    undefined,
    {
      successMessage: "Play updated successfully!",
      errorMessage: "Failed to update play. Please try again.",
      ...options,
    },
  );
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

export function useUpdateCleaning(
  id: string,
  options?: ApiMutationOptions<UpdateCleaningResponse, UpdateCleaningRequest>,
) {
  return useApiPut<UpdateCleaningResponse, UpdateCleaningRequest>(
    CLEANING_HISTORY_ENDPOINTS.UPDATE(id),
    undefined,
    {
      successMessage: "Cleaning updated successfully!",
      errorMessage: "Failed to update cleaning. Please try again.",
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

// Combined Play & Cleaning History API Hook
export function useLogBoth(options?: ApiMutationOptions<LogBothResponse, LogBothRequest>) {
  return useApiPost<LogBothResponse, LogBothRequest>(HISTORY_ENDPOINTS.LOG_BOTH, undefined, {
    successMessage: "Play and cleaning logged successfully!",
    errorMessage: "Failed to log play and cleaning. Please try again.",
    ...options,
  });
}

// Admin - Monthly Downloads
export function useDownloadStatus() {
  return useApiQuery<DownloadStatusResponse>(
    ["admin", "downloads", "status"],
    ADMIN_ENDPOINTS.DOWNLOADS_STATUS,
  );
}

export function useTriggerDownload() {
  return useApiPost<void, void>(ADMIN_ENDPOINTS.DOWNLOADS_TRIGGER, undefined, {
    invalidateQueries: [["admin", "downloads", "status"]],
    successMessage: "Download triggered successfully",
    errorMessage: "Failed to trigger download",
  });
}

export function useTriggerReprocess() {
  return useApiPost<void, void>(ADMIN_ENDPOINTS.DOWNLOADS_REPROCESS, undefined, {
    invalidateQueries: [["admin", "downloads", "status"]],
    successMessage: "Reprocessing triggered successfully",
    errorMessage: "Failed to trigger reprocessing",
  });
}

export function useResetDownload() {
  return useApiPost<void, void>(ADMIN_ENDPOINTS.DOWNLOADS_RESET, undefined, {
    invalidateQueries: [["admin", "downloads", "status"]],
    successMessage: "Download reset successfully. You can now trigger a new download.",
    errorMessage: "Failed to reset download",
  });
}

// Admin - File Management
export function useStoredFiles(options?: ApiQueryOptions<StoredFilesResponse>) {
  return useApiQuery<StoredFilesResponse>(
    ["admin", "files", "stored"],
    ADMIN_ENDPOINTS.FILES_LIST,
    undefined,
    options,
  );
}

export function useCleanupFiles(options?: ApiMutationOptions<void, void>) {
  return useApiDelete<void>(ADMIN_ENDPOINTS.FILES_CLEANUP, undefined, {
    invalidateQueries: [["admin", "files", "stored"]],
    successMessage: "Files cleaned up successfully",
    errorMessage: "Failed to cleanup files",
    ...options,
  });
}

// Recommendation API Hooks
export function useMarkRecommendationListened(
  options?: ApiMutationOptions<void, { recommendationId: string }>,
) {
  return useApiPost<void, { recommendationId: string }>(
    (variables) => RECOMMENDATION_ENDPOINTS.MARK_LISTENED(variables.recommendationId),
    undefined,
    {
      invalidateQueries: [["user"]],
      successMessage: "Play logged and recommendation marked as listened!",
      errorMessage: "Failed to mark recommendation as listened",
      ...options,
    },
  );
}
