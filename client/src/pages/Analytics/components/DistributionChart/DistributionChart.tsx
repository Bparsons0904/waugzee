import { Select } from "@components/common/forms/Select/Select";
import { formatDuration } from "@utils/dates";
import { Pie } from "solid-chartjs";
import { type Component, createMemo, createSignal, Show } from "solid-js";
import type { DateRange, DistributionDataItem, DistributionType } from "src/types/Analytics";
import type { PlayHistory } from "src/types/Release";
import type { UserRelease } from "src/types/User";
import { calculateDistribution } from "../../utils/chartUtils";
import styles from "./DistributionChart.module.scss";

interface DistributionChartProps {
  playHistory: PlayHistory[];
  releases: UserRelease[];
  dateRange: DateRange;
}

export const DistributionChart: Component<DistributionChartProps> = (props) => {
  const [distributionType, setDistributionType] = createSignal<DistributionType>("artist");
  const [topCount, setTopCount] = createSignal(10);

  const distributionTypeOptions = [
    { value: "artist", label: "By Artist" },
    { value: "genre", label: "By Genre" },
    { value: "release", label: "By Album" },
  ];

  const topCountOptions = [
    { value: "5", label: "Top 5" },
    { value: "10", label: "Top 10" },
    { value: "15", label: "Top 15" },
    { value: "20", label: "Top 20" },
    { value: "50", label: "Top 50" },
  ];

  const data = createMemo((): DistributionDataItem[] => {
    return calculateDistribution(
      props.playHistory,
      props.releases,
      distributionType(),
      topCount(),
      props.dateRange,
    );
  });

  const byCountChartData = createMemo(() => {
    const items = data();
    return {
      labels: items.map((item) => item.label),
      datasets: [
        {
          label: "Play Count",
          data: items.map((item) => item.count),
          backgroundColor: [
            "rgba(99, 102, 241, 0.8)",
            "rgba(34, 197, 94, 0.8)",
            "rgba(251, 191, 36, 0.8)",
            "rgba(239, 68, 68, 0.8)",
            "rgba(168, 85, 247, 0.8)",
            "rgba(236, 72, 153, 0.8)",
            "rgba(59, 130, 246, 0.8)",
            "rgba(16, 185, 129, 0.8)",
            "rgba(245, 158, 11, 0.8)",
            "rgba(220, 38, 38, 0.8)",
          ],
          borderWidth: 1,
        },
      ],
    };
  });

  const byDurationChartData = createMemo(() => {
    const items = data();
    return {
      labels: items.map((item) => item.label),
      datasets: [
        {
          label: "Listening Time",
          data: items.map((item) => item.duration),
          backgroundColor: [
            "rgba(99, 102, 241, 0.8)",
            "rgba(34, 197, 94, 0.8)",
            "rgba(251, 191, 36, 0.8)",
            "rgba(239, 68, 68, 0.8)",
            "rgba(168, 85, 247, 0.8)",
            "rgba(236, 72, 153, 0.8)",
            "rgba(59, 130, 246, 0.8)",
            "rgba(16, 185, 129, 0.8)",
            "rgba(245, 158, 11, 0.8)",
            "rgba(220, 38, 38, 0.8)",
          ],
          borderWidth: 1,
        },
      ],
    };
  });

  const countChartOptions = {
    responsive: true,
    maintainAspectRatio: false,
    plugins: {
      legend: {
        position: "bottom" as const,
        labels: {
          boxWidth: 12,
          padding: 10,
          font: {
            size: 11,
          },
        },
      },
      title: {
        display: true,
        text: "By Play Count",
        font: {
          size: 14,
          weight: "bold" as const,
        },
      },
      tooltip: {
        callbacks: {
          label: (context: { parsed: number; label?: string }) => {
            const count = context.parsed;
            const label = context.label || "";
            const total = data().reduce((sum, item) => sum + item.count, 0);
            const percentage = ((count / total) * 100).toFixed(1);
            return `${label}: ${count} plays (${percentage}%)`;
          },
        },
      },
    },
  };

  const durationChartOptions = {
    responsive: true,
    maintainAspectRatio: false,
    plugins: {
      legend: {
        position: "bottom" as const,
        labels: {
          boxWidth: 12,
          padding: 10,
          font: {
            size: 11,
          },
        },
      },
      title: {
        display: true,
        text: "By Listening Time",
        font: {
          size: 14,
          weight: "bold" as const,
        },
      },
      tooltip: {
        callbacks: {
          label: (context: { parsed: number; label?: string }) => {
            const minutes = context.parsed;
            const label = context.label || "";
            const total = data().reduce((sum, item) => sum + item.duration, 0);
            const percentage = ((minutes / total) * 100).toFixed(1);
            return `${label}: ${formatDuration(minutes)} (${percentage}%)`;
          },
        },
      },
    },
  };

  return (
    <div class={styles.chartContainer}>
      <div class={styles.controlsRow}>
        <Select
          label="Distribution Type"
          options={distributionTypeOptions}
          value={distributionType()}
          onChange={(value) => setDistributionType(value as DistributionType)}
        />
        <Select
          label="Show"
          options={topCountOptions}
          value={topCount().toString()}
          onChange={(value) => setTopCount(Number.parseInt(value, 10))}
        />
      </div>

      <Show
        when={data().length > 0}
        fallback={<div class={styles.noData}>No data available for distribution analysis</div>}
      >
        <div class={styles.chartsGrid}>
          <div class={styles.chartWrapper}>
            <Pie data={byCountChartData()} options={countChartOptions} />
          </div>
          <div class={styles.chartWrapper}>
            <Pie data={byDurationChartData()} options={durationChartOptions} />
          </div>
        </div>
      </Show>
    </div>
  );
};
