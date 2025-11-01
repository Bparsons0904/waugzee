import type { PlayHistory } from "@models/Release";
import type { UserRelease } from "@models/User";
import { TbWashTemperature5 } from "solid-icons/tb";
import { VsNote } from "solid-icons/vs";
import clsx from "clsx";
import { type Component, Show } from "solid-js";
import styles from "./PlayHistory.module.scss";

interface PlayItemCardProps {
  play: PlayHistory;
  release: UserRelease | undefined;
  hasCleaning: boolean;
  onClick: () => void;
}

export const PlayItemCard: Component<PlayItemCardProps> = (props) => {
  const thumb = () => props.release?.release?.thumb;
  const title = () => props.release?.release?.title || "Unknown Album";
  const artists = () => props.release?.release?.artists || [];

  return (
    <div class={styles.playItem} onClick={props.onClick}>
      <div class={styles.albumArt}>
        {thumb() ? (
          <img src={thumb()} alt={title()} class={styles.albumImage} />
        ) : (
          <div class={styles.noImage}>No Image</div>
        )}
      </div>

      <div class={styles.playDetails}>
        <h3 class={styles.albumTitle}>{title()}</h3>
        <p class={styles.artistName}>
          {artists()
            .map((artist) => artist.name)
            .join(", ") || "Unknown Artist"}
        </p>
      </div>

      <div class={styles.indicators}>
        <Show when={props.play.notes && props.play.notes.trim() !== ""}>
          <span class={clsx(styles.indicator, styles.hasNotes)}>
            <VsNote size={14} />
            <span>Notes</span>
          </span>
        </Show>

        <Show when={props.hasCleaning}>
          <span class={clsx(styles.indicator, styles.hasCleaning)}>
            <TbWashTemperature5 size={14} />
            <span>Cleaned</span>
          </span>
        </Show>
      </div>
    </div>
  );
};
