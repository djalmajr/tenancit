import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ResourceOriginField } from "./resource-origin-field";

const commonProps = {
  independentHint: "Os valores pertencem apenas a este recurso.",
  independentLabel: "Independente",
  label: "Origem dos valores",
  linkedHint: "Campos vazios herdam a origem.",
  onValueChange: vi.fn(),
  value: "",
};

describe("ResourceOriginField", () => {
  // Mutation captured: rendering an origin selector without candidates creates
  // a control with one option that cannot change the resource configuration.
  it("does not render when independent is the only possible origin", () => {
    render(<ResourceOriginField {...commonProps} candidates={[]} />);

    expect(screen.queryByText("Origem dos valores")).not.toBeInTheDocument();
  });

  it("renders in one grid column when a compatible origin exists", () => {
    const { container } = render(<ResourceOriginField
      {...commonProps}
      candidates={[{ id: "source-1", label: "Vincular a minio.base" }]}
    />);

    const field = container.querySelector('[data-slot="resource-origin-field"]');
    expect(field).toBeInTheDocument();
    expect(field).not.toHaveClass("md:col-span-2");
    expect(screen.getByText("Origem dos valores")).toBeInTheDocument();
  });
});
