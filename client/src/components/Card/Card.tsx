import type { Component, JSX } from "solid-js";
import styles from "./Card.module.scss";

export interface CardProps {
  children: JSX.Element;
  size?: "small" | "medium" | "large";
  padding?: "none" | "tight" | "normal" | "loose";
  variant?: "default" | "elevated" | "outlined";
  class?: string;
  onClick?: () => void;
}

export const Card: Component<CardProps> = (props) => {
  const sizeClass = () => {
    switch (props.size) {
      case "small":
        return styles.cardSmall;
      case "large":
        return styles.cardLarge;
      default:
        return styles.cardMedium;
    }
  };

  const paddingClass = () => {
    switch (props.padding) {
      case "none":
        return styles.paddingNone;
      case "tight":
        return styles.paddingTight;
      case "loose":
        return styles.paddingLoose;
      default:
        return styles.paddingNormal;
    }
  };

  const variantClass = () => {
    switch (props.variant) {
      case "elevated":
        return styles.cardElevated;
      case "outlined":
        return styles.cardOutlined;
      default:
        return styles.cardDefault;
    }
  };

  return (
    <div
      class={`${styles.card} ${sizeClass()} ${paddingClass()} ${variantClass()} ${
        props.class || ""
      }`}
      onClick={props.onClick}
    >
      {props.children}
    </div>
  );
};
