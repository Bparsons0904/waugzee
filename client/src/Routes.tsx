import { Route } from "@solidjs/router";
import { Component, lazy } from "solid-js";
import { useAuth } from "@context/AuthContext";
import { useNavigate } from "@solidjs/router";
import { createEffect } from "solid-js";
import { ROUTES } from "@constants/api.constants";
import { ConditionalRoot } from "@components/ConditionalRoot/ConditionalRoot";

const LoginPage = lazy(() => import("@pages/Auth/Login"));
const OidcCallbackPage = lazy(() => import("@pages/Auth/OidcCallback"));
const SilentCallbackPage = lazy(() => import("@pages/Auth/SilentCallback"));
const ProfilePage = lazy(() => import("@pages/Profile/Profile"));

const LogPlayPageLoader = () => import("@pages/LogPlay/LogPlay");
const LogPlayPage = lazy(LogPlayPageLoader);

const ProtectedRoute = (Component: Component) => {
  return () => {
    const { isAuthenticated } = useAuth();
    const navigate = useNavigate();

    createEffect(() => {
      if (isAuthenticated() === false) {
        navigate(ROUTES.HOME, { replace: true });
      } else if (isAuthenticated() === true) {
        LogPlayPageLoader();
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
    </>
  );
};
