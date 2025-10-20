import { AlertCircleIcon } from "@components/icons/AlertCircleIcon";
import { AlertTriangleIcon } from "@components/icons/AlertTriangleIcon";
import { CheckCircleIcon } from "@components/icons/CheckCircleIcon";
import clsx from "clsx";
import type { Component } from "solid-js";
import { Button, type ButtonVariant } from "../Button/Button";
import { Modal, ModalSize } from "../Modal/Modal";
import styles from "./ConfirmPopup.module.scss";

export interface ConfirmPopupProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: () => void;
  title?: string;
  description?: string;
  confirmText?: string;
  cancelText?: string;
  variant?: ButtonVariant;
  size?: ModalSize;
}

export const ConfirmPopup: Component<ConfirmPopupProps> = (props) => {
  const getIcon = () => {
    switch (props.variant) {
      case "danger":
        return <AlertCircleIcon class={styles.icon} />;
      case "warning":
        return <AlertTriangleIcon class={styles.icon} />;
      default:
        return <CheckCircleIcon class={styles.icon} />;
    }
  };

  return (
    <Modal
      isOpen={props.isOpen}
      onClose={props.onClose}
      title={props.title || "Confirm Action"}
      size={props.size || ModalSize.Small}
    >
      <div class={styles.confirmPopup}>
        <div class={clsx(styles.iconContainer, styles[props.variant || "primary"])}>
          {getIcon()}
        </div>

        <div class={styles.content}>
          <p class={styles.description}>
            {props.description || "Are you sure you want to proceed with this action?"}
          </p>
        </div>

        <div class={styles.actions}>
          <Button variant="tertiary" onClick={props.onClose}>
            {props.cancelText || "Cancel"}
          </Button>
          <Button variant={props.variant || "primary"} onClick={props.onConfirm}>
            {props.confirmText || "Confirm"}
          </Button>
        </div>
      </div>
    </Modal>
  );
};
