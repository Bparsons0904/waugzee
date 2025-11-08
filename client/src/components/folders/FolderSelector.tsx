import { USER_ENDPOINTS } from "@constants/api.constants";
import { useUserData } from "@context/UserDataContext";
import { useApiPut } from "@services/apiHooks";
import { type Component, Match, Switch } from "solid-js";
import type { UpdateSelectedFolderRequest, UpdateSelectedFolderResponse } from "src/types/User";
import { CompactFolderSelector } from "./CompactFolderSelector";
import { NavbarFolderSelector } from "./NavbarFolderSelector";

interface FolderSelectorProps {
  navbar?: boolean;
}

export const FolderSelector: Component<FolderSelectorProps> = (props) => {
  const userData = useUserData();

  const user = userData.user;
  const folders = userData.folders;

  const updateFolderMutation = useApiPut<UpdateSelectedFolderResponse, UpdateSelectedFolderRequest>(
    USER_ENDPOINTS.ME_FOLDER,
    undefined,
    {
      invalidateQueries: [["user"]],
      successMessage: (_, variables) => {
        const folderName = folders().find((f) => f.id === variables.folderId)?.name || "Unknown";
        return `Folder changed to "${folderName}"`;
      },
      errorMessage: "Failed to update folder selection",
    },
  );

  const selectedFolderId = () => user()?.configuration?.selectedFolderId;

  const selectedFolder = () => {
    const folderId = selectedFolderId();
    if (folderId === null || folderId === undefined) {
      const allFolders = folders();
      return allFolders.length > 0 ? allFolders[0] : null;
    }
    return folders().find((folder) => folder.id === folderId) || null;
  };

  const handleFolderChange = (folderId: number) => {
    updateFolderMutation.mutate({ folderId });
  };

  const selectionData = {
    folders,
    selectedFolderId,
    selectedFolder,
    handleFolderChange,
    isLoading: updateFolderMutation.isPending,
  };

  return (
    <Switch>
      <Match when={props.navbar}>
        <NavbarFolderSelector {...selectionData} />
      </Match>
      <Match when={!props.navbar}>
        <CompactFolderSelector {...selectionData} />
      </Match>
    </Switch>
  );
};
