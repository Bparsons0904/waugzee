import { describe, expect, it } from "vitest";
import { formatBytes, formatDuration, formatTimestamp, truncateHash } from "./formatters";

describe("formatDuration", () => {
  it("formats mixed hours, minutes, and seconds", () => {
    expect(formatDuration(3665)).toBe("1h 1m 5s");
  });

  it("handles null/undefined values", () => {
    expect(formatDuration(null as unknown as number)).toBe("0s");
  });
});

describe("formatBytes", () => {
  it("formats various byte sizes correctly", () => {
    expect(formatBytes(0)).toBe("0 Bytes");
    expect(formatBytes(1024)).toBe("1.00 KB");
    expect(formatBytes(1048576)).toBe("1.00 MB");
    expect(formatBytes(1073741824)).toBe("1.00 GB");
  });
});

describe("truncateHash", () => {
  it("truncates long hash with ellipsis", () => {
    expect(truncateHash("abcdef1234567890abcdef1234567890")).toBe("abcdef...567890");
  });

  it("handles null/undefined gracefully", () => {
    expect(truncateHash(undefined as unknown as string)).toBe("");
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
});
