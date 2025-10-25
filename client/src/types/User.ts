import type { CleaningHistory, PlayHistory } from "./Release";

export interface UserConfiguration {
  id: string;
  userId: string;
  discogsToken?: string;
  discogsUsername?: string;
  selectedFolderId?: number;
}

export interface Folder {
  id: number;
  name: string;
  count: number;
  public: boolean;
}

export interface User {
  id: string;
  firstName: string;
  lastName: string;
  fullName: string;
  displayName: string;
  email?: string;
  isAdmin: boolean;
  isActive: boolean;
  lastLoginAt?: string;
  profileVerified: boolean;
  configuration?: UserConfiguration;
}

export interface UserWithFoldersResponse {
  user: User;
  folders: Folder[];
}

// OIDC Authentication Types
export interface AuthConfig {
  configured: boolean;
  domain?: string;
  instanceUrl?: string;
  clientId?: string;
}

export interface TokenExchangeRequest {
  code: string;
  redirect_uri: string;
  state?: string;
}

export interface TokenExchangeResponse {
  access_token: string;
  token_type: string;
  expires_in: number;
  user: User;
}

export interface LoginURLResponse {
  authorizationUrl: string;
  state: string;
}

// Legacy types - keep for backward compatibility but not used in OIDC flow
export interface LoginRequest {
  login: string;
  password: string;
}

// Discogs Integration Types
export interface UpdateDiscogsTokenRequest {
  token: string;
}

export interface UpdateDiscogsTokenResponse {
  user: User;
}

export interface UpdateSelectedFolderRequest {
  folderId: number;
}

export interface UpdateSelectedFolderResponse {
  user: User;
}

// Release and UserRelease Types
export interface Artist {
  id: number;
  name: string;
  resourceUrl?: string;
}

export interface Genre {
  id: number;
  name: string;
}

export interface Label {
  id: number;
  name: string;
  resourceUrl?: string;
  entityType?: string;
}

export interface Release {
  id: number;
  title: string;
  labelId?: number;
  masterId?: number;
  year?: number;
  country?: string;
  format: string;
  trackCount?: number;
  notes?: string;
  resourceUrl?: string;
  uri?: string;
  dateAdded?: string;
  dateChanged?: string;
  lastSynced?: string;
  thumb?: string;
  coverImage?: string;
  data?: Record<string, unknown>; // JSON data
  artists?: Artist[];
  genres?: Genre[];
  labels?: Label[];
}

export interface UserRelease {
  id: string;
  userId: string;
  releaseId: number;
  release: Release;
  instanceId: number;
  folderId: number;
  rating?: number;
  notes?: Record<string, unknown>; // JSON data
  dateAdded: string;
  active: boolean;
  playHistory: PlayHistory[];
  cleaningHistory: CleaningHistory[];
}

export interface Stylus {
  id: string;
  brand: string;
  model: string;
  type?: string;
  cartridgeType?: string;
  recommendedReplaceHours?: number;
  userGeneratedId?: string;
  isVerified?: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface UserStylus {
  id: string;
  userId: string;
  stylusId: string;
  stylus?: Stylus;
  purchaseDate?: string;
  installDate?: string;
  hoursUsed?: number;
  notes?: string;
  isActive: boolean;
  isPrimary: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface UserWithFoldersAndReleasesResponse {
  user: User;
  folders: Folder[];
  releases: UserRelease[];
  styluses: UserStylus[];
  playHistory: PlayHistory[];
}
