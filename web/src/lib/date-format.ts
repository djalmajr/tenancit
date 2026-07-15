import type { Locale } from "./i18n";

export function formatDate(locale: Locale, value: Date | number | string, options: Intl.DateTimeFormatOptions = {}): string {
  const date = value instanceof Date ? value : new Date(value);
  if (Number.isNaN(date.getTime())) return "—";
  return new Intl.DateTimeFormat(locale, options).format(date);
}
