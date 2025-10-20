import { DateInput } from "@components/common/forms/DateInput/DateInput";
import {
  SearchableSelect,
  type SearchableSelectOption,
} from "@components/common/forms/SearchableSelect/SearchableSelect";
import { Textarea } from "@components/common/forms/Textarea/Textarea";
import { Toggle } from "@components/common/forms/Toggle/Toggle";
import { Button } from "@components/common/ui/Button/Button";
import type { CreateUserStylusRequest } from "@models/Stylus";
import { useAvailableStyluses, useCreateUserStylus } from "@services/apiHooks";
import { formatDateForInput } from "@utils/dates";
import type { Component } from "solid-js";
import { createStore } from "solid-js/store";
import styles from "./AddEquipmentModal.module.scss";

interface AddEquipmentModalProps {
  onClose: () => void;
  onOpenCustomModal: () => void;
}

interface FormState {
  selectedStylusId: string;
  purchaseDate: string;
  installDate: string;
  notes: string;
  isActive: boolean;
  isPrimary: boolean;
}

const AddEquipmentModal: Component<AddEquipmentModalProps> = (props) => {
  const availableStylusesQuery = useAvailableStyluses();
  const createUserStylusMutation = useCreateUserStylus();

  const [formState, setFormState] = createStore<FormState>({
    selectedStylusId: "",
    purchaseDate: formatDateForInput(new Date()),
    installDate: formatDateForInput(new Date()),
    notes: "",
    isActive: true,
    isPrimary: false,
  });

  const resetForm = () => {
    setFormState({
      selectedStylusId: "",
      purchaseDate: formatDateForInput(new Date()),
      installDate: formatDateForInput(new Date()),
      notes: "",
      isActive: true,
      isPrimary: false,
    });
  };

  const handleClose = () => {
    resetForm();
    props.onClose();
  };

  const handleSubmit = (e: Event) => {
    e.preventDefault();
    if (!formState.selectedStylusId) return;

    const request: CreateUserStylusRequest = {
      stylusId: formState.selectedStylusId,
      purchaseDate: formState.purchaseDate || undefined,
      installDate: formState.installDate || undefined,
      notes: formState.notes || undefined,
      isActive: formState.isActive,
      isPrimary: formState.isPrimary,
    };

    createUserStylusMutation.mutate(request, {
      onSuccess: () => {
        handleClose();
      },
    });
  };

  const availableStyluses = () => availableStylusesQuery.data?.styluses || [];

  const stylusSelectOptions = (): SearchableSelectOption[] => {
    return availableStyluses().map((stylus) => ({
      value: stylus.id,
      label: `${stylus.brand} ${stylus.model}`,
      metadata: stylus.type,
    }));
  };

  return (
    <form class={styles.form} onSubmit={handleSubmit}>
      <div class={styles.stylusSelectRow}>
        <div class={styles.stylusSelectWrapper}>
          <SearchableSelect
            label="Select Stylus"
            name="stylusSelect"
            placeholder="-- Select a stylus --"
            searchPlaceholder="Search styluses..."
            options={stylusSelectOptions()}
            value={formState.selectedStylusId}
            onChange={(value) => setFormState("selectedStylusId", value)}
            required
            emptyMessage="No styluses found"
          />
        </div>
        <Button
          type="button"
          variant="secondary"
          onClick={props.onOpenCustomModal}
          class={styles.createCustomButton}
        >
          Create Custom
        </Button>
      </div>

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
          onClick={handleClose}
          disabled={createUserStylusMutation.isPending}
        >
          Cancel
        </Button>
        <Button type="submit" variant="primary" disabled={createUserStylusMutation.isPending}>
          {createUserStylusMutation.isPending ? "Adding..." : "Add Stylus"}
        </Button>
      </div>
    </form>
  );
};

export default AddEquipmentModal;
