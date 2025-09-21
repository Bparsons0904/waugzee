import { api } from "./api";

export interface SyncResponse {
  status: string;
  message: string;
}

export const syncService = {
  initiateCollectionSync: async (): Promise<SyncResponse> => {
    return api.post<SyncResponse>("/sync/syncCollection");
  },
};
