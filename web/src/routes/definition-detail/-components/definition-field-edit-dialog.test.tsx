import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { I18nProvider } from "@/lib/i18n";
import { DefinitionFieldEditDialog } from "./definition-field-edit-dialog";

const field = {
  data_type: "string",
  hint: "",
  id: "field-1",
  is_secret: true,
  key: "host",
  label: "Host",
  required: true,
  resource_definition_id: "definition-1",
  sort_order: 0,
};

describe("DefinitionFieldEditDialog", () => {
  // Mutation captured: making contract fields editable would allow the UI to
  // request schema/storage changes that the safe metadata endpoint rejects.
  it("edits label and required while keeping contract fields read-only", () => {
    const onSave = vi.fn();
    render(<I18nProvider><DefinitionFieldEditDialog
      error=""
      field={field}
      form={{ label: "Host", required: true }}
      isSaving={false}
      onFormChange={vi.fn()}
      onOpenChange={vi.fn()}
      onSave={onSave}
    /></I18nProvider>);

    expect(screen.getByLabelText("Key")).toBeDisabled();
    expect(screen.getByLabelText("Tipo")).toBeDisabled();
    const requiredCheckbox = screen.getByRole("checkbox", { name: "Obrigatório" });
    const secretCheckbox = screen.getByRole("checkbox", { name: "Segredo" });
    expect(secretCheckbox).toHaveAttribute("aria-disabled", "true");
    expect(screen.getByLabelText("Label")).toBeEnabled();
    expect(requiredCheckbox).not.toHaveAttribute("aria-disabled", "true");

    for (const checkbox of [requiredCheckbox, secretCheckbox]) {
      const row = checkbox.parentElement;
      expect(row).toHaveClass("flex", "gap-2");
      expect(row).not.toHaveClass("border", "rounded-md", "p-3");
      expect(row?.firstElementChild).toBe(checkbox);
    }

    expect(screen.getByText(
      /Para alterar chave, tipo ou segredo, crie um novo campo e migre os valores\./,
    )).toHaveAttribute("data-slot", "dialog-description");

    fireEvent.click(screen.getByRole("button", { name: "Salvar" }));
    expect(onSave).toHaveBeenCalledOnce();
  });
});
