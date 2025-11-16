import { SearchInput } from "@components/common/forms/SearchInput/SearchInput";
import type { Component } from "solid-js";
import styles from "./BottomSearchBar.module.scss";

interface BottomSearchBarProps {
  value: string;
  onInput: (value: string) => void;
  placeholder?: string;
}

export const BottomSearchBar: Component<BottomSearchBarProps> = (props) => {
  return (
    <div class={styles.bottomSearchBar}>
      <div class={styles.container}>
        <SearchInput
          value={props.value}
          onInput={props.onInput}
          placeholder={props.placeholder || "Search..."}
          class={styles.searchInput}
        />
      </div>
    </div>
  );
};
