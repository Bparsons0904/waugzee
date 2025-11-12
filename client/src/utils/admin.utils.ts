import type { FileDownloadStatus, ProcessingStatus } from "../types/Admin";

// Generic status formatter - works for both ProcessingStatus and FileDownloadStatus
export function formatStatusLabel(status: string): string {
  return status
    .split("_")
    .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
    .join(" ");
}

// Status color mapping for ProcessingStatus
export function getProcessingStatusColor(status: ProcessingStatus): string {
  switch (status) {
    case "not_started":
      return "gray";
    case "downloading":
    case "processing":
      return "blue";
    case "ready_for_processing":
      return "yellow";
    case "completed":
      return "green";
    case "failed":
      return "red";
    default:
      return "gray";
  }
}

// Status color mapping for FileDownloadStatus
export function getFileStatusColor(status: FileDownloadStatus): string {
  switch (status) {
    case "not_started":
      return "gray";
    case "downloading":
      return "blue";
    case "failed":
      return "red";
    case "validated":
      return "green";
    default:
      return "gray";
  }
}

// Format bytes to human-readable size
export function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 Bytes";
  const k = 1024;
  const sizes = ["Bytes", "KB", "MB", "GB"];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return `${Math.round((bytes / k ** i) * 100) / 100} ${sizes[i]}`;
}

// Format Go duration string to human-readable format
export function formatDuration(durationStr?: string): string {
  if (!durationStr) return "N/A";

  // Parse Go duration format (e.g., "1h2m3.5s", "45m30s", "1.5h")
  const regex = /(?:(\d+(?:\.\d+)?)h)?(?:(\d+(?:\.\d+)?)m)?(?:(\d+(?:\.\d+)?)s)?/;
  const matches = durationStr.match(regex);

  if (!matches) return durationStr;

  const hours = parseFloat(matches[1] || "0");
  const minutes = parseFloat(matches[2] || "0");
  const seconds = parseFloat(matches[3] || "0");

  const parts: string[] = [];
  if (hours > 0) parts.push(`${Math.floor(hours)}h`);
  if (minutes > 0) parts.push(`${Math.floor(minutes)}m`);
  if (seconds > 0) parts.push(`${Math.floor(seconds)}s`);

  return parts.length > 0 ? parts.join(" ") : "0s";
}

// Truncate checksum for display
export function truncateChecksum(checksum: string): string {
  if (checksum.length <= 20) return checksum;
  return `${checksum.slice(0, 8)}...${checksum.slice(-8)}`;
}

// Format timestamp
export function formatTimestamp(timestamp?: string): string {
  if (!timestamp) return "N/A";
  return new Date(timestamp).toLocaleString();
}

// Format processing step name to readable label
export function getStepLabel(step: string): string {
  return formatStatusLabel(step);
}
