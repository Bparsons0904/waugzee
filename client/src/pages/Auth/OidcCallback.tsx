import { Component, createEffect, createSignal } from "solid-js";
import { useSearchParams } from "@solidjs/router";
import { useAuth } from "@context/AuthContext";
import styles from "./Auth.module.scss";

const OidcCallback: Component = () => {
  const [searchParams] = useSearchParams();
  const { handleOIDCCallback } = useAuth();
  const [loading, setLoading] = createSignal(true);
  const [error, setError] = createSignal<string | null>(null);

  createEffect(async () => {
    try {
      const code = searchParams.code;
      const state = searchParams.state;
      const redirectUri = `${window.location.origin}/auth/callback`;

      if (!code) {
        throw new Error("Authorization code not received");
      }

      if (!state) {
        throw new Error("State parameter not received");
      }

      await handleOIDCCallback(code, state, redirectUri);
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : "Authentication failed";
      setError(errorMessage);
      console.error("OIDC callback error:", err);
    } finally {
      setLoading(false);
    }
  });

  return (
    <div class={styles.authContainer}>
      <div class={styles.authCard}>
        <div class={styles.authHeader}>
          <h1>Completing Sign In...</h1>
        </div>
        
        <div class={styles.authContent}>
          {loading() ? (
            <div class={styles.loadingContainer}>
              <div class={styles.spinner} />
              <p>Processing authentication...</p>
            </div>
          ) : error() ? (
            <div class={styles.errorContainer}>
              <h3>Authentication Error</h3>
              <p class={styles.errorMessage}>{error()}</p>
              <button 
                class={styles.retryButton}
                onClick={() => window.location.href = '/login'}
              >
                Try Again
              </button>
            </div>
          ) : (
            <div class={styles.successContainer}>
              <p>Authentication successful! Redirecting...</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default OidcCallback;