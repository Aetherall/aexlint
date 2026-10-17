type Item = { kind: "cuboid" | "pyramid"; nested?: { value: string } };
export const first = (item: Item) => item.kind === "cuboid" ? { value: item.nested?.value } : {};
export const second = (item: Item) => item.kind === "cuboid" ? { value: item.nested!.value } : {};
