type Item = { kind: "zigzag" | "chevron"; value: string };
export function first(item: Item) {
  const key = () => "bogie";
  return item.kind === "zigzag" ? { [key()]: item.value } : {};
}
export function second(item: Item) {
  const key = () => "pantograph";
  return item.kind === "zigzag" ? { [key()]: item.value } : {};
}
