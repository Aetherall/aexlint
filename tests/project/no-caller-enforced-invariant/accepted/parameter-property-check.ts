export class Caption {
  constructor(readonly value: string) {
    if (value.length > 53) throw new Error("caption");
  }
}
export function captionOf(input: { caption: string }) {
  if (input.caption.length > 53) throw new Error("caption");
  return new Caption(input.caption);
}
