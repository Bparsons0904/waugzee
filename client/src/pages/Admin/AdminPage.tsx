import { MonthlyDownloadsSection } from "@components/admin/MonthlyDownloadsSection";
import type { Component } from "solid-js";
import styles from "./AdminPage.module.scss";

const AdminPage: Component = () => {
  return (
    <div class={styles.adminPage}>
      <header class={styles.header}>
        <h1>Admin Dashboard</h1>
        <p class={styles.subtitle}>System administration and management</p>
      </header>

      <div class={styles.container}>
        <MonthlyDownloadsSection />

        <section class={styles.section}>
          <h2>Background Jobs</h2>
          <p>TODO: Implement BackgroundJobsSection</p>
        </section>

        <section class={styles.section}>
          <h2>User Management</h2>
          <p>TODO: Implement UserManagementSection</p>
        </section>

        <section class={styles.section}>
          <h2>Cache Management</h2>
          <p>TODO: Implement CacheManagementSection</p>
        </section>
      </div>
    </div>
  );
};

export default AdminPage;
