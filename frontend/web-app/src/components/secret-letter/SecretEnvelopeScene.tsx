import { useState, useCallback, useEffect, useMemo, useRef } from "react";
import gsap from "gsap";
import type { SecretSceneState } from "../../lib/types";
import { useEnvelopeTimeline } from "../../hooks/useEnvelopeTimeline";
import { useReducedMotion } from "../../hooks/useReducedMotion";
import { TactileButton } from "../TactileButton";
import { Envelope } from "./Envelope";
import { MagicDissolveEffect } from "./MagicDissolveEffect";
import { SecretStatusScreen } from "./SecretStatusScreen";
import "./secret-letter.css";

type StatusSceneState = Extract<SecretSceneState, "loading" | "vanished" | "expired" | "consumed" | "error">;
type EnvelopeSceneState = Extract<SecretSceneState, "sealed" | "opening" | "revealed" | "closing" | "burning">;

type SecretEnvelopeSceneProps = {
  state: SecretSceneState;
  secretContent: string;
  expiresAt?: string;
  message?: string;
  onOpen: () => void;
  onOpenComplete: () => void;
  onClose: () => void;
  onBurning: () => void;
  onBurnComplete: () => void;
  onRetry?: () => void;
};

export function SecretEnvelopeScene({
  state,
  secretContent,
  expiresAt,
  message,
  onOpen,
  onOpenComplete,
  onClose,
  onBurning,
  onBurnComplete,
  onRetry,
}: SecretEnvelopeSceneProps) {
  const reducedMotion = useReducedMotion();
  const timeline = useEnvelopeTimeline(reducedMotion);
  const [consentGranted, setConsentGranted] = useState(false);
  const currentStatusData = useMemo(() => getStatusData(state, message), [message, state]);
  const [visibleStatusData, setVisibleStatusData] = useState(currentStatusData);
  const [statusTransition, setStatusTransition] = useState<"enter" | "exit">("enter");
  const statusTransitionTimerRef = useRef<number | null>(null);

  // Hold-to-open belongs to the envelope, after the consent warning is accepted.
  const [isHolding, setIsHolding] = useState(false);
  const holdProgressRef = useRef<HTMLSpanElement>(null);
  const holdTweenRef = useRef<gsap.core.Tween | null>(null);
  const hasTriggeredOpenRef = useRef(false);

  const setHoldProgress = useCallback((progress: number) => {
    if (!holdProgressRef.current) {
      return;
    }

    gsap.set(holdProgressRef.current, {
      scaleX: progress,
      transformOrigin: "0% 50%",
    });
  }, []);

  const triggerOpen = useCallback(() => {
    if (hasTriggeredOpenRef.current || state !== "sealed") {
      return;
    }

    hasTriggeredOpenRef.current = true;
    holdTweenRef.current?.kill();
    holdTweenRef.current = null;
    setConsentGranted(true);
    setIsHolding(false);
    setHoldProgress(1);
    onOpen();
  }, [onOpen, setHoldProgress, state]);

  const startEnvelopeHold = () => {
    if (hasTriggeredOpenRef.current || state !== "sealed" || !consentGranted) {
      return;
    }

    if (reducedMotion) {
      triggerOpen();
      return;
    }

    holdTweenRef.current?.kill();
    holdTweenRef.current = null;

    setIsHolding(true);
    setHoldProgress(0);

    if (!holdProgressRef.current) {
      triggerOpen();
      return;
    }

    holdTweenRef.current = gsap.to(holdProgressRef.current, {
      scaleX: 1,
      duration: 0.85,
      ease: "power2.out",
      overwrite: true,
      onComplete: () => {
        holdTweenRef.current = null;
        triggerOpen();
      },
    });
  };

  const stopEnvelopeHold = () => {
    if (reducedMotion || hasTriggeredOpenRef.current || state !== "sealed") return;
    setIsHolding(false);
    holdTweenRef.current?.kill();
    holdTweenRef.current = null;

    if (!holdProgressRef.current) {
      return;
    }

    holdTweenRef.current = gsap.to(holdProgressRef.current, {
      scaleX: 0,
      duration: 0.22,
      ease: "power3.out",
      overwrite: true,
      onComplete: () => {
        holdTweenRef.current = null;
      },
    });
  };

  useEffect(() => {
    return () => {
      holdTweenRef.current?.kill();
      if (statusTransitionTimerRef.current !== null) {
        window.clearTimeout(statusTransitionTimerRef.current);
      }
    };
  }, []);

  useEffect(() => {
    if (statusTransitionTimerRef.current !== null) {
      window.clearTimeout(statusTransitionTimerRef.current);
      statusTransitionTimerRef.current = null;
    }

    if (!currentStatusData) {
      if (visibleStatusData) {
        setStatusTransition("exit");
        statusTransitionTimerRef.current = window.setTimeout(() => {
          statusTransitionTimerRef.current = null;
          setVisibleStatusData(null);
        }, reducedMotion ? 1 : 170);
      }
      return;
    }

    if (!visibleStatusData) {
      setVisibleStatusData(currentStatusData);
      setStatusTransition("enter");
      return;
    }

    if (visibleStatusData.state !== currentStatusData.state) {
      setStatusTransition("exit");
      statusTransitionTimerRef.current = window.setTimeout(() => {
        statusTransitionTimerRef.current = null;
        setVisibleStatusData(currentStatusData);
        setStatusTransition("enter");
      }, reducedMotion ? 1 : 170);
      return;
    }

    if (visibleStatusData.message !== currentStatusData.message) {
      setVisibleStatusData(currentStatusData);
    }

    setStatusTransition("enter");
  }, [currentStatusData, reducedMotion, visibleStatusData]);

  useEffect(() => {
    if (state === "sealed") {
      hasTriggeredOpenRef.current = false;
      setConsentGranted(false);
      setHoldProgress(0);
    }
  }, [setHoldProgress, state]);

  const handleOpen = useCallback(() => {
    if (state === "sealed" && consentGranted) {
      triggerOpen();
    }
  }, [state, consentGranted, triggerOpen]);

  const handleConsentAccept = useCallback(() => {
    setConsentGranted(true);
    setHoldProgress(0);
    setIsHolding(false);
  }, [setHoldProgress]);

  const handleClose = useCallback(() => {
    if (state === "revealed") {
      onClose();
    }
  }, [onClose, state]);

  useEffect(() => {
    if (state === "opening") {
      timeline.playOpen(onOpenComplete);
    }
  }, [onOpenComplete, state, timeline]);

  useEffect(() => {
    if (state === "closing") {
      timeline.playCloseAndBurn(onBurning, onBurnComplete);
    }
  }, [onBurnComplete, onBurning, state, timeline]);

  if (visibleStatusData) {
    return (
      <SecretStatusScreen
        state={visibleStatusData.state}
        message={visibleStatusData.message}
        onRetry={onRetry}
        transitionState={statusTransition}
      />
    );
  }

  if (currentStatusData) {
    return (
      <SecretStatusScreen
        state={currentStatusData.state}
        message={currentStatusData.message}
        onRetry={onRetry}
      />
    );
  }

  if (!isEnvelopeState(state)) {
    return null;
  }

  return (
    <main
      ref={timeline.sceneRef}
      className={`secret-letter-scene secret-letter-scene-${state}`}
      data-reduced-motion={reducedMotion ? "true" : "false"}
      data-consent-open={(!consentGranted && state === "sealed") ? "true" : "false"}
    >
      <div className="scene-light" />
      <MagicDissolveEffect dissolveRef={timeline.dissolveRef} active={state === "burning"} />

      <Envelope
        state={state}
        refs={timeline}
        secretContent={secretContent}
        expiresAt={expiresAt}
        onOpen={handleOpen}
        onOpenHoldStart={startEnvelopeHold}
        onOpenHoldCancel={stopEnvelopeHold}
        openHoldProgressRef={holdProgressRef}
        isOpenHolding={isHolding}
        canHoldOpen={consentGranted}
        onClose={handleClose}
      />

      {!consentGranted && state === "sealed" && (
        <div className="otl-consent-overlay glass-blur">
          <div className="otl-consent-card vellum-card">
            <div className="otl-consent-badge-minimal">
              <svg className="otl-shining-star-icon" viewBox="0 0 24 24" fill="currentColor">
                <path d="M12 0C12 8 16 12 24 12C16 12 12 16 12 24C12 16 8 12 0 12C8 12 12 8 12 0Z" />
              </svg>
              Mã hóa cục bộ
            </div>
            
            <h2>Phong thư mật</h2>
            
            <div className="otl-visual-warning">
              <div className="hourglass-container">
                <svg className="otl-hourglass-svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round">
                  <path d="M5 2h14M5 22h14" />
                  <path d="M19 2v4a7 7 0 0 1-7 7 7 7 0 0 1-7-7V2" />
                  <path d="M5 22v-4a7 7 0 0 1 7-7 7 7 0 0 1 7 7v4" />
                  <circle cx="12" cy="5" r="1" className="hourglass-sand-top" />
                  <path d="M12 11v3" className="hourglass-stream" />
                  <path d="M10 18h4" className="hourglass-sand-bottom" />
                </svg>
              </div>
              <p className="otl-visual-warning-text">
                Mở một lần duy nhất.<br />Mật thư sẽ tự hủy sau khi đọc.
              </p>
            </div>

            <TactileButton
              type="button"
              className="otl-hold-unlock-btn"
              onClick={handleConsentAccept}
              aria-label="Tôi hiểu rồi, tiếp tục tới phong bì"
            >
              <div className="hold-btn-content">
                <svg className="hold-lock-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
                  <path d="M9 11V7a3 3 0 0 1 6 0v4" />
                  <rect x="6" y="11" width="12" height="9" rx="1.5" fill="currentColor" />
                </svg>
                <span>Tôi hiểu rồi!</span>
              </div>
            </TactileButton>
          </div>
        </div>
      )}
    </main>
  );
}

function isEnvelopeState(state: SecretSceneState): state is EnvelopeSceneState {
  return (
    state === "sealed" ||
    state === "opening" ||
    state === "revealed" ||
    state === "closing" ||
    state === "burning"
  );
}

function getStatusData(state: SecretSceneState, message?: string): { state: StatusSceneState; message?: string } | null {
  if (
    state === "loading" ||
    state === "vanished" ||
    state === "expired" ||
    state === "consumed" ||
    state === "error"
  ) {
    return { state, message };
  }

  return null;
}
