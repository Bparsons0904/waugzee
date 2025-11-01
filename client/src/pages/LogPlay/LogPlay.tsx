import { Field } from "@components/common/forms/Field/Field";
import { SearchInput } from "@components/common/forms/SearchInput/SearchInput";
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

const LogPlay: Component = () => {
  const userData = useUserData();
  const releases = () => userData.releases();

  const recentlyPlayedThreshold = () =>
    userData.user()?.configuration?.recentlyPlayedThresholdDays ?? 90;

  const sortOptions = createMemo((): SelectOption[] => [
    { value: "album", label: "Album (A-Z)" },
    { value: "artist", label: "Artist (A-Z)" },
    { value: "genre", label: "Genre (A-Z)" },
    { value: "lastPlayed", label: "Last Played (newest first)" },
    { value: "longestUnplayed", label: "Longest Unplayed (oldest first)" },
    { value: "needsCleaning", label: "Needs Cleaning (most dirty first)" },
    { value: "recentlyPlayed", label: `Recently Played (${recentlyPlayedThreshold()} days)` },
    { value: "year", label: "Release Year (newest first)" },
    { value: "playCount", label: "Most Played" },
  ]);
  const [searchTerm, setSearchTerm] = createSignal("");
  const [selectedReleaseId, setSelectedReleaseId] = createSignal<string | null>(null);
  const [isModalOpen, setIsModalOpen] = createSignal(false);
  const [showStatusDetails, setShowStatusDetails] = createSignal(false);
  const [sortBy, setSortBy] = createSignal("artist");

  const selectedRelease = createMemo(() => {
    const releaseId = selectedReleaseId();
    if (!releaseId) return null;
    return releases().find((r) => r.id === releaseId) || null;
  });

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
    const now = Date.now();
    const threshold = recentlyPlayedThreshold();
    const thresholdAgo = now - threshold * 24 * 60 * 60 * 1000;

    switch (sortOption) {
      case "album":
        return sorted.sort((a, b) => (a.release.title || "").localeCompare(b.release.title || ""));

      case "artist":
        return sorted.sort((a, b) => {
          const artistA = a.release.artists?.[0]?.name || "Unknown Artist";
          const artistB = b.release.artists?.[0]?.name || "Unknown Artist";
          return artistA.localeCompare(artistB);
        });

      case "genre":
        return sorted.sort((a, b) => {
          const genreA = a.release.genres?.[0]?.name || "";
          const genreB = b.release.genres?.[0]?.name || "";
          return genreA.localeCompare(genreB);
        });

      case "year":
        return sorted.sort((a, b) => {
          const yearA = a.release.year || 0;
          const yearB = b.release.year || 0;
          return yearB - yearA;
        });

      case "lastPlayed":
        return sorted.sort((a, b) => {
          const lastPlayedA = a.playHistory?.[0]?.playedAt
            ? new Date(a.playHistory[0].playedAt).getTime()
            : 0;
          const lastPlayedB = b.playHistory?.[0]?.playedAt
            ? new Date(b.playHistory[0].playedAt).getTime()
            : 0;
          return lastPlayedB - lastPlayedA;
        });

      case "longestUnplayed":
        return sorted.sort((a, b) => {
          const lastPlayedA = a.playHistory?.[0]?.playedAt
            ? new Date(a.playHistory[0].playedAt).getTime()
            : 0;
          const lastPlayedB = b.playHistory?.[0]?.playedAt
            ? new Date(b.playHistory[0].playedAt).getTime()
            : 0;
          return lastPlayedA - lastPlayedB;
        });

      case "recentlyPlayed":
        return sorted
          .filter((release) => {
            const lastPlayed = release.playHistory?.[0]?.playedAt
              ? new Date(release.playHistory[0].playedAt).getTime()
              : 0;
            return lastPlayed >= thresholdAgo;
          })
          .sort((a, b) => {
            const lastPlayedA = new Date(a.playHistory?.[0]?.playedAt || 0).getTime();
            const lastPlayedB = new Date(b.playHistory?.[0]?.playedAt || 0).getTime();
            return lastPlayedB - lastPlayedA;
          });

      case "playCount":
        return sorted.sort((a, b) => {
          const countA = a.playHistory?.length || 0;
          const countB = b.playHistory?.length || 0;
          return countB - countA;
        });

      case "needsCleaning":
        return sorted.sort((a, b) => {
          const playsA = a.playHistory?.length || 0;
          const cleansA = a.cleaningHistory?.length || 0;
          const playsB = b.playHistory?.length || 0;
          const cleansB = b.cleaningHistory?.length || 0;

          const ratioA = cleansA > 0 ? playsA / cleansA : playsA;
          const ratioB = cleansB > 0 ? playsB / cleansB : playsB;

          return ratioB - ratioA;
        });

      default:
        return sorted.sort((a, b) => (a.release.title || "").localeCompare(b.release.title || ""));
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
    setSelectedReleaseId(release.id);
    setIsModalOpen(true);
  };

  const handleCloseModal = () => {
    setIsModalOpen(false);
  };

  const handleQuickPlay = (release: UserRelease) => {
    const primaryStylus = userData.styluses().find((s) => s.isPrimary && s.isActive);
    logPlayMutation.mutate({
      userReleaseId: release.id,
      playedAt: new Date().toISOString(),
      userStylusId: primaryStylus?.id,
    });
  };

  const handleQuickCleaning = (release: UserRelease) => {
    logCleaningMutation.mutate({
      userReleaseId: release.id,
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
            <label class={styles.label} for="releaseSearch">
              Search Your Collection
            </label>
            <SearchInput
              id="releaseSearch"
              value={searchTerm()}
              onInput={setSearchTerm}
              placeholder="Search by title or artist..."
            />
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
