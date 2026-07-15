/** Initials for an avatar fallback, limited to one or two characters. */
export function initials(name: string | undefined | null): string {
  const parts = (name ?? "").trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) return "?";
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
}

/** Stable avatar color derived from the displayed identity. */
export function avatarHue(seed: string | undefined | null): string {
  const value = seed ?? "";
  let hash = 0;
  for (let index = 0; index < value.length; index += 1) {
    hash = (hash * 31 + value.charCodeAt(index)) >>> 0;
  }
  return `hsl(${hash % 360} 45% 42%)`;
}
