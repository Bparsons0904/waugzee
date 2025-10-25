import { fireEvent, render, screen, waitFor } from "@solidjs/testing-library";
import { afterEach, describe, expect, it, vi } from "vitest";
import { DiscogsTokenModal } from "./DiscogsTokenModal";

describe("DiscogsTokenModal", () => {
  const mockOnClose = vi.fn();

  afterEach(() => {
    vi.clearAllMocks();
  });

  it("renders modal content", () => {
    render(() => <DiscogsTokenModal onClose={mockOnClose} />);

    expect(screen.getByText("What is a Discogs Token?")).toBeInTheDocument();
    expect(screen.getByText("How to Get Your Token")).toBeInTheDocument();
    expect(screen.getByLabelText("Your Discogs API Token *")).toBeInTheDocument();
    expect(screen.getByText("Save Token")).toBeInTheDocument();
  });

  it("handles token input", async () => {
    render(() => <DiscogsTokenModal onClose={mockOnClose} />);

    const input = screen.getByLabelText("Your Discogs API Token *") as HTMLInputElement;
    fireEvent.input(input, { target: { value: "test-token-123" } });

    expect(input.value).toBe("test-token-123");
  });

  it("calls onClose after successful token submission", async () => {
    render(() => <DiscogsTokenModal onClose={mockOnClose} />);

    const input = screen.getByLabelText("Your Discogs API Token *");
    fireEvent.input(input, { target: { value: "test-token-123" } });

    const submitButton = screen.getByText("Save Token");
    fireEvent.click(submitButton);

    await waitFor(() => {
      expect(mockOnClose).toHaveBeenCalled();
    });
  });

  it("contains link to Discogs developer settings", () => {
    render(() => <DiscogsTokenModal onClose={mockOnClose} />);

    const link = screen.getByText("Discogs Developer Settings");
    expect(link).toBeInTheDocument();
    expect(link).toHaveAttribute("href", "https://www.discogs.com/settings/developers");
    expect(link).toHaveAttribute("target", "_blank");
  });
});
