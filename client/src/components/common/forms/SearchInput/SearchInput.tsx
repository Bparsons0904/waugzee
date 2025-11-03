import { SearchIcon } from "@components/icons/SearchIcon";
import clsx from "clsx";
import type { Component } from "solid-js";
import styles from "./SearchInput.module.scss";

interface SearchInputProps {
  value: string;
  onInput: (value: string) => void;
  placeholder?: string;
  class?: string;
  id?: string;
}

export const SearchInput: Component<SearchInputProps> = (props) => {
  return (
    <div class={clsx(styles.searchInputWrapper, props.class)}>
      <SearchIcon size={20} class={styles.searchIcon} />
      <input
        id={props.id}
        type="text"
        placeholder={props.placeholder || "Search..."}
        value={props.value}
        onInput={(e) => props.onInput(e.currentTarget.value)}
        class={styles.searchInput}
      />
    </div>
  );
};
