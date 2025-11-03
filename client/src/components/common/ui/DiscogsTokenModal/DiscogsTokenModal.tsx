import { USER_ENDPOINTS } from "@constants/api.constants";
import { useApiPost, useApiPut } from "@services/apiHooks";
import { type Component, createSignal, Show } from "solid-js";
import type { UpdateDiscogsTokenRequest, UpdateDiscogsTokenResponse } from "src/types/User";
import { TextInput } from "../../forms/TextInput/TextInput";
import styles from "./DiscogsTokenModal.module.scss";

interface SyncResponse {
  status: string;
  message: string;
}

interface DiscogsTokenModalProps {
  onClose: () => void;
}

export const DiscogsTokenModal: Component<DiscogsTokenModalProps> = (props) => {
  const [token, setToken] = createSignal("");
  const [localError, setLocalError] = createSignal<string | null>(null);

  const syncMutation = useApiPost<SyncResponse, void>("/sync/syncCollection", undefined, {
    successMessage: "Collection sync started successfully!",
    errorMessage: "Failed to start collection sync. You can sync manually from the dashboard.",
  });

  const updateTokenMutation = useApiPut<UpdateDiscogsTokenResponse, UpdateDiscogsTokenRequest>(
    USER_ENDPOINTS.ME_DISCOGS,
    undefined,
    {
      invalidateQueries: [["user"]],
      successMessage: "Discogs token saved! Starting collection sync...",
      errorMessage: "Failed to save token. Please try again.",
      onSuccess: () => {
        setToken("");
        props.onClose();
        syncMutation.mutate();
      },
      onError: (error) => {
        setLocalError(
          error instanceof Error ? error.message : "Failed to save token. Please try again.",
        );
      },
    },
  );

  const handleSubmit = (e: Event) => {
    e.preventDefault();
    const tokenValue = token().trim();

    if (!tokenValue) {
      setLocalError("Please enter a valid token");
      return;
    }

    setLocalError(null);
    updateTokenMutation.mutate({ token: tokenValue });
  };

  const displayError = () => localError();

  return (
    <div class={styles.container}>
      <div class={styles.infoSection}>
        <h3 class={styles.sectionTitle}>What is a Discogs Token?</h3>
        <p class={styles.text}>
          A Discogs API token allows Waugzee to securely access your Discogs collection, search the
          Discogs database, and sync your vinyl records without requiring your full Discogs
          credentials.
        </p>

        <h3 class={styles.sectionTitle}>How to Get Your Token</h3>
        <ol class={styles.instructionList}>
          <li>
            Go to your{" "}
            <a
              href="https://www.discogs.com/settings/developers"
              target="_blank"
              rel="noopener noreferrer"
              class={styles.link}
            >
              Discogs Developer Settings
            </a>
          </li>
          <li>Sign in to your Discogs account if you're not already logged in</li>
          <li>Under "Personal access token", generate a new token or copy your existing one</li>
          <li>Paste the token in the field below</li>
        </ol>
      </div>

      <form onSubmit={handleSubmit} class={styles.form}>
        <div class={styles.formGroup}>
          <TextInput
            label="Your Discogs API Token"
            value={token()}
            onInput={(value) => setToken(value)}
            placeholder="Paste your token here"
            required
            name="discogsToken"
          />
        </div>

        <Show when={displayError()}>
          <div class={styles.errorMessage}>{displayError()}</div>
        </Show>

        <div class={styles.actions}>
          <button
            type="submit"
            class={styles.primaryButton}
            disabled={!token().trim() || updateTokenMutation.isPending}
          >
            {updateTokenMutation.isPending ? "Saving..." : "Save Token"}
          </button>
        </div>
      </form>

      <div class={styles.footer}>
        <p class={styles.footerText}>
          Your token is stored securely and only used to access the Discogs API on your behalf. We
          never share your token or use it for any other purpose.
        </p>
      </div>
    </div>
  );
};
