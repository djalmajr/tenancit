import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { ResourceOriginIcon } from "./resource-origin-icon";

describe("ResourceOriginIcon", () => {
  // Mutation captured: removing shrink-0 lets long resource names collapse the
  // linked icon until it is barely visible in the tenant resources table.
  it("keeps linked and independent icons at the same readable visual scale", () => {
    const { rerender } = render(<ResourceOriginIcon label="Vinculado" linked />);

    expect(screen.getByLabelText("Vinculado")).toHaveClass(
      "size-5",
      "shrink-0",
      "stroke-[2.25]",
    );

    rerender(<ResourceOriginIcon label="Independente" linked={false} />);

    expect(screen.getByLabelText("Independente")).toHaveClass(
      "size-5",
      "shrink-0",
      "stroke-[2.25]",
    );
  });
});
