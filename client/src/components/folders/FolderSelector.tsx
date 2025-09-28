import { Component, createSignal, Show, For } from "solid-js";
import { useAuth } from "@context/AuthContext";
import { updateSelectedFolder } from "@services/user.service";
import { useToast } from "@context/ToastContext";
import styles from "./FolderSelector.module.scss";

interface FolderSelectorProps {
  class?: string;
  showCounts?: boolean;
  label?: string;
  navbar?: boolean; // Simplified navbar mode
}

export const FolderSelector: Component<FolderSelectorProps> = (props) => {
  const auth = useAuth();
  const toast = useToast();
  const [isUpdating, setIsUpdating] = createSignal(false);

  // Claude I don't think we should destructure auth here, won't this break reactivity?
  const user = auth.user;
  const folders = auth.folders;

  const selectedFolderId = () => user()?.configuration?.selectedFolderId;

  const selectedFolder = () => {
    const folderId = selectedFolderId();
    if (!folderId) {
      // Default to first folder if no selection and folders exist
      const allFolders = folders();
      return allFolders.length > 0 ? allFolders[0] : null;
    }
    return folders().find((folder) => folder.id === folderId) || null;
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

      const selectedFolderName =
        folders().find((f) => f.id === folderId)?.name || "Unknown";
      toast.showSuccess(`Folder changed to "${selectedFolderName}"`);
    } catch (error) {
      console.error("Failed to update selected folder:", error);
      toast.showError("Failed to update folder selection");
    } finally {
      setIsUpdating(false);
    }
  };

  return (
    <div class={`${styles.folderSelector} ${props.navbar ? styles.navbarMode : ""} ${props.class || ""}`}>
      <Show
        when={folders().length > 0}
        fallback={
          <div class={styles.noFolders}>
            <span class={styles.noFoldersText}>No folders available</span>
          </div>
        }
      >
        <Show
          when={props.navbar}
          fallback={
            // Original compact mode for non-navbar use
            <div class={styles.compactWrapper}>
              <Show when={props.label}>
                <label class={styles.dropdownLabel}>
                  {props.label}:
                </label>
              </Show>

              <div class={styles.currentViewing}>
                Viewing:
                <Show
                  when={selectedFolder()}
                  fallback={<span class={styles.placeholder}>Select folder</span>}
                >
                  <strong>{selectedFolder()?.name}</strong>
                  <Show when={props.showCounts !== false && selectedFolder()?.count}>
                    <span class={styles.itemCount}>
                      ({selectedFolder()?.count} items)
                    </span>
                  </Show>
                </Show>
              </div>

              <div class={styles.compactSelector}>
                <select
                  class={styles.folderSelect}
                  value={selectedFolderId() || selectedFolder()?.id || ""}
                  disabled={isUpdating()}
                  onChange={(e) => {
                    const value = parseInt(e.target.value);
                    if (!isNaN(value)) {
                      handleFolderChange(value);
                    }
                  }}
                >
                  <option value="" disabled>
                    Choose a folder
                  </option>
                  <For each={folders()}>
                    {(folder) => (
                      <option value={folder.id}>
                        {folder.name}
                        <Show when={props.showCounts !== false}>
                          {` (${folder.count})`}
                        </Show>
                      </option>
                    )}
                  </For>
                </select>
                <div class={styles.selectIcon}>
                  <Show
                    when={isUpdating()}
                    fallback={
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
                    }
                  >
                    <div class={styles.loadingSpinner}></div>
                  </Show>
                </div>
              </div>
            </div>
          }
        >
          {/* Simplified navbar mode - just the dropdown */}
          <div class={styles.navbarSelector}>
            <select
              class={styles.navbarSelect}
              value={selectedFolderId() || selectedFolder()?.id || ""}
              disabled={isUpdating()}
              onChange={(e) => {
                const value = parseInt(e.target.value);
                if (!isNaN(value)) {
                  handleFolderChange(value);
                }
              }}
            >
              <For each={folders()}>
                {(folder) => (
                  <option value={folder.id}>
                    {folder.name}
                    <Show when={props.showCounts !== false}>
                      {` (${folder.count})`}
                    </Show>
                  </option>
                )}
              </For>
            </select>
            <div class={styles.navbarSelectIcon}>
              <Show
                when={isUpdating()}
                fallback={
                  <svg
                    width="14"
                    height="14"
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
                }
              >
                <div class={styles.navbarLoadingSpinner}></div>
              </Show>
            </div>
          </div>
        </Show>
      </Show>
    </div>
  );
};
