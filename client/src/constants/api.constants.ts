/**
 * Centralized API endpoint constants
 * All API endpoints in one place for easy management and consistency
 */

// Base API paths
export const API_PATHS = {
  AUTH: "/auth",
  USERS: "/users",
  HEALTH: "/health",
} as const;

// Authentication endpoints
export const AUTH_ENDPOINTS = {
  CONFIG: `${API_PATHS.AUTH}/config`,
  CALLBACK: `${API_PATHS.AUTH}/callback`,
  LOGOUT: `${API_PATHS.AUTH}/logout`,
} as const;

// User endpoints
export const USER_ENDPOINTS = {
  LIST: API_PATHS.USERS,
  ME: `${API_PATHS.USERS}/me`,
  ME_DISCOGS: `${API_PATHS.USERS}/me/discogs`,
  ME_FOLDER: `${API_PATHS.USERS}/me/folder`,
  PROFILE: (id: string) => `${API_PATHS.USERS}/${id}`,
} as const;

// Stylus endpoints
export const STYLUS_ENDPOINTS = {
  AVAILABLE: "/styluses/available",
  USER_STYLUSES: "/styluses",
  CREATE: "/styluses",
  CUSTOM: "/styluses/custom",
  UPDATE: (id: string) => `/styluses/${id}`,
  DELETE: (id: string) => `/styluses/${id}`,
} as const;

// Play History endpoints
export const PLAY_HISTORY_ENDPOINTS = {
  CREATE: "/plays",
  DELETE: (id: string) => `/plays/${id}`,
} as const;

// Cleaning History endpoints
export const CLEANING_HISTORY_ENDPOINTS = {
  CREATE: "/cleanings",
  DELETE: (id: string) => `/cleanings/${id}`,
} as const;

// Combined History endpoints
export const HISTORY_ENDPOINTS = {
  LOG_BOTH: "/logBoth",
} as const;

// Health endpoints
export const HEALTH_ENDPOINTS = {
  CHECK: API_PATHS.HEALTH,
} as const;

// Frontend route constants (for consistency with backend auth routes)
export const ROUTES = {
  HOME: "/",
  LOGIN: "/auth/login",
  CALLBACK: "/auth/callback",
  SILENT_CALLBACK: "/auth/silentCallback",
  PROFILE: "/profile",
  LOG_PLAY: "/log",
  COLLECTION: "/collection",
  PLAY_HISTORY: "/playHistory",
  EQUIPMENT: "/equipment",
  DASHBOARD: "/dashboard",
} as const;

// Error retry configuration
export const RETRY_CONFIG = {
  MAX_ATTEMPTS: 3,
  BASE_DELAY_MS: 1000,
  MAX_DELAY_MS: 10000,
  EXPONENTIAL_BASE: 2,
} as const;

// Token expiry handling
export const TOKEN_CONFIG = {
  EXPIRY_BUFFER_MINUTES: 5, // Refresh token 5 minutes before expiry
  MAX_RETRY_ATTEMPTS: 2,
} as const;
