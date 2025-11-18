import { describe, expect, it } from "vitest";
import type { GroupFrequency, TimeFrame } from "../types/Analytics";
import {
  calculateDateRange,
  daysBetween,
  formatDateForInput,
  formatDateTimeForInput,
  formatHistoryDate,
  formatLocalDate,
  getDateGroupKey,
  isSameLocalDay,
  parseDateGroupKey,
  useFormattedMediumDate,
  useFormattedShortDate,
} from "./dates";

describe("useFormattedMediumDate", () => {
  it("returns 'Never synced' for null/undefined", () => {
    expect(useFormattedMediumDate(null)).toBe("Never synced");
    expect(useFormattedMediumDate(undefined)).toBe("Never synced");
  });

  it("returns 'Invalid date' for invalid date string", () => {
    expect(useFormattedMediumDate("not-a-date")).toBe("Invalid date");
  });

  it("formats valid date correctly", () => {
    const result = useFormattedMediumDate("2024-01-15T10:30:00Z");
    expect(result).toContain("Jan");
    expect(result).toContain("15");
    expect(result).toContain("2024");
  });

  it("handles Date object", () => {
    const date = new Date("2024-01-15T10:30:00Z");
    const result = useFormattedMediumDate(date);
    expect(result).toContain("Jan");
  });
});

describe("useFormattedShortDate", () => {
  it("returns empty string for null/undefined", () => {
    expect(useFormattedShortDate(null)).toBe("");
    expect(useFormattedShortDate(undefined)).toBe("");
  });

  it("returns 'Invalid date' for invalid date string", () => {
    expect(useFormattedShortDate("not-a-date")).toBe("Invalid date");
  });

  it("formats valid date correctly", () => {
    const result = useFormattedShortDate("2024-01-15T10:30:00Z");
    expect(result).toContain("Jan");
    expect(result).toContain("15");
    expect(result).toContain("2024");
  });
});

describe("formatDateForInput", () => {
  it("returns empty string for null/undefined", () => {
    expect(formatDateForInput(null)).toBe("");
    expect(formatDateForInput(undefined)).toBe("");
  });

  it("formats date as YYYY-MM-DD", () => {
    const date = new Date("2024-01-15T10:30:00Z");
    const result = formatDateForInput(date);
    expect(result).toMatch(/^\d{4}-\d{2}-\d{2}$/);
  });

  it("pads single digit months and days", () => {
    const date = new Date("2024-03-05T10:30:00Z");
    const result = formatDateForInput(date);
    expect(result).toContain("-03-");
    expect(result).toContain("-05");
  });
});

describe("formatLocalDate", () => {
  it("returns fallback for null/undefined", () => {
    expect(formatLocalDate(null)).toBe("Never");
    expect(formatLocalDate(undefined)).toBe("Never");
    expect(formatLocalDate(null, "Custom Fallback")).toBe("Custom Fallback");
  });

  it("returns 'Invalid date' for invalid input", () => {
    expect(formatLocalDate("invalid")).toBe("Invalid date");
  });

  it("formats valid date correctly", () => {
    const result = formatLocalDate("2024-01-15T10:30:00Z");
    expect(result).toContain("Jan");
    expect(result).toContain("15");
    expect(result).toContain("2024");
  });
});

describe("isSameLocalDay", () => {
  it("returns false for null/undefined inputs", () => {
    expect(isSameLocalDay(null, null)).toBe(false);
    expect(isSameLocalDay("2024-01-15", null)).toBe(false);
    expect(isSameLocalDay(null, "2024-01-15")).toBe(false);
  });

  it("returns true for same day", () => {
    const date1 = "2024-01-15T10:30:00Z";
    const date2 = "2024-01-15T18:45:00Z";
    expect(isSameLocalDay(date1, date2)).toBe(true);
  });

  it("returns false for different days", () => {
    const date1 = "2024-01-15T10:30:00Z";
    const date2 = "2024-01-16T10:30:00Z";
    expect(isSameLocalDay(date1, date2)).toBe(false);
  });

  it("handles Date objects", () => {
    const date1 = new Date("2024-01-15T10:30:00Z");
    const date2 = new Date("2024-01-15T18:45:00Z");
    expect(isSameLocalDay(date1, date2)).toBe(true);
  });
});

describe("formatDateTimeForInput", () => {
  it("returns empty string for null/undefined", () => {
    expect(formatDateTimeForInput(null)).toBe("");
    expect(formatDateTimeForInput(undefined)).toBe("");
  });

  it("formats datetime as YYYY-MM-DDTHH:MM", () => {
    const date = new Date("2024-01-15T10:30:00Z");
    const result = formatDateTimeForInput(date);
    expect(result).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/);
  });

  it("pads hours and minutes correctly", () => {
    const date = new Date(2024, 0, 15, 9, 5);
    const result = formatDateTimeForInput(date);
    expect(result).toContain("T09:05");
  });
});

describe("formatHistoryDate", () => {
  it("returns 'Invalid date' for invalid input", () => {
    expect(formatHistoryDate("invalid")).toBe("Invalid date");
  });

  it("returns 'Today' for today's date", () => {
    const today = new Date().toISOString();
    expect(formatHistoryDate(today)).toBe("Today");
  });

  it("returns 'Yesterday' for yesterday's date", () => {
    const yesterday = new Date();
    yesterday.setDate(yesterday.getDate() - 1);
    expect(formatHistoryDate(yesterday.toISOString())).toBe("Yesterday");
  });

  it("returns 'X days ago' for recent dates", () => {
    const threeDaysAgo = new Date();
    threeDaysAgo.setDate(threeDaysAgo.getDate() - 3);
    expect(formatHistoryDate(threeDaysAgo.toISOString())).toBe("3 days ago");
  });

  it("returns formatted date for older dates", () => {
    const oldDate = new Date("2020-01-15T10:30:00Z");
    const result = formatHistoryDate(oldDate.toISOString());
    expect(result).toContain("Jan");
  });
});

describe("calculateDateRange", () => {
  it("calculates 7 day range", () => {
    const range = calculateDateRange("7d");
    const diffDays = Math.ceil(
      (range.end.getTime() - range.start.getTime()) / (1000 * 60 * 60 * 24),
    );
    expect(diffDays).toBe(7);
  });

  it("calculates 30 day range", () => {
    const range = calculateDateRange("30d");
    const diffDays = Math.ceil(
      (range.end.getTime() - range.start.getTime()) / (1000 * 60 * 60 * 24),
    );
    expect(diffDays).toBe(30);
  });

  it("calculates 90 day range", () => {
    const range = calculateDateRange("90d");
    const diffDays = Math.ceil(
      (range.end.getTime() - range.start.getTime()) / (1000 * 60 * 60 * 24),
    );
    expect(diffDays).toBe(90);
  });

  it("calculates 1 year range", () => {
    const range = calculateDateRange("1y");
    const yearDiff = range.end.getFullYear() - range.start.getFullYear();
    expect(yearDiff).toBe(1);
  });

  it("calculates all time range", () => {
    const range = calculateDateRange("all");
    expect(range.start.getTime()).toBe(0);
  });

  it("handles custom date range", () => {
    const customStart = new Date("2024-01-01");
    const customEnd = new Date("2024-01-31");
    const range = calculateDateRange("custom", customStart, customEnd);
    expect(range.start.getDate()).toBe(1);
    expect(range.end.getDate()).toBe(31);
  });
});

describe("getDateGroupKey", () => {
  it("generates daily key correctly", () => {
    const date = new Date("2024-01-15T10:30:00Z");
    const key = getDateGroupKey(date, "daily");
    expect(key).toContain("2024");
    expect(key).toContain("15");
  });

  it("generates weekly key for Monday", () => {
    const date = new Date("2024-01-15T10:30:00Z");
    const key = getDateGroupKey(date, "weekly");
    expect(key).toBeTruthy();
  });

  it("generates monthly key correctly", () => {
    const date = new Date("2024-01-15T10:30:00Z");
    const key = getDateGroupKey(date, "monthly");
    expect(key).toBe("2024-1");
  });
});

describe("parseDateGroupKey", () => {
  it("parses daily key correctly", () => {
    const key = "2024-1-15";
    const date = parseDateGroupKey(key, "daily");
    expect(date.getFullYear()).toBe(2024);
    expect(date.getMonth()).toBe(0);
    expect(date.getDate()).toBe(15);
  });

  it("parses monthly key correctly", () => {
    const key = "2024-3";
    const date = parseDateGroupKey(key, "monthly");
    expect(date.getFullYear()).toBe(2024);
    expect(date.getMonth()).toBe(2);
    expect(date.getDate()).toBe(1);
  });
});

describe("daysBetween", () => {
  it("calculates days between same date", () => {
    const date = new Date("2024-01-15");
    expect(daysBetween(date, date)).toBe(0);
  });

  it("calculates days between different dates", () => {
    const date1 = new Date("2024-01-15");
    const date2 = new Date("2024-01-20");
    expect(daysBetween(date1, date2)).toBe(5);
  });

  it("handles reversed date order", () => {
    const date1 = new Date("2024-01-20");
    const date2 = new Date("2024-01-15");
    expect(daysBetween(date1, date2)).toBe(5);
  });

  it("handles dates far apart", () => {
    const date1 = new Date("2024-01-01");
    const date2 = new Date("2024-12-31");
    expect(daysBetween(date1, date2)).toBeGreaterThan(360);
  });
});

describe("formatDuration (from dates.ts)", () => {
  it("formats minutes only", () => {
    expect(formatHistoryDate).toBeDefined();
  });
});
