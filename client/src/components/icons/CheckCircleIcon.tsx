import type { Component } from "solid-js";
import { createSignal } from "solid-js";

interface CheckCircleIconProps {
  size?: number;
  class?: string;
}

export const CheckCircleIcon: Component<CheckCircleIconProps> = (props) => {
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
      aria-label="Check icon"
    >
      <title>Check</title>
      <circle cx="12" cy="12" r="10" />
      <path d="m9 12 2 2 4-4" />
    </svg>
  );
};
