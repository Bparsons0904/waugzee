import { DateInput } from "@components/common/forms/DateInput/DateInput";
import {
  SearchableSelect,
  type SearchableSelectOption,
} from "@components/common/forms/SearchableSelect/SearchableSelect";
import { Textarea } from "@components/common/forms/Textarea/Textarea";
import { Button } from "@components/common/ui/Button/Button";
import { Image } from "@components/common/ui/Image/Image";
import { useUserData } from "@context/UserDataContext";
import type { CleaningHistory, PlayHistory } from "@models/Release";
import type { UserRelease } from "@models/User";
import { useLogCleaning, useLogPlay } from "@services/apiHooks";
import { formatDateForInput } from "@utils/dates";
import { type Component, createMemo, For, Show } from "solid-js";
import { createStore } from "solid-js/store";
import styles from "./RecordActionModal.module.scss";

interface RecordActionModalProps {
  isOpen: boolean;
  onClose: () => void;
  release: UserRelease;
}

type HistoryItem = (PlayHistory | CleaningHistory) & {
  type: "play" | "cleaning";
  timestamp: string;
};

const RecordActionModal: Component<RecordActionModalProps> = (props) => {
  const userData = useUserData();

  const [formState, setFormState] = createStore({
    date: formatDateForInput(new Date()),
    selectedStylusId: userData.styluses().find((s) => s.isPrimary && s.isActive)?.id,
    notes: "",
  });

  const logPlayMutation = useLogPlay({
    invalidateQueries: [["user"]],
    onSuccess: () => {
      resetForm();
      props.onClose();
    },
  });

  const logCleaningMutation = useLogCleaning({
    invalidateQueries: [["user"]],
    onSuccess: () => {
      resetForm();
      props.onClose();
    },
  });

  const resetForm = () => {
    setFormState({
      date: formatDateForInput(new Date()),
      selectedStylusId: null,
      notes: "",
    });
  };

  const handleLogPlay = () => {
    const dateObj = new Date(formState.date);
    logPlayMutation.mutate({
      releaseId: props.release.releaseId,
      playedAt: dateObj.toISOString(),
      userStylusId: formState.selectedStylusId || undefined,
      notes: formState.notes || undefined,
    });
  };

  const handleLogCleaning = () => {
    const dateObj = new Date(formState.date);
    logCleaningMutation.mutate({
      releaseId: props.release.releaseId,
      cleanedAt: dateObj.toISOString(),
      notes: formState.notes || undefined,
      isDeepClean: false,
    });
  };

  const handleLogBoth = () => {
    const dateObj = new Date(formState.date);
    const playData = {
      releaseId: props.release.releaseId,
      playedAt: dateObj.toISOString(),
      userStylusId: formState.selectedStylusId || undefined,
      notes: formState.notes || undefined,
    };

    const cleaningData = {
      releaseId: props.release.releaseId,
      cleanedAt: dateObj.toISOString(),
      notes: formState.notes || undefined,
      isDeepClean: false,
    };

    logPlayMutation.mutate(playData, {
      onSuccess: () => {
        logCleaningMutation.mutate(cleaningData);
      },
    });
  };

  const stylusOptions = (): SearchableSelectOption[] => {
    const styluses = userData.styluses();
    return [
      { value: "", label: "None" },
      ...styluses
        .filter((s) => s.isActive)
        .map((s) => ({
          value: s.id,
          label: s.stylus
            ? `${s.stylus.brand} ${s.stylus.model}`
            : `Stylus ${s.id.substring(0, 8)}`,
          metadata: s.isPrimary ? "Primary" : undefined,
        })),
    ];
  };

  const releaseHistory = createMemo((): HistoryItem[] => {
    const plays = props.release.playHistory || [];
    const cleanings = props.release.cleaningHistory || [];

    const playItems: HistoryItem[] = plays.map((p) => ({
      ...p,
      type: "play" as const,
      timestamp: p.playedAt,
    }));

    const cleaningItems: HistoryItem[] = cleanings.map((c) => ({
      ...c,
      type: "cleaning" as const,
      timestamp: c.cleanedAt,
    }));

    return [...playItems, ...cleaningItems].sort(
      (a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime(),
    );
  });

  const formatHistoryDate = (dateString: string): string => {
    const date = new Date(dateString);
    const now = new Date();
    const diffInDays = Math.floor((now.getTime() - date.getTime()) / (1000 * 60 * 60 * 24));

    if (diffInDays === 0) return "Today";
    if (diffInDays === 1) return "Yesterday";
    if (diffInDays < 7) return `${diffInDays} days ago`;

    return date.toLocaleDateString("en-US", {
      month: "short",
      day: "numeric",
      year: date.getFullYear() !== now.getFullYear() ? "numeric" : undefined,
    });
  };

  return (
    <Show when={props.isOpen}>
      <div class={styles.modalOverlay} onClick={props.onClose}>
        <div class={styles.modal} onClick={(e) => e.stopPropagation()}>
          <div class={styles.modalHeader}>
            <button type="button" class={styles.closeButton} onClick={props.onClose}>
              ×
            </button>
            <h2 class={styles.modalTitle}>Record Actions</h2>
          </div>

          <div class={styles.recordDetails}>
            <div class={styles.recordImage}>
              <Image
                src={props.release.release.thumb || ""}
                alt={props.release.release.title || "Release"}
                aspectRatio="square"
                showSkeleton={true}
              />
            </div>
            <div class={styles.recordInfo}>
              <h3 class={styles.recordTitle}>{props.release.release.title}</h3>
              <p class={styles.recordArtist}>{props.release.release.format || "Unknown Format"}</p>
              {props.release.release.year && (
                <p class={styles.recordYear}>{props.release.release.year}</p>
              )}
            </div>
          </div>

          <div class={styles.formSection}>
            <div class={styles.formRow}>
              <div class={styles.formGroup}>
                <DateInput
                  label="Date"
                  name="actionDate"
                  value={formState.date}
                  onChange={(value) => setFormState("date", value)}
                />
              </div>

              <div class={styles.formGroup}>
                <SearchableSelect
                  label="Stylus Used"
                  name="stylusSelect"
                  placeholder="Select a stylus"
                  searchPlaceholder="Search styluses..."
                  options={stylusOptions()}
                  value={formState.selectedStylusId || ""}
                  onChange={(val) => setFormState("selectedStylusId", val || null)}
                  emptyMessage="No styluses found"
                />
              </div>
            </div>

            <div class={styles.formGroup}>
              <Textarea
                label="Notes"
                name="notes"
                value={formState.notes}
                placeholder="Enter any notes about this play or cleaning..."
                rows={3}
                onChange={(value) => setFormState("notes", value)}
              />
            </div>
          </div>

          <div class={styles.actionButtons}>
            <Button
              variant="primary"
              onClick={handleLogPlay}
              disabled={logPlayMutation.isPending}
              class={styles.playButton}
            >
              {logPlayMutation.isPending ? "Logging..." : "Log Play"}
            </Button>
            <Button
              variant="secondary"
              onClick={handleLogBoth}
              disabled={logPlayMutation.isPending || logCleaningMutation.isPending}
              class={styles.bothButton}
            >
              {logPlayMutation.isPending || logCleaningMutation.isPending
                ? "Logging..."
                : "Log Both"}
            </Button>
            <Button
              variant="tertiary"
              onClick={handleLogCleaning}
              disabled={logCleaningMutation.isPending}
              class={styles.cleaningButton}
            >
              {logCleaningMutation.isPending ? "Logging..." : "Log Cleaning"}
            </Button>
          </div>

          <div class={styles.historySection}>
            <h3 class={styles.historyTitle}>Record History</h3>

            <div class={styles.historyList}>
              <Show
                when={releaseHistory().length > 0}
                fallback={
                  <div class={styles.noHistory}>
                    No play or cleaning history for this record yet.
                  </div>
                }
              >
                <For each={releaseHistory()}>
                  {(item) => (
                    <div class={styles.historyItem}>
                      <div class={styles.historyItemHeader}>
                        <span
                          class={item.type === "play" ? styles.playBadge : styles.cleaningBadge}
                        >
                          {item.type === "play" ? "Play" : "Cleaning"}
                        </span>
                        <span class={styles.historyDate}>{formatHistoryDate(item.timestamp)}</span>
                      </div>
                      <Show when={item.type === "play" && "userStylus" in item && item.userStylus}>
                        {(stylus) => (
                          <div class={styles.historyStylus}>
                            Stylus:{" "}
                            {stylus().stylus
                              ? `${stylus().stylus.brand} ${stylus().stylus.model}`
                              : "Unknown"}
                          </div>
                        )}
                      </Show>
                      <Show when={item.notes}>
                        <div class={styles.historyNotes}>{item.notes}</div>
                      </Show>
                    </div>
                  )}
                </For>
              </Show>
            </div>
          </div>
        </div>
      </div>
    </Show>
  );
};

export default RecordActionModal;
