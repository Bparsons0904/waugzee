import { ConditionalRoot } from "@components/ConditionalRoot/ConditionalRoot";
import { ROUTES } from "@constants/api.constants";
import { useAuth } from "@context/AuthContext";
import { Route, useNavigate } from "@solidjs/router";
import { type Component, createEffect, lazy } from "solid-js";

const LoginPage = lazy(() => import("@pages/Auth/Login"));
const OidcCallbackPage = lazy(() => import("@pages/Auth/OidcCallback"));
const SilentCallbackPage = lazy(() => import("@pages/Auth/SilentCallback"));
const ProfilePage = lazy(() => import("@pages/Profile/Profile"));
const LogPlayPage = lazy(() => import("@pages/LogPlay/LogPlay"));
const EquipmentPage = lazy(() => import("@pages/Equipment/Equipment"));
const PlayHistoryPage = lazy(() => import("@pages/PlayHistory/PlayHistory"));

const ProtectedRoute = (Component: Component) => {
  return () => {
    const { isAuthenticated } = useAuth();
    const navigate = useNavigate();

    createEffect(() => {
      if (isAuthenticated() === false) {
        navigate(ROUTES.HOME, { replace: true });
      }
    });

    return isAuthenticated() === true ? <Component /> : null;
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
    </>
  );
};
