export function total(counts: Map<string, number>, key: string): number {
  if (!counts.has(key)) return 0;
  return counts.get(key)! + 1;
}

export function edges(rows: readonly number[]): [number, number] {
  if (rows.length === 0) return [0, 0];
  return [rows.at(0)!, rows.at(-1)!];
}

export function every(rows: string[], width: number): string[] {
  if (!rows.length) return [];
  const out: string[] = [];
  for (let i = 0; i < width; i++) out.push(rows.at(-1)!);
  return out;
}

export function held(rows: string[], counts: Map<string, number>, key: string): string {
  const count = rows.length;
  const known = counts.has(key);
  if (count < 2 || !known) return "";
  return rows.at(1)! + counts.get(key)!;
}

declare function ensure(condition: unknown, message?: string): asserts condition;

export function ensured(rows: string[]): string {
  ensure(rows.length > 2, "three rows");
  return rows.at(2)!;
}

function requireRows(rows: readonly string[]): void {
  if (rows.length < 4) throw new Error("four rows");
}

export function required(rows: readonly string[]): string {
  requireRows(rows);
  return rows.at(3)!;
}
