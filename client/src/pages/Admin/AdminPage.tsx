import { FileManagementSection } from "@components/admin/FileManagementSection";
// TODO: REMOVE_AFTER_MIGRATION - KleioImportSection is for one-time data import
import { KleioImportSection } from "@components/admin/KleioImportSection";
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
        {/* TODO: REMOVE_AFTER_MIGRATION - KleioImportSection is for one-time data import */}
        <KleioImportSection />

        <MonthlyDownloadsSection />

        <FileManagementSection />

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
