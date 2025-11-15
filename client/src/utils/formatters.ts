export function formatDuration(seconds: number): string {
  if (!seconds || seconds === 0) return "0s";

  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  const secs = Math.floor(seconds % 60);

  const parts: string[] = [];
  if (hours > 0) parts.push(`${hours}h`);
  if (minutes > 0) parts.push(`${minutes}m`);
  if (secs > 0) parts.push(`${secs}s`);

  return parts.length > 0 ? parts.join(" ") : "0s";
}

export function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 Bytes";

  const k = 1024;
  const sizes = ["Bytes", "KB", "MB", "GB", "TB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));

  return `${(bytes / k ** i).toFixed(2)} ${sizes[i]}`;
}

export function truncateHash(hash: string, startChars = 6, endChars = 6): string {
  if (!hash) return "";
  if (hash.length <= startChars + endChars) return hash;

  return `${hash.slice(0, startChars)}...${hash.slice(-endChars)}`;
}

export function formatTimestamp(isoString: string | undefined): string {
  if (!isoString) return "N/A";

  try {
    return new Date(isoString).toLocaleString();
  } catch {
    return "Invalid Date";
  }
}
