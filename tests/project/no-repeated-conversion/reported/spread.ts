type Item = { kind: "hydrofoil" | "pedalo"; value: string };
const extra = { value: "chassis" };
export const first = (item: Item) => item.kind === "hydrofoil" ? { ...extra, value: item.value } : {};
export const second = (other: Item) => other.kind === "hydrofoil" ? { ...extra, value: other.value } : {};
