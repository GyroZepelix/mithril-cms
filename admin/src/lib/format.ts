/**
 * Shared formatting and MIME-type utility functions.
 */

const BYTE_UNITS = ["B", "KB", "MB", "GB", "TB"];
const LOG_1024 = Math.log(1024);

/** Format a byte count into a human-readable string (e.g. "4.2 MB"). */
export function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 B";
  const i = Math.min(
    Math.floor(Math.log(bytes) / LOG_1024),
    BYTE_UNITS.length - 1,
  );
  const value = bytes / Math.pow(1024, i);
  return `${value.toFixed(i === 0 ? 0 : 1)} ${BYTE_UNITS[i]}`;
}

/** Format an ISO date string for short display (e.g. "Jan 5, 2025"). */
export function formatDateShort(dateStr: string): string {
  try {
    return new Date(dateStr).toLocaleDateString(undefined, {
      month: "short",
      day: "numeric",
      year: "numeric",
    });
  } catch {
    return dateStr;
  }
}

/** Format an ISO date string for detailed display (e.g. "January 5, 2025, 02:30 PM"). */
export function formatDateLong(dateStr: string): string {
  try {
    return new Date(dateStr).toLocaleDateString(undefined, {
      year: "numeric",
      month: "long",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  } catch {
    return dateStr;
  }
}

/** Check whether a MIME type represents an image. */
export function isImageMime(mime: string): boolean {
  return mime.startsWith("image/");
}
