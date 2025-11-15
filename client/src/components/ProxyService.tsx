import { Events, useWebSocket, type WebSocketMessage } from "@context/WebSocketContext";
import axios, { type AxiosError, type AxiosRequestConfig, type AxiosResponse } from "axios";
import { createEffect } from "solid-js";
import type {
  ApiError,
  ApiRequestPayload,
  ApiResponsePayload,
  ExternalHttpRequest,
  ExternalHttpResponse,
} from "src/types/ApiProxy";

// Error classes for external API handling
class ExternalApiError extends Error {
  constructor(
    message: string,
    public status?: number,
    public code?: string,
    public details?: Record<string, unknown>,
  ) {
    super(message);
    this.name = "ExternalApiError";
  }
}

class ExternalNetworkError extends Error {
  constructor(
    message: string,
    public originalError?: Error,
  ) {
    super(message);
    this.name = "ExternalNetworkError";
  }
}

// Handle external API errors consistently
const handleExternalApiError = (error: AxiosError): ExternalApiError | ExternalNetworkError => {
  if (error.response) {
    const response = error.response as AxiosResponse<{ message?: string }>;
    const errorMessage = response.data?.message || error.message || "External API error";

    return new ExternalApiError(errorMessage, error.response.status, error.code, {
      url: error.config?.url,
      method: error.config?.method,
      responseData: response.data,
    });
  } else if (error.request) {
    return new ExternalNetworkError("Network error: No response from external API", error);
  } else {
    return new ExternalNetworkError(error.message || "An unexpected error occurred", error);
  }
};

// Core external API request function
const makeExternalRequest = async <T = unknown>(
  request: ExternalHttpRequest,
): Promise<ExternalHttpResponse<T>> => {
  try {
    const config: AxiosRequestConfig = {
      method: request.method,
      url: request.url,
      headers: request.headers,
      data: request.body,
      timeout: request.timeout || 30000,
      validateStatus: () => true,
    };

    const response = await axios.request(config);

    return {
      data: response.data,
      status: response.status,
      statusText: response.statusText,
    };
  } catch (error) {
    throw handleExternalApiError(error as AxiosError);
  }
};

// Convert external API errors to standardized format
const formatError = (error: unknown): ApiError => {
  if (error instanceof ExternalApiError) {
    return {
      message: error.message,
      status: error.status,
      code: error.code,
      details: error.details,
    };
  } else if (error instanceof ExternalNetworkError) {
    return {
      message: error.message,
      code: "NETWORK_ERROR",
      details: {
        originalError: error.originalError?.message,
      },
    };
  } else if (error instanceof Error) {
    return {
      message: error.message,
      code: "UNKNOWN_ERROR",
      details: {
        name: error.name,
      },
    };
  } else {
    return {
      message: "An unknown error occurred",
      code: "UNKNOWN_ERROR",
      details: {
        error: String(error),
      },
    };
  }
};

// Validate external HTTP request
const validateRequest = (request: ExternalHttpRequest): void => {
  if (!request.url) {
    throw new Error("URL is required for external API request");
  }

  if (!request.method) {
    throw new Error("HTTP method is required for external API request");
  }

  const validMethods = ["GET", "POST", "PUT", "PATCH", "DELETE"];
  if (!validMethods.includes(request.method)) {
    throw new Error(`Invalid HTTP method: ${request.method}`);
  }

  try {
    new URL(request.url);
  } catch {
    throw new Error(`Invalid URL format: ${request.url}`);
  }
};

// Validate WebSocket API request payload
const validateApiRequestPayload = (payload: unknown): payload is ApiRequestPayload => {
  if (!payload || typeof payload !== "object") {
    throw new Error("API request payload is required and must be an object");
  }

  const payloadObj = payload as Record<string, unknown>;
  const requiredFields = [
    "requestId",
    "requestType",
    "url",
    "method",
    "headers",
    "callbackService",
    "callbackEvent",
  ];

  for (const field of requiredFields) {
    if (!payloadObj[field]) {
      throw new Error(`Missing required field in API request: ${field}`);
    }
  }

  validateRequest({
    url: payloadObj.url as string,
    method: payloadObj.method as "GET" | "POST" | "PUT" | "PATCH" | "DELETE",
    headers: payloadObj.headers as Record<string, string>,
    body: payloadObj.body,
  });

  return true;
};

// Create API response payload
const createResponsePayload = (
  requestId: string,
  requestType: string,
  data?: unknown,
  error?: ApiError,
): ApiResponsePayload => {
  return {
    requestId,
    requestType,
    data,
    error,
  };
};

/**
 * ProxyService - Provider-style component that handles all API proxy logic
 *
 * This component:
 * - Listens to WebSocket messages reactively via lastMessage signal
 * - Handles API requests when event === "api_request"
 * - Makes external HTTP requests using built-in logic
 * - Sends responses back via WebSocket
 * - Maintains no UI (renders null)
 */
export function ProxyService() {
  const { lastMessage, sendMessage } = useWebSocket();

  // React to incoming WebSocket messages
  createEffect(() => {
    const messageData = lastMessage();
    if (!messageData) return;

    try {
      const message: WebSocketMessage = JSON.parse(messageData);

      // Only handle API request events
      if (message.event === Events.API_REQUEST) {
        handleApiRequest(message);
      }
    } catch {
      // Silently ignore malformed messages
    }
  });

  /**
   * Handle incoming API request from server
   */
  const handleApiRequest = async (message: WebSocketMessage) => {
    if (!message.payload) {
      return;
    }

    try {
      // Validate the API request payload
      validateApiRequestPayload(message.payload);
      const payload = message.payload as unknown as ApiRequestPayload;

      // Extract HTTP request configuration
      const httpRequest: ExternalHttpRequest = {
        url: payload.url,
        method: payload.method,
        headers: payload.headers,
        body: payload.body,
        timeout: 30000, // 30 second timeout for external APIs
      };

      // Make the external HTTP request
      const response = await makeExternalRequest(httpRequest);

      // Create success response payload
      const responsePayload = createResponsePayload(payload.requestId, payload.requestType, {
        status: response.status,
        statusText: response.statusText,
        data: response.data,
      });

      // Send response back to server
      const responseMessage: WebSocketMessage = {
        id: crypto.randomUUID(),
        service: payload.callbackService,
        event: payload.callbackEvent,
        payload: responsePayload as unknown as Record<string, unknown>,
        timestamp: new Date().toISOString(),
      };

      sendMessage(responseMessage);
    } catch (error) {
      // We need payload for error handling, so we'll try to extract it if validation failed
      let errorPayload: ApiResponsePayload;
      let callbackService = "orchestration";
      let callbackEvent = "api_response";

      try {
        // Try to get the payload even if validation failed
        const payload = message.payload as unknown as Partial<ApiRequestPayload>;
        errorPayload = createResponsePayload(
          payload.requestId || "unknown",
          payload.requestType || "unknown",
          undefined,
          formatError(error),
        );
        callbackService = payload.callbackService || "orchestration";
        callbackEvent = payload.callbackEvent || "api_response";
      } catch {
        // If we can't parse payload at all, create a generic error
        errorPayload = createResponsePayload("unknown", "unknown", undefined, formatError(error));
      }

      // Send error response back to server
      const errorMessage: WebSocketMessage = {
        id: crypto.randomUUID(),
        service: callbackService,
        event: callbackEvent,
        payload: errorPayload as unknown as Record<string, unknown>,
        timestamp: new Date().toISOString(),
      };

      sendMessage(errorMessage);
    }
  };

  // This component renders nothing - it's pure logic
  return null;
}
