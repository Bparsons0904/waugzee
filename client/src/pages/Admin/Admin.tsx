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
}

interface ProcessingRequest {
  fileTypes: string[];
  limits?: ProcessingLimits;
}

const Admin: Component = () => {
  const { user } = useAuth();

  // Form state
  const [selectedFileTypes, setSelectedFileTypes] = createSignal<string[]>([
    "labels",
  ]);
  const [maxRecords, setMaxRecords] = createSignal<string>("10");
  const [maxBatchSize, setMaxBatchSize] = createSignal<string>("2000");

  // State
  const [isProcessing, setIsProcessing] = createSignal(false);
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

    return {
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

      setError("Processing started successfully!");
    } catch (err: any) {
      setError(err.response?.data?.error || "Failed to start processing");
      console.error("Error starting processing:", err);
    } finally {
      setIsProcessing(false);
    }
  };

  const handleParseOnly = async () => {
    if (selectedFileTypes().length === 0) {
      setError("Please select at least one file type to parse");
      return;
    }

    try {
      setIsProcessing(true);
      setError("");

      const request = buildProcessingRequest();
      await api.post(ADMIN_ENDPOINTS.DISCOGS_PARSE, request);

      setError("Parse completed successfully!");
    } catch (err: any) {
      setError(err.response?.data?.error || "Failed to parse files");
      console.error("Error parsing files:", err);
    } finally {
      setIsProcessing(false);
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
              </div>
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
                {isProcessing() ? "Starting..." : "Full Process"}
              </Button>
              <Button
                onClick={handleParseOnly}
                disabled={isProcessing() || selectedFileTypes().length === 0}
                variant="secondary"
                class={styles.parseButton}
              >
                {isProcessing() ? "Parsing..." : "Parse Only"}
              </Button>
            </div>
          </Card>
        </div>
      </div>
    </div>
  );
};

export default Admin;
