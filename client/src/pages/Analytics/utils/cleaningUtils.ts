import { daysBetween, generateDateKeys, getDateGroupKey, parseDateGroupKey } from "@utils/dates";
import type {
  CleaningFrequencyDataPoint,
  CleaningStats,
  DateRange,
  GroupFrequency,
} from "src/types/Analytics";
import type { CleaningHistory } from "src/types/Release";

export function calculateCleaningFrequency(
  cleaningHistory: CleaningHistory[],
  dateRange: DateRange,
  frequency: GroupFrequency,
): CleaningFrequencyDataPoint[] {
  const filteredHistory = cleaningHistory.filter((clean) => {
    const cleanDate = new Date(clean.cleanedAt);
    return cleanDate >= dateRange.start && cleanDate <= dateRange.end;
  });

  const dateKeys = generateDateKeys(dateRange.start, dateRange.end, frequency);
  const countsByDate = new Map<
    string,
    {
      regular: number;
      deep: number;
    }
  >();

  dateKeys.forEach((key) => {
    countsByDate.set(key, { regular: 0, deep: 0 });
  });

  filteredHistory.forEach((clean) => {
    const cleanDate = new Date(clean.cleanedAt);
    const key = getDateGroupKey(cleanDate, frequency);
    const existing = countsByDate.get(key) || { regular: 0, deep: 0 };

    if (clean.isDeepClean) {
      countsByDate.set(key, { ...existing, deep: existing.deep + 1 });
    } else {
      countsByDate.set(key, { ...existing, regular: existing.regular + 1 });
    }
  });

  return Array.from(countsByDate.entries())
    .map(([key, counts]) => ({
      date: parseDateGroupKey(key, frequency),
      regularCount: counts.regular,
      deepCleanCount: counts.deep,
      totalCount: counts.regular + counts.deep,
    }))
    .sort((a, b) => a.date.getTime() - b.date.getTime());
}

export function calculateCleaningStats(
  cleaningHistory: CleaningHistory[],
  dateRange: DateRange,
): CleaningStats {
  const filteredHistory = cleaningHistory.filter((clean) => {
    const cleanDate = new Date(clean.cleanedAt);
    return cleanDate >= dateRange.start && cleanDate <= dateRange.end;
  });

  const totalCleans = filteredHistory.length;
  const deepCleans = filteredHistory.filter((c) => c.isDeepClean).length;
  const regularCleans = totalCleans - deepCleans;

  let averageDaysBetweenCleans = 0;
  let lastCleanDate: Date | undefined;

  if (filteredHistory.length > 0) {
    const sortedCleans = filteredHistory
      .map((c) => new Date(c.cleanedAt))
      .sort((a, b) => a.getTime() - b.getTime());

    lastCleanDate = sortedCleans[sortedCleans.length - 1];

    if (sortedCleans.length > 1) {
      let totalDays = 0;
      for (let i = 1; i < sortedCleans.length; i++) {
        totalDays += daysBetween(sortedCleans[i], sortedCleans[i - 1]);
      }
      averageDaysBetweenCleans = Math.round(totalDays / (sortedCleans.length - 1));
    }
  }

  return {
    totalCleans,
    deepCleans,
    regularCleans,
    averageDaysBetweenCleans,
    lastCleanDate,
  };
}

export interface MostCleanedRecord {
  userReleaseId: string;
  releaseId: number;
  title: string;
  artistNames: string[];
  coverImage?: string;
  totalCleans: number;
  deepCleans: number;
  lastCleanedAt?: Date;
}

export function getMostCleanedRecords(
  cleaningHistory: CleaningHistory[],
  dateRange: DateRange,
  limit: number = 10,
): MostCleanedRecord[] {
  const filteredHistory = cleaningHistory.filter((clean) => {
    const cleanDate = new Date(clean.cleanedAt);
    return cleanDate >= dateRange.start && cleanDate <= dateRange.end;
  });

  const recordMap = new Map<
    string,
    {
      userRelease: CleaningHistory["userRelease"];
      totalCleans: number;
      deepCleans: number;
      lastCleanedAt: Date;
    }
  >();

  filteredHistory.forEach((clean) => {
    const existing = recordMap.get(clean.userReleaseId);
    const cleanDate = new Date(clean.cleanedAt);

    if (existing) {
      recordMap.set(clean.userReleaseId, {
        userRelease: clean.userRelease,
        totalCleans: existing.totalCleans + 1,
        deepCleans: existing.deepCleans + (clean.isDeepClean ? 1 : 0),
        lastCleanedAt: cleanDate > existing.lastCleanedAt ? cleanDate : existing.lastCleanedAt,
      });
    } else {
      recordMap.set(clean.userReleaseId, {
        userRelease: clean.userRelease,
        totalCleans: 1,
        deepCleans: clean.isDeepClean ? 1 : 0,
        lastCleanedAt: cleanDate,
      });
    }
  });

  return Array.from(recordMap.entries())
    .map(([userReleaseId, data]) => ({
      userReleaseId,
      releaseId: data.userRelease?.release?.id || 0,
      title: data.userRelease?.release?.title || "Unknown",
      artistNames: data.userRelease?.release?.artists?.map((a) => a.name) || [],
      coverImage: data.userRelease?.release?.coverImage || data.userRelease?.release?.thumb,
      totalCleans: data.totalCleans,
      deepCleans: data.deepCleans,
      lastCleanedAt: data.lastCleanedAt,
    }))
    .sort((a, b) => b.totalCleans - a.totalCleans)
    .slice(0, limit);
}
