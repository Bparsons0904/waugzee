import { Button } from "@components/common/ui/Button/Button";
import { ConfirmationModal } from "@components/common/ui/ConfirmationModal/ConfirmationModal";
import { LoadingSpinner } from "@components/icons/LoadingSpinner";
import { useAdminWebSocket } from "@hooks/useAdminWebSocket";
import {
  useDownloadStatus,
  useResetDownload,
  useTriggerDownload,
  useTriggerReprocess,
} from "@services/apiHooks";
import {
  formatFileType,
  formatProcessingLabel,
  formatStatusLabel,
  formatTimestamp,
  getProcessingStatusColor,
} from "@utils/admin.utils";
import clsx from "clsx";
import { createSignal, For, Show } from "solid-js";
import styles from "./MonthlyDownloadsSection.module.scss";
import { ProcessingStepRow } from "./ProcessingStepRow";
import { ProgressBar } from "./ProgressBar";

export function MonthlyDownloadsSection() {
  const statusQuery = useDownloadStatus();
  const triggerDownload = useTriggerDownload();
  const triggerReprocess = useTriggerReprocess();
  const resetDownload = useResetDownload();
  const { progress, processingProgress } = useAdminWebSocket();
  const [showTriggerModal, setShowTriggerModal] = createSignal(false);
  const [showReprocessModal, setShowReprocessModal] = createSignal(false);
  const [showResetModal, setShowResetModal] = createSignal(false);

  const processingSteps = () => {
    const steps = statusQuery.data?.processing_steps;
    if (!steps) return [];
    return Object.entries(steps).map(([step, status]) => ({
      step,
      status,
    }));
  };

  const canTriggerDownload = () => {
    const status = statusQuery.data?.status;
    return status !== "downloading" && status !== "processing";
  };

  const canReprocess = () => {
    const status = statusQuery.data?.status;
    return status === "ready_for_processing" || status === "completed" || status === "failed";
  };

  const canReset = () => {
    const status = statusQuery.data?.status;
    return status === "downloading" || status === "processing" || status === "failed";
  };

  return (
    <section class={styles.section}>
      <h2 class={styles.sectionTitle}>Monthly Downloads Management</h2>

      <Show when={!statusQuery.isLoading} fallback={<LoadingState />}>
        <Show
          when={statusQuery.data}
          fallback={
            <div class={styles.emptyContainer}>
              <p class={styles.emptyText}>
                No download records found. Trigger a download to get started.
              </p>
              <Button onClick={() => setShowTriggerModal(true)}>Trigger New Download</Button>
            </div>
          }
        >
          {(data) => (
            <div class={styles.content}>
              <Show when={progress() && progress()?.stage !== "completed"}>
                <div class={styles.progressSection}>
                  <h3>Download Progress</h3>
                  <ProgressBar
                    label={formatFileType(progress()?.fileType || "Unknown")}
                    percentage={progress()?.percentage || 0}
                  />
                  <Show when={progress()?.errorMessage}>
                    <p class={styles.errorMessage}>{progress()?.errorMessage}</p>
                  </Show>
                </div>
              </Show>

              <Show when={processingProgress() && processingProgress()?.stage !== "completed"}>
                <div class={styles.progressSection}>
                  <h3>Processing Progress</h3>
                  <ProgressBar
                    label={formatProcessingLabel(
                      processingProgress()?.fileType || "Unknown",
                      processingProgress()?.step,
                    )}
                    subLabel={
                      processingProgress()?.totalFiles
                        ? `${processingProgress()?.filesProcessed || 0} / ${processingProgress()?.totalFiles} files`
                        : undefined
                    }
                    percentage={processingProgress()?.percentage || 0}
                  />
                  <Show when={processingProgress()?.errorMessage}>
                    <p class={styles.errorMessage}>{processingProgress()?.errorMessage}</p>
                  </Show>
                </div>
              </Show>

              <div class={styles.statusOverview}>
                <div class={styles.statusHeader}>
                  <h3 class={styles.yearMonth}>{data().year_month}</h3>
                  <span
                    class={clsx(
                      styles.statusBadge,
                      styles[getProcessingStatusColor(data().status)],
                    )}
                  >
                    {formatStatusLabel(data().status)}
                  </span>
                </div>

                <div class={styles.statsGrid}>
                  <div class={styles.statCard}>
                    <div class={styles.statLabel}>Started</div>
                    <div class={styles.statValue}>{formatTimestamp(data().started_at)}</div>
                  </div>
                  <div class={styles.statCard}>
                    <div class={styles.statLabel}>Download Complete</div>
                    <div class={styles.statValue}>
                      {formatTimestamp(data().download_completed_at)}
                    </div>
                  </div>
                  <div class={styles.statCard}>
                    <div class={styles.statLabel}>Processing Complete</div>
                    <div class={styles.statValue}>
                      {formatTimestamp(data().processing_completed_at)}
                    </div>
                  </div>
                  <div class={styles.statCard}>
                    <div class={styles.statLabel}>Retry Count</div>
                    <div class={styles.statValue}>{data().retry_count}</div>
                  </div>
                </div>

                <Show when={data().error_message}>
                  <div class={styles.errorMessage}>
                    <strong>Error:</strong> {data().error_message}
                  </div>
                </Show>
              </div>

              <Show when={processingSteps().length > 0}>
                <div class={styles.processingSection}>
                  <h3 class={styles.subsectionTitle}>Processing Steps</h3>
                  <div class={styles.processingTable}>
                    <For each={processingSteps()}>
                      {(item) => <ProcessingStepRow step={item.step} status={item.status} />}
                    </For>
                  </div>
                </div>
              </Show>

              <div class={styles.actionsSection}>
                <Button onClick={() => setShowTriggerModal(true)} disabled={!canTriggerDownload()}>
                  Trigger New Download
                </Button>

                <Button
                  onClick={() => setShowReprocessModal(true)}
                  disabled={!canReprocess()}
                  variant="secondary"
                >
                  Reprocess Data
                </Button>

                <Button
                  onClick={() => setShowResetModal(true)}
                  disabled={!canReset()}
                  variant="danger"
                >
                  Reset Download
                </Button>
              </div>
            </div>
          )}
        </Show>
      </Show>

      <ConfirmationModal
        isOpen={showTriggerModal()}
        onClose={() => setShowTriggerModal(false)}
        onConfirm={() => {
          triggerDownload.mutate();
          setShowTriggerModal(false);
        }}
        title="Trigger Download"
        message="Are you sure you want to trigger a new download? This will start downloading the latest Discogs data files."
        isLoading={triggerDownload.isPending}
      />

      <ConfirmationModal
        isOpen={showReprocessModal()}
        onClose={() => setShowReprocessModal(false)}
        onConfirm={() => {
          triggerReprocess.mutate();
          setShowReprocessModal(false);
        }}
        title="Reprocess Data"
        message="Are you sure you want to reprocess the data? This will reset processing steps and start from the beginning."
        isLoading={triggerReprocess.isPending}
        variant="danger"
      />

      <ConfirmationModal
        isOpen={showResetModal()}
        onClose={() => setShowResetModal(false)}
        onConfirm={() => {
          resetDownload.mutate();
          setShowResetModal(false);
        }}
        title="Reset Download"
        message="Are you sure you want to reset this download? This will stop any in-progress downloads, delete all downloaded files, and reset the status. You'll need to trigger a new download afterward."
        isLoading={resetDownload.isPending}
        variant="danger"
      />
    </section>
  );
}

const LoadingState = () => (
  <div class={styles.loadingContainer}>
    <LoadingSpinner />
    <p class={styles.loadingText}>Loading download status...</p>
  </div>
);
