import { Component, createSignal, onMount, createMemo } from "solid-js";
import { useNavigate } from "@solidjs/router";
import { useAuth } from "@context/AuthContext";
import { useWebSocket } from "@context/WebSocketContext";
import { Modal, ModalSize } from "@components/common/ui/Modal/Modal";
import { DiscogsTokenModal } from "@components/common/ui/DiscogsTokenModal";
import {
  StatsSection,
  StatItem,
} from "@components/dashboard/StatsSection/StatsSection";
import {
  ActionsSection,
  ActionItem,
} from "@components/dashboard/ActionsSection/ActionsSection";
import { discogsProxyService } from "@services/discogs/discogsProxy.service";
import styles from "./Home.module.scss";

interface DashboardStats {
  totalRecords: number;
  totalPlays: number;
  listeningHours: number;
  favoriteGenre: string;
}

const Home: Component = () => {
  const navigate = useNavigate();
  const { user } = useAuth();
  const webSocket = useWebSocket();

  const [stats, setStats] = createSignal<DashboardStats>({
    totalRecords: 0,
    totalPlays: 0,
    listeningHours: 0,
    favoriteGenre: "Loading...",
  });
  const [isLoading, setIsLoading] = createSignal(true);
  const [showTokenModal, setShowTokenModal] = createSignal(false);
  const [isSyncing, setIsSyncing] = createSignal(false);
  const [syncStatus, setSyncStatus] = createSignal<string>("");

  // const hasDiscogsToken = user()?.discogsToken;
  //
  // console.log("hasDiscogsToken:", hasDiscogsToken);

  const handleLogPlay = () => {
    navigate("/log");
  };

  const handleViewCollection = () => {
    navigate("/collection");
  };

  const handleViewPlayHistory = () => {
    navigate("/playHistory");
  };

  const handleViewStyluses = () => {
    navigate("/equipment");
  };

  const handleSyncCollection = async () => {
    if (!user()?.discogsToken) {
      setShowTokenModal(true);
      return;
    }

    try {
      setIsSyncing(true);
      setSyncStatus("Starting sync...");

      const syncSession = await discogsProxyService.initiateCollectionSync({
        syncType: "collection",
        fullSync: false, // Start with incremental sync
        pageLimit: 10, // Limit pages for testing
      });

      setSyncStatus(`Sync started! Session: ${syncSession.sessionId}`);
      console.log("Collection sync initiated:", syncSession);

      // Set up progress callbacks
      const unsubscribeProgress = discogsProxyService.onSyncProgress((progress) => {
        setSyncStatus(`Syncing... ${progress.percentComplete.toFixed(1)}% complete`);
        console.log("Sync progress:", progress);
      });

      const unsubscribeComplete = discogsProxyService.onSyncComplete((sessionId) => {
        setSyncStatus("Sync completed successfully!");
        setIsSyncing(false);
        console.log("Sync completed:", sessionId);
        unsubscribeProgress();
        unsubscribeComplete();
        unsubscribeError();
      });

      const unsubscribeError = discogsProxyService.onSyncError((sessionId, error) => {
        setSyncStatus(`Sync failed: ${error}`);
        setIsSyncing(false);
        console.error("Sync error:", sessionId, error);
        unsubscribeProgress();
        unsubscribeComplete();
        unsubscribeError();
      });

    } catch (error) {
      console.error("Failed to start sync:", error);
      setSyncStatus("Failed to start sync");
      setIsSyncing(false);
    }
  };

  const handleTokenModalClose = () => {
    setShowTokenModal(false);
  };

  const handleViewAnalytics = () => {
    navigate("/analytics");
  };

  const statsItems = createMemo((): StatItem[] => [
    {
      icon: "💽",
      value: isLoading() ? "--" : stats().totalRecords.toLocaleString(),
      label: "Records",
      isLoading: isLoading(),
    },
    {
      icon: "▶️",
      value: isLoading() ? "--" : stats().totalPlays.toLocaleString(),
      label: "Plays",
      isLoading: isLoading(),
    },
    {
      icon: "⏱️",
      value: isLoading() ? "--h" : `${stats().listeningHours}h`,
      label: "Hours",
      isLoading: isLoading(),
    },
    {
      icon: "🎯",
      value: isLoading() ? "--" : stats().favoriteGenre,
      label: "Top Genre",
      isLoading: isLoading(),
    },
  ]);

  const actionItems = createMemo((): ActionItem[] => [
    {
      title: "Log Play",
      description: "Record when you play a record from your collection.",
      buttonText: "Log Now",
      onClick: handleLogPlay,
    },
    {
      title: "View Play History",
      description: "View your play history and listening statistics.",
      buttonText: "View Stats",
      onClick: handleViewPlayHistory,
    },
    {
      title: "View Collection",
      description: "Browse and search through your vinyl collection.",
      buttonText: "View Collection",
      onClick: handleViewCollection,
    },
    {
      title: "View Styluses",
      description: "View, edit and add styluses to track wear.",
      buttonText: "View Styluses",
      onClick: handleViewStyluses,
    },
    {
      title: "Sync Collection",
      description: user()?.discogsToken
        ? isSyncing()
          ? syncStatus() || "Syncing your collection..."
          : "Sync your Waugzee collection with your Discogs library."
        : "Connect your Discogs account to sync your collection.",
      buttonText: user()?.discogsToken
        ? isSyncing()
          ? "Syncing..."
          : "Sync Now"
        : "Connect Discogs",
      onClick: handleSyncCollection,
      disabled: isSyncing(),
    },
    {
      title: "View Analytics",
      description:
        "Explore insights about your collection and listening habits.",
      buttonText: "View Insights",
      onClick: handleViewAnalytics,
    },
  ]);

  onMount(async () => {
    try {
      // Initialize the Discogs proxy service with WebSocket context
      discogsProxyService.initialize(webSocket);

      await new Promise((resolve) => setTimeout(resolve, 1000));

      setStats({
        totalRecords: 247,
        totalPlays: 1430,
        listeningHours: 89,
        favoriteGenre: "Jazz",
      });
    } catch (error) {
      console.error("Failed to load dashboard data:", error);
    } finally {
      setIsLoading(false);
    }
  });

  return (
    <div class={styles.container}>
      <div class={styles.header}>
        <div>
          <h1 class={styles.title}>
            Welcome back, {user()?.firstName || "User"}!
          </h1>
          <p class={styles.subtitle}>Your personal vinyl collection tracker</p>
        </div>
        <button
          class={styles.primaryButton}
          onClick={() => setShowTokenModal(true)}
        >
          {user()?.discogsToken ? "Update Discogs Token" : "Connect Discogs"}
        </button>
      </div>

      <StatsSection stats={statsItems()} />

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
