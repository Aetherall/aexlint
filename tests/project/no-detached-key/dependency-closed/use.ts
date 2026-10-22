import { Slot } from "./slot.js";

export function slots(names: string[]): string[] {
  const all = new Map<string, Slot>();
  for (const name of names) all.set(name, new Slot());
  const out: string[] = [];
  for (const [name, slot] of all) out.push(slot.fill(name));
  return out;
}
