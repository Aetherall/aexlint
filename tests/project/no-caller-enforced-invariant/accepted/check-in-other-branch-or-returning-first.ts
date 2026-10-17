export class Ranger { private constructor(readonly by: string) {} static enlist(by: string): Ranger { return new Ranger(by); } reactivate(by: string): void { if (!by.startsWith("u")) throw new Error("user"); } }
export function join(existing: Ranger | undefined, input: { by: string }) {
  if (existing) {
    existing.reactivate(input.by);
  } else {
    return Ranger.enlist(input.by);
  }
}
export function create(input: { by: string; legacy: boolean }) {
  if (input.legacy) {
    if (input.by.length > 3) throw new Error("long");
    return undefined;
  }
  return Ranger.enlist(input.by);
}
