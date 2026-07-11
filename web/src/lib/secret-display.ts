export const SECRET_MASK = "••••••••••••";

export function displaySecretValue({
  isSecret,
  revealed,
  value,
}: {
  isSecret: boolean;
  revealed: boolean;
  value: string;
}): string {
  if (!isSecret) return value;
  return revealed ? value : SECRET_MASK;
}
