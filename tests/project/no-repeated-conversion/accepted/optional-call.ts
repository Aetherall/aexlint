type Item = { kind: "dodecahedron" | "icosahedron"; value: string; format: (value: string) => string };
export const first = (item: Item) => item.kind === "dodecahedron" ? { value: item.format?.(item.value) } : {};
export const second = (item: Item) => item.kind === "dodecahedron" ? { value: item.format(item.value) } : {};
