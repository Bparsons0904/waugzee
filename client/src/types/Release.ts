export interface Artist {
  id: number;
  name: string;
  resourceUrl: string;
}

export interface ReleaseArtist {
  releaseId: number;
  artistId: number;
  joinRelation: string;
  anv: string;
  tracks: string;
  role: string;
  artist?: Artist;
}

export interface Label {
  id: number;
  name: string;
  resourceUrl: string;
  entityType: string;
}

export interface ReleaseLabel {
  releaseId: number;
  labelId: number;
  catNo: string;
  label?: Label;
}

export interface Format {
  id: number;
  releaseId: number;
  name: string;
  qty: number;
  descriptions: string[];
}

export interface Genre {
  id: number;
  name: string;
}

export interface Style {
  id: number;
  name: string;
}

export interface PlayHistory {
  id: string;
  userId: string;
  releaseId: number;
  release?: Release;
  userStylusId?: string;
  userStylus?: import("./User").UserStylus;
  playedAt: string;
  notes: string;
  createdAt: string;
  updatedAt: string;
}

export interface LogPlayRequest {
  releaseId: number;
  userStylusId?: string;
  playedAt: string;
  notes?: string;
}

export interface LogPlayResponse {
  playHistory: PlayHistory;
}

export interface PlayHistoryListResponse {
  playHistory: PlayHistory[];
  total: number;
  page: number;
  limit: number;
}

export interface CleaningHistory {
  id: string;
  userId: string;
  releaseId: number;
  release?: Release;
  cleanedAt: string;
  notes: string;
  isDeepClean: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface LogCleaningRequest {
  releaseId: number;
  cleanedAt: string;
  notes?: string;
  isDeepClean?: boolean;
}

export interface LogCleaningResponse {
  cleaningHistory: CleaningHistory;
}

export interface CleaningHistoryListResponse {
  cleaningHistory: CleaningHistory[];
  total: number;
  page: number;
  limit: number;
}

export interface Release {
  id: number;
  instanceId: number;
  folderId: number;
  rating: number;
  title: string;
  year: number | null;
  resourceUrl: string;
  thumb: string;
  coverImage: string;
  createdAt: string;
  updatedAt: string;
  lastSynced: string;

  artists: ReleaseArtist[];
  labels: ReleaseLabel[];
  formats: Format[];
  genres: Genre[];
  styles: Style[];
}

export interface EditItem {
  id: number;
  type: "play" | "cleaning";
  date: Date;
  notes?: string;
  stylus?: string;
  stylusId?: number;
  releaseId: number;
}
