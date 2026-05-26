import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { CreateSecretForm } from "../components/CreateSecretForm";
import { clampUtf8Text, getUtf8ByteLength } from "../lib/utf8";

describe("CreateSecretForm", () => {
  it("clamps pasted content to the byte limit without corrupting emoji", () => {
    render(<CreateSecretForm />);

    const textarea = screen.getByLabelText("Nội dung bí mật");
    const payload = "😀".repeat(3000);
    const expected = clampUtf8Text(payload, 10 * 1024);

    fireEvent.change(textarea, { target: { value: payload } });

    expect((textarea as HTMLTextAreaElement).value).toBe(expected.text);
    expect(getUtf8ByteLength((textarea as HTMLTextAreaElement).value)).toBe(10 * 1024);
    expect((textarea as HTMLTextAreaElement).value.length).toBe(expected.text.length);

    const formattedLimit = (10 * 1024).toLocaleString();
    expect(
      screen.getByText(`${formattedLimit} / ${formattedLimit} bytes`)
    ).toBeInTheDocument();
  });
});
