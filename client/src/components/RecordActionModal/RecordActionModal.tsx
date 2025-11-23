import { DateTimeInput } from "@components/common/forms/DateTimeInput/DateTimeInput";
import {
  SearchableSelect,
  type SearchableSelectOption,
} from "@components/common/forms/SearchableSelect/SearchableSelect";
import { Textarea } from "@components/common/forms/Textarea/Textarea";
import { Toggle } from "@components/common/forms/Toggle/Toggle";
import { Button } from "@components/common/ui/Button/Button";
import { EditHistoryPanel } from "@components/common/ui/EditHistoryPanel/EditHistoryPanel";
import { Image } from "@components/common/ui/Image/Image";
import { Modal, ModalSize } from "@components/common/ui/Modal/Modal";
import { RecordHistoryItem } from "@components/RecordHistoryItem/RecordHistoryItem";
import { useUserData } from "@context/UserDataContext";
import type { CleaningHistory, PlayHistory } from "@models/Release";
import type { UserRelease } from "@models/User";
import {
  useArchiveRelease,
  useDeleteRelease,
  useLogBoth,
  useLogCleaning,
  useLogPlay,
  useUnarchiveRelease,
} from "@services/apiHooks";
import { formatDateTimeForInput } from "@utils/dates";
import { type Component, createMemo, createSignal, For, Show } from "solid-js";
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
    dateTime: formatDateTimeForInput(new Date()),
    selectedStylusId: userData.styluses().find((s) => s.isPrimary && s.isActive)?.id,
    notes: "",
    isDeepClean: false,
  });

  const [isEditPanelOpen, setIsEditPanelOpen] = createSignal(false);
  const [editItem, setEditItem] = createSignal<
    ((PlayHistory | CleaningHistory) & { type: "play" | "cleaning" }) | null
  >(null);
  const [showDeleteConfirm, setShowDeleteConfirm] = createSignal(false);

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

  const logBothMutation = useLogBoth({
    invalidateQueries: [["user"]],
    onSuccess: () => {
      resetForm();
      props.onClose();
    },
  });

  const archiveMutation = useArchiveRelease({
    onSuccess: () => {
      props.onClose();
    },
  });

  const unarchiveMutation = useUnarchiveRelease({
    onSuccess: () => {
      props.onClose();
    },
  });

  const deleteMutation = useDeleteRelease({
    onSuccess: () => {
      setShowDeleteConfirm(false);
      props.onClose();
    },
  });

  const handleArchive = () => {
    archiveMutation.mutate(props.release.id);
  };

  const handleUnarchive = () => {
    unarchiveMutation.mutate(props.release.id);
  };

  const handleDelete = () => {
    deleteMutation.mutate(props.release.id);
  };

  const resetForm = () => {
    setFormState({
      dateTime: formatDateTimeForInput(new Date()),
      selectedStylusId: null,
      notes: "",
      isDeepClean: false,
    });
  };

  const handleLogPlay = () => {
    logPlayMutation.mutate({
      userReleaseId: props.release.id,
      playedAt: new Date(formState.dateTime).toISOString(),
      userStylusId: formState.selectedStylusId || undefined,
      notes: formState.notes || undefined,
    });
  };

  const handleLogCleaning = () => {
    logCleaningMutation.mutate({
      userReleaseId: props.release.id,
      cleanedAt: new Date(formState.dateTime).toISOString(),
      notes: formState.notes || undefined,
      isDeepClean: formState.isDeepClean,
    });
  };

  const handleLogBoth = () => {
    logBothMutation.mutate({
      userReleaseId: props.release.id,
      userStylusId: formState.selectedStylusId || undefined,
      timestamp: new Date(formState.dateTime).toISOString(),
      notes: formState.notes || undefined,
      isDeepClean: formState.isDeepClean,
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

  const handleEdit = (item: (PlayHistory | CleaningHistory) & { type: "play" | "cleaning" }) => {
    setEditItem(item);
    setIsEditPanelOpen(true);
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

  return (
    <>
      <Modal
        isOpen={props.isOpen}
        onClose={props.onClose}
        title="Record Actions"
        size={ModalSize.Large}
      >
        <div class={styles.modalContent}>
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
              <h3 class={styles.recordTitle}>
                {props.release.release.title}
                <Show when={props.release.archived}>
                  <span class={styles.archivedBadge}>Archived</span>
                </Show>
              </h3>
              <p class={styles.recordArtist}>{props.release.release.format || "Unknown Format"}</p>
              {props.release.release.year && (
                <p class={styles.recordYear}>{props.release.release.year}</p>
              )}
            </div>
          </div>

          <div class={styles.formSection}>
            <div class={styles.formRow}>
              <div class={styles.formGroup}>
                <DateTimeInput
                  label="Date & Time"
                  name="actionDateTime"
                  value={formState.dateTime}
                  onChange={(value) => setFormState("dateTime", value)}
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

            <div class={styles.formGroup}>
              <Toggle
                label="Deep Clean"
                name="isDeepClean"
                checked={formState.isDeepClean}
                onChange={(checked) => setFormState("isDeepClean", checked)}
              />
            </div>
          </div>

          <Show when={props.release.archived}>
            <div class={styles.archivedNotice}>
              This record is archived. Unarchive it to log new plays or cleanings.
            </div>
          </Show>

          <div class={styles.actionButtons}>
            <Button
              variant="primary"
              onClick={handleLogPlay}
              disabled={logPlayMutation.isPending || props.release.archived}
              class={styles.playButton}
            >
              {logPlayMutation.isPending ? "Logging..." : "Log Play"}
            </Button>
            <Button
              variant="secondary"
              onClick={handleLogBoth}
              disabled={logBothMutation.isPending || props.release.archived}
              class={styles.bothButton}
            >
              {logBothMutation.isPending ? "Logging..." : "Log Both"}
            </Button>
            <Button
              variant="tertiary"
              onClick={handleLogCleaning}
              disabled={logCleaningMutation.isPending || props.release.archived}
              class={styles.cleaningButton}
            >
              {logCleaningMutation.isPending ? "Logging..." : "Log Cleaning"}
            </Button>
          </div>

          <div class={styles.managementButtons}>
            <Show
              when={props.release.archived}
              fallback={
                <Button
                  variant="secondary"
                  onClick={handleArchive}
                  disabled={archiveMutation.isPending}
                  class={styles.archiveButton}
                >
                  {archiveMutation.isPending ? "Archiving..." : "Archive Record"}
                </Button>
              }
            >
              <Button
                variant="secondary"
                onClick={handleUnarchive}
                disabled={unarchiveMutation.isPending}
                class={styles.unarchiveButton}
              >
                {unarchiveMutation.isPending ? "Unarchiving..." : "Unarchive Record"}
              </Button>
            </Show>
            <Button
              variant="danger"
              onClick={() => setShowDeleteConfirm(true)}
              class={styles.deleteButton}
            >
              Delete Record
            </Button>
          </div>

          <Show when={showDeleteConfirm()}>
            <div class={styles.deleteConfirm}>
              <p>Are you sure you want to delete this record? This action cannot be undone.</p>
              <div class={styles.confirmButtons}>
                <Button variant="secondary" onClick={() => setShowDeleteConfirm(false)}>
                  Cancel
                </Button>
                <Button
                  variant="danger"
                  onClick={handleDelete}
                  disabled={deleteMutation.isPending}
                >
                  {deleteMutation.isPending ? "Deleting..." : "Confirm Delete"}
                </Button>
              </div>
            </div>
          </Show>

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
                  {(item) => <RecordHistoryItem item={item} onEdit={handleEdit} />}
                </For>
              </Show>
            </div>
          </div>
        </div>
      </Modal>

      <EditHistoryPanel
        isOpen={isEditPanelOpen()}
        onClose={() => setIsEditPanelOpen(false)}
        editItem={editItem()}
        styluses={userData.styluses()}
      />
    </>
  );
};

export default RecordActionModal;
