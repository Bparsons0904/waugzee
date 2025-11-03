import type { Component, JSX } from "solid-js";
import styles from "./Card.module.scss";

export type CardVariant = "default" | "success" | "warning" | "primary" | "secondary" | "error";

interface CardProps {
  variant?: CardVariant;
  children: JSX.Element;
  class?: string;
  onClick?: () => void;
}

export const Card: Component<CardProps> = (props) => {
  const variant = props.variant || "default";

  const cardClass = () => {
    const baseClass = styles.card;
    const variantClass = styles[variant];
    const customClass = props.class || "";
    const clickableClass = props.onClick ? styles.clickable : "";
    return `${baseClass} ${variantClass} ${customClass} ${clickableClass}`.trim();
  };

  return (
    <div class={cardClass()} onClick={props.onClick}>
      {props.children}
    </div>
  );
};
