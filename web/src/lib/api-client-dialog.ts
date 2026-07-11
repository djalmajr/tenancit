export function canChangeAPIClientDialogOpen(nextOpen: boolean, isCreating: boolean): boolean {
  return nextOpen || !isCreating;
}

export function isDuplicateAPIClientName(name: string, existingNames: string[]): boolean {
  const normalizedName = name.trim().toLowerCase();
  if (!normalizedName) return false;
  return existingNames.some((candidate) => candidate.trim().toLowerCase() === normalizedName);
}
