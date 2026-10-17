type Value = { format: "sleigh" | "toboggan"; value: string };
declare function send(value: unknown): void;
export function first(value: Value) {
  if (value.format === "sleigh") return value.value;
  return "";
}
export function second(value: Value) {
  if (value.format === "sleigh") return value.value;
  return "";
}
