import { formatDateForDisplay } from "@utils/dates";
import { Line } from "solid-chartjs";
import { type Component, createMemo, Show } from "solid-js";
import type {
  ChartFilter,
  DateRange,
  GroupFrequency,
  PlayFrequencyDataPoint,
} from "src/types/Analytics";
import type { PlayHistory } from "src/types/Release";
import type { UserRelease } from "src/types/User";
import { calculatePlayFrequency, getFilterLabel } from "../../utils/chartUtils";
import styles from "./PlayFrequencyChart.module.scss";

interface PlayFrequencyChartProps {
  playHistory: PlayHistory[];
  releases: UserRelease[];
  dateRange: DateRange;
  frequency: GroupFrequency;
  filter: ChartFilter | null;
}

export const PlayFrequencyChart: Component<PlayFrequencyChartProps> = (props) => {
  const data = createMemo((): PlayFrequencyDataPoint[] => {
    return calculatePlayFrequency(
      props.playHistory,
      props.releases,
      props.dateRange,
      props.frequency,
      props.filter,
    );
  });

  const chartTitle = createMemo(() => {
    const filterText = getFilterLabel(props.filter);
    return filterText ? `Records ${filterText} Played Over Time` : "Records Played Over Time";
  });

  const chartData = createMemo(() => {
    const dataPoints = data();
    return {
      labels: dataPoints.map((d) => formatDateForDisplay(d.date, props.frequency)),
      datasets: [
        {
          label: "Records Played",
          data: dataPoints.map((d) => d.count),
          borderColor: "rgb(99, 102, 241)",
          backgroundColor: "rgba(99, 102, 241, 0.1)",
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
        text: chartTitle(),
        font: {
          size: 16,
          weight: "bold" as const,
        },
      },
      tooltip: {
        callbacks: {
          label: (context: { parsed: { y: number } }) => {
            const count = context.parsed.y;
            return `${count} record${count !== 1 ? "s" : ""} played`;
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
          text: "Number of Records Played",
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
    <div class={styles.chartContainer}>
      <Show
        when={data().length > 0}
        fallback={
          <div class={styles.noData}>No play history available for the selected time period</div>
        }
      >
        <div class={styles.chartWrapper}>
          <Line data={chartData()} options={chartOptions} />
        </div>
      </Show>
    </div>
  );
};
