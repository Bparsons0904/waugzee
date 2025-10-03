import { Component, createSignal, createMemo, For, Show } from "solid-js";
import styles from "./LogPlay.module.scss";
import RecordActionModal from "@components/RecordActionModal/RecordActionModal";
import { RecordStatusIndicator } from "@components/StatusIndicators/StatusIndicators";
import { Select, SelectOption } from "@components/common/forms/Select/Select";
import { Field } from "@components/common/forms/Field/Field";
import { Image } from "@components/common/ui/Image/Image";
import { Toggle } from "@components/common/forms/Toggle/Toggle";
import {} from // getLastPlayDate,
// getCleanlinessScore,
// countPlaysSinceCleaning,
// getLastCleaningDate,
"@utils/playStatus";
// import { fuzzySearchReleases } from "@utils/fuzzy";
import { useUserData } from "@context/UserDataContext";
import { UserRelease } from "@models/User";

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
  console.log("releases", releases());
  const [searchTerm, setSearchTerm] = createSignal("");
  const [selectedRelease, setSelectedRelease] =
    createSignal<UserRelease | null>(null);
  const [isModalOpen, setIsModalOpen] = createSignal(false);
  const [showStatusDetails, setShowStatusDetails] = createSignal(false);
  const [sortBy, setSortBy] = createSignal("artist");

  const sortReleases = (
    releases: UserRelease[],
    sortOption: string,
  ): UserRelease[] => {
    const sorted = [...releases];

    switch (sortOption) {
      case "album":
        return sorted.sort((a, b) =>
          (a.release.title || "").localeCompare(b.release.title || ""),
        );

      case "year":
        return sorted.sort((a, b) => {
          const yearA = a.release.year || 0;
          const yearB = b.release.year || 0;
          return yearB - yearA;
        });

      case "artist":
      case "genre":
      case "lastPlayed":
      case "longestUnplayed":
      case "recentlyPlayed":
      case "playCount":
      case "needsCleaning":
      default:
        return sorted.sort((a, b) =>
          (a.release.title || "").localeCompare(b.release.title || ""),
        );
    }
  };

  const filteredReleases = createMemo(() => {
    let filtered = releases();

    if (searchTerm()) {
      filtered = filtered.filter(
        (r) =>
          r.release.title?.toLowerCase().includes(searchTerm().toLowerCase()) ||
          false,
      );
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
    console.log("Quick Play:", {
      releaseId: release.releaseId,
      releaseTitle: release.release.title,
    });
  };

  const handleQuickCleaning = (release: UserRelease) => {
    console.log("Quick Cleaning:", {
      releaseId: release.releaseId,
      releaseTitle: release.release.title,
    });
  };

  return (
    <div class={styles.container}>
      <h1 class={styles.title}>Log Play & Cleaning</h1>
      <p class={styles.subtitle}>
        Record when you play or clean records from your collection.
      </p>

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
            <Toggle
              checked={showStatusDetails()}
              onChange={setShowStatusDetails}
            />
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
                          src={
                            userRelease.release.thumb ||
                            userRelease.release.coverImage ||
                            ""
                          }
                          alt={userRelease.release.title || "Release"}
                          aspectRatio="square"
                          showSkeleton={false}
                          loading="lazy"
                          className={styles.releaseImage}
                        />
                        {userRelease.release.year && (
                          <div class={styles.releaseYear}>
                            {userRelease.release.year}
                          </div>
                        )}
                      </div>
                      <div class={styles.releaseInfo}>
                        <h3 class={styles.releaseTitle}>
                          {userRelease.release.title}
                        </h3>
                        <p class={styles.releaseArtist}>
                          {userRelease.release.format || "Unknown Format"}
                        </p>

                        <div class={styles.statusSection}>
                          <RecordStatusIndicator
                            playHistory={[]}
                            cleaningHistory={[]}
                            showDetails={false}
                            onPlayClick={() => handleQuickPlay(userRelease)}
                            onCleanClick={() =>
                              handleQuickCleaning(userRelease)
                            }
                          />
                        </div>
                      </div>
                    </div>

                    <Show when={showStatusDetails()}>
                      <div class={styles.fullWidthDetails}>
                        <RecordStatusIndicator
                          playHistory={[]}
                          cleaningHistory={[]}
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
          release={selectedRelease()!}
        />
      </Show>
    </div>
  );
};

export default LogPlay;
