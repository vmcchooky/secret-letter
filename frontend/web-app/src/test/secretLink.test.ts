import { describe, expect, it } from "vitest";
import { splitSecretLink } from "../lib/secretLink";

describe("splitSecretLink", () => {
  it("separates the public URL from the fragment key", () => {
    expect(
      splitSecretLink("https://secret.example/s/demo-token#demo-fragment-key")
    ).toEqual({
      fullLink: "https://secret.example/s/demo-token#demo-fragment-key",
      publicUrl: "https://secret.example/s/demo-token",
      fragmentKey: "demo-fragment-key",
    });
  });

  it("preserves links without a fragment key", () => {
    expect(splitSecretLink("https://secret.example/s/demo-token")).toEqual({
      fullLink: "https://secret.example/s/demo-token",
      publicUrl: "https://secret.example/s/demo-token",
      fragmentKey: "",
    });
  });

  it("normalizes surrounding whitespace", () => {
    expect(
      splitSecretLink("  https://secret.example/s/demo-token#demo-fragment-key  ")
    ).toEqual({
      fullLink: "https://secret.example/s/demo-token#demo-fragment-key",
      publicUrl: "https://secret.example/s/demo-token",
      fragmentKey: "demo-fragment-key",
    });
  });
});
