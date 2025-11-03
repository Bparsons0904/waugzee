import { Button } from "@components/common/ui/Button/Button";
import { DiscogsTokenModal } from "@components/common/ui/DiscogsTokenModal/DiscogsTokenModal";
import { FolderSelector } from "@components/folders/FolderSelector";
import { USER_ENDPOINTS } from "@constants/api.constants";
import { useAuth } from "@context/AuthContext";
import { useUserData } from "@context/UserDataContext";
import { useApiPut } from "@services/apiHooks";
import clsx from "clsx";
import { type Component, createSignal, Show } from "solid-js";
import type { UpdateUserPreferencesRequest, UpdateUserPreferencesResponse } from "src/types/User";
import styles from "./Profile.module.scss";

const Profile: Component = () => {
  const { logout } = useAuth();
  const { user } = useUserData();

  const [isDiscogsModalOpen, setIsDiscogsModalOpen] = createSignal(false);

  const [recentlyPlayedThreshold, setRecentlyPlayedThreshold] = createSignal<number>(
    user()?.configuration?.recentlyPlayedThresholdDays ?? 90,
  );
  const [cleaningFrequency, setCleaningFrequency] = createSignal<number>(
    user()?.configuration?.cleaningFrequencyPlays ?? 5,
  );
  const [neglectedThreshold, setNeglectedThreshold] = createSignal<number>(
    user()?.configuration?.neglectedRecordsThresholdDays ?? 180,
  );

  const updatePreferencesMutation = useApiPut<
    UpdateUserPreferencesResponse,
    UpdateUserPreferencesRequest
  >(USER_ENDPOINTS.ME_PREFERENCES, undefined, {
    invalidateQueries: [["user"]],
    successMessage: "Preferences updated successfully!",
    errorMessage: "Failed to update preferences. Please try again.",
  });

  const handleSavePreferences = (e: Event) => {
    e.preventDefault();

    const preferences: UpdateUserPreferencesRequest = {
      recentlyPlayedThresholdDays: recentlyPlayedThreshold(),
      cleaningFrequencyPlays: cleaningFrequency(),
      neglectedRecordsThresholdDays: neglectedThreshold(),
    };

    updatePreferencesMutation.mutate(preferences);
  };

  const formatDate = (dateString: string | undefined) => {
    if (!dateString) return "Not available";
    return new Date(dateString).toLocaleDateString("en-US", {
      year: "numeric",
      month: "long",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  };

  return (
    <div class={styles.container}>
      <div class={styles.header}>
        <h1 class={styles.title}>Profile</h1>
        <p class={styles.subtitle}>Manage your account settings and preferences</p>
      </div>

      <Show when={user()} fallback={<div class={styles.loading}>Loading profile...</div>}>
        <div class={styles.section}>
          <div class={styles.sectionHeader}>
            <h2 class={styles.sectionTitle}>Account Information</h2>
          </div>
          <div class={styles.infoGrid}>
            <div class={styles.infoItem}>
              <span class={styles.infoLabel}>Username</span>
              <span class={styles.infoValue}>{user()?.displayName || "Not available"}</span>
            </div>
            <div class={styles.infoItem}>
              <span class={styles.infoLabel}>Full Name</span>
              <span class={styles.infoValue}>
                {user()?.firstName} {user()?.lastName}
              </span>
            </div>
            <div class={styles.infoItem}>
              <span class={styles.infoLabel}>Email</span>
              <span class={styles.infoValue}>{user()?.email || "Not available"}</span>
            </div>
            <div class={styles.infoItem}>
              <span class={styles.infoLabel}>Last Login</span>
              <span class={styles.infoValue}>{formatDate(user()?.lastLoginAt)}</span>
            </div>
          </div>
        </div>

        <div class={styles.section}>
          <div class={styles.sectionHeader}>
            <h2 class={styles.sectionTitle}>Discogs Integration</h2>
          </div>
          <div class={styles.discogsInfo}>
            <div class={styles.statusInfo}>
              <span class={styles.infoLabel}>Connection Status</span>
              <span
                class={clsx(styles.statusBadge, {
                  [styles.statusConnected]: user()?.configuration?.discogsToken,
                  [styles.statusDisconnected]: !user()?.configuration?.discogsToken,
                })}
              >
                {user()?.configuration?.discogsToken
                  ? `✓ Connected as ${user()?.configuration?.discogsUsername}`
                  : "✗ Not Connected"}
              </span>
            </div>
            <Button variant="secondary" onClick={() => setIsDiscogsModalOpen(true)}>
              {user()?.configuration?.discogsToken ? "Update Discogs Token" : "Connect Discogs"}
            </Button>
          </div>
        </div>

        <div class={styles.section}>
          <div class={styles.sectionHeader}>
            <h2 class={styles.sectionTitle}>Folder Selection</h2>
          </div>
          <FolderSelector />
        </div>

        <div class={styles.section}>
          <div class={styles.sectionHeader}>
            <h2 class={styles.sectionTitle}>User Preferences</h2>
          </div>
          <form onSubmit={handleSavePreferences} class={styles.preferencesForm}>
            <div class={styles.preferencesGrid}>
              <div class={styles.formGroup}>
                <label for="recentlyPlayed" class={styles.label}>
                  Recently Played Threshold
                </label>
                <input
                  type="number"
                  id="recentlyPlayed"
                  class={styles.input}
                  min="1"
                  max="365"
                  value={recentlyPlayedThreshold()}
                  onInput={(e) => setRecentlyPlayedThreshold(Number.parseInt(e.target.value, 10))}
                />
                <p class={styles.helpText}>
                  Records played within this many days are considered "recently played"
                </p>
              </div>

              <div class={styles.formGroup}>
                <label for="cleaningFrequency" class={styles.label}>
                  Cleaning Frequency
                </label>
                <input
                  type="number"
                  id="cleaningFrequency"
                  class={styles.input}
                  min="1"
                  max="50"
                  value={cleaningFrequency()}
                  onInput={(e) => setCleaningFrequency(Number.parseInt(e.target.value, 10))}
                />
                <p class={styles.helpText}>
                  Number of plays before a record needs cleaning (affects play status indicators)
                </p>
              </div>

              <div class={styles.formGroup}>
                <label for="neglectedThreshold" class={styles.label}>
                  Neglected Records Threshold
                </label>
                <input
                  type="number"
                  id="neglectedThreshold"
                  class={styles.input}
                  min="1"
                  max="730"
                  value={neglectedThreshold()}
                  onInput={(e) => setNeglectedThreshold(Number.parseInt(e.target.value, 10))}
                />
                <p class={styles.helpText}>
                  Records not played within this many days are considered "neglected" in analytics
                </p>
              </div>

              <div class={styles.formActions}>
                <Button
                  type="button"
                  variant="secondary"
                  onClick={() => {
                    setRecentlyPlayedThreshold(
                      user()?.configuration?.recentlyPlayedThresholdDays ?? 90,
                    );
                    setCleaningFrequency(user()?.configuration?.cleaningFrequencyPlays ?? 5);
                    setNeglectedThreshold(
                      user()?.configuration?.neglectedRecordsThresholdDays ?? 180,
                    );
                  }}
                >
                  Reset
                </Button>
                <Button
                  type="submit"
                  variant="primary"
                  disabled={updatePreferencesMutation.isPending}
                >
                  {updatePreferencesMutation.isPending ? "Saving..." : "Save Preferences"}
                </Button>
              </div>
            </div>
          </form>
        </div>

        <div class={styles.section}>
          <div class={styles.sectionHeader}>
            <h2 class={styles.sectionTitle}>Data Management</h2>
          </div>
          <div class={styles.dataManagementActions}>
            <div class={styles.dataAction}>
              <div class={styles.buttonWrapper}>
                <Button variant="secondary" disabled>
                  Import Data
                </Button>
                <span class={styles.comingSoon}>Coming Soon</span>
              </div>
              <p class={styles.helpText}>
                Import play history and cleaning records from CSV or JSON files
              </p>
            </div>
            <div class={styles.dataAction}>
              <div class={styles.buttonWrapper}>
                <Button variant="secondary" disabled>
                  Export Data
                </Button>
                <span class={styles.comingSoon}>Coming Soon</span>
              </div>
              <p class={styles.helpText}>
                Export your play history, cleaning records, and collection data
              </p>
            </div>
          </div>
        </div>

        <div class={styles.section}>
          <Button variant="danger" onClick={logout} class={styles.signOutButton}>
            Sign Out
          </Button>
        </div>
      </Show>

      <Show when={isDiscogsModalOpen()}>
        <DiscogsTokenModal onClose={() => setIsDiscogsModalOpen(false)} />
      </Show>
    </div>
  );
};

export default Profile;
