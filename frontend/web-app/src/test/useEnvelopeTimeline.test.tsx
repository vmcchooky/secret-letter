import { renderHook } from "@testing-library/react";
import gsap from "gsap";
import { describe, expect, it, vi } from "vitest";
import { useEnvelopeTimeline } from "../hooks/useEnvelopeTimeline";

describe("useEnvelopeTimeline", () => {
  it("does not create the idle tween when reduced motion is enabled", () => {
    const timelineSpy = vi.spyOn(gsap, "timeline");

    renderHook(() => useEnvelopeTimeline(true));

    expect(timelineSpy).not.toHaveBeenCalled();
  });
});
