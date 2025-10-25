import type { Component } from "solid-js";

interface EyeIconProps {
  size?: number;
  class?: string;
}

export const EyeIcon: Component<EyeIconProps> = (props) => {
  const size = () => props.size || 24;

  return (
    <svg
      width={size()}
      height={size()}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      class={props.class}
      role="img"
      aria-label="Show password"
    >
      <title>Show password</title>
      <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
      <circle cx="12" cy="12" r="3" />
    </svg>
  );
};
