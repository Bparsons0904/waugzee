import { Component, createSignal, Show, For } from "solid-js";
import { useAuth } from "@context/AuthContext";
import { updateSelectedFolder } from "@services/user.service";
import { useToast } from "@context/ToastContext";
import { Button } from "@components/common/ui/Button/Button";
import type { Folder } from "@types/User";
import styles from "./FolderSelector.module.scss";

interface FolderSelectorProps {
  class?: string;
  variant?: "compact" | "detailed";
  showCounts?: boolean;
}

export const FolderSelector: Component<FolderSelectorProps> = (props) => {
  const auth = useAuth();
  const toast = useToast();
  const [isUpdating, setIsUpdating] = createSignal(false);

  const user = auth.user;
  const folders = auth.folders;

  const selectedFolderId = () => user()?.configuration?.selectedFolderId;

  const selectedFolder = () => {
    const folderId = selectedFolderId();
    if (!folderId) return null;
    return folders().find(folder => folder.id === folderId) || null;
  };

  const handleFolderChange = async (folderId: number) => {
    if (isUpdating()) return;

    const currentUserId = user()?.id;
    if (!currentUserId) {
      toast.showError("Authentication required");
      return;
    }

    setIsUpdating(true);

    try {
      const response = await updateSelectedFolder({ folderId });

      // Update user in auth context with optimistic update
      auth.updateUser(response.user);

      const selectedFolderName = folders().find(f => f.id === folderId)?.name || 'Unknown';
      toast.showSuccess(`Folder changed to "${selectedFolderName}"`);

    } catch (error) {
      console.error("Failed to update selected folder:", error);
      toast.showError("Failed to update folder selection");
    } finally {
      setIsUpdating(false);
    }
  };

  return (
    <div class={`${styles.folderSelector} ${props.class || ""}`}>
      <Show when={folders().length > 0} fallback={
        <div class={styles.noFolders}>
          <p class={styles.noFoldersText}>No folders available</p>
          <p class={styles.noFoldersSubtext}>Connect your Discogs account to view your folders</p>
        </div>
      }>
        <div class={styles.selectorHeader}>
          <h3 class={styles.selectorTitle}>Collection Folder</h3>
          <Show when={selectedFolder()}>
            <span class={styles.currentSelection}>
              Current: <strong>{selectedFolder()?.name}</strong>
              <Show when={props.showCounts !== false && selectedFolder()?.count}>
                <span class={styles.folderCount}>({selectedFolder()?.count} items)</span>
              </Show>
            </span>
          </Show>
        </div>

        <Show when={props.variant === "compact"} fallback={
          <div class={styles.folderGrid}>
            <For each={folders()}>
              {(folder) => (
                <div
                  class={`${styles.folderCard} ${folder.id === selectedFolderId() ? styles.folderCardSelected : ""}`}
                >
                  <div class={styles.folderInfo}>
                    <h4 class={styles.folderName}>{folder.name}</h4>
                    <Show when={props.showCounts !== false}>
                      <p class={styles.folderMeta}>
                        {folder.count} items
                        <Show when={folder.public}>
                          <span class={styles.publicBadge}>Public</span>
                        </Show>
                      </p>
                    </Show>
                  </div>
                  <Show when={folder.id !== selectedFolderId()}>
                    <Button
                      variant="secondary"
                      size="sm"
                      disabled={isUpdating()}
                      onClick={() => handleFolderChange(folder.id)}
                    >
                      Select
                    </Button>
                  </Show>
                  <Show when={folder.id === selectedFolderId()}>
                    <div class={styles.selectedBadge}>
                      <svg
                        width="16"
                        height="16"
                        viewBox="0 0 16 16"
                        fill="none"
                        xmlns="http://www.w3.org/2000/svg"
                      >
                        <path
                          d="M13.5 4L6 11.5L2.5 8"
                          stroke="currentColor"
                          stroke-width="2"
                          stroke-linecap="round"
                          stroke-linejoin="round"
                        />
                      </svg>
                      Selected
                    </div>
                  </Show>
                </div>
              )}
            </For>
          </div>
        }>
          <div class={styles.compactSelector}>
            <select
              class={styles.folderSelect}
              value={selectedFolderId() || ""}
              disabled={isUpdating()}
              onChange={(e) => {
                const value = parseInt(e.target.value);
                if (!isNaN(value)) {
                  handleFolderChange(value);
                }
              }}
            >
              <option value="" disabled>Choose a folder</option>
              <For each={folders()}>
                {(folder) => (
                  <option value={folder.id}>
                    {folder.name}
                    <Show when={props.showCounts !== false}>
                      {` (${folder.count} items)`}
                    </Show>
                  </option>
                )}
              </For>
            </select>
            <div class={styles.selectIcon}>
              <svg
                width="16"
                height="16"
                viewBox="0 0 16 16"
                fill="none"
                xmlns="http://www.w3.org/2000/svg"
              >
                <path
                  d="M4 6L8 10L12 6"
                  stroke="currentColor"
                  stroke-width="1.5"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
              </svg>
            </div>
          </div>
        </Show>

        <Show when={isUpdating()}>
          <div class={styles.loadingOverlay}>
            <div class={styles.spinner}></div>
            <span class={styles.loadingText}>Updating folder...</span>
          </div>
        </Show>
      </Show>
    </div>
  );
};