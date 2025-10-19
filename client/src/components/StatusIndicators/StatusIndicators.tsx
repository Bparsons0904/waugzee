import { formatLocalDate } from "@utils/dates";
import {
  countPlaysSinceCleaning,
  getCleanlinessColor,
  getCleanlinessScore,
  getCleanlinessText,
  getLastCleaningDate,
  getLastPlayDate,
  getPlayRecencyColor,
  getPlayRecencyScore,
  getPlayRecencyText,
} from "@utils/playStatus";
import { ImHeadphones } from "solid-icons/im";
import { TbWashTemperature5 } from "solid-icons/tb";
import { type Component, Show } from "solid-js";
import styles from "./StatusIndicators.module.scss";

export interface StatusIndicatorProps {
  playHistory?: { playedAt: string }[];
  cleaningHistory?: { cleanedAt: string }[];
  showDetails?: boolean;
  onPlayClick?: () => void;
  onCleanClick?: () => void;
}

export const RecordStatusIndicator: Component<StatusIndicatorProps> = (props) => {
  const lastPlayDate = () => getLastPlayDate(props.playHistory);
  const lastCleanDate = () => getLastCleaningDate(props.cleaningHistory);

  const playsSinceCleaning = () =>
    countPlaysSinceCleaning(props.playHistory || [], lastCleanDate());

  const cleanlinessScore = () => getCleanlinessScore(lastCleanDate(), playsSinceCleaning());

  const playRecencyScore = () => getPlayRecencyScore(lastPlayDate());

  return (
    <div class={styles.container}>
      <Show when={!props.showDetails}>
        <div class={styles.indicatorRow}>
          <PlayStatusIndicator
            score={playRecencyScore()}
            lastPlayed={lastPlayDate()}
            onClick={props.onPlayClick}
          />
          <CleaningStatusIndicator
            score={cleanlinessScore()}
            lastCleaned={lastCleanDate()}
            playsSinceCleaning={playsSinceCleaning()}
            onClick={props.onCleanClick}
          />
        </div>
      </Show>

      <Show when={props.showDetails}>
        <div class={styles.detailsSection}>
          <div class={styles.detailRow}>
            <span class={styles.detailLabel}>Last played:</span>
            <span class={styles.detailValue}>{formatLocalDate(lastPlayDate())}</span>
          </div>
          <div class={styles.detailRow}>
            <span class={styles.detailLabel}>Last cleaned:</span>
            <span class={styles.detailValue}>{formatLocalDate(lastCleanDate())}</span>
          </div>
          <div class={styles.detailRow}>
            <span class={styles.detailLabel}>Plays since cleaning:</span>
            <span class={styles.detailValue}>{playsSinceCleaning()}</span>
          </div>
        </div>
      </Show>
    </div>
  );
};

interface PlayStatusProps {
  score: number;
  lastPlayed: Date | null;
  showDetails?: boolean;
  onClick?: () => void;
}

// Claude are both of the indicators needed?  Seems like there is a lot of shared logic between the 2 of them
const PlayStatusIndicator: Component<PlayStatusProps> = (props) => {
  const getColorWithOpacity = (colorHex: string): string => {
    return colorHex + "CC";
  };

  const color = () => getColorWithOpacity(getPlayRecencyColor(props.score));
  const text = () => getPlayRecencyText(props.lastPlayed);

  const handleClick = (e: MouseEvent) => {
    if (props.onClick) {
      e.stopPropagation();
      props.onClick();
    }
  };

  return (
    <div class={styles.indicator}>
      <div
        class={`${styles.iconContainer} ${props.onClick ? styles.clickable : ""}`}
        style={{ "background-color": color() }}
        onClick={handleClick}
        title="Click to log a play"
      >
        <ImHeadphones size={15} color="white" />
      </div>
      <span class={styles.tooltipText}>{text()}</span>
    </div>
  );
};

interface CleaningStatusProps {
  score: number;
  lastCleaned: Date | null;
  playsSinceCleaning: number;
  showDetails?: boolean;
  onClick?: () => void;
}

const CleaningStatusIndicator: Component<CleaningStatusProps> = (props) => {
  const getColorWithOpacity = (colorHex: string): string => {
    return colorHex + "CC";
  };

  const color = () => getColorWithOpacity(getCleanlinessColor(props.score));
  const text = () => getCleanlinessText(props.score);

  const handleClick = (e: MouseEvent) => {
    if (props.onClick) {
      e.stopPropagation();
      props.onClick();
    }
  };

  return (
    <div class={styles.indicator}>
      <div
        class={`${styles.iconContainer} ${props.onClick ? styles.clickable : ""}`}
        style={{ "background-color": color() }}
        onClick={handleClick}
        title="Click to log a cleaning"
      >
        <TbWashTemperature5 size={20} color="white" />
      </div>
      <span class={styles.tooltipText}>{text()}</span>
    </div>
  );
};
