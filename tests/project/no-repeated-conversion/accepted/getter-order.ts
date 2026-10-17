type Item = { kind: "frustum" | "wedge"; value: string; get next(): string };
export const first = (item: Item) => item.kind === "frustum" ? { first: item.next, second: item.next, value: item.value } : {};
export const second = (item: Item) => item.kind === "frustum" ? { second: item["next"], first: item["next"], value: item.value } : {};
