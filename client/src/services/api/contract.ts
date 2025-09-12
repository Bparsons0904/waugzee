import { z } from "zod";
import {
  User,
  LoginRequest,
  AuthConfig,
  TokenExchangeRequest,
  TokenExchangeResponse,
  LoginURLResponse,
} from "../../types/User";

export const apiContract = {
  auth: {
    // OIDC endpoints
    config: {
      path: "/auth/config",
      method: "GET",
      response: AuthConfig,
    },
    loginUrl: {
      path: "/auth/login-url",
      method: "GET",
      response: LoginURLResponse,
    },
    tokenExchange: {
      path: "/auth/token-exchange",
      method: "POST",
      body: TokenExchangeRequest,
      response: TokenExchangeResponse,
    },
    logout: {
      path: "/auth/logout",
      method: "POST",
      response: z.object({ message: z.string() }),
    },
    // Legacy endpoints - kept for backward compatibility
    login: {
      path: "/users/login",
      method: "POST",
      body: LoginRequest,
      response: z.object({ user: User, token: z.string() }),
    },
  },
  users: {
    me: {
      path: "/users/me",
      method: "GET",
      response: z.object({ user: User }),
    },
    get: {
      path: "/users",
      method: "GET",
      response: z.array(User),
    },
    getById: {
      path: (id: string) => `/api/users/${id}`,
      method: "GET",
      response: User,
    },
  },
} as const;
