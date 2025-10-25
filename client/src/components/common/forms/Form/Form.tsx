import { FormProvider } from "@context/FormContext";
import type { Component, JSX } from "solid-js";

interface FormProps {
  children: JSX.Element;
  onSubmit?: (formData: Record<string, unknown>) => void;
  class?: string;
}

export const Form: Component<FormProps> = (props) => {
  return (
    <FormProvider onSubmit={props.onSubmit} class={props.class}>
      {props.children}
    </FormProvider>
  );
};
