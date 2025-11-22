// TODO: REMOVE_AFTER_MIGRATION - This entire component is for one-time Kleio data import
import { Button } from "@components/common/ui/Button/Button";
import { LoadingSpinner } from "@components/icons/LoadingSpinner";
import { api } from "@services/api";
import { createMutation } from "@tanstack/solid-query";
import { createSignal, Show } from "solid-js";
import styles from "./KleioImportSection.module.scss";

interface ImportSummary {
  plays_imported: number;
  cleanings_imported: number;
  skipped_plays: number;
  skipped_cleanings: number;
  errors: string[];
}

interface ImportResponse {
  message: string;
  summary: ImportSummary;
}

export function KleioImportSection() {
  const [selectedFile, setSelectedFile] = createSignal<File | null>(null);

  const importMutation = createMutation(() => ({
    mutationFn: async (file: File) => {
      const formData = new FormData();
      formData.append("file", file);

      return api.post<ImportResponse>("/admin/import-kleio-data", formData, {
        headers: {
          "Content-Type": "multipart/form-data",
        },
        timeout: 60000,
      });
    },
  }));

  const handleFileChange = (e: Event) => {
    const input = e.target as HTMLInputElement;
    const file = input.files?.[0] || null;
    setSelectedFile(file);
  };

  const handleImport = () => {
    const file = selectedFile();
    if (file) {
      importMutation.mutate(file);
    }
  };

  return (
    <section class={styles.section}>
      <div class={styles.header}>
        <div>
          <h2 class={styles.sectionTitle}>Kleio Data Import</h2>
          <p class={styles.sectionDescription}>
            One-time import of play and cleaning history from Kleio export
          </p>
        </div>
      </div>

      <div class={styles.content}>
        <div class={styles.uploadArea}>
          <input
            type="file"
            accept=".json"
            onChange={handleFileChange}
            class={styles.fileInput}
            id="kleio-file-input"
          />
          <label for="kleio-file-input" class={styles.fileLabel}>
            {selectedFile()?.name || "Choose JSON file..."}
          </label>

          <Button
            onClick={handleImport}
            disabled={!selectedFile() || importMutation.isPending}
          >
            <Show when={importMutation.isPending} fallback="Import Data">
              <LoadingSpinner />
              Importing...
            </Show>
          </Button>
        </div>

        <Show when={importMutation.isSuccess}>
          <div class={styles.successMessage}>
            <strong>Import Complete!</strong>
            <ul>
              <li>Plays imported: {importMutation.data?.summary.plays_imported}</li>
              <li>Cleanings imported: {importMutation.data?.summary.cleanings_imported}</li>
              <Show when={importMutation.data?.summary.skipped_plays}>
                <li>Plays skipped: {importMutation.data?.summary.skipped_plays}</li>
              </Show>
              <Show when={importMutation.data?.summary.skipped_cleanings}>
                <li>Cleanings skipped: {importMutation.data?.summary.skipped_cleanings}</li>
              </Show>
            </ul>
          </div>
        </Show>

        <Show when={importMutation.isError}>
          <div class={styles.errorMessage}>
            Error: {(importMutation.error as Error)?.message || "Import failed"}
          </div>
        </Show>
      </div>
    </section>
  );
}
