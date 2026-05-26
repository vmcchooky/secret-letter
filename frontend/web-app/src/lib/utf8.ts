function utf8ByteLengthForCodePoint(codePoint: number): number {
  if (codePoint <= 0x7f) {
    return 1;
  }

  if (codePoint <= 0x7ff) {
    return 2;
  }

  if (codePoint <= 0xffff) {
    return 3;
  }

  return 4;
}

export type Utf8ClampResult = {
  text: string;
  byteLength: number;
  truncated: boolean;
};

export function getUtf8ByteLength(text: string): number {
  let bytes = 0;

  for (const char of text) {
    bytes += utf8ByteLengthForCodePoint(char.codePointAt(0)!);
  }

  return bytes;
}

export function clampUtf8Text(text: string, maxBytes: number): Utf8ClampResult {
  if (maxBytes <= 0) {
    return {
      text: "",
      byteLength: 0,
      truncated: text.length > 0,
    };
  }

  let bytes = 0;
  let clamped = "";

  for (const char of text) {
    const charBytes = utf8ByteLengthForCodePoint(char.codePointAt(0)!);
    if (bytes + charBytes > maxBytes) {
      return {
        text: clamped,
        byteLength: bytes,
        truncated: true,
      };
    }

    bytes += charBytes;
    clamped += char;
  }

  return {
    text: clamped,
    byteLength: bytes,
    truncated: false,
  };
}
