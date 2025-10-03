import { Component, JSX } from "solid-js";
import styles from "./Field.module.scss";

export interface FieldProps {
  label: string;
  required?: boolean;
  children: JSX.Element;
  className?: string;
  htmlFor?: string;
}

export const Field: Component<FieldProps> = (props) => {
  return (
    <div class={`${styles.fieldContainer} ${props.className || ""}`}>
      <label class={styles.label} for={props.htmlFor}>
        {props.label}
        {props.required && <span class={styles.required}>*</span>}
      </label>
      <div class={styles.fieldContent}>{props.children}</div>
    </div>
  );
};

export default Field;
