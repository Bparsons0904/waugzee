import clsx from "clsx";
import { type Component, Show } from "solid-js";
import styles from "./StreakBadge.module.scss";

interface StreakBadgeProps {
  currentStreak: number;
  longestStreak: number;
  class?: string;
}

export const StreakBadge: Component<StreakBadgeProps> = (props) => {
  return (
    <Show when={props.currentStreak > 0} fallback={null}>
      <div
        class={clsx(styles.streakBadge, props.class)}
        title={`Longest streak: ${props.longestStreak} days`}
      >
        <span class={styles.fireEmoji}>🔥</span>
        <span class={styles.streakCount}>{props.currentStreak}</span>
      </div>
    </Show>
  );
};
