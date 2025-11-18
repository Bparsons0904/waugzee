import { describe, expect, it } from "vitest";
import { formatBytes, formatDuration, formatTimestamp, truncateHash } from "./formatters";

describe("formatDuration", () => {
  it("formats zero seconds correctly", () => {
    expect(formatDuration(0)).toBe("0s");
  });

  it("formats seconds only", () => {
    expect(formatDuration(45)).toBe("45s");
  });

  it("formats minutes only", () => {
    expect(formatDuration(120)).toBe("2m");
  });

  it("formats hours only", () => {
    expect(formatDuration(7200)).toBe("2h");
  });

  it("formats mixed hours, minutes, and seconds", () => {
    expect(formatDuration(3665)).toBe("1h 1m 5s");
  });

  it("handles large durations", () => {
    expect(formatDuration(86400)).toBe("24h");
  });

  it("handles undefined/falsy values", () => {
    expect(formatDuration(undefined as unknown as number)).toBe("0s");
    expect(formatDuration(null as unknown as number)).toBe("0s");
  });
});

describe("formatBytes", () => {
  it("formats zero bytes", () => {
    expect(formatBytes(0)).toBe("0 Bytes");
  });

  it("formats bytes correctly", () => {
    expect(formatBytes(500)).toBe("500.00 Bytes");
  });

  it("formats kilobytes correctly", () => {
    expect(formatBytes(1024)).toBe("1.00 KB");
    expect(formatBytes(2048)).toBe("2.00 KB");
  });

  it("formats megabytes correctly", () => {
    expect(formatBytes(1048576)).toBe("1.00 MB");
  });

  it("formats gigabytes correctly", () => {
    expect(formatBytes(1073741824)).toBe("1.00 GB");
  });

  it("formats terabytes correctly", () => {
    expect(formatBytes(1099511627776)).toBe("1.00 TB");
  });

  it("handles fractional values correctly", () => {
    expect(formatBytes(1536)).toBe("1.50 KB");
  });
});

describe("truncateHash", () => {
  it("returns empty string for empty input", () => {
    expect(truncateHash("")).toBe("");
  });

  it("returns full hash if shorter than truncation length", () => {
    expect(truncateHash("abc123")).toBe("abc123");
  });

  it("truncates long hash with default parameters", () => {
    expect(truncateHash("abcdef1234567890abcdef1234567890")).toBe(
      "abcdef...567890",
    );
  });

  it("truncates with custom start and end chars", () => {
    expect(truncateHash("abcdef1234567890abcdef", 4, 4)).toBe("abcd...cdef");
  });

  it("handles exactly matching length", () => {
    expect(truncateHash("abcdef123456", 6, 6)).toBe("abcdef123456");
  });

  it("handles undefined/null gracefully", () => {
    expect(truncateHash(undefined as unknown as string)).toBe("");
    expect(truncateHash(null as unknown as string)).toBe("");
  });
});

describe("formatTimestamp", () => {
  it("returns N/A for undefined", () => {
    expect(formatTimestamp(undefined)).toBe("N/A");
  });

  it("formats valid ISO string", () => {
    const result = formatTimestamp("2024-01-15T10:30:00Z");
    expect(result).not.toBe("N/A");
    expect(result).not.toBe("Invalid Date");
  });

  it("handles invalid date strings", () => {
    expect(formatTimestamp("not-a-date")).toBe("Invalid Date");
  });

  it("handles empty string", () => {
    expect(formatTimestamp("")).toBe("N/A");
  });

  it("formats recent dates correctly", () => {
    const now = new Date();
    const isoString = now.toISOString();
    const result = formatTimestamp(isoString);
    expect(result).toContain(now.getFullYear().toString());
  });
});
