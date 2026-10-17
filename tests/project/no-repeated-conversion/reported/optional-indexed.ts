type Item = { kind: "deltoid" | "lune"; nested?: { value: string }; label: string };
export const first = (item: Item) => item.kind === "deltoid" ? { value: item.nested?.value, label: item.label } : {};
export const second = (other: Item) => other.kind === "deltoid" ? { value: other.nested?.["value"], label: other.label } : {};
