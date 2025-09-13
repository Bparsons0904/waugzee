import { Component, createSignal, onMount, createMemo } from "solid-js";
import { useNavigate } from "@solidjs/router";
import { useAuth } from "@context/AuthContext";
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
import styles from "./Home.module.scss";
import { FRONTEND_ROUTES } from "@constants/api.constants";

interface DashboardStats {
  totalRecords: number;
  totalPlays: number;
  listeningHours: number;
  favoriteGenre: string;
}

const Home: Component = () => {
  const navigate = useNavigate();
  const { user } = useAuth();

  const [stats, setStats] = createSignal<DashboardStats>({
    totalRecords: 0,
    totalPlays: 0,
    listeningHours: 0,
    favoriteGenre: "Loading...",
  });
  const [isLoading, setIsLoading] = createSignal(true);
  const [showTokenModal, setShowTokenModal] = createSignal(false);

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

  const handleSyncCollection = () => {
    if (!user?.discogsToken) {
      setShowTokenModal(true);
    } else {
      // TODO: Implement sync with Discogs
      console.log("Sync collection functionality not yet implemented");
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
      description: user?.discogsToken
        ? "Sync your Waugzee collection with your Discogs library."
        : "Connect your Discogs account to sync your collection.",
      buttonText: user?.discogsToken ? "Sync Now" : "Connect Discogs",
      onClick: handleSyncCollection,
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
            Welcome back, {user?.firstName || "User"}!
          </h1>
          <p class={styles.subtitle}>Your personal vinyl collection tracker</p>
        </div>
        <button
          class={styles.primaryButton}
          onClick={() => setShowTokenModal(true)}
        >
          {user?.discogsToken ? "Update Discogs Token" : "Connect Discogs"}
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
