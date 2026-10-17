import { Item, Kind } from "./kinds.js";
const LABELS: Record<Kind, string> = { monsoon: "Monsoon", typhoon: "Typhoon", cyclone: "Cyclone" };
export function g(item: Item, other: Item, text: string) {
  switch (item.kind) { case "monsoon": return 1; case "typhoon": return 2; case "cyclone": return 3; }
  return [LABELS[item.kind], item.kind === other.kind, item.kind === text, item.label === "rainbow", item.enabled === true, typeof item.kind === "string"];
}
