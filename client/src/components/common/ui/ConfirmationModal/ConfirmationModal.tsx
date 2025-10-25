import { type Component, Show } from "solid-js";
import styles from "./ConfirmationModal.module.scss";

export interface ConfirmationModalProps {
  isOpen: boolean;
  title: string;
  message: string;
  confirmText?: string;
  cancelText?: string;
  onConfirm: () => void;
  onCancel: () => void;
}

export const ConfirmationModal: Component<ConfirmationModalProps> = (props) => {
  const confirmText = () => props.confirmText || "Confirm";
  const cancelText = () => props.cancelText || "Cancel";

  return (
    <Show when={props.isOpen}>
      <div class={styles.modalOverlay} onClick={props.onCancel}>
        <div class={styles.modal} onClick={(e) => e.stopPropagation()}>
          <div class={styles.modalHeader}>
            <h2 class={styles.modalTitle}>{props.title}</h2>
            <button type="button" class={styles.closeButton} onClick={props.onCancel}>
              ×
            </button>
          </div>

          <div class={styles.modalBody}>
            <p class={styles.message}>{props.message}</p>
          </div>

          <div class={styles.modalFooter}>
            <button type="button" class={styles.cancelButton} onClick={props.onCancel}>
              {cancelText()}
            </button>
            <button
              type="button"
              class={`${styles.confirmButton} ${styles.destructive}`}
              onClick={props.onConfirm}
            >
              {confirmText()}
            </button>
          </div>
        </div>
      </div>
    </Show>
  );
};
