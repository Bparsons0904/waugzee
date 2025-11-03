import { useFormField } from "@hooks/useFormField";
import { useValidation } from "@hooks/useValidation";
import clsx from "clsx";
import { type Component, createUniqueId, Show } from "solid-js";
import type { ValidatorFunction } from "../../../../utils/validation";
import styles from "./DateInput.module.scss";

export interface DateInputProps {
  name?: string;
  label?: string;
  value?: string;
  required?: boolean;
  disabled?: boolean;
  min?: string;
  max?: string;
  customValidators?: ValidatorFunction[];
  class?: string;
  onChange?: (value: string) => void;
  onBlur?: (event: FocusEvent) => void;
  defaultValue?: string;
}

export const DateInput: Component<DateInputProps> = (props) => {
  const id = createUniqueId();

  const validation = useValidation({
    initialValue: props.defaultValue,
    required: props.required,
    customValidators: props.customValidators,
    fieldName: props.label,
  });

  const formField = useFormField({
    name: props.name,
    required: props.required,
    initialValue: props.defaultValue,
  });

  const handleChange = (event: Event) => {
    const target = event.target as HTMLInputElement;
    const newValue = target.value;

    validation.setValue(newValue);
    props.onChange?.(newValue);
  };

  const handleBlur = (event: FocusEvent) => {
    const target = event.target as HTMLInputElement;
    const value = target.value;

    validation.markAsBlurred();
    const validationResult = validation.validate(value, true);

    if (formField.isConnectedToForm) {
      formField.updateFormField({
        isValid: validationResult.isValid,
        errorMessage: validationResult.errorMessage,
        value: value,
      });
    }

    props.onBlur?.(event);
  };

  return (
    <div class={clsx(styles.dateInputContainer, props.class)}>
      <Show when={props.label}>
        <label for={id} class={styles.label}>
          {props.label}
          <Show when={props.required}>
            <span class={styles.required}>*</span>
          </Show>
        </label>
      </Show>

      <input
        id={id}
        type="date"
        name={props.name}
        class={clsx(styles.dateInput, {
          [styles.dateInputError]: !validation.isValid(),
        })}
        value={props.value || validation.value()}
        min={props.min}
        max={props.max}
        disabled={props.disabled}
        required={props.required}
        onChange={handleChange}
        onBlur={handleBlur}
      />

      <Show when={!validation.isValid() && validation.errorMessage()}>
        <div class={styles.errorMessage} role="alert">
          {validation.errorMessage()}
        </div>
      </Show>
    </div>
  );
};
