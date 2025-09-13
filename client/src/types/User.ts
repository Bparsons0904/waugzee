import { z } from 'zod';

export const User = z.object({
  id: z.string(),
  firstName: z.string(),
  lastName: z.string(),
  fullName: z.string(),
  displayName: z.string(),
  email: z.string().email().optional(),
  isAdmin: z.boolean(),
  isActive: z.boolean(),
  lastLoginAt: z.string().datetime().optional(),
  profileVerified: z.boolean(),
});

export type User = z.infer<typeof User>;

// OIDC Authentication Types
export const AuthConfig = z.object({
  configured: z.boolean(),
  domain: z.string().optional(),
  instanceUrl: z.string().optional(),
  clientId: z.string().optional(),
});

export type AuthConfig = z.infer<typeof AuthConfig>;

export const TokenExchangeRequest = z.object({
  code: z.string(),
  redirect_uri: z.string(),
  state: z.string().optional(),
});

export type TokenExchangeRequest = z.infer<typeof TokenExchangeRequest>;

export const TokenExchangeResponse = z.object({
  access_token: z.string(),
  token_type: z.string(),
  expires_in: z.number(),
  user: User,
});

export type TokenExchangeResponse = z.infer<typeof TokenExchangeResponse>;

export const LoginURLResponse = z.object({
  authorizationUrl: z.string(),
  state: z.string(),
});

export type LoginURLResponse = z.infer<typeof LoginURLResponse>;

// Legacy types - keep for backward compatibility but not used in OIDC flow
export const LoginRequest = z.object({
  login: z.string(),
  password: z.string(),
});

export type LoginRequest = z.infer<typeof LoginRequest>;

