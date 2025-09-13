import { Component } from "solid-js";
import { useNavigate } from "@solidjs/router";
import styles from "./Home.module.scss";

const Home: Component = () => {
  const navigate = useNavigate();

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

  return (
    <div class={styles.container}>
      <h1 class={styles.title}>Welcome to Waugzee</h1>
      <p class={styles.subtitle}>Your personal vinyl collection tracker</p>

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
