import { api } from './api';
import { USER_ENDPOINTS } from '../constants/api.constants';
import type {
  UpdateDiscogsTokenRequest,
  UpdateDiscogsTokenResponse,
  UpdateSelectedFolderRequest,
  UpdateSelectedFolderResponse,
  UserWithFoldersResponse
} from '../types/User';

/**
 * User service for user-related API operations
 */
export class UserService {
  /**
   * Get current user profile with folders
   */
  static async getCurrentUser(): Promise<UserWithFoldersResponse> {
    return api.get<UserWithFoldersResponse>(USER_ENDPOINTS.ME);
  }

  /**
   * Update user's Discogs token
   */
  static async updateDiscogsToken(
    request: UpdateDiscogsTokenRequest
  ): Promise<UpdateDiscogsTokenResponse> {
    return api.put<UpdateDiscogsTokenResponse>(USER_ENDPOINTS.ME_DISCOGS, request);
  }

  /**
   * Update user's selected folder
   */
  static async updateSelectedFolder(
    request: UpdateSelectedFolderRequest
  ): Promise<UpdateSelectedFolderResponse> {
    return api.put<UpdateSelectedFolderResponse>(USER_ENDPOINTS.ME_FOLDER, request);
  }
}

// Export individual functions for convenience
export const getCurrentUser = UserService.getCurrentUser;
export const updateDiscogsToken = UserService.updateDiscogsToken;
export const updateSelectedFolder = UserService.updateSelectedFolder;