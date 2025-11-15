import { formatStatusLabel, getFileStatusColor } from "@utils/admin.utils";
import { formatBytes, formatTimestamp, truncateHash } from "@utils/formatters";
import clsx from "clsx";
import { Show } from "solid-js";
import type { FileDownloadInfo } from "../../types/Admin";
import styles from "./FileDownloadRow.module.scss";

interface FileDownloadRowProps {
  fileName: string;
  fileInfo: FileDownloadInfo | undefined;
  checksum: string | undefined;
}

export function FileDownloadRow(props: FileDownloadRowProps) {
  const statusColor = () => {
    if (!props.fileInfo) return "gray";
    return getFileStatusColor(props.fileInfo.status);
  };

  const isValidated = () => props.fileInfo?.validated ?? false;

  return (
    <div
      class={clsx(
        styles.fileRow,
        isValidated() && styles.validated,
        props.fileInfo?.error_message && styles.error,
      )}
    >
      <div class={styles.fileName}>{props.fileName}</div>

      <Show when={props.fileInfo} fallback={<div class={styles.fileStatus}>N/A</div>}>
        {(info) => (
          <>
            <div class={styles.fileStatus}>
              <span class={clsx(styles.statusBadge, styles[statusColor()])}>
                {formatStatusLabel(info().status)}
              </span>
            </div>

            <div class={styles.fileSize}>{formatBytes(info().size)}</div>

            <div class={styles.fileChecksum}>
              {props.checksum ? truncateHash(props.checksum) : "N/A"}
            </div>

            <div class={styles.fileTimestamp}>{formatTimestamp(info().downloaded_at)}</div>

            <div class={styles.fileValidation}>{isValidated() ? "✓ Validated" : "Pending"}</div>
          </>
        )}
      </Show>

      <Show when={props.fileInfo?.error_message}>
        <div class={styles.fileError}>{props.fileInfo?.error_message}</div>
      </Show>
    </div>
  );
}
