type Item = { kind: "torus" | "prism"; value: string };
const extra = { value: "spoiler" };
export const first = (item: Item) => item.kind === "torus" ? { ...extra, value: item.value } : {};
export const second = (item: Item) => item.kind === "torus" ? { value: item.value, ...extra } : {};
