import { Field } from "@components/common/forms/Field/Field";
import { Select, type SelectOption } from "@components/common/forms/Select/Select";
import { Toggle } from "@components/common/forms/Toggle/Toggle";
import { Image } from "@components/common/ui/Image/Image";
import RecordActionModal from "@components/RecordActionModal/RecordActionModal";
import { RecordStatusIndicator } from "@components/StatusIndicators/StatusIndicators";
import { useUserData } from "@context/UserDataContext";
import type { UserRelease } from "@models/User";
import { useLogCleaning, useLogPlay } from "@services/apiHooks";
import { fuzzySearchUserReleases } from "@utils/fuzzy";

import { type Component, createMemo, createSignal, For, Show } from "solid-js";
import styles from "./LogPlay.module.scss";

const sortOptions: SelectOption[] = [
  { value: "album", label: "Album (A-Z)" },
  { value: "artist", label: "Artist (A-Z)" },
  { value: "genre", label: "Genre (A-Z)" },
  { value: "lastPlayed", label: "Last Played (newest first)" },
  { value: "longestUnplayed", label: "Longest Unplayed (oldest first)" },
  { value: "needsCleaning", label: "Needs Cleaning (most dirty first)" },
  { value: "recentlyPlayed", label: "Recently Played (30 days)" },
  { value: "year", label: "Release Year (newest first)" },
  { value: "playCount", label: "Most Played" },
];

const LogPlay: Component = () => {
  const userData = useUserData();
  const releases = () => userData.releases();
  const [searchTerm, setSearchTerm] = createSignal("");
  const [selectedRelease, setSelectedRelease] = createSignal<UserRelease | null>(null);
  const [isModalOpen, setIsModalOpen] = createSignal(false);
  const [showStatusDetails, setShowStatusDetails] = createSignal(false);
  const [sortBy, setSortBy] = createSignal("artist");

  const logPlayMutation = useLogPlay({
    invalidateQueries: [["user"]],
    successMessage: "Play logged successfully!",
  });

  const logCleaningMutation = useLogCleaning({
    invalidateQueries: [["user"]],
    successMessage: "Cleaning logged successfully!",
  });

  const sortReleases = (releases: UserRelease[], sortOption: string): UserRelease[] => {
    const sorted = [...releases];

    switch (sortOption) {
      case "album":
        return sorted.sort((a, b) => (a.release.title || "").localeCompare(b.release.title || ""));

      case "year":
        return sorted.sort((a, b) => {
          const yearA = a.release.year || 0;
          const yearB = b.release.year || 0;
          return yearB - yearA;
        });

      // case "artist":
      // case "genre":
      // case "lastPlayed":
      // case "longestUnplayed":
      // case "recentlyPlayed":
      // case "playCount":
      // case "needsCleaning":
      default: {
        return sorted.sort((a, b) => (a.release.title || "").localeCompare(b.release.title || ""));
      }
    }
  };

  const filteredReleases = createMemo(() => {
    let filtered = releases();

    if (searchTerm()) {
      filtered = fuzzySearchUserReleases(filtered, searchTerm());
    }

    return sortReleases(filtered, sortBy());
  });

  const handleReleaseClick = (release: UserRelease) => {
    setSelectedRelease(release);
    setIsModalOpen(true);
  };

  const handleCloseModal = () => {
    setIsModalOpen(false);
  };

  const handleQuickPlay = (release: UserRelease) => {
    const primaryStylus = userData.styluses().find((s) => s.isPrimary && s.isActive);
    logPlayMutation.mutate({
      releaseId: release.releaseId,
      playedAt: new Date().toISOString(),
      userStylusId: primaryStylus?.id,
    });
  };

  const handleQuickCleaning = (release: UserRelease) => {
    logCleaningMutation.mutate({
      releaseId: release.releaseId,
      cleanedAt: new Date().toISOString(),
      isDeepClean: false,
    });
  };

  return (
    <div class={styles.container}>
      <h1 class={styles.title}>Log Play & Cleaning</h1>
      <p class={styles.subtitle}>Record when you play or clean records from your collection.</p>

      <div class={styles.logForm}>
        <div class={styles.controlsRow}>
          <div class={styles.searchSection}>
            <Field label="Search Your Collection" htmlFor="releaseSearch">
              <input
                type="text"
                id="releaseSearch"
                class={styles.searchInput}
                value={searchTerm()}
                onInput={(e) => setSearchTerm(e.target.value)}
                placeholder="Search by title or artist..."
              />
            </Field>
          </div>

          <div class={styles.sortSection}>
            <Field label="Sort By" htmlFor="sortOptions">
              <Select
                name="sortOptions"
                options={sortOptions}
                value={sortBy()}
                onChange={setSortBy}
              />
            </Field>
          </div>
        </div>

        <div class={styles.optionsSection}>
          <Field label="Show status details">
            <Toggle checked={showStatusDetails()} onChange={setShowStatusDetails} />
          </Field>
        </div>

        <h2 class={styles.sectionTitle}>Your Collection</h2>

        <div class={styles.releasesSection}>
          <Show
            when={filteredReleases().length > 0}
            fallback={
              <p class={styles.noResults}>
                No releases found. Try a different search term or sort option.
              </p>
            }
          >
            <div class={styles.releasesList}>
              <For each={filteredReleases()}>
                {(userRelease) => (
                  <div
                    class={`${styles.releaseCard} ${selectedRelease()?.id === userRelease.id ? styles.selected : ""}`}
                    onClick={() => handleReleaseClick(userRelease)}
                  >
                    <div class={styles.releaseCardContainer}>
                      <div class={styles.releaseImageContainer}>
                        <Image
                          src={userRelease.release.thumb || userRelease.release.coverImage || ""}
                          alt={userRelease.release.title || "Release"}
                          aspectRatio="square"
                          showSkeleton={false}
                          loading="lazy"
                          className={styles.releaseImage}
                        />
                        {userRelease.release.year && (
                          <div class={styles.releaseYear}>{userRelease.release.year}</div>
                        )}
                      </div>
                      <div class={styles.releaseInfo}>
                        <h3 class={styles.releaseTitle}>{userRelease.release.title}</h3>
                        <p class={styles.releaseArtist}>
                          {userRelease.release.artists?.[0]?.name || "Unknown Artist"}
                        </p>

                        <div class={styles.statusSection}>
                          <RecordStatusIndicator
                            playHistory={userRelease.playHistory || []}
                            cleaningHistory={userRelease.cleaningHistory || []}
                            showDetails={false}
                            onPlayClick={() => handleQuickPlay(userRelease)}
                            onCleanClick={() => handleQuickCleaning(userRelease)}
                          />
                        </div>
                      </div>
                    </div>

                    <Show when={showStatusDetails()}>
                      <div class={styles.fullWidthDetails}>
                        <RecordStatusIndicator
                          playHistory={userRelease.playHistory || []}
                          cleaningHistory={userRelease.cleaningHistory || []}
                          showDetails={true}
                        />
                      </div>
                    </Show>
                  </div>
                )}
              </For>
            </div>
          </Show>
        </div>
      </div>

      {/* Claude shouldn't this be a modal? If the modal is in the RecordActionModal component, then it should be moved here   */}
      <Show when={selectedRelease()}>
        <RecordActionModal
          isOpen={isModalOpen()}
          onClose={handleCloseModal}
          release={selectedRelease() as never}
        />
      </Show>
    </div>
  );
};

export default LogPlay;
