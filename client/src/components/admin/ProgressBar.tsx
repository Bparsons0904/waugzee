import styles from "./ProgressBar.module.scss";

interface ProgressBarProps {
  percentage: number;
  label?: string;
  variant?: "primary" | "success" | "error";
}

export const ProgressBar = (props: ProgressBarProps) => {
  const safePercentage = () => Math.min(100, Math.max(0, props.percentage));

  return (
    <div class={styles.progressContainer}>
      {props.label && <span class={styles.label}>{props.label}</span>}
      <div class={styles.progressBar}>
        <div
          class={`${styles.progressFill} ${styles[props.variant || "primary"]}`}
          style={{ width: `${safePercentage()}%` }}
        />
      </div>
      <span class={styles.percentage}>{safePercentage().toFixed(1)}%</span>
    </div>
  );
};
