import { describe, expect, it } from "vitest";
import { daysBetween, formatHistoryDate, isSameLocalDay } from "./dates";

describe("formatHistoryDate", () => {
  it("returns 'Invalid date' for invalid input", () => {
    expect(formatHistoryDate("invalid")).toBe("Invalid date");
  });

  it("returns 'Today' for today's date", () => {
    const today = new Date().toISOString();
    expect(formatHistoryDate(today)).toBe("Today");
  });

  it("returns formatted date for older dates", () => {
    const oldDate = new Date("2020-01-15T10:30:00Z");
    const result = formatHistoryDate(oldDate.toISOString());
    expect(result).toContain("Jan");
  });
});

describe("isSameLocalDay", () => {
  it("returns false for null inputs", () => {
    expect(isSameLocalDay(null, null)).toBe(false);
  });

  it("returns true for same day different times", () => {
    const date1 = "2024-01-15T10:30:00Z";
    const date2 = "2024-01-15T18:45:00Z";
    expect(isSameLocalDay(date1, date2)).toBe(true);
  });

  it("returns false for different days", () => {
    const date1 = "2024-01-15T10:30:00Z";
    const date2 = "2024-01-16T10:30:00Z";
    expect(isSameLocalDay(date1, date2)).toBe(false);
  });
});

describe("daysBetween", () => {
  it("calculates days between different dates", () => {
    const date1 = new Date("2024-01-15");
    const date2 = new Date("2024-01-20");
    expect(daysBetween(date1, date2)).toBe(5);
  });
});
