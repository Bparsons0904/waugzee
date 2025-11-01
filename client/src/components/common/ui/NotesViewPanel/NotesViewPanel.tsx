import { formatHistoryDate } from "@utils/dates";
import { AiOutlineClose, AiTwotoneCalendar } from "solid-icons/ai";
import { ImHeadphones } from "solid-icons/im";
import clsx from "clsx";
import { type Component, Show } from "solid-js";
import styles from "./NotesViewPanel.module.scss";

export interface NotesViewPanelProps {
  isOpen: boolean;
  onClose: () => void;
  title: string;
  date: string;
  stylusName?: string;
  notes: string;
}

export const NotesViewPanel: Component<NotesViewPanelProps> = (props) => {
  return (
    <div class={clsx(styles.panelWrapper, { [styles.open]: props.isOpen })}>
      <div class={styles.overlay} onClick={props.onClose} />

      <div class={styles.panel}>
        <div class={styles.panelHeader}>
          <h2 class={styles.panelTitle}>{props.title}</h2>
          <button type="button" class={styles.closeButton} onClick={props.onClose}>
            <AiOutlineClose size={20} />
          </button>
        </div>

        <div class={styles.panelBody}>
          <div class={styles.metadata}>
            <div class={styles.metadataItem}>
              <AiTwotoneCalendar size={18} />
              <span>{formatHistoryDate(props.date)}</span>
            </div>

            <Show when={props.stylusName}>
              <div class={styles.metadataItem}>
                <ImHeadphones size={18} />
                <span>{props.stylusName}</span>
              </div>
            </Show>
          </div>

          <div class={styles.notesSection}>
            <h3 class={styles.notesSectionTitle}>Notes</h3>
            <div class={styles.notesContent}>{props.notes}</div>
          </div>
        </div>
      </div>
    </div>
  );
};
