import type { Component } from "solid-js";

interface XIconProps {
  size?: number;
  class?: string;
}

export const XIcon: Component<XIconProps> = (props) => {
  const size = () => props.size || 24;

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
      aria-label="Close"
    >
      <title>Close</title>
      <line x1="18" y1="6" x2="6" y2="18" />
      <line x1="6" y1="6" x2="18" y2="18" />
    </svg>
  );
};
