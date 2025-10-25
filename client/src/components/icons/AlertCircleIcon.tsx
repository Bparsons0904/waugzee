import type { Component } from "solid-js";
import { createSignal } from "solid-js";

interface AlertCircleIconProps {
  size?: number;
  class?: string;
}

export const AlertCircleIcon: Component<AlertCircleIconProps> = (props) => {
  const [size] = createSignal(props.size || 24);

  return (
    <svg
      width={size()}
      height={size()}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      stroke-width="2"
      stroke-linecap="round"
      stroke-linejoin="round"
      class={props.class}
      role="img"
      aria-label="Alert icon"
    >
      <title>Alert</title>
      <circle cx="12" cy="12" r="10" />
      <line x1="15" y1="9" x2="9" y2="15" />
      <line x1="9" y1="9" x2="15" y2="15" />
    </svg>
  );
};
