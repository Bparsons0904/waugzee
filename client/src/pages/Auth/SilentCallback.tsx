import { oidcService } from "@services/oidc.service";
import { type Component, createEffect } from "solid-js";

/**
 * Silent Callback Component
 *
 * This component handles the silent token renewal callback.
 * It runs in a hidden iframe to refresh tokens without user interaction.
 *
 * This page should be minimal and fast-loading to ensure smooth token renewal.
 */
const SilentCallback: Component = () => {
  createEffect(async () => {
    try {
      // Handle the silent renewal callback
      await oidcService.signInSilentCallback();
      console.debug("Silent token renewal successful");
    } catch (error) {
      console.error("Silent token renewal failed:", error);
      // The parent window will handle the error through the UserManager events
    }
  });

  // This component should render nothing or minimal content
  // as it's used in a hidden iframe
  return <div style={{ display: "none" }}>Silent token renewal in progress...</div>;
};

export default SilentCallback;
