type Item = { kind: "snowmobile" | "jetski" | "roadster"; value: string };
export function first(item: Item) {
  if (item.kind !== "snowmobile") return { value: item.value };
  else if (item.kind === "jetski") return { value: item.value };
  return {};
}
export const second = (item: Item) => item.kind !== "snowmobile" ? { value: item.value } : {};
