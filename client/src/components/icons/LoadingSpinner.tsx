import type { Component } from "solid-js";
import styles from "./LoadingSpinner.module.scss";

interface LoadingSpinnerProps {
  class?: string;
}

export const LoadingSpinner: Component<LoadingSpinnerProps> = (props) => {
  return <div class={props.class || styles.loadingSpinner}></div>;
};
