import { Component, Show, createSignal } from "solid-js";
import styles from "./RecordActionModal.module.scss";
import { UserRelease } from "@models/User";
import { Stylus } from "@models/Release";
import { formatDateForInput } from "@utils/dates";
import { Image } from "@components/common/ui/Image/Image";
import {
  SearchableSelect,
  SearchableSelectOption,
} from "@components/common/forms/SearchableSelect/SearchableSelect";

interface RecordActionModalProps {
  isOpen: boolean;
  onClose: () => void;
  release: UserRelease;
}

const RecordActionModal: Component<RecordActionModalProps> = (props) => {
  const [date, setDate] = createSignal(formatDateForInput(new Date()));
  const [selectedStylusId, setSelectedStylusId] = createSignal<number | null>(
    null,
  );
  const [notes, setNotes] = createSignal("");

  const mockStyluses: Stylus[] = [
    {
      id: 1,
      name: "Ortofon 2M Blue",
      manufacturer: "Ortofon",
      active: true,
      primary: true,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    },
    {
      id: 2,
      name: "AT-VM95E",
      manufacturer: "Audio-Technica",
      active: true,
      primary: false,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    },
  ];

  const handleLogPlay = () => {
    console.log("Log Play:", {
      releaseId: props.release.releaseId,
      releaseTitle: props.release.release.title,
      date: date(),
      stylusId: selectedStylusId(),
      notes: notes(),
    });
    setNotes("");
  };

  const handleLogCleaning = () => {
    console.log("Log Cleaning:", {
      releaseId: props.release.releaseId,
      releaseTitle: props.release.release.title,
      date: date(),
      notes: notes(),
    });
    setNotes("");
  };

  const handleLogBoth = () => {
    console.log("Log Both:", {
      releaseId: props.release.releaseId,
      releaseTitle: props.release.release.title,
      date: date(),
      stylusId: selectedStylusId(),
      notes: notes(),
    });
    setNotes("");
  };

  const stylusOptions = (): SearchableSelectOption[] => [
    { value: "", label: "None" },
    ...mockStyluses.map((s) => ({
      value: s.id.toString(),
      label: `${s.manufacturer} ${s.name}`,
      metadata: s.primary ? "Primary" : undefined,
    })),
  ];

  return (
    <Show when={props.isOpen}>
      <div class={styles.modalOverlay} onClick={props.onClose}>
        <div class={styles.modal} onClick={(e) => e.stopPropagation()}>
          <div class={styles.modalHeader}>
            <button class={styles.closeButton} onClick={props.onClose}>
              ×
            </button>
            <h2 class={styles.modalTitle}>Record Actions</h2>
          </div>

          <div class={styles.recordDetails}>
            <div class={styles.recordImage}>
              <Image
                src={props.release.release.thumb || ""}
                alt={props.release.release.title || "Release"}
                aspectRatio="square"
                showSkeleton={true}
              />
            </div>
            <div class={styles.recordInfo}>
              <h3 class={styles.recordTitle}>{props.release.release.title}</h3>
              <p class={styles.recordArtist}>
                {props.release.release.format || "Unknown Format"}
              </p>
              {props.release.release.year && (
                <p class={styles.recordYear}>{props.release.release.year}</p>
              )}
            </div>
          </div>

          <div class={styles.formSection}>
            <div class={styles.formRow}>
              <div class={styles.formGroup}>
                <label class={styles.label} for="actionDate">
                  Date
                </label>
                <input
                  type="date"
                  id="actionDate"
                  class={styles.dateInput}
                  value={date()}
                  onInput={(e) => setDate(e.target.value)}
                />
              </div>

              <div class={styles.formGroup}>
                <SearchableSelect
                  label="Stylus Used"
                  name="stylusSelect"
                  placeholder="Select a stylus"
                  searchPlaceholder="Search styluses..."
                  options={stylusOptions()}
                  value={selectedStylusId()?.toString() || ""}
                  onChange={(val) =>
                    setSelectedStylusId(val ? parseInt(val) : null)
                  }
                  emptyMessage="No styluses found"
                />
              </div>
            </div>

            <div class={styles.formGroup}>
              <label class={styles.label} for="notes">
                Notes
              </label>
              <textarea
                id="notes"
                class={styles.textarea}
                value={notes()}
                onInput={(e) => setNotes(e.target.value)}
                placeholder="Enter any notes about this play or cleaning..."
                rows="3"
              />
            </div>
          </div>

          <div class={styles.actionButtons}>
            <button class={styles.playButton} onClick={handleLogPlay}>
              Log Play
            </button>
            <button class={styles.bothButton} onClick={handleLogBoth}>
              Log Both
            </button>
            <button class={styles.cleaningButton} onClick={handleLogCleaning}>
              Log Cleaning
            </button>
          </div>

          <div class={styles.historySection}>
            <h3 class={styles.historyTitle}>Record History</h3>

            <div class={styles.historyList}>
              <div class={styles.noHistory}>
                No play or cleaning history for this record yet.
              </div>
            </div>
          </div>
        </div>
      </div>
    </Show>
  );
};

export default RecordActionModal;
