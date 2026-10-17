type Item = { kind: "gyrocopter" | "dirigible"; value: string; mode: "tractor" | "harvester" };
export const first = (item: Item) => item.kind === "gyrocopter" && item.mode === "tractor" ? { value: item.value } : {};
export const second = (item: Item) => item.kind === "gyrocopter" ? { value: item.value } : {};
