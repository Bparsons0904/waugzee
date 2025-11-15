import { Button } from "@components/common/ui/Button/Button";
import { LoadingSpinner } from "@components/icons/LoadingSpinner";
import { ROUTES } from "@constants/api.constants";
import { useAuth } from "@context/AuthContext";
import { useUserData } from "@context/UserDataContext";
import { useLocation, useNavigate } from "@solidjs/router";
import { type Component, createEffect, type JSX, Show } from "solid-js";
import styles from "./ProtectedRoute.module.scss";

type ProtectionLevel = "authenticated" | "admin";

interface ProtectedRouteProps {
  protectionLevel?: ProtectionLevel;
  component: Component;
}

const ErrorDisplay = (props: { error: string | null }) => {
  return (
    <div class={styles.errorContainer}>
      <h2>Failed to load user data</h2>
      <p>{props.error}</p>
      <Button onClick={() => window.location.reload()}>Reload Page</Button>
    </div>
  );
};

const LoadingDisplay = () => {
  return (
    <div class={styles.loadingContainer}>
      <LoadingSpinner />
    </div>
  );
};

const ProtectedRoute = (props: ProtectedRouteProps): JSX.Element => {
  const protectionLevel = props.protectionLevel || "authenticated";
  const { authState } = useAuth();
  const { isLoading, error, user } = useUserData();
  const navigate = useNavigate();
  const location = useLocation();

  createEffect(() => {
    if (authState.status === "loading") return;

    if (authState.status === "unauthenticated") {
      sessionStorage.setItem("returnTo", location.pathname);
      navigate(ROUTES.LOGIN, { replace: true });
    }
  });

  createEffect(() => {
    if (!isLoading() && user() && protectionLevel === "admin" && !user()?.isAdmin) {
      navigate(ROUTES.HOME, { replace: true });
    }
  });

  const shouldRender = () => {
    if (protectionLevel === "authenticated") {
      return !isLoading();
    }
    if (protectionLevel === "admin") {
      return user()?.isAdmin && !isLoading();
    }
    return false;
  };

  return (
    <Show when={authState.status === "authenticated"}>
      <Show when={!error()} fallback={<ErrorDisplay error={error()} />}>
        <Show when={shouldRender()} fallback={<LoadingDisplay />}>
          <props.component />
        </Show>
      </Show>
    </Show>
  );
};

export default ProtectedRoute;
