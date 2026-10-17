type Item = { kind: "rotor" | "stator"; value: string };
const extra = "bumper";
export const first = (item: Item) => item.kind === "rotor" ? { value: extra + item.value } : {};
export const second = (other: Item) => other.kind === "rotor" ? { value: extra + other.value } : {};
