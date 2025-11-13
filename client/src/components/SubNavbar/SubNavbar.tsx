import { Select } from "@components/common/forms/Select/Select";
import { Button } from "@components/common/ui/Button/Button";
import { useUserData } from "@context/UserDataContext";
import { useToast } from "@context/ToastContext";
import type { UserRelease } from "@models/User";
import { useLogPlay, useMarkRecommendationListened } from "@services/apiHooks";
import { suggestByGenre, suggestLeastPlayed, suggestRandom } from "@utils/recommendationAlgorithms";
import { type Component, createMemo, createSignal, For, lazy, Show } from "solid-js";
import styles from "./SubNavbar.module.scss";

const RecordActionModal = lazy(() => import("@components/RecordActionModal/RecordActionModal"));

type SuggestionMode = "one" | "several" | "leastPlayed" | "randomGenre";

export const SubNavbar: Component = () => {
  const { user, releases, playHistory, styluses, dailyRecommendation } = useUserData();
  const [suggestionMode, setSuggestionMode] = createSignal<SuggestionMode>("one");
  const [showSuggestions, setShowSuggestions] = createSignal(false);
  const [suggestions, setSuggestions] = createSignal<UserRelease[]>([]);
  const [selectedGenre, setSelectedGenre] = createSignal<string>("");
  const [isModalOpen, setIsModalOpen] = createSignal(false);
  const [selectedRelease, setSelectedRelease] = createSignal<UserRelease | null>(null);

  const isListened = createMemo(() => !!dailyRecommendation()?.listenedAt);
  const primaryStylus = createMemo(() => styluses().find((s) => s.isPrimary));

  const logPlayMutation = useLogPlay({
    invalidateQueries: [["user"]],
    onSuccess: async () => {
      await markListenedMutation.mutateAsync({});
    },
  });

  const markListenedMutation = useMarkRecommendationListened(dailyRecommendation()?.id || "");

  const handleQuickPlay = () => {
    const recommendation = dailyRecommendation();
    if (!recommendation) return;

    const stylus = primaryStylus();
    logPlayMutation.mutate({
      userReleaseId: recommendation.userReleaseId,
      userStylusId: stylus?.id,
      playedAt: new Date().toISOString(),
      notes: "Quick play from daily recommendation",
    });
  };

  const handleSuggest = () => {
    const mode = suggestionMode();
    const allReleases = releases();

    if (allReleases.length === 0) {
      setSuggestions([]);
      setShowSuggestions(false);
      return;
    }

    let result: UserRelease[] = [];

    switch (mode) {
      case "one":
        result = suggestRandom(allReleases, 1);
        break;
      case "several":
        result = suggestRandom(allReleases, 3);
        break;
      case "leastPlayed":
        result = suggestLeastPlayed(allReleases, playHistory(), 3);
        break;
      case "randomGenre": {
        const { genre, releases: genreReleases } = suggestByGenre(allReleases);
        setSelectedGenre(genre);
        result = genreReleases;
        break;
      }
    }

    setSuggestions(result);
    setShowSuggestions(true);
  };

  const suggestionPlayMutation = useLogPlay({
    invalidateQueries: [["user"]],
    successMessage: "Play logged successfully!",
  });

  const handleSuggestionPlay = (e: MouseEvent, releaseId: string) => {
    e.stopPropagation();
    const stylus = primaryStylus();

    const payload = {
      userReleaseId: releaseId,
      userStylusId: stylus?.id,
      playedAt: new Date().toISOString(),
      notes: "From suggestion system",
    };

    suggestionPlayMutation.mutate(payload);
  };

  const handleCardClick = (release: UserRelease) => {
    setSelectedRelease(release);
    setIsModalOpen(true);
  };

  const handleModalClose = () => {
    setIsModalOpen(false);
    setSelectedRelease(null);
  };

  return (
    <>
      <div class={styles.subNavbar}>
        <div class={styles.container}>
          <div class={styles.recordOfTheDay}>
            <Show when={dailyRecommendation()}>
              {(rec) => (
                <div class={styles.recordCard}>
                  <div class={styles.albumArt}>
                    <img
                      src={rec().userRelease?.release?.thumb || "/placeholder-vinyl.png"}
                      alt={rec().userRelease?.release?.title || "Album"}
                    />
                  </div>
                  <div class={styles.recordInfo}>
                    <div class={styles.title}>
                      {rec().userRelease?.release?.title || "Unknown Album"}
                    </div>
                    <div class={styles.artist}>
                      {rec()
                        .userRelease?.release?.artists?.map((a) => a.name)
                        .join(", ") || "Unknown Artist"}
                    </div>
                  </div>
                  <Show
                    when={!isListened()}
                    fallback={<div class={styles.playedBadge}>Played ✓</div>}
                  >
                    <Button onClick={handleQuickPlay} class={styles.playButton}>
                      Play
                    </Button>
                  </Show>
                </div>
              )}
            </Show>
            <Show when={!dailyRecommendation()}>
              <div class={styles.noRecommendation}>No recommendation for today</div>
            </Show>
          </div>

          <div class={styles.suggestionSystem}>
            <Select
              value={suggestionMode()}
              onChange={(value) => setSuggestionMode(value as SuggestionMode)}
              options={[
                { value: "one", label: "Suggest One" },
                { value: "several", label: "Suggest Several" },
                { value: "leastPlayed", label: "Least Played" },
                { value: "randomGenre", label: "Random Genre" },
              ]}
            />
            <Button onClick={handleSuggest} class={styles.suggestButton}>
              Suggest
            </Button>
          </div>
        </div>
      </div>

      <Show when={showSuggestions()}>
        <div class={styles.suggestionsDropdown}>
          <div class={styles.suggestionsContainer}>
            <div class={styles.suggestionsContent}>
              <Show when={suggestionMode() === "randomGenre"}>
                <div class={styles.genreLabel}>
                  <span class={styles.genreName}>Genre: {selectedGenre()}</span>
                </div>
              </Show>
              <div class={styles.suggestionsGrid}>
                <For each={suggestions()}>
                  {(release) => (
                    <div class={styles.suggestionCard} onClick={() => handleCardClick(release)}>
                      <img
                        src={release.release?.thumb || "/placeholder-vinyl.png"}
                        alt={release.release?.title || "Album"}
                        class={styles.suggestionImage}
                      />
                      <div class={styles.suggestionInfo}>
                        <div class={styles.suggestionTitle}>
                          {release.release?.title || "Unknown"}
                        </div>
                        <div class={styles.suggestionArtist}>
                          {release.release?.artists?.map((a) => a.name).join(", ") || "Unknown"}
                        </div>
                      </div>
                      <Button
                        onClick={(e) => handleSuggestionPlay(e, release.id)}
                        class={styles.suggestionPlayButton}
                      >
                        Play
                      </Button>
                    </div>
                  )}
                </For>
              </div>
            </div>
            <Button
              onClick={() => setShowSuggestions(false)}
              variant="ghost"
              class={styles.closeButton}
            >
              ×
            </Button>
          </div>
        </div>
      </Show>

      <Show when={selectedRelease()}>
        {(release) => (
          <RecordActionModal
            isOpen={isModalOpen()}
            onClose={handleModalClose}
            release={release()}
          />
        )}
      </Show>
    </>
  );
};
