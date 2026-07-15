import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "./select";

describe("Select", () => {
  it("renders the selected item label instead of its raw value", () => {
    render(
      <Select items={{ active: "Ativo", inactive: "Inativo" }} value="active">
        <SelectTrigger>
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          <SelectGroup>
            <SelectItem value="active">Ativo</SelectItem>
            <SelectItem value="inactive">Inativo</SelectItem>
          </SelectGroup>
        </SelectContent>
      </Select>,
    );

    expect(screen.getByText("Ativo")).toBeInTheDocument();
    expect(screen.queryByText("active")).not.toBeInTheDocument();
  });
});
