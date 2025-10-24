function parseLocalDate(date: string | Date | null | undefined): Date | null {
  if (!date) return null;

  try {
    const dateObj = date instanceof Date ? date : new Date(date);
    if (Number.isNaN(dateObj.getTime())) {
      return null;
    }
    return dateObj;
  } catch (error) {
    console.error("Error parsing date:", error);
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

// Claude this is essentially the same as getLocalDateGroupKey
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

// Claude this is essentially the same as getLocalDateGroupKey
export function getLocalDateGroupKey(date: string | Date): string {
  const dateObj = parseLocalDate(date);
  if (!dateObj) return "";

  const year = dateObj.getFullYear();
  const month = String(dateObj.getMonth() + 1).padStart(2, "0");
  const day = String(dateObj.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

export function isSameLocalDay(date1: string | Date | null, date2: string | Date | null): boolean {
  if (!date1 || !date2) return false;

  const dateObj1 = parseLocalDate(date1);
  const dateObj2 = parseLocalDate(date2);

  if (!dateObj1 || !dateObj2) return false;

  return getLocalDateGroupKey(dateObj1) === getLocalDateGroupKey(dateObj2);
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
