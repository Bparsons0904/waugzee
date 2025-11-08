import { Events, useWebSocket } from "@context/WebSocketContext";
import { useQueryClient } from "@tanstack/solid-query";
import {
  createContext,
  createEffect,
  createSignal,
  type JSX,
  onCleanup,
  useContext,
} from "solid-js";

interface SyncStatusContextValue {
  isSyncing: () => boolean;
  syncError: () => string | null;
}

const SyncStatusContext = createContext<SyncStatusContextValue>({} as SyncStatusContextValue);

interface SyncStatusProviderProps {
  children: JSX.Element;
}

export function SyncStatusProvider(props: SyncStatusProviderProps) {
  const { lastMessage } = useWebSocket();
  const queryClient = useQueryClient();
  const [isSyncing, setIsSyncing] = createSignal(false);
  const [syncError, setSyncError] = createSignal<string | null>(null);

  createEffect(() => {
    const message = lastMessage();
    if (!message) return;

    try {
      const parsedMessage = JSON.parse(message);

      switch (parsedMessage.event) {
        case Events.SYNC_START:
          setIsSyncing(true);
          setSyncError(null);
          break;

        case Events.SYNC_COMPLETE:
          setIsSyncing(false);
          setSyncError(null);
          // Invalidate user query to refresh data after sync completes
          queryClient.invalidateQueries({ queryKey: ["user"] });
          break;

        case Events.SYNC_ERROR:
          setIsSyncing(false);
          setSyncError(parsedMessage.payload?.message || "An error occurred during sync");
          break;
      }
    } catch (_error) {
      // Silently ignore parse errors
    }
  });

  onCleanup(() => {
    setIsSyncing(false);
    setSyncError(null);
  });

  const contextValue: SyncStatusContextValue = {
    isSyncing,
    syncError,
  };

  return (
    <SyncStatusContext.Provider value={contextValue}>{props.children}</SyncStatusContext.Provider>
  );
}

export function useSyncStatus() {
  const context = useContext(SyncStatusContext);
  if (!context) {
    throw new Error("useSyncStatus must be used within SyncStatusProvider");
  }
  return context;
}
