declare const brand: unique symbol;
export type Checked = { readonly [brand]: true; model: string };
export function check(input: { model: string }): Checked {
  if (!input.model) throw new Error("model");
  return input as Checked;
}
