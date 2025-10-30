import { DiscogsTokenModal } from "@components/common/ui/DiscogsTokenModal";
import { Modal, ModalSize } from "@components/common/ui/Modal/Modal";
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

  const [showTokenModal, setShowTokenModal] = createSignal(false);
  const [syncStatus, setSyncStatus] = createSignal<string>("");

  const syncMutation = useApiPost<SyncResponse, void>("/sync/syncCollection", undefined, {
    successMessage: "Collection sync started successfully!",
    errorMessage: "Failed to start collection sync. Please try again.",
    onSuccess: (data) => {
      setSyncStatus(data.message);
      console.log("Sync response:", data);
    },
    onError: (error) => {
      console.error("Sync failed:", error);
      setSyncStatus("Sync failed. Please try again.");
    },
  });

  const handleNavigation = (route: string) => {
    navigate(route);
  };

  const handleSyncCollection = () => {
    if (!user()?.configuration?.discogsToken) {
      setShowTokenModal(true);
      return;
    }

    setSyncStatus("Initiating collection sync...");
    syncMutation.mutate();
  };

  const handleTokenModalClose = () => {
    setShowTokenModal(false);
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
      description: "Record when you play a record from your collection.",
      buttonText: "Log Now",
      onClick: () => handleNavigation(ROUTES.LOG_PLAY),
    },
    {
      title: "View Play History",
      description: "View your play history and listening statistics.",
      buttonText: "View Stats",
      onClick: () => handleNavigation(ROUTES.PLAY_HISTORY),
    },
    {
      title: "View Collection",
      description: "Browse and search through your vinyl collection.",
      buttonText: "View Collection",
      onClick: () => handleNavigation(ROUTES.COLLECTION),
    },
    {
      title: "View Styluses",
      description: "View, edit and add styluses to track wear.",
      buttonText: "View Styluses",
      onClick: () => handleNavigation(ROUTES.EQUIPMENT),
    },
    {
      title: "Sync Collection",
      description: user()?.configuration?.discogsToken
        ? syncStatus() || "Sync your Waugzee collection with your Discogs library."
        : "Connect your Discogs account to sync your collection.",
      buttonText: getButtonText(),
      onClick: handleSyncCollection,
      disabled: syncMutation.isPending,
    },
    {
      title: "View Analytics",
      description: "Explore insights about your collection and listening habits.",
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
        <button type="button" class={styles.primaryButton} onClick={() => setShowTokenModal(true)}>
          {user()?.configuration?.discogsToken ? "Update Discogs Token" : "Connect Discogs"}
        </button>
      </div>

      <StatsSection />

      <ActionsSection actions={actionItems()} />

      <Modal
        isOpen={showTokenModal()}
        onClose={handleTokenModalClose}
        size={ModalSize.Large}
        title="Discogs API Configuration"
      >
        <DiscogsTokenModal onClose={handleTokenModalClose} />
      </Modal>
    </div>
  );
};

export default Home;
