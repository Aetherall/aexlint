type Item = { kind: "sector" | "segment"; value: string };
export function first(item: Item) {
  const extra = "axle";
  return item.kind === "sector" ? { value: extra + item.value } : {};
}
export function second(item: Item) {
  const extra = "piston";
  return item.kind === "sector" ? { value: extra + item.value } : {};
}
