import { useState, useEffect, useRef, type CSSProperties, type KeyboardEvent, type MouseEvent, type PointerEvent, type RefObject } from "react";
import type { EnvelopeTimelineRefs } from "../../hooks/useEnvelopeTimeline";
import { useTactilePress } from "../../hooks/useTactilePress";

type EnvelopeProps = {
  state: "sealed" | "opening" | "revealed" | "closing" | "burning";
  refs: Pick<EnvelopeTimelineRefs, "envelopeRef" | "flapRef" | "letterRef" | "contentRef">;
  secretContent: string;
  expiresAt?: string;
  onOpen: () => void;
  onOpenHoldStart: () => void;
  onOpenHoldCancel: () => void;
  openHoldProgressRef: RefObject<HTMLSpanElement | null>;
  isOpenHolding: boolean;
  canHoldOpen: boolean;
  onClose: () => void;
};

type StarrySpark = {
  id: number;
  x: number;
  y: number;
  dx: number;
  dy: number;
  rotate: number;
  scale: number;
};

type DestroyShard = {
  id: number;
  dx: number;
  dy: number;
  rotate: number;
  delay: number;
  size: number;
};

const DESTROY_ARM_MS = 430;

// Synchronous and bulletproof copy-to-clipboard function
const performCopyToClipboard = (text: string): boolean => {
  let success = false;
  try {
    const textArea = document.createElement("textarea");
    textArea.value = text;
    // Hide the element from view
    textArea.style.position = "fixed";
    textArea.style.left = "-99999px";
    textArea.style.top = "-99999px";
    textArea.style.width = "2em";
    textArea.style.height = "2em";
    textArea.style.padding = "0";
    textArea.style.border = "none";
    textArea.style.outline = "none";
    textArea.style.boxShadow = "none";
    textArea.style.background = "transparent";
    document.body.appendChild(textArea);
    textArea.focus();
    textArea.select();
    
    success = document.execCommand("copy");
    document.body.removeChild(textArea);
  } catch (err) {
    console.error("Synchronous fallback copy failed", err);
  }

  // Also attempt modern Clipboard API as secondary fallback (async, runs in parallel)
  if (!success && navigator.clipboard) {
    navigator.clipboard.writeText(text).catch((err) => {
      console.error("Modern Clipboard API write failed", err);
    });
  }

  return success || !!navigator.clipboard;
};

export function Envelope({
  state,
  refs,
  secretContent,
  expiresAt,
  onOpen,
  onOpenHoldStart,
  onOpenHoldCancel,
  openHoldProgressRef,
  isOpenHolding,
  canHoldOpen,
  onClose,
}: EnvelopeProps) {
  const [copied, setCopied] = useState(false);
  const [sparks, setSparks] = useState<StarrySpark[]>([]);
  const [destroyArmed, setDestroyArmed] = useState(false);
  const destroyTimerRef = useRef<number | null>(null);
  const isSealed = state === "sealed";
  const isReading = state === "revealed";
  const savePress = useTactilePress<HTMLButtonElement>(!isReading);
  const destroyPress = useTactilePress<HTMLButtonElement>(!isReading);

  const destroyShards: DestroyShard[] = [
    { id: 1, dx: -54, dy: -66, rotate: -26, delay: 0, size: 8 },
    { id: 2, dx: 58, dy: -58, rotate: 34, delay: 20, size: 7 },
    { id: 3, dx: -72, dy: 8, rotate: -62, delay: 36, size: 5 },
    { id: 4, dx: 74, dy: 10, rotate: 58, delay: 48, size: 6 },
    { id: 5, dx: -34, dy: 58, rotate: 18, delay: 68, size: 6 },
    { id: 6, dx: 36, dy: 64, rotate: -18, delay: 78, size: 5 },
  ];

  useEffect(() => {
    if (copied) {
      const timer = setTimeout(() => setCopied(false), 1100);
      return () => clearTimeout(timer);
    }
  }, [copied]);

  useEffect(() => {
    if (sparks.length > 0) {
      const timer = setTimeout(() => setSparks([]), 1500);
      return () => clearTimeout(timer);
    }
  }, [sparks]);

  useEffect(() => {
    return () => {
      if (destroyTimerRef.current !== null) {
        window.clearTimeout(destroyTimerRef.current);
      }
    };
  }, []);

  const handleCopy = (e: MouseEvent<HTMLButtonElement>) => {
    e.stopPropagation();
    if (!isReading) return;

    // Perform copy
    performCopyToClipboard(secretContent);
    setCopied(true);

    // Visual sparks at the button coordinates
    const rect = e.currentTarget.getBoundingClientRect();
    const clickX = e.clientX - rect.left + e.currentTarget.offsetLeft;
    const clickY = e.clientY - rect.top + e.currentTarget.offsetTop;

    const newSparks = Array.from({ length: 14 }).map((_, i) => ({
      id: Date.now() + i,
      x: clickX,
      y: clickY,
      dx: (Math.random() - 0.5) * 240,
      dy: (Math.random() - 0.5) * 240,
      rotate: (Math.random() - 0.5) * 220,
      scale: 0.72 + Math.random() * 0.62,
    }));
    setSparks(newSparks);
  };

  const handleDestroyClick = (e: MouseEvent<HTMLButtonElement>) => {
    e.stopPropagation();
    if (!isReading || destroyArmed) return;

    setDestroyArmed(true);
    destroyPress.clearPress();
    destroyTimerRef.current = window.setTimeout(() => {
      destroyTimerRef.current = null;
      onClose();
    }, DESTROY_ARM_MS);
  };

  const hint =
    state === "sealed"
      ? canHoldOpen
        ? "Nhấn giữ phong bì để mở lá thư."
        : "Đọc cảnh báo trước khi chạm vào phong bì."
      : state === "opening"
        ? "Phong thư đang tan khỏi mặt giấy."
        : state === "closing"
          ? "Ấn chú đang gom những dòng cuối."
          : "Lá thư đang tan thành bụi sao.";

  const handleEnvelopePointerDown = (e: PointerEvent<HTMLDivElement>) => {
    if (!isSealed || !canHoldOpen) return;
    e.preventDefault();
    e.currentTarget.setPointerCapture(e.pointerId);
    onOpenHoldStart();
  };

  const handleEnvelopePointerEnd = (e: PointerEvent<HTMLDivElement>) => {
    if (!isSealed || !canHoldOpen) return;
    if (e.currentTarget.hasPointerCapture(e.pointerId)) {
      e.currentTarget.releasePointerCapture(e.pointerId);
    }
    onOpenHoldCancel();
  };

  const handleEnvelopeKeyDown = (e: KeyboardEvent<HTMLDivElement>) => {
    if (!isSealed || !canHoldOpen) return;
    if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      onOpen();
    }
  };

  return (
    <div className="secret-letter-stage" aria-live="polite">
      <div
        ref={refs.envelopeRef}
        role="button"
        tabIndex={isSealed ? 0 : -1}
        className={`secret-envelope secret-envelope-${state} ${isOpenHolding ? "secret-envelope-is-holding" : ""} ${destroyArmed && isReading ? "secret-envelope-destroy-armed" : ""}`}
        aria-label={isSealed ? "Mở lá thư bí mật" : "Phong bì bí mật"}
        aria-describedby={isSealed ? "secret-envelope-hint" : undefined}
        onPointerDown={handleEnvelopePointerDown}
        onPointerUp={handleEnvelopePointerEnd}
        onPointerCancel={handleEnvelopePointerEnd}
        onPointerLeave={handleEnvelopePointerEnd}
        onKeyDown={handleEnvelopeKeyDown}
        onContextMenu={isSealed ? (e) => e.preventDefault() : undefined}
      >
        {state === "sealed" || state === "opening" ? (
          <div className="envelope-back" />
        ) : null}

        <div
          ref={refs.letterRef}
          className={`secret-paper ${copied ? "play-shine play-border-flash" : ""} ${destroyArmed && isReading ? "is-destroy-armed" : ""}`}
          aria-hidden={!isReading}
        >
          {/* Pixel-perfect vector border ray trace animation on copy click */}
          {copied && (
            <svg className="border-trace-svg" viewBox="0 0 480 410" preserveAspectRatio="none">
              <path d="M 240 410 L 0 410 L 0 0 L 240 0" className="trace-path trace-path-left" />
              <path d="M 240 410 L 480 410 L 480 0 L 240 0" className="trace-path trace-path-right" />
            </svg>
          )}

          <div ref={refs.contentRef} className="secret-paper-content view-mode-vintage">
            <pre>{secretContent}</pre>
          </div>

          {expiresAt && (
            <p className="secret-paper-meta">
              Hết hạn {new Date(expiresAt).toLocaleString("vi-VN")}
            </p>
          )}



          {/* Artistic bottom-right Action Icons */}
          {isReading && (
            <div className="letter-actions" onClick={(e) => e.stopPropagation()}>
              <button
                {...savePress.pressProps}
                type="button"
                className={`magic-mind-etch-btn ${copied ? "copied" : ""} ${savePress.isPressing ? "is-pressing" : ""}`}
                data-pressed={savePress.isPressing ? "true" : undefined}
                onClick={handleCopy}
                onContextMenu={(e) => e.preventDefault()}
                title="Lưu giữ nội dung mật thư"
                aria-label="Lưu giữ mật thư"
              >
                <svg className="etch-btn-icon" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  {copied ? (
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M5 13l4 4L19 7" />
                  ) : (
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 3v4M3 5h4M6 17v4m-2-2h4m5-16l2.286 6.857L21 12l-5.714 2.143L13 21l-2.286-6.857L5 12l5.714-2.143L13 3z" />
                  )}
                </svg>
              </button>
            </div>
          )}

          {/* Magical Starry Sparks Burst */}
          {sparks.map((spark) => (
            <span
              key={spark.id}
              className="starry-spark"
              style={{
                left: spark.x,
                top: spark.y,
                "--dx": `${spark.dx}px`,
                "--dy": `${spark.dy}px`,
                "--spark-rotate": `${spark.rotate}deg`,
                "--spark-scale": spark.scale,
              } as CSSProperties}
            />
          ))}

          {/* Small highly artistic message/toast overlay to confirm preservation */}
          {copied && (
            <div className="artistic-toast">
              <span>Đã lưu giữ!</span>
            </div>
          )}
        </div>
        
        {state === "sealed" || state === "opening" ? (
          <>
            <div className="envelope-pockets-wrapper" style={{ position: 'absolute', inset: 0, zIndex: 6 }}>
              <div className="envelope-pocket envelope-pocket-left" />
              <div className="envelope-pocket envelope-pocket-right" />
              <div className="envelope-pocket envelope-pocket-bottom" />
            </div>
            <div ref={refs.flapRef} className="envelope-flap" />
            <div className="envelope-seal">
              <svg viewBox="0 0 24 24" fill="currentColor" style={{ width: '26px', opacity: 0.9 }}>
                <path d="M12 0C12 8 16 12 24 12C16 12 12 16 12 24C12 16 8 12 0 12C8 12 12 8 12 0Z" />
              </svg>
            </div>
            {isSealed && canHoldOpen && (
              <div className="envelope-hold-meter" aria-hidden="true">
                <span ref={openHoldProgressRef} />
              </div>
            )}
          </>
        ) : null}
      </div>

      {isReading ? (
        <button
          {...destroyPress.pressProps}
          type="button"
          className={`secret-destroy-button magic-star-seal ${destroyPress.isPressing ? "is-pressing" : ""} ${destroyArmed ? "is-destroying" : ""}`}
          data-pressed={destroyPress.isPressing ? "true" : undefined}
          onClick={handleDestroyClick}
          onContextMenu={(e) => e.preventDefault()}
          disabled={destroyArmed}
          aria-label="Tiêu hủy lá thư"
        >
          <div className="seal-cosmic-aura" />
          <div className="destroy-shockwave destroy-shockwave-one" aria-hidden="true" />
          <div className="destroy-shockwave destroy-shockwave-two" aria-hidden="true" />
          <div className="seal-orbit-ring">
            <span className="orbit-moon" />
          </div>
          <div className="seal-sparkle-core">
            <span className="star-four-points" />
          </div>
          {destroyShards.map((shard) => (
            <span
              key={shard.id}
              className="destroy-star-shard"
              aria-hidden="true"
              style={{
                "--destroy-dx": `${shard.dx}px`,
                "--destroy-dy": `${shard.dy}px`,
                "--destroy-rotate": `${shard.rotate}deg`,
                "--destroy-delay": `${shard.delay}ms`,
                "--destroy-size": `${shard.size}px`,
              } as CSSProperties}
            />
          ))}
          <span className="spell-button-star spell-button-star-one" aria-hidden="true" />
          <span className="spell-button-star spell-button-star-two" aria-hidden="true" />
          <span className="spell-button-star spell-button-star-three" aria-hidden="true" />
          <span className="spell-button-star spell-button-star-four" aria-hidden="true" />
        </button>
      ) : (
        <p id="secret-envelope-hint" className="secret-letter-hint">{hint}</p>
      )}
    </div>
  );
}
