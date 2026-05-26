/**
 * Web Crypto API helpers for AES-GCM encryption/decryption
 * Following specs from docs/contracts/crypto-and-api-decisions.md
 */

const ALGORITHM = "AES-GCM";
const KEY_LENGTH = 256;
const NONCE_LENGTH = 12; // 12 bytes = 96 bits
const KEY_LENGTH_BYTES = KEY_LENGTH / 8;
const BASE64URL_PATTERN = /^[A-Za-z0-9_-]+$/;

export class Base64UrlDecodeError extends Error {
  constructor(message = "Invalid base64url data") {
    super(message);
    this.name = "Base64UrlDecodeError";
  }
}

export class InvalidKeyMaterialError extends Error {
  constructor(message = "Invalid AES-GCM key material") {
    super(message);
    this.name = "InvalidKeyMaterialError";
  }
}

function asBufferSource(bytes: Uint8Array): ArrayBuffer {
  const copy = new ArrayBuffer(bytes.byteLength);
  new Uint8Array(copy).set(bytes);
  return copy;
}

/**
 * Generate a 256-bit AES-GCM key
 */
export async function generateKey(): Promise<CryptoKey> {
  return await crypto.subtle.generateKey(
    {
      name: ALGORITHM,
      length: KEY_LENGTH,
    },
    true, // extractable
    ["encrypt", "decrypt"]
  );
}

/**
 * Generate a 12-byte (96-bit) nonce for AES-GCM
 */
export function generateNonce(): Uint8Array {
  return crypto.getRandomValues(new Uint8Array(NONCE_LENGTH));
}

/**
 * Encrypt plaintext using AES-GCM
 * @param plaintext - The secret text to encrypt
 * @param key - The AES-GCM key
 * @param nonce - The 12-byte nonce
 * @returns The ciphertext as Uint8Array
 */
export async function encryptSecret(
  plaintext: string,
  key: CryptoKey,
  nonce: Uint8Array
): Promise<Uint8Array> {
  const encoder = new TextEncoder();
  const plaintextBytes = encoder.encode(plaintext);

  const ciphertextBuffer = await crypto.subtle.encrypt(
    {
      name: ALGORITHM,
      iv: asBufferSource(nonce),
    },
    key,
    asBufferSource(plaintextBytes)
  );

  return new Uint8Array(ciphertextBuffer);
}

/**
 * Decrypt ciphertext using AES-GCM
 * @param ciphertext - The encrypted data
 * @param key - The AES-GCM key
 * @param nonce - The 12-byte nonce
 * @returns The decrypted plaintext string
 */
export async function decryptSecret(
  ciphertext: Uint8Array,
  key: CryptoKey,
  nonce: Uint8Array
): Promise<string> {
  const plaintextBuffer = await crypto.subtle.decrypt(
    {
      name: ALGORITHM,
      iv: asBufferSource(nonce),
    },
    key,
    asBufferSource(ciphertext)
  );

  const decoder = new TextDecoder();
  return decoder.decode(plaintextBuffer);
}

/**
 * Encode bytes to base64url format (RFC 4648)
 * Safe for URLs and fragments
 */
export function encodeBase64Url(bytes: Uint8Array): string {
  // Convert to base64
  let base64 = btoa(String.fromCharCode(...bytes));
  
  // Convert to base64url: replace +/= with -_
  return base64
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=/g, "");
}

/**
 * Decode base64url format to bytes
 */
export function decodeBase64Url(base64url: string): Uint8Array {
  const input = base64url.trim();
  if (!input || !BASE64URL_PATTERN.test(input) || input.length % 4 === 1) {
    throw new Base64UrlDecodeError();
  }

  // Convert base64url to base64: replace -_ with +/
  let base64 = input
    .replace(/-/g, "+")
    .replace(/_/g, "/");
  
  // Add padding if needed
  while (base64.length % 4 !== 0) {
    base64 += "=";
  }
  
  // Decode base64 to bytes
  let binaryString: string;
  try {
    binaryString = atob(base64);
  } catch {
    throw new Base64UrlDecodeError();
  }

  const bytes = new Uint8Array(binaryString.length);
  for (let i = 0; i < binaryString.length; i++) {
    bytes[i] = binaryString.charCodeAt(i);
  }
  
  return bytes;
}

/**
 * Export a CryptoKey to base64url format for URL fragments
 */
export async function exportKeyToBase64Url(key: CryptoKey): Promise<string> {
  const rawKey = await crypto.subtle.exportKey("raw", key);
  return encodeBase64Url(new Uint8Array(rawKey));
}

/**
 * Import a key from base64url format
 */
export async function importKeyFromBase64Url(base64url: string): Promise<CryptoKey> {
  const keyBytes = decodeBase64Url(base64url);
  if (keyBytes.byteLength !== KEY_LENGTH_BYTES) {
    throw new InvalidKeyMaterialError();
  }
  
  return await crypto.subtle.importKey(
    "raw",
    asBufferSource(keyBytes),
    {
      name: ALGORITHM,
      length: KEY_LENGTH,
    },
    true,
    ["encrypt", "decrypt"]
  );
}
