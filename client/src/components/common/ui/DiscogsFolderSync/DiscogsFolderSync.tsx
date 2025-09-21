import { Component, createSignal, Show } from "solid-js";
import { Button } from "@components/common/ui/Button/Button";
import { useToast } from "@context/ToastContext";
import { discogsFolderSyncService } from "@services/discogs/discogsFolderSync.service";
import { FolderSyncStatus } from "../../../../types/DiscogsFolderSync";
import styles from "./DiscogsFolderSync.module.scss";

interface DiscogsFolderSyncProps {
  variant?: "primary" | "secondary";
  size?: "sm" | "md" | "lg";
  onSyncComplete?: (foldersCount: number) => void;
  onSyncError?: (error: string) => void;
}

export const DiscogsFolderSync: Component<DiscogsFolderSyncProps> = (props) => {
  const toast = useToast();

  const [syncStatus, setSyncStatus] = createSignal<FolderSyncStatus>({
    isLoading: false,
    error: null,
    success: false,
    stage: 'idle',
    message: '',
  });

  const updateSyncStatus = (updates: Partial<FolderSyncStatus>) => {
    setSyncStatus(prev => ({ ...prev, ...updates }));
  };

  const handleSync = async () => {
    console.log('[DiscogsFolderSync] Starting sync process...');

    // Reset state
    updateSyncStatus({
      isLoading: true,
      error: null,
      success: false,
      stage: 'starting',
      message: 'Initializing sync...',
    });

    try {
      // Stage 1: Starting sync
      updateSyncStatus({
        stage: 'starting',
        message: 'Getting sync URL from server...',
      });

      const startResponse = await discogsFolderSyncService.startFolderSync();
      console.log('[DiscogsFolderSync] Got sync URL:', startResponse.url);

      // Stage 2: Fetching from Discogs
      updateSyncStatus({
        stage: 'fetching',
        message: 'Fetching folders from Discogs...',
      });

      const foldersData = await discogsFolderSyncService.fetchDiscogsFolders(startResponse.url);
      console.log('[DiscogsFolderSync] Fetched folders:', foldersData.folders.length);

      // Stage 3: Sending to backend
      updateSyncStatus({
        stage: 'sending',
        message: `Sending ${foldersData.folders.length} folders to server...`,
      });

      const syncResult = await discogsFolderSyncService.sendFoldersResponse(foldersData);
      console.log('[DiscogsFolderSync] Sync completed:', syncResult);

      // Stage 4: Completed
      updateSyncStatus({
        isLoading: false,
        success: true,
        stage: 'completed',
        message: `Successfully synced ${foldersData.folders.length} folders!`,
      });

      // Show success toast
      toast.showSuccess(`Folder sync completed! Found ${foldersData.folders.length} folders.`);

      // Call completion callback
      props.onSyncComplete?.(foldersData.folders.length);

    } catch (error) {
      console.error('[DiscogsFolderSync] Sync failed:', error);

      const errorMessage = error instanceof Error ? error.message : 'Unknown error occurred';

      updateSyncStatus({
        isLoading: false,
        error: errorMessage,
        success: false,
        stage: 'error',
        message: `Sync failed: ${errorMessage}`,
      });

      // Show error toast
      toast.showError(`Folder sync failed: ${errorMessage}`);

      // Call error callback
      props.onSyncError?.(errorMessage);
    }
  };

  const getButtonText = () => {
    const status = syncStatus();
    if (!status.isLoading) {
      return status.success ? 'Sync Complete ✓' : 'Sync Now';
    }

    switch (status.stage) {
      case 'starting':
        return 'Starting...';
      case 'fetching':
        return 'Fetching...';
      case 'sending':
        return 'Syncing...';
      default:
        return 'Syncing...';
    }
  };

  const getButtonVariant = () => {
    const status = syncStatus();
    if (status.success) return 'secondary';
    if (status.error) return 'danger';
    return props.variant || 'primary';
  };

  return (
    <div class={styles.syncContainer}>
      <Button
        variant={getButtonVariant()}
        size={props.size || 'md'}
        disabled={syncStatus().isLoading}
        onClick={handleSync}
        class={styles.syncButton}
      >
        {getButtonText()}
      </Button>

      <Show when={syncStatus().isLoading || syncStatus().message}>
        <div class={styles.statusContainer}>
          <Show when={syncStatus().isLoading}>
            <div class={styles.loadingSpinner} />
          </Show>

          <span
            class={styles.statusMessage}
            classList={{
              [styles.success]: syncStatus().success,
              [styles.error]: !!syncStatus().error,
              [styles.loading]: syncStatus().isLoading,
            }}
          >
            {syncStatus().message}
          </span>
        </div>
      </Show>

      <Show when={syncStatus().error}>
        <div class={styles.errorContainer}>
          <span class={styles.errorIcon}>⚠️</span>
          <span class={styles.errorMessage}>{syncStatus().error}</span>
        </div>
      </Show>
    </div>
  );
};