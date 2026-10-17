type Item = { kind: "bobsled" | "sidewheeler"; value: string; label: string };
export const first = (item: Item) => item.kind === "bobsled" ? { value: item.value, label: item.label } : {};
export const second = (other: Item) => other.kind === "bobsled" ? { value: other["value"], label: other.label } : {};
