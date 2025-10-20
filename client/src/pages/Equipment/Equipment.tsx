import AddEquipmentModal from "@components/AddEquipmentModal/AddEquipmentModal";
import CreateCustomStylusModal from "@components/CreateCustomStylusModal/CreateCustomStylusModal";
import { Button } from "@components/common/ui/Button/Button";
import { Modal, ModalSize } from "@components/common/ui/Modal/Modal";
import EditEquipmentModal from "@components/EditEquipmentModal/EditEquipmentModal";
import type { UserStylus } from "@models/Stylus";
import { useDeleteUserStylus, useUserStyluses } from "@services/apiHooks";
import { formatLocalDate } from "@utils/dates";
import { type Component, For, Show } from "solid-js";
import { createStore } from "solid-js/store";
import styles from "./Equipment.module.scss";

type ModalMode = "add" | "custom" | "edit" | null;

const Equipment: Component = () => {
  const userStylusesQuery = useUserStyluses();
  const deleteUserStylusMutation = useDeleteUserStylus();

  const [modalMode, setModalMode] = createStore<{ current: ModalMode }>({
    current: null,
  });

  const [editingStylus, setEditingStylus] = createStore<{
    stylus: UserStylus | null;
  }>({
    stylus: null,
  });

  const openAddModal = () => {
    setModalMode("current", "add");
  };

  const openCustomModal = () => {
    setModalMode("current", "custom");
  };

  const openEditModal = (stylus: UserStylus) => {
    setEditingStylus("stylus", stylus);
    setModalMode("current", "edit");
  };

  const closeModal = () => {
    setModalMode("current", null);
    setEditingStylus("stylus", null);
  };

  const handleDelete = (stylus: UserStylus) => {
    const displayName = stylus.stylus ? `${stylus.stylus.brand} ${stylus.stylus.model}` : "Unknown";

    if (!confirm(`Are you sure you want to remove "${displayName}" from your equipment?`)) {
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
            <Button variant="primary" onClick={openAddModal} disabled={userStylusesQuery.isLoading}>
              Add Stylus
            </Button>
          </Show>
        </div>
      </div>

      <Modal
        isOpen={modalMode.current === "add"}
        onClose={closeModal}
        title="Add Stylus"
        size={ModalSize.Medium}
      >
        <AddEquipmentModal onClose={closeModal} onOpenCustomModal={openCustomModal} />
      </Modal>

      <Modal
        isOpen={modalMode.current === "custom"}
        onClose={closeModal}
        title="Create Custom Stylus"
        size={ModalSize.Medium}
      >
        <CreateCustomStylusModal onClose={closeModal} />
      </Modal>

      <Modal
        isOpen={modalMode.current === "edit"}
        onClose={closeModal}
        title="Edit Equipment Details"
        size={ModalSize.Medium}
      >
        <EditEquipmentModal stylus={editingStylus.stylus as never} onClose={closeModal} />
      </Modal>

      <Show when={!modalMode.current && userStylusesQuery.isLoading}>
        <p class={styles.loading}>Loading equipment...</p>
      </Show>

      <Show when={!modalMode.current && !userStylusesQuery.isLoading && styluses().length === 0}>
        <p class={styles.noStyluses}>
          No styluses found. Click "Add Stylus" to get started.
        </p>
      </Show>

      <Show when={!modalMode.current && activeStyluses().length > 0}>
        <div class={styles.section}>
          <h3 class={styles.sectionTitle}>Active Styluses</h3>
          <div class={styles.stylusList}>
            <For each={activeStyluses()}>
              {(stylus) => (
                <StylusCard
                  stylus={stylus}
                  showFullDetails={true}
                  onEdit={openEditModal}
                  onDelete={handleDelete}
                  deleteDisabled={deleteUserStylusMutation.isPending}
                />
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
                <StylusCard
                  stylus={stylus}
                  showFullDetails={false}
                  onEdit={openEditModal}
                  onDelete={handleDelete}
                  deleteDisabled={deleteUserStylusMutation.isPending}
                />
              )}
            </For>
          </div>
        </div>
      </Show>
    </div>
  );
};

export default Equipment;

interface StylusCardProps {
  stylus: UserStylus;
  showFullDetails?: boolean;
  onEdit: (stylus: UserStylus) => void;
  onDelete: (stylus: UserStylus) => void;
  deleteDisabled?: boolean;
}

const StylusCard: Component<StylusCardProps> = (props) => {
  const getStylusDisplayName = () => {
    if (!props.stylus.stylus) return "Unknown";
    return `${props.stylus.stylus.brand} ${props.stylus.stylus.model}`;
  };

  return (
    <div class={styles.stylusCard}>
      <div class={styles.stylusInfo}>
        <h3 class={styles.stylusName}>
          {getStylusDisplayName()}
          <Show when={props.showFullDetails}>
            <div class={styles.tagsContainer}>
              <span class={styles.activeTag}>Active</span>
              <Show when={props.stylus.isPrimary}>
                <span class={styles.primaryTag}>Primary</span>
              </Show>
            </div>
          </Show>
        </h3>

        <Show when={props.stylus.stylus?.type}>
          <p class={styles.stylusDetail}>
            <strong>Stylus Type:</strong> {props.stylus.stylus?.type}
          </p>
        </Show>

        <Show when={props.stylus.stylus?.cartridgeType}>
          <p class={styles.stylusDetail}>
            <strong>Cartridge Type:</strong> {props.stylus.stylus?.cartridgeType}
          </p>
        </Show>

        <Show when={props.stylus.purchaseDate}>
          <p class={styles.stylusDetail}>
            <strong>Purchased:</strong> {formatLocalDate(props.stylus.purchaseDate as string)}
          </p>
        </Show>

        <Show when={props.showFullDetails && props.stylus.installDate}>
          <p class={styles.stylusDetail}>
            <strong>Installed:</strong> {formatLocalDate(props.stylus.installDate as string)}
          </p>
        </Show>

        <Show when={props.showFullDetails && props.stylus.hoursUsed}>
          <p class={styles.stylusDetail}>
            <strong>Hours Used:</strong> {props.stylus.hoursUsed}
          </p>
        </Show>

        <Show when={props.showFullDetails && props.stylus.notes}>
          <p class={styles.stylusDetail}>
            <strong>Notes:</strong> {props.stylus.notes}
          </p>
        </Show>
      </div>

      <div class={styles.stylusActions}>
        <Button variant="secondary" size="sm" onClick={() => props.onEdit(props.stylus)}>
          Edit
        </Button>
        <Button
          variant="danger"
          size="sm"
          onClick={() => props.onDelete(props.stylus)}
          disabled={props.deleteDisabled}
        >
          Delete
        </Button>
      </div>
    </div>
  );
};
