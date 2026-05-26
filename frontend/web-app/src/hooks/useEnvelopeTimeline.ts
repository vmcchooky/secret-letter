import { useCallback, useEffect, useMemo, useRef } from "react";
import type React from "react";
import gsap from "gsap";

type TimelineElements = {
  scene: HTMLDivElement | null;
  envelope: HTMLDivElement | null;
  flap: HTMLDivElement | null;
  letter: HTMLDivElement | null;
  content: HTMLDivElement | null;
  dissolve: HTMLDivElement | null;
};

export type EnvelopeTimelineRefs = {
  sceneRef: React.RefObject<HTMLDivElement | null>;
  envelopeRef: React.RefObject<HTMLDivElement | null>;
  flapRef: React.RefObject<HTMLDivElement | null>;
  letterRef: React.RefObject<HTMLDivElement | null>;
  contentRef: React.RefObject<HTMLDivElement | null>;
  dissolveRef: React.RefObject<HTMLDivElement | null>;
};

export function useEnvelopeTimeline(reducedMotion: boolean) {
  const sceneRef = useRef<HTMLDivElement>(null);
  const envelopeRef = useRef<HTMLDivElement>(null);
  const flapRef = useRef<HTMLDivElement>(null);
  const letterRef = useRef<HTMLDivElement>(null);
  const contentRef = useRef<HTMLDivElement>(null);
  const dissolveRef = useRef<HTMLDivElement>(null);

  const idleTimeline = useRef<gsap.core.Timeline | null>(null);

  const elements = useCallback(
    (): TimelineElements => ({
      scene: sceneRef.current,
      envelope: envelopeRef.current,
      flap: flapRef.current,
      letter: letterRef.current,
      content: contentRef.current,
      dissolve: dissolveRef.current,
    }),
    [],
  );

  const envelopePieces = (envelope: HTMLDivElement) =>
    envelope.querySelectorAll(
      ".envelope-back, .envelope-pocket, .envelope-flap, .envelope-seal",
    );

  useEffect(() => {
    const { envelope } = elements();
    if (!envelope || reducedMotion) {
      return;
    }

    const prefersTouch = window.matchMedia("(pointer: coarse), (max-width: 680px)").matches;
    if (prefersTouch) {
      return;
    }

    idleTimeline.current = gsap
      .timeline({ repeat: -1, yoyo: true })
      .to(envelope, {
        y: -5,
        scale: 1.012,
        duration: 1.8,
        ease: "sine.inOut",
      });

    return () => {
      idleTimeline.current?.kill();
      idleTimeline.current = null;
    };
  }, [elements, reducedMotion]);

  const playOpen = useCallback(
    (onComplete?: () => void) => {
      const { scene, envelope, flap, letter, content } = elements();
      if (!scene || !envelope || !flap || !letter || !content) {
        onComplete?.();
        return;
      }

      idleTimeline.current?.kill();

      if (reducedMotion) {
        gsap.set([flap, letter, content], { clearProps: "all" });
        gsap.set(letter, { y: -154, opacity: 1 });
        gsap.set(content, { opacity: 1 });
        gsap.set(envelopePieces(envelope), { opacity: 0 });
        onComplete?.();
        return;
      }

      const isMobile = window.innerWidth < 680;
      const targetScale = isMobile ? 1.05 : 1.32;
      const visualY = isMobile ? -4 : -16;
      const clearWillChange = () => {
        gsap.set([envelope, flap, letter, content, envelopePieces(envelope)], {
          willChange: "auto",
        });
        gsap.set(scene, { "--letter-glow": 1 });
        onComplete?.();
      };
      const pieces = envelopePieces(envelope);

      gsap.killTweensOf([envelope, flap, letter, content, pieces, scene]);

      gsap
        .timeline({ onComplete: clearWillChange })
        .set([envelope, letter, content, pieces], {
          force3D: true,
          willChange: "transform, opacity",
        }, 0)
        .set(envelope, {
          y: 0,
          rotate: 0,
          scale: 1,
        }, 0)
        .set(letter, {
          y: isMobile ? 52 : 76,
          scale: isMobile ? 0.98 : 1.06,
          opacity: 0,
        }, 0)
        .to(pieces, {
          opacity: 0,
          y: isMobile ? 18 : 24,
          duration: isMobile ? 0.22 : 0.28,
          ease: "sine.out",
        }, 0.08)
        .to(letter, {
          y: visualY,
          scale: targetScale,
          opacity: 1,
          duration: isMobile ? 0.48 : 0.62,
          ease: "power3.out",
        }, 0.06)
        .to(content, {
          opacity: 1,
          y: 0,
          duration: isMobile ? 0.26 : 0.34,
          ease: "power2.out",
        }, isMobile ? 0.36 : 0.48);
    },
    [elements, reducedMotion],
  );

  const playCloseAndBurn = useCallback(
    (onBurning?: () => void, onComplete?: () => void) => {
      const { scene, envelope, letter, content, dissolve } = elements();
      if (!scene || !envelope || !letter || !content || !dissolve) {
        onBurning?.();
        onComplete?.();
        return;
      }

      if (reducedMotion) {
        gsap
          .timeline({ onComplete })
          .call(() => onBurning?.())
          .to([content, letter, envelope], { opacity: 0, duration: 0.2 });
        return;
      }

      const isMobile = window.innerWidth < 680;
      const dissolvePeakOpacity = isMobile ? 0.68 : 1;
      const dissolvePeakScale = isMobile ? 0.96 : 1;
      const contentDriftY = isMobile ? -6 : -10;
      const letterExitScale = isMobile ? 0.88 : 0.78;
      const letterExitY = isMobile ? -26 : -50;
      const letterExitRotate = isMobile ? 0.6 : 1.5;
      const dissolveFadeStart = isMobile ? 0.64 : 1.2;
      const glowFadeStart = isMobile ? 0.7 : 1.25;
      const envelopeFadeStart = isMobile ? 0.82 : 1.38;
      const clearWillChange = () => {
        gsap.set([envelope, letter, content, dissolve], { willChange: "auto" });
        onComplete?.();
      };

      gsap
        .timeline({ onComplete: clearWillChange })
        .call(() => onBurning?.())
        .set([envelope, letter, content, dissolve], {
          force3D: true,
          willChange: "transform, opacity",
        }, 0)
        .to(dissolve, {
          opacity: dissolvePeakOpacity,
          scale: dissolvePeakScale,
          duration: isMobile ? 0.16 : 0.3,
          ease: "sine.out",
        }, 0)
        .to(content, {
          opacity: 0,
          y: contentDriftY,
          ...(isMobile ? {} : { filter: "blur(8px)" }),
          duration: isMobile ? 0.3 : 0.62,
          ease: "sine.inOut",
        }, isMobile ? 0 : 0.08)
        .to(letter, {
          opacity: 0,
          scale: letterExitScale,
          y: letterExitY,
          rotate: letterExitRotate,
          ...(isMobile ? {} : { filter: "blur(10px)" }),
          duration: isMobile ? 0.58 : 1.22,
          ease: "power2.inOut",
        }, isMobile ? 0.04 : 0.08)
        .to(envelope, isMobile
          ? { opacity: 0.94, scale: 0.99, duration: 0.28, ease: "sine.out" }
          : {
              filter: "brightness(1.18) saturate(1.3)",
              duration: 0.44,
            }, 0.12)
        .to(scene, {
          "--letter-glow": 1.6,
          duration: isMobile ? 0.24 : 0.38,
          ease: "sine.out",
        }, 0)
        .to(dissolve, {
          opacity: 0,
          scale: isMobile ? 1.02 : 1.08,
          duration: isMobile ? 0.34 : 0.72,
          ease: "sine.in",
        }, dissolveFadeStart)
        .to(scene, {
          "--letter-glow": 0,
          duration: isMobile ? 0.34 : 0.55,
          ease: "sine.in",
        }, glowFadeStart)
        .to(envelope, {
          opacity: 0,
          duration: isMobile ? 0.18 : 0.2,
        }, envelopeFadeStart);
    },
    [elements, reducedMotion],
  );

  return useMemo(
    () => ({
      sceneRef,
      envelopeRef,
      flapRef,
      letterRef,
      contentRef,
      dissolveRef,
      playOpen,
      playCloseAndBurn,
    }),
    [playCloseAndBurn, playOpen],
  );
}
