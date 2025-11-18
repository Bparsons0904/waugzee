import { cleanup, fireEvent, render, screen } from "@solidjs/testing-library";
import { createSignal } from "solid-js";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import Login from "./Login";

const mockLoginWithOIDC = vi.fn();
const mockAuthConfig = createSignal({ configured: true });
const mockAuthContext = {
  loginWithOIDC: mockLoginWithOIDC,
  authConfig: mockAuthConfig[0],
  isAuthenticated: createSignal(false)[0],
  user: null,
  authToken: createSignal(null)[0],
  login: vi.fn(),
  register: vi.fn(),
  logout: vi.fn(),
};

vi.mock("@context/AuthContext", () => ({
  useAuth: () => mockAuthContext,
}));

describe("Login", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    sessionStorage.clear();
  });

  afterEach(() => {
    cleanup();
  });

  it("renders login page with OIDC button", () => {
    render(() => <Login />);

    expect(screen.getByText("Welcome to Waugzee")).toBeInTheDocument();
    expect(
      screen.getByText("Sign in or create an account to start tracking your vinyl collection"),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: /continue/i })).toBeInTheDocument();
  });

  it("shows authentication info message", () => {
    render(() => <Login />);

    expect(
      screen.getByText("You'll be securely redirected to complete sign in or create your account."),
    ).toBeInTheDocument();
  });

  it("shows error message when auth is not configured", () => {
    mockAuthConfig[1]({ configured: false });

    render(() => <Login />);

    expect(screen.getByText("Authentication is not configured")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: /sign in/i })).not.toBeInTheDocument();

    mockAuthConfig[1]({ configured: true });
  });

  it("calls loginWithOIDC when button is clicked", async () => {
    render(() => <Login />);

    const loginButton = screen.getByRole("button", { name: /continue/i });
    fireEvent.click(loginButton);

    expect(mockLoginWithOIDC).toHaveBeenCalledWith(undefined);
  });

  it("retrieves and uses returnTo from sessionStorage", async () => {
    sessionStorage.setItem("returnTo", "/dashboard");

    render(() => <Login />);

    const loginButton = screen.getByRole("button", { name: /continue/i });
    fireEvent.click(loginButton);

    expect(mockLoginWithOIDC).toHaveBeenCalledWith("/dashboard");
    expect(sessionStorage.getItem("returnTo")).toBeNull();
  });

  it("shows loading state during login", async () => {
    mockLoginWithOIDC.mockImplementation(() => new Promise((resolve) => setTimeout(resolve, 100)));

    render(() => <Login />);

    const loginButton = screen.getByRole("button", { name: /continue/i });
    fireEvent.click(loginButton);

    expect(await screen.findByText("Signing In...")).toBeInTheDocument();
    expect(loginButton).toBeDisabled();
  });

  it("shows error message when login fails", async () => {
    const consoleErrorSpy = vi.spyOn(console, "error").mockImplementation(() => {
      // Suppress console.error output during test
    });
    mockLoginWithOIDC.mockRejectedValue(new Error("Login failed"));

    render(() => <Login />);

    const loginButton = screen.getByRole("button", { name: /continue/i });
    fireEvent.click(loginButton);

    expect(await screen.findByText("Login failed")).toBeInTheDocument();
    consoleErrorSpy.mockRestore();
  });
});
