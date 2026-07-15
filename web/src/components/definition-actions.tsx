import { EllipsisVertical, Pencil, Power, PowerOff, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import type { Definition } from "@/lib/api";
import { useI18n } from "@/lib/i18n";

export function DefinitionActions({
  definition,
  onEdit,
  onRemove,
  onToggleStatus,
}: {
  definition: Definition;
  onEdit: () => void;
  onRemove: () => void;
  onToggleStatus: () => void;
}) {
  const { t } = useI18n();

  return (
    <DropdownMenu>
      <DropdownMenuTrigger
        aria-label={t("common.actions")}
        onClick={(event) => event.stopPropagation()}
        render={<Button size="icon-sm" title={t("common.actions")} variant="ghost" />}
      >
        <EllipsisVertical />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" onClick={(event) => event.stopPropagation()}>
        <DropdownMenuItem onClick={onEdit}>
          <Pencil /> {t("common.edit")}
        </DropdownMenuItem>
        <DropdownMenuItem onClick={onToggleStatus}>
          {definition.status === "active" ? <PowerOff /> : <Power />}
          {definition.status === "active" ? t("definitionDetail.deactivate") : t("definitionDetail.activate")}
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuItem onClick={onRemove} variant="destructive">
          <Trash2 /> {t("common.remove")}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
