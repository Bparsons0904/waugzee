import { CheckIcon } from "@components/icons/CheckIcon";
import { ChevronDownIcon } from "@components/icons/ChevronDownIcon";
import { useSelectBase } from "@hooks/useSelectBase";
import clsx from "clsx";
import { type Component, createEffect, createSignal, createUniqueId, For, Show } from "solid-js";
import type { ValidatorFunction } from "../../../../utils/validation";
import styles from "./MultiSelect.module.scss";

export interface MultiSelectOption {
  value: string;
  label: string;
  disabled?: boolean;
}

export interface MultiSelectProps {
  name?: string;
  label?: string;
  placeholder?: string;
  options: MultiSelectOption[];
  value?: string[];
  required?: boolean;
  disabled?: boolean;
  customValidators?: ValidatorFunction[];
  class?: string;
  onChange?: (values: string[]) => void;
  onBlur?: (event: FocusEvent) => void;
  defaultValue?: string[];
  maxSelections?: number;
}

export const MultiSelect: Component<MultiSelectProps> = (props) => {
  const id = createUniqueId();
  const [selectedValues, setSelectedValues] = createSignal<string[]>(
    props.value || props.defaultValue || [],
  );

  createEffect(() => {
    if (props.value !== undefined) {
      setSelectedValues(props.value);
    }
  });

  const { isOpen, setIsOpen, focusedIndex, setFocusedIndex, validation, formField } = useSelectBase(
    {
      name: props.name,
      label: props.label,
      required: props.required,
      disabled: props.disabled,
      customValidators: props.customValidators,
      onBlur: props.onBlur,
    },
    props.value || props.defaultValue,
    (values) => values.join(","),
  );

  const handleOptionToggle = (optionValue: string) => {
    if (props.disabled) return;

    const current = selectedValues();
    const isSelected = current.includes(optionValue);

    let newValues: string[];
    if (isSelected) {
      newValues = current.filter((v) => v !== optionValue);
    } else {
      if (props.maxSelections && current.length >= props.maxSelections) {
        return;
      }
      newValues = [...current, optionValue];
    }

    setSelectedValues(newValues);
    validation.setValue(newValues.join(","));
    props.onChange?.(newValues);

    if (formField.isConnectedToForm) {
      const validationResult = validation.validate(newValues.join(","), true);
      formField.updateFormField({
        isValid: validationResult.isValid,
        errorMessage: validationResult.errorMessage,
        value: newValues.join(","),
      });
    }
  };

  const handleBlur = (event: FocusEvent) => {
    const relatedTarget = event.relatedTarget as HTMLElement;
    if (relatedTarget?.closest("[data-multiselect-options]")) return;

    setIsOpen(false);
    setFocusedIndex(-1);

    const stringValue = selectedValues().join(",");
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

  const handleKeyDown = (event: KeyboardEvent) => {
    if (props.disabled) return;

    const availableOptions = props.options.filter((option) => !option.disabled);
    const maxIndex = availableOptions.length - 1;

    switch (event.key) {
      case "Enter":
      case " ":
        event.preventDefault();
        if (!isOpen()) {
          setIsOpen(true);
          setFocusedIndex(0);
        } else if (focusedIndex() >= 0 && focusedIndex() <= maxIndex) {
          const option = availableOptions[focusedIndex()];
          handleOptionToggle(option.value);
        }
        break;

      case "Escape":
        event.preventDefault();
        setIsOpen(false);
        setFocusedIndex(-1);
        break;

      case "ArrowDown":
        event.preventDefault();
        if (!isOpen()) {
          setIsOpen(true);
          setFocusedIndex(0);
        } else {
          setFocusedIndex((prev) => Math.min(prev + 1, maxIndex));
        }
        break;

      case "ArrowUp":
        event.preventDefault();
        if (isOpen()) {
          setFocusedIndex((prev) => Math.max(prev - 1, 0));
        }
        break;

      case "Tab":
        if (isOpen()) {
          setIsOpen(false);
          setFocusedIndex(-1);
        }
        break;
    }
  };

  const getSelectedLabels = () => {
    const selected = selectedValues();
    return props.options
      .filter((option) => selected.includes(option.value))
      .map((option) => option.label);
  };

  const getDisplayText = () => {
    const selectedLabels = getSelectedLabels();

    switch (selectedLabels.length) {
      case 0:
        return props.placeholder || "Select options...";
      case 1:
        return selectedLabels[0];
      default:
        return `${selectedLabels.length} items selected`;
    }
  };

  return (
    <div class={clsx(styles.multiSelectContainer, props.class)}>
      <Show when={props.label}>
        <label for={id} class={styles.label}>
          {props.label}
          <Show when={props.required}>
            <span class={styles.required}>*</span>
          </Show>
        </label>
      </Show>

      <div class={styles.multiSelectWrapper}>
        <div
          class={clsx(styles.multiSelectTrigger, {
            [styles.selectError]: !validation.isValid(),
            [styles.open]: isOpen(),
          })}
          onClick={() => !props.disabled && setIsOpen(!isOpen())}
          onBlur={handleBlur}
          onKeyDown={handleKeyDown}
          tabIndex={0}
          role="combobox"
          aria-expanded={isOpen()}
          aria-haspopup="listbox"
          aria-owns={`${id}-listbox`}
        >
          <span
            class={clsx({
              [styles.placeholder]: selectedValues().length === 0,
              [styles.selectedText]: selectedValues().length > 0,
            })}
          >
            {getDisplayText()}
          </span>

          <div class={styles.selectIcon}>
            <ChevronDownIcon class={clsx({ [styles.iconRotated]: isOpen() })} />
          </div>
        </div>

        <Show when={isOpen()}>
          <div
            class={styles.optionsContainer}
            data-multiselect-options
            id={`${id}-listbox`}
            role="listbox"
            aria-multiselectable="true"
          >
            <For each={props.options}>
              {(option) => (
                <MultiSelectOptionItem
                  option={option}
                  allOptions={props.options}
                  isSelected={selectedValues().includes(option.value)}
                  focusedIndex={focusedIndex()}
                  onClick={() => handleOptionToggle(option.value)}
                />
              )}
            </For>
          </div>
        </Show>
      </div>

      <Show when={!validation.isValid() && validation.errorMessage()}>
        <div class={styles.errorMessage} role="alert">
          {validation.errorMessage()}
        </div>
      </Show>

      {/* Hidden input for form submission */}
      <input type="hidden" name={props.name} value={selectedValues().join(",")} />
    </div>
  );
};

interface MultiSelectOptionItemProps {
  option: MultiSelectOption;
  allOptions: MultiSelectOption[];
  isSelected: boolean;
  focusedIndex: number;
  onClick: () => void;
}

const MultiSelectOptionItem: Component<MultiSelectOptionItemProps> = (props) => {
  const availableOptions = () => props.allOptions.filter((opt) => !opt.disabled);
  const availableIndex = () =>
    availableOptions().findIndex((opt) => opt.value === props.option.value);
  const isFocused = () => availableIndex() === props.focusedIndex && !props.option.disabled;

  return (
    <div
      class={clsx(styles.option, {
        [styles.selected]: props.isSelected,
        [styles.disabled]: props.option.disabled,
        [styles.focused]: isFocused(),
      })}
      onClick={() => !props.option.disabled && props.onClick()}
      role="option"
      aria-selected={props.isSelected}
      aria-disabled={props.option.disabled}
      tabIndex={-1}
    >
      <div class={styles.checkbox}>
        <Show when={props.isSelected}>
          <CheckIcon />
        </Show>
      </div>
      <span class={styles.optionLabel}>{props.option.label}</span>
    </div>
  );
};
