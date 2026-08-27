import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { ResourceNameCell } from "./resource-name-cell";

describe("ResourceNameCell", () => {
  // Mutation captured: removing the inner truncation contract clips long names
  // at the table-cell boundary without rendering an ellipsis.
  it("ellipsizes long names while preserving the complete value on hover", () => {
    const name = "PostgreSQL Agility V2 — identificação longa";
    render(<ResourceNameCell label="Vinculado" linked name={name} />);

    expect(screen.getByText(name)).toHaveClass("min-w-0", "flex-1", "truncate");
    expect(screen.getByText(name)).toHaveAttribute("title", name);
  });
});
