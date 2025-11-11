import { SearchInput } from "@components/common/forms/SearchInput/SearchInput";
import { Select } from "@components/common/forms/Select/Select";
import { Button } from "@components/common/ui/Button/Button";
import FilterIcon from "@components/icons/FilterIcon";
import GridIcon from "@components/icons/GridIcon";
import RecordActionModal from "@components/RecordActionModal/RecordActionModal";
import { useUserData } from "@context/UserDataContext";
import { fuzzySearchUserReleases } from "@utils/fuzzy";
import clsx from "clsx";
import { type Component, createMemo, For, Show } from "solid-js";
import { createStore } from "solid-js/store";
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

          <Select
            name="sortBy"
            options={[
              { value: "album", label: "Album (A-Z)" },
              { value: "artist", label: "Artist (A-Z)" },
              { value: "year", label: "Year (newest first)" },
              { value: "recentlyAdded", label: "Recently Added" },
            ]}
            value={props.sortBy}
            onChange={props.onSortChange}
            class={styles.sortSelect}
          />

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

  const [viewState, setViewState] = createStore({
    searchTerm: "",
    sortBy: "artist",
    gridSize: "medium" as "small" | "medium" | "large",
    showFilters: false,
    genreFilter: [] as string[],
    selectedReleaseId: null as string | null,
    isModalOpen: false,
  });

  const selectedRelease = createMemo(() => {
    if (!viewState.selectedReleaseId) return null;
    return releases().find((r) => r.id === viewState.selectedReleaseId) || null;
  });

  const availableGenres = createMemo(() => {
    const genreSet = new Set<string>();
    releases().forEach((userRelease) => {
      userRelease.release.genres?.forEach((genre) => {
        if (genre.type !== "genre") return;
        genreSet.add(genre.name);
      });
    });
    return Array.from(genreSet).sort();
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

  const filteredReleases = createMemo(() => {
    let filtered = releases();

    if (viewState.genreFilter.length > 0) {
      filtered = filtered.filter((userRelease) =>
        userRelease.release.genres?.some((genre) => viewState.genreFilter.includes(genre.name)),
      );
    }

    if (viewState.searchTerm) {
      filtered = fuzzySearchUserReleases(filtered, viewState.searchTerm);
    }

    return sortReleases(filtered, viewState.sortBy);
  });

  const toggleGenre = (genre: string) => {
    if (viewState.genreFilter.includes(genre)) {
      setViewState("genreFilter", (prev) => prev.filter((g) => g !== genre));
    } else {
      setViewState("genreFilter", (prev) => [...prev, genre]);
    }
  };

  const clearFilters = () => {
    setViewState({
      genreFilter: [],
      searchTerm: "",
    });
  };

  const handleReleaseClick = (release: UserRelease) => {
    setViewState({
      selectedReleaseId: release.id,
      isModalOpen: true,
    });
  };

  const handleCloseModal = () => {
    setViewState("isModalOpen", false);
  };

  return (
    <div class={styles.container}>
      <h1 class={styles.title}>Your Collection</h1>

      <CollectionControls
        searchTerm={viewState.searchTerm}
        onSearchChange={(value) => setViewState("searchTerm", value)}
        sortBy={viewState.sortBy}
        onSortChange={(value) => setViewState("sortBy", value)}
        gridSize={viewState.gridSize}
        onGridSizeChange={(size) => setViewState("gridSize", size)}
        showFilters={viewState.showFilters}
        onToggleFilters={() => setViewState("showFilters", !viewState.showFilters)}
        availableGenres={availableGenres()}
        selectedGenres={viewState.genreFilter}
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

      <div class={clsx(styles.albumGrid, styles[viewState.gridSize])}>
        <For each={filteredReleases()}>
          {(userRelease) => <AlbumCard userRelease={userRelease} onClick={handleReleaseClick} />}
        </For>
      </div>

      <Show when={selectedRelease()}>
        <RecordActionModal
          isOpen={viewState.isModalOpen}
          onClose={handleCloseModal}
          release={selectedRelease() as never}
        />
      </Show>
    </div>
  );
};

export default ViewCollection;
