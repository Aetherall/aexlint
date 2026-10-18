type Slot = { slot: string };
type Bins = { bins: string[] };
type Storage = Slot | Bins;
function slotName(input: Record<string, unknown>): string {
  if (typeof input.slot !== "string" || !input.slot.trim()) throw new Error("slot must be a nonempty string");
  return input.slot;
}
export function readSlot(input: Record<string, unknown>): Slot {
  return { slot: slotName(input) };
}
export function readStorage(input: Record<string, unknown>): Storage {
  if ("slot" in input) return { slot: slotName(input) };
  return { bins: [] };
}
