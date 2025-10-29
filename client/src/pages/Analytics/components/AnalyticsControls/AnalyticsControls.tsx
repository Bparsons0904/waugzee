import { DateInput } from "@components/common/forms/DateInput/DateInput";
import { SearchableSelect } from "@components/common/forms/SearchableSelect/SearchableSelect";
import { Select } from "@components/common/forms/Select/Select";
import type { Component } from "solid-js";
import { createMemo, Show } from "solid-js";
import type { ChartFilter, DropdownOption, GroupFrequency, TimeFrame } from "src/types/Analytics";
import type { UserRelease } from "src/types/User";
import styles from "./AnalyticsControls.module.scss";

interface AnalyticsControlsProps {
  timeFrame: TimeFrame;
  groupFrequency: GroupFrequency;
  filter: ChartFilter | null;
  customStartDate?: string;
  customEndDate?: string;
  releases: UserRelease[];
  onTimeFrameChange: (timeFrame: TimeFrame) => void;
  onGroupFrequencyChange: (frequency: GroupFrequency) => void;
  onFilterChange: (filter: ChartFilter | null) => void;
  onCustomStartDateChange: (date: string) => void;
  onCustomEndDateChange: (date: string) => void;
}

export const AnalyticsControls: Component<AnalyticsControlsProps> = (props) => {
  const timeFrameOptions = [
    { value: "7d", label: "Last 7 Days" },
    { value: "30d", label: "Last 30 Days" },
    { value: "90d", label: "Last 90 Days" },
    { value: "1y", label: "Last Year" },
    { value: "all", label: "All Time" },
    { value: "custom", label: "Custom Range" },
  ];

  const frequencyOptions = [
    { value: "daily", label: "Daily" },
    { value: "weekly", label: "Weekly" },
    { value: "monthly", label: "Monthly" },
  ];

  const filterOptions = createMemo((): DropdownOption[] => {
    const options: DropdownOption[] = [{ value: "", label: "All Records" }];

    const artists = new Set<string>();
    const genres = new Set<string>();

    props.releases.forEach((userRelease) => {
      userRelease.release.artists?.forEach((artist) => {
        artists.add(artist.name);
      });

      userRelease.release.genres?.forEach((genre) => {
        genres.add(genre.name);
      });
    });

    if (props.releases.length > 0) {
      options.push({ value: "records-header", label: "--- Records ---", disabled: true });
      props.releases.forEach((userRelease) => {
        const title = userRelease.release.title || "Unknown";
        const artistNames =
          userRelease.release.artists?.map((a) => a.name).join(", ") || "Unknown Artist";
        options.push({
          value: `record:${userRelease.id}`,
          label: title,
          metadata: artistNames,
        });
      });
    }

    if (artists.size > 0) {
      options.push({ value: "artists-header", label: "--- Artists ---", disabled: true });
      Array.from(artists)
        .sort()
        .forEach((artist) => {
          options.push({
            value: `artist:${artist}`,
            label: artist,
          });
        });
    }

    if (genres.size > 0) {
      options.push({ value: "genres-header", label: "--- Genres ---", disabled: true });
      Array.from(genres)
        .sort()
        .forEach((genre) => {
          options.push({
            value: `genre:${genre}`,
            label: genre,
          });
        });
    }

    return options;
  });

  const handleFilterChange = (value: string) => {
    if (!value) {
      props.onFilterChange(null);
      return;
    }

    const [type, ...valueParts] = value.split(":");
    const filterValue = valueParts.join(":");

    const option = filterOptions().find((opt) => opt.value === value);
    const label = option?.label || filterValue;

    if (type === "record" || type === "artist" || type === "genre") {
      props.onFilterChange({
        type: type as "record" | "artist" | "genre",
        value: filterValue,
        label,
      });
    }
  };

  const currentFilterValue = () => {
    if (!props.filter) return "";
    return `${props.filter.type}:${props.filter.value}`;
  };

  return (
    <div class={styles.controlsContainer}>
      <div class={styles.controlsGrid}>
        <div class={styles.controlGroup}>
          <Select
            label="Time Period"
            options={timeFrameOptions}
            value={props.timeFrame}
            onChange={(value) => props.onTimeFrameChange(value as TimeFrame)}
          />
        </div>

        <Show when={props.timeFrame === "custom"}>
          <div class={styles.controlGroup}>
            <DateInput
              label="Start Date"
              value={props.customStartDate || ""}
              onChange={props.onCustomStartDateChange}
            />
          </div>

          <div class={styles.controlGroup}>
            <DateInput
              label="End Date"
              value={props.customEndDate || ""}
              onChange={props.onCustomEndDateChange}
            />
          </div>
        </Show>

        <div class={styles.controlGroup}>
          <Select
            label="Group By"
            options={frequencyOptions}
            value={props.groupFrequency}
            onChange={(value) => props.onGroupFrequencyChange(value as GroupFrequency)}
          />
        </div>

        <div class={styles.controlGroup}>
          <SearchableSelect
            label="Filter"
            placeholder="All Records"
            searchPlaceholder="Search records, artists, or genres..."
            options={filterOptions()}
            value={currentFilterValue()}
            onChange={handleFilterChange}
            emptyMessage="No matching records found"
          />
        </div>
      </div>
    </div>
  );
};
