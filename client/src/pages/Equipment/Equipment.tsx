import { Component, For, Show } from "solid-js";
import { createStore } from "solid-js/store";
import {
  useUserStyluses,
  useUpdateUserStylus,
  useDeleteUserStylus,
} from "@services/apiHooks";
import type { UserStylus } from "@models/Stylus";
import { Button } from "@components/common/ui/Button/Button";
import { Modal, ModalSize } from "@components/common/ui/Modal/Modal";
import { DateInput } from "@components/common/forms/DateInput/DateInput";
import { Textarea } from "@components/common/forms/Textarea/Textarea";
import { Toggle } from "@components/common/forms/Toggle/Toggle";
import AddEquipmentModal from "@components/AddEquipmentModal/AddEquipmentModal";
import CreateCustomStylusModal from "@components/CreateCustomStylusModal/CreateCustomStylusModal";
import styles from "./Equipment.module.scss";
import { formatDateForInput, formatLocalDate } from "@utils/dates";

type ModalMode = "add" | "custom" | "edit" | null;

interface EditFormState {
  purchaseDate: string;
  installDate: string;
  notes: string;
  isActive: boolean;
  isPrimary: boolean;
}

const Equipment: Component = () => {
  const userStylusesQuery = useUserStyluses();
  const updateUserStylusMutation = useUpdateUserStylus();
  const deleteUserStylusMutation = useDeleteUserStylus();

  const [modalMode, setModalMode] = createStore<{ current: ModalMode }>({
    current: null,
  });

  const [editingStylus, setEditingStylus] = createStore<{
    stylus: UserStylus | null;
  }>({
    stylus: null,
  });

  const [editFormState, setEditFormState] = createStore<EditFormState>({
    purchaseDate: "",
    installDate: "",
    notes: "",
    isActive: true,
    isPrimary: false,
  });

  const resetEditForm = () => {
    setEditFormState({
      purchaseDate: "",
      installDate: "",
      notes: "",
      isActive: true,
      isPrimary: false,
    });
    setEditingStylus("stylus", null);
  };

  const openAddModal = () => {
    setModalMode("current", "add");
  };

  const openCustomModal = () => {
    setModalMode("current", "custom");
  };

  const openEditModal = (stylus: UserStylus) => {
    setEditFormState({
      purchaseDate: stylus.purchaseDate
        ? formatDateForInput(stylus.purchaseDate)
        : "",
      installDate: stylus.installDate
        ? formatDateForInput(stylus.installDate)
        : "",
      notes: stylus.notes || "",
      isActive: stylus.isActive,
      isPrimary: stylus.isPrimary,
    });
    setEditingStylus("stylus", stylus);
    setModalMode("current", "edit");
  };

  const closeModal = () => {
    setModalMode("current", null);
    resetEditForm();
  };

  const handleUpdate = (e: Event) => {
    e.preventDefault();
    if (!editingStylus.stylus) return;

    updateUserStylusMutation.mutate(
      {
        id: editingStylus.stylus.id,
        data: {
          purchaseDate: editFormState.purchaseDate || undefined,
          installDate: editFormState.installDate || undefined,
          notes: editFormState.notes || undefined,
          isActive: editFormState.isActive,
          isPrimary: editFormState.isPrimary,
        },
      },
      {
        onSuccess: () => {
          closeModal();
        },
      },
    );
  };

  const getStylusDisplayName = (stylus: UserStylus) => {
    if (!stylus.stylus) return "Unknown";
    return `${stylus.stylus.brand} ${stylus.stylus.model}`;
  };

  const handleDelete = (stylus: UserStylus) => {
    if (
      !confirm(
        `Are you sure you want to remove "${getStylusDisplayName(stylus)}" from your equipment?`,
      )
    ) {
      return;
    }

    deleteUserStylusMutation.mutate(stylus.id);
  };

  const styluses = () => userStylusesQuery.data?.styluses || [];
  const activeStyluses = () => styluses().filter((s) => s.isActive);
  const inactiveStyluses = () => styluses().filter((s) => !s.isActive);

  return (
    <div class={styles.container}>
      <div class={styles.header}>
        <h2 class={styles.title}>Equipment Manager</h2>
        <div class={styles.headerButtons}>
          <Show when={!modalMode.current}>
            <Button
              variant="primary"
              onClick={openAddModal}
              disabled={userStylusesQuery.isLoading}
            >
              Add Equipment
            </Button>

            <Button
              variant="secondary"
              onClick={openCustomModal}
              disabled={userStylusesQuery.isLoading}
            >
              Create Custom Stylus
            </Button>
          </Show>
        </div>
      </div>

      <AddEquipmentModal
        isOpen={modalMode.current === "add"}
        onClose={closeModal}
      />

      <CreateCustomStylusModal
        isOpen={modalMode.current === "custom"}
        onClose={closeModal}
      />

      <Modal
        isOpen={modalMode.current === "edit"}
        onClose={closeModal}
        title="Edit Equipment Details"
        size={ModalSize.Medium}
      >
        <form class={styles.editForm} onSubmit={handleUpdate}>
          <div class={styles.formRow}>
            <DateInput
              name="purchaseDate"
              label="Purchase Date"
              value={editFormState.purchaseDate}
              onChange={(value) => setEditFormState("purchaseDate", value)}
            />

            <DateInput
              name="installDate"
              label="Install Date"
              value={editFormState.installDate}
              onChange={(value) => setEditFormState("installDate", value)}
            />
          </div>

          <div class={`${styles.formRow} ${styles.full}`}>
            <Textarea
              name="notes"
              label="Notes"
              value={editFormState.notes}
              onChange={(value) => setEditFormState("notes", value)}
              rows={3}
            />
          </div>

          <div class={styles.statusSection}>
            <h3>Status</h3>
            <div class={styles.toggleGroup}>
              <Toggle
                label="Active"
                checked={editFormState.isActive}
                onChange={(checked) => setEditFormState("isActive", checked)}
              />

              <Toggle
                label="Primary"
                checked={editFormState.isPrimary}
                onChange={(checked) => setEditFormState("isPrimary", checked)}
              />
            </div>
          </div>

          <div class={styles.formActions}>
            <Button
              type="button"
              variant="tertiary"
              onClick={closeModal}
              disabled={updateUserStylusMutation.isPending}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              variant="primary"
              disabled={updateUserStylusMutation.isPending}
            >
              {updateUserStylusMutation.isPending ? "Updating..." : "Update"}
            </Button>
          </div>
        </form>
      </Modal>

      <Show when={!modalMode.current && userStylusesQuery.isLoading}>
        <p class={styles.loading}>Loading equipment...</p>
      </Show>

      <Show
        when={
          !modalMode.current &&
          !userStylusesQuery.isLoading &&
          styluses().length === 0
        }
      >
        <p class={styles.noStyluses}>
          No equipment found. Click "Add Equipment" to select a stylus or
          "Create Custom Stylus" to add your own.
        </p>
      </Show>

      <Show when={!modalMode.current && activeStyluses().length > 0}>
        <div class={styles.section}>
          <h3 class={styles.sectionTitle}>Active Styluses</h3>
          <div class={styles.stylusList}>
            <For each={activeStyluses()}>
              {(stylus) => (
                <div class={styles.stylusCard}>
                  <div class={styles.stylusInfo}>
                    <h3 class={styles.stylusName}>
                      {getStylusDisplayName(stylus)}
                      <div class={styles.tagsContainer}>
                        <span class={styles.activeTag}>Active</span>
                        <Show when={stylus.isPrimary}>
                          <span class={styles.primaryTag}>Primary</span>
                        </Show>
                      </div>
                    </h3>

                    <Show when={stylus.stylus?.type}>
                      <p class={styles.stylusDetail}>
                        <strong>Stylus Type:</strong> {stylus.stylus?.type}
                      </p>
                    </Show>

                    <Show when={stylus.stylus?.cartridgeType}>
                      <p class={styles.stylusDetail}>
                        <strong>Cartridge Type:</strong>{" "}
                        {stylus.stylus?.cartridgeType}
                      </p>
                    </Show>

                    <Show when={stylus.purchaseDate}>
                      <p class={styles.stylusDetail}>
                        <strong>Purchased:</strong>{" "}
                        {formatLocalDate(stylus.purchaseDate!)}
                      </p>
                    </Show>

                    <Show when={stylus.installDate}>
                      <p class={styles.stylusDetail}>
                        <strong>Installed:</strong>{" "}
                        {formatLocalDate(stylus.installDate!)}
                      </p>
                    </Show>

                    <Show when={stylus.hoursUsed}>
                      <p class={styles.stylusDetail}>
                        <strong>Hours Used:</strong> {stylus.hoursUsed}
                      </p>
                    </Show>

                    <Show when={stylus.notes}>
                      <p class={styles.stylusDetail}>
                        <strong>Notes:</strong> {stylus.notes}
                      </p>
                    </Show>
                  </div>

                  <div class={styles.stylusActions}>
                    <Button
                      variant="secondary"
                      size="sm"
                      onClick={() => openEditModal(stylus)}
                      disabled={updateUserStylusMutation.isPending}
                    >
                      Edit
                    </Button>
                    <Button
                      variant="danger"
                      size="sm"
                      onClick={() => handleDelete(stylus)}
                      disabled={deleteUserStylusMutation.isPending}
                    >
                      Delete
                    </Button>
                  </div>
                </div>
              )}
            </For>
          </div>
        </div>
      </Show>

      <Show when={!modalMode.current && inactiveStyluses().length > 0}>
        <div class={styles.section}>
          <h3 class={styles.sectionTitle}>Inactive Styluses</h3>
          <div class={styles.stylusList}>
            <For each={inactiveStyluses()}>
              {(stylus) => (
                <div class={styles.stylusCard}>
                  <div class={styles.stylusInfo}>
                    <h3 class={styles.stylusName}>
                      {getStylusDisplayName(stylus)}
                    </h3>

                    <Show when={stylus.stylus?.type}>
                      <p class={styles.stylusDetail}>
                        <strong>Stylus Type:</strong> {stylus.stylus?.type}
                      </p>
                    </Show>

                    <Show when={stylus.stylus?.cartridgeType}>
                      <p class={styles.stylusDetail}>
                        <strong>Cartridge Type:</strong>{" "}
                        {stylus.stylus?.cartridgeType}
                      </p>
                    </Show>

                    <Show when={stylus.purchaseDate}>
                      <p class={styles.stylusDetail}>
                        <strong>Purchased:</strong>{" "}
                        {formatLocalDate(stylus.purchaseDate!)}
                      </p>
                    </Show>
                  </div>

                  <div class={styles.stylusActions}>
                    <Button
                      variant="secondary"
                      size="sm"
                      onClick={() => openEditModal(stylus)}
                      disabled={updateUserStylusMutation.isPending}
                    >
                      Edit
                    </Button>
                    <Button
                      variant="danger"
                      size="sm"
                      onClick={() => handleDelete(stylus)}
                      disabled={deleteUserStylusMutation.isPending}
                    >
                      Delete
                    </Button>
                  </div>
                </div>
              )}
            </For>
          </div>
        </div>
      </Show>
    </div>
  );
};

export default Equipment;
