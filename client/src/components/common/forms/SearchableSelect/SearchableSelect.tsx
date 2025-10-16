import {
  Component,
  createSignal,
  createUniqueId,
  For,
  Show,
  createEffect,
} from "solid-js";
import { ValidatorFunction } from "../../../../utils/validation";
import { useSelectBase } from "@hooks/useSelectBase";
import { useSelectKeyboard } from "@hooks/useSelectKeyboard";
import { ChevronDownIcon } from "@components/icons/ChevronDownIcon";
import { SearchIcon } from "@components/icons/SearchIcon";
import Fuse from "fuse.js";
import clsx from "clsx";
import styles from "./SearchableSelect.module.scss";

export interface SearchableSelectOption {
  value: string;
  label: string;
  disabled?: boolean;
  metadata?: string;
}

export interface SearchableSelectProps {
  name?: string;
  label?: string;
  placeholder?: string;
  searchPlaceholder?: string;
  options: SearchableSelectOption[];
  value?: string;
  required?: boolean;
  disabled?: boolean;
  customValidators?: ValidatorFunction[];
  class?: string;
  onChange?: (value: string) => void;
  onBlur?: (event: FocusEvent) => void;
  defaultValue?: string;
  emptyMessage?: string;
}

export const SearchableSelect: Component<SearchableSelectProps> = (props) => {
  const id = createUniqueId();
  const [selectedValue, setSelectedValue] = createSignal<string>(
    props.value || props.defaultValue || "",
  );
  const [searchQuery, setSearchQuery] = createSignal("");
  let containerRef: HTMLDivElement | undefined;
  let searchInputRef: HTMLInputElement | undefined;

  createEffect(() => {
    if (props.value !== undefined) {
      setSelectedValue(props.value);
    }
  });

  const {
    isOpen,
    setIsOpen,
    focusedIndex,
    setFocusedIndex,
    validation,
    formField,
    handleBlur: baseHandleBlur,
    registerClickOutsideListener,
  } = useSelectBase(
    {
      name: props.name,
      label: props.label,
      required: props.required,
      disabled: props.disabled,
      customValidators: props.customValidators,
      onBlur: props.onBlur,
    },
    props.value || props.defaultValue
  );

  registerClickOutsideListener(containerRef);

  const filteredOptions = () => {
    const query = searchQuery().trim();
    if (!query) return props.options;

    const fuse = new Fuse(props.options, {
      keys: [
        { name: "label", weight: 2 },
        { name: "metadata", weight: 1 },
        { name: "value", weight: 0.5 },
      ],
      threshold: 0.4,
      distance: 100,
      minMatchCharLength: 2,
    });

    const results = fuse.search(query);
    return results.map((result) => result.item);
  };

  const getSelectedOption = () => {
    return props.options.find((opt) => opt.value === selectedValue());
  };

  const handleOptionSelect = (optionValue: string) => {
    if (props.disabled) return;

    setSelectedValue(optionValue);
    validation.setValue(optionValue);
    props.onChange?.(optionValue);

    if (formField.isConnectedToForm) {
      const validationResult = validation.validate(optionValue, true);
      formField.updateFormField({
        isValid: validationResult.isValid,
        errorMessage: validationResult.errorMessage,
        value: optionValue,
      });
    }

    setIsOpen(false);
    setSearchQuery("");
    setFocusedIndex(-1);
  };

  const handleBlur = (event: FocusEvent) => {
    const relatedTarget = event.relatedTarget as HTMLElement;
    if (relatedTarget && containerRef?.contains(relatedTarget)) {
      return;
    }

    setSearchQuery("");
    baseHandleBlur(event, selectedValue());
  };

  const handleTriggerClick = () => {
    if (props.disabled) return;

    const newOpenState = !isOpen();
    setIsOpen(newOpenState);

    if (newOpenState) {
      setSearchQuery("");
      setFocusedIndex(0);
      setTimeout(() => {
        searchInputRef?.focus();
      }, 0);
    }
  };

  const handleSearchInput = (event: InputEvent) => {
    const value = (event.target as HTMLInputElement).value;
    setSearchQuery(value);
    setFocusedIndex(0);
  };

  const { handleKeyDown } = useSelectKeyboard({
    isOpen,
    setIsOpen,
    focusedIndex,
    setFocusedIndex,
    options: () => filteredOptions(),
    onSelect: (option) => handleOptionSelect(option.value),
    disabled: props.disabled,
    onClose: () => setSearchQuery(""),
  });

  const getDisplayText = () => {
    const selected = getSelectedOption();
    if (selected) {
      return selected.label;
    }
    return props.placeholder || "Select an option...";
  };

  return (
    <div
      ref={containerRef}
      class={clsx(styles.searchableSelectContainer, props.class)}
    >
      <Show when={props.label}>
        <label for={id} class={styles.label}>
          {props.label}
          <Show when={props.required}>
            <span class={styles.required}>*</span>
          </Show>
        </label>
      </Show>

      <div class={styles.searchableSelectWrapper}>
        <div
          class={clsx(styles.trigger, {
            [styles.selectError]: !validation.isValid(),
            [styles.open]: isOpen(),
            [styles.disabled]: props.disabled,
          })}
          onClick={handleTriggerClick}
          onBlur={handleBlur}
          tabIndex={props.disabled ? -1 : 0}
          role="combobox"
          aria-expanded={isOpen()}
          aria-haspopup="listbox"
          aria-owns={`${id}-listbox`}
          aria-disabled={props.disabled}
        >
          <span
            class={clsx({
              [styles.placeholder]: !selectedValue(),
              [styles.selectedText]: selectedValue(),
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
            class={styles.dropdownContainer}
            id={`${id}-listbox`}
            role="listbox"
            aria-labelledby={id}
          >
            <div class={styles.searchContainer}>
              <div class={styles.searchIcon}>
                <SearchIcon />
              </div>
              <input
                ref={searchInputRef}
                type="text"
                class={styles.searchInput}
                placeholder={props.searchPlaceholder || "Search..."}
                value={searchQuery()}
                onInput={handleSearchInput}
                onKeyDown={handleKeyDown}
                onBlur={handleBlur}
              />
            </div>

            <div class={styles.optionsContainer}>
              <Show
                when={filteredOptions().length > 0}
                fallback={
                  <div class={styles.emptyMessage}>
                    {props.emptyMessage || "No options found"}
                  </div>
                }
              >
                <For each={filteredOptions()}>
                  {(option) => (
                    <SearchableSelectOptionItem
                      option={option}
                      allOptions={filteredOptions()}
                      isSelected={selectedValue() === option.value}
                      selectedValue={selectedValue()}
                      focusedIndex={focusedIndex()}
                      onClick={() => handleOptionSelect(option.value)}
                    />
                  )}
                </For>
              </Show>
            </div>
          </div>
        </Show>
      </div>

      <Show when={!validation.isValid() && validation.errorMessage()}>
        <div class={styles.errorMessage} role="alert">
          {validation.errorMessage()}
        </div>
      </Show>

      <input type="hidden" name={props.name} value={selectedValue()} />
    </div>
  );
};

interface SearchableSelectOptionItemProps {
  option: SearchableSelectOption;
  allOptions: SearchableSelectOption[];
  isSelected: boolean;
  selectedValue: string;
  focusedIndex: number;
  onClick: () => void;
}

const SearchableSelectOptionItem: Component<SearchableSelectOptionItemProps> = (
  props,
) => {
  const availableOptions = () =>
    props.allOptions.filter((opt) => !opt.disabled);
  const availableIndex = () =>
    availableOptions().findIndex((opt) => opt.value === props.option.value);
  const isFocused = () =>
    availableIndex() === props.focusedIndex && !props.option.disabled;

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
      <span class={styles.optionLabel}>{props.option.label}</span>
      <Show when={props.option.metadata}>
        <span class={styles.optionMetadata}>{props.option.metadata}</span>
      </Show>
    </div>
  );
};
