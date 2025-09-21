import {
  createContext,
  useContext,
  createSignal,
  createEffect,
  onCleanup,
  JSX,
} from "solid-js";
import {
  createReconnectingWS,
  createWSState,
} from "@solid-primitives/websocket";
import { env } from "@services/env.service";
import { useAuth } from "./AuthContext";
import { externalApiService } from "@services/externalApi.service";
import {
  DiscogsApiRequestPayload,
  DiscogsApiResponsePayload,
  ApiRequestMessage,
  ApiResponseMessage,
} from "src/types/DiscogsApiProxy";

// Event constants matching server-side implementation
export const Events = {
  PING: "ping",
  PONG: "pong",
  MESSAGE: "message",
  BROADCAST: "broadcast",
  ERROR: "error",
  USER_JOIN: "user_join",
  USER_LEAVE: "user_leave",
  AUTH_REQUEST: "auth_request",
  AUTH_RESPONSE: "auth_response",
  AUTH_SUCCESS: "auth_success",
  AUTH_FAILURE: "auth_failure",
  INVALIDATE_CACHE: "invalidateCache",
  API_REQUEST: "api_request",
  API_RESPONSE: "api_response",
  API_PROGRESS: "api_progress",
  API_COMPLETE: "api_complete",
  API_ERROR: "api_error",
} as const;

// Service types matching server-side implementation
export type ServiceType = "system" | "user" | "api";

export interface WebSocketMessage {
  id: string;
  service?: ServiceType | string;
  event: string;
  userId?: string;
  payload?: Record<string, unknown>;
  timestamp: string;
}

export enum ConnectionState {
  Connecting = "connecting",
  Connected = "connected",
  Authenticating = "authenticating",
  Authenticated = "authenticated",
  Disconnecting = "disconnecting",
  Disconnected = "disconnected",
  Failed = "failed",
}

interface WebSocketContextValue {
  connectionState: () => ConnectionState;
  isConnected: () => boolean;
  isAuthenticated: () => boolean;
  lastError: () => string | null;
  lastMessage: () => string;
  sendMessage: (message: string) => void;
  reconnect: () => void;
  onCacheInvalidation: (
    callback: (resourceType: string, resourceId: string) => void,
  ) => () => void;
  onApiMessage: (callback: (message: WebSocketMessage) => void) => () => void;
}

const WebSocketContext = createContext<WebSocketContextValue>(
  {} as WebSocketContextValue,
);

interface WebSocketProviderProps {
  children: JSX.Element;
  debug?: boolean;
}

export function WebSocketProvider(props: WebSocketProviderProps) {
  const { isAuthenticated, authToken } = useAuth();

  const [lastError, setLastError] = createSignal<string | null>(null);
  const [lastMessage, setLastMessage] = createSignal<string>("");
  const [wsInstance, setWsInstance] = createSignal<ReturnType<
    typeof createReconnectingWS
  > | null>(null);
  const [wsAuthenticated, setWsAuthenticated] = createSignal<boolean>(false);

  // Cache invalidation callbacks
  const [cacheInvalidationCallbacks, setCacheInvalidationCallbacks] =
    createSignal<Array<(resourceType: string, resourceId: string) => void>>([]);

  // API message callbacks
  const [apiMessageCallbacks, setApiMessageCallbacks] = createSignal<
    Array<(message: WebSocketMessage) => void>
  >([]);

  const log = (..._args: unknown[]) => {
    // Debug logging disabled for production
    // if (debug) {
    console.log(`[WebSocket] ${_args[0]}`, ..._args.slice(1));
    // }
  };

  const getWebSocketUrl = () => {
    if (!isAuthenticated() || !authToken()) {
      return null;
    }
    return env.wsUrl;
  };

  const handleAuthRequest = () => {
    log("Handling auth request");
    const token = authToken();

    if (!token) {
      log("No auth token available");
      setLastError("No authentication token available");
      return;
    }

    const authResponse: WebSocketMessage = {
      id: crypto.randomUUID(),
      service: "system",
      event: Events.AUTH_RESPONSE,
      payload: { token },
      timestamp: new Date().toISOString(),
    };

    const ws = wsInstance();
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify(authResponse));
      log("Auth response sent");
    }
  };

  const handleApiRequest = async (message: ApiRequestMessage) => {
    const requestId = message.payload?.requestId || "unknown";
    log("Handling API request:", requestId, message.payload?.url);

    // Extract payload with fallbacks
    const payload = message.payload;
    if (!payload) {
      log("Invalid API request: missing payload");
      return;
    }

    const {
      url,
      method,
      headers,
      callbackService,
      callbackEvent,
      requestType,
      body
    } = payload;

    // Helper function to send response
    const sendResponse = (responsePayload: DiscogsApiResponsePayload) => {
      const responseMessage: ApiResponseMessage = {
        id: crypto.randomUUID(),
        service: callbackService,
        event: callbackEvent,
        payload: responsePayload,
        timestamp: new Date().toISOString(),
      };

      const ws = wsInstance();
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify(responseMessage));
        return true;
      } else {
        log("Cannot send response: WebSocket not open");
        return false;
      }
    };

    try {
      // Validate the API request payload
      externalApiService.validateApiRequestPayload(payload);

      // Make the external HTTP request using our service
      const response = await externalApiService.makeRequest({
        url,
        method,
        headers,
        body,
        timeout: 30000, // 30 second timeout for external APIs
      });

      // Create success response
      const responsePayload: DiscogsApiResponsePayload = {
        requestId,
        requestType,
        data: response.data,
      };

      if (sendResponse(responsePayload)) {
        log("API response sent:", requestId, response.status);
      }
    } catch (error) {
      log("API request failed:", requestId, error);

      // Create error response
      const errorPayload: DiscogsApiResponsePayload = {
        requestId,
        requestType,
        error: externalApiService.formatError(error),
      };

      if (sendResponse(errorPayload)) {
        log("API error response sent:", requestId);
      }
    }
  };

  const handleMessage = (event: MessageEvent) => {
    try {
      const message: WebSocketMessage = JSON.parse(event.data);
      log("Received message:", message);
      setLastMessage(event.data);

      // Validate basic message structure
      if (!message.event) {
        log("Invalid message: missing event field");
        return;
      }

      switch (message.event) {
        case Events.AUTH_REQUEST:
          handleAuthRequest();
          break;

        case Events.AUTH_SUCCESS:
          log("Authentication successful");
          setWsAuthenticated(true);
          setLastError(null);
          break;

        case Events.AUTH_FAILURE:
          log("Authentication failed:", message.payload?.reason);
          setWsAuthenticated(false);
          setLastError(
            typeof message.payload?.reason === "string"
              ? message.payload.reason
              : "Authentication failed",
          );
          break;

        case Events.API_REQUEST:
          // Handle API request by making external HTTP call
          log("API request received:", message);
          try {
            handleApiRequest(message as unknown as ApiRequestMessage);
          } catch (error) {
            log("Error handling API request:", error);
          }
          break;

        case Events.API_RESPONSE:
        case Events.API_PROGRESS:
        case Events.API_COMPLETE:
        case Events.API_ERROR:
          // Handle API-related messages
          log("API message received:", message.event, message);
          apiMessageCallbacks().forEach((callback) => {
            try {
              callback(message);
            } catch (error) {
              log("API message callback error:", error);
            }
          });
          break;

        default:
          // Handle cache invalidation and other message types
          if (message.payload?.action === "invalidateCache" && message.payload) {
            const resourceType = message.payload.resourceType as string;
            const resourceId = message.payload.resourceId as string;

            if (resourceType && resourceId) {
              log("Cache invalidation received:", resourceType, resourceId);
              // Notify all cache invalidation callbacks
              cacheInvalidationCallbacks().forEach((callback) => {
                try {
                  callback(resourceType, resourceId);
                } catch (error) {
                  log("Cache invalidation callback error:", error);
                }
              });
            }
          }
          break;
      }
    } catch (error) {
      log("Failed to parse message:", error);
    }
  };

  createEffect(() => {
    const url = getWebSocketUrl();

    if (!url) {
      log("No URL available, clearing WebSocket");
      setWsInstance(null);
      setWsAuthenticated(false);
      setLastError("Authentication required");
      return;
    }

    log("Creating WebSocket");
    setWsAuthenticated(false);

    try {
      const ws = createReconnectingWS(url);

      // Set up event listeners
      ws.addEventListener("open", () => {
        log("WebSocket connected, waiting for auth request");
        setLastError(null);
      });

      ws.addEventListener("message", handleMessage);

      ws.addEventListener("error", (event) => {
        log("WebSocket error:", event);
        setLastError("Connection error occurred");
      });

      ws.addEventListener("close", (event) => {
        log("WebSocket closed:", event.code, event.reason);
        setWsAuthenticated(false);
        if (event.code !== 1000) {
          // Not normal closure
          setLastError(
            `Connection closed unexpectedly: ${event.reason || "Unknown reason"}`,
          );
        }
      });

      setWsInstance(ws);
      setLastError(null);
    } catch (error) {
      log("Failed to create WebSocket:", error);
      setLastError(
        error instanceof Error ? error.message : "Failed to create connection",
      );
      setWsInstance(null);
    }
  });

  const wsState = () => {
    const ws = wsInstance();
    return ws ? createWSState(ws)() : WebSocket.CLOSED;
  };

  const connectionState = (): ConnectionState => {
    if (!wsInstance()) {
      return ConnectionState.Disconnected;
    }

    const rawState = wsState();

    switch (rawState) {
      case WebSocket.CONNECTING:
        return ConnectionState.Connecting;
      case WebSocket.OPEN:
        return wsAuthenticated()
          ? ConnectionState.Authenticated
          : ConnectionState.Authenticating;
      case WebSocket.CLOSING:
        return ConnectionState.Disconnecting;
      case WebSocket.CLOSED:
        return ConnectionState.Disconnected;
      default:
        return ConnectionState.Failed;
    }
  };

  const isConnected = () => {
    const state = connectionState();
    return (
      state === ConnectionState.Connected ||
      state === ConnectionState.Authenticating ||
      state === ConnectionState.Authenticated
    );
  };

  const isWebSocketAuthenticated = () => wsAuthenticated();

  const sendMessage = (message: string) => {
    if (!message) {
      log("Cannot send message: Message is empty");
      setLastError("Cannot send message: Message is empty");
      return;
    }

    const ws = wsInstance();

    if (!ws) {
      log("Cannot send message: No WebSocket instance");
      setLastError("Cannot send message: not connected");
      return;
    }

    if (!wsAuthenticated()) {
      log("Cannot send message: WebSocket not authenticated");
      setLastError("Cannot send message: not authenticated");
      return;
    }

    if (wsState() !== WebSocket.OPEN) {
      log("Cannot send message: WebSocket not open");
      setLastError("Cannot send message: connection not ready");
      return;
    }

    try {
      ws.send(message);
      log("Message sent:", message);
    } catch (error) {
      log("Failed to send message:", error);
      setLastError(
        error instanceof Error ? error.message : "Failed to send message",
      );
    }
  };

  const reconnect = () => {
    const ws = wsInstance();
    if (ws && "reconnect" in ws && typeof ws.reconnect === "function") {
      log("Manually triggering reconnection");
      setWsAuthenticated(false);
      ws.reconnect();
    } else {
      log("No reconnect method available, recreating connection");
      setWsInstance(null);
      setWsAuthenticated(false);
    }
  };

  onCleanup(() => {
    log("Cleaning up WebSocket connection");
    const ws = wsInstance();
    if (ws) {
      ws.close(1000, "Component cleanup");
    }
  });

  createEffect(() => {
    const handleBeforeUnload = () => {
      log("Page unloading, closing WebSocket");
      const ws = wsInstance();
      if (ws) {
        ws.close(1000, "Page unload");
      }
    };

    window.addEventListener("beforeunload", handleBeforeUnload);

    onCleanup(() => {
      window.removeEventListener("beforeunload", handleBeforeUnload);
    });
  });

  const onCacheInvalidation = (
    callback: (resourceType: string, resourceId: string) => void,
  ) => {
    setCacheInvalidationCallbacks((prev) => [...prev, callback]);

    // Return cleanup function
    return () => {
      setCacheInvalidationCallbacks((prev) =>
        prev.filter((cb) => cb !== callback),
      );
    };
  };

  const onApiMessage = (callback: (message: WebSocketMessage) => void) => {
    setApiMessageCallbacks((prev) => [...prev, callback]);

    // Return cleanup function
    return () => {
      setApiMessageCallbacks((prev) => prev.filter((cb) => cb !== callback));
    };
  };

  const contextValue: WebSocketContextValue = {
    connectionState,
    isConnected,
    isAuthenticated: isWebSocketAuthenticated,
    lastError,
    lastMessage,
    sendMessage,
    reconnect,
    onCacheInvalidation,
    onApiMessage,
  };

  return (
    <WebSocketContext.Provider value={contextValue}>
      {props.children}
    </WebSocketContext.Provider>
  );
}

export function useWebSocket() {
  const context = useContext(WebSocketContext);
  if (!context) {
    throw new Error("useWebSocket must be used within WebSocketProvider");
  }
  return context;
}
