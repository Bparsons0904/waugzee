/**
 * Centralized API endpoint constants
 * All API endpoints in one place for easy management and consistency
 */

// Base API paths
export const API_PATHS = {
  AUTH: '/auth',
  USERS: '/users',
  HEALTH: '/health',
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
  PROFILE: (id: string) => `${API_PATHS.USERS}/${id}`,
} as const;

// Health endpoints
export const HEALTH_ENDPOINTS = {
  CHECK: API_PATHS.HEALTH,
} as const;

// Frontend route constants (for consistency with backend auth routes)
export const FRONTEND_ROUTES = {
  LANDING: '/',
  HOME: '/home',
  LOGIN: '/auth/login',
  CALLBACK: '/auth/callback',
  SILENT_CALLBACK: '/auth/silent-callback',
  DASHBOARD: '/dashboard',
  PROFILE: '/profile',
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