import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { DomainStatus } from "./domain-status";

describe("DomainStatus", () => {
  it("renders status labels without a bullet indicator", () => {
    const { container } = render(<DomainStatus label="ativo" value="active" />);

    expect(screen.getByText("ativo")).toBeInTheDocument();
    expect(container.querySelector('[data-slot="status-indicator"]')).toBeNull();
  });
});
