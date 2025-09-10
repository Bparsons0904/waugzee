import { z } from 'zod';
import { 
  User, 
  LoginRequest, 
  RegisterRequest, 
  AuthConfig, 
  TokenExchangeRequest, 
  TokenExchangeResponse,
  LoginURLResponse
} from '../../types/User';

export const apiContract = {
  auth: {
    // OIDC endpoints
    config: {
      path: '/api/auth/config',
      method: 'GET',
      response: AuthConfig,
    },
    loginUrl: {
      path: '/api/auth/login-url',
      method: 'GET',
      response: LoginURLResponse,
    },
    tokenExchange: {
      path: '/api/auth/token-exchange',
      method: 'POST',
      body: TokenExchangeRequest,
      response: TokenExchangeResponse,
    },
    me: {
      path: '/api/auth/me',
      method: 'GET',
      response: z.object({ user: User }),
    },
    logout: {
      path: '/api/auth/logout',
      method: 'POST',
      response: z.object({ message: z.string() }),
    },
    // Legacy endpoints - kept for backward compatibility
    login: {
      path: '/api/users/login',
      method: 'POST',
      body: LoginRequest,
      response: z.object({ user: User, token: z.string() }),
    },
  },
  users: {
    get: {
      path: '/api/users',
      method: 'GET',
      response: z.array(User),
    },
    getById: {
      path: (id: string) => `/api/users/${id}`,
      method: 'GET',
      response: User,
    },
    create: {
      path: '/api/users',
      method: 'POST',
      body: RegisterRequest,
      response: User,
    },
  },
} as const;
