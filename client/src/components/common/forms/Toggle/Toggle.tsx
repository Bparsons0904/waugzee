import type { Component } from "solid-js";
import styles from "./Toggle.module.scss";

export interface ToggleProps {
  checked: boolean;
  onChange: (checked: boolean) => void;
  label?: string;
  disabled?: boolean;
  name?: string;
  className?: string;
}

export const Toggle: Component<ToggleProps> = (props) => {
  const handleChange = (event: Event) => {
    const target = event.target as HTMLInputElement;
    props.onChange(target.checked);
  };

  return (
    <label class={`${styles.toggleContainer} ${props.className || ""}`}>
      <input
        type="checkbox"
        name={props.name}
        checked={props.checked}
        onChange={handleChange}
        disabled={props.disabled}
        class={styles.toggleInput}
        aria-checked={props.checked}
        role="switch"
      />
      <span class={styles.switch} aria-hidden="true" />
      {props.label && <span class={styles.label}>{props.label}</span>}
    </label>
  );
};

export default Toggle;
