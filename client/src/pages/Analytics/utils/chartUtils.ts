import { generateDateKeys, getDateGroupKey, parseDateGroupKey } from "@utils/dates";
import type {
  ChartFilter,
  DateRange,
  DistributionDataItem,
  DistributionType,
  GroupFrequency,
  PlayDurationDataPoint,
  PlayFrequencyDataPoint,
} from "src/types/Analytics";
import type { PlayHistory } from "src/types/Release";
import type { UserRelease } from "src/types/User";

const ESTIMATED_VINYL_DURATION_MINUTES = 40;

export function calculatePlayFrequency(
  playHistory: PlayHistory[],
  releases: UserRelease[],
  dateRange: DateRange,
  frequency: GroupFrequency,
  filter?: ChartFilter | null,
): PlayFrequencyDataPoint[] {
  let filteredHistory = playHistory.filter((play) => {
    const playDate = new Date(play.playedAt);
    return playDate >= dateRange.start && playDate <= dateRange.end;
  });

  if (filter) {
    filteredHistory = applyFilter(filteredHistory, releases, filter);
  }

  const dateKeys = generateDateKeys(dateRange.start, dateRange.end, frequency);
  const countsByDate = new Map<string, number>();

  dateKeys.forEach((key) => {
    countsByDate.set(key, 0);
  });

  filteredHistory.forEach((play) => {
    const playDate = new Date(play.playedAt);
    const key = getDateGroupKey(playDate, frequency);
    countsByDate.set(key, (countsByDate.get(key) || 0) + 1);
  });

  return Array.from(countsByDate.entries())
    .map(([key, count]) => ({
      date: parseDateGroupKey(key, frequency),
      count,
    }))
    .sort((a, b) => a.date.getTime() - b.date.getTime());
}

export function calculatePlayDuration(
  playHistory: PlayHistory[],
  releases: UserRelease[],
  dateRange: DateRange,
  frequency: GroupFrequency,
  filter?: ChartFilter | null,
): PlayDurationDataPoint[] {
  let filteredHistory = playHistory.filter((play) => {
    const playDate = new Date(play.playedAt);
    return playDate >= dateRange.start && playDate <= dateRange.end;
  });

  if (filter) {
    filteredHistory = applyFilter(filteredHistory, releases, filter);
  }

  const dateKeys = generateDateKeys(dateRange.start, dateRange.end, frequency);
  const durationsByDate = new Map<string, number>();

  dateKeys.forEach((key) => {
    durationsByDate.set(key, 0);
  });

  filteredHistory.forEach((play) => {
    const playDate = new Date(play.playedAt);
    const key = getDateGroupKey(playDate, frequency);

    const duration = ESTIMATED_VINYL_DURATION_MINUTES;
    durationsByDate.set(key, (durationsByDate.get(key) || 0) + duration);
  });

  return Array.from(durationsByDate.entries())
    .map(([key, minutes]) => ({
      date: parseDateGroupKey(key, frequency),
      minutes,
    }))
    .sort((a, b) => a.date.getTime() - b.date.getTime());
}

export function calculateDistribution(
  playHistory: PlayHistory[],
  releases: UserRelease[],
  distributionType: DistributionType,
  topCount: number,
  dateRange: DateRange,
): DistributionDataItem[] {
  const filteredHistory = playHistory.filter((play) => {
    const playDate = new Date(play.playedAt);
    return playDate >= dateRange.start && playDate <= dateRange.end;
  });

  const distributionMap = new Map<
    string,
    {
      count: number;
      duration: number;
    }
  >();

  filteredHistory.forEach((play) => {
    const userRelease = releases.find((r) => r.id === play.userReleaseId);
    if (!userRelease) return;

    const keys = getDistributionKeys(userRelease, distributionType);

    keys.forEach((key) => {
      const existing = distributionMap.get(key) || { count: 0, duration: 0 };
      distributionMap.set(key, {
        count: existing.count + 1,
        duration: existing.duration + ESTIMATED_VINYL_DURATION_MINUTES,
      });
    });
  });

  return Array.from(distributionMap.entries())
    .map(([label, data]) => ({
      label,
      count: data.count,
      duration: data.duration,
    }))
    .sort((a, b) => b.count - a.count)
    .slice(0, topCount);
}

function getDistributionKeys(
  userRelease: UserRelease,
  distributionType: DistributionType,
): string[] {
  switch (distributionType) {
    case "artist":
      return userRelease.release.artists?.map((a) => a.name) || ["Unknown Artist"];

    case "genre":
      return userRelease.release.genres?.map((g) => g.name) || ["Unknown Genre"];

    case "release":
      return [userRelease.release.title || "Unknown Release"];

    default:
      return ["Unknown"];
  }
}

function applyFilter(
  playHistory: PlayHistory[],
  releases: UserRelease[],
  filter: ChartFilter,
): PlayHistory[] {
  return playHistory.filter((play) => {
    const userRelease = releases.find((r) => r.id === play.userReleaseId);
    if (!userRelease) return false;

    switch (filter.type) {
      case "record":
        return userRelease.id === filter.value;

      case "artist":
        return userRelease.release.artists?.some((a) => a.name === filter.value) || false;

      case "genre":
        return userRelease.release.genres?.some((g) => g.name === filter.value) || false;

      default:
        return true;
    }
  });
}

export function getFilterLabel(filter: ChartFilter | null): string {
  if (!filter) return "";

  switch (filter.type) {
    case "record":
      return filter.label;
    case "artist":
      return `by ${filter.label}`;
    case "genre":
      return `in ${filter.label}`;
    default:
      return "";
  }
}
