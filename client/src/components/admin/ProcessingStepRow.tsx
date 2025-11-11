import { formatDuration, formatTimestamp, getStepLabel } from "@utils/admin.utils";
import clsx from "clsx";
import { Show } from "solid-js";
import type { StepStatus } from "../../types/Admin";
import styles from "./ProcessingStepRow.module.scss";

interface ProcessingStepRowProps {
  step: string;
  status: StepStatus;
}

export function ProcessingStepRow(props: ProcessingStepRowProps) {
  return (
    <div
      class={clsx(
        styles.processingRow,
        props.status.completed && styles.completed,
        props.status.error_message && styles.error,
      )}
    >
      <div class={styles.stepName}>{getStepLabel(props.step)}</div>
      <div class={styles.stepStatus}>{props.status.completed ? "✓ Completed" : "Pending"}</div>
      <Show when={props.status.records_count !== undefined}>
        <div class={styles.stepRecords}>{props.status.records_count} records</div>
      </Show>
      <Show when={props.status.duration}>
        <div class={styles.stepDuration}>{formatDuration(props.status.duration)}</div>
      </Show>
      <Show when={props.status.completed_at}>
        <div class={styles.stepTime}>{formatTimestamp(props.status.completed_at)}</div>
      </Show>
      <Show when={props.status.error_message}>
        <div class={styles.stepError}>{props.status.error_message}</div>
      </Show>
    </div>
  );
}
