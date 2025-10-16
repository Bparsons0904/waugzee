import { Component, createSignal, For, Show } from "solid-js";
import {
  useAvailableStyluses,
  useUserStyluses,
  useCreateUserStylus,
  useCreateCustomStylus,
  useUpdateUserStylus,
  useDeleteUserStylus,
} from "@services/apiHooks";
import type {
  UserStylus,
  CreateUserStylusRequest,
  CreateCustomStylusRequest,
} from "@models/Stylus";
import styles from "./Equipment.module.scss";
import { formatDateForInput, formatLocalDate } from "@utils/dates";

type ModalMode = "add" | "custom" | "edit" | null;

const Equipment: Component = () => {
  const userStylusesQuery = useUserStyluses();
  const availableStylusesQuery = useAvailableStyluses();

  const [modalMode, setModalMode] = createSignal<ModalMode>(null);
  const [editingStylus, setEditingStylus] = createSignal<UserStylus | null>(null);

  const [selectedStylusId, setSelectedStylusId] = createSignal("");
  const [brand, setBrand] = createSignal("");
  const [model, setModel] = createSignal("");
  const [type, setType] = createSignal("");
  const [cartridgeType, setCartridgeType] = createSignal("");
  const [recommendedReplaceHours, setRecommendedReplaceHours] = createSignal<number | undefined>();
  const [purchaseDate, setPurchaseDate] = createSignal("");
  const [installDate, setInstallDate] = createSignal("");
  const [hoursUsed, setHoursUsed] = createSignal<number | undefined>();
  const [notes, setNotes] = createSignal("");
  const [isActive, setIsActive] = createSignal(false);
  const [isPrimary, setIsPrimary] = createSignal(false);

  const createUserStylusMutation = useCreateUserStylus();
  const createCustomStylusMutation = useCreateCustomStylus();
  const updateUserStylusMutation = useUpdateUserStylus();
  const deleteUserStylusMutation = useDeleteUserStylus();

  const resetForm = () => {
    setSelectedStylusId("");
    setBrand("");
    setModel("");
    setType("");
    setCartridgeType("");
    setRecommendedReplaceHours(undefined);
    setPurchaseDate("");
    setInstallDate("");
    setHoursUsed(undefined);
    setNotes("");
    setIsActive(false);
    setIsPrimary(false);
    setEditingStylus(null);
  };

  const openAddModal = () => {
    resetForm();
    setModalMode("add");
  };

  const openCustomModal = () => {
    resetForm();
    setModalMode("custom");
  };

  const openEditModal = (stylus: UserStylus) => {
    if (stylus.purchaseDate) {
      setPurchaseDate(formatDateForInput(stylus.purchaseDate));
    } else {
      setPurchaseDate("");
    }

    if (stylus.installDate) {
      setInstallDate(formatDateForInput(stylus.installDate));
    } else {
      setInstallDate("");
    }

    setHoursUsed(stylus.hoursUsed);
    setNotes(stylus.notes || "");
    setIsActive(stylus.isActive);
    setIsPrimary(stylus.isPrimary);
    setEditingStylus(stylus);
    setModalMode("edit");
  };

  const closeModal = () => {
    setModalMode(null);
    resetForm();
  };

  const handleAddEquipment = (e: Event) => {
    e.preventDefault();
    if (!selectedStylusId()) return;

    const request: CreateUserStylusRequest = {
      stylusId: selectedStylusId(),
      purchaseDate: purchaseDate() || undefined,
      installDate: installDate() || undefined,
      hoursUsed: hoursUsed(),
      notes: notes() || undefined,
      isActive: isActive(),
      isPrimary: isPrimary(),
    };

    createUserStylusMutation.mutate(request, {
      onSuccess: () => {
        closeModal();
      },
    });
  };

  const handleCreateCustom = (e: Event) => {
    e.preventDefault();
    if (!brand() || !model()) return;

    const request: CreateCustomStylusRequest = {
      brand: brand(),
      model: model(),
      type: type() || undefined,
      cartridgeType: cartridgeType() || undefined,
      recommendedReplaceHours: recommendedReplaceHours(),
    };

    createCustomStylusMutation.mutate(request, {
      onSuccess: () => {
        closeModal();
      },
    });
  };

  const handleUpdate = (e: Event) => {
    e.preventDefault();
    const editing = editingStylus();
    if (!editing) return;

    updateUserStylusMutation.mutate(
      {
        id: editing.id,
        data: {
          purchaseDate: purchaseDate() || undefined,
          installDate: installDate() || undefined,
          hoursUsed: hoursUsed(),
          notes: notes() || undefined,
          isActive: isActive(),
          isPrimary: isPrimary(),
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
  const availableStyluses = () => availableStylusesQuery.data?.styluses || [];

  return (
    <div class={styles.container}>
      <div class={styles.header}>
        <h2 class={styles.title}>Equipment Manager</h2>
        <div class={styles.headerButtons}>
          <Show when={!modalMode()}>
            <button
              class={styles.addButton}
              onClick={openAddModal}
              disabled={userStylusesQuery.isLoading}
            >
              Add Equipment
            </button>
            <button
              class={styles.customButton}
              onClick={openCustomModal}
              disabled={userStylusesQuery.isLoading}
            >
              Create Custom Stylus
            </button>
          </Show>
        </div>
      </div>

      <Show when={modalMode() === "add"}>
        <div class={styles.formContainer}>
          <h3>Add Equipment</h3>
          <form class={styles.form} onSubmit={handleAddEquipment}>
            <div class={styles.formGroup}>
              <label for="stylusSelect" class={styles.label}>
                Select Stylus *
              </label>
              <select
                id="stylusSelect"
                class={styles.input}
                value={selectedStylusId()}
                onChange={(e) => setSelectedStylusId(e.target.value)}
                required
              >
                <option value="">-- Select a stylus --</option>
                <For each={availableStyluses()}>
                  {(stylus) => (
                    <option value={stylus.id}>
                      {stylus.brand} - {stylus.model}
                      {stylus.isVerified ? " ✓" : ""}
                    </option>
                  )}
                </For>
              </select>
            </div>

            <div class={styles.formGroup}>
              <label for="purchaseDate" class={styles.label}>
                Purchase Date
              </label>
              <input
                type="date"
                id="purchaseDate"
                class={styles.input}
                value={purchaseDate()}
                onInput={(e) => setPurchaseDate(e.target.value)}
              />
            </div>

            <div class={styles.formGroup}>
              <label for="installDate" class={styles.label}>
                Install Date
              </label>
              <input
                type="date"
                id="installDate"
                class={styles.input}
                value={installDate()}
                onInput={(e) => setInstallDate(e.target.value)}
              />
            </div>

            <div class={styles.formGroup}>
              <label for="hoursUsed" class={styles.label}>
                Hours Used
              </label>
              <input
                type="number"
                id="hoursUsed"
                class={styles.input}
                value={hoursUsed() || ""}
                onInput={(e) => setHoursUsed(parseInt(e.target.value) || undefined)}
                min="0"
              />
            </div>

            <div class={styles.formGroup}>
              <label for="notes" class={styles.label}>
                Notes
              </label>
              <textarea
                id="notes"
                class={styles.textarea}
                value={notes()}
                onInput={(e) => setNotes(e.target.value)}
                rows="3"
              />
            </div>

            <div class={styles.checkboxGroup}>
              <label class={styles.checkboxLabel}>
                <input
                  type="checkbox"
                  checked={isActive()}
                  onChange={(e) => setIsActive(e.target.checked)}
                />
                Active (Currently in use)
              </label>
            </div>

            <div class={styles.checkboxGroup}>
              <label class={styles.checkboxLabel}>
                <input
                  type="checkbox"
                  checked={isPrimary()}
                  onChange={(e) => setIsPrimary(e.target.checked)}
                />
                Primary (Currently installed on turntable)
              </label>
            </div>

            <div class={styles.formActions}>
              <button
                type="button"
                class={styles.cancelButton}
                onClick={closeModal}
                disabled={createUserStylusMutation.isPending}
              >
                Cancel
              </button>
              <button
                type="submit"
                class={styles.submitButton}
                disabled={createUserStylusMutation.isPending}
              >
                {createUserStylusMutation.isPending ? "Adding..." : "Add to Equipment"}
              </button>
            </div>
          </form>
        </div>
      </Show>

      <Show when={modalMode() === "custom"}>
        <div class={styles.formContainer}>
          <h3>Create Custom Stylus</h3>
          <form class={styles.form} onSubmit={handleCreateCustom}>
            <div class={styles.formGroup}>
              <label for="brand" class={styles.label}>
                Brand *
              </label>
              <input
                type="text"
                id="brand"
                class={styles.input}
                value={brand()}
                onInput={(e) => setBrand(e.target.value)}
                required
              />
            </div>

            <div class={styles.formGroup}>
              <label for="model" class={styles.label}>
                Model *
              </label>
              <input
                type="text"
                id="model"
                class={styles.input}
                value={model()}
                onInput={(e) => setModel(e.target.value)}
                required
              />
            </div>

            <div class={styles.formGroup}>
              <label for="type" class={styles.label}>
                Stylus Type
              </label>
              <select
                id="type"
                class={styles.input}
                value={type()}
                onChange={(e) => setType(e.target.value)}
              >
                <option value="">-- Select type --</option>
                <option value="Conical">Conical</option>
                <option value="Elliptical">Elliptical</option>
                <option value="Microline">Microline</option>
                <option value="Shibata">Shibata</option>
                <option value="Line Contact">Line Contact</option>
                <option value="Other">Other</option>
              </select>
            </div>

            <div class={styles.formGroup}>
              <label for="cartridgeType" class={styles.label}>
                Cartridge Type
              </label>
              <select
                id="cartridgeType"
                class={styles.input}
                value={cartridgeType()}
                onChange={(e) => setCartridgeType(e.target.value)}
              >
                <option value="">-- Select cartridge type --</option>
                <option value="Moving Magnet">Moving Magnet (MM)</option>
                <option value="Moving Coil">Moving Coil (MC)</option>
                <option value="Ceramic">Ceramic</option>
                <option value="Other">Other</option>
              </select>
            </div>

            <div class={styles.formGroup}>
              <label for="recommendedHours" class={styles.label}>
                Recommended Replace Hours
              </label>
              <input
                type="number"
                id="recommendedHours"
                class={styles.input}
                value={recommendedReplaceHours() || ""}
                onInput={(e) =>
                  setRecommendedReplaceHours(parseInt(e.target.value) || undefined)
                }
                min="0"
              />
            </div>

            <div class={styles.formActions}>
              <button
                type="button"
                class={styles.cancelButton}
                onClick={closeModal}
                disabled={createCustomStylusMutation.isPending}
              >
                Cancel
              </button>
              <button
                type="submit"
                class={styles.submitButton}
                disabled={createCustomStylusMutation.isPending}
              >
                {createCustomStylusMutation.isPending ? "Creating..." : "Create & Add"}
              </button>
            </div>
          </form>
        </div>
      </Show>

      <Show when={modalMode() === "edit"}>
        <div class={styles.formContainer}>
          <h3>Edit Equipment Details</h3>
          <form class={styles.form} onSubmit={handleUpdate}>
            <div class={styles.formGroup}>
              <label for="purchaseDate" class={styles.label}>
                Purchase Date
              </label>
              <input
                type="date"
                id="purchaseDate"
                class={styles.input}
                value={purchaseDate()}
                onInput={(e) => setPurchaseDate(e.target.value)}
              />
            </div>

            <div class={styles.formGroup}>
              <label for="installDate" class={styles.label}>
                Install Date
              </label>
              <input
                type="date"
                id="installDate"
                class={styles.input}
                value={installDate()}
                onInput={(e) => setInstallDate(e.target.value)}
              />
            </div>

            <div class={styles.formGroup}>
              <label for="hoursUsed" class={styles.label}>
                Hours Used
              </label>
              <input
                type="number"
                id="hoursUsed"
                class={styles.input}
                value={hoursUsed() || ""}
                onInput={(e) => setHoursUsed(parseInt(e.target.value) || undefined)}
                min="0"
              />
            </div>

            <div class={styles.formGroup}>
              <label for="notes" class={styles.label}>
                Notes
              </label>
              <textarea
                id="notes"
                class={styles.textarea}
                value={notes()}
                onInput={(e) => setNotes(e.target.value)}
                rows="3"
              />
            </div>

            <div class={styles.checkboxGroup}>
              <label class={styles.checkboxLabel}>
                <input
                  type="checkbox"
                  checked={isActive()}
                  onChange={(e) => setIsActive(e.target.checked)}
                />
                Active (Currently in use)
              </label>
            </div>

            <div class={styles.checkboxGroup}>
              <label class={styles.checkboxLabel}>
                <input
                  type="checkbox"
                  checked={isPrimary()}
                  onChange={(e) => setIsPrimary(e.target.checked)}
                />
                Primary (Currently installed on turntable)
              </label>
            </div>

            <div class={styles.formActions}>
              <button
                type="button"
                class={styles.cancelButton}
                onClick={closeModal}
                disabled={updateUserStylusMutation.isPending}
              >
                Cancel
              </button>
              <button
                type="submit"
                class={styles.submitButton}
                disabled={updateUserStylusMutation.isPending}
              >
                {updateUserStylusMutation.isPending ? "Updating..." : "Update"}
              </button>
            </div>
          </form>
        </div>
      </Show>

      <Show when={!modalMode() && userStylusesQuery.isLoading}>
        <p class={styles.loading}>Loading equipment...</p>
      </Show>

      <Show when={!modalMode() && !userStylusesQuery.isLoading && styluses().length === 0}>
        <p class={styles.noStyluses}>
          No equipment found. Click "Add Equipment" to select a stylus or "Create Custom Stylus" to
          add your own.
        </p>
      </Show>

      <Show when={!modalMode() && activeStyluses().length > 0}>
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
                        <strong>Cartridge Type:</strong> {stylus.stylus?.cartridgeType}
                      </p>
                    </Show>

                    <Show when={stylus.purchaseDate}>
                      <p class={styles.stylusDetail}>
                        <strong>Purchased:</strong> {formatLocalDate(stylus.purchaseDate!)}
                      </p>
                    </Show>

                    <Show when={stylus.installDate}>
                      <p class={styles.stylusDetail}>
                        <strong>Installed:</strong> {formatLocalDate(stylus.installDate!)}
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
                    <button
                      class={styles.editButton}
                      onClick={() => openEditModal(stylus)}
                      disabled={updateUserStylusMutation.isPending}
                    >
                      Edit
                    </button>
                    <button
                      class={styles.deleteButton}
                      onClick={() => handleDelete(stylus)}
                      disabled={deleteUserStylusMutation.isPending}
                    >
                      Delete
                    </button>
                  </div>
                </div>
              )}
            </For>
          </div>
        </div>
      </Show>

      <Show when={!modalMode() && inactiveStyluses().length > 0}>
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
                        <strong>Cartridge Type:</strong> {stylus.stylus?.cartridgeType}
                      </p>
                    </Show>

                    <Show when={stylus.purchaseDate}>
                      <p class={styles.stylusDetail}>
                        <strong>Purchased:</strong> {formatLocalDate(stylus.purchaseDate!)}
                      </p>
                    </Show>
                  </div>

                  <div class={styles.stylusActions}>
                    <button
                      class={styles.editButton}
                      onClick={() => openEditModal(stylus)}
                      disabled={updateUserStylusMutation.isPending}
                    >
                      Edit
                    </button>
                    <button
                      class={styles.deleteButton}
                      onClick={() => handleDelete(stylus)}
                      disabled={deleteUserStylusMutation.isPending}
                    >
                      Delete
                    </button>
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
