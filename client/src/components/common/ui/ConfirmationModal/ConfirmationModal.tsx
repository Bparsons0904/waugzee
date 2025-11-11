import { Button } from "@components/common/ui/Button/Button";
import { Modal, ModalSize } from "@components/common/ui/Modal/Modal";
import { type Component, createEffect, onCleanup } from "solid-js";
import styles from "./ConfirmationModal.module.scss";

export interface ConfirmationModalProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: () => void;
  title: string;
  message: string;
  confirmText?: string;
  cancelText?: string;
  variant?: "default" | "danger";
  isLoading?: boolean;
}

export const ConfirmationModal: Component<ConfirmationModalProps> = (props) => {
  const confirmText = () => props.confirmText || "Confirm";
  const cancelText = () => props.cancelText || "Cancel";
  const variant = () => props.variant || "default";

  createEffect(() => {
    if (props.isOpen && !props.isLoading) {
      const handleEnter = (e: KeyboardEvent) => {
        if (e.key === "Enter") {
          props.onConfirm();
        }
      };

      window.addEventListener("keydown", handleEnter);

      onCleanup(() => {
        window.removeEventListener("keydown", handleEnter);
      });
    }
  });

  return (
    <Modal
      isOpen={props.isOpen}
      onClose={props.onClose}
      title={props.title}
      size={ModalSize.Small}
      closeOnEscape={!props.isLoading}
      closeOnBackdropClick={!props.isLoading}
    >
      <div class={styles.content}>
        <p class={styles.message}>{props.message}</p>

        <div class={styles.buttonContainer}>
          <Button variant="secondary" onClick={props.onClose} disabled={props.isLoading}>
            {cancelText()}
          </Button>

          <Button
            variant={variant() === "danger" ? "danger" : "primary"}
            onClick={props.onConfirm}
            disabled={props.isLoading}
          >
            {props.isLoading ? "Processing..." : confirmText()}
          </Button>
        </div>
      </div>
    </Modal>
  );
};
