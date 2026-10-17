type Item = { kind: "gimbal" | "sprocket"; value: string };
const scale = 2;
export const first = (item: Item) => item.kind === "gimbal" ? { scale, size: 0x10, value: item.value } : {};
export const second = (other: Item) => other.kind === "gimbal" ? { scale: scale, size: 16, value: other.value } : {};
