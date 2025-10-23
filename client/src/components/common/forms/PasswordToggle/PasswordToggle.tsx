import { EyeIcon } from "@components/icons/EyeIcon";
import { EyeOffIcon } from "@components/icons/EyeOffIcon";
import { type Component, Show } from "solid-js";
import styles from "./PasswordToggle.module.scss";

interface PasswordToggleProps {
  showPassword: boolean;
  onToggle: () => void;
}

export const PasswordToggle: Component<PasswordToggleProps> = (props) => {
  return (
    <button
      type="button"
      class={styles.toggleButton}
      onClick={props.onToggle}
      aria-label={props.showPassword ? "Hide password" : "Show password"}
    >
      <Show when={props.showPassword} fallback={<EyeIcon class={styles.icon} />}>
        <EyeOffIcon class={styles.icon} />
      </Show>
    </button>
  );
};
