import { discogsApiService, DiscogsApiRequest, DiscogsApiResponse } from './discogsApi.service';
import { api } from '@services/api';

// WebSocket message types
export interface WebSocketMessage {
  id: string;
  type: string;
  channel: 'system' | 'user' | 'sync';
  action: string;
  userId?: string;
  data?: Record<string, unknown>;
  timestamp: string;
}

export interface WebSocketContextValue {
  connectionState: () => string;
  isConnected: () => boolean;
  isAuthenticated: () => boolean;
  sendMessage: (message: string) => void;
  onCacheInvalidation: (callback: (resourceType: string, resourceId: string) => void) => () => void;
  onSyncMessage?: (callback: (message: WebSocketMessage) => void) => () => void;
}

// Specific message data types
export interface DiscogsApiRequestData {
  requestId: string;
  url: string;
  method: string;
  headers: Record<string, string>;
}

export interface SyncProgressData {
  progress: SyncProgress;
}

export interface SyncCompleteData {
  sessionId: string;
}

export interface SyncErrorData {
  sessionId: string;
  error: string;
}

// Type guards for WebSocket message validation
export function isWebSocketMessage(obj: unknown): obj is WebSocketMessage {
  if (!obj || typeof obj !== 'object') return false;
  
  const message = obj as Record<string, unknown>;
  return (
    typeof message.id === 'string' &&
    typeof message.type === 'string' &&
    typeof message.channel === 'string' &&
    typeof message.action === 'string' &&
    typeof message.timestamp === 'string' &&
    (message.userId === undefined || typeof message.userId === 'string') &&
    (message.data === undefined || (typeof message.data === 'object' && message.data !== null))
  );
}

export function isDiscogsApiRequestData(obj: unknown): obj is DiscogsApiRequestData {
  if (!obj || typeof obj !== 'object') return false;
  
  const data = obj as Record<string, unknown>;
  return (
    typeof data.requestId === 'string' &&
    typeof data.url === 'string' &&
    typeof data.method === 'string' &&
    typeof data.headers === 'object' &&
    data.headers !== null
  );
}

export function isSyncProgressData(obj: unknown): obj is SyncProgressData {
  if (!obj || typeof obj !== 'object') return false;
  
  const data = obj as Record<string, unknown>;
  return (
    typeof data.progress === 'object' &&
    data.progress !== null
  );
}

export function isSyncCompleteData(obj: unknown): obj is SyncCompleteData {
  if (!obj || typeof obj !== 'object') return false;
  
  const data = obj as Record<string, unknown>;
  return typeof data.sessionId === 'string';
}

export function isSyncErrorData(obj: unknown): obj is SyncErrorData {
  if (!obj || typeof obj !== 'object') return false;
  
  const data = obj as Record<string, unknown>;
  return (
    typeof data.sessionId === 'string' &&
    typeof data.error === 'string'
  );
}

export interface SyncProgress {
  sessionId: string;
  status: string;
  syncType: string;
  totalRequests: number;
  completedRequests: number;
  failedRequests: number;
  percentComplete: number;
  estimatedTimeLeft?: string;
  startedAt: string;
  currentAction: string;
}

export interface SyncSession {
  sessionId: string;
  status: string;
  message: string;
}

export interface InitiateSyncRequest {
  syncType: 'collection' | 'wantlist';
  fullSync: boolean;
  pageLimit?: number;
}

export type SyncProgressCallback = (progress: SyncProgress) => void;
export type SyncCompleteCallback = (sessionId: string) => void;
export type SyncErrorCallback = (sessionId: string, error: string) => void;

export class DiscogsProxyService {
  private webSocket: WebSocketContextValue | null = null;
  private progressCallbacks: Set<SyncProgressCallback> = new Set();
  private completeCallbacks: Set<SyncCompleteCallback> = new Set();
  private errorCallbacks: Set<SyncErrorCallback> = new Set();
  private isInitialized = false;

  constructor() {
    console.log('[DiscogsProxy] Service created');
  }

  /**
   * Initialize the proxy service with WebSocket context
   */
  initialize(webSocket: WebSocketContextValue): void {
    if (this.isInitialized) {
      console.warn('[DiscogsProxy] Already initialized');
      return;
    }

    this.webSocket = webSocket;
    this.setupMessageHandlers();
    this.isInitialized = true;

    console.log('[DiscogsProxy] Service initialized with WebSocket context');
  }

  /**
   * Clean up the service
   */
  cleanup(): void {
    this.progressCallbacks.clear();
    this.completeCallbacks.clear();
    this.errorCallbacks.clear();
    this.webSocket = null;
    this.isInitialized = false;

    console.log('[DiscogsProxy] Service cleaned up');
  }

  /**
   * Check if the service is ready to handle sync operations
   */
  isReady(): boolean {
    return this.isInitialized &&
           this.webSocket !== null &&
           this.webSocket.isConnected() &&
           this.webSocket.isAuthenticated();
  }

  /**
   * Initiate a collection sync
   */
  async initiateCollectionSync(request: InitiateSyncRequest): Promise<SyncSession> {
    console.log('[DiscogsProxy] Initiating collection sync', request);

    try {
      const response = await api.post<SyncSession>('/discogs/sync-collection', request);
      console.log('[DiscogsProxy] Sync initiated successfully', response);
      return response;
    } catch (error) {
      console.error('[DiscogsProxy] Failed to initiate sync', error);
      throw error;
    }
  }

  /**
   * Get sync status
   */
  async getSyncStatus(sessionId: string): Promise<SyncProgress> {
    console.log('[DiscogsProxy] Getting sync status for session', sessionId);

    try {
      const response = await api.get<SyncProgress>(`/discogs/sync-status/${sessionId}`);
      console.log('[DiscogsProxy] Sync status retrieved', response);
      return response;
    } catch (error) {
      console.error('[DiscogsProxy] Failed to get sync status', error);
      throw error;
    }
  }

  /**
   * Cancel a sync session
   */
  async cancelSync(sessionId: string): Promise<void> {
    console.log('[DiscogsProxy] Cancelling sync session', sessionId);

    try {
      await api.post(`/discogs/sync-cancel/${sessionId}`, {});
      console.log('[DiscogsProxy] Sync cancelled successfully');
    } catch (error) {
      console.error('[DiscogsProxy] Failed to cancel sync', error);
      throw error;
    }
  }

  /**
   * Pause a sync session
   */
  async pauseSync(sessionId: string): Promise<void> {
    console.log('[DiscogsProxy] Pausing sync session', sessionId);

    try {
      await api.post(`/discogs/sync-pause/${sessionId}`, {});
      console.log('[DiscogsProxy] Sync paused successfully');
    } catch (error) {
      console.error('[DiscogsProxy] Failed to pause sync', error);
      throw error;
    }
  }

  /**
   * Resume a sync session
   */
  async resumeSync(sessionId: string): Promise<void> {
    console.log('[DiscogsProxy] Resuming sync session', sessionId);

    try {
      await api.post(`/discogs/sync-resume/${sessionId}`, {});
      console.log('[DiscogsProxy] Sync resumed successfully');
    } catch (error) {
      console.error('[DiscogsProxy] Failed to resume sync', error);
      throw error;
    }
  }

  /**
   * Update user's Discogs token
   */
  async updateDiscogsToken(token: string): Promise<{ message: string; tokenValid: boolean }> {
    console.log('[DiscogsProxy] Updating Discogs token');

    try {
      const response = await api.put<{ message: string; tokenValid: boolean }>('/discogs/token', {
        discogsToken: token,
      });
      console.log('[DiscogsProxy] Token updated successfully');
      return response;
    } catch (error) {
      console.error('[DiscogsProxy] Failed to update token', error);
      throw error;
    }
  }

  /**
   * Validate user's Discogs token
   */
  async validateDiscogsToken(): Promise<{ hasToken: boolean; tokenValid: boolean }> {
    console.log('[DiscogsProxy] Validating Discogs token');

    try {
      const response = await api.get<{ hasToken: boolean; tokenValid: boolean }>('/discogs/token/validate');
      console.log('[DiscogsProxy] Token validation result', response);
      return response;
    } catch (error) {
      console.error('[DiscogsProxy] Failed to validate token', error);
      throw error;
    }
  }

  /**
   * Get current rate limit information
   */
  async getRateLimit(): Promise<{
    remaining: number;
    limit: number;
    windowReset: string;
    recommendedDelay: string;
  }> {
    console.log('[DiscogsProxy] Getting rate limit info');

    try {
      const response = await api.get<{
        remaining: number;
        limit: number;
        windowReset: string;
        recommendedDelay: string;
      }>('/discogs/rate-limit');
      console.log('[DiscogsProxy] Rate limit info retrieved', response);
      return response;
    } catch (error) {
      console.error('[DiscogsProxy] Failed to get rate limit info', error);
      throw error;
    }
  }

  /**
   * Register callback for sync progress updates
   */
  onSyncProgress(callback: SyncProgressCallback): () => void {
    this.progressCallbacks.add(callback);
    return () => this.progressCallbacks.delete(callback);
  }

  /**
   * Register callback for sync completion
   */
  onSyncComplete(callback: SyncCompleteCallback): () => void {
    this.completeCallbacks.add(callback);
    return () => this.completeCallbacks.delete(callback);
  }

  /**
   * Register callback for sync errors
   */
  onSyncError(callback: SyncErrorCallback): () => void {
    this.errorCallbacks.add(callback);
    return () => this.errorCallbacks.delete(callback);
  }

  /**
   * Set up WebSocket message handlers for sync operations
   */
  private setupMessageHandlers(): void {
    if (!this.webSocket) {
      console.error('[DiscogsProxy] Cannot setup handlers - no WebSocket context');
      return;
    }

    // Check if the WebSocket context supports sync message handling
    if (!this.webSocket.onSyncMessage) {
      console.warn('[DiscogsProxy] WebSocket context does not support sync message handling');
      return;
    }

    console.log('[DiscogsProxy] Setting up WebSocket message handlers');

    // Register for sync-related messages
    this.webSocket.onSyncMessage((message: WebSocketMessage) => {
      this.handleIncomingMessage(message);
    });
  }

  /**
   * Handle incoming WebSocket messages
   */
  private handleIncomingMessage(message: WebSocketMessage): void {
    console.log('[DiscogsProxy] Received WebSocket message', message.type, message.action);

    try {
      switch (message.type) {
        case 'discogs_api_request':
          this.handleApiRequest(message);
          break;

        case 'sync_progress':
          this.handleSyncProgress(message);
          break;

        case 'sync_complete':
          this.handleSyncComplete(message);
          break;

        case 'sync_error':
          this.handleSyncError(message);
          break;

        default:
          console.log('[DiscogsProxy] Ignoring unknown message type:', message.type);
          break;
      }
    } catch (error) {
      console.error('[DiscogsProxy] Error handling message:', error, message);
    }
  }

  /**
   * Handle incoming API request from server
   */
  private async handleApiRequest(message: WebSocketMessage): Promise<void> {
    console.log('[DiscogsProxy] Received API request from server', message);

    if (!message.data) {
      console.error('[DiscogsProxy] API request message missing data');
      return;
    }

    if (!isDiscogsApiRequestData(message.data)) {
      console.error('[DiscogsProxy] Invalid API request data format', message.data);
      return;
    }

    const { requestId, url, method, headers } = message.data;
    const request: DiscogsApiRequest = { requestId, url, method, headers };

    // Validate the request
    const validationError = discogsApiService.validateRequest(request);
    if (validationError) {
      console.error('[DiscogsProxy] Request validation failed', validationError);
      this.sendApiResponse(discogsApiService.createErrorResponse(requestId, validationError));
      return;
    }

    try {
      // Make the actual API call
      const response = await discogsApiService.makeRequest(request);

      // Send response back to server
      this.sendApiResponse(response);
    } catch (error) {
      console.error('[DiscogsProxy] Failed to make API request', error);

      const errorResponse = discogsApiService.createErrorResponse(
        requestId,
        error instanceof Error ? error.message : 'Unknown error'
      );

      this.sendApiResponse(errorResponse);
    }
  }

  /**
   * Send API response back to server via WebSocket
   */
  private sendApiResponse(response: DiscogsApiResponse): void {
    if (!this.webSocket) {
      console.error('[DiscogsProxy] Cannot send response - no WebSocket context');
      return;
    }

    const message = {
      id: crypto.randomUUID(),
      type: 'discogs_api_response',
      channel: 'sync',
      action: 'request_complete',
      data: response,
      timestamp: new Date().toISOString(),
    };

    console.log('[DiscogsProxy] Sending API response to server', {
      requestId: response.requestId,
      status: response.status
    });

    this.webSocket.sendMessage(JSON.stringify(message));
  }

  /**
   * Handle sync progress updates from server
   */
  private handleSyncProgress(message: WebSocketMessage): void {
    console.log('[DiscogsProxy] Received sync progress update', message);

    if (!message.data) {
      console.warn('[DiscogsProxy] Sync progress message missing data');
      return;
    }

    if (!isSyncProgressData(message.data)) {
      console.warn('[DiscogsProxy] Invalid sync progress data format', message.data);
      return;
    }

    const { progress } = message.data;

    // Validate that progress has required SyncProgress properties
    if (!this.isValidSyncProgress(progress)) {
      console.warn('[DiscogsProxy] Progress data does not match SyncProgress interface', progress);
      return;
    }

    // Notify all progress callbacks
    this.progressCallbacks.forEach(callback => {
      try {
        callback(progress);
      } catch (error) {
        console.error('[DiscogsProxy] Error in progress callback', error);
      }
    });
  }

  /**
   * Type guard to validate SyncProgress object
   */
  private isValidSyncProgress(obj: unknown): obj is SyncProgress {
    if (!obj || typeof obj !== 'object') return false;

    const progress = obj as Record<string, unknown>;
    return (
      typeof progress.sessionId === 'string' &&
      typeof progress.status === 'string' &&
      typeof progress.syncType === 'string' &&
      typeof progress.totalRequests === 'number' &&
      typeof progress.completedRequests === 'number' &&
      typeof progress.failedRequests === 'number' &&
      typeof progress.percentComplete === 'number' &&
      typeof progress.startedAt === 'string' &&
      typeof progress.currentAction === 'string' &&
      (progress.estimatedTimeLeft === undefined || typeof progress.estimatedTimeLeft === 'string')
    );
  }

  /**
   * Handle sync completion from server
   */
  private handleSyncComplete(message: WebSocketMessage): void {
    console.log('[DiscogsProxy] Received sync complete notification', message);

    if (!message.data) {
      console.warn('[DiscogsProxy] Sync complete message missing data');
      return;
    }

    if (!isSyncCompleteData(message.data)) {
      console.warn('[DiscogsProxy] Invalid sync complete data format', message.data);
      return;
    }

    const { sessionId } = message.data;

    // Notify all completion callbacks
    this.completeCallbacks.forEach(callback => {
      try {
        callback(sessionId);
      } catch (error) {
        console.error('[DiscogsProxy] Error in complete callback', error);
      }
    });
  }

  /**
   * Handle sync errors from server
   */
  private handleSyncError(message: WebSocketMessage): void {
    console.log('[DiscogsProxy] Received sync error notification', message);

    if (!message.data) {
      console.warn('[DiscogsProxy] Sync error message missing data');
      return;
    }

    if (!isSyncErrorData(message.data)) {
      console.warn('[DiscogsProxy] Invalid sync error data format', message.data);
      return;
    }

    const { sessionId, error } = message.data;

    // Notify all error callbacks
    this.errorCallbacks.forEach(callback => {
      try {
        callback(sessionId, error);
      } catch (callbackError) {
        console.error('[DiscogsProxy] Error in error callback', callbackError);
      }
    });
  }
}

// Export a singleton instance
export const discogsProxyService = new DiscogsProxyService();