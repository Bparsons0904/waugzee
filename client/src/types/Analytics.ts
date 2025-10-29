export type TimeFrame = "7d" | "30d" | "90d" | "1y" | "all" | "custom";
export type GroupFrequency = "daily" | "weekly" | "monthly";
export type DistributionType = "artist" | "genre" | "release";
export type FilterType = "record" | "artist" | "genre";

export interface DateRange {
  start: Date;
  end: Date;
}

export interface PlayFrequencyDataPoint {
  date: Date;
  count: number;
}

export interface PlayDurationDataPoint {
  date: Date;
  minutes: number;
}

export interface DistributionDataItem {
  label: string;
  count: number;
  duration: number;
  color?: string;
}

export interface ChartFilter {
  type: FilterType;
  value: string | number;
  label: string;
}

export interface CleaningFrequencyDataPoint {
  date: Date;
  regularCount: number;
  deepCleanCount: number;
  totalCount: number;
}

export interface CleaningStats {
  totalCleans: number;
  deepCleans: number;
  regularCleans: number;
  averageDaysBetweenCleans: number;
  lastCleanDate?: Date;
}

export interface NeglectedRecord {
  userReleaseId: string;
  releaseId: number;
  title: string;
  artistNames: string[];
  coverImage?: string;
  daysSinceLastPlay: number;
  lastPlayedAt?: Date;
  totalPlays: number;
}

export interface DropdownOption {
  value: string;
  label: string;
  disabled?: boolean;
  isHeader?: boolean;
  metadata?: string;
}
