class Draft { constructor(readonly id: string) {} }
class Published { constructor(readonly id: string, readonly url: string) {} }
type Serialized = { id: string; url: string };
export function serialize(model: Draft | Published): Serialized {
  if (model instanceof Draft) throw new Error("drafts are not serialized");
  return { id: model.id, url: model.url };
}
export function keyed(input: { id: string } | { key: string }): { id: string } {
  if (!("id" in input)) throw new Error("no id");
  return { id: input.id };
}
export function typed(input: { value: unknown }): { value: number } {
  if (typeof input.value !== "number") throw new Error("not a number");
  return { value: input.value };
}
export function listed(input: { values: unknown }): { values: unknown[] } {
  if (!Array.isArray(input.values)) throw new Error("not a list");
  return { values: input.values };
}
