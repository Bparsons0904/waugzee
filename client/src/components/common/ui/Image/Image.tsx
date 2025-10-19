import { type Component, createSignal, Show } from "solid-js";
import styles from "./Image.module.scss";

export interface ImageProps {
  src: string;
  alt: string;
  fallback?: string;
  loading?: "lazy" | "eager";
  showSkeleton?: boolean;
  className?: string;
  aspectRatio?: "square" | "album" | "wide" | "hero";
  sizes?: string;
  width?: number;
  height?: number;
  onClick?: () => void;
  smoothFallback?: boolean; // Whether to reset opacity when falling back
}

export const Image: Component<ImageProps> = (props) => {
  const [loaded, setLoaded] = createSignal(false);
  const [error, setError] = createSignal(false);
  const [imageError, setImageError] = createSignal(false);

  const handleLoad = () => {
    setLoaded(true);
    setImageError(false);
  };

  const handleError = () => {
    if (!error() && props.fallback) {
      setError(true);
      if (props.smoothFallback !== false) {
        setLoaded(false); // Reset loaded state when switching to fallback (default behavior)
      }
    } else {
      setImageError(true);
      setLoaded(true);
    }
  };

  const getAspectRatioClass = () => {
    switch (props.aspectRatio) {
      case "square":
        return styles.aspectSquare;
      case "album":
        return styles.aspectAlbum;
      case "wide":
        return styles.aspectWide;
      case "hero":
        return styles.aspectHero;
      default:
        return "";
    }
  };

  const currentSrc = () => {
    if (imageError()) return "/images/placeholders/image-error.svg";
    if (error() && props.fallback) return props.fallback;
    return props.src;
  };

  return (
    <div
      class={`${styles.imageContainer} ${getAspectRatioClass()} ${props.className || ""}`}
      onClick={props.onClick}
    >
      <Show when={props.showSkeleton && !loaded()}>
        <div class={styles.skeleton} />
      </Show>

      <Show when={!imageError()}>
        <img
          src={currentSrc()}
          alt={props.alt}
          loading={props.loading || "lazy"}
          width={props.width}
          height={props.height}
          sizes={props.sizes}
          onLoad={handleLoad}
          onError={handleError}
          class={`${styles.image} ${loaded() ? styles.loaded : ""}`}
        />
      </Show>

      <Show when={imageError()}>
        <div class={styles.errorState}>
          <div class={styles.errorIcon}>📷</div>
          <p class={styles.errorText}>Image not available</p>
        </div>
      </Show>
    </div>
  );
};

export default Image;
