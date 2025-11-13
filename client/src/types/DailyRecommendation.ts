import type { UserRelease } from "./User";

export interface DailyRecommendation {
  id: string;
  userId: string;
  userReleaseId: string;
  userRelease: UserRelease;
  date: string;
  listenedAt: string | null;
  algorithm: "smart" | "random";
  createdAt: string;
  updatedAt: string;
}
