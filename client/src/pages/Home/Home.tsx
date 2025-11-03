import {
  type ActionItem,
  ActionsSection,
} from "@components/dashboard/ActionsSection/ActionsSection";
import { StatsSection } from "@components/dashboard/StatsSection/StatsSection";
import { ROUTES } from "@constants/api.constants";
import { useUserData } from "@context/UserDataContext";
import { useApiPost } from "@services/apiHooks";
import { useNavigate } from "@solidjs/router";
import { type Component, createMemo, createSignal, onMount } from "solid-js";
import styles from "./Home.module.scss";

const preloadLogPlay = () => import("@pages/LogPlay/LogPlay");

interface SyncResponse {
  status: string;
  message: string;
}

const Home: Component = () => {
  const navigate = useNavigate();
  const { user } = useUserData();

  const [syncStatus, setSyncStatus] = createSignal<string>("");

  const syncMutation = useApiPost<SyncResponse, void>("/sync/syncCollection", undefined, {
    successMessage: "Collection sync started successfully!",
    errorMessage: "Failed to start collection sync. Please try again.",
    onSuccess: (data) => {
      setSyncStatus(data.message);
    },
    onError: () => {
      setSyncStatus("Sync failed. Please try again.");
    },
  });

  const handleNavigation = (route: string) => {
    navigate(route);
  };

  const handleSyncCollection = () => {
    if (!user()?.configuration?.discogsToken) {
      navigate(ROUTES.PROFILE);
      return;
    }

    setSyncStatus("Initiating collection sync...");
    syncMutation.mutate();
  };

  const handleViewAnalytics = () => {
    navigate("/analytics");
  };

  const getButtonText = () => {
    if (!user()?.configuration?.discogsToken) {
      return "Connect Discogs";
    }
    return syncMutation.isPending ? "Syncing..." : "Sync Collection";
  };

  const actionItems = createMemo((): ActionItem[] => [
    {
      title: "Log Play",
      description: "Track your listening sessions with notes and equipment details.",
      buttonText: "Log Now",
      onClick: () => handleNavigation(ROUTES.LOG_PLAY),
    },
    {
      title: "Play History",
      description: "Review your listening sessions and track record plays.",
      buttonText: "View History",
      onClick: () => handleNavigation(ROUTES.PLAY_HISTORY),
    },
    {
      title: "View Collection",
      description: "Browse and search through your vinyl collection.",
      buttonText: "Browse Collection",
      onClick: () => handleNavigation(ROUTES.COLLECTION),
    },
    {
      title: "My Styluses",
      description: "Manage your styluses and track needle wear.",
      buttonText: "Manage Styluses",
      onClick: () => handleNavigation(ROUTES.EQUIPMENT),
    },
    {
      title: "Sync Collection",
      description: user()?.configuration?.discogsToken
        ? syncStatus() || "Import your vinyl collection from Discogs."
        : "Connect your Discogs account to sync your collection.",
      buttonText: getButtonText(),
      onClick: handleSyncCollection,
      disabled: syncMutation.isPending,
    },
    {
      title: "Listening Insights",
      description: "Discover patterns in your listening habits.",
      buttonText: "View Insights",
      onClick: handleViewAnalytics,
    },
  ]);

  onMount(async () => {
    preloadLogPlay();
  });

  return (
    <div class={styles.container}>
      <div class={styles.header}>
        <div>
          <h1 class={styles.title}>Welcome back, {user()?.firstName || "User"}!</h1>
          <p class={styles.subtitle}>Your personal vinyl collection tracker</p>
        </div>
      </div>

      <StatsSection />

      <ActionsSection actions={actionItems()} />
    </div>
  );
};

export default Home;
