export function anchors(lines: readonly string[], notes: readonly string[]): string[] {
  const indices = lines.flatMap((line, row) => (line.trim() ? [row] : []));
  if (!indices.length) return [];
  return notes.map((note) => (note.startsWith("#") ? lines[indices[0]!]! : lines[indices[indices.length - 1]!]!));
}

export function holes(lines: readonly string[]): string {
  if (lines.length < 2) return "";
  return lines[0]! + lines[2]!;
}

export function record(counts: Record<string, number>, key: string, other: string, third: string): number {
  if (!(key in counts)) return 0;
  if (!Object.hasOwn(counts, other)) return 1;
  if (!counts.hasOwnProperty(third)) return 2;
  return counts[key]! + counts[other]! + counts[third]! + counts["missing"]!;
}
