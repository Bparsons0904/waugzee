import { createContext, useContext, JSX } from "solid-js";
import { User, Folder, UserRelease, UserStylus, UserWithFoldersAndReleasesResponse } from "src/types/User";
import { useAuth } from "./AuthContext";
import { useApiQuery } from "@services/apiHooks";
import { useQueryClient } from "@tanstack/solid-query";
import { USER_ENDPOINTS } from "@constants/api.constants";

type UserDataContextValue = {
  user: () => User | null;
  folders: () => Folder[];
  releases: () => UserRelease[];
  styluses: () => UserStylus[];
  isLoading: () => boolean;
  error: () => string | null;
  updateUser: (user: User) => void;
  refreshUser: () => Promise<void>;
};

const UserDataContext = createContext<UserDataContextValue>(
  {} as UserDataContextValue,
);

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
      console.warn("Cannot refresh user - not authenticated");
      return;
    }

    await queryClient.invalidateQueries({ queryKey: ["user"] });
  };

  return (
    <UserDataContext.Provider
      value={{
        user: () => userQuery.data?.user || null,
        folders: () => userQuery.data?.folders || [],
        releases: () => userQuery.data?.releases || [],
        styluses: () => userQuery.data?.styluses || [],
        isLoading: () => userQuery.isLoading,
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
