import { Component, Show, For } from "solid-js";
import { useUserData } from "@context/UserDataContext";
import { useApiPut } from "@services/apiHooks";
import { USER_ENDPOINTS } from "@constants/api.constants";
import type { UpdateSelectedFolderRequest, UpdateSelectedFolderResponse } from "src/types/User";
import styles from "./FolderSelector.module.scss";

interface FolderSelectorProps {
  class?: string;
  showCounts?: boolean;
  label?: string;
  navbar?: boolean; // Simplified navbar mode
}

export const FolderSelector: Component<FolderSelectorProps> = (props) => {
  const userData = useUserData();

  const user = userData.user;
  const folders = userData.folders;

  const updateFolderMutation = useApiPut<UpdateSelectedFolderResponse, UpdateSelectedFolderRequest>(
    USER_ENDPOINTS.ME_FOLDER,
    undefined,
    {
      invalidateQueries: [["user"]],
      successMessage: (data, variables) => {
        const folderName = folders().find((f) => f.id === variables.folderId)?.name || "Unknown";
        return `Folder changed to "${folderName}"`;
      },
      errorMessage: "Failed to update folder selection",
    }
  );

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

  const handleFolderChange = (folderId: number) => {
    if (updateFolderMutation.isPending) return;

    updateFolderMutation.mutate({ folderId });
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
                  disabled={updateFolderMutation.isPending}
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
                    when={updateFolderMutation.isPending}
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
              disabled={updateFolderMutation.isPending}
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
                when={updateFolderMutation.isPending}
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
