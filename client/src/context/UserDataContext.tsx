import { USER_ENDPOINTS } from "@constants/api.constants";
import type { CleaningHistory, PlayHistory } from "@models/Release";
import { useApiQuery } from "@services/apiHooks";
import { useQueryClient } from "@tanstack/solid-query";
import { createContext, type JSX, useContext } from "solid-js";
import type {
  Folder,
  User,
  UserRelease,
  UserStylus,
  UserWithFoldersAndReleasesResponse,
} from "src/types/User";
import { useAuth } from "./AuthContext";

type UserDataContextValue = {
  user: () => User | null;
  folders: () => Folder[];
  releases: () => UserRelease[];
  styluses: () => UserStylus[];
  playHistory: () => PlayHistory[];
  cleaningHistory: () => CleaningHistory[];
  isLoading: () => boolean;
  error: () => string | null;
  updateUser: (user: User) => void;
  refreshUser: () => Promise<void>;
};

const UserDataContext = createContext<UserDataContextValue>({} as UserDataContextValue);

export function UserDataProvider(props: { children: JSX.Element }) {
  const auth = useAuth();
  const queryClient = useQueryClient();

  const userQuery = useApiQuery<UserWithFoldersAndReleasesResponse>(
    ["user"],
    USER_ENDPOINTS.ME,
    undefined,
    {
      enabled: () => auth.isAuthenticated() && !!auth.authToken(),
    },
  );

  const updateUser = (user: User) => {
    if (!auth.isAuthenticated()) {
      return;
    }

    // Optimistic update - update cache with new user data
    queryClient.setQueryData(
      ["user"],
      (oldData: UserWithFoldersAndReleasesResponse | undefined) => {
        if (!oldData) return oldData;
        return {
          ...oldData,
          user: user,
        };
      },
    );
  };

  const refreshUser = async () => {
    if (!auth.isAuthenticated()) {
      return;
    }

    await queryClient.invalidateQueries({ queryKey: ["user"] });
  };

  const cleaningHistory = () => {
    const releases = userQuery.data?.releases || [];
    return releases.flatMap((release) => release.cleaningHistory || []);
  };

  return (
    <UserDataContext.Provider
      value={{
        user: () => userQuery.data?.user || null,
        folders: () => userQuery.data?.folders || [],
        releases: () => userQuery.data?.releases || [],
        styluses: () => userQuery.data?.styluses || [],
        playHistory: () => userQuery.data?.playHistory || [],
        cleaningHistory,
        isLoading: () => userQuery.isPending,
        error: () => userQuery.error?.message || null,
        updateUser,
        refreshUser,
      }}
    >
      {props.children}
    </UserDataContext.Provider>
  );
}

export function useUserData() {
  return useContext(UserDataContext);
}
