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

export const MessageType = {
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
  DISCOGS_API_REQUEST: "discogs_api_request",
  DISCOGS_API_RESPONSE: "discogs_api_response",
  SYNC_PROGRESS: "sync_progress",
  SYNC_COMPLETE: "sync_complete",
  SYNC_ERROR: "sync_error",
} as const;

export type ChannelType = "system" | "user" | "sync";

export interface WebSocketMessage {
  id: string;
  type: string;
  channel: ChannelType;
  action: string;
  userId?: string;
  data?: Record<string, unknown>;
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
  onSyncMessage: (
    callback: (message: WebSocketMessage) => void,
  ) => () => void;
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

  // Sync message callbacks
  const [syncMessageCallbacks, setSyncMessageCallbacks] =
    createSignal<Array<(message: WebSocketMessage) => void>>([]);

  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const log = (..._args: unknown[]) => {
    // Debug logging disabled for production
    // if (debug) {
    //   console.log(`[WebSocket] ${_args[0]}`, ..._args.slice(1));
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
      type: MessageType.AUTH_RESPONSE,
      channel: "system",
      action: "authenticate",
      data: { token },
      timestamp: new Date().toISOString(),
    };

    const ws = wsInstance();
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify(authResponse));
      log("Auth response sent");
    }
  };

  const handleMessage = (event: MessageEvent) => {
    try {
      const message: WebSocketMessage = JSON.parse(event.data);
      log("Received message:", message);
      setLastMessage(event.data);

      switch (message.type) {
        case MessageType.AUTH_REQUEST:
          handleAuthRequest();
          break;

        case MessageType.AUTH_SUCCESS:
          log("Authentication successful");
          setWsAuthenticated(true);
          setLastError(null);
          break;

        case MessageType.AUTH_FAILURE:
          log("Authentication failed:", message.data?.reason);
          setWsAuthenticated(false);
          setLastError(
            typeof message.data?.reason === "string"
              ? message.data.reason
              : "Authentication failed",
          );
          break;

        case MessageType.DISCOGS_API_REQUEST:
        case MessageType.SYNC_PROGRESS:
        case MessageType.SYNC_COMPLETE:
        case MessageType.SYNC_ERROR:
          // Handle sync-related messages
          log("Sync message received:", message.type, message);
          syncMessageCallbacks().forEach((callback) => {
            try {
              callback(message);
            } catch (error) {
              log("Sync message callback error:", error);
            }
          });
          break;

        default:
          // Handle cache invalidation and other message types
          if (message.action === "invalidateCache" && message.data) {
            const resourceType = message.data.resourceType as string;
            const resourceId = message.data.resourceId as string;

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

  const onSyncMessage = (
    callback: (message: WebSocketMessage) => void,
  ) => {
    setSyncMessageCallbacks((prev) => [...prev, callback]);

    // Return cleanup function
    return () => {
      setSyncMessageCallbacks((prev) =>
        prev.filter((cb) => cb !== callback),
      );
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
    onSyncMessage,
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
