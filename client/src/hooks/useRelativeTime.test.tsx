import { render } from "@solidjs/testing-library";
import { describe, expect, it } from "vitest";
import { useRelativeTime } from "./useRelativeTime";

describe("useRelativeTime", () => {
  it("displays 'Just now' for recent dates", () => {
    const TestComponent = () => {
      const now = new Date();
      const relativeTime = useRelativeTime(now);
      return <div data-testid="time">{relativeTime()}</div>;
    };

    const { getByTestId } = render(() => <TestComponent />);
    expect(getByTestId("time").textContent).toBe("Just now");
  });

  it("displays minutes ago for times < 1 hour", () => {
    const TestComponent = () => {
      const fiveMinutesAgo = new Date(Date.now() - 5 * 60 * 1000);
      const relativeTime = useRelativeTime(fiveMinutesAgo);
      return <div data-testid="time">{relativeTime()}</div>;
    };

    const { getByTestId } = render(() => <TestComponent />);
    expect(getByTestId("time").textContent).toBe("5m ago");
  });

  it("displays hours ago for times < 24 hours", () => {
    const TestComponent = () => {
      const threeHoursAgo = new Date(Date.now() - 3 * 60 * 60 * 1000);
      const relativeTime = useRelativeTime(threeHoursAgo);
      return <div data-testid="time">{relativeTime()}</div>;
    };

    const { getByTestId } = render(() => <TestComponent />);
    expect(getByTestId("time").textContent).toBe("3h ago");
  });

  it("displays days ago for times < 7 days", () => {
    const TestComponent = () => {
      const threeDaysAgo = new Date(Date.now() - 3 * 24 * 60 * 60 * 1000);
      const relativeTime = useRelativeTime(threeDaysAgo);
      return <div data-testid="time">{relativeTime()}</div>;
    };

    const { getByTestId } = render(() => <TestComponent />);
    expect(getByTestId("time").textContent).toBe("3d ago");
  });

  it("displays formatted date for times > 7 days", () => {
    const TestComponent = () => {
      const tenDaysAgo = new Date(Date.now() - 10 * 24 * 60 * 60 * 1000);
      const relativeTime = useRelativeTime(tenDaysAgo);
      return <div data-testid="time">{relativeTime()}</div>;
    };

    const { getByTestId } = render(() => <TestComponent />);
    const text = getByTestId("time").textContent || "";
    expect(text).not.toBe("Just now");
    expect(text).not.toContain("ago");
  });
});
