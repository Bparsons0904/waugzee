import { ConditionalRoot } from "@components/ConditionalRoot/ConditionalRoot";
import { LoadingSpinner } from "@components/icons/LoadingSpinner";
import { ROUTES } from "@constants/api.constants";
import { useAuth } from "@context/AuthContext";
import { useUserData } from "@context/UserDataContext";
import { Route, useLocation, useNavigate } from "@solidjs/router";
import { type Component, createEffect, lazy, Show } from "solid-js";

const LoginPage = lazy(() => import("@pages/Auth/Login"));
const OidcCallbackPage = lazy(() => import("@pages/Auth/OidcCallback"));
const SilentCallbackPage = lazy(() => import("@pages/Auth/SilentCallback"));
const ProfilePage = lazy(() => import("@pages/Profile/Profile"));
const LogPlayPage = lazy(() => import("@pages/LogPlay/LogPlay"));
const EquipmentPage = lazy(() => import("@pages/Equipment/Equipment"));
const PlayHistoryPage = lazy(() => import("@pages/PlayHistory/PlayHistory"));
const ViewCollectionPage = lazy(() => import("@pages/ViewCollection/ViewCollection"));
const AnalyticsPage = lazy(() => import("@pages/Analytics/Analytics"));

const ProtectedRoute = (Component: Component) => {
  return () => {
    const { authState } = useAuth();
    const { isLoading, error } = useUserData();
    const navigate = useNavigate();
    const location = useLocation();

    createEffect(() => {
      if (authState.status === "loading") return;

      if (authState.status === "unauthenticated") {
        sessionStorage.setItem("returnTo", location.pathname);
        navigate(ROUTES.LOGIN, { replace: true });
      }
    });

    return (
      <Show when={authState.status === "authenticated"}>
        <Show
          when={!error()}
          fallback={
            <div
              style={{
                display: "flex",
                "flex-direction": "column",
                "justify-content": "center",
                "align-items": "center",
                height: "100vh",
                padding: "2rem",
              }}
            >
              <h2>Failed to load user data</h2>
              <p>{error()}</p>
              <button type="button" onClick={() => window.location.reload()}>
                Reload Page
              </button>
            </div>
          }
        >
          <Show
            when={!isLoading()}
            fallback={
              <div
                style={{
                  display: "flex",
                  "justify-content": "center",
                  "align-items": "center",
                  height: "100vh",
                }}
              >
                <LoadingSpinner />
              </div>
            }
          >
            <Component />
          </Show>
        </Show>
      </Show>
    );
  };
};

export const Routes: Component = () => {
  return (
    <>
      <Route path="/" component={ConditionalRoot} />
      <Route path={ROUTES.LOGIN} component={LoginPage} />
      <Route path={ROUTES.CALLBACK} component={OidcCallbackPage} />
      <Route path={ROUTES.SILENT_CALLBACK} component={SilentCallbackPage} />
      <Route path={ROUTES.PROFILE} component={ProtectedRoute(ProfilePage)} />
      <Route path={ROUTES.LOG_PLAY} component={ProtectedRoute(LogPlayPage)} />
      <Route path={ROUTES.EQUIPMENT} component={ProtectedRoute(EquipmentPage)} />
      <Route path={ROUTES.PLAY_HISTORY} component={ProtectedRoute(PlayHistoryPage)} />
      <Route path={ROUTES.COLLECTION} component={ProtectedRoute(ViewCollectionPage)} />
      <Route path={ROUTES.ANALYTICS} component={ProtectedRoute(AnalyticsPage)} />
    </>
  );
};
