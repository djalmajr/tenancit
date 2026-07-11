export async function writeClipboardText(value: string): Promise<void> {
  if (!navigator.clipboard) throw new Error("clipboard API unavailable");
  await navigator.clipboard.writeText(value);
}
