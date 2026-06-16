import { useState, useEffect, useRef, useCallback, type CSSProperties, type ChangeEvent, type FormEvent, type KeyboardEvent, type MouseEvent } from "react";
import { createPortal } from "react-dom";
import { TactileButton } from "./TactileButton";
import { createSecret } from "../lib/api";
import {
  generateKey,
  generateNonce,
  encryptSecret,
  encodeBase64Url,
  exportKeyToBase64Url,
} from "../lib/crypto";
import { splitSecretLink } from "../lib/secretLink";
import { clampUtf8Text } from "../lib/utf8";
import { useTactilePress } from "../hooks/useTactilePress";

const TTL_OPTIONS = [
  { value: 3600, label: "1 giờ" },
  { value: 86400, label: "24 giờ" },
  { value: 604800, label: "7 ngày" },
];

const MAX_PLAINTEXT_SIZE = 10 * 1024; // 10KB

export function CreateSecretForm() {
  const [plaintext, setPlaintext] = useState("");
  const [plaintextBytes, setPlaintextBytes] = useState(0);
  const [ttl, setTtl] = useState(3600);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [secretLink, setSecretLink] = useState("");
  const [copyStatus, setCopyStatus] = useState("");
  const [shareFeedback, setShareFeedback] = useState("");
  const [showQr, setShowQr] = useState(false);
  const [showShareMenu, setShowShareMenu] = useState(false);
  const [isFormExiting, setIsFormExiting] = useState(false);
  const [isResultExiting, setIsResultExiting] = useState(false);
  const linkTokenPress = useTactilePress<HTMLDivElement>(!secretLink);

  const qrCanvasRef = useRef<HTMLCanvasElement | null>(null);
  const formTransitionTimerRef = useRef<number | null>(null);
  const resultTransitionTimerRef = useRef<number | null>(null);
  const qrPresence = useDelayedPresence(showQr, 180);
  const sharePresence = useDelayedPresence(showShareMenu, 180);
  const secretLinkParts = splitSecretLink(secretLink);

  useEffect(() => {
    if (!secretLink || !showQr || !qrPresence.shouldRender || !qrCanvasRef.current) {
      return;
    }

    let cancelled = false;

    void import("qrcode")
      .then((QRCodeModule) => {
        if (cancelled || !qrCanvasRef.current) {
          return;
        }

        const QRCode = (QRCodeModule as any).default || QRCodeModule;

        QRCode.toCanvas(
          qrCanvasRef.current,
          secretLink,
          {
            width: 180,
            margin: 1.5,
            color: {
              dark: "#171016",
              light: "#ffffff",
            },
          },
          (err: any) => {
            if (err) console.error("QR Code generation error:", err);
          }
        );
      })
      .catch((err) => {
        if (!cancelled) {
          console.error("QR Code module load error:", err);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [qrPresence.shouldRender, secretLink, showQr]);

  useEffect(() => {
    return () => {
      if (formTransitionTimerRef.current !== null) {
        window.clearTimeout(formTransitionTimerRef.current);
      }
      if (resultTransitionTimerRef.current !== null) {
        window.clearTimeout(resultTransitionTimerRef.current);
      }
    };
  }, []);

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError("");
    setSecretLink("");
    setCopyStatus("");

    // Validate plaintext size
    if (plaintextBytes > MAX_PLAINTEXT_SIZE) {
      setError(`Nội dung vượt quá giới hạn ${MAX_PLAINTEXT_SIZE / 1024}KB`);
      return;
    }

    if (!plaintext.trim()) {
      setError("Vui lòng nhập nội dung bí mật");
      return;
    }

    setLoading(true);

    try {
      // Generate encryption key and nonce
      const key = await generateKey();
      const nonce = generateNonce();

      // Encrypt the plaintext
      const ciphertextBytes = await encryptSecret(plaintext, key, nonce);

      // Encode to base64url
      const ciphertext = encodeBase64Url(ciphertextBytes);
      const nonceB64 = encodeBase64Url(nonce);

      // Create secret via API
      const response = await createSecret({
        ciphertext,
        nonce: nonceB64,
        algorithm: "AES-GCM",
        ttlSeconds: ttl,
      });

      // Export key for URL fragment
      const keyB64 = await exportKeyToBase64Url(key);

      // Build secret link
      const baseUrl =
        import.meta.env.VITE_PUBLIC_SECRET_ORIGIN?.trim().replace(/\/+$/, "") ||
        window.location.origin;
      const token = response.token || response.secretId;
      const link = `${baseUrl}/${token}#${keyB64}`;

      setIsFormExiting(true);
      formTransitionTimerRef.current = window.setTimeout(() => {
        formTransitionTimerRef.current = null;
        setSecretLink(link);
        setPlaintext(""); // Clear form
        setPlaintextBytes(0);
        setIsFormExiting(false);
      }, 150);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : "Không thể tạo liên kết bí mật"
      );
    } finally {
      setLoading(false);
    }
  };

  const setTransientShareFeedback = (message: string) => {
    setShareFeedback(message);
    window.setTimeout(() => setShareFeedback(""), 2200);
  };

  const copyToClipboard = async (shareMessage?: string) => {
    try {
      await navigator.clipboard.writeText(secretLink);
      setCopyStatus("copied");
      if (shareMessage) {
        setTransientShareFeedback(shareMessage);
      }
      setTimeout(() => setCopyStatus(""), 1500);
    } catch (err) {
      setCopyStatus("error");
      if (shareMessage) {
        setTransientShareFeedback("Không thể sao chép liên kết này.");
      }
      setTimeout(() => setCopyStatus(""), 2000);
    }
  };

  const copySharePart = async (value: string, successMessage: string) => {
    if (!value) {
      setTransientShareFeedback("Thiếu dữ liệu chia sẻ để sao chép.");
      return;
    }

    try {
      await navigator.clipboard.writeText(value);
      setTransientShareFeedback(successMessage);
    } catch {
      setTransientShareFeedback("Không thể sao chép mục chia sẻ này.");
    }
  };

  const closeShareMenu = () => {
    setShowShareMenu(false);
    setShareFeedback("");
  };

  const openShareMenu = () => {
    setShareFeedback("");
    setShowShareMenu(true);
  };

  const qrPressTimerRef = useRef<number | null>(null);

  const downloadQrCode = useCallback(() => {
    if (!qrCanvasRef.current) return;
    const dataUrl = qrCanvasRef.current.toDataURL("image/png");
    const link = document.createElement("a");
    link.download = "secret-qr.png";
    link.href = dataUrl;
    link.click();
  }, []);

  const handleQrPointerDown = () => {
    qrPressTimerRef.current = window.setTimeout(() => {
      downloadQrCode();
    }, 600); // 600ms for long press
  };

  const handleQrPointerUpOrLeave = () => {
    if (qrPressTimerRef.current !== null) {
      window.clearTimeout(qrPressTimerRef.current);
      qrPressTimerRef.current = null;
    }
  };

  const copySecretLinkFromKeyboard = (event: KeyboardEvent<HTMLDivElement>) => {
    if (event.key !== "Enter" && event.key !== " ") {
      return;
    }

    event.preventDefault();
    copyToClipboard();
  };

  const blockDecorativeContextMenu = (event: MouseEvent<HTMLElement>) => {
    const target = event.target;
    if (!(target instanceof Element)) {
      event.preventDefault();
      return;
    }

    if (target.closest("input, textarea, canvas, a, [contenteditable='true']")) {
      return;
    }

    event.preventDefault();
  };

  const resetToCompose = () => {
    if (isResultExiting) return;

    setIsResultExiting(true);
    resultTransitionTimerRef.current = window.setTimeout(() => {
      resultTransitionTimerRef.current = null;
      setSecretLink("");
      setPlaintext("");
      setPlaintextBytes(0);
      setCopyStatus("");
      setShowQr(false);
      closeShareMenu();
      setIsResultExiting(false);
    }, 160);
  };

  const bytePercentage = Math.min((plaintextBytes / MAX_PLAINTEXT_SIZE) * 100, 100);
  const selectedTtlIndex = Math.max(0, TTL_OPTIONS.findIndex((option) => option.value === ttl));
  const ttlIndicatorStyle = {
    "--selected-offset": `calc(${selectedTtlIndex} * (100% + var(--segment-gap)))`,
  } as CSSProperties;

  let progressBarColorClass = "qx-progress-bar-green";
  if (bytePercentage >= 90) {
    progressBarColorClass = "qx-progress-bar-red";
  } else if (bytePercentage >= 60) {
    progressBarColorClass = "qx-progress-bar-amber";
  }

  const handleTextChange = (e: ChangeEvent<HTMLTextAreaElement>) => {
    const nextValue = clampUtf8Text(e.target.value, MAX_PLAINTEXT_SIZE);
    setPlaintext(nextValue.text);
    setPlaintextBytes(nextValue.byteLength);
  };

  return (
    <div className="create-secret-form">
      {!secretLink ? (
        <form
          onSubmit={handleSubmit}
          className={`secret-compose-form view-transition-panel ${isFormExiting ? "is-exiting" : ""}`}
          onContextMenu={blockDecorativeContextMenu}
        >
          <div className="qx-form-group">
            <div className="secret-label-row">
              <label className="qx-label" htmlFor="plaintext">Nội dung bí mật</label>
              <div
                className={`otl-secure-badge ${plaintext.length > 0 ? "is-encrypted" : "is-unencrypted"}`}
                aria-live="polite"
              >
                <svg className="otl-secure-lock" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
                  <path className="lock-shackle" d="M9 11V7a3 3 0 0 1 6 0v4" />
                  <rect className="lock-body" x="6" y="11" width="12" height="9" rx="1.5" fill="currentColor" />
                  <path className="lock-keyhole" d="M12 14v2" strokeWidth="1.5" />
                </svg>
                <span className="otl-secure-indicator-dot" />
              </div>
            </div>
            <div className={`secret-textarea-wrap ${plaintext.length > 0 ? "is-encrypting" : ""}`}>
              <textarea
                id="plaintext"
                className="qx-textarea qx-textarea-glass secret-textarea"
                value={plaintext}
                onChange={handleTextChange}
                placeholder="Soạn mật thư của bạn tại đây..."
                rows={7}
                disabled={loading || isFormExiting}
                aria-invalid={Boolean(error)}
                autoComplete="off"
                autoCorrect="off"
                autoCapitalize="none"
                spellCheck={false}
                data-lpignore="true"
                data-1p-ignore="true"
                data-form-type="other"
              />
              <div className="secret-textarea-footer">
                <span className="qx-byte-counter">
                  {plaintextBytes.toLocaleString()} / {MAX_PLAINTEXT_SIZE.toLocaleString()} bytes
                </span>
              </div>
              <div className="qx-progress-wrapper">
                <div className="qx-progress-track">
                  <div
                    className={`qx-progress-bar ${progressBarColorClass}`}
                    style={{ width: `${bytePercentage}%` }}
                  />
                </div>
              </div>
            </div>
          </div>

          <div className="qx-form-group">
            <label className="qx-label">Hết hạn sau</label>
            <div className="qx-segmented-container" style={ttlIndicatorStyle}>
              <span className="qx-segmented-indicator" aria-hidden="true" />
              {TTL_OPTIONS.map((option) => (
                <TactileButton
                  key={option.value}
                  type="button"
                  className={`qx-segmented-btn ${ttl === option.value ? "active" : ""}`}
                  onClick={() => setTtl(option.value)}
                  disabled={loading || isFormExiting}
                  aria-pressed={ttl === option.value}
                >
                  {option.label}
                </TactileButton>
              ))}
            </div>
          </div>

          {error && <div className="qx-error-message secret-form-error" role="alert">{error}</div>}

          <TactileButton
            type="submit"
            className="qx-btn qx-btn-primary qx-btn-lg secret-submit"
            disabled={loading || isFormExiting || !plaintext.trim()}
          >
            <svg aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor">
              {loading ? (
                <path d="M12 3v3m6.36-.36-2.12 2.12M21 12h-3m.36 6.36-2.12-2.12M12 21v-3m-6.36.36 2.12-2.12M3 12h3m-.36-6.36 2.12 2.12" />
              ) : (
                <path d="M12 2l2.4 7.2L22 12l-7.6 2.8L12 22l-2.4-7.2L2 12l7.6-2.8L12 2Z" />
              )}
            </svg>
            {loading ? "Đang mã hóa..." : "Tạo link bí mật"}
          </TactileButton>
        </form>
      ) : (
        <div
          className={`success-result animate-success view-transition-panel ${isResultExiting ? "is-exiting" : ""}`}
          aria-live="polite"
          onContextMenu={blockDecorativeContextMenu}
        >
          <div className="success-heading">
            <div className="success-seal-container">
              <div className="success-seal-ring success-seal-ring-1"></div>
              <div className="success-seal-ring success-seal-ring-2"></div>
              <div className="success-star-wax">
                <svg viewBox="0 0 24 24" fill="currentColor">
                  <path d="M12 0C12 8 16 12 24 12C16 12 12 16 12 24C12 16 8 12 0 12C8 12 12 8 12 0Z" />
                </svg>
              </div>
            </div>
            <h2>Mật thư đã niêm phong</h2>
          </div>

          <div
            {...linkTokenPress.pressProps}
            className={`secret-link-box parchment-token ${linkTokenPress.isPressing ? "is-pressing" : ""}`}
            data-pressed={linkTokenPress.isPressing ? "true" : undefined}
            onClick={() => { void copyToClipboard(); }}
            onKeyDown={copySecretLinkFromKeyboard}
            onContextMenu={blockDecorativeContextMenu}
            title="Nhấn để sao chép liên kết"
            role="button"
            tabIndex={0}
          >
            <div className="parchment-token-glow"></div>
            <svg className="parchment-lock-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
              <rect x="3" y="11" width="18" height="11" rx="2" ry="2"></rect>
              <path d="M7 11V7a5 5 0 0 1 10 0v4"></path>
            </svg>
            <input
              className="qx-input parchment-input"
              type="text"
              value={secretLink}
              readOnly
              aria-label="Secret link"
              autoComplete="off"
              autoCorrect="off"
              autoCapitalize="none"
              spellCheck={false}
              data-lpignore="true"
              data-1p-ignore="true"
              data-form-type="other"
            />
            <span className="parchment-copy-badge">
              {copyStatus === "copied" ? "Đã chép" : "Sao chép"}
            </span>
          </div>

          <div className="success-action-group">
            <TactileButton className="minimal-icon-action" onClick={() => { void copyToClipboard(); }} title={copyStatus === "copied" ? "Đã sao chép" : "Sao chép"}>
              <svg width="20" height="20" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                {copyStatus === "copied" ? (
                  <path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" />
                ) : (
                  <path strokeLinecap="round" strokeLinejoin="round" d="M8 5H6a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2v-1M8 5a2 2 0 002 2h2a2 2 0 002-2M8 5a2 2 0 012-2h2a2 2 0 012 2m0 0h2a2 2 0 012 2v3m2 4H10m0 0l3-3m-3 3l3 3" />
                )}
              </svg>
            </TactileButton>
            <TactileButton className="minimal-icon-action" onClick={openShareMenu} title="Chia sẻ an toàn">
              <svg width="20" height="20" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M8.684 13.342C8.886 12.938 9 12.482 9 12c0-.482-.114-.938-.316-1.342m0 2.684a3 3 0 110-2.684m0 2.684l6.632 3.316m-6.632-6l6.632-3.316m0 0a3 3 0 105.367-2.684 3 3 0 00-5.367 2.684zm0 9.316a3 3 0 105.368 2.684 3 3 0 00-5.368-2.684z" />
              </svg>
            </TactileButton>
            <TactileButton className="minimal-icon-action" onClick={() => setShowQr(true)} title="Mã QR">
              <svg width="20" height="20" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M12 4v1m6 11h2m-6 0h-2v4m0-11v3m0 0h.01M12 12h4.01M16 20h4M4 12h4m12 0h.01M5 8h2a1 1 0 001-1V5a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1zm12 0h2a1 1 0 001-1V5a1 1 0 00-1-1h-2a1 1 0 00-1 1v2a1 1 0 001 1zM5 20h2a1 1 0 001-1v-2a1 1 0 00-1-1H5a1 1 0 00-1 1v2a1 1 0 001 1z" />
              </svg>
            </TactileButton>
          </div>

          {qrPresence.shouldRender && createPortal(
            <div
              className={`otl-modal-overlay ${qrPresence.isExiting ? "is-exiting" : ""}`}
              onClick={() => setShowQr(false)}
              onContextMenu={blockDecorativeContextMenu}
            >
              <div className="otl-modal-card" onClick={(e) => e.stopPropagation()} onContextMenu={blockDecorativeContextMenu}>
                <h3>Mã QR Giải Mã</h3>
                <div 
                  className="otl-qrcode-canvas-wrapper"
                  onPointerDown={handleQrPointerDown}
                  onPointerUp={handleQrPointerUpOrLeave}
                  onPointerLeave={handleQrPointerUpOrLeave}
                  onPointerCancel={handleQrPointerUpOrLeave}
                  style={{ touchAction: "none" }}
                >
                  <canvas ref={qrCanvasRef} />
                </div>
                <p className="otl-modal-hint">Nhấn giữ để tải xuống mã</p>
                <TactileButton className="otl-modal-close-btn" onClick={() => setShowQr(false)}>Đóng</TactileButton>
              </div>
            </div>,
            document.body
          )}

          {sharePresence.shouldRender && createPortal(
            <div
              className={`otl-modal-overlay ${sharePresence.isExiting ? "is-exiting" : ""}`}
              onClick={closeShareMenu}
              onContextMenu={blockDecorativeContextMenu}
            >
              <div className="otl-modal-card share-card" onClick={(e) => e.stopPropagation()} onContextMenu={blockDecorativeContextMenu}>
                <h3 style={{ fontFamily: "var(--qx-font-ui, Inter, sans-serif)", fontStyle: "normal", fontWeight: 600, fontSize: "1.25rem" }}>Chia sẻ an toàn</h3>
                <p style={{ margin: "0 0 1rem", color: "var(--otl-muted)", lineHeight: 1.5, fontSize: "0.9rem" }}>
                  <strong>Tại sao phải sao chép thủ công?</strong> Các ứng dụng nhắn tin (Zalo, Telegram...) thường tự động quét link để tạo ảnh xem trước. Quá trình này có nguy cơ làm rò rỉ khóa giải mã lên máy chủ của họ.
                  <br /><br />
                  Để bảo mật tuyệt đối, hãy sao chép toàn bộ liên kết, hoặc an toàn nhất là gửi URL và Khóa qua hai kênh nhắn tin khác nhau.
                </p>
                <div className="otl-share-grid">
                  <TactileButton className="otl-share-item-btn" onClick={() => { void copyToClipboard("Đã sao chép liên kết đầy đủ."); }}>
                    <div className="share-icon-circle bg-gold-gradient">
                      <svg width="18" height="18" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                        <path strokeLinecap="round" strokeLinejoin="round" d="M8 5H6a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2v-1M8 5a2 2 0 002 2h2a2 2 0 002-2M8 5a2 2 0 012-2h2a2 2 0 012 2m0 0h2a2 2 0 012 2v3m2 4H10m0 0l3-3m-3 3l3 3" />
                      </svg>
                    </div>
                    <span>Link đầy đủ</span>
                  </TactileButton>
                  <TactileButton className="otl-share-item-btn" onClick={() => { void copySharePart(secretLinkParts.publicUrl, "Đã sao chép URL không kèm khóa."); }}>
                    <div className="share-icon-circle bg-telegram-gradient">
                      <svg width="18" height="18" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                        <path strokeLinecap="round" strokeLinejoin="round" d="M4 12h16M13 5l7 7-7 7" />
                      </svg>
                    </div>
                    <span>Chỉ URL</span>
                  </TactileButton>
                  <TactileButton className="otl-share-item-btn" onClick={() => { void copySharePart(secretLinkParts.fragmentKey, "Đã sao chép khóa giải mã."); }}>
                    <div className="share-icon-circle bg-zalo-gradient">
                      <svg width="18" height="18" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                        <path strokeLinecap="round" strokeLinejoin="round" d="M7 10V7a5 5 0 1110 0v3m-8 0h6m-6 0a2 2 0 00-2 2v5a2 2 0 002 2h6a2 2 0 002-2v-5a2 2 0 00-2-2m-6 0V9a3 3 0 116 0v1" />
                      </svg>
                    </div>
                    <span>Chỉ khóa</span>
                  </TactileButton>
                </div>
                {shareFeedback && (
                  <p role="status" style={{ margin: "1rem 0 0", color: "var(--otl-muted)", lineHeight: 1.5 }}>
                    {shareFeedback}
                  </p>
                )}
                <TactileButton className="otl-modal-close-btn" onClick={closeShareMenu}>Đóng</TactileButton>
              </div>
            </div>,
            document.body
          )}

          <TactileButton
            onClick={resetToCompose}
            className="qx-btn"
            disabled={isResultExiting}
          >
            ← Soạn thư mới
          </TactileButton>
        </div>
      )}
    </div>
  );
}

function useDelayedPresence(open: boolean, exitMs: number) {
  const [shouldRender, setShouldRender] = useState(open);
  const [isExiting, setIsExiting] = useState(false);

  useEffect(() => {
    if (open) {
      setShouldRender(true);
      setIsExiting(false);
      return;
    }

    if (!shouldRender) {
      return;
    }

    setIsExiting(true);
    const timer = window.setTimeout(() => {
      setShouldRender(false);
      setIsExiting(false);
    }, exitMs);

    return () => window.clearTimeout(timer);
  }, [exitMs, open, shouldRender]);

  return { shouldRender, isExiting };
}
