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
      <div class={styles.stepName}>
        {props.status.completed ? "✓ " : ""}
        {getStepLabel(props.step)}
      </div>
      <div class={styles.stepDuration}>
        {props.status.duration ? formatDuration(props.status.duration) : ""}
      </div>
      <div class={styles.stepTime}>
        {props.status.completed_at ? formatTimestamp(props.status.completed_at) : ""}
      </div>
      <Show when={props.status.error_message}>
        <div class={styles.stepError}>{props.status.error_message}</div>
      </Show>
    </div>
  );
}
