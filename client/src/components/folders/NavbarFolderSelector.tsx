import { ChevronDownIcon } from "@components/icons/ChevronDownIcon";
import { LoadingSpinner } from "@components/icons/LoadingSpinner";
import clsx from "clsx";
import { type Accessor, type Component, For, Match, Show, Switch } from "solid-js";
import type { Folder } from "src/types/User";
import styles from "./FolderSelector.module.scss";

interface NavbarFolderSelectorProps {
  folders: Accessor<Folder[]>;
  selectedFolderId: Accessor<number | undefined>;
  selectedFolder: Accessor<Folder | null>;
  handleFolderChange: (folderId: number) => void;
  isLoading: boolean;
}

export const NavbarFolderSelector: Component<NavbarFolderSelectorProps> = (props) => {
  const { folders, selectedFolderId, selectedFolder, handleFolderChange, isLoading } = props;

  return (
    <div class={clsx(styles.folderSelector, styles.navbarMode)}>
      <Switch>
        <Match when={folders().length <= 0}>
          <div class={styles.noFolders}>
            <span class={styles.noFoldersText}>No folders available</span>
          </div>
        </Match>

        <Match when={folders().length > 0}>
          <div class={styles.navbarSelector}>
            <select
              class={styles.navbarSelect}
              value={selectedFolderId() || selectedFolder()?.id || ""}
              disabled={isLoading}
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
                    {folder.name} ({folder.count})
                  </option>
                )}
              </For>
            </select>
            <div class={styles.navbarSelectIcon}>
              <Show when={isLoading} fallback={<ChevronDownIcon size={14} />}>
                <LoadingSpinner class={styles.navbarLoadingSpinner} />
              </Show>
            </div>
          </div>
        </Match>
      </Switch>
    </div>
  );
};
