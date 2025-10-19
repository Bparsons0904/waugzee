import { DateInput } from "@components/common/forms/DateInput/DateInput";
import { Textarea } from "@components/common/forms/Textarea/Textarea";
import { Toggle } from "@components/common/forms/Toggle/Toggle";
import { Button } from "@components/common/ui/Button/Button";
import type { UserStylus } from "@models/Stylus";
import { useUpdateUserStylus } from "@services/apiHooks";
import { formatDateForInput } from "@utils/dates";
import type { Component } from "solid-js";
import { createStore } from "solid-js/store";
import styles from "./EditEquipmentModal.module.scss";

interface EditEquipmentModalProps {
  stylus: UserStylus;
  onClose: () => void;
}

interface EditFormState {
  purchaseDate: string;
  installDate: string;
  notes: string;
  isActive: boolean;
  isPrimary: boolean;
}

const EditEquipmentModal: Component<EditEquipmentModalProps> = (props) => {
  const updateUserStylusMutation = useUpdateUserStylus();

  const [formState, setFormState] = createStore<EditFormState>({
    purchaseDate: props.stylus.purchaseDate ? formatDateForInput(props.stylus.purchaseDate) : "",
    installDate: props.stylus.installDate ? formatDateForInput(props.stylus.installDate) : "",
    notes: props.stylus.notes || "",
    isActive: props.stylus.isActive,
    isPrimary: props.stylus.isPrimary,
  });

  const handleSubmit = (e: Event) => {
    e.preventDefault();

    updateUserStylusMutation.mutate(
      {
        id: props.stylus.id,
        data: {
          purchaseDate: formState.purchaseDate || undefined,
          installDate: formState.installDate || undefined,
          notes: formState.notes || undefined,
          isActive: formState.isActive,
          isPrimary: formState.isPrimary,
        },
      },
      {
        onSuccess: () => {
          props.onClose();
        },
      },
    );
  };

  return (
    <form class={styles.form} onSubmit={handleSubmit}>
      <div class={styles.formRow}>
        <DateInput
          name="purchaseDate"
          label="Purchase Date"
          value={formState.purchaseDate}
          onChange={(value) => setFormState("purchaseDate", value)}
        />

        <DateInput
          name="installDate"
          label="Install Date"
          value={formState.installDate}
          onChange={(value) => setFormState("installDate", value)}
        />
      </div>

      <div class={`${styles.formRow} ${styles.full}`}>
        <Textarea
          name="notes"
          label="Notes"
          value={formState.notes}
          onChange={(value) => setFormState("notes", value)}
          rows={3}
        />
      </div>

      <div class={styles.statusSection}>
        <h3>Status</h3>
        <div class={styles.toggleGroup}>
          <Toggle
            label="Active"
            checked={formState.isActive}
            onChange={(checked) => setFormState("isActive", checked)}
          />

          <Toggle
            label="Primary"
            checked={formState.isPrimary}
            onChange={(checked) => setFormState("isPrimary", checked)}
          />
        </div>
      </div>

      <div class={styles.formActions}>
        <Button
          type="button"
          variant="tertiary"
          onClick={props.onClose}
          disabled={updateUserStylusMutation.isPending}
        >
          Cancel
        </Button>
        <Button type="submit" variant="primary" disabled={updateUserStylusMutation.isPending}>
          {updateUserStylusMutation.isPending ? "Updating..." : "Update"}
        </Button>
      </div>
    </form>
  );
};

export default EditEquipmentModal;
