import { Button } from "@components/common/ui/Button/Button";
import { ConfirmationModal } from "@components/common/ui/ConfirmationModal/ConfirmationModal";
import { LoadingSpinner } from "@components/icons/LoadingSpinner";
import { TrashIcon } from "@components/icons/TrashIcon";
import type { StoredFileInfo } from "@models/Admin";
import { useCleanupFiles, useStoredFiles } from "@services/apiHooks";
import { formatBytes, formatTimestamp } from "@utils/formatters";
import { createSignal, For, Show } from "solid-js";
import styles from "./FileManagementSection.module.scss";

export function FileManagementSection() {
  const [showCleanupModal, setShowCleanupModal] = createSignal(false);

  const filesQuery = useStoredFiles();
  const cleanupMutation = useCleanupFiles({
    onSuccess: () => {
      setShowCleanupModal(false);
    },
  });

  const hasFiles = () => {
    const data = filesQuery.data;
    return data?.files && data.files.length > 0;
  };

  const getFileTypeLabel = (file: StoredFileInfo) => {
    if (file.is_xml && file.is_gz) return "XML.GZ";
    if (file.is_xml) return "XML";
    if (file.is_gz) return "GZ";
    return "Unknown";
  };

  const handleCleanup = () => {
    cleanupMutation.mutate();
  };

  return (
    <section class={styles.section}>
      <div class={styles.header}>
        <div>
          <h2 class={styles.sectionTitle}>File Management</h2>
          <p class={styles.sectionDescription}>
            Manage downloaded and processed Discogs data files
          </p>
        </div>
        <Button
          onClick={() => setShowCleanupModal(true)}
          disabled={!hasFiles() || filesQuery.isLoading}
          variant="danger"
        >
          <TrashIcon size={16} />
          Cleanup Files
        </Button>
      </div>

      <Show when={!filesQuery.isLoading} fallback={<LoadingState />}>
        <Show when={hasFiles()} fallback={<NoFiles />}>
          <div class={styles.content}>
            <div class={styles.statsBar}>
              <div class={styles.statItem}>
                <span class={styles.statLabel}>Total Files:</span>
                <span class={styles.statValue}>{filesQuery.data?.total_count}</span>
              </div>
              <div class={styles.statItem}>
                <span class={styles.statLabel}>Total Size:</span>
                <span class={styles.statValue}>
                  {filesQuery.data?.total_size ? formatBytes(filesQuery.data.total_size) : "0 B"}
                </span>
              </div>
            </div>

            <div class={styles.filesTable}>
              <div class={styles.tableHeader}>
                <div class={styles.headerCell}>File Path</div>
                <div class={styles.headerCell}>Type</div>
                <div class={styles.headerCell}>Size</div>
                <div class={styles.headerCell}>Modified</div>
              </div>
              <div class={styles.tableBody}>
                <For each={filesQuery.data?.files}>
                  {(file) => (
                    <div class={styles.tableRow}>
                      <div class={styles.filePath} title={file.path}>
                        {file.path}
                      </div>
                      <div class={styles.fileType}>
                        <span class={styles.typeBadge}>{getFileTypeLabel(file)}</span>
                      </div>
                      <div class={styles.fileSize}>{formatBytes(file.size)}</div>
                      <div class={styles.fileModified}>{formatTimestamp(file.modified_at)}</div>
                    </div>
                  )}
                </For>
              </div>
            </div>
          </div>
        </Show>
      </Show>

      <ConfirmationModal
        isOpen={showCleanupModal()}
        onClose={() => setShowCleanupModal(false)}
        onConfirm={handleCleanup}
        title="Cleanup Files"
        message="Are you sure you want to delete all stored files? This will remove all downloaded and processed files from the server. This action cannot be undone."
        isLoading={cleanupMutation.isPending}
        variant="danger"
      />
    </section>
  );
}

const LoadingState = () => (
  <div class={styles.loadingContainer}>
    <LoadingSpinner />
    <p class={styles.loadingText}>Loading stored files...</p>
  </div>
);

const NoFiles = () => (
  <div class={styles.emptyContainer}>
    <p class={styles.emptyText}>
      No stored files found. Files will appear here after downloads are processed.
    </p>
  </div>
);
