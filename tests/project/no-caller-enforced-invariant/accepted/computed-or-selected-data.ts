export class Person { private constructor(readonly id: string) {} static withId(id: string): Person { return new Person(id); } }
export function parse(input: { last?: string }) {
  if (!input.last || input.last.length > 40) throw new Error("last");
  const id = input.last.split(":")[1] ?? "generated";
  return Person.withId(id);
}
export function pick(input: { format: string; folio: string; pamphlet: string }) {
  if (input.format !== "folio" && input.format !== "pamphlet") throw new Error("format");
  return Person.withId(input.format === "folio" ? input.folio : input.pamphlet);
}
