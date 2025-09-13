import { Component, createSignal, onMount } from "solid-js";
import { useNavigate } from "@solidjs/router";
import { useAuth } from "@context/AuthContext";
import styles from "./Dashboard.module.scss";
import { Button } from "@components/common/ui/Button/Button";

interface DashboardStats {
  totalRecords: number;
  totalPlays: number;
  listeningHours: number;
  favoriteGenre: string;
}

const Dashboard: Component = () => {
  const { user } = useAuth();
  const navigate = useNavigate();
  
  const [stats, setStats] = createSignal<DashboardStats>({
    totalRecords: 0,
    totalPlays: 0,
    listeningHours: 0,
    favoriteGenre: "Loading...",
  });
  const [isLoading, setIsLoading] = createSignal(true);

  onMount(async () => {
    try {
      // TODO: Replace with actual API call
      await new Promise(resolve => setTimeout(resolve, 1000));

      setStats({
        totalRecords: 247,
        totalPlays: 1430,
        listeningHours: 89,
        favoriteGenre: "Jazz",
      });
    } catch (error) {
      console.error("Failed to load dashboard stats:", error);
    } finally {
      setIsLoading(false);
    }
  });

  const actionCards = [
    {
      title: "Log Play",
      description: "Record when you play a record from your collection.",
      icon: "🎵",
      action: () => navigate("/log"),
    },
    {
      title: "View Collection",
      description: "Browse and search through your vinyl collection.",
      icon: "💽",
      action: () => navigate("/collection"),
    },
    {
      title: "Play History",
      description: "View your listening history and statistics.",
      icon: "📊",
      action: () => navigate("/history"),
    },
    {
      title: "Equipment",
      description: "Manage your turntables, cartridges, and styluses.",
      icon: "🎧",
      action: () => navigate("/equipment"),
    },
    {
      title: "Sync Collection",
      description: "Sync your collection with your Discogs library.",
      icon: "🔄",
      action: () => handleSync(),
    },
    {
      title: "Analytics",
      description: "Explore insights about your collection and listening habits.",
      icon: "📈",
      action: () => navigate("/analytics"),
    },
  ];

  const handleSync = () => {
    // TODO: Implement collection sync
    console.log("Syncing collection...");
  };

  return (
    <div class={styles.dashboard}>
      <div class={styles.container}>
        <header class={styles.header}>
          <h1 class={styles.headerTitle}>
            Welcome back, {user()?.firstName || "User"}!
          </h1>
          <p class={styles.headerSubtitle}>
            Manage your vinyl collection and track your listening sessions.
          </p>
        </header>

        <section class={styles.section}>
          <h2 class={styles.sectionTitle}>Quick Actions</h2>
          
          <div class={styles.cardGrid}>
            {actionCards.map((card) => (
              <div 
                class={styles.actionCard}
                onClick={card.action}
              >
                <div class={styles.cardIcon}>{card.icon}</div>
                <h3 class={styles.cardTitle}>{card.title}</h3>
                <p class={styles.cardDescription}>{card.description}</p>
                <Button
                  variant="primary"
                  size="sm"
                  onClick={card.action}
                >
                  Get Started
                </Button>
              </div>
            ))}
          </div>
        </section>

        <section class={styles.section}>
          <h2 class={styles.sectionTitle}>Collection Overview</h2>
          
          <div class={styles.overviewGrid}>
            <div class={styles.statCard}>
              <div class={styles.statIcon}>💽</div>
              <div class={styles.statContent}>
                <h3 class={styles.statNumber}>
                  {isLoading() ? "--" : stats().totalRecords.toLocaleString()}
                </h3>
                <p class={styles.statLabel}>Total Records</p>
              </div>
            </div>
            
            <div class={styles.statCard}>
              <div class={styles.statIcon}>▶️</div>
              <div class={styles.statContent}>
                <h3 class={styles.statNumber}>
                  {isLoading() ? "--" : stats().totalPlays.toLocaleString()}
                </h3>
                <p class={styles.statLabel}>Total Plays</p>
              </div>
            </div>
            
            <div class={styles.statCard}>
              <div class={styles.statIcon}>⏱️</div>
              <div class={styles.statContent}>
                <h3 class={styles.statNumber}>
                  {isLoading() ? "--h" : `${stats().listeningHours}h`}
                </h3>
                <p class={styles.statLabel}>Listening Time</p>
              </div>
            </div>
            
            <div class={styles.statCard}>
              <div class={styles.statIcon}>🎯</div>
              <div class={styles.statContent}>
                <h3 class={styles.statNumber}>
                  {isLoading() ? "--" : stats().favoriteGenre}
                </h3>
                <p class={styles.statLabel}>Favorite Genre</p>
              </div>
            </div>
          </div>
        </section>
      </div>
    </div>
  );
};

export default Dashboard;
