export class Item { private constructor(readonly owner: string) {} static create(owner: string): Item { return new Item(owner); } }
export function ensure(existing: { owner: string } | undefined, owner: string) {
  if (existing && existing.owner !== owner) throw new Error("other owner");
  if (!existing) return Item.create(owner);
  return undefined;
}
