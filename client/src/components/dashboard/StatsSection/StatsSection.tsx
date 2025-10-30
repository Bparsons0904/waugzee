import { Card } from "@components/common/ui/Card/Card";
import { type Component, createMemo, createSignal, For, onMount } from "solid-js";
import styles from "./StatsSection.module.scss";

export interface StatItem {
  icon: string;
  value: string | number;
  label: string;
  isLoading?: boolean;
}

interface DashboardStats {
  totalRecords: number;
  totalPlays: number;
  listeningHours: number;
  favoriteGenre: string;
}

export const StatsSection: Component = () => {
  const [stats, setStats] = createSignal<DashboardStats>({
    totalRecords: 0,
    totalPlays: 0,
    listeningHours: 0,
    favoriteGenre: "Loading...",
  });
  const [isLoading, setIsLoading] = createSignal(true);

  const statsItems = createMemo((): StatItem[] => [
    {
      icon: "💽",
      value: isLoading() ? "--" : stats().totalRecords.toLocaleString(),
      label: "Records",
      isLoading: isLoading(),
    },
    {
      icon: "▶",
      value: isLoading() ? "--" : stats().totalPlays.toLocaleString(),
      label: "Plays",
      isLoading: isLoading(),
    },
    {
      icon: "⏱",
      value: isLoading() ? "--h" : `${stats().listeningHours}h`,
      label: "Hours",
      isLoading: isLoading(),
    },
    {
      icon: "🎯",
      value: isLoading() ? "--" : stats().favoriteGenre,
      label: "Top Genre",
      isLoading: isLoading(),
    },
  ]);

  onMount(async () => {
    try {
      await new Promise((resolve) => setTimeout(resolve, 1000));

      setStats({
        totalRecords: 247,
        totalPlays: 1430,
        listeningHours: 89,
        favoriteGenre: "Jazz",
      });
    } catch (error) {
      console.error("Failed to load dashboard data:", error);
    } finally {
      setIsLoading(false);
    }
  });

  return (
    <section class={styles.statsSection}>
      <div class={styles.statsGrid}>
        <For each={statsItems()}>
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
