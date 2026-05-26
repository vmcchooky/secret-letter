import "@testing-library/jest-dom/vitest";
import { afterEach, vi } from "vitest";

afterEach(() => {
  vi.useRealTimers();
  vi.clearAllMocks();
  vi.restoreAllMocks();
});

if (typeof window !== "undefined") {
  Object.defineProperty(window, "matchMedia", {
    writable: true,
    value: vi.fn().mockImplementation((query: string) => ({
      matches: false,
      media: query,
      onchange: null,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
      addListener: vi.fn(),
      removeListener: vi.fn(),
      dispatchEvent: vi.fn(),
    })),
  });

  const rafHandles = new Map<number, ReturnType<typeof setTimeout>>();
  let rafCounter = 0;

  Object.defineProperty(window, "requestAnimationFrame", {
    writable: true,
    value: (callback: FrameRequestCallback) => {
      rafCounter += 1;
      const handle = rafCounter;
      const timeout = globalThis.setTimeout(() => {
        rafHandles.delete(handle);
        callback(performance.now());
      }, 16);
      rafHandles.set(handle, timeout);
      return handle;
    },
  });

  Object.defineProperty(window, "cancelAnimationFrame", {
    writable: true,
    value: (handle: number) => {
      const timeout = rafHandles.get(handle);
      if (timeout !== undefined) {
        globalThis.clearTimeout(timeout);
        rafHandles.delete(handle);
      }
    },
  });

  if (!navigator.clipboard) {
    Object.defineProperty(navigator, "clipboard", {
      writable: true,
      value: {
        writeText: vi.fn().mockResolvedValue(undefined),
      },
    });
  }
}
