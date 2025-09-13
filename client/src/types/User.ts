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
  discogsToken?: string;
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

