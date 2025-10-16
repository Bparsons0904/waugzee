import { Accessor } from "solid-js";

interface SelectOption {
  value: string;
  disabled?: boolean;
}

interface UseSelectKeyboardProps<T extends SelectOption> {
  isOpen: Accessor<boolean>;
  setIsOpen: (value: boolean) => void;
  focusedIndex: Accessor<number>;
  setFocusedIndex: (value: number | ((prev: number) => number)) => void;
  options: Accessor<T[]>;
  onSelect: (option: T) => void;
  disabled?: boolean;
  onClose?: () => void;
}

export function useSelectKeyboard<T extends SelectOption>(
  props: UseSelectKeyboardProps<T>,
) {
  const handleKeyDown = (event: KeyboardEvent) => {
    if (props.disabled) return;

    const availableOptions = props
      .options()
      .filter((option) => !option.disabled);
    const maxIndex = availableOptions.length - 1;

    switch (event.key) {
      case "Enter":
        event.preventDefault();
        if (props.focusedIndex() >= 0 && props.focusedIndex() <= maxIndex) {
          const option = availableOptions[props.focusedIndex()];
          props.onSelect(option);
        }
        break;

      case "Escape":
        event.preventDefault();
        props.setIsOpen(false);
        props.setFocusedIndex(-1);
        props.onClose?.();
        break;

      case "ArrowDown":
        event.preventDefault();
        if (!props.isOpen()) {
          props.setIsOpen(true);
          props.setFocusedIndex(0);
        } else {
          props.setFocusedIndex((prev) => Math.min(prev + 1, maxIndex));
        }
        break;

      case "ArrowUp":
        event.preventDefault();
        if (props.isOpen()) {
          props.setFocusedIndex((prev) => Math.max(prev - 1, 0));
        }
        break;

      case "Tab":
        if (props.isOpen()) {
          props.setIsOpen(false);
          props.setFocusedIndex(-1);
          props.onClose?.();
        }
        break;
    }
  };

  return {
    handleKeyDown,
  };
}
