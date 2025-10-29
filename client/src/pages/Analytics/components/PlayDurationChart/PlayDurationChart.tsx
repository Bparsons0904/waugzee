import { formatDateForDisplay, formatDuration } from "@utils/dates";
import { Line } from "solid-chartjs";
import { type Component, createMemo, Show } from "solid-js";
import type {
  ChartFilter,
  DateRange,
  GroupFrequency,
  PlayDurationDataPoint,
} from "src/types/Analytics";
import type { PlayHistory } from "src/types/Release";
import type { UserRelease } from "src/types/User";
import { calculatePlayDuration, getFilterLabel } from "../../utils/chartUtils";
import styles from "./PlayDurationChart.module.scss";

interface PlayDurationChartProps {
  playHistory: PlayHistory[];
  releases: UserRelease[];
  dateRange: DateRange;
  frequency: GroupFrequency;
  filter: ChartFilter | null;
}

export const PlayDurationChart: Component<PlayDurationChartProps> = (props) => {
  const data = createMemo((): PlayDurationDataPoint[] => {
    return calculatePlayDuration(
      props.playHistory,
      props.releases,
      props.dateRange,
      props.frequency,
      props.filter,
    );
  });

  const chartTitle = createMemo(() => {
    const filterText = getFilterLabel(props.filter);
    return filterText ? `Listening Time ${filterText} Over Time` : "Listening Time Over Time";
  });

  const maxMinutes = createMemo(() => {
    const dataPoints = data();
    return Math.max(...dataPoints.map((d) => d.minutes), 0);
  });

  const chartData = createMemo(() => {
    const dataPoints = data();
    return {
      labels: dataPoints.map((d) => formatDateForDisplay(d.date, props.frequency)),
      datasets: [
        {
          label: "Listening Time",
          data: dataPoints.map((d) => d.minutes),
          borderColor: "rgb(34, 197, 94)",
          backgroundColor: "rgba(34, 197, 94, 0.1)",
          tension: 0.1,
          fill: true,
        },
      ],
    };
  });

  const chartOptions = createMemo(() => ({
    responsive: true,
    maintainAspectRatio: false,
    plugins: {
      legend: {
        position: "top" as const,
      },
      title: {
        display: true,
        text: chartTitle(),
        font: {
          size: 16,
          weight: "bold" as const,
        },
      },
      tooltip: {
        callbacks: {
          label: (context: { parsed: { y: number } }) => {
            const minutes = context.parsed.y;
            return `Listening time: ${formatDuration(minutes)}`;
          },
        },
      },
    },
    scales: {
      y: {
        beginAtZero: true,
        ticks: {
          callback: (value: number | string) => {
            const numValue = typeof value === "string" ? Number.parseFloat(value) : value;
            if (maxMinutes() >= 60) {
              return formatDuration(numValue);
            }
            return `${numValue}m`;
          },
        },
        title: {
          display: true,
          text: maxMinutes() >= 60 ? "Listening Time (hours)" : "Listening Time (minutes)",
        },
      },
      x: {
        title: {
          display: true,
          text: "Date",
        },
      },
    },
  }));

  return (
    <div class={styles.chartContainer}>
      <Show
        when={data().length > 0}
        fallback={
          <div class={styles.noData}>No play history available for the selected time period</div>
        }
      >
        <div class={styles.chartWrapper}>
          <Line data={chartData()} options={chartOptions()} />
        </div>
      </Show>
    </div>
  );
};
