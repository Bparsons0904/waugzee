import { Select } from "@components/common/forms/Select/Select";
import { Button } from "@components/common/ui/Button/Button";
import { type Component, createMemo, createSignal, For, Show } from "solid-js";
import type { NeglectedRecord } from "src/types/Analytics";
import type { CleaningHistory, PlayHistory } from "src/types/Release";
import type { UserRelease } from "src/types/User";
import {
  findNeglectedRecords,
  formatDaysSinceActivity,
  type NeglectMode,
} from "../../utils/neglectedUtils";
import styles from "./NeglectedRecords.module.scss";

interface NeglectedRecordsProps {
  releases: UserRelease[];
  playHistory: PlayHistory[];
  cleaningHistory: CleaningHistory[];
  onPlayRecord?: (userReleaseId: string) => void;
  onCleanRecord?: (userReleaseId: string) => void;
}

export const NeglectedRecords: Component<NeglectedRecordsProps> = (props) => {
  const [mode, setMode] = createSignal<NeglectMode>("play");
  const [daysThreshold, setDaysThreshold] = createSignal(90);
  const [displayLimit, setDisplayLimit] = createSignal(10);

  const modeOptions = [
    { value: "play", label: "Not Played" },
    { value: "cleaning", label: "Not Cleaned" },
  ];

  const thresholdOptions = [
    { value: "30", label: "30 Days" },
    { value: "60", label: "60 Days" },
    { value: "90", label: "90 Days" },
    { value: "180", label: "180 Days" },
    { value: "365", label: "1 Year" },
  ];

  const limitOptions = [
    { value: "10", label: "Show 10" },
    { value: "20", label: "Show 20" },
    { value: "50", label: "Show 50" },
    { value: "100", label: "Show 100" },
  ];

  const neglectedRecords = createMemo((): NeglectedRecord[] => {
    const all = findNeglectedRecords(
      props.releases,
      props.playHistory,
      props.cleaningHistory,
      daysThreshold(),
      mode(),
    );
    return all.slice(0, displayLimit());
  });

  const totalNeglected = createMemo(() => {
    return findNeglectedRecords(
      props.releases,
      props.playHistory,
      props.cleaningHistory,
      daysThreshold(),
      mode(),
    ).length;
  });

  const modeConfig = createMemo(() => {
    if (mode() === "play") {
      return {
        title: "Neglected Records - Not Played",
        subtitle: "Records that haven't been played recently - time to give them some love!",
        thresholdLabel: "Not Played In",
        emptyMessage:
          "Great! All your records have been played recently. Keep spinning those vinyls!",
        actionButton: "Log a Play",
        onAction: props.onPlayRecord,
      };
    }
    return {
      title: "Neglected Records - Not Cleaned",
      subtitle: "Records that need cleaning - time to show them some care!",
      thresholdLabel: "Not Cleaned In",
      emptyMessage: "Great! All your records have been cleaned recently. Keep them in shape!",
      actionButton: "Log Cleaning",
      onAction: props.onCleanRecord,
    };
  });

  return (
    <div class={styles.container}>
      <div class={styles.header}>
        <h2 class={styles.sectionTitle}>{modeConfig().title}</h2>
        <p class={styles.subtitle}>{modeConfig().subtitle}</p>
      </div>

      <div class={styles.controlsRow}>
        <Select
          label="Show"
          options={modeOptions}
          value={mode()}
          onChange={(value) => setMode(value as NeglectMode)}
        />
        <Select
          label={modeConfig().thresholdLabel}
          options={thresholdOptions}
          value={daysThreshold().toString()}
          onChange={(value) => setDaysThreshold(Number.parseInt(value, 10))}
        />
        <Select
          label="Display Limit"
          options={limitOptions}
          value={displayLimit().toString()}
          onChange={(value) => setDisplayLimit(Number.parseInt(value, 10))}
        />
        <div class={styles.countBadge}>
          <span class={styles.countValue}>{totalNeglected()}</span>
          <span class={styles.countLabel}>neglected</span>
        </div>
      </div>

      <Show
        when={neglectedRecords().length > 0}
        fallback={<div class={styles.noData}>{modeConfig().emptyMessage}</div>}
      >
        <div class={styles.recordsGrid}>
          <For each={neglectedRecords()}>
            {(record) => (
              <div class={styles.recordCard}>
                <Show
                  when={record.coverImage}
                  fallback={
                    <div class={styles.placeholderImage}>
                      <span>No Image</span>
                    </div>
                  }
                >
                  <img src={record.coverImage} alt={record.title} class={styles.coverImage} />
                </Show>

                <div class={styles.recordInfo}>
                  <h3 class={styles.recordTitle}>{record.title}</h3>
                  <p class={styles.artistNames}>{record.artistNames.join(", ")}</p>

                  <div class={styles.metaInfo}>
                    <span class={styles.daysSince}>
                      {formatDaysSinceActivity(record.daysSinceLastActivity)}
                    </span>

                    <Show when={mode() === "play"}>
                      <Show when={record.totalPlays > 0}>
                        <span class={styles.activityCount}>
                          {record.totalPlays} play{record.totalPlays !== 1 ? "s" : ""} total
                        </span>
                      </Show>
                      <Show when={record.totalPlays === 0}>
                        <span class={styles.neverActivity}>Never played</span>
                      </Show>
                    </Show>

                    <Show when={mode() === "cleaning"}>
                      <Show when={record.totalCleans && record.totalCleans > 0}>
                        <span class={styles.activityCount}>
                          {record.totalCleans} clean{record.totalCleans !== 1 ? "s" : ""} total
                        </span>
                      </Show>
                      <Show when={!record.totalCleans || record.totalCleans === 0}>
                        <span class={styles.neverActivity}>Never cleaned</span>
                      </Show>
                    </Show>
                  </div>

                  <Show when={modeConfig().onAction}>
                    <Button
                      variant="primary"
                      size="sm"
                      onClick={() => modeConfig().onAction?.(record.userReleaseId)}
                      class={styles.actionButton}
                    >
                      {modeConfig().actionButton}
                    </Button>
                  </Show>
                </div>
              </div>
            )}
          </For>
        </div>

        <Show when={totalNeglected() > displayLimit()}>
          <div class={styles.showingInfo}>
            Showing {displayLimit()} of {totalNeglected()} neglected records
          </div>
        </Show>
      </Show>
    </div>
  );
};
