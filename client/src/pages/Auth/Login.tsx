import { Button } from "@components/common/ui/Button/Button";
import { useAuth } from "@context/AuthContext";
import { type Component, createSignal, Show } from "solid-js";
import styles from "./Auth.module.scss";

const Login: Component = () => {
  const { loginWithOIDC, authConfig } = useAuth();
  const [loading, setLoading] = createSignal(false);
  const [error, setError] = createSignal<string | null>(null);

  const handleOIDCLogin = async () => {
    try {
      setLoading(true);
      setError(null);

      // Retrieve stored return location from sessionStorage
      const returnTo = sessionStorage.getItem("returnTo");
      if (returnTo) sessionStorage.removeItem("returnTo");

      await loginWithOIDC(returnTo || undefined);
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : "Login failed";
      setError(errorMessage);
      console.error("Login error:", err);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div class={styles.authPage}>
      <div class={styles.authContainer}>
        <div class={styles.authCard}>
          <div class={styles.authHeader}>
            <h1 class={styles.authTitle}>Welcome to Waugzee</h1>
            <p class={styles.authSubtitle}>
              Sign in or create an account to start tracking your vinyl collection
            </p>
          </div>

          <div class={styles.authContent}>
            <Show
              when={authConfig()?.configured}
              fallback={
                <div class={styles.errorContainer}>
                  <p class={styles.errorMessage}>Authentication is not configured</p>
                </div>
              }
            >
              <Show when={error()}>
                <div class={styles.errorContainer}>
                  <p class={styles.errorMessage}>{error()}</p>
                </div>
              </Show>

              <div class={styles.authActions}>
                <Button variant="gradient" size="lg" onClick={handleOIDCLogin} disabled={loading()}>
                  {loading() ? "Signing In..." : "Continue"}
                </Button>
              </div>

              <div class={styles.authInfo}>
                <p class={styles.authInfoText}>
                  You'll be securely redirected to complete sign in or create your account.
                </p>
              </div>
            </Show>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Login;
