export function second(rows: number[], counts: Map<string, number>, key: string): number {
  if (counts.has(key) && counts.get(key)! > 2) return 0;
  return rows.length > 1 ? rows.at(1)! : 0;
}

export function holes(rows: (string | undefined)[], counts: Map<string, number | null>, key: string): unknown[] {
  if (!rows.length || !counts.has(key)) return [];
  return [rows.at(0)!, counts.get(key)!];
}

export function short(rows: string[]): string {
  if (!rows.length) return "";
  return rows.at(1)! + rows.at(-2)!;
}

export function removed(rows: string[], counts: Map<string, number>, key: string): unknown[] {
  if (!rows.length || !counts.has(key)) return [];
  rows.pop();
  counts.clear();
  return [rows.at(0)!, counts.get(key)!];
}

export function rebound(rows: string[]): string {
  if (!rows.length) return "";
  rows = [];
  return rows.at(0)!;
}

export function otherKey(counts: Map<string, number>, key: string, other: string): number {
  if (!counts.has(key)) return 0;
  return counts.get(other)!;
}

export function otherList(rows: string[], others: string[]): string {
  if (!rows.length) return "";
  return others.at(0)!;
}

export function inLoop(rows: string[]): string[] {
  const out: string[] = [];
  if (!rows.length) return out;
  while (rows.length) out.push(rows.at(0)!, rows.shift()!);
  return out;
}

export function checked(rows: string[]): string {
  const head = rows.at(0);
  if (!head) return "";
  return head;
}

export function loose(rows: any, things: unknown[], none: never[], missing: Missing[]): unknown[] {
  if (!rows.length || !things.length || !none.length || !missing.length) return [];
  return [rows.at(0)!, things.at(0)!, none.at(0)!, missing.at(0)!];
}

export function unconstrained<T>(rows: readonly T[]): T | undefined {
  if (!rows.length) return undefined;
  return rows.at(0)!;
}
