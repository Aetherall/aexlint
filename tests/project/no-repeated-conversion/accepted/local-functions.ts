type Item = { kind: "lozenge" | "crescent"; value: string };
export function first(item: Item) {
  const convert = (value: string) => value.toUpperCase();
  return item.kind === "lozenge" ? { value: convert(item.value) } : {};
}
export function second(item: Item) {
  const convert = (value: string) => value.toLowerCase();
  return item.kind === "lozenge" ? { value: convert(item.value) } : {};
}
