import { useCallback, useEffect, useMemo, useState } from "react";
import { useParams, useLocation } from "react-router-dom";
import { SecretEnvelopeScene } from "../components/secret-letter/SecretEnvelopeScene";
import { createRevealSession, getSecretStatus, openSecret } from "../lib/api";
import {
  Base64UrlDecodeError,
  decryptSecret,
  decodeBase64Url,
  importKeyFromBase64Url,
  InvalidKeyMaterialError,
} from "../lib/crypto";
import { SecretApiError, type SecretSceneState } from "../lib/types";

type SceneData = {
  state: SecretSceneState;
  secretContent: string;
  expiresAt?: string;
  message?: string;
};

const initialScene: SceneData = {
  state: "loading",
  secretContent: "",
};

const invalidKeyFormatMessage =
  "Khóa giải mã trong liên kết không đúng định dạng. Hãy kiểm tra lại phần sau dấu #.";

export function RevealPage() {
  const { secretId } = useParams<{ secretId: string }>();
  const location = useLocation();
  const [scene, setScene] = useState<SceneData>(initialScene);

  const fragmentKey = useMemo(() => location.hash.slice(1), [location.hash]);

  const loadStatus = useCallback(async () => {
    setScene(initialScene);

    if (!secretId) {
      setScene({
        state: "error",
        secretContent: "",
        message: "Liên kết không hợp lệ. Thiếu mã lá thư.",
      });
      return;
    }

    if (!fragmentKey) {
      setScene({
        state: "error",
        secretContent: "",
        message: "Liên kết không hợp lệ. Thiếu khóa giải mã.",
      });
      return;
    }

    if (!isValidFragmentKeyFormat(fragmentKey)) {
      setScene({
        state: "error",
        secretContent: "",
        message: invalidKeyFormatMessage,
      });
      return;
    }

    try {
      const status = await getSecretStatus(secretId);

      if (status.status === "active" || status.status === "pending") {
        setScene({
          state: "sealed",
          secretContent: "",
          expiresAt: status.expiresAt,
        });
        return;
      }

      if (status.status === "consumed") {
        setScene({
          state: "consumed",
          secretContent: "",
          expiresAt: status.expiresAt,
        });
        return;
      }

      if (status.status === "expired") {
        setScene({
          state: "expired",
          secretContent: "",
          expiresAt: status.expiresAt,
        });
        return;
      }

      setScene({
        state: "error",
        secretContent: "",
        message: "Không tìm thấy lá thư bí mật nào ở đây.",
      });
    } catch (error) {
      setScene({
        state: "error",
        secretContent: "",
        message:
          error instanceof Error
            ? error.message
            : "Không thể mở lá thư bí mật này. Hãy thử lại.",
      });
    }
  }, [fragmentKey, secretId]);

  useEffect(() => {
    void loadStatus();
  }, [loadStatus]);

  const handleOpen = useCallback(async () => {
    if (!secretId || !fragmentKey || scene.state !== "sealed") {
      return;
    }

    let key: CryptoKey;
    try {
      key = await importKeyFromBase64Url(fragmentKey);
    } catch (error) {
      const { state, message } = mapFragmentKeyError(error);
      setScene({
        state,
        secretContent: "",
        message,
      });
      return;
    }

    try {
      const revealSession = await createRevealSession(secretId);
      const response = await openSecret(secretId, revealSession.sessionId);
      if (!response.ciphertext || !response.nonce) {
        throw new Error("Secret payload is missing.");
      }

      const ciphertext = decodeBase64Url(response.ciphertext);
      const nonce = decodeBase64Url(response.nonce);
      const plaintext = await decryptSecret(ciphertext, key, nonce);

      setScene((current) => ({
        ...current,
        state: "opening",
        secretContent: plaintext,
      }));
    } catch (error) {
      const { state, message } = mapOpenError(error);
      setScene({
        state,
        secretContent: "",
        message,
      });
    }
  }, [fragmentKey, scene.state, secretId]);

  const handleOpenComplete = useCallback(() => {
    setScene((current) =>
      current.state === "opening" ? { ...current, state: "revealed" } : current,
    );
  }, []);

  const handleClose = useCallback(() => {
    setScene((current) =>
      current.state === "revealed" ? { ...current, state: "closing" } : current,
    );
  }, []);

  const handleBurning = useCallback(() => {
    setScene((current) =>
      current.state === "closing" ? { ...current, state: "burning" } : current,
    );
  }, []);

  const handleBurnComplete = useCallback(() => {
    setScene({
      state: "vanished",
      secretContent: "",
    });
  }, []);

  return (
    <SecretEnvelopeScene
      state={scene.state}
      secretContent={scene.secretContent}
      expiresAt={scene.expiresAt}
      message={scene.message}
      onOpen={handleOpen}
      onOpenComplete={handleOpenComplete}
      onClose={handleClose}
      onBurning={handleBurning}
      onBurnComplete={handleBurnComplete}
      onRetry={loadStatus}
    />
  );
}

function isValidFragmentKeyFormat(fragmentKey: string): boolean {
  try {
    const keyBytes = decodeBase64Url(fragmentKey);
    return keyBytes.byteLength === 32;
  } catch {
    return false;
  }
}

function mapOpenError(error: unknown): Pick<SceneData, "state" | "message"> {
  if (error instanceof SecretApiError) {
    if (error.code === "SECRET_CONSUMED" || error.code === "already_consumed") {
      return { state: "consumed" };
    }

    if (error.code === "SECRET_EXPIRED") {
      return { state: "expired" };
    }

    return {
      state: "error",
      message: error.message,
    };
  }

  if (error instanceof DOMException) {
    return {
      state: "error",
      message: "Khóa giải mã không khớp hoặc dữ liệu đã bị hỏng.",
    };
  }

  if (error instanceof Base64UrlDecodeError) {
    return {
      state: "error",
      message: "Dữ liệu mã hóa trả về không đúng định dạng.",
    };
  }

  return {
    state: "error",
    message: "Không thể mở lá thư bí mật này. Hãy thử lại.",
  };
}

function mapFragmentKeyError(error: unknown): Pick<SceneData, "state" | "message"> {
  if (error instanceof Base64UrlDecodeError || error instanceof InvalidKeyMaterialError) {
    return {
      state: "error",
      message: invalidKeyFormatMessage,
    };
  }

  return {
    state: "error",
    message: "Không thể đọc khóa giải mã trong liên kết.",
  };
}
