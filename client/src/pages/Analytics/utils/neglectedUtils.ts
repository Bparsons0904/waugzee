import { daysBetween } from "@utils/dates";
import type { NeglectedRecord } from "src/types/Analytics";
import type { CleaningHistory, PlayHistory } from "src/types/Release";
import type { UserRelease } from "src/types/User";

export type NeglectMode = "play" | "cleaning";

export function findNeglectedRecords(
  releases: UserRelease[],
  playHistory: PlayHistory[],
  cleaningHistory: CleaningHistory[],
  daysThreshold: number,
  mode: NeglectMode = "play",
): NeglectedRecord[] {
  const now = new Date();
  const releasePlayMap = new Map<string, { lastPlayed?: Date; totalPlays: number }>();
  const releaseCleanMap = new Map<string, { lastCleaned?: Date; totalCleans: number }>();

  playHistory.forEach((play) => {
    const playDate = new Date(play.playedAt);
    const existing = releasePlayMap.get(play.userReleaseId);

    if (existing) {
      releasePlayMap.set(play.userReleaseId, {
        lastPlayed:
          !existing.lastPlayed || playDate > existing.lastPlayed ? playDate : existing.lastPlayed,
        totalPlays: existing.totalPlays + 1,
      });
    } else {
      releasePlayMap.set(play.userReleaseId, {
        lastPlayed: playDate,
        totalPlays: 1,
      });
    }
  });

  cleaningHistory.forEach((clean) => {
    const cleanDate = new Date(clean.cleanedAt);
    const existing = releaseCleanMap.get(clean.userReleaseId);

    if (existing) {
      releaseCleanMap.set(clean.userReleaseId, {
        lastCleaned:
          !existing.lastCleaned || cleanDate > existing.lastCleaned
            ? cleanDate
            : existing.lastCleaned,
        totalCleans: existing.totalCleans + 1,
      });
    } else {
      releaseCleanMap.set(clean.userReleaseId, {
        lastCleaned: cleanDate,
        totalCleans: 1,
      });
    }
  });

  const neglectedRecords: NeglectedRecord[] = [];

  releases.forEach((userRelease) => {
    const playData = releasePlayMap.get(userRelease.id);
    const cleanData = releaseCleanMap.get(userRelease.id);

    let daysSinceLastActivity: number;
    let lastActivityAt: Date | undefined;

    if (mode === "play") {
      if (playData?.lastPlayed) {
        lastActivityAt = playData.lastPlayed;
        daysSinceLastActivity = daysBetween(now, lastActivityAt);
      } else {
        const addedDate = new Date(userRelease.dateAdded);
        daysSinceLastActivity = daysBetween(now, addedDate);
      }
    } else {
      if (cleanData?.lastCleaned) {
        lastActivityAt = cleanData.lastCleaned;
        daysSinceLastActivity = daysBetween(now, lastActivityAt);
      } else {
        const addedDate = new Date(userRelease.dateAdded);
        daysSinceLastActivity = daysBetween(now, addedDate);
      }
    }

    if (daysSinceLastActivity >= daysThreshold) {
      neglectedRecords.push({
        userReleaseId: userRelease.id,
        releaseId: userRelease.release.id,
        title: userRelease.release.title || "Unknown",
        artistNames: userRelease.release.artists?.map((a) => a.name) || [],
        coverImage: userRelease.release.coverImage || userRelease.release.thumb,
        daysSinceLastActivity,
        lastActivityAt,
        totalPlays: playData?.totalPlays || 0,
        totalCleans: cleanData?.totalCleans || 0,
        lastPlayedAt: playData?.lastPlayed,
        lastCleanedAt: cleanData?.lastCleaned,
      });
    }
  });

  return neglectedRecords.sort((a, b) => b.daysSinceLastActivity - a.daysSinceLastActivity);
}

export function formatDaysSinceActivity(days: number): string {
  if (days === 0) {
    return "Today";
  }
  if (days === 1) {
    return "Yesterday";
  }
  if (days < 7) {
    return `${days} days ago`;
  }
  if (days < 30) {
    const weeks = Math.floor(days / 7);
    return `${weeks} week${weeks > 1 ? "s" : ""} ago`;
  }
  if (days < 365) {
    const months = Math.floor(days / 30);
    return `${months} month${months > 1 ? "s" : ""} ago`;
  }
  const years = Math.floor(days / 365);
  return `${years} year${years > 1 ? "s" : ""} ago`;
}
