import type { 
  HealthResponse, 
  CreateSecretRequest, 
  CreateSecretResponse,
  RevealSessionResponse,
  SecretStatus,
  ConsumeSecretResponse,
  SecretApiErrorCode
} from "./types";
import { SecretApiError } from "./types";

const defaultApiBaseUrl = "http://localhost:8080";
export const REQUEST_TIMEOUT_MS = 10_000;

export const apiBaseUrl =
  import.meta.env.VITE_API_BASE_URL?.trim() || defaultApiBaseUrl;

/**
 * Generate a UUID v4 for request tracking
 */
function generateRequestId(): string {
  return crypto.randomUUID();
}

export async function fetchHealth(): Promise<HealthResponse> {
  const response = await safeFetch(`${apiBaseUrl}/healthz`, {
    headers: {
      Accept: "application/json",
      "X-Request-ID": generateRequestId(),
    },
  });

  if (!response.ok) {
    throw new Error(`Health check failed with status ${response.status}`);
  }

  return (await response.json()) as HealthResponse;
}

export async function createSecret(
  request: CreateSecretRequest
): Promise<CreateSecretResponse> {
  const requestId = generateRequestId();
  
  const response = await safeFetch(`${apiBaseUrl}/api/secrets`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Accept: "application/json",
      "X-Request-ID": requestId,
    },
    body: JSON.stringify(request),
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw toSecretApiError(response, errorData);
  }

  return (await response.json()) as CreateSecretResponse;
}

export async function getSecretStatus(
  secretId: string
): Promise<SecretStatus> {
  const requestId = generateRequestId();
  
  const response = await safeFetch(`${apiBaseUrl}/api/secrets/${secretId}/status`, {
    method: "GET",
    cache: "no-store",
    headers: {
      Accept: "application/json",
      "X-Request-ID": requestId,
    },
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw toSecretApiError(response, errorData);
  }

  return (await response.json()) as SecretStatus;
}

export async function createRevealSession(
  secretId: string
): Promise<RevealSessionResponse> {
  const requestId = generateRequestId();

  const response = await safeFetch(`${apiBaseUrl}/api/reveal-sessions`, {
    method: "POST",
    cache: "no-store",
    headers: {
      "Content-Type": "application/json",
      Accept: "application/json",
      "X-Request-ID": requestId,
    },
    body: JSON.stringify({ secretId }),
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw toSecretApiError(response, errorData);
  }

  return (await response.json()) as RevealSessionResponse;
}

export async function openSecret(
  secretId: string,
  revealSessionId?: string
): Promise<ConsumeSecretResponse> {
  return openSecretAtPath(secretId, "open", revealSessionId);
}

export async function consumeSecret(
  secretId: string
): Promise<ConsumeSecretResponse> {
  return openSecretAtPath(secretId, "consume");
}

async function openSecretAtPath(
  secretId: string,
  action: "open" | "consume",
  revealSessionId?: string
): Promise<ConsumeSecretResponse> {
  const requestId = generateRequestId();
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    Accept: "application/json",
    "X-Request-ID": requestId,
  };

  if (revealSessionId) {
    headers["X-Reveal-Session"] = revealSessionId;
  }

  const response = await safeFetch(`${apiBaseUrl}/api/secrets/${secretId}/${action}`, {
    method: "POST",
    cache: "no-store",
    headers,
    body: JSON.stringify({}),
  });

  if (!response.ok) {
    const errorData = await response.json().catch(() => ({}));
    throw toSecretApiError(response, errorData);
  }

  return (await response.json()) as ConsumeSecretResponse;
}

async function safeFetch(
  input: RequestInfo | URL,
  init?: RequestInit
): Promise<Response> {
  const controller = new AbortController();
  let timedOut = false;
  const timeoutId = globalThis.setTimeout(() => {
    timedOut = true;
    controller.abort();
  }, REQUEST_TIMEOUT_MS);

  try {
    return await fetch(input, {
      ...init,
      signal: controller.signal,
    });
  } catch (error) {
    if (timedOut) {
      throw new SecretApiError(
        "timeout_error",
        "Máy chủ phản hồi quá lâu. Hãy thử lại.",
        0
      );
    }

    throw new SecretApiError(
      "network_error",
      "Không kết nối được máy chủ. Hãy kiểm tra mạng hoặc thử lại sau.",
      0
    );
  } finally {
    globalThis.clearTimeout(timeoutId);
  }
}

function toSecretApiError(
  response: Response,
  errorData: { error?: string; message?: string }
): SecretApiError {
  const code = normalizeSecretErrorCode(errorData.error, response.status);
  const message =
    friendlySecretApiMessage(code, response.status) ||
    errorData.message ||
    "Không thể mở lá thư bí mật này.";

  return new SecretApiError(code, message, response.status);
}

function normalizeSecretErrorCode(
  code: string | undefined,
  status: number
): SecretApiErrorCode {
  if (
    code === "SECRET_CONSUMED" ||
    code === "SECRET_EXPIRED" ||
    code === "SECRET_NOT_FOUND" ||
    code === "already_consumed" ||
    code === "not_found" ||
    code === "invalid_secret_id" ||
    code === "rate_limit_exceeded"
  ) {
    return code;
  }

  if (status === 408 || status === 504) {
    return "timeout_error";
  }

  if (status === 410) {
    return "SECRET_CONSUMED";
  }

  if (status === 404) {
    return "SECRET_NOT_FOUND";
  }

  if (status === 429) {
    return "rate_limit_exceeded";
  }

  if (status >= 400 && status < 500) {
    return "client_error";
  }

  if (status >= 500) {
    return "server_error";
  }

  return "unknown_error";
}

function friendlySecretApiMessage(
  code: SecretApiErrorCode,
  status: number
): string {
  switch (code) {
    case "SECRET_CONSUMED":
    case "already_consumed":
      return "Liên kết này đã được mở một lần trước đó.";
    case "SECRET_EXPIRED":
      return "Lá thư này đã hết hạn.";
    case "SECRET_NOT_FOUND":
    case "not_found":
      return "Không tìm thấy lá thư bí mật nào ở đây.";
    case "invalid_secret_id":
      return "Liên kết không hợp lệ hoặc đã bị thiếu ký tự.";
    case "rate_limit_exceeded":
      return "Bạn thao tác hơi nhanh. Hãy thử lại sau ít phút.";
    case "client_error":
      return "Yêu cầu không hợp lệ. Hãy kiểm tra lại liên kết và thử lại.";
    case "timeout_error":
      return "Máy chủ phản hồi quá lâu. Hãy thử lại.";
    case "network_error":
      return "Không kết nối được máy chủ. Hãy kiểm tra mạng hoặc thử lại sau.";
    case "server_error":
      return "Máy chủ đang gặp sự cố. Bí mật chưa được giải mã, hãy thử lại sau.";
    default:
      return status === 429
        ? "Bạn thao tác hơi nhanh. Hãy thử lại sau ít phút."
        : "";
  }
}
