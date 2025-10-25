import { EditHistoryPanel } from "@components/common/ui/EditHistoryPanel/EditHistoryPanel";
import type { PlayHistory } from "@models/Release";
import { useUserData } from "@context/UserDataContext";
import { getLocalDateGroupKey, isSameLocalDay, useFormattedShortDate } from "@utils/dates";
import { fuzzySearchPlayHistory } from "@utils/fuzzy";
import { TbWashTemperature5 } from "solid-icons/tb";
import { VsNote } from "solid-icons/vs";
import { type Component, createMemo, createSignal, For, Show } from "solid-js";
import styles from "./PlayHistory.module.scss";

const PlayHistoryPage: Component = () => {
  const { playHistory, releases, styluses } = useUserData();
  const [timeFilter, setTimeFilter] = createSignal("month");
  const [searchTerm, setSearchTerm] = createSignal("");
  const [groupBy, setGroupBy] = createSignal("date");

  const [selectedPlay, setSelectedPlay] = createSignal<PlayHistory | null>(null);
  const [isEditPanelOpen, setIsEditPanelOpen] = createSignal(false);

  const hasCleaning = (userReleaseId: string, playDate: string) => {
    const release = releases().find((r) => r.id === userReleaseId);
    if (!release?.cleaningHistory || release.cleaningHistory.length === 0) {
      return false;
    }

    return release.cleaningHistory.some((cleaning) => {
      return isSameLocalDay(playDate, cleaning.cleanedAt);
    });
  };

  const getFilteredDate = () => {
    const now = new Date();
    switch (timeFilter()) {
      case "week": {
        const lastWeek = new Date();
        lastWeek.setDate(now.getDate() - 7);
        return lastWeek;
      }
      case "month": {
        const lastMonth = new Date();
        lastMonth.setMonth(now.getMonth() - 1);
        return lastMonth;
      }
      case "year": {
        const lastYear = new Date();
        lastYear.setFullYear(now.getFullYear() - 1);
        return lastYear;
      }
      default:
        return new Date(0);
    }
  };

  const filteredHistory = createMemo(() => {
    const filtered = [...playHistory()];

    const filterDate = getFilteredDate();
    const dateFiltered = filtered.filter((play) => new Date(play.playedAt) >= filterDate);

    if (searchTerm().trim()) {
      return fuzzySearchPlayHistory(dateFiltered, searchTerm());
    }

    return dateFiltered;
  });

  const groupedHistory = createMemo(() => {
    const history = filteredHistory();
    if (groupBy() === "none") return { "": history };

    const grouped = history.reduce(
      (acc, play) => {
        let key: string;
        if (groupBy() === "date") {
          key = getLocalDateGroupKey(play.playedAt);
        } else if (groupBy() === "artist") {
          const release = releases().find((r) => r.id === play.userReleaseId);
          key = release?.release?.artists?.[0]?.name || "Unknown Artist";
        } else if (groupBy() === "album") {
          const release = releases().find((r) => r.id === play.userReleaseId);
          key = release?.release?.title || "Unknown Album";
        } else {
          key = "";
        }

        if (!acc[key]) acc[key] = [];
        acc[key].push(play);
        return acc;
      },
      {} as Record<string, PlayHistory[]>,
    );

    return Object.fromEntries(
      Object.entries(grouped).sort(([, playsA], [, playsB]) => {
        const latestA = new Date(playsA[0].playedAt).getTime();
        const latestB = new Date(playsB[0].playedAt).getTime();
        return latestB - latestA;
      }),
    );
  });

  const handleItemClick = (play: PlayHistory) => {
    setSelectedPlay(play);
    setIsEditPanelOpen(true);
  };

  const handleCloseEditPanel = () => {
    setIsEditPanelOpen(false);
  };

  return (
    <div class={styles.container}>
      <h1 class={styles.title}>Play History</h1>

      <div class={styles.filters}>
        <div class={styles.filterGroup}>
          <label class={styles.label} for="time-filter">
            Time Period:
          </label>
          <select
            id="time-filter"
            class={styles.select}
            value={timeFilter()}
            onInput={(e) => setTimeFilter(e.currentTarget.value)}
          >
            <option value="all">All Time</option>
            <option value="week">Last Week</option>
            <option value="month">Last Month</option>
            <option value="year">Last Year</option>
          </select>
        </div>

        <div class={styles.filterGroup}>
          <label class={styles.label} for="group-by-filter">
            Group By:
          </label>
          <select
            id="group-by-filter"
            class={styles.select}
            value={groupBy()}
            onInput={(e) => setGroupBy(e.currentTarget.value)}
          >
            <option value="none">None</option>
            <option value="date">Date</option>
            <option value="artist">Artist</option>
            <option value="album">Album</option>
          </select>
        </div>

        <div class={styles.searchBox}>
          <input
            type="text"
            class={styles.searchInput}
            placeholder="Search by artist, album, stylus or notes..."
            value={searchTerm()}
            onInput={(e) => setSearchTerm(e.currentTarget.value)}
          />
        </div>
      </div>

      <Show when={filteredHistory().length === 0}>
        <div class={styles.noResults}>
          <p>No play history found for the selected filters.</p>
        </div>
      </Show>

      <div class={styles.historyList}>
        <For each={Object.entries(groupedHistory())}>
          {([groupName, plays]: [string, PlayHistory[]]) => (
            <>
              <Show when={groupName && groupBy() !== "none"}>
                <div class={styles.groupHeader}>
                  {groupBy() === "date" ? useFormattedShortDate(groupName) : groupName}
                </div>
              </Show>

              <div class={styles.playsGrid}>
                <For each={plays}>
                  {(play) => {
                    const release = releases().find((r) => r.id === play.userReleaseId);
                    const thumb = release?.release?.thumb;
                    const title = release?.release?.title || "Unknown Album";
                    const artists = release?.release?.artists || [];

                    return (
                      <div class={styles.playItem} onClick={() => handleItemClick(play)}>
                        <div class={styles.albumArt}>
                          {thumb ? (
                            <img src={thumb} alt={title} class={styles.albumImage} />
                          ) : (
                            <div class={styles.noImage}>No Image</div>
                          )}
                        </div>

                        <div class={styles.playDetails}>
                          <h3 class={styles.albumTitle}>{title}</h3>
                          <p class={styles.artistName}>
                            {artists.map((artist) => artist.name).join(", ") || "Unknown Artist"}
                          </p>
                        </div>

                        <div class={styles.indicators}>
                          <Show when={play.notes && play.notes.trim() !== ""}>
                            <span class={`${styles.indicator} ${styles.hasNotes}`}>
                              <VsNote size={14} />
                              <span>Notes</span>
                            </span>
                          </Show>

                          <Show when={hasCleaning(play.userReleaseId, play.playedAt)}>
                            <span class={`${styles.indicator} ${styles.hasCleaning}`}>
                              <TbWashTemperature5 size={14} />
                              <span>Cleaned</span>
                            </span>
                          </Show>
                        </div>
                      </div>
                    );
                  }}
                </For>
              </div>
            </>
          )}
        </For>
      </div>

      <Show when={selectedPlay()}>
        {(play) => (
          <EditHistoryPanel
            isOpen={isEditPanelOpen()}
            onClose={handleCloseEditPanel}
            editItem={{ ...play(), type: "play" }}
            styluses={styluses()}
          />
        )}
      </Show>
    </div>
  );
};

export default PlayHistoryPage;
