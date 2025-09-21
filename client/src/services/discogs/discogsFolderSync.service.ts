import { api } from '@services/api';
import {
  SyncStartResponse,
  DiscogsFoldersResponse,
  SyncResponseResult
} from '../../types/DiscogsFolderSync';

export class DiscogsFolderSyncService {
  /**
   * Start the folder sync process and get the Discogs URL
   */
  async startFolderSync(): Promise<SyncStartResponse> {
    console.log('[DiscogsFolderSync] Starting folder sync...');

    try {
      const response = await api.post<SyncStartResponse>('/discogs/sync/folders/start');
      console.log('[DiscogsFolderSync] Sync start response:', response);
      return response;
    } catch (error) {
      console.error('[DiscogsFolderSync] Failed to start folder sync:', error);
      throw error;
    }
  }

  /**
   * Fetch folders data from Discogs using the provided URL
   */
  async fetchDiscogsFolders(url: string): Promise<DiscogsFoldersResponse> {
    console.log('[DiscogsFolderSync] Fetching folders from Discogs:', url);

    try {
      // Create axios instance for direct Discogs API call (not through our backend)
      const response = await fetch(url, {
        method: 'GET',
        headers: {
          'User-Agent': 'WaugzeeApp/1.0',
          'Accept': 'application/vnd.discogs.v2.discogs+json',
        },
      });

      if (!response.ok) {
        throw new Error(`Discogs API request failed: ${response.status} ${response.statusText}`);
      }

      const data = await response.json() as DiscogsFoldersResponse;
      console.log('[DiscogsFolderSync] Folders fetched successfully:', data);
      return data;
    } catch (error) {
      console.error('[DiscogsFolderSync] Failed to fetch folders from Discogs:', error);
      throw error;
    }
  }

  /**
   * Send the folders data back to our backend
   */
  async sendFoldersResponse(foldersData: DiscogsFoldersResponse): Promise<SyncResponseResult> {
    console.log('[DiscogsFolderSync] Sending folders data to backend...');

    try {
      const response = await api.post<SyncResponseResult>('/discogs/sync/folders/response', foldersData);
      console.log('[DiscogsFolderSync] Folders data sent successfully:', response);
      return response;
    } catch (error) {
      console.error('[DiscogsFolderSync] Failed to send folders data:', error);
      throw error;
    }
  }

  /**
   * Complete folder sync flow - orchestrates all steps
   */
  async performCompleteSync(): Promise<{
    success: boolean;
    message: string;
    foldersCount?: number;
  }> {
    console.log('[DiscogsFolderSync] Starting complete sync flow...');

    try {
      // Step 1: Start sync and get Discogs URL
      const startResponse = await this.startFolderSync();

      // Step 2: Fetch folders from Discogs
      const foldersData = await this.fetchDiscogsFolders(startResponse.url);

      // Step 3: Send folders data to backend
      const syncResult = await this.sendFoldersResponse(foldersData);

      console.log('[DiscogsFolderSync] Complete sync flow successful');
      return {
        success: true,
        message: syncResult.message,
        foldersCount: foldersData.folders.length,
      };
    } catch (error) {
      console.error('[DiscogsFolderSync] Complete sync flow failed:', error);
      throw error;
    }
  }
}

// Export singleton instance
export const discogsFolderSyncService = new DiscogsFolderSyncService();