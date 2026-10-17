type Item = { kind: "helix" | "spiral"; value: string };
declare const enabled: boolean;
export const first = (item: Item) => item.kind === "helix" && enabled ? { value: item.value } : {};
export const second = (item: Item) => item.kind === "helix" ? { value: item.value } : {};
