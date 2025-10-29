import { daysBetween } from "@utils/dates";
import type { NeglectedRecord } from "src/types/Analytics";
import type { PlayHistory } from "src/types/Release";
import type { UserRelease } from "src/types/User";

export function findNeglectedRecords(
  releases: UserRelease[],
  playHistory: PlayHistory[],
  daysThreshold: number,
): NeglectedRecord[] {
  const now = new Date();
  const releasePlayMap = new Map<string, { lastPlayed?: Date; totalPlays: number }>();

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

  const neglectedRecords: NeglectedRecord[] = [];

  releases.forEach((userRelease) => {
    const playData = releasePlayMap.get(userRelease.id);

    let daysSinceLastPlay: number;
    let lastPlayedAt: Date | undefined;

    if (playData?.lastPlayed) {
      lastPlayedAt = playData.lastPlayed;
      daysSinceLastPlay = daysBetween(now, lastPlayedAt);
    } else {
      const addedDate = new Date(userRelease.dateAdded);
      daysSinceLastPlay = daysBetween(now, addedDate);
    }

    if (daysSinceLastPlay >= daysThreshold) {
      neglectedRecords.push({
        userReleaseId: userRelease.id,
        releaseId: userRelease.release.id,
        title: userRelease.release.title || "Unknown",
        artistNames: userRelease.release.artists?.map((a) => a.name) || [],
        coverImage: userRelease.release.coverImage || userRelease.release.thumb,
        daysSinceLastPlay,
        lastPlayedAt,
        totalPlays: playData?.totalPlays || 0,
      });
    }
  });

  return neglectedRecords.sort((a, b) => b.daysSinceLastPlay - a.daysSinceLastPlay);
}

export function formatDaysSincePlay(days: number): string {
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
