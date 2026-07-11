import { addMonths, format, isSameMonth } from "date-fns";
import { enUS, es, ptBR, type Locale as DateFnsLocale } from "date-fns/locale";
import { ChevronLeft, ChevronRight } from "lucide-react";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { type Locale, useI18n } from "@/lib/i18n";

const YEARS_PER_PAGE = 12;
const pageStartOf = (year: number) => Math.floor(year / YEARS_PER_PAGE) * YEARS_PER_PAGE;
const capitalize = (value: string) => value.charAt(0).toUpperCase() + value.slice(1);
function dateFnsLocale(locale: Locale): DateFnsLocale { return locale === "es-ES" ? es : locale === "en-US" ? enUS : ptBR; }

function MonthYearPicker({ selected, onSelect }: { selected: Date; onSelect: (date: Date) => void }) {
  const { t, locale } = useI18n();
  const [year, setYear] = useState(selected.getFullYear());
  const [view, setView] = useState<"months" | "years">("months");
  const values = view === "years"
    ? Array.from({ length: YEARS_PER_PAGE }, (_, index) => pageStartOf(year) + index)
    : Array.from({ length: 12 }, (_, index) => index);
  return <div className="flex w-65 flex-col gap-3 p-3">
    <div className="flex items-center justify-between gap-1">
      <Button aria-label={t("usage.prevYear")} size="icon-sm" type="button" variant="ghost" onClick={() => setYear((value) => value - (view === "years" ? YEARS_PER_PAGE : 1))}><ChevronLeft /></Button>
      <Button className="flex-1 text-sm font-semibold" type="button" variant="ghost" onClick={() => setView((value) => value === "months" ? "years" : "months")}>{view === "years" ? `${pageStartOf(year)}–${pageStartOf(year) + YEARS_PER_PAGE - 1}` : year}</Button>
      <Button aria-label={t("usage.nextYear")} size="icon-sm" type="button" variant="ghost" onClick={() => setYear((value) => value + (view === "years" ? YEARS_PER_PAGE : 1))}><ChevronRight /></Button>
    </div>
    <div className="grid grid-cols-3 gap-1">{values.map((value) => {
      const isYear = view === "years";
      const active = isYear ? value === selected.getFullYear() : value === selected.getMonth() && year === selected.getFullYear();
      return <Button key={value} size="sm" type="button" variant={active ? "secondary" : "ghost"} onClick={() => isYear ? (setYear(value), setView("months")) : onSelect(new Date(year, value, 1))}>{isYear ? value : capitalize(format(new Date(year, value, 1), "LLL", { locale: dateFnsLocale(locale) }))}</Button>;
    })}</div>
  </div>;
}

export function MonthYearNav({ value, onChange }: { value: Date; onChange: (date: Date) => void }) {
  const { t, locale } = useI18n();
  const [open, setOpen] = useState(false);
  const today = new Date();
  return <div className="flex items-center gap-2">
    <div className="inline-flex h-9 items-center rounded-lg border border-input">
      <Button aria-label={t("usage.prevMonth")} className="size-9 shrink-0 rounded-r-none border-0" size="icon" type="button" variant="ghost" onClick={() => onChange(addMonths(value, -1))}><ChevronLeft /></Button>
      <Popover open={open} onOpenChange={setOpen}>
        <PopoverTrigger render={<Button aria-label={t("usage.month")} className="h-9 min-w-32 rounded-none border-0 border-x border-input px-3 text-sm font-medium capitalize" type="button" variant="ghost" />}>{capitalize(format(value, "MMM yyyy", { locale: dateFnsLocale(locale) }))}</PopoverTrigger>
        <PopoverContent align="start" className="w-auto p-0"><MonthYearPicker key={value.getTime()} selected={value} onSelect={(date) => { onChange(date); setOpen(false); }} /></PopoverContent>
      </Popover>
      <Button aria-label={t("usage.nextMonth")} className="size-9 shrink-0 rounded-l-none border-0" size="icon" type="button" variant="ghost" onClick={() => onChange(addMonths(value, 1))}><ChevronRight /></Button>
    </div>
    <Button disabled={isSameMonth(value, today)} type="button" variant="outline" onClick={() => onChange(new Date(today.getFullYear(), today.getMonth(), 1))}>{t("usage.today")}</Button>
  </div>;
}
