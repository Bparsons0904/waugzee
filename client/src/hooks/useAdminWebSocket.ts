import { Events, useWebSocket } from "@context/WebSocketContext";
import { createEffect, createSignal } from "solid-js";
import type { DownloadProgressEvent } from "../types/Admin";

export const useAdminWebSocket = () => {
  const { lastMessage } = useWebSocket();
  const [progress, setProgress] = createSignal<DownloadProgressEvent | null>(null);

  createEffect(() => {
    const message = lastMessage();
    if (!message) return;

    try {
      const parsed = typeof message === "string" ? JSON.parse(message) : message;

      if (parsed.event === Events.ADMIN_DOWNLOAD_PROGRESS && parsed.payload) {
        setProgress(parsed.payload as DownloadProgressEvent);
      }
    } catch {
      // Silent fail - WebSocket context already handles logging
    }
  });

  return { progress };
};
