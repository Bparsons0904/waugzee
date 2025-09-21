import { WebSocketMessage } from "@context/WebSocketContext";

export interface WebSocketContextValue {
  connectionState: () => string;
  isConnected: () => boolean;
  isAuthenticated: () => boolean;
  sendMessage: (message: string) => void;
  onCacheInvalidation: (
    callback: (resourceType: string, resourceId: string) => void,
  ) => () => void;
  onSyncMessage?: (callback: (message: WebSocketMessage) => void) => () => void;
}

export interface ProxyRequest {
  requestId: string;
  url: string;
  method: string;
  headers: Record<string, string>;
  body?: unknown;
}

export interface ProxyResponse {
  requestId: string;
  status: number;
  headers: Record<string, string>;
  body: unknown;
  error?: string;
}

export class DiscogsProxyService {
  private webSocket: WebSocketContextValue | null = null;
  private isInitialized = false;

  constructor() {
    console.log("[DiscogsProxy] Service created");
  }

  initialize(webSocket: WebSocketContextValue): void {
    if (this.isInitialized) {
      console.warn("[DiscogsProxy] Already initialized");
      return;
    }

    this.webSocket = webSocket;
    this.setupMessageHandlers();
    this.isInitialized = true;

    console.log("[DiscogsProxy] Service initialized");
  }

  cleanup(): void {
    this.webSocket = null;
    this.isInitialized = false;
    console.log("[DiscogsProxy] Service cleaned up");
  }

  isReady(): boolean {
    return (
      this.isInitialized &&
      this.webSocket !== null &&
      this.webSocket.isConnected() &&
      this.webSocket.isAuthenticated()
    );
  }

  private setupMessageHandlers(): void {
    if (!this.webSocket?.onSyncMessage) {
      console.warn("[DiscogsProxy] WebSocket context does not support sync messages");
      return;
    }

    this.webSocket.onSyncMessage((message: WebSocketMessage) => {
      if (message.type === "discogs_api_request") {
        this.handleApiRequest(message);
      }
    });
  }

  private async handleApiRequest(message: WebSocketMessage): Promise<void> {
    console.log("[DiscogsProxy] Received API request", message.id);

    if (!message.data || !this.isValidProxyRequest(message.data)) {
      console.error("[DiscogsProxy] Invalid request data");
      return;
    }

    const requestData = message.data as unknown as ProxyRequest;
    const { requestId, url, method, headers } = requestData;

    try {
      const response = await this.makeHttpRequest(requestId, url, method, headers);
      this.sendResponse(response);
    } catch (error) {
      console.error("[DiscogsProxy] Request failed", error);
      this.sendResponse({
        requestId,
        status: 0,
        headers: {},
        body: null,
        error: error instanceof Error ? error.message : "Unknown error",
      });
    }
  }

  private isValidProxyRequest(data: unknown): boolean {
    if (!data || typeof data !== "object") return false;
    const req = data as Record<string, unknown>;
    return (
      typeof req.requestId === "string" &&
      typeof req.url === "string" &&
      typeof req.method === "string" &&
      typeof req.headers === "object" &&
      req.headers !== null
    );
  }

  private async makeHttpRequest(
    requestId: string,
    url: string,
    method: string,
    headers: Record<string, string>
  ): Promise<ProxyResponse> {
    console.log(`[DiscogsProxy] Making ${method} request to ${url}`);

    const response = await fetch(url, {
      method,
      headers,
    });

    // Extract response headers
    const responseHeaders: Record<string, string> = {};
    response.headers.forEach((value, key) => {
      responseHeaders[key.toLowerCase()] = value;
    });

    // Get response body
    let body: unknown;
    const contentType = response.headers.get('content-type');

    if (contentType && contentType.includes('application/json')) {
      body = await response.json();
    } else {
      body = await response.text();
    }

    const result: ProxyResponse = {
      requestId,
      status: response.status,
      headers: responseHeaders,
      body,
    };

    if (!response.ok) {
      result.error = `HTTP ${response.status}: ${response.statusText}`;
      console.error(`[DiscogsProxy] Request failed:`, result.error);
    } else {
      console.log(`[DiscogsProxy] Request completed successfully`, {
        requestId,
        status: response.status,
      });
    }

    return result;
  }

  private sendResponse(response: ProxyResponse): void {
    if (!this.webSocket) {
      console.error("[DiscogsProxy] Cannot send response - no WebSocket");
      return;
    }

    const message = {
      id: crypto.randomUUID(),
      type: "discogs_api_response",
      channel: "sync",
      action: "request_complete",
      data: response,
      timestamp: new Date().toISOString(),
    };

    console.log("[DiscogsProxy] Sending response", response.requestId);
    this.webSocket.sendMessage(JSON.stringify(message));
  }
}

export const discogsProxyService = new DiscogsProxyService();