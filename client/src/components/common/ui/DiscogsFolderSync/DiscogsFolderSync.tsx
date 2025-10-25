import { Button } from "@components/common/ui/Button/Button";
import { useToast } from "@context/ToastContext";
import type { Component } from "solid-js";
import styles from "./DiscogsFolderSync.module.scss";

interface DiscogsFolderSyncProps {
  variant?: "primary" | "secondary";
  size?: "sm" | "md" | "lg";
  onSyncComplete?: (foldersCount: number) => void;
  onSyncError?: (error: string) => void;
}

export const DiscogsFolderSync: Component<DiscogsFolderSyncProps> = (props) => {
  const toast = useToast();

  const handleSync = () => {
    toast.showInfo("Folder sync will be available soon with the new simplified architecture");
  };

  return (
    <div class={styles.syncContainer}>
      <Button
        variant={props.variant || "primary"}
        size={props.size || "md"}
        onClick={handleSync}
        class={styles.syncButton}
      >
        Sync Coming Soon
      </Button>
    </div>
  );
};
