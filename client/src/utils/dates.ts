import type { DateRange, GroupFrequency, TimeFrame } from "src/types/Analytics";

function parseLocalDate(date: string | Date | null | undefined): Date | null {
  if (!date) return null;

  try {
    const dateObj = date instanceof Date ? date : new Date(date);
    if (Number.isNaN(dateObj.getTime())) {
      return null;
    }
    return dateObj;
  } catch {
    return null;
  }
}

export function useFormattedMediumDate(date: string | Date | null | undefined): string {
  if (!date) return "Never synced";

  const dateObj = parseLocalDate(date);
  if (!dateObj) return "Invalid date";

  return dateObj.toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "numeric",
    minute: "numeric",
    hour12: true,
  });
}

export function useFormattedShortDate(date: string | Date | null | undefined): string {
  if (!date) return "";

  const dateObj = parseLocalDate(date);
  if (!dateObj) return "Invalid date";

  return dateObj.toLocaleDateString(undefined, {
    month: "short",
    day: "numeric",
    year: "numeric",
  });
}

export function formatDateForInput(date: Date | string | null | undefined): string {
  const dateObj = parseLocalDate(date);
  if (!dateObj) return "";

  const year = dateObj.getFullYear();
  const month = String(dateObj.getMonth() + 1).padStart(2, "0");
  const day = String(dateObj.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

export function formatLocalDate(
  date: Date | string | null | undefined,
  fallback: string = "Never",
): string {
  if (!date) return fallback;

  const dateObj = parseLocalDate(date);
  if (!dateObj) return "Invalid date";

  return dateObj.toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
}

export function isSameLocalDay(date1: string | Date | null, date2: string | Date | null): boolean {
  if (!date1 || !date2) return false;

  const dateObj1 = parseLocalDate(date1);
  const dateObj2 = parseLocalDate(date2);

  if (!dateObj1 || !dateObj2) return false;

  return formatDateForInput(dateObj1) === formatDateForInput(dateObj2);
}

export function formatDateTimeForInput(date: Date | string | null | undefined): string {
  const dateObj = parseLocalDate(date);
  if (!dateObj) return "";

  const year = dateObj.getFullYear();
  const month = String(dateObj.getMonth() + 1).padStart(2, "0");
  const day = String(dateObj.getDate()).padStart(2, "0");
  const hours = String(dateObj.getHours()).padStart(2, "0");
  const minutes = String(dateObj.getMinutes()).padStart(2, "0");

  return `${year}-${month}-${day}T${hours}:${minutes}`;
}

export function formatHistoryDate(dateString: string): string {
  const date = parseLocalDate(dateString);
  if (!date) return "Invalid date";

  const now = new Date();
  const diffInDays = Math.floor((now.getTime() - date.getTime()) / (1000 * 60 * 60 * 24));

  switch (true) {
    case diffInDays === 0:
      return "Today";
    case diffInDays === 1:
      return "Yesterday";
    case diffInDays < 7:
      return `${diffInDays} days ago`;
    default:
      return date.toLocaleDateString("en-US", {
        month: "short",
        day: "numeric",
        year: date.getFullYear() !== now.getFullYear() ? "numeric" : undefined,
      });
  }
}

export function calculateDateRange(
  timeFrame: TimeFrame,
  customStart?: Date,
  customEnd?: Date,
): DateRange {
  const end = new Date();
  end.setHours(23, 59, 59, 999);

  let start = new Date();
  start.setHours(0, 0, 0, 0);

  switch (timeFrame) {
    case "7d":
      start.setDate(start.getDate() - 6);
      break;
    case "30d":
      start.setDate(start.getDate() - 29);
      break;
    case "90d":
      start.setDate(start.getDate() - 89);
      break;
    case "1y":
      start.setFullYear(start.getFullYear() - 1);
      break;
    case "all":
      start = new Date(0);
      break;
    case "custom":
      if (customStart && customEnd) {
        start = new Date(customStart);
        start.setHours(0, 0, 0, 0);
        const customEndDate = new Date(customEnd);
        customEndDate.setHours(23, 59, 59, 999);
        return { start, end: customEndDate };
      }
      break;
  }

  return { start, end };
}

export function getDateGroupKey(date: Date, frequency: GroupFrequency): string {
  const d = new Date(date);
  d.setHours(0, 0, 0, 0);

  switch (frequency) {
    case "daily":
      return `${d.getFullYear()}-${d.getMonth() + 1}-${d.getDate()}`;

    case "weekly": {
      const monday = new Date(d);
      const dayOfWeek = monday.getDay();
      const diff = dayOfWeek === 0 ? -6 : 1 - dayOfWeek;
      monday.setDate(monday.getDate() + diff);
      return `${monday.getFullYear()}-${monday.getMonth() + 1}-${monday.getDate()}`;
    }

    case "monthly":
      return `${d.getFullYear()}-${d.getMonth() + 1}`;

    default:
      return `${d.getFullYear()}-${d.getMonth() + 1}-${d.getDate()}`;
  }
}

export function parseDateGroupKey(key: string, frequency: GroupFrequency): Date {
  const parts = key.split("-").map(Number);

  if (frequency === "monthly") {
    return new Date(parts[0], parts[1] - 1, 1);
  }

  return new Date(parts[0], parts[1] - 1, parts[2]);
}

export function generateDateKeys(start: Date, end: Date, frequency: GroupFrequency): string[] {
  const keys: string[] = [];
  const current = new Date(start);
  current.setHours(0, 0, 0, 0);

  while (current <= end) {
    keys.push(getDateGroupKey(current, frequency));

    switch (frequency) {
      case "daily":
        current.setDate(current.getDate() + 1);
        break;
      case "weekly":
        current.setDate(current.getDate() + 7);
        break;
      case "monthly":
        current.setMonth(current.getMonth() + 1);
        break;
    }
  }

  return Array.from(new Set(keys)).sort();
}

export function formatDateForDisplay(date: Date, frequency: GroupFrequency): string {
  switch (frequency) {
    case "daily":
      return date.toLocaleDateString("en-US", { month: "short", day: "numeric", year: "numeric" });

    case "weekly": {
      const endOfWeek = new Date(date);
      endOfWeek.setDate(endOfWeek.getDate() + 6);
      return `${date.toLocaleDateString("en-US", { month: "short", day: "numeric" })} - ${endOfWeek.toLocaleDateString("en-US", { month: "short", day: "numeric" })}`;
    }

    case "monthly":
      return date.toLocaleDateString("en-US", { month: "long", year: "numeric" });

    default:
      return date.toLocaleDateString("en-US");
  }
}

export function daysBetween(date1: Date, date2: Date): number {
  const oneDay = 24 * 60 * 60 * 1000;
  return Math.round(Math.abs((date1.getTime() - date2.getTime()) / oneDay));
}

export function formatDuration(minutes: number): string {
  if (minutes < 60) {
    return `${minutes}m`;
  }

  const hours = Math.floor(minutes / 60);
  const mins = minutes % 60;

  if (mins === 0) {
    return `${hours}h`;
  }

  return `${hours}h ${mins}m`;
}
