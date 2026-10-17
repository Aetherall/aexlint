type Named = { name: string } & { id: string };
export function pick(input: { items: Named[]; name: string }): { item: Named } {
  const item = input.items.find((entry) => entry.name === input.name);
  if (!item) throw new Error("missing");
  return { item };
}
export function first<T extends { id: string }>(input: { items: T[] }): { item: T } {
  const item = input.items[0] as T | undefined;
  if (!item) throw new Error("empty");
  return { item };
}
