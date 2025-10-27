import { SearchInput } from "@components/common/forms/SearchInput/SearchInput";
import { Button } from "@components/common/ui/Button/Button";
import FilterIcon from "@components/icons/FilterIcon";
import GridIcon from "@components/icons/GridIcon";
import RecordActionModal from "@components/RecordActionModal/RecordActionModal";
import { useUserData } from "@context/UserDataContext";
import { fuzzySearchUserReleases } from "@utils/fuzzy";
import { type Component, createEffect, createMemo, createSignal, For, Show } from "solid-js";
import type { UserRelease } from "src/types/User";
import styles from "./ViewCollection.module.scss";

interface CollectionControlsProps {
  searchTerm: string;
  onSearchChange: (value: string) => void;
  sortBy: string;
  onSortChange: (value: string) => void;
  gridSize: "small" | "medium" | "large";
  onGridSizeChange: (size: "small" | "medium" | "large") => void;
  showFilters: boolean;
  onToggleFilters: () => void;
  availableGenres: string[];
  selectedGenres: string[];
  onGenreToggle: (genre: string) => void;
  onClearFilters: () => void;
}

const CollectionControls: Component<CollectionControlsProps> = (props) => {
  return (
    <>
      <div class={styles.controls}>
        <SearchInput
          value={props.searchTerm}
          onInput={props.onSearchChange}
          placeholder="Search by album or artist..."
          class={styles.searchBar}
        />

        <div class={styles.controlButtons}>
          <button
            class={styles.filterButton}
            onClick={props.onToggleFilters}
            classList={{ [styles.active]: props.showFilters }}
            type="button"
          >
            <FilterIcon size={20} />
            <span>Filter</span>
          </button>

          <div class={styles.sortContainer}>
            <select
              value={props.sortBy}
              onInput={(e) => props.onSortChange(e.currentTarget.value)}
              class={styles.sortSelect}
            >
              <option value="album">Album (A-Z)</option>
              <option value="artist">Artist (A-Z)</option>
              <option value="year">Year (newest first)</option>
              <option value="recentlyAdded">Recently Added</option>
            </select>
          </div>

          <div class={styles.gridSizeToggle}>
            <button
              onClick={() => props.onGridSizeChange("small")}
              classList={{ [styles.active]: props.gridSize === "small" }}
              title="Small grid"
              type="button"
            >
              <GridIcon size={18} variant="small" />
            </button>
            <button
              onClick={() => props.onGridSizeChange("medium")}
              classList={{ [styles.active]: props.gridSize === "medium" }}
              title="Medium grid"
              type="button"
            >
              <GridIcon size={22} variant="medium" />
            </button>
            <button
              onClick={() => props.onGridSizeChange("large")}
              classList={{ [styles.active]: props.gridSize === "large" }}
              title="Large grid"
              type="button"
            >
              <GridIcon size={22} variant="large" />
            </button>
          </div>
        </div>
      </div>

      <Show when={props.showFilters}>
        <div class={styles.filterPanel}>
          <div class={styles.filterSection}>
            <h3 class={styles.filterTitle}>Genres</h3>
            <div class={styles.filterOptions}>
              <For each={props.availableGenres}>
                {(genre) => (
                  <label class={styles.filterOption}>
                    <input
                      type="checkbox"
                      checked={props.selectedGenres.includes(genre)}
                      onChange={() => props.onGenreToggle(genre)}
                    />
                    <span>{genre}</span>
                  </label>
                )}
              </For>
            </div>
          </div>

          <div class={styles.filterActions}>
            <Button variant="secondary" size="sm" onClick={props.onClearFilters}>
              Clear All Filters
            </Button>
          </div>
        </div>
      </Show>
    </>
  );
};

interface AlbumCardProps {
  userRelease: UserRelease;
  onClick: (userRelease: UserRelease) => void;
}

const AlbumCard: Component<AlbumCardProps> = (props) => {
  return (
    <div class={styles.albumCard} onClick={() => props.onClick(props.userRelease)}>
      <div class={styles.albumArtwork}>
        <Show
          when={props.userRelease.release.coverImage}
          fallback={
            <div class={styles.noImage}>
              <span>{props.userRelease.release.title}</span>
            </div>
          }
        >
          <img
            src={props.userRelease.release.coverImage}
            alt={props.userRelease.release.title}
            class={styles.albumImage}
          />
        </Show>

        <div class={styles.albumHover}>
          <div class={styles.trackList}>
            <h4 class={styles.trackListTitle}>Tracks</h4>
            <ol class={styles.tracks}>
              <Show
                when={
                  props.userRelease.release.tracksJson &&
                  props.userRelease.release.tracksJson.length > 0
                }
                fallback={<li class={styles.noTracks}>No track data available</li>}
              >
                <For each={props.userRelease.release.tracksJson}>
                  {(track) => (
                    <li>
                      <span class={styles.trackPosition}>{track.position}</span>
                      {track.title}
                      <Show when={track.duration}>
                        <span class={styles.trackDuration}>{track.duration}</span>
                      </Show>
                    </li>
                  )}
                </For>
              </Show>
            </ol>
          </div>
        </div>
      </div>

      <div class={styles.albumInfo}>
        <h3 class={styles.albumTitle}>{props.userRelease.release.title}</h3>
        <p class={styles.albumArtist}>
          {props.userRelease.release.artists?.map((artist) => artist.name).join(", ") ||
            "Unknown Artist"}
        </p>
      </div>
    </div>
  );
};

const ViewCollection: Component = () => {
  const { releases } = useUserData();

  const [filteredReleases, setFilteredReleases] = createSignal<UserRelease[]>([]);
  const [searchTerm, setSearchTerm] = createSignal("");
  const [sortBy, setSortBy] = createSignal("artist");
  const [gridSize, setGridSize] = createSignal<"small" | "medium" | "large">("medium");
  const [showFilters, setShowFilters] = createSignal(false);
  const [genreFilter, setGenreFilter] = createSignal<string[]>([]);
  const [selectedReleaseId, setSelectedReleaseId] = createSignal<string | null>(null);
  const [isModalOpen, setIsModalOpen] = createSignal(false);

  const selectedRelease = createMemo(() => {
    const releaseId = selectedReleaseId();
    if (!releaseId) return null;
    return releases().find((r) => r.id === releaseId) || null;
  });

  const availableGenres = () => {
    const genreSet = new Set<string>();
    releases().forEach((userRelease) => {
      userRelease.release.genres?.forEach((genre) => {
        if (genre.type !== "genre") return;
        genreSet.add(genre.name);
      });
    });
    return Array.from(genreSet).sort();
  };

  createEffect(() => {
    let filtered = releases();

    if (genreFilter().length > 0) {
      filtered = filtered.filter((userRelease) =>
        userRelease.release.genres?.some((genre) => genreFilter().includes(genre.name)),
      );
    }

    if (searchTerm()) {
      filtered = fuzzySearchUserReleases(filtered, searchTerm());
    }

    filtered = sortReleases(filtered, sortBy());

    setFilteredReleases(filtered);
  });

  const sortReleases = (releasesToSort: UserRelease[], sortOption: string): UserRelease[] => {
    switch (sortOption) {
      case "artist": {
        return [...releasesToSort].sort((a, b) => {
          const artistA = a.release.artists?.[0]?.name || "Unknown";
          const artistB = b.release.artists?.[0]?.name || "Unknown";
          return artistA.localeCompare(artistB);
        });
      }

      case "album": {
        return [...releasesToSort].sort((a, b) => a.release.title.localeCompare(b.release.title));
      }

      case "year": {
        return [...releasesToSort].sort((a, b) => {
          const yearA = a.release.year || 0;
          const yearB = b.release.year || 0;
          return yearB - yearA;
        });
      }

      case "recentlyAdded": {
        return [...releasesToSort].sort((a, b) => {
          const dateA = new Date(a.dateAdded).getTime();
          const dateB = new Date(b.dateAdded).getTime();
          return dateB - dateA;
        });
      }

      default: {
        return [...releasesToSort].sort((a, b) => a.release.title.localeCompare(b.release.title));
      }
    }
  };

  const toggleGenre = (genre: string) => {
    if (genreFilter().includes(genre)) {
      setGenreFilter((prev) => prev.filter((g) => g !== genre));
    } else {
      setGenreFilter((prev) => [...prev, genre]);
    }
  };

  const clearFilters = () => {
    setGenreFilter([]);
    setSearchTerm("");
  };

  const handleReleaseClick = (release: UserRelease) => {
    setSelectedReleaseId(release.id);
    setIsModalOpen(true);
  };

  const handleCloseModal = () => {
    setIsModalOpen(false);
  };

  return (
    <div class={styles.container}>
      <h1 class={styles.title}>Your Collection</h1>

      <CollectionControls
        searchTerm={searchTerm()}
        onSearchChange={setSearchTerm}
        sortBy={sortBy()}
        onSortChange={setSortBy}
        gridSize={gridSize()}
        onGridSizeChange={setGridSize}
        showFilters={showFilters()}
        onToggleFilters={() => setShowFilters(!showFilters())}
        availableGenres={availableGenres()}
        selectedGenres={genreFilter()}
        onGenreToggle={toggleGenre}
        onClearFilters={clearFilters}
      />

      <Show when={filteredReleases().length === 0}>
        <div class={styles.noResults}>
          <p>No albums match your search or filters.</p>
          <Button variant="secondary" size="sm" onClick={clearFilters}>
            Clear All Filters
          </Button>
        </div>
      </Show>

      <div class={`${styles.albumGrid} ${styles[gridSize()]}`}>
        <For each={filteredReleases()}>
          {(userRelease) => <AlbumCard userRelease={userRelease} onClick={handleReleaseClick} />}
        </For>
      </div>

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

export default ViewCollection;
