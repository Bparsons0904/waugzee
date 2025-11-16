import type { Component, JSX } from "solid-js";

interface ChevronDownIconProps {
  size?: number;
  class?: string;
  classList?: JSX.CustomAttributes<HTMLElement>["classList"];
}

export const ChevronDownIcon: Component<ChevronDownIconProps> = (props) => {
  const size = () => props.size || 16;

  return (
    <svg
      width={size()}
      height={size()}
      viewBox="0 0 16 16"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      class={props.class}
      classList={props.classList}
      role="img"
      aria-label="Chevron down"
    >
      <title>Chevron down</title>
      <path
        d="M4 6L8 10L12 6"
        stroke="currentColor"
        stroke-width="1.5"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
    </svg>
  );
};
