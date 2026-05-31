import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { RevealValue } from "./reveal-value";

describe("RevealValue", () => {
  // Mutation captured: removing the `shown ? value : "••••"` masking (always
  // showing value) makes the secret visible before clicking → this fails.
  it("masks the secret by default and does not render the cleartext", () => {
    render(<RevealValue value="s3cr3t" />);
    expect(screen.queryByText("s3cr3t")).not.toBeInTheDocument();
    expect(screen.getByText("••••••••••••")).toBeInTheDocument();
  });

  // Mutation captured: dropping the onClick toggle (setShown) leaves it masked
  // forever → cleartext never appears.
  it("reveals the cleartext after clicking the toggle", async () => {
    const user = userEvent.setup();
    render(<RevealValue value="s3cr3t" />);
    await user.click(screen.getByRole("button", { name: "Revelar" }));
    expect(screen.getByText("s3cr3t")).toBeInTheDocument();
  });
});
