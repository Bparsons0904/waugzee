import type { PlayHistory } from "@models/Release";
import type { UserRelease } from "@models/User";

export function suggestRandom(releases: UserRelease[], count: number = 1): UserRelease[] {
  const shuffled = [...releases].sort(() => Math.random() - 0.5);
  return shuffled.slice(0, count);
}

export function suggestLeastPlayed(
  releases: UserRelease[],
  playHistory: PlayHistory[],
  count: number = 3,
): UserRelease[] {
  const playCounts = new Map<string, number>();
  playHistory.forEach((play) => {
    const currentCount = playCounts.get(play.userReleaseId) || 0;
    playCounts.set(play.userReleaseId, currentCount + 1);
  });

  const weighted = releases.map((release) => {
    const playCount = playCounts.get(release.id) || 0;
    const weight = 100 - Math.min(playCount * 10, 95) + Math.random() * 5;
    return { release, weight, playCount };
  });

  weighted.sort((a, b) => b.weight - a.weight);
  return weighted.slice(0, count).map((w) => w.release);
}

export function suggestByGenre(releases: UserRelease[]): {
  genre: string;
  releases: UserRelease[];
} {
  const genreSet = new Set<string>();
  releases.forEach((release) => {
    release.release?.genres?.forEach((genre) => {
      if (genre.name && genre.type === "genre") {
        genreSet.add(genre.name);
      }
    });
  });

  const genres = Array.from(genreSet);
  if (genres.length === 0) {
    return { genre: "All", releases: releases.slice(0, 3) };
  }

  const randomGenre = genres[Math.floor(Math.random() * genres.length)];

  const filtered = releases.filter((release) =>
    release.release?.genres?.some((g) => g.name === randomGenre && g.type === "genre"),
  );

  return { genre: randomGenre, releases: filtered.slice(0, 3) };
}
