import { Select } from "@components/common/forms/Select/Select";
import { TextInput } from "@components/common/forms/TextInput/TextInput";
import { EditHistoryPanel } from "@components/common/ui/EditHistoryPanel/EditHistoryPanel";
import { useUserData } from "@context/UserDataContext";
import type { PlayHistory } from "@models/Release";
import { getLocalDateGroupKey, isSameLocalDay, useFormattedShortDate } from "@utils/dates";
import { fuzzySearchPlayHistory } from "@utils/fuzzy";
import { type Component, createMemo, createSignal, For, Show } from "solid-js";
import styles from "./PlayHistory.module.scss";
import { PlayItemCard } from "./PlayItemCard";

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
        switch (groupBy()) {
          case "date":
            key = getLocalDateGroupKey(play.playedAt);
            break;
          case "artist": {
            const release = releases().find((r) => r.id === play.userReleaseId);
            key = release?.release?.artists?.[0]?.name || "Unknown Artist";
            break;
          }
          case "album": {
            const release = releases().find((r) => r.id === play.userReleaseId);
            key = release?.release?.title || "Unknown Album";
            break;
          }
          default:
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
          <Select
            label="Time Period"
            value={timeFilter()}
            options={[
              { value: "all", label: "All Time" },
              { value: "week", label: "Last Week" },
              { value: "month", label: "Last Month" },
              { value: "year", label: "Last Year" },
            ]}
            onChange={(value) => setTimeFilter(value)}
          />
        </div>

        <div class={styles.filterGroup}>
          <Select
            label="Group By"
            value={groupBy()}
            options={[
              { value: "none", label: "None" },
              { value: "date", label: "Date" },
              { value: "artist", label: "Artist" },
              { value: "album", label: "Album" },
            ]}
            onChange={(value) => setGroupBy(value)}
          />
        </div>

        <div class={styles.searchBox}>
          <TextInput
            label="Search"
            placeholder="Search by artist, album, stylus or notes..."
            value={searchTerm()}
            onInput={(value) => setSearchTerm(value)}
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
                    return (
                      <PlayItemCard
                        play={play}
                        release={release}
                        hasCleaning={hasCleaning(play.userReleaseId, play.playedAt)}
                        onClick={() => handleItemClick(play)}
                      />
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
