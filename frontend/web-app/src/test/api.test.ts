// @vitest-environment node
import { describe, expect, it, vi } from "vitest";
import { REQUEST_TIMEOUT_MS, createRevealSession, createSecret, fetchHealth, getSecretStatus, openSecret } from "../lib/api";
import { SecretApiError } from "../lib/types";

function makeJsonResponse(body: unknown, status: number) {
  return new Response(JSON.stringify(body), {
    status,
    headers: {
      "Content-Type": "application/json",
    },
  });
}

describe("secret api", () => {
  it("surfaces offline failures as network_error", async () => {
    vi.spyOn(globalThis, "fetch").mockRejectedValueOnce(new TypeError("Failed to fetch"));

    await expect(fetchHealth()).rejects.toMatchObject({
      code: "network_error",
      status: 0,
    });
  });

  it("surfaces client timeouts as timeout_error", async () => {
    vi.useFakeTimers();
    vi.spyOn(globalThis, "fetch").mockImplementationOnce((_input, init) => {
      return new Promise<Response>((_resolve, reject) => {
        const signal = init?.signal;
        if (signal?.aborted) {
          reject(new DOMException("Aborted", "AbortError"));
          return;
        }

        signal?.addEventListener(
          "abort",
          () => reject(new DOMException("Aborted", "AbortError")),
          { once: true }
        );
      });
    });

    const request = fetchHealth().catch((error) => error);
    await vi.advanceTimersByTimeAsync(REQUEST_TIMEOUT_MS + 1);

    await expect(request).resolves.toMatchObject({
      code: "timeout_error",
      status: 0,
    });
  });

  it("maps consumed secrets to SECRET_CONSUMED", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      makeJsonResponse({ error: "SECRET_CONSUMED" }, 410)
    );

    await expect(getSecretStatus("demo")).rejects.toMatchObject({
      code: "SECRET_CONSUMED",
      status: 410,
    });
  });

  it("maps generic 4xx responses to client_error", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      makeJsonResponse({ message: "Bad request" }, 422)
    );

    await expect(openSecret("demo")).rejects.toMatchObject({
      code: "client_error",
      status: 422,
    });
  });

  it("sends reveal session headers when opening a secret", async () => {
    const fetchSpy = vi.spyOn(globalThis, "fetch").mockImplementationOnce((_input, init) => {
      const headers = new Headers(init?.headers);
      expect(headers.get("X-Reveal-Session")).toBe("session-123");
      return Promise.resolve(
        makeJsonResponse({
          secretId: "demo",
          ciphertext: "dGVzdA",
          nonce: "MTIzNDU2Nzg5MDEy",
          algorithm: "AES-GCM",
          consumedAt: "2026-04-16T12:30:00Z",
        }, 200),
      );
    });

    await expect(openSecret("demo", "session-123")).resolves.toMatchObject({
      secretId: "demo",
    });
    expect(fetchSpy).toHaveBeenCalledTimes(1);
  });

  it("creates reveal sessions", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      makeJsonResponse({
        sessionId: "session-123",
        secretId: "demo",
        status: "active",
        expiresAt: "2026-04-16T12:35:00Z",
      }, 201),
    );

    await expect(createRevealSession("demo")).resolves.toMatchObject({
      sessionId: "session-123",
      secretId: "demo",
    });
  });

  it("maps 5xx responses to server_error", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      makeJsonResponse({ message: "Server exploded" }, 500)
    );

    await expect(createSecret({
      ciphertext: "x",
      nonce: "y",
      algorithm: "AES-GCM",
      ttlSeconds: 60,
    })).rejects.toMatchObject({
      code: "server_error",
      status: 500,
    });
  });

  it("throws SecretApiError instances for transport failures", async () => {
    vi.spyOn(globalThis, "fetch").mockRejectedValueOnce(new Error("boom"));

    await expect(fetchHealth()).rejects.toBeInstanceOf(SecretApiError);
  });
});
