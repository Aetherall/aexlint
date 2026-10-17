type Order = { items: number[] };
function sum(values: number[]): number { return values.reduce((a, b) => a + b, 0); }
export function order(input: { items: number[] }): Order {
  const total = sum(input.items);
  if (total > 10) throw new Error("too large");
  return { items: input.items };
}
export function collect(input: string[]): { values: string[] } {
  const out: { values: string[] } = { values: [] };
  for (const value of input) {
    if (!value.trim()) throw new Error("blank");
    out.values.push(value);
  }
  return out;
}
