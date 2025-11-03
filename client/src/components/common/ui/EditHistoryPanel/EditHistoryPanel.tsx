import { DateTimeInput } from "@components/common/forms/DateTimeInput/DateTimeInput";
import {
  SearchableSelect,
  type SearchableSelectOption,
} from "@components/common/forms/SearchableSelect/SearchableSelect";
import { Textarea } from "@components/common/forms/Textarea/Textarea";
import { Button } from "@components/common/ui/Button/Button";
import type { CleaningHistory, PlayHistory } from "@models/Release";
import type { UserStylus } from "@models/User";
import { useUpdateCleaning, useUpdatePlay } from "@services/apiHooks";
import { formatDateTimeForInput } from "@utils/dates";
import clsx from "clsx";
import { AiOutlineClose } from "solid-icons/ai";
import { type Component, createEffect, createSignal, Show } from "solid-js";
import styles from "./EditHistoryPanel.module.scss";

export interface EditHistoryPanelProps {
  isOpen: boolean;
  onClose: () => void;
  editItem: ((PlayHistory | CleaningHistory) & { type: "play" | "cleaning" }) | null;
  styluses: UserStylus[];
}

export const EditHistoryPanel: Component<EditHistoryPanelProps> = (props) => {
  const [dateTime, setDateTime] = createSignal("");
  const [notes, setNotes] = createSignal("");
  const [stylusId, setStylusId] = createSignal<string | null>(null);

  createEffect(() => {
    if (props.editItem) {
      const timestamp =
        props.editItem.type === "play"
          ? (props.editItem as PlayHistory).playedAt
          : (props.editItem as CleaningHistory).cleanedAt;

      setDateTime(formatDateTimeForInput(timestamp));
      setNotes(props.editItem.notes || "");

      if (props.editItem.type === "play") {
        setStylusId((props.editItem as PlayHistory).userStylusId || null);
      }
    }
  });

  const updatePlayMutation = useUpdatePlay(props.editItem?.id || "", {
    invalidateQueries: [["user"]],
    onSuccess: () => {
      props.onClose();
    },
  });

  const updateCleaningMutation = useUpdateCleaning(props.editItem?.id || "", {
    invalidateQueries: [["user"]],
    onSuccess: () => {
      props.onClose();
    },
  });

  const handleSave = () => {
    if (!props.editItem) return;

    if (props.editItem.type === "play") {
      updatePlayMutation.mutate({
        playedAt: new Date(dateTime()).toISOString(),
        userStylusId: stylusId() || undefined,
        notes: notes() || undefined,
      });
    } else {
      updateCleaningMutation.mutate({
        cleanedAt: new Date(dateTime()).toISOString(),
        notes: notes() || undefined,
        isDeepClean: (props.editItem as CleaningHistory).isDeepClean || false,
      });
    }
  };

  const stylusOptions = (): SearchableSelectOption[] => {
    return [
      { value: "", label: "None" },
      ...props.styluses
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

  const isSubmitting = () => updatePlayMutation.isPending || updateCleaningMutation.isPending;

  return (
    <div class={clsx(styles.panelWrapper, { [styles.open]: props.isOpen })}>
      <div class={styles.overlay} onClick={props.onClose} />

      <div class={styles.panel}>
        <div class={styles.panelHeader}>
          <h2 class={styles.panelTitle}>
            Edit {props.editItem?.type === "play" ? "Play" : "Cleaning"} Record
          </h2>
          <button type="button" class={styles.closeButton} onClick={props.onClose}>
            <AiOutlineClose size={20} />
          </button>
        </div>

        <div class={styles.panelBody}>
          <div class={styles.formGroup}>
            <DateTimeInput
              label="Date & Time"
              name="editDateTime"
              value={dateTime()}
              onChange={setDateTime}
            />
          </div>

          <Show when={props.editItem?.type === "play"}>
            <div class={styles.formGroup}>
              <SearchableSelect
                label="Stylus Used"
                name="editStylusSelect"
                placeholder="Select a stylus"
                searchPlaceholder="Search styluses..."
                options={stylusOptions()}
                value={stylusId() || ""}
                onChange={(val) => setStylusId(val || null)}
                emptyMessage="No styluses found"
              />
            </div>
          </Show>

          <div class={styles.formGroup}>
            <Textarea
              label="Notes"
              name="editNotes"
              value={notes()}
              placeholder="Add notes about this record..."
              rows={4}
              onChange={setNotes}
            />
          </div>
        </div>

        <div class={styles.panelFooter}>
          <Button variant="secondary" onClick={props.onClose} disabled={isSubmitting()}>
            Cancel
          </Button>
          <Button variant="primary" onClick={handleSave} disabled={isSubmitting()}>
            {isSubmitting() ? "Saving..." : "Save Changes"}
          </Button>
        </div>
      </div>
    </div>
  );
};
