import { fireEvent, render, screen, waitFor } from "@solidjs/testing-library";
import { QueryClient, QueryClientProvider } from "@tanstack/solid-query";
import type { JSX } from "solid-js";
import { afterEach, describe, expect, it, vi } from "vitest";
import { DiscogsTokenModal } from "./DiscogsTokenModal";

vi.mock("@context/ToastContext", () => ({
  useToast: () => ({
    showSuccess: vi.fn(),
    showError: vi.fn(),
    showInfo: vi.fn(),
    showWarning: vi.fn(),
  }),
}));

const mockMutate = vi.fn();

vi.mock("@services/apiHooks", () => ({
  useApiPost: vi.fn(() => ({
    mutate: mockMutate,
    isPending: false,
    isLoading: false,
    isError: false,
    error: null,
    data: null,
  })),
  useApiPut: vi.fn(() => ({
    mutate: vi.fn(),
    isPending: false,
    isLoading: false,
    isError: false,
    error: null,
    data: null,
  })),
}));

const TestWrapper = (props: { children: JSX.Element }) => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });

  return <QueryClientProvider client={queryClient}>{props.children}</QueryClientProvider>;
};

describe("DiscogsTokenModal", () => {
  const mockOnClose = vi.fn();

  afterEach(() => {
    vi.clearAllMocks();
  });

  it("renders modal content", () => {
    render(() => (
      <TestWrapper>
        <DiscogsTokenModal onClose={mockOnClose} />
      </TestWrapper>
    ));

    expect(screen.getByText("What is a Discogs Token?")).toBeInTheDocument();
    expect(screen.getByText("How to Get Your Token")).toBeInTheDocument();
    expect(screen.getByLabelText("Your Discogs API Token *")).toBeInTheDocument();
    expect(screen.getByText("Save Token")).toBeInTheDocument();
  });

  it("handles token input", async () => {
    render(() => (
      <TestWrapper>
        <DiscogsTokenModal onClose={mockOnClose} />
      </TestWrapper>
    ));

    const input = screen.getByLabelText("Your Discogs API Token *") as HTMLInputElement;
    fireEvent.input(input, { target: { value: "test-token-123" } });

    expect(input.value).toBe("test-token-123");
  });

  it("has a submit button that is enabled when token is entered", async () => {
    render(() => (
      <TestWrapper>
        <DiscogsTokenModal onClose={mockOnClose} />
      </TestWrapper>
    ));

    const submitButton = screen.getByText("Save Token") as HTMLButtonElement;
    expect(submitButton).toBeDisabled();

    const input = screen.getByLabelText("Your Discogs API Token *");
    fireEvent.input(input, { target: { value: "test-token-123" } });

    await waitFor(() => {
      expect(submitButton).not.toBeDisabled();
    });
  });

  it("contains link to Discogs developer settings", () => {
    render(() => (
      <TestWrapper>
        <DiscogsTokenModal onClose={mockOnClose} />
      </TestWrapper>
    ));

    const link = screen.getByText("Discogs Developer Settings");
    expect(link).toBeInTheDocument();
    expect(link).toHaveAttribute("href", "https://www.discogs.com/settings/developers");
    expect(link).toHaveAttribute("target", "_blank");
  });
});
