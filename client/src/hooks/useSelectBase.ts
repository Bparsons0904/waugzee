import { useFormField } from "@hooks/useFormField";
import { useValidation } from "@hooks/useValidation";
import { type Accessor, createEffect, createSignal, onCleanup } from "solid-js";
import type { ValidatorFunction } from "../utils/validation";

interface UseSelectBaseProps {
  name?: string;
  label?: string;
  required?: boolean;
  disabled?: boolean;
  customValidators?: ValidatorFunction[];
  onBlur?: (event: FocusEvent) => void;
}

interface UseSelectBaseReturn<T> {
  isOpen: Accessor<boolean>;
  setIsOpen: (value: boolean) => void;
  focusedIndex: Accessor<number>;
  setFocusedIndex: (value: number | ((prev: number) => number)) => void;
  validation: ReturnType<typeof useValidation>;
  formField: ReturnType<typeof useFormField>;
  handleBlur: (event: FocusEvent, value: T) => void;
  handleClickOutside: (event: MouseEvent, containerRef: HTMLDivElement | undefined) => void;
  registerClickOutsideListener: (containerRef: HTMLDivElement | undefined) => void;
}

export function useSelectBase<T>(
  props: UseSelectBaseProps,
  initialValue?: T,
  valueToString: (value: T) => string = (v) => String(v),
): UseSelectBaseReturn<T> {
  const [isOpen, setIsOpen] = createSignal(false);
  const [focusedIndex, setFocusedIndex] = createSignal(-1);

  const validation = useValidation({
    initialValue: initialValue ? valueToString(initialValue) : undefined,
    required: props.required,
    customValidators: props.customValidators,
    fieldName: props.label,
  });

  const formField = useFormField({
    name: props.name,
    required: props.required,
    initialValue: initialValue ? valueToString(initialValue) : undefined,
  });

  const handleBlur = (event: FocusEvent, value: T) => {
    setIsOpen(false);
    setFocusedIndex(-1);

    const stringValue = valueToString(value);
    validation.markAsBlurred();
    const validationResult = validation.validate(stringValue, true);

    if (formField.isConnectedToForm) {
      formField.updateFormField({
        isValid: validationResult.isValid,
        errorMessage: validationResult.errorMessage,
        value: stringValue,
      });
    }

    props.onBlur?.(event);
  };

  const handleClickOutside = (event: MouseEvent, containerRef: HTMLDivElement | undefined) => {
    if (containerRef && !containerRef.contains(event.target as Node)) {
      setIsOpen(false);
      setFocusedIndex(-1);
    }
  };

  const registerClickOutsideListener = (containerRef: HTMLDivElement | undefined) => {
    createEffect(() => {
      if (isOpen()) {
        const handler = (event: MouseEvent) => handleClickOutside(event, containerRef);
        document.addEventListener("mousedown", handler);
        onCleanup(() => {
          document.removeEventListener("mousedown", handler);
        });
      }
    });
  };

  return {
    isOpen,
    setIsOpen,
    focusedIndex,
    setFocusedIndex,
    validation,
    formField,
    handleBlur,
    handleClickOutside,
    registerClickOutsideListener,
  };
}
