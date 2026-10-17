type Id = string & { brand: "manuscript-id" };
export function named(input: { id: Id }): { id: Id } {
  if (!input.id) throw new Error("empty id");
  return { id: input.id };
}
