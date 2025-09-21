// Types for the new Discogs folder sync functionality

export interface SyncStartResponse {
  url: string;
  message: string;
}

export interface DiscogsFoldersResponse {
  folders: Array<{
    id: number;
    name: string;
    count: number;
    resource_url: string;
  }>;
}

export interface SyncResponseResult {
  message: string;
  status: string;
}

export interface FolderSyncStatus {
  isLoading: boolean;
  error: string | null;
  success: boolean;
  stage: 'idle' | 'starting' | 'fetching' | 'sending' | 'completed' | 'error';
  message: string;
}