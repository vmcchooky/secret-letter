import React, { useState, useEffect, useRef } from "react";
import type { SecretSceneState } from "../../lib/types";
import { TactileButton } from "../TactileButton";
import { useReducedMotion } from "../../hooks/useReducedMotion";

type SecretStatusScreenProps = {
  state: Extract<SecretSceneState, "loading" | "vanished" | "expired" | "consumed" | "error">;
  message?: string;
  onRetry?: () => void;
  transitionState?: "enter" | "exit";
};

const copy: Record<
  Extract<SecretSceneState, "loading" | "vanished" | "expired" | "consumed" | "error">,
  { title: React.ReactNode; body: string }
> = {
  loading: {
    title: (
      <>
        Đang tìm <span style={{ whiteSpace: "nowrap" }}>lá thư...</span>
      </>
    ),
    body: "Ánh sáng đang dò theo dấu niêm phong.",
  },
  vanished: {
    title: (
      <>
        <span style={{ whiteSpace: "nowrap" }}>Lá thư</span> đã tan vào{" "}
        <span style={{ whiteSpace: "nowrap" }}>ánh sao.</span>
      </>
    ),
    body: "Những dòng cuối đã rời khỏi phiên này.",
  },
  consumed: {
    title: "Dấu phép đã nguội.",
    body: "Liên kết này đã được mở một lần trước đó.",
  },
  expired: {
    title: (
      <>
        <span style={{ whiteSpace: "nowrap" }}>Lá thư</span> đã lỡ thời khắc mở.
      </>
    ),
    body: "Niêm phong còn đó, nhưng bí mật đã hết hạn.",
  },
  error: {
    title: (
      <>
        Ánh sáng chưa chạm tới <span style={{ whiteSpace: "nowrap" }}>lá thư.</span>
      </>
    ),
    body: "Giải mã thất bại hoặc liên kết không hợp lệ. Hãy thử lại.",
  },
};

type ShootingStar = {
  id: number;
  top: number;
  left: number;
  length: number;
  duration: number;
};

export function SecretStatusScreen({ state, message, onRetry, transitionState = "enter" }: SecretStatusScreenProps) {
  const current = copy[state];
  const [shootingStars, setShootingStars] = useState<ShootingStar[]>([]);
  const [mousePos, setMousePos] = useState({ x: 50, y: 50 });
  const reducedMotion = useReducedMotion();
  const pointerFrameRef = useRef<number | null>(null);
  const pendingMousePosRef = useRef<{ x: number; y: number } | null>(null);

  const handleMouseMove = (e: React.MouseEvent<HTMLDivElement>) => {
    if (reducedMotion) {
      return;
    }

    const rect = e.currentTarget.getBoundingClientRect();
    const x = ((e.clientX - rect.left) / rect.width) * 100;
    const y = ((e.clientY - rect.top) / rect.height) * 100;
    pendingMousePosRef.current = { x, y };

    if (pointerFrameRef.current !== null) {
      return;
    }

    pointerFrameRef.current = window.requestAnimationFrame(() => {
      pointerFrameRef.current = null;
      if (pendingMousePosRef.current) {
        setMousePos(pendingMousePosRef.current);
        pendingMousePosRef.current = null;
      }
    });
  };

  // Automatic Shooting Star Generator running every 2 seconds on status screens!
  useEffect(() => {
    if (state === "loading" || reducedMotion) {
      setShootingStars([]);
      return;
    }

    const cleanupTimers = new Set<number>();
    const spawnStar = () => {
      const id = Date.now() + Math.random();
      const top = Math.random() * 45;      // random top position in the upper sky (0% to 45%)
      const left = Math.random() * 85;     // random left position (0% to 85%)
      const length = 110 + Math.random() * 140; // random tail length (110px to 250px)
      const duration = 0.8 + Math.random() * 0.7; // random sliding speed (0.8s to 1.5s)

      const newStar = { id, top, left, length, duration };
      setShootingStars((currentStars) => [...currentStars, newStar]);

      // Safely cleanup the star after its sliding animation ends
      const cleanupTimer = window.setTimeout(() => {
        setShootingStars((currentStars) => currentStars.filter((star) => star.id !== id));
        cleanupTimers.delete(cleanupTimer);
      }, duration * 1000 + 100);
      cleanupTimers.add(cleanupTimer);
    };

    // Spawn first star quickly, then run interval every 2000ms
    const firstSpawn = window.setTimeout(spawnStar, 600);
    const interval = window.setInterval(spawnStar, 2000);

    return () => {
      window.clearTimeout(firstSpawn);
      window.clearInterval(interval);
      cleanupTimers.forEach((timer) => window.clearTimeout(timer));
      cleanupTimers.clear();
    };
  }, [reducedMotion, state]);

  useEffect(() => {
    return () => {
      if (pointerFrameRef.current !== null) {
        window.cancelAnimationFrame(pointerFrameRef.current);
      }
    };
  }, []);

  return (
    <div
      className={`secret-status secret-status-${state}`}
      data-transition={transitionState}
      role="status"
      aria-live="polite"
      onMouseMove={handleMouseMove}
      style={{
        ["--mouse-x" as any]: `${mousePos.x}%`,
        ["--mouse-y" as any]: `${mousePos.y}%`,
      }}
    >
      {/* Interactive mouse spotlight glow overlay */}
      <div className="status-spotlight" />

      {/* Background shooting stars rendered absolutely inside the status viewport */}
      {shootingStars.map((star) => (
        <span
          key={star.id}
          className="shooting-star"
          style={{
            top: `${star.top}%`,
            left: `${star.left}%`,
            ["--star-length" as any]: `${star.length}px`,
            ["--star-duration" as any]: `${star.duration}s`,
          }}
        />
      ))}

      <div className="status-visual">
        {(state === "consumed" || state === "vanished") && (
          <>
            <div className="status-starlight" />
            <div className="status-orbit status-orbit-one" />
            <div className="status-orbit status-orbit-two" />
            <div className="status-star-cluster">
              <span />
              <span />
              <span />
              <span />
              <span />
            </div>
          </>
        )}

        {state === "expired" && (
          <div className="expired-hourglass">
            <div className="expired-dust-field">
              <span className="dust-particle dust-1" />
              <span className="dust-particle dust-2" />
              <span className="dust-particle dust-3" />
              <span className="dust-particle dust-4" />
            </div>
            <svg className="expired-hourglass-svg" viewBox="0 0 100 150">
              <defs>
                <linearGradient id="hourglassGold" x1="0%" y1="0%" x2="100%" y2="100%">
                  <stop offset="0%" stopColor="var(--secret-gold)" stopOpacity="0.8" />
                  <stop offset="100%" stopColor="#cca03f" stopOpacity="0.4" />
                </linearGradient>
              </defs>
              
              {/* Hourglass Glass Body */}
              <path
                d="M 25 20 L 75 20 L 75 35 C 75 62, 53 70, 53 75 C 53 80, 75 88, 75 115 L 75 130 L 25 130 L 25 115 C 25 88, 47 80, 47 75 C 47 70, 25 62, 25 35 Z"
                className="hourglass-glass"
              />
              
              {/* Wooden Cap plates */}
              <rect x="20" y="10" width="60" height="10" rx="2" className="hourglass-cap" />
              <rect x="20" y="130" width="60" height="10" rx="2" className="hourglass-cap" />
              
              {/* Empty top chamber internal outline */}
              <path d="M 28 35 C 28 58, 46 66, 48 72 L 52 72 C 54 66, 72 58, 72 35 Z" className="hourglass-chamber-line" />
              
              {/* Accumulated Sand at the bottom (flat cold void pink-red color) */}
              <path d="M 27 115 C 38 105, 62 105, 73 115 L 73 129 L 27 129 Z" className="hourglass-sand-cold" />
              
              {/* Decay crack across the upper glass */}
              <path d="M 32 40 L 40 50 L 36 62" className="glass-decay-crack" />
            </svg>
          </div>
        )}

        {/* Shattered Constellation Error Redesign - anchor visual using a broken celestial map */}
        {state === "error" && (
          <div className="error-constellation">
            <div className="constellation-bg-glow" />
            <svg className="shattered-constellation-svg" viewBox="0 0 200 200">
              <defs>
                <radialGradient id="coreGlowOuter" cx="50%" cy="50%" r="50%">
                  <stop offset="0%" stopColor="#e67a86" stopOpacity="0.55" />
                  <stop offset="30%" stopColor="#e67a86" stopOpacity="0.25" />
                  <stop offset="100%" stopColor="#e67a86" stopOpacity="0" />
                </radialGradient>
              </defs>
              {/* Broken, dashed constellation connections */}
              <line x1="30" y1="50" x2="100" y2="100" className="const-line const-line-broken-1" />
              <line x1="100" y1="100" x2="170" y2="60" className="const-line const-line-broken-2" />
              <line x1="170" y1="60" x2="140" y2="150" className="const-line const-line-broken-3" />
              <line x1="140" y1="150" x2="60" y2="140" className="const-line const-line-broken-4" />
              <line x1="60" y1="140" x2="30" y2="50" className="const-line const-line-broken-5" />
              
              {/* Electro-sparking unstable jagged paths */}
              <path d="M 30 50 L 52 64 L 46 76 L 78 82 L 100 100" className="electro-spark-line spark-delay-1" />
              <path d="M 170 60 L 148 72 L 154 84 L 126 88 L 100 100" className="electro-spark-line spark-delay-2" />
              <path d="M 140 150 L 118 138 L 124 148 L 92 134 L 60 140" className="electro-spark-line spark-delay-3" />
              
              {/* Glowing, out-of-alignment star nodes */}
              <g className="const-star const-star-1">
                <circle cx="30" cy="50" r="3" />
                <path d="M30 44 L31 49 L36 50 L31 51 L30 56 L29 51 L24 50 L29 49 Z" className="star-sparkle" />
              </g>
              <g className="const-star const-star-2">
                <circle cx="170" cy="60" r="3" />
                <path d="M170 54 L171 59 L176 60 L171 61 L170 66 L169 61 L164 60 L169 59 Z" className="star-sparkle" />
              </g>
              <g className="const-star const-star-3">
                <circle cx="140" cy="150" r="3" />
                <path d="M140 144 L141 149 L146 150 L141 151 L140 156 L139 151 L134 150 L139 149 Z" className="star-sparkle" />
              </g>
              <g className="const-star const-star-4">
                <circle cx="60" cy="140" r="3" />
                <path d="M60 134 L61 139 L66 140 L61 141 L60 146 L59 141 L54 140 L59 139 Z" className="star-sparkle" />
              </g>
              
              {/* Collapsed core cosmic black hole / shattered void */}
              <g className="const-core">
                <circle cx="100" cy="100" r="22" className="core-glow-outer" />
                <circle cx="100" cy="100" r="14" className="core-glow-inner" />
                <circle cx="100" cy="100" r="6" className="core-void" />
                {/* Cracks in core */}
                <path d="M96 92 L100 100 L95 107 M104 93 L100 100 L106 109" className="core-crack" />
              </g>
            </svg>
          </div>
        )}

        {state === "loading" && (
          <div className="cosmic-radar">
            <div className="radar-ring radar-ring-1" />
            <div className="radar-ring radar-ring-2" />
            <div className="radar-ring radar-ring-3" />
            <div className="radar-sweep" />
            <div className="radar-stars">
              <div className="radar-star-node node-1">
                <span className="star-four-points" />
              </div>
              <div className="radar-star-node node-2">
                <span className="star-four-points" />
              </div>
              <div className="radar-star-node node-3">
                <span className="star-four-points" />
              </div>
            </div>
            <div className="radar-center-glow" />
          </div>
        )}
      </div>

      <h1>{current.title}</h1>
      <p className="status-body-text">{message || current.body}</p>

      {state === "error" && onRetry && (
        <div className="status-actions">
          <TactileButton type="button" className="qx-btn qx-btn-primary" onClick={onRetry}>
            Thử lại
          </TactileButton>
          <a className="qx-btn qx-btn-secondary" href="/">Tạo thư mới</a>
        </div>
      )}
    </div>
  );
}
