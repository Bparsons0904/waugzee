import { render } from "@solidjs/testing-library";
import { describe, expect, it, vi } from "vitest";
import { Card } from "./Card";

describe("Card", () => {
  it("renders children correctly", () => {
    const { getByText } = render(() => (
      <Card>
        <div>Test content</div>
      </Card>
    ));

    expect(getByText("Test content")).toBeInTheDocument();
  });

  it("applies size classes correctly", () => {
    const { container } = render(() => (
      <Card size="large">
        <div>Content</div>
      </Card>
    ));

    const card = container.firstChild as HTMLElement;
    expect(card.className).toContain("cardLarge");
  });

  it("applies custom class name", () => {
    const { container } = render(() => (
      <Card class="custom-class">
        <div>Content</div>
      </Card>
    ));

    const card = container.firstChild as HTMLElement;
    expect(card.className).toContain("custom-class");
  });

  it("handles click events", () => {
    const handleClick = vi.fn();
    const { container } = render(() => (
      <Card onClick={handleClick}>
        <div>Content</div>
      </Card>
    ));

    const card = container.firstChild as HTMLElement;
    card.click();
    expect(handleClick).toHaveBeenCalledOnce();
  });
});
