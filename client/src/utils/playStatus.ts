export function getCleanlinessScore(
  lastCleanedDate: Date | null,
  playsSinceCleaning: number,
): number {
  if (!lastCleanedDate) {
    return 100;
  }

  const playScore = Math.min(100, (playsSinceCleaning / 5.01) * 100);

  return playScore;
}

export function getCleanlinessColor(score: number): string {
  if (score < 20) return "#35a173";
  if (score < 40) return "#59c48c";
  if (score < 60) return "#80d6aa";
  if (score < 80) return "#f59e0b";
  return "#e9493e";
}

export function getPlayRecencyScore(lastPlayedDate: Date | null): number {
  if (!lastPlayedDate) return 0;

  const now = new Date();
  const daysElapsed = (now.getTime() - lastPlayedDate.getTime()) / (24 * 60 * 60 * 1000);

  if (daysElapsed <= 7) return 100;
  if (daysElapsed <= 30) return 80;
  if (daysElapsed <= 90) return 60;
  if (daysElapsed <= 180) return 40;
  if (daysElapsed <= 365) return 20;
  return 0;
}

export function getPlayRecencyColor(score: number): string {
  if (score >= 80) return "#35a173";
  if (score >= 60) return "#59c48c";
  if (score >= 40) return "#80d6aa";
  if (score >= 20) return "#f59e0b";
  return "#e9493e";
}

export function getPlayRecencyText(lastPlayedDate: Date | null): string {
  if (!lastPlayedDate) return "Never played";

  const now = new Date();
  const daysElapsed = (now.getTime() - lastPlayedDate.getTime()) / (24 * 60 * 60 * 1000);

  if (daysElapsed <= 7) return "Played this week";
  if (daysElapsed <= 30) return "Played this month";
  // Claude could the months be Played in the last ${}months?
  if (daysElapsed <= 90) return "Played in the last 3 months";
  if (daysElapsed <= 180) return "Played in the last 6 months";
  // Maybe just return Played over a year and that is the default
  if (daysElapsed <= 365) return "Played in the last year";
  return "Not played recently";
}

export function getCleanlinessText(score: number): string {
  if (score < 20) return "Recently cleaned";
  if (score < 40) return "Clean";
  if (score < 60) return "May need cleaning soon";
  if (score < 80) return "Due for cleaning";
  return "Needs cleaning";
}

export function countPlaysSinceCleaning(
  playHistory: { playedAt: string }[],
  lastCleanedDate: Date | null,
): number {
  if (!lastCleanedDate) return playHistory.length;

  const lastCleanedTime = lastCleanedDate.getTime();

  return playHistory.filter((play) => {
    const playDate = new Date(play.playedAt);
    const playTime = playDate.getTime();

    return playTime > lastCleanedTime + 1;
  }).length;
}

export function getLastCleaningDate(
  cleaningHistory: { cleanedAt: string }[] | undefined,
): Date | null {
  if (!cleaningHistory || cleaningHistory.length === 0) return null;

  const sortedHistory = [...cleaningHistory].sort((a, b) => {
    return new Date(b.cleanedAt).getTime() - new Date(a.cleanedAt).getTime();
  });

  return new Date(sortedHistory[0].cleanedAt);
}

export function getLastPlayDate(playHistory: { playedAt: string }[] | undefined): Date | null {
  if (!playHistory || playHistory.length === 0) return null;

  const sortedHistory = [...playHistory].sort((a, b) => {
    return new Date(b.playedAt).getTime() - new Date(a.playedAt).getTime();
  });

  return new Date(sortedHistory[0].playedAt);
}
