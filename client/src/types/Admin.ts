export interface DownloadStatusResponse {
  year_month: string;
  status: ProcessingStatus;
  started_at?: string;
  download_completed_at?: string;
  processing_completed_at?: string;
  file_checksums?: FileChecksums;
  files?: FileStatusInfo;
  processing_steps?: Record<ProcessingStep, StepStatus>;
  retry_count: number;
  error_message?: string;
}

export type ProcessingStatus =
  | "not_started"
  | "downloading"
  | "ready_for_processing"
  | "processing"
  | "completed"
  | "failed";

export interface FileChecksums {
  artists_dump?: string;
  labels_dump?: string;
  masters_dump?: string;
  releases_dump?: string;
}

export interface FileStatusInfo {
  artists?: FileDownloadInfo;
  labels?: FileDownloadInfo;
  masters?: FileDownloadInfo;
  releases?: FileDownloadInfo;
}

export interface FileDownloadInfo {
  status: FileDownloadStatus;
  downloaded: boolean;
  validated: boolean;
  size: number;
  downloaded_at?: string;
  validated_at?: string;
  error_message?: string;
}

export type FileDownloadStatus = "not_started" | "downloading" | "failed" | "validated";

export type ProcessingStep =
  | "labels_processing"
  | "artists_processing"
  | "masters_processing"
  | "releases_processing"
  | "master_genres_collection"
  | "master_genres_upsert"
  | "master_genre_associations"
  | "release_genres_collection"
  | "release_genres_upsert"
  | "release_genre_associations"
  | "release_label_associations"
  | "master_artist_associations"
  | "release_artist_associations";

export interface StepStatus {
  completed: boolean;
  completed_at?: string;
  error_message?: string;
  records_count?: number;
  duration?: string;
}
