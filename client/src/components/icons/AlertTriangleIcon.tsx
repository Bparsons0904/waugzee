import type { Component } from "solid-js";
import { createSignal } from "solid-js";

interface AlertTriangleIconProps {
  size?: number;
  class?: string;
}

export const AlertTriangleIcon: Component<AlertTriangleIconProps> = (props) => {
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
      aria-label="Warning icon"
    >
      <title>Warning</title>
      <path d="m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3Z" />
      <line x1="12" y1="9" x2="12" y2="13" />
      <line x1="12" y1="17" x2="12.01" y2="17" />
    </svg>
  );
};
