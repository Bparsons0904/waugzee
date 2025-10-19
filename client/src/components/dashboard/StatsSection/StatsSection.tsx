import { Card } from "@components/common/ui/Card/Card";
import { type Component, For } from "solid-js";
import styles from "./StatsSection.module.scss";

export interface StatItem {
  icon: string;
  value: string | number;
  label: string;
  isLoading?: boolean;
}

interface StatsSectionProps {
  stats: StatItem[];
}

export const StatsSection: Component<StatsSectionProps> = (props) => {
  return (
    <section class={styles.statsSection}>
      <div class={styles.statsGrid}>
        <For each={props.stats}>
          {(stat) => (
            <Card class={styles.statCard}>
              <div class={styles.statIcon}>{stat.icon}</div>
              <div class={styles.statContent}>
                <h3 class={styles.statNumber}>{stat.isLoading ? "--" : stat.value}</h3>
                <p class={styles.statLabel}>{stat.label}</p>
              </div>
            </Card>
          )}
        </For>
      </div>
    </section>
  );
};
