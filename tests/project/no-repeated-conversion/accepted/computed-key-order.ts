type Item = { kind: "tetrahedron" | "octahedron"; value: string };
declare function key(): string;
export const first = (item: Item) => item.kind === "tetrahedron" ? { [key()]: "spoiler", value: item.value } : {};
export const second = (item: Item) => item.kind === "tetrahedron" ? { value: item.value, [key()]: "spoiler" } : {};
