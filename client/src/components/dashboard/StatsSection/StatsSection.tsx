import { Card } from "@components/common/ui/Card/Card";
import { ROUTES } from "@constants/api.constants";
import { useUserData } from "@context/UserDataContext";
import { useNavigate } from "@solidjs/router";
import { type Component, createMemo, For } from "solid-js";
import styles from "./StatsSection.module.scss";

export interface StatItem {
  icon: string;
  value: string | number;
  label: string;
  isLoading?: boolean;
  onClick?: () => void;
}

export const StatsSection: Component = () => {
  const { releases, playHistory, isLoading } = useUserData();
  const navigate = useNavigate();

  const stats = createMemo(() => {
    const allReleases = releases();
    const allPlays = playHistory();

    const totalRecords = allReleases.length;
    const totalPlays = allPlays.length;

    const listeningHours = Math.round(
      allPlays.reduce((total, play) => {
        const duration = play.userRelease?.release?.totalDuration || 0;
        return total + duration;
      }, 0) / 3600,
    );

    const genreCounts = new Map<string, number>();
    for (const play of allPlays) {
      const genres = play.userRelease?.release?.genres || [];
      for (const genre of genres) {
        genreCounts.set(genre.name, (genreCounts.get(genre.name) || 0) + 1);
      }
    }

    let topGenre = "No plays yet";
    let maxCount = 0;
    for (const [genre, count] of genreCounts.entries()) {
      if (count > maxCount) {
        maxCount = count;
        topGenre = genre;
      }
    }

    return {
      totalRecords,
      totalPlays,
      listeningHours,
      favoriteGenre: topGenre,
    };
  });

  const statsItems = createMemo((): StatItem[] => [
    {
      icon: "🎶",
      value: isLoading() ? "--" : stats().totalRecords.toLocaleString(),
      label: "Records",
      isLoading: isLoading(),
      onClick: () => navigate(ROUTES.COLLECTION),
    },
    {
      icon: "▶",
      value: isLoading() ? "--" : stats().totalPlays.toLocaleString(),
      label: "Plays",
      isLoading: isLoading(),
      onClick: () => navigate(ROUTES.PLAY_HISTORY),
    },
    {
      icon: "⏱",
      value: isLoading() ? "--h" : `${stats().listeningHours}h`,
      label: "Hours",
      isLoading: isLoading(),
      onClick: () => navigate(ROUTES.ANALYTICS),
    },
    {
      icon: "☆",
      value: isLoading() ? "--" : stats().favoriteGenre,
      label: "Top Genre",
      isLoading: isLoading(),
      onClick: () => navigate(ROUTES.ANALYTICS),
    },
  ]);

  return (
    <section class={styles.statsSection}>
      <div class={styles.statsGrid}>
        <For each={statsItems()}>
          {(stat) => (
            <Card class={styles.statCard} onClick={stat.onClick}>
              <div class={styles.statIcon}>{stat.icon}</div>
              <div class={styles.statContent}>
                <div class={styles.statNumber}>{stat.isLoading ? "--" : stat.value}</div>
                <p class={styles.statLabel}>{stat.label}</p>
              </div>
            </Card>
          )}
        </For>
      </div>
    </section>
  );
};
