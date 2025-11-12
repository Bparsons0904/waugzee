import styles from "./ProgressBar.module.scss";

interface ProgressBarProps {
  percentage: number;
  label?: string;
  subLabel?: string;
}

export const ProgressBar = (props: ProgressBarProps) => {
  const safePercentage = () => Math.min(100, Math.max(0, props.percentage));

  return (
    <div class={styles.progressContainer}>
      <div class={styles.labelRow}>
        {props.label && <span class={styles.label}>{props.label}</span>}
        {props.subLabel && <span class={styles.subLabel}>{props.subLabel}</span>}
        <span class={styles.percentage}>{safePercentage().toFixed(1)}%</span>
      </div>
      <div class={styles.progressBar}>
        <div class={styles.progressFill} style={{ width: `${safePercentage()}%` }} />
      </div>
    </div>
  );
};
