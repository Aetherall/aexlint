type Rows<T> = readonly T[];

export function first<T extends object>(rows: Rows<T>): T | undefined {
  if (!rows.length) return undefined;
  return rows.at(0)!;
}

export function value<K, V extends string>(entries: Map<K, V>, key: K): V {
  if (!entries.has(key)) return "" as V;
  return entries.get(key)!;
}

export function either(rows: string[] | readonly string[]): string {
  if (rows.length < 1) return "";
  return rows.at(0)!;
}
