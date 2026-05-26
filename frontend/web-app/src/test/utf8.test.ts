// @vitest-environment node
import { describe, expect, it } from "vitest";
import { clampUtf8Text, getUtf8ByteLength } from "../lib/utf8";

describe("utf8 helpers", () => {
  it("measures UTF-8 bytes accurately", () => {
    expect(getUtf8ByteLength("a😀é")).toBe(7);
  });

  it("clamps on UTF-8 byte boundaries", () => {
    const result = clampUtf8Text("a😀b", 5);

    expect(result).toEqual({
      text: "a😀",
      byteLength: 5,
      truncated: true,
    });
  });

  it("returns empty text when max bytes is zero", () => {
    const result = clampUtf8Text("abc", 0);

    expect(result).toEqual({
      text: "",
      byteLength: 0,
      truncated: true,
    });
  });
});
