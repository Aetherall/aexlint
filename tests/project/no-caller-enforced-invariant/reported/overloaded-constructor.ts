export class Tag {
  constructor(value: string);
  constructor(value: string, color: string);
  constructor(readonly value: string, readonly color = "grey") {}
}
export function tagOf(input: { tag: string }) {
  if (input.tag.length > 31) throw new Error("tag");
  return new Tag(input.tag);
}
