import { Select } from "@components/common/forms/Select/Select";
import { Button } from "@components/common/ui/Button/Button";
import { type Component, createMemo, createSignal, For, Show } from "solid-js";
import type { NeglectedRecord } from "src/types/Analytics";
import type { PlayHistory } from "src/types/Release";
import type { UserRelease } from "src/types/User";
import { findNeglectedRecords, formatDaysSincePlay } from "../../utils/neglectedUtils";
import styles from "./NeglectedRecords.module.scss";

interface NeglectedRecordsProps {
  releases: UserRelease[];
  playHistory: PlayHistory[];
  onPlayRecord?: (userReleaseId: string) => void;
}

export const NeglectedRecords: Component<NeglectedRecordsProps> = (props) => {
  const [daysThreshold, setDaysThreshold] = createSignal(90);
  const [displayLimit, setDisplayLimit] = createSignal(10);

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
    const all = findNeglectedRecords(props.releases, props.playHistory, daysThreshold());
    return all.slice(0, displayLimit());
  });

  const totalNeglected = createMemo(() => {
    return findNeglectedRecords(props.releases, props.playHistory, daysThreshold()).length;
  });

  return (
    <div class={styles.container}>
      <div class={styles.header}>
        <h2 class={styles.sectionTitle}>Neglected Records</h2>
        <p class={styles.subtitle}>
          Records that haven't been played recently - time to give them some love!
        </p>
      </div>

      <div class={styles.controlsRow}>
        <Select
          label="Not Played In"
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
        fallback={
          <div class={styles.noData}>
            Great! All your records have been played recently. Keep spinning those vinyls!
          </div>
        }
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
                      {formatDaysSincePlay(record.daysSinceLastPlay)}
                    </span>
                    <Show when={record.totalPlays > 0}>
                      <span class={styles.playCount}>
                        {record.totalPlays} play{record.totalPlays !== 1 ? "s" : ""} total
                      </span>
                    </Show>
                    <Show when={record.totalPlays === 0}>
                      <span class={styles.neverPlayed}>Never played</span>
                    </Show>
                  </div>

                  <Show when={props.onPlayRecord}>
                    <Button
                      variant="primary"
                      size="sm"
                      onClick={() => props.onPlayRecord?.(record.userReleaseId)}
                      class={styles.playButton}
                    >
                      Log a Play
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
