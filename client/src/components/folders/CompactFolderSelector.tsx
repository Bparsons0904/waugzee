import { ChevronDownIcon } from "@components/icons/ChevronDownIcon";
import { LoadingSpinner } from "@components/icons/LoadingSpinner";
import { type Accessor, type Component, For, Match, Show, Switch } from "solid-js";
import type { Folder } from "src/types/User";
import styles from "./FolderSelector.module.scss";

interface CompactFolderSelectorProps {
  folders: Accessor<Folder[]>;
  selectedFolderId: Accessor<number | undefined>;
  selectedFolder: Accessor<Folder | null>;
  handleFolderChange: (folderId: number) => void;
  isLoading: boolean;
}

export const CompactFolderSelector: Component<CompactFolderSelectorProps> = (props) => {
  const { folders, selectedFolderId, selectedFolder, handleFolderChange, isLoading } = props;

  return (
    <div class={styles.folderSelector}>
      <Switch>
        <Match when={folders().length <= 0}>
          <div class={styles.noFolders}>
            <span class={styles.noFoldersText}>No folders available</span>
          </div>
        </Match>

        <Match when={folders().length > 0}>
          <div class={styles.compactWrapper}>
            <div class={styles.currentViewing}>
              Viewing:
              <Show
                when={selectedFolder()}
                fallback={<span class={styles.placeholder}>Select folder</span>}
              >
                <strong>{selectedFolder()?.name}</strong>
                <Show when={selectedFolder()?.count}>
                  <span class={styles.itemCount}>({selectedFolder()?.count} items)</span>
                </Show>
              </Show>
            </div>

            <div class={styles.compactSelector}>
              <select
                class={styles.folderSelect}
                value={selectedFolderId() || selectedFolder()?.id || ""}
                disabled={isLoading}
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
                      {folder.name} ({folder.count})
                    </option>
                  )}
                </For>
              </select>
              <div class={styles.selectIcon}>
                <Show when={isLoading} fallback={<ChevronDownIcon />}>
                  <LoadingSpinner />
                </Show>
              </div>
            </div>
          </div>
        </Match>
      </Switch>
    </div>
  );
};
