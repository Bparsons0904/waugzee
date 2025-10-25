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
      await loginWithOIDC();
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
            <h1 class={styles.authTitle}>Welcome Back</h1>
            <p class={styles.authSubtitle}>Sign in to continue your vinyl journey</p>
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
                  {loading() ? "Signing In..." : "Sign In with Zitadel"}
                </Button>
              </div>

              <div class={styles.authInfo}>
                <p class={styles.authInfoText}>
                  You'll be redirected to our secure authentication provider to sign in.
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
