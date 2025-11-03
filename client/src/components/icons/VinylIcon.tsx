import type { Component } from "solid-js";

interface VinylIconProps {
  size?: number;
  class?: string;
}

const VinylIcon: Component<VinylIconProps> = (props) => {
  const size = () => props.size ?? 24;

  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width={size()}
      height={size()}
      viewBox="0 0 24 24"
      fill="currentColor"
      class={props.class}
    >
      <title>Vinyl Record</title>
      {/* Outer record disc */}
      <circle cx="12" cy="12" r="10" fill="currentColor" />
      {/* Groove rings */}
      <circle cx="12" cy="12" r="9" fill="none" stroke="rgba(0,0,0,0.15)" stroke-width="0.5" />
      <circle cx="12" cy="12" r="8" fill="none" stroke="rgba(0,0,0,0.15)" stroke-width="0.5" />
      <circle cx="12" cy="12" r="7" fill="none" stroke="rgba(0,0,0,0.15)" stroke-width="0.5" />
      <circle cx="12" cy="12" r="6" fill="none" stroke="rgba(0,0,0,0.15)" stroke-width="0.5" />
      <circle cx="12" cy="12" r="5" fill="none" stroke="rgba(0,0,0,0.15)" stroke-width="0.5" />
      {/* Center label */}
      <circle cx="12" cy="12" r="3.5" fill="rgba(255,255,255,0.9)" />
      {/* Center hole */}
      <circle cx="12" cy="12" r="1.2" fill="rgba(0,0,0,0.3)" />
    </svg>
  );
};

export default VinylIcon;
