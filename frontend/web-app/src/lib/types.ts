export type HealthResponse = {
  service: string;
  status: string;
  timestamp: string;
  version: string;
  dependencies?: Record<string, string>;
};

export type CreateSecretRequest = {
  ciphertext: string;
  nonce: string;
  algorithm: string;
  ttlSeconds: number;
};

export type CreateSecretResponse = {
  secretId: string;
  token?: string;
  url?: string;
  expiresAt: string;
};

export type RevealSessionResponse = {
  sessionId: string;
  secretId: string;
  status: string;
  expiresAt: string;
};

export type SecretSceneState =
  | "loading"
  | "sealed"
  | "opening"
  | "revealed"
  | "closing"
  | "burning"
  | "vanished"
  | "expired"
  | "consumed"
  | "error";

export type SecretStatus = {
  secretId: string;
  status: "active" | "consumed" | "expired" | "deleted" | "not_found" | "pending";
  createdAt?: string;
  expiresAt?: string;
  message?: string;
};

export type ConsumeSecretResponse = {
  secretId: string;
  ciphertext: string;
  nonce: string;
  algorithm: string;
  consumedAt: string;
};

export type SecretApiErrorCode =
  | "SECRET_CONSUMED"
  | "SECRET_EXPIRED"
  | "SECRET_NOT_FOUND"
  | "already_consumed"
  | "not_found"
  | "invalid_secret_id"
  | "rate_limit_exceeded"
  | "client_error"
  | "timeout_error"
  | "network_error"
  | "server_error"
  | "unknown_error";

export class SecretApiError extends Error {
  code: SecretApiErrorCode;
  status: number;

  constructor(code: SecretApiErrorCode, message: string, status: number) {
    super(message);
    this.name = "SecretApiError";
    this.code = code;
    this.status = status;
  }
}
