import VinylIcon from "@components/icons/VinylIcon";
import { type Component, Show } from "solid-js";
import styles from "./SyncStatusIndicator.module.scss";

interface SyncStatusIndicatorProps {
  isSyncing: boolean;
}

export const SyncStatusIndicator: Component<SyncStatusIndicatorProps> = (props) => {
  return (
    <Show when={props.isSyncing}>
      <div class={styles.syncIndicator}>
        <VinylIcon size={20} class={styles.syncIcon} />
        <span class={styles.syncText}>Syncing Collection...</span>
      </div>
    </Show>
  );
};
