import {
  formatBytes,
  formatStatusLabel,
  formatTimestamp,
  getFileStatusColor,
  truncateChecksum,
} from "@utils/admin.utils";
import clsx from "clsx";
import { Show } from "solid-js";
import type { FileDownloadInfo } from "../../types/Admin";
import styles from "./FileCard.module.scss";

interface FileCardProps {
  title: string;
  file: FileDownloadInfo;
  checksum?: string;
}

export function FileCard(props: FileCardProps) {
  return (
    <div class={styles.fileCard}>
      <h4 class={styles.fileTitle}>{props.title}</h4>
      <div class={styles.fileDetails}>
        <div class={styles.detailRow}>
          <span class={styles.detailLabel}>Status:</span>
          <span class={clsx(styles.statusBadge, styles[getFileStatusColor(props.file.status)])}>
            {formatStatusLabel(props.file.status)}
          </span>
        </div>
        <div class={styles.detailRow}>
          <span class={styles.detailLabel}>Size:</span>
          <span>{formatBytes(props.file.size)}</span>
        </div>
        <Show when={props.checksum}>
          {(checksum) => (
            <div class={styles.detailRow}>
              <span class={styles.detailLabel}>Checksum:</span>
              <span class={styles.checksum}>{truncateChecksum(checksum())}</span>
            </div>
          )}
        </Show>
        <Show when={props.file.downloaded_at}>
          <div class={styles.detailRow}>
            <span class={styles.detailLabel}>Downloaded:</span>
            <span>{formatTimestamp(props.file.downloaded_at)}</span>
          </div>
        </Show>
        <Show when={props.file.error_message}>
          <div class={clsx(styles.detailRow, styles.error)}>
            <span class={styles.detailLabel}>Error:</span>
            <span>{props.file.error_message}</span>
          </div>
        </Show>
      </div>
    </div>
  );
}
