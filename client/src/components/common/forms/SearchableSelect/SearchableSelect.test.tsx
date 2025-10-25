import { fireEvent, render, screen } from "@solidjs/testing-library";
import { describe, expect, it, vi } from "vitest";
import { SearchableSelect, type SearchableSelectOption } from "./SearchableSelect";

describe("SearchableSelect", () => {
  const mockOptions: SearchableSelectOption[] = [
    { value: "1", label: "Ortofon 2M Blue", metadata: "Elliptical" },
    { value: "2", label: "Audio-Technica AT-VM95E", metadata: "Elliptical" },
    { value: "3", label: "Shure M97xE", metadata: "Hyperelliptical" },
    { value: "4", label: "Nagaoka MP-110", metadata: "Elliptical" },
  ];

  it("renders with label", () => {
    render(() => (
      <SearchableSelect label="Select Stylus" options={mockOptions} placeholder="Choose a stylus" />
    ));

    expect(screen.getByText("Select Stylus")).toBeInTheDocument();
  });

  it("shows placeholder when no value is selected", () => {
    render(() => <SearchableSelect options={mockOptions} placeholder="Choose a stylus" />);

    expect(screen.getByText("Choose a stylus")).toBeInTheDocument();
  });

  it("displays selected option", () => {
    render(() => (
      <SearchableSelect options={mockOptions} value="1" placeholder="Choose a stylus" />
    ));

    expect(screen.getByText("Ortofon 2M Blue")).toBeInTheDocument();
  });

  it("opens dropdown when trigger is clicked", async () => {
    render(() => (
      <SearchableSelect
        options={mockOptions}
        placeholder="Choose a stylus"
        searchPlaceholder="Search styluses..."
      />
    ));

    const trigger = screen.getByRole("combobox");
    fireEvent.click(trigger);

    expect(screen.getByPlaceholderText("Search styluses...")).toBeInTheDocument();
  });

  it("filters options based on search query", async () => {
    render(() => (
      <SearchableSelect
        options={mockOptions}
        placeholder="Choose a stylus"
        searchPlaceholder="Search..."
      />
    ));

    const trigger = screen.getByRole("combobox");
    fireEvent.click(trigger);

    const searchInput = screen.getByPlaceholderText("Search...");
    fireEvent.input(searchInput, { target: { value: "ortofon" } });

    expect(screen.getByText("Ortofon 2M Blue")).toBeInTheDocument();
    expect(screen.queryByText("Audio-Technica AT-VM95E")).not.toBeInTheDocument();
  });

  it("filters options using fuzzy search", async () => {
    render(() => (
      <SearchableSelect
        options={mockOptions}
        placeholder="Choose a stylus"
        searchPlaceholder="Search..."
      />
    ));

    const trigger = screen.getByRole("combobox");
    fireEvent.click(trigger);

    const searchInput = screen.getByPlaceholderText("Search...");
    fireEvent.input(searchInput, { target: { value: "shure" } });

    expect(screen.getByText("Shure M97xE")).toBeInTheDocument();
    expect(screen.queryByText("Ortofon 2M Blue")).not.toBeInTheDocument();
    expect(screen.queryByText("Audio-Technica AT-VM95E")).not.toBeInTheDocument();
  });

  it("shows empty message when no options match search", async () => {
    render(() => (
      <SearchableSelect
        options={mockOptions}
        placeholder="Choose a stylus"
        searchPlaceholder="Search..."
        emptyMessage="No styluses found"
      />
    ));

    const trigger = screen.getByRole("combobox");
    fireEvent.click(trigger);

    const searchInput = screen.getByPlaceholderText("Search...");
    fireEvent.input(searchInput, { target: { value: "xyz123" } });

    expect(screen.getByText("No styluses found")).toBeInTheDocument();
  });

  it("calls onChange when option is selected", async () => {
    const handleChange = vi.fn();

    render(() => (
      <SearchableSelect
        options={mockOptions}
        placeholder="Choose a stylus"
        onChange={handleChange}
      />
    ));

    const trigger = screen.getByRole("combobox");
    fireEvent.click(trigger);

    const option = screen.getByText("Ortofon 2M Blue");
    fireEvent.click(option);

    expect(handleChange).toHaveBeenCalledWith("1");
  });

  it("closes dropdown after selecting an option", async () => {
    render(() => (
      <SearchableSelect
        options={mockOptions}
        placeholder="Choose a stylus"
        searchPlaceholder="Search..."
      />
    ));

    const trigger = screen.getByRole("combobox");
    fireEvent.click(trigger);

    expect(screen.getByPlaceholderText("Search...")).toBeInTheDocument();

    const option = screen.getByText("Ortofon 2M Blue");
    fireEvent.click(option);

    expect(screen.queryByPlaceholderText("Search...")).not.toBeInTheDocument();
  });

  it("shows required indicator when required prop is true", () => {
    render(() => <SearchableSelect label="Select Stylus" options={mockOptions} required={true} />);

    expect(screen.getByText("*")).toBeInTheDocument();
  });

  it("disables trigger when disabled prop is true", () => {
    render(() => (
      <SearchableSelect options={mockOptions} placeholder="Choose a stylus" disabled={true} />
    ));

    const trigger = screen.getByRole("combobox");
    expect(trigger).toHaveAttribute("aria-disabled", "true");
  });

  it("clears search query when dropdown closes", async () => {
    render(() => (
      <SearchableSelect
        options={mockOptions}
        placeholder="Choose a stylus"
        searchPlaceholder="Search..."
      />
    ));

    const trigger = screen.getByRole("combobox");
    fireEvent.click(trigger);

    const searchInput = screen.getByPlaceholderText("Search...");
    fireEvent.input(searchInput, { target: { value: "ortofon" } });

    fireEvent.keyDown(searchInput, { key: "Escape" });

    fireEvent.click(trigger);

    const reopenedSearchInput = screen.getByPlaceholderText("Search...");
    expect(reopenedSearchInput).toHaveValue("");
  });
});
