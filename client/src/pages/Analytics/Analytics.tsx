import { useUserData } from "@context/UserDataContext";
import { calculateDateRange } from "@utils/dates";
import type { Component } from "solid-js";
import { createEffect, onMount, Show } from "solid-js";
import { createStore } from "solid-js/store";
import type { ChartFilter, DateRange, GroupFrequency, TimeFrame } from "src/types/Analytics";
import styles from "./Analytics.module.scss";
import { registerChartComponents } from "./chartConfig";
import { AnalyticsControls } from "./components/AnalyticsControls/AnalyticsControls";
import { DistributionChart } from "./components/DistributionChart/DistributionChart";
import { NeglectedRecords } from "./components/NeglectedRecords/NeglectedRecords";
import { PlayDurationChart } from "./components/PlayDurationChart/PlayDurationChart";
import { PlayFrequencyChart } from "./components/PlayFrequencyChart/PlayFrequencyChart";

interface AnalyticsState {
  timeFrame: TimeFrame;
  groupFrequency: GroupFrequency;
  filter: ChartFilter | null;
  customStartDate: string;
  customEndDate: string;
  dateRange: DateRange;
}

const Analytics: Component = () => {
  const userData = useUserData();

  const [state, setState] = createStore<AnalyticsState>({
    timeFrame: "30d",
    groupFrequency: "daily",
    filter: null,
    customStartDate: "",
    customEndDate: "",
    dateRange: calculateDateRange("30d"),
  });

  onMount(() => {
    registerChartComponents();
  });

  createEffect(() => {
    const start = state.customStartDate ? new Date(state.customStartDate) : undefined;
    const end = state.customEndDate ? new Date(state.customEndDate) : undefined;
    const range = calculateDateRange(state.timeFrame, start, end);
    setState("dateRange", range);
  });

  return (
    <div class={styles.analyticsPage}>
      <div class={styles.pageHeader}>
        <h1 class={styles.pageTitle}>Listening Analytics</h1>
        <p class={styles.pageDescription}>
          Explore your listening habits, track your collection usage, and discover neglected records
        </p>
      </div>

      <Show
        when={!userData.isLoading()}
        fallback={<div class={styles.loading}>Loading analytics data...</div>}
      >
        <Show
          when={userData.playHistory().length > 0}
          fallback={
            <div class={styles.emptyState}>
              <h2>No Data Available</h2>
              <p>Start logging plays to see your listening analytics!</p>
            </div>
          }
        >
          <AnalyticsControls
            timeFrame={state.timeFrame}
            groupFrequency={state.groupFrequency}
            filter={state.filter}
            customStartDate={state.customStartDate}
            customEndDate={state.customEndDate}
            releases={userData.releases()}
            onTimeFrameChange={(value) => setState("timeFrame", value)}
            onGroupFrequencyChange={(value) => setState("groupFrequency", value)}
            onFilterChange={(value) => setState("filter", value)}
            onCustomStartDateChange={(value) => setState("customStartDate", value)}
            onCustomEndDateChange={(value) => setState("customEndDate", value)}
          />

          <PlayFrequencyChart
            playHistory={userData.playHistory()}
            releases={userData.releases()}
            dateRange={state.dateRange}
            frequency={state.groupFrequency}
            filter={state.filter}
          />

          <PlayDurationChart
            playHistory={userData.playHistory()}
            releases={userData.releases()}
            dateRange={state.dateRange}
            frequency={state.groupFrequency}
            filter={state.filter}
          />

          <DistributionChart
            playHistory={userData.playHistory()}
            releases={userData.releases()}
            dateRange={state.dateRange}
          />

          <NeglectedRecords
            releases={userData.releases()}
            playHistory={userData.playHistory()}
            cleaningHistory={userData.cleaningHistory()}
          />
        </Show>
      </Show>
    </div>
  );
};

export default Analytics;
