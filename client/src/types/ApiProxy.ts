// Types for the client-as-proxy pattern for external API requests

export interface ApiRequestPayload {
  requestId: string;
  requestType: string;
  url: string;
  method: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  headers: Record<string, string>;
  callbackService: string;
  callbackEvent: string;
  body?: unknown;
}

export interface ApiResponsePayload {
  requestId: string;
  requestType: string;
  data?: unknown;
  error?: ApiError;
}

export interface ApiError {
  message: string;
  status?: number;
  code?: string;
  details?: Record<string, unknown>;
}

export interface ApiProgressPayload {
  requestId: string;
  requestType: string;
  progress: {
    current: number;
    total: number;
    message?: string;
  };
}

// Extended WebSocket message types for API requests
export interface ApiRequestMessage {
  id: string;
  service?: string;
  event: "api_request";
  userId?: string;
  payload: ApiRequestPayload;
  timestamp: string;
}

export interface ApiResponseMessage {
  id: string;
  service: string;
  event: string; // Will be set from callbackEvent
  userId?: string;
  payload: ApiResponsePayload;
  timestamp: string;
}

export interface ApiProgressMessage {
  id: string;
  service?: string;
  event: "api_progress";
  userId?: string;
  payload: ApiProgressPayload;
  timestamp: string;
}

// Utility type for external HTTP requests
export interface ExternalHttpRequest {
  url: string;
  method: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  headers?: Record<string, string>;
  body?: unknown;
  timeout?: number;
}

export interface ExternalHttpResponse<T = unknown> {
  data: T;
  status: number;
  statusText: string;
}
