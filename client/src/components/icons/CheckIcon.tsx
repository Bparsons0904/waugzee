import type { Component } from "solid-js";

interface CheckIconProps {
  size?: number;
  class?: string;
}

export const CheckIcon: Component<CheckIconProps> = (props) => {
  const size = () => props.size || 12;

  return (
    <svg
      width={size()}
      height={size()}
      viewBox="0 0 12 12"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      class={props.class}
    >
      <path
        d="M2 6L5 9L10 3"
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
    </svg>
  );
};
