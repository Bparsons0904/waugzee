import { Component, createSignal, onMount } from "solid-js";
import { useNavigate } from "@solidjs/router";
import { useAuth } from "@context/AuthContext";
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
  
  const [stats, setStats] = createSignal<DashboardStats>({
    totalRecords: 0,
    totalPlays: 0,
    listeningHours: 0,
    favoriteGenre: "Loading...",
  });
  const [isLoading, setIsLoading] = createSignal(true);

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
    // TODO: Implement sync with Discogs
    console.log("Sync collection functionality not yet implemented");
  };

  const handleViewAnalytics = () => {
    navigate("/analytics");
  };

  onMount(async () => {
    try {
      await new Promise(resolve => setTimeout(resolve, 1000));
      
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
      <h1 class={styles.title}>Welcome back, {user?.firstName || "User"}!</h1>
      <p class={styles.subtitle}>Your personal vinyl collection tracker</p>

      <section class={styles.statsSection}>
        <div class={styles.statsGrid}>
          <div class={styles.statCard}>
            <div class={styles.statIcon}>💽</div>
            <div class={styles.statContent}>
              <h3 class={styles.statNumber}>
                {isLoading() ? "--" : stats().totalRecords.toLocaleString()}
              </h3>
              <p class={styles.statLabel}>Records</p>
            </div>
          </div>
          
          <div class={styles.statCard}>
            <div class={styles.statIcon}>▶️</div>
            <div class={styles.statContent}>
              <h3 class={styles.statNumber}>
                {isLoading() ? "--" : stats().totalPlays.toLocaleString()}
              </h3>
              <p class={styles.statLabel}>Plays</p>
            </div>
          </div>
          
          <div class={styles.statCard}>
            <div class={styles.statIcon}>⏱️</div>
            <div class={styles.statContent}>
              <h3 class={styles.statNumber}>
                {isLoading() ? "--h" : `${stats().listeningHours}h`}
              </h3>
              <p class={styles.statLabel}>Hours</p>
            </div>
          </div>
          
          <div class={styles.statCard}>
            <div class={styles.statIcon}>🎯</div>
            <div class={styles.statContent}>
              <h3 class={styles.statNumber}>
                {isLoading() ? "--" : stats().favoriteGenre}
              </h3>
              <p class={styles.statLabel}>Top Genre</p>
            </div>
          </div>
        </div>
      </section>

      <div class={styles.cardGrid}>
        <div class={styles.card}>
          <div class={styles.cardHeader}>
            <h2>Log Play</h2>
          </div>
          <div class={styles.cardBody}>
            <p>Record when you play a record from your collection.</p>
          </div>
          <div class={styles.cardFooter}>
            <button class={styles.button} onClick={handleLogPlay}>
              Log Now
            </button>
          </div>
        </div>

        <div class={styles.card}>
          <div class={styles.cardHeader}>
            <h2>View Play History</h2>
          </div>
          <div class={styles.cardBody}>
            <p>View your play history and listening statistics.</p>
          </div>
          <div class={styles.cardFooter}>
            <button class={styles.button} onClick={handleViewPlayHistory}>
              View Stats
            </button>
          </div>
        </div>

        <div class={styles.card}>
          <div class={styles.cardHeader}>
            <h2>View Collection</h2>
          </div>
          <div class={styles.cardBody}>
            <p>Browse and search through your vinyl collection.</p>
          </div>
          <div class={styles.cardFooter}>
            <button class={styles.button} onClick={handleViewCollection}>
              View Collection
            </button>
          </div>
        </div>

        <div class={styles.card}>
          <div class={styles.cardHeader}>
            <h2>View Styluses</h2>
          </div>
          <div class={styles.cardBody}>
            <p>View, edit and add styluses to track wear.</p>
          </div>
          <div class={styles.cardFooter}>
            <button class={styles.button} onClick={handleViewStyluses}>
              View Styluses
            </button>
          </div>
        </div>

        <div class={styles.card}>
          <div class={styles.cardHeader}>
            <h2>Sync Collection</h2>
          </div>
          <div class={styles.cardBody}>
            <p>Sync your Waugzee collection with your Discogs library.</p>
          </div>
          <div class={styles.cardFooter}>
            <button class={styles.button} onClick={handleSyncCollection}>
              Sync Now
            </button>
          </div>
        </div>

        <div class={styles.card}>
          <div class={styles.cardHeader}>
            <h2>View Analytics</h2>
          </div>
          <div class={styles.cardBody}>
            <p>Explore insights about your collection and listening habits.</p>
          </div>
          <div class={styles.cardFooter}>
            <button class={styles.button} onClick={handleViewAnalytics}>
              View Insights
            </button>
          </div>
        </div>
      </div>
    </div>
  );
};

export default Home;
