import { describe, expect, it, vi } from "vitest";
import { writeClipboardText } from "@/lib/clipboard";

describe("writeClipboardText", () => {
  it("writes through the browser clipboard API", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", { configurable: true, value: { writeText } });

    await writeClipboardText("tnc_test");

    expect(writeText).toHaveBeenCalledWith("tnc_test");
  });

  it("rejects when the clipboard API is unavailable", async () => {
    Object.defineProperty(navigator, "clipboard", { configurable: true, value: undefined });
    await expect(writeClipboardText("tnc_test")).rejects.toThrow("clipboard API unavailable");
  });
});
