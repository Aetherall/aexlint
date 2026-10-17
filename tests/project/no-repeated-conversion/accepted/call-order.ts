type Item = { kind: "ellipsoid" | "paraboloid"; value: string };
declare function next(): string;
export const first = (item: Item) => item.kind === "ellipsoid" ? { first: next(), second: next(), value: item.value } : {};
export const second = (item: Item) => item.kind === "ellipsoid" ? { second: next(), first: next(), value: item.value } : {};
