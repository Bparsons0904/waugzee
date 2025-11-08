import { ConditionalRoot } from "@components/ConditionalRoot/ConditionalRoot";
import ProtectedRoute from "@components/ProtectedRoute";
import { ROUTES } from "@constants/api.constants";
import { Route } from "@solidjs/router";
import { type Component, lazy } from "solid-js";

const LoginPage = lazy(() => import("@pages/Auth/Login"));
const OidcCallbackPage = lazy(() => import("@pages/Auth/OidcCallback"));
const SilentCallbackPage = lazy(() => import("@pages/Auth/SilentCallback"));
const ProfilePage = lazy(() => import("@pages/Profile/Profile"));
const LogPlayPage = lazy(() => import("@pages/LogPlay/LogPlay"));
const EquipmentPage = lazy(() => import("@pages/Equipment/Equipment"));
const PlayHistoryPage = lazy(() => import("@pages/PlayHistory/PlayHistory"));
const ViewCollectionPage = lazy(() => import("@pages/ViewCollection/ViewCollection"));
const AnalyticsPage = lazy(() => import("@pages/Analytics/Analytics"));
const AdminPage = lazy(() => import("@pages/Admin/AdminPage"));

export const Routes: Component = () => {
  return (
    <>
      <Route path="/" component={ConditionalRoot} />
      <Route path={ROUTES.LOGIN} component={LoginPage} />
      <Route path={ROUTES.CALLBACK} component={OidcCallbackPage} />
      <Route path={ROUTES.SILENT_CALLBACK} component={SilentCallbackPage} />
      <Route
        path={ROUTES.PROFILE}
        component={() => <ProtectedRoute protectionLevel="authenticated" component={ProfilePage} />}
      />
      <Route
        path={ROUTES.LOG_PLAY}
        component={() => <ProtectedRoute protectionLevel="authenticated" component={LogPlayPage} />}
      />
      <Route
        path={ROUTES.EQUIPMENT}
        component={() => (
          <ProtectedRoute protectionLevel="authenticated" component={EquipmentPage} />
        )}
      />
      <Route
        path={ROUTES.PLAY_HISTORY}
        component={() => (
          <ProtectedRoute protectionLevel="authenticated" component={PlayHistoryPage} />
        )}
      />
      <Route
        path={ROUTES.COLLECTION}
        component={() => (
          <ProtectedRoute protectionLevel="authenticated" component={ViewCollectionPage} />
        )}
      />
      <Route
        path={ROUTES.ANALYTICS}
        component={() => (
          <ProtectedRoute protectionLevel="authenticated" component={AnalyticsPage} />
        )}
      />
      <Route
        path={ROUTES.ADMIN}
        component={() => <ProtectedRoute protectionLevel="admin" component={AdminPage} />}
      />
    </>
  );
};
