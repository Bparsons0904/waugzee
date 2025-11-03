import { formatDateForDisplay } from "@utils/dates";
import { Line } from "solid-chartjs";
import { type Component, createMemo, Show } from "solid-js";
import type { DateRange, GroupFrequency } from "src/types/Analytics";
import type { CleaningHistory } from "src/types/Release";
import { calculateCleaningFrequency, calculateCleaningStats } from "../../utils/cleaningUtils";
import styles from "./CleaningAnalytics.module.scss";

interface CleaningAnalyticsProps {
  cleaningHistory: CleaningHistory[];
  dateRange: DateRange;
  frequency: GroupFrequency;
}

export const CleaningAnalytics: Component<CleaningAnalyticsProps> = (props) => {
  const frequencyData = createMemo(() => {
    return calculateCleaningFrequency(props.cleaningHistory, props.dateRange, props.frequency);
  });

  const stats = createMemo(() => {
    return calculateCleaningStats(props.cleaningHistory, props.dateRange);
  });

  const chartData = createMemo(() => {
    const dataPoints = frequencyData();
    return {
      labels: dataPoints.map((d) => formatDateForDisplay(d.date, props.frequency)),
      datasets: [
        {
          label: "Regular Cleans",
          data: dataPoints.map((d) => d.regularCount),
          borderColor: "rgb(59, 130, 246)",
          backgroundColor: "rgba(59, 130, 246, 0.1)",
          tension: 0.1,
          fill: true,
        },
        {
          label: "Deep Cleans",
          data: dataPoints.map((d) => d.deepCleanCount),
          borderColor: "rgb(168, 85, 247)",
          backgroundColor: "rgba(168, 85, 247, 0.1)",
          tension: 0.1,
          fill: true,
        },
      ],
    };
  });

  const chartOptions = {
    responsive: true,
    maintainAspectRatio: false,
    plugins: {
      legend: {
        position: "top" as const,
      },
      title: {
        display: true,
        text: "Cleaning Frequency Over Time",
        font: {
          size: 16,
          weight: "bold" as const,
        },
      },
      tooltip: {
        callbacks: {
          label: (context: { dataset: { label?: string }; parsed: { y: number } }) => {
            const count = context.parsed.y;
            const label = context.dataset.label || "";
            return `${label}: ${count} clean${count !== 1 ? "s" : ""}`;
          },
        },
      },
    },
    scales: {
      y: {
        beginAtZero: true,
        ticks: {
          stepSize: 1,
          precision: 0,
        },
        title: {
          display: true,
          text: "Number of Cleans",
        },
      },
      x: {
        title: {
          display: true,
          text: "Date",
        },
      },
    },
  };

  return (
    <div class={styles.container}>
      <h2 class={styles.sectionTitle}>Cleaning Analytics</h2>

      <Show
        when={props.cleaningHistory.length > 0}
        fallback={
          <div class={styles.noData}>
            No cleaning history available. Start logging your record cleanings to see analytics
            here.
          </div>
        }
      >
        <div class={styles.statsGrid}>
          <div class={styles.statCard}>
            <div class={styles.statValue}>{stats().totalCleans}</div>
            <div class={styles.statLabel}>Total Cleans</div>
          </div>
          <div class={styles.statCard}>
            <div class={styles.statValue}>{stats().deepCleans}</div>
            <div class={styles.statLabel}>Deep Cleans</div>
          </div>
          <div class={styles.statCard}>
            <div class={styles.statValue}>{stats().regularCleans}</div>
            <div class={styles.statLabel}>Regular Cleans</div>
          </div>
          <div class={styles.statCard}>
            <div class={styles.statValue}>
              {stats().averageDaysBetweenCleans > 0 ? stats().averageDaysBetweenCleans : "N/A"}
            </div>
            <div class={styles.statLabel}>Avg Days Between Cleans</div>
          </div>
        </div>

        <div class={styles.chartContainer}>
          <div class={styles.chartWrapper}>
            <Line data={chartData()} options={chartOptions} />
          </div>
        </div>
      </Show>
    </div>
  );
};
