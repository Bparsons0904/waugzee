import { ConfirmationModal } from "@components/common/ui/ConfirmationModal/ConfirmationModal";
import { NotesViewPanel } from "@components/common/ui/NotesViewPanel/NotesViewPanel";
import type { CleaningHistory, PlayHistory } from "@models/Release";
import { useDeleteCleaning, useDeletePlay } from "@services/apiHooks";
import { formatHistoryDate } from "@utils/dates";
import { BiSolidEdit } from "solid-icons/bi";
import { BsVinylFill } from "solid-icons/bs";
import { FaSolidTrash } from "solid-icons/fa";
import { TbWashTemperature5 } from "solid-icons/tb";
import { VsNote } from "solid-icons/vs";
import clsx from "clsx";
import { type Component, createSignal, Match, Show, Switch } from "solid-js";
import styles from "./RecordHistoryItem.module.scss";

export interface RecordHistoryItemProps {
  item: (PlayHistory | CleaningHistory) & { type: "play" | "cleaning" };
  onEdit: (item: (PlayHistory | CleaningHistory) & { type: "play" | "cleaning" }) => void;
}

export const RecordHistoryItem: Component<RecordHistoryItemProps> = (props) => {
  const [isNotesPanelOpen, setIsNotesPanelOpen] = createSignal(false);
  const [isDeleteConfirmOpen, setIsDeleteConfirmOpen] = createSignal(false);

  const deletePlayMutation = useDeletePlay({
    invalidateQueries: [["user"]],
    onSuccess: () => {
      setIsDeleteConfirmOpen(false);
    },
  });

  const deleteCleaningMutation = useDeleteCleaning({
    invalidateQueries: [["user"]],
    onSuccess: () => {
      setIsDeleteConfirmOpen(false);
    },
  });

  const openNotesPanel = (e: Event) => {
    e.stopPropagation();
    setIsNotesPanelOpen(true);
  };

  const handleEdit = (e: Event) => {
    e.stopPropagation();
    props.onEdit(props.item);
  };

  const handleDelete = () => {
    if (props.item.type === "play") {
      deletePlayMutation.mutate(props.item.id);
    } else {
      deleteCleaningMutation.mutate(props.item.id);
    }
  };

  const timestamp = () =>
    props.item.type === "play"
      ? (props.item as PlayHistory).playedAt
      : (props.item as CleaningHistory).cleanedAt;

  const stylusName = () => {
    if (props.item.type === "play") {
      const playItem = props.item as PlayHistory;
      if (playItem.userStylus?.stylus) {
        return `${playItem.userStylus.stylus.brand} ${playItem.userStylus.stylus.model}`;
      }
      if (playItem.userStylusId) {
        return `Stylus ${playItem.userStylusId.substring(0, 8)}`;
      }
    }
    return undefined;
  };

  return (
    <>
      <div
        class={clsx(styles.historyItem, {
          [styles.playItem]: props.item.type === "play",
          [styles.cleaningItem]: props.item.type === "cleaning",
        })}
      >
        <div class={styles.historyItemContent}>
          <div class={styles.historyItemHeader}>
            <div class={styles.typeAndNotes}>
              <span class={styles.historyItemType}>
                <Switch>
                  <Match when={props.item.type === "play"}>
                    <span class={styles.historyItems}>
                      <BsVinylFill size={18} /> Played
                    </span>
                  </Match>
                  <Match when={props.item.type === "cleaning"}>
                    <span class={styles.historyItems}>
                      <TbWashTemperature5 size={20} /> Cleaned
                      <Show when={(props.item as CleaningHistory).isDeepClean}>
                        <span class={styles.deepCleanBadge}>Deep Clean</span>
                      </Show>
                    </span>
                  </Match>
                </Switch>
              </span>

              <Show when={stylusName()}>
                <div class={styles.historyItemStylus}>Stylus: {stylusName()}</div>
              </Show>

              <Show when={props.item.notes}>
                <button
                  type="button"
                  class={styles.noteButton}
                  onClick={openNotesPanel}
                  title="View notes"
                >
                  <VsNote class={styles.noteIcon} size={18} />
                </button>
              </Show>
            </div>

            <div class={styles.dateAndActions}>
              <div class={styles.actionIcons}>
                <button type="button" class={styles.editButton} onClick={handleEdit} title="Edit">
                  <BiSolidEdit size={16} />
                </button>
                <button
                  type="button"
                  class={styles.deleteButton}
                  onClick={(e) => {
                    e.stopPropagation();
                    setIsDeleteConfirmOpen(true);
                  }}
                  title="Delete"
                >
                  <FaSolidTrash size={16} />
                </button>
              </div>
              <span class={styles.historyItemDate}>{formatHistoryDate(timestamp())}</span>
            </div>
          </div>
        </div>
      </div>

      <Show when={props.item.notes}>
        <NotesViewPanel
          isOpen={isNotesPanelOpen()}
          onClose={() => setIsNotesPanelOpen(false)}
          title={`${props.item.type === "play" ? "Play" : "Cleaning"} Record Details`}
          date={timestamp()}
          stylusName={stylusName()}
          notes={props.item.notes || ""}
        />
      </Show>

      <ConfirmationModal
        isOpen={isDeleteConfirmOpen()}
        title="Confirm Delete"
        message={`Are you sure you want to delete this ${props.item.type === "play" ? "play" : "cleaning"} record? This action cannot be undone.`}
        confirmText="Delete"
        onConfirm={handleDelete}
        onCancel={() => setIsDeleteConfirmOpen(false)}
      />
    </>
  );
};
