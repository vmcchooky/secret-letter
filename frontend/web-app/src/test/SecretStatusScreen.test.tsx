import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { SecretStatusScreen } from "../components/secret-letter/SecretStatusScreen";

function mockMatchMedia(matches: boolean) {
  vi.mocked(window.matchMedia).mockImplementation((query: string) => ({
    matches: query.includes("prefers-reduced-motion") ? matches : false,
    media: query,
    onchange: null,
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    addListener: vi.fn(),
    removeListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }));
}

describe("SecretStatusScreen", () => {
  it("skips shooting star timers when reduced motion is enabled", () => {
    mockMatchMedia(true);
    const intervalSpy = vi.spyOn(window, "setInterval");

    render(<SecretStatusScreen state="consumed" />);

    expect(intervalSpy).not.toHaveBeenCalled();
  });

  it("coalesces mousemove updates into a single animation frame", () => {
    mockMatchMedia(false);
    const rafSpy = vi.spyOn(window, "requestAnimationFrame");

    render(<SecretStatusScreen state="loading" />);

    const status = screen.getByRole("status");
    vi.spyOn(status, "getBoundingClientRect").mockReturnValue({
      left: 0,
      top: 0,
      width: 200,
      height: 200,
      right: 200,
      bottom: 200,
      x: 0,
      y: 0,
      toJSON: () => ({}),
    } as DOMRect);

    fireEvent.mouseMove(status, { clientX: 20, clientY: 20 });
    fireEvent.mouseMove(status, { clientX: 50, clientY: 80 });
    fireEvent.mouseMove(status, { clientX: 120, clientY: 140 });

    expect(rafSpy).toHaveBeenCalledTimes(1);
  });
});
