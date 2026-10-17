type Summary = { summary: string };
export async function load(input: { text: string }): Promise<Summary> {
  const text = input.text.trim();
  if (!text) throw new Error("empty");
  return { summary: text };
}
export function tier(input: string): Summary {
  switch (input) {
    case "folio": return { summary: input };
    default: throw new Error("tier");
  }
}
export function named({ name }: { name: string }): Summary { if (!name) throw new Error("x"); return { summary: name }; }
export function each(input: string[]): Summary { for (const item of input) { if (!item) throw new Error("x"); } return { summary: input.join() }; }
export const arrow = (input: string): Summary => { if (!input) throw new Error("x"); return { summary: input }; };
