type Item = { kind: "velomobile" | "quadbike"; nested?: { value: string } };
export const first = (item: Item) => item.kind === "velomobile" ? { value: item.nested?.value } : {};
export const second = (other: Item) => other.kind === "velomobile" ? { value: other.nested?.value } : {};
