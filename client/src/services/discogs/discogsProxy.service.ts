import { discogsApiService, DiscogsApiRequest, DiscogsApiResponse } from './discogsApi.service';
import { apiService } from '@services/api/api.service';

export interface WebSocketContextValue {
  connectionState: () => string;
  isConnected: () => boolean;
  isAuthenticated: () => boolean;
  sendMessage: (message: string) => void;
  onCacheInvalidation: (callback: (resourceType: string, resourceId: string) => void) => () => void;
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
      const response = await apiService.post<SyncSession>('/discogs/sync-collection', request);
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
      const response = await apiService.get<SyncProgress>(`/discogs/sync-status/${sessionId}`);
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
      await apiService.post(`/discogs/sync-cancel/${sessionId}`, {});
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
      await apiService.post(`/discogs/sync-pause/${sessionId}`, {});
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
      await apiService.post(`/discogs/sync-resume/${sessionId}`, {});
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
      const response = await apiService.put<{ message: string; tokenValid: boolean }>('/discogs/token', {
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
      const response = await apiService.get<{ hasToken: boolean; tokenValid: boolean }>('/discogs/token/validate');
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
      const response = await apiService.get<{
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

    console.log('[DiscogsProxy] Setting up WebSocket message handlers');

    // Listen for incoming messages and parse them manually
    // Note: This assumes the WebSocket context provides a way to listen to raw messages
    // We'll need to enhance the WebSocketContext to support custom message handlers
  }

  /**
   * Handle incoming API request from server
   */
  private async handleApiRequest(message: { data: { requestId?: string; url?: string; method?: string; headers?: Record<string, string> } }): Promise<void> {
    console.log('[DiscogsProxy] Received API request from server', message);

    const { requestId, url, method, headers } = message.data;

    if (!requestId || !url || !method || !headers) {
      console.error('[DiscogsProxy] Invalid API request format', message);
      return;
    }

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
  private handleSyncProgress(message: { data?: { progress?: unknown } }): void {
    console.log('[DiscogsProxy] Received sync progress update', message);

    const progress = message.data?.progress;
    if (!progress) {
      console.warn('[DiscogsProxy] Invalid progress message format');
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
   * Handle sync completion from server
   */
  private handleSyncComplete(message: { data?: { sessionId?: string } }): void {
    console.log('[DiscogsProxy] Received sync complete notification', message);

    const sessionId = message.data?.sessionId;
    if (!sessionId) {
      console.warn('[DiscogsProxy] Invalid sync complete message format');
      return;
    }

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
  private handleSyncError(message: { data?: { sessionId?: string; error?: string } }): void {
    console.log('[DiscogsProxy] Received sync error notification', message);

    const { sessionId, error } = message.data || {};
    if (!sessionId || !error) {
      console.warn('[DiscogsProxy] Invalid sync error message format');
      return;
    }

    // Notify all error callbacks
    this.errorCallbacks.forEach(callback => {
      try {
        callback(sessionId, error);
      } catch (error) {
        console.error('[DiscogsProxy] Error in error callback', error);
      }
    });
  }
}

// Export a singleton instance
export const discogsProxyService = new DiscogsProxyService();