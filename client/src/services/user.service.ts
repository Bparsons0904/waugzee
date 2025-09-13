import { api } from './api';
import { USER_ENDPOINTS } from '../constants/api.constants';
import type {
  User,
  UpdateDiscogsTokenRequest,
  UpdateDiscogsTokenResponse
} from '../types/User';

/**
 * User service for user-related API operations
 */
export class UserService {
  /**
   * Get current user profile
   */
  static async getCurrentUser(): Promise<{ user: User }> {
    return api.get<{ user: User }>(USER_ENDPOINTS.ME);
  }

  /**
   * Update user's Discogs token
   */
  static async updateDiscogsToken(
    request: UpdateDiscogsTokenRequest
  ): Promise<UpdateDiscogsTokenResponse> {
    return api.put<UpdateDiscogsTokenResponse>(USER_ENDPOINTS.ME_DISCOGS, request);
  }
}

// Export individual functions for convenience
export const getCurrentUser = UserService.getCurrentUser;
export const updateDiscogsToken = UserService.updateDiscogsToken;