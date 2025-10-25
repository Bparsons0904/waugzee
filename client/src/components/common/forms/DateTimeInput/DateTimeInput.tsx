import { useFormField } from "@hooks/useFormField";
import { useValidation } from "@hooks/useValidation";
import { type Component, createUniqueId, Show } from "solid-js";
import type { ValidatorFunction } from "../../../../utils/validation";
import styles from "./DateTimeInput.module.scss";

export interface DateTimeInputProps {
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

export const DateTimeInput: Component<DateTimeInputProps> = (props) => {
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
    <div class={`${styles.dateTimeInputContainer} ${props.class || ""}`}>
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
        type="datetime-local"
        name={props.name}
        class={`${styles.dateTimeInput} ${!validation.isValid() ? styles.dateTimeInputError : ""}`}
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
