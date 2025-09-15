import { Component, createSignal, For, Show, onMount } from "solid-js";
import { useAuth } from "@context/AuthContext";
import styles from "./Admin.module.scss";
import { Button } from "@components/common/ui/Button/Button";
import { Card } from "@components/common/ui/Card/Card";
import { TextInput } from "@components/common/forms/TextInput/TextInput";
import { Checkbox } from "@components/common/forms/Checkbox/Checkbox";
import { ADMIN_ENDPOINTS } from "@constants/api.constants";
import { api } from "@services/api";

interface ProcessingLimits {
  maxRecords?: number;
  maxBatchSize?: number;
  progressInterval?: number;
  enableDebugLogging?: boolean;
}

interface ProcessingRequest {
  yearMonth: string;
  fileTypes: string[];
  limits?: ProcessingLimits;
}

interface ProcessingStatusResponse {
  yearMonth: string;
  status: string;
  stats?: {
    totalRecords: number;
    labelsProcessed: number;
    artistsProcessed: number;
    mastersProcessed: number;
    releasesProcessed: number;
    failedRecords: number;
  };
  error?: string;
  startedAt?: string;
}

const Admin: Component = () => {
  const { user } = useAuth();

  // Form state
  const [yearMonth, setYearMonth] = createSignal(
    new Date().toISOString().slice(0, 7),
  ); // YYYY-MM format
  const [selectedFileTypes, setSelectedFileTypes] = createSignal<string[]>([
    "labels",
  ]);
  const [maxRecords, setMaxRecords] = createSignal<string>("");
  const [maxBatchSize, setMaxBatchSize] = createSignal<string>("2000");
  const [progressInterval, setProgressInterval] = createSignal<string>("1000");
  const [enableDebugLogging, setEnableDebugLogging] =
    createSignal<boolean>(true);

  // State
  const [isProcessing, setIsProcessing] = createSignal(false);
  const [processingStatuses, setProcessingStatuses] = createSignal<
    ProcessingStatusResponse[]
  >([]);
  const [isLoading, setIsLoading] = createSignal(true);
  const [error, setError] = createSignal<string>("");

  const fileTypes = [
    { id: "labels", label: "Labels", description: "Record label information" },
    {
      id: "artists",
      label: "Artists",
      description: "Artist data and biographies",
    },
    {
      id: "masters",
      label: "Masters",
      description: "Master release information",
    },
    {
      id: "releases",
      label: "Releases",
      description: "Individual release data",
    },
  ];

  onMount(async () => {
    await loadProcessingStatuses();
  });

  const loadProcessingStatuses = async () => {
    try {
      setIsLoading(true);
      const response = await api.get<ProcessingStatusResponse[]>(
        ADMIN_ENDPOINTS.DISCOGS_STATUS(),
      );
      setProcessingStatuses(response);
    } catch (err) {
      setError("Failed to load processing statuses");
      console.error("Error loading processing statuses:", err);
    } finally {
      setIsLoading(false);
    }
  };

  const handleFileTypeToggle = (fileType: string) => {
    const current = selectedFileTypes();
    if (current.includes(fileType)) {
      setSelectedFileTypes(current.filter((t) => t !== fileType));
    } else {
      setSelectedFileTypes([...current, fileType]);
    }
  };

  const buildProcessingRequest = (): ProcessingRequest => {
    const limits: ProcessingLimits = {};

    if (maxRecords()) {
      const parsed = parseInt(maxRecords(), 10);
      if (!isNaN(parsed) && parsed > 0) {
        limits.maxRecords = parsed;
      }
    }

    if (maxBatchSize()) {
      const parsed = parseInt(maxBatchSize(), 10);
      if (!isNaN(parsed) && parsed > 0) {
        limits.maxBatchSize = parsed;
      }
    }

    if (progressInterval()) {
      const parsed = parseInt(progressInterval(), 10);
      if (!isNaN(parsed) && parsed > 0) {
        limits.progressInterval = parsed;
      }
    }

    if (enableDebugLogging()) {
      limits.enableDebugLogging = true;
    }

    return {
      yearMonth: yearMonth(),
      fileTypes: selectedFileTypes(),
      limits: Object.keys(limits).length > 0 ? limits : undefined,
    };
  };

  const handleStartProcessing = async () => {
    if (selectedFileTypes().length === 0) {
      setError("Please select at least one file type to process");
      return;
    }

    try {
      setIsProcessing(true);
      setError("");

      const request = buildProcessingRequest();
      await api.post(ADMIN_ENDPOINTS.DISCOGS_PROCESS, request);

      // Refresh statuses after starting processing
      setTimeout(() => loadProcessingStatuses(), 1000);
    } catch (err: any) {
      setError(err.response?.data?.error || "Failed to start processing");
      console.error("Error starting processing:", err);
    } finally {
      setIsProcessing(false);
    }
  };

  const handleStartDirectProcessing = async () => {
    if (selectedFileTypes().length === 0) {
      setError("Please select at least one file type to process");
      return;
    }

    try {
      setIsProcessing(true);
      setError("");

      const request = buildProcessingRequest();
      await api.post(ADMIN_ENDPOINTS.DISCOGS_PROCESS_DIRECT, request);

      // Refresh statuses after starting processing
      setTimeout(() => loadProcessingStatuses(), 1000);
    } catch (err: any) {
      setError(
        err.response?.data?.error || "Failed to start direct processing",
      );
      console.error("Error starting direct processing:", err);
    } finally {
      setIsProcessing(false);
    }
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case "completed":
        return "success";
      case "processing":
        return "warning";
      case "failed":
        return "danger";
      default:
        return "info";
    }
  };

  return (
    <div class={styles.admin}>
      <header class={styles.header}>
        <h1>Admin Panel</h1>
        <p>
          Welcome, {user()?.firstName || "Admin"}. Manage Discogs data
          processing.
        </p>
      </header>

      <div class={styles.content}>
        <div class={styles.grid}>
          {/* Processing Controls */}
          <Card class={styles.processingCard}>
            <h2>Manual Processing Control</h2>

            <div class={styles.formSection}>
              <label class={styles.label}>Year-Month (YYYY-MM)</label>
              <TextInput
                value={yearMonth()}
                onInput={(value) => setYearMonth(value)}
                placeholder="2024-01"
                pattern="[0-9]{4}-[0-9]{2}"
              />
            </div>

            <div class={styles.formSection}>
              <label class={styles.label}>File Types to Process</label>
              <div class={styles.checkboxGrid}>
                <For each={fileTypes}>
                  {(fileType) => (
                    <div class={styles.checkboxItem}>
                      <Checkbox
                        name={fileType.id}
                        checked={selectedFileTypes().includes(fileType.id)}
                        onChange={() => handleFileTypeToggle(fileType.id)}
                        label={fileType.label}
                      />
                      <span class={styles.description}>
                        {fileType.description}
                      </span>
                    </div>
                  )}
                </For>
              </div>
            </div>

            <div class={styles.formSection}>
              <label class={styles.label}>Processing Limits</label>
              <div class={styles.limitsGrid}>
                <TextInput
                  label="Max Records"
                  value={maxRecords()}
                  onInput={(value) => setMaxRecords(value)}
                  placeholder="Leave empty for no limit"
                />
                <TextInput
                  label="Batch Size"
                  value={maxBatchSize()}
                  onInput={(value) => setMaxBatchSize(value)}
                  placeholder="2000"
                />
                <TextInput
                  label="Progress Interval"
                  value={progressInterval()}
                  onInput={(value) => setProgressInterval(value)}
                  placeholder="1000"
                />
              </div>
            </div>

            <div class={styles.formSection}>
              <Checkbox
                name="enableDebugLogging"
                checked={enableDebugLogging()}
                onChange={(checked) => setEnableDebugLogging(checked)}
                label="Enable Enhanced Debug Logging"
              />
              <span class={styles.description}>
                Log detailed XML data and conversion info for debugging (first
                record from each batch)
              </span>
            </div>

            <Show when={error()}>
              <div class={styles.error}>{error()}</div>
            </Show>

            <div class={styles.buttonGroup}>
              <Button
                onClick={handleStartProcessing}
                disabled={isProcessing() || selectedFileTypes().length === 0}
                variant="primary"
                class={styles.startButton}
              >
                {isProcessing() ? "Starting..." : "Start Processing"}
              </Button>

              <Button
                onClick={handleStartDirectProcessing}
                disabled={isProcessing() || selectedFileTypes().length === 0}
                variant="secondary"
                class={styles.directButton}
              >
                {isProcessing()
                  ? "Starting..."
                  : "Direct Process (Skip Checks)"}
              </Button>
            </div>
          </Card>

          {/* Processing Status */}
          <Card class={styles.statusCard}>
            <div class={styles.statusHeader}>
              <h2>Processing Status</h2>
              <Button
                onClick={loadProcessingStatuses}
                disabled={isLoading()}
                variant="secondary"
                size="sm"
              >
                {isLoading() ? "Loading..." : "Refresh"}
              </Button>
            </div>

            <Show when={isLoading()}>
              <div class={styles.loading}>Loading processing statuses...</div>
            </Show>

            <Show when={!isLoading() && processingStatuses().length === 0}>
              <div class={styles.empty}>No processing records found.</div>
            </Show>

            <Show when={!isLoading() && processingStatuses().length > 0}>
              <div class={styles.statusList}>
                <For each={processingStatuses()}>
                  {(status) => (
                    <div
                      class={`${styles.statusItem} ${styles[getStatusColor(status.status)]}`}
                    >
                      <div class={styles.statusItemHeader}>
                        <h3>{status.yearMonth}</h3>
                        <span
                          class={`${styles.statusBadge} ${styles[getStatusColor(status.status)]}`}
                        >
                          {status.status}
                        </span>
                      </div>

                      <Show when={status.stats}>
                        <div class={styles.stats}>
                          <div class={styles.statItem}>
                            <span>Total:</span>
                            <span>
                              {status.stats!.totalRecords.toLocaleString()}
                            </span>
                          </div>
                          <div class={styles.statItem}>
                            <span>Labels:</span>
                            <span>
                              {status.stats!.labelsProcessed.toLocaleString()}
                            </span>
                          </div>
                          <div class={styles.statItem}>
                            <span>Artists:</span>
                            <span>
                              {status.stats!.artistsProcessed.toLocaleString()}
                            </span>
                          </div>
                          <div class={styles.statItem}>
                            <span>Masters:</span>
                            <span>
                              {status.stats!.mastersProcessed.toLocaleString()}
                            </span>
                          </div>
                          <div class={styles.statItem}>
                            <span>Releases:</span>
                            <span>
                              {status.stats!.releasesProcessed.toLocaleString()}
                            </span>
                          </div>
                          <Show when={status.stats!.failedRecords > 0}>
                            <div class={styles.statItem}>
                              <span>Failed:</span>
                              <span class={styles.errorText}>
                                {status.stats!.failedRecords.toLocaleString()}
                              </span>
                            </div>
                          </Show>
                        </div>
                      </Show>

                      <Show when={status.error}>
                        <div class={styles.errorMessage}>{status.error}</div>
                      </Show>

                      <Show when={status.startedAt}>
                        <div class={styles.timestamp}>
                          Started:{" "}
                          {new Date(status.startedAt!).toLocaleString()}
                        </div>
                      </Show>
                    </div>
                  )}
                </For>
              </div>
            </Show>
          </Card>
        </div>
      </div>
    </div>
  );
};

export default Admin;
