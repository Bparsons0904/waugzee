import { Component, createSignal, Show } from "solid-js";
import { useApiPut } from "@services/apiHooks";
import { TextInput } from "../../forms/TextInput/TextInput";
import { USER_ENDPOINTS } from "@constants/api.constants";
import type { UpdateDiscogsTokenRequest, UpdateDiscogsTokenResponse } from "src/types/User";
import styles from "./DiscogsTokenModal.module.scss";

interface DiscogsTokenModalProps {
  onClose: () => void;
}

export const DiscogsTokenModal: Component<DiscogsTokenModalProps> = (props) => {
  const [token, setToken] = createSignal("");
  const [localError, setLocalError] = createSignal<string | null>(null);

  const updateTokenMutation = useApiPut<UpdateDiscogsTokenResponse, UpdateDiscogsTokenRequest>(
    USER_ENDPOINTS.ME_DISCOGS,
    undefined,
    {
      invalidateQueries: [["user"]],
    }
  );

  const handleSubmit = async (e: Event) => {
    e.preventDefault();
    const tokenValue = token().trim();

    if (!tokenValue) {
      setLocalError("Please enter a valid token");
      return;
    }

    setLocalError(null);

    try {
      await updateTokenMutation.mutateAsync({ token: tokenValue });
      console.log("Discogs token saved successfully");

      setToken("");
      props.onClose();
    } catch (error) {
      console.error("Token submission failed:", error);
      setLocalError(
        error instanceof Error
          ? error.message
          : "Failed to save token. Please try again.",
      );
    }
  };

  const displayError = () => localError();

  return (
    <div class={styles.container}>
      <div class={styles.infoSection}>
        <h3 class={styles.sectionTitle}>What is a Discogs Token?</h3>
        <p class={styles.text}>
          A Discogs API token allows Waugzee to securely access your Discogs
          collection, search the Discogs database, and sync your vinyl records
          without requiring your full Discogs credentials.
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
          <li>
            Sign in to your Discogs account if you're not already logged in
          </li>
          <li>
            Under "Personal access token", generate a new token or copy your
            existing one
          </li>
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
          Your token is stored securely and only used to access the Discogs API
          on your behalf. We never share your token or use it for any other
          purpose.
        </p>
      </div>
    </div>
  );
};
