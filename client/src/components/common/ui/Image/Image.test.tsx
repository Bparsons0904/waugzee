import { fireEvent, render, screen, waitFor } from "@solidjs/testing-library";
import { describe, expect, it, vi } from "vitest";
import { Image } from "./Image";

describe("Image Component", () => {
  it("renders with basic props", () => {
    render(() => <Image src="/test-image.jpg" alt="Test image" />);

    const img = screen.getByAltText("Test image");
    expect(img).toBeInTheDocument();
    expect(img).toHaveAttribute("src", "/test-image.jpg");
    expect(img).toHaveAttribute("loading", "lazy");
  });

  it("shows skeleton when showSkeleton is true and image hasn't loaded", () => {
    const { container } = render(() => (
      <Image src="/test-image.jpg" alt="Test image" showSkeleton={true} />
    ));

    const skeleton = container.querySelector("[class*='skeleton']");
    expect(skeleton).toBeInTheDocument();
  });

  it("applies aspect ratio classes correctly", () => {
    const { container } = render(() => (
      <Image src="/test-image.jpg" alt="Test image" aspectRatio="album" />
    ));

    const imageContainer = container.querySelector("[class*='imageContainer']");
    expect(imageContainer?.className).toContain("aspectAlbum");
  });

  it("handles image load event", async () => {
    render(() => <Image src="/test-image.jpg" alt="Test image" showSkeleton={true} />);

    const img = screen.getByAltText("Test image");

    // Simulate image load
    fireEvent.load(img);

    await waitFor(() => {
      expect(img.className).toContain("loaded");
    });
  });

  it("handles image error with fallback", async () => {
    render(() => (
      <Image src="/nonexistent-image.jpg" alt="Test image" fallback="/fallback-image.jpg" />
    ));

    const img = screen.getByAltText("Test image");

    // Simulate image error
    fireEvent.error(img);

    await waitFor(() => {
      expect(img).toHaveAttribute("src", "/fallback-image.jpg");
    });
  });

  it("shows error state when both src and fallback fail", async () => {
    render(() => (
      <Image src="/nonexistent-image.jpg" alt="Test image" fallback="/nonexistent-fallback.jpg" />
    ));

    const img = screen.getByAltText("Test image");

    // Simulate first error (tries fallback)
    fireEvent.error(img);

    await waitFor(() => {
      expect(img).toHaveAttribute("src", "/nonexistent-fallback.jpg");
    });

    // Simulate second error (shows error state)
    fireEvent.error(img);

    await waitFor(() => {
      expect(screen.getByText("Image not available")).toBeInTheDocument();
    });
  });

  it("handles click events", () => {
    const handleClick = vi.fn();

    const { container } = render(() => (
      <Image src="/test-image.jpg" alt="Test image" onClick={handleClick} />
    ));

    const imageContainer = container.querySelector("[class*='imageContainer']");
    if (imageContainer) {
      fireEvent.click(imageContainer);
    }

    expect(handleClick).toHaveBeenCalledTimes(1);
  });

  it("applies custom className", () => {
    const { container } = render(() => (
      <Image src="/test-image.jpg" alt="Test image" className="custom-class" />
    ));

    const imageContainer = container.querySelector("[class*='imageContainer']");
    expect(imageContainer).toHaveClass("custom-class");
  });

  it("sets loading attribute correctly", () => {
    render(() => <Image src="/test-image.jpg" alt="Test image" loading="eager" />);

    const img = screen.getByAltText("Test image");
    expect(img).toHaveAttribute("loading", "eager");
  });

  it("sets width and height attributes", () => {
    render(() => <Image src="/test-image.jpg" alt="Test image" width={300} height={200} />);

    const img = screen.getByAltText("Test image");
    expect(img).toHaveAttribute("width", "300");
    expect(img).toHaveAttribute("height", "200");
  });

  it("sets sizes attribute for responsive images", () => {
    render(() => (
      <Image src="/test-image.jpg" alt="Test image" sizes="(max-width: 768px) 100vw, 50vw" />
    ));

    const img = screen.getByAltText("Test image");
    expect(img).toHaveAttribute("sizes", "(max-width: 768px) 100vw, 50vw");
  });
});
